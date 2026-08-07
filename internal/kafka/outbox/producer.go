package outbox

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/internal/wire"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"github.com/segmentio/kafka-go"
)

func outboxProducerClient(ctx context.Context, cfg *config.Config, h *wire.MassageQueue) {
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
			syncOutboxToMassageQueue(h, &writer)
		case <-ctx.Done():
			logger.Info("exit outbox producer")
			return
		}
	}
}

func syncOutboxToMassageQueue(h *wire.MassageQueue, w *kafka.Writer) {
	for {
		var err error
		msg, err := h.Handler.OutboxRepository.GetEventByTableName(context.Background(), h.Handler.DB, model.Outbox{})
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

		err = h.Handler.OutboxRepository.FinishEventByTableNameAndID(context.Background(), h.Handler.DB, model.Outbox{}, msg.ID)
		if err != nil {
			logger.Error(fmt.Sprintf("syncOutboxToMassageQueue FinishEventByTableName error: %v", err))
			logger.Warn(fmt.Sprintf("syncOutboxToMassageQueue FinishEventByTableName error: %v", err))
			return
		}
	}
}
