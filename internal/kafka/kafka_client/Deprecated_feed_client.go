package kafka_client

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

const (
	FeedPushMessageType = "feed_push"
	maxFeedMessageUsers = 50
)

type FeedClient interface {
	PublishFeeds(ctx context.Context, cfg *config.Config, feeds []*model.FeedInbox, users []string) error
}

type feedClient struct {
	writer *kafka.Writer
}

// Deprecated: FeedPushMessage 旧版 feed 推送消息结构
type FeedPushMessage struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Feeds     []*model.FeedInbox `json:"feeds"`
	Users     []string           `json:"users"`
	CreatedAt time.Time          `json:"created_at"`
}

func NewFeedClient(cfg *config.Config) FeedClient {
	writer, _ := InitFeedWriter(cfg)
	return &feedClient{
		writer: writer,
	}
}

func InitFeedWriter(cfg *config.Config) (*kafka.Writer, error) {
	writer := kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Addr...),
		Topic:    cfg.Kafka.Topic.Feed,
		Balancer: &kafka.LeastBytes{},
	}
	go func() {
		select {
		case <-constant.DoneServerCtx.Done():
			err := writer.Close()
			if err != nil {
				logger.Error("failed to close kafka writer", zap.Error(err))
			}
		}
	}()
	return &writer, nil
}

// PublishFeeds 把 Feed 列表和对应用户列表发送到 Kafka。
// 内部按用户数分片，避免单条消息过大。
func (c *feedClient) PublishFeeds(
	ctx context.Context,
	cfg *config.Config,
	feeds []*model.FeedInbox,
	users []string,
) error {
	if len(feeds) == 0 || len(users) == 0 {
		return nil
	}

	if len(feeds) != len(users) {
		logger.Error(
			"count mismatch between feeds and users",
			zap.Int("feed_count", len(feeds)),
			zap.Int("user_count", len(users)),
		)
		return errors.New("count mismatch between feeds and users")
	}
	var partitionID string = uuid.New().String()

	for start := 0; start < len(feeds); start += maxFeedMessageUsers {
		end := start + maxFeedMessageUsers
		if end > len(feeds) {
			end = len(feeds)
		}

		messageEncode, err := json.Marshal(FeedPushMessage{
			ID:        uuid.New().String(),
			Type:      FeedPushMessageType,
			Feeds:     feeds[start:end],
			Users:     users[start:end],
			CreatedAt: time.Now(),
		})
		if err != nil {
			return err
		}

		if err := c.writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(partitionID),
			Value: messageEncode,
		}); err != nil {
			return err
		}
	}

	return nil
}
