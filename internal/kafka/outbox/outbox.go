package outbox

import (
	"context"
	"fiction-book/internal/wire"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"

	"github.com/segmentio/kafka-go"
)

func KafkaInit(ctx context.Context, cfg *config.Config, h *wire.MassageQueue) {

	go outboxClient(ctx, cfg, h)
}

func createTopic(k *kafka.Conn, topicName string) {
	err := k.CreateTopics(kafka.TopicConfig{
		Topic:             topicName,
		NumPartitions:     1,
		ReplicationFactor: 1,
	})
	if err != nil && err != kafka.TopicAlreadyExists {
		logger.Fatal(fmt.Sprintf("outbox client create topic err: %v", err))
	}
}
