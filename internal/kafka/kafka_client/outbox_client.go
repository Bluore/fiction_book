package kafka_client

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type OutboxClient interface {
	OutboxProducerClient(ctx context.Context, cfg *config.Config)
}
type outboxClient struct {
	OutboxRepository        repository.OutboxRepository
	BookESRepository        repository.BookESRepository
	BookChapterESRepository repository.BookChapterESRepository
	DB                      *gorm.DB
}

func NewOutboxClient(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) OutboxClient {
	return &outboxClient{
		OutboxRepository:        outboxRepository,
		BookESRepository:        bookESRepository,
		BookChapterESRepository: bookChapterESRepository,
		DB:                      db,
	}
}

func (c *outboxClient) OutboxProducerClient(ctx context.Context, cfg *config.Config) {
	writer := kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Addr...),
		Topic:    cfg.Kafka.Topic.DatabaseSync,
		Balancer: &kafka.LeastBytes{},
	}

	tick := time.NewTicker(1 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			c.syncOutboxToMassageQueue(&writer)
		case <-ctx.Done():
			logger.Info("exit outbox producer")
			return
		}
	}
}

func (c *outboxClient) syncOutboxToMassageQueue(w *kafka.Writer) {
	for {
		var err error
		msg, err := c.OutboxRepository.GetEventByTableName(context.Background(), c.DB, model.Outbox{})
		if err != nil {
			logger.Error(fmt.Sprintf("syncOutboxToMassageQueue GetEventByTableName error: %v", err))
			return
		}
		if msg == nil {
			return
		}

		jsonEncode, err := json.Marshal(msg)
		if err != nil {
			logger.Error(fmt.Sprintf("syncOutboxToMassageQueue json.Marshal error: %v", err))
			return
		}

		err = w.WriteMessages(context.Background(),
			kafka.Message{
				Value: jsonEncode,
			})
		if err != nil {
			logger.Error(fmt.Sprintf("syncOutboxToMassageQueue WriteMessages error: %v", err))
			return
		}

		err = c.OutboxRepository.FinishEventByTableNameAndID(context.Background(), c.DB, model.Outbox{}, msg.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("syncOutboxToMassageQueue FinishEventByTableName error: %v", err))
			logger.Warn(fmt.Sprintf("syncOutboxToMassageQueue FinishEventByTableName error: %v", err))
			return
		}
	}
}
