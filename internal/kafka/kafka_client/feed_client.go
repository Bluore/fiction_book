package kafka_client

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// FeedClient Feed 推送客户端接口
type FeedClient interface {
	// PublishFeed 将 Feed 列表写入消息队列，由消费端扇出到关注者收件箱
	PublishFeed(ctx context.Context, feed model.UserFeed) error
}

type feedClient struct {
	writer *kafka.Writer
}

// NewFeedClient 创建 Feed 推送客户端
func NewFeedClient(cfg *config.Config) FeedClient {
	writer, _ := InitFeedWriter(cfg)
	return &feedClient{
		writer: writer,
	}
}

// InitFeedWriter 初始化 Feed 主题写入器
func InitFeedWriter(cfg *config.Config) (*kafka.Writer, error) {
	writer := kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Addr...),
		Topic:    cfg.Kafka.Topic.Feed,
		Balancer: &kafka.LeastBytes{},
	}
	go func() {
		select {
		case <-constant.DoneServerCtx.Done():
			if err := writer.Close(); err != nil {
				logger.Error("failed to close kafka writer", zap.Error(err))
			}
		}
	}()
	return &writer, nil
}

// PublishFeed 将 Feed 写入消息队列 UserFeedMassageV2
// 数据量可能过大时，由消费端异步消化消息并扇出到关注者收件箱。
func (c *feedClient) PublishFeed(ctx context.Context, feed model.UserFeed) error {
	message := &model.UserFeedMassageV2{
		ID:       uuid.New().String(),
		Operator: model.UserFeedMassageOperatorPush,
		UserFeed: feed,
	}

	messageEncode, err := json.Marshal(message)
	if err != nil {
		return err
	}

	if err := c.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(message.ID),
		Value: messageEncode,
	}); err != nil {
		return err
	}

	return nil
}
