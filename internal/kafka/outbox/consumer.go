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

		err = syncData(ctx, h, massageDecodeData)
		if err != nil {
			logger.Error(fmt.Sprintf("Error syncing data: %s", err))
			logger.Warn(fmt.Sprintf("Error syncing data: %s", err))
			return
		}

		err = r.CommitMessages(context.Background(), massage)
		if err != nil {
			logger.Error(fmt.Sprintf("Error committing message: %s", err))
			logger.Warn(fmt.Sprintf("Error committing message: %s", err))
			return
		}
	}
}

// syncData 同步数据
func syncData(ctx context.Context, h *wire.MassageQueue, massageDecodeData *model.Outbox) error {
	switch massageDecodeData.Type {
	case model.Book{}.TableName():
		return syncBookES(ctx, h, massageDecodeData.Operation, &massageDecodeData.Payload)
	case model.BookChapter{}.TableName():
		return syncBookChapterES(ctx, h, massageDecodeData.Operation, &massageDecodeData.Payload)
	default:
	}
	
	logger.Warn(fmt.Sprintf("Unknown table name: %s", massageDecodeData.TableName()))
	return nil
}

// syncBookES 同步图书ES
func syncBookES(ctx context.Context, h *wire.MassageQueue, operation model.DatabaseOperation, bookDecode *json.RawMessage) error {
	var book model.Book
	err := json.Unmarshal(*bookDecode, &book)
	if err != nil {
		return err
	}

	switch operation {
	case model.DatabaseOperationCreate:
		return h.Handler.BookESRepository.Create(ctx, &book)
	case model.DatabaseOperationUpdate:
		return h.Handler.BookESRepository.Update(ctx, &book)
	case model.DatabaseOperationDelete:
		return h.Handler.BookESRepository.Delete(ctx, book.ID)
	}
	return nil
}

// syncBookChapterES 同步图书章节ES
func syncBookChapterES(ctx context.Context, h *wire.MassageQueue, operation model.DatabaseOperation, chapterDecode *json.RawMessage) error {
	var chapter model.BookChapter
	err := json.Unmarshal(*chapterDecode, &chapter)
	if err != nil {
		return err
	}

	switch operation {
	case model.DatabaseOperationCreate:
		return h.Handler.BookChapterESRepository.Create(ctx, &chapter)
	case model.DatabaseOperationUpdate:
		return h.Handler.BookChapterESRepository.Update(ctx, &chapter)
	case model.DatabaseOperationDelete:
		return h.Handler.BookChapterESRepository.Delete(ctx, chapter.ID)
	}
	return nil
}
