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
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const feedConsumerGroupID = "feed_syncer"

type FeedServer interface {
	FeedConsumerClient(ctx context.Context, cfg *config.Config)
}

type feedServer struct {
	feedRepo          repository.FeedRepository
	feedCache         cache.FeedCache
	consumeRecordRepo repository.ConsumeRecordRepository
	db                *gorm.DB
}

func NewFeedServer(
	feedRepo repository.FeedRepository,
	feedCache cache.FeedCache,
	consumeRecordRepo repository.ConsumeRecordRepository,
	db *gorm.DB,
) FeedServer {
	return &feedServer{
		feedRepo:          feedRepo,
		feedCache:         feedCache,
		consumeRecordRepo: consumeRecordRepo,
		db:                db,
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
	if errors.Is(err, io.EOF) {
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
	var message kafka_client.FeedPushMessage
	if err := json.Unmarshal(messageValue, &message); err != nil {
		return err
	}
	if message.Type != kafka_client.FeedPushMessageType {
		return fmt.Errorf("invalid feed message type: %v", message.Type)
	}
	if message.ID == "" {
		return fmt.Errorf("feed push message id is empty")
	}
	if len(message.Feeds) == 0 {
		return nil
	}
	if len(message.Feeds) != len(message.Users) {
		return fmt.Errorf("feed push message feeds and users count mismatch")
	}

	err := s.consumeFeedsInTx(ctx, &message)
	if err != nil {
		return err
	}
	if err := s.warmFeeds(ctx, message.Feeds); err != nil {
		logger.Warn("warm feed error", zap.Error(err))
	}
	return nil
}

func (s *feedServer) consumeFeedsInTx(
	ctx context.Context,
	message *kafka_client.FeedPushMessage,
) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		firstConsume, err := s.consumeRecordRepo.TryMarkConsumed(ctx, tx, message.ID)
		if err != nil {
			return err
		}
		if !firstConsume {
			return nil
		}

		// 写推送：创建Feed
		if _, err := s.feedRepo.CreateBatchIgnoreConflict(ctx, tx, message.Feeds); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *feedServer) warmFeeds(ctx context.Context, feeds []*model.FeedInbox) error {
	userFeeds := make(map[string][]*model.FeedInbox)
	for _, feed := range feeds {
		if feed == nil || feed.UserID == "" {
			continue
		}
		userFeeds[feed.UserID] = append(userFeeds[feed.UserID], feed)
	}

	for userID, feeds := range userFeeds {
		if _, err := s.feedCache.WarmFeeds(ctx, userID, feeds); err != nil {
			return err
		}
	}
	return nil
}
