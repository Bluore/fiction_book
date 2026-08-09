package kafka_client

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/pkg/config"
	"time"

	"github.com/segmentio/kafka-go"
)

const (
	FeedPushMessageType = "feed_push"
	maxFeedMessageFeeds = 50
)

type FeedClient interface {
	PublishFeeds(ctx context.Context, cfg *config.Config, feeds []*model.FeedInbox) error
}

type feedClient struct{}

type FeedPushMessage struct {
	Type      string             `json:"type"`
	Feeds     []*model.FeedInbox `json:"feeds"`
	CreatedAt time.Time          `json:"created_at"`
}

func NewFeedClient() FeedClient {
	return &feedClient{}
}

// PublishFeeds 把已经构造好的 FeedInbox 批量发送到 Kafka。
// 调用方负责决定何时走 Kafka，比如推送用户数大于 50 时使用。
func (c *feedClient) PublishFeeds(ctx context.Context, cfg *config.Config, feeds []*model.FeedInbox) error {
	if len(feeds) == 0 {
		return nil
	}

	writer := kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Addr...),
		Topic:    cfg.Kafka.Topic.Feed,
		Balancer: &kafka.LeastBytes{},
	}
	defer writer.Close()

	for start := 0; start < len(feeds); start += maxFeedMessageFeeds {
		end := start + maxFeedMessageFeeds
		if end > len(feeds) {
			end = len(feeds)
		}

		messageEncode, err := json.Marshal(FeedPushMessage{
			Type:      FeedPushMessageType,
			Feeds:     feeds[start:end],
			CreatedAt: time.Now(),
		})
		if err != nil {
			return err
		}

		if err := writer.WriteMessages(ctx, kafka.Message{
			Key:   []byte(feeds[start].BookID),
			Value: messageEncode,
		}); err != nil {
			return err
		}
	}

	return nil
}
