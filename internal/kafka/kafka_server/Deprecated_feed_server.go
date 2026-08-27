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

const feedConsumerGroupIDV1 = "feed_syncer"

// Deprecated: FeedServerV1 旧版 Feed 推送服务端，已被 V2 取代，代码仅保留
type FeedServerV1 interface {
	FeedConsumerClient(ctx context.Context, cfg *config.Config)
}

type feedServerV1 struct {
	feedRepo          repository.FeedRepository
	feedCache         cache.FeedCache
	consumeRecordRepo repository.ConsumeRecordRepository
	db                *gorm.DB
}

func NewFeedServerV1(
	feedRepo repository.FeedRepository,
	feedCache cache.FeedCache,
	consumeRecordRepo repository.ConsumeRecordRepository,
	db *gorm.DB,
) FeedServerV1 {
	return &feedServerV1{
		feedRepo:          feedRepo,
		feedCache:         feedCache,
		consumeRecordRepo: consumeRecordRepo,
		db:                db,
	}
}

func (s *feedServerV1) FeedConsumerClient(ctx context.Context, cfg *config.Config) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Addr,
		Topic:    cfg.Kafka.Topic.Feed,
		GroupID:  feedConsumerGroupIDV1,
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

func (s *feedServerV1) consumeMessage(ctx context.Context, reader *kafka.Reader) error {
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

func (s *feedServerV1) handleFeedMessage(ctx context.Context, messageValue []byte) error {
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

func (s *feedServerV1) consumeFeedsInTx(
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

func (s *feedServerV1) warmFeeds(ctx context.Context, feeds []*model.FeedInbox) error {
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
