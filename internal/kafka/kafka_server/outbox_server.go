package kafka_server

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"io"
	"time"

	"github.com/segmentio/kafka-go"
	"gorm.io/gorm"
)

type OutboxServer interface {
	OutboxConsumerClient(ctx context.Context, cfg *config.Config)
}
type outboxServer struct {
	OutboxRepository        repository.OutboxRepository
	BookESRepository        repository.BookESRepository
	BookChapterESRepository repository.BookChapterESRepository
	DB                      *gorm.DB
}

func NewOutboxServer(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) OutboxServer {
	return &outboxServer{
		OutboxRepository:        outboxRepository,
		BookESRepository:        bookESRepository,
		BookChapterESRepository: bookChapterESRepository,
		DB:                      db,
	}
}

func (s *outboxServer) OutboxConsumerClient(ctx context.Context, cfg *config.Config) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:   cfg.Kafka.Addr,
		Topic:     cfg.Kafka.Topic.DatabaseSync,
		GroupID:   "database_syncer",
		Partition: 0,
		MinBytes:  10e3,
		MaxBytes:  10e6,
	})

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			s.consumeMassage(ctx, reader)
		case <-ctx.Done():
			logger.Info("exit outbox consumer")
			return
		}
	}
}

func (s *outboxServer) consumeMassage(ctx context.Context, r *kafka.Reader) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		massage, err := r.FetchMessage(ctx)
		if errors.Is(err, io.ErrUnexpectedEOF) {
			return
		}
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error(fmt.Sprintf("Error reading message: %s", err))
			return
		}

		var massageDecodeData *model.Outbox

		err = json.Unmarshal(massage.Value, &massageDecodeData)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error(fmt.Sprintf("Error unmarshalling message: %s", err))
			return
		}

		err = s.syncData(ctx, massageDecodeData)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error(fmt.Sprintf("Error syncing data: %s", err))
			logger.Warn(fmt.Sprintf("Error syncing data: %s", err))
			return
		}

		err = r.CommitMessages(ctx, massage)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			logger.Error(fmt.Sprintf("Error committing message: %s", err))
			logger.Warn(fmt.Sprintf("Error committing message: %s", err))
			return
		}
	}
}

// syncData 同步数据
func (s *outboxServer) syncData(ctx context.Context, massageDecodeData *model.Outbox) error {
	switch massageDecodeData.Type {
	case (&model.Book{}).TableName():
		return s.syncBookES(ctx, massageDecodeData.Operation, &massageDecodeData.Payload)
	case (&model.BookChapter{}).TableName():
		return s.syncBookChapterES(ctx, massageDecodeData.Operation, &massageDecodeData.Payload)
	default:
	}

	logger.Warn(fmt.Sprintf("Unknown table name: %s", massageDecodeData.TableName()))
	return nil
}

// syncBookES 同步图书ES
func (s *outboxServer) syncBookES(ctx context.Context, operation model.DatabaseOperation, bookDecode *json.RawMessage) error {
	var book model.Book
	err := json.Unmarshal(*bookDecode, &book)
	if err != nil {
		return err
	}

	switch operation {
	case model.DatabaseOperationCreate:
		return s.BookESRepository.Create(ctx, &book)
	case model.DatabaseOperationUpdate:
		return s.BookESRepository.Update(ctx, &book)
	case model.DatabaseOperationDelete:
		return s.BookESRepository.Delete(ctx, book.ID)
	}
	return nil
}

// syncBookChapterES 同步图书章节ES
func (s *outboxServer) syncBookChapterES(ctx context.Context, operation model.DatabaseOperation, chapterDecode *json.RawMessage) error {
	var chapter model.BookChapter
	err := json.Unmarshal(*chapterDecode, &chapter)
	if err != nil {
		return err
	}

	switch operation {
	case model.DatabaseOperationCreate:
		return s.BookChapterESRepository.Create(ctx, &chapter)
	case model.DatabaseOperationUpdate:
		return s.BookChapterESRepository.Update(ctx, &chapter)
	case model.DatabaseOperationDelete:
		return s.BookChapterESRepository.Delete(ctx, chapter.ID)
	}
	return nil
}
