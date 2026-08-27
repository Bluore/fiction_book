package kafka

import (
	"context"
	"errors"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/kafka/kafka_server"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"

	"github.com/segmentio/kafka-go"
)

type Handler struct {
	outboxClient kafka_client.OutboxClient
	outboxServer kafka_server.OutboxServer

	feedServer kafka_server.FeedServer
}

func NewHandler(
	outboxClient kafka_client.OutboxClient,
	outboxServer kafka_server.OutboxServer,
	feedServer kafka_server.FeedServer,
) *Handler {
	return &Handler{
		outboxClient: outboxClient,
		outboxServer: outboxServer,
		feedServer:   feedServer,
	}
}

func (h *Handler) KafkaInit(ctx context.Context, cfg *config.Config) {

	go h.outboxClient.OutboxProducerClient(ctx, cfg)
	go h.outboxServer.OutboxConsumerClient(ctx, cfg)

	go h.feedServer.FeedConsumerClient(ctx, cfg)
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
