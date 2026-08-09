package kafka_server

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const feedConsumerGroupID = "feed_syncer"

type FeedServer interface {
	FeedConsumerClient(ctx context.Context, cfg *config.Config)
}

type feedServer struct {
	feedCursorRepo repository.FeedCursorRepository
	feedCache      cache.FeedCache
	db             *gorm.DB
}

type feedPushMessage struct {
	Type      string             `json:"type"`
	Feeds     []*model.FeedInbox `json:"feeds"`
	CreatedAt time.Time          `json:"created_at"`
}

func NewFeedServer(
	feedCursorRepo repository.FeedCursorRepository,
	feedCache cache.FeedCache,
	db *gorm.DB,
) FeedServer {
	return &feedServer{
		feedCursorRepo: feedCursorRepo,
		feedCache:      feedCache,
		db:             db,
	}
}

func (s *feedServer) FeedConsumerClient(ctx context.Context, cfg *config.Config) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Addr,
		Topic:    cfg.Kafka.Topic.Feed,
		GroupID:  feedConsumerGroupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := s.consumeMessage(ctx, reader); err != nil {
				if errors.Is(err, context.Canceled) {
					logger.Info("exit feed consumer")
					return
				}
				logger.Error(fmt.Sprintf("feed consumer error: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *feedServer) consumeMessage(ctx context.Context, reader *kafka.Reader) error {
	message, err := reader.FetchMessage(ctx)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := s.handleFeedMessage(ctx, message.Value); err != nil {
		return err
	}
	return reader.CommitMessages(ctx, message)
}

func (s *feedServer) handleFeedMessage(ctx context.Context, messageValue []byte) error {
	var message feedPushMessage
	if err := json.Unmarshal(messageValue, &message); err != nil {
		return err
	}
	if message.Type != kafka_client.FeedPushMessageType {
		return nil
	}
	if len(message.Feeds) == 0 {
		return nil
	}

	newFeeds, err := s.filterNewFeeds(ctx, message.Feeds)
	if err != nil {
		return err
	}

	if len(newFeeds) > 0 {
		if err := s.createFeeds(ctx, newFeeds); err != nil {
			return err
		}
		if err := s.increaseUnread(ctx, newFeeds); err != nil {
			return err
		}
	}

	return s.warmFeeds(ctx, message.Feeds)
}

func (s *feedServer) filterNewFeeds(
	ctx context.Context,
	feeds []*model.FeedInbox,
) ([]*model.FeedInbox, error) {
	feedIDs := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		feedIDs = append(feedIDs, feed.ID)
	}

	existingFeedIDs := make([]string, 0)
	if err := s.db.WithContext(ctx).
		Model(&model.FeedInbox{}).
		Where("id IN ?", feedIDs).
		Pluck("id", &existingFeedIDs).Error; err != nil {
		return nil, err
	}

	existing := make(map[string]struct{}, len(existingFeedIDs))
	for _, feedID := range existingFeedIDs {
		existing[feedID] = struct{}{}
	}

	newFeeds := make([]*model.FeedInbox, 0, len(feeds))
	for _, feed := range feeds {
		if _, ok := existing[feed.ID]; ok {
			continue
		}
		newFeeds = append(newFeeds, feed)
	}
	return newFeeds, nil
}

func (s *feedServer) createFeeds(ctx context.Context, feeds []*model.FeedInbox) error {
	return s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(feeds, 1000).Error
}

func (s *feedServer) increaseUnread(ctx context.Context, feeds []*model.FeedInbox) error {
	counts := feedUnreadCounts(feeds)
	for count, userIDs := range groupUserIDsByCount(counts) {
		if err := s.feedCursorRepo.BatchUpdateUnreadCount(ctx, userIDs, count); err != nil {
			return err
		}
		if err := s.feedCache.BatchIncrUnreadCount(ctx, userIDs, int64(count)); err != nil {
			return err
		}
	}
	return nil
}

func (s *feedServer) warmFeeds(ctx context.Context, feeds []*model.FeedInbox) error {
	userFeeds := make(map[string][]*model.FeedInbox)
	for _, feed := range feeds {
		userFeeds[feed.UserID] = append(userFeeds[feed.UserID], feed)
	}

	for userID, feeds := range userFeeds {
		if err := s.feedCache.PushFeeds(ctx, userID, feeds); err != nil {
			return err
		}
	}
	return nil
}

func feedUnreadCounts(feeds []*model.FeedInbox) map[string]int {
	counts := make(map[string]int)
	for _, feed := range feeds {
		counts[feed.UserID]++
	}
	return counts
}

func groupUserIDsByCount(counts map[string]int) map[int][]string {
	groups := make(map[int][]string)
	for userID, count := range counts {
		groups[count] = append(groups[count], userID)
	}
	return groups
}
