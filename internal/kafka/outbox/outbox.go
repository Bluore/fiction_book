package outbox

import (
	"context"
	"errors"
	"fiction-book/internal/wire"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"

	"github.com/segmentio/kafka-go"
)

func KafkaInit(ctx context.Context, cfg *config.Config, h *wire.MassageQueue) {

	go outboxProducerClient(ctx, cfg, h)
	go outboxConsumerClient(ctx, cfg, h)
}

func createTopic(k *kafka.Conn, topicName string) {
	err := k.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && !errors.Is(err, kafka.TopicAlreadyExists) {
		logger.Fatal(fmt.Sprintf("outbox client create topic err: %v", err))
	}
}
