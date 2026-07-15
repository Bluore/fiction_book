package outbox

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/internal/wire"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
)

func outboxConsumerClient(ctx context.Context, cfg *config.Config, h *wire.MassageQueue) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   cfg.Kafka.Addr,
		Topic:     cfg.Kafka.Topic.DatabaseSync,
		GroupID:   "database_syncer",
		Partition: 0,
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	ticker := time.NewTicker(1 * time.Second)
	select {
	case <-ticker.C:
		consumeMassage(ctx, h, reader)
	case <-ctx.Done():
		return
	}
}

func consumeMassage(ctx context.Context, h *wire.MassageQueue, r *kafka.Reader) {
	for {
		massage, err := r.FetchMessage(ctx)
		if err == io.ErrUnexpectedEOF {
			return
		}
		if err != nil {
			logger.Error(fmt.Sprintf("Error reading message: %s", err))
			return
		}

		var massageDecodeData *model.Outbox

		err = json.Unmarshal(massage.Value, &massageDecodeData)
		if err != nil {
			logger.Error(fmt.Sprintf("Error unmarshalling message: %s", err))
			return
		}

		//todo 同步数据

		err = r.CommitMessages(context.Background(), massage)
		if err != nil {
			logger.Error(fmt.Sprintf("Error committing message: %s", err))
			logger.Warn(fmt.Sprintf("Error committing message: %s", err))
			return
		}
	}
}
