package wire

import (
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/kafka/kafka_server"
	"fiction-book/internal/repository"

	"github.com/google/wire"
	"gorm.io/gorm"
)

// MassageQueueSet 消息队列层 Provider 集合
var MassageQueueSet = wire.NewSet(
	ProvideOutboxClient,
	ProvideOutboxServer,
)

// ProvideOutboxClient 提供 outbox 生产者客户端
func ProvideOutboxClient(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) kafka_client.OutboxClient {
	return kafka_client.NewOutboxClient(outboxRepository, bookESRepository, bookChapterESRepository, db)
}

// ProvideOutboxServer 提供 outbox 消费者服务端
func ProvideOutboxServer(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) kafka_server.OutboxServer {
	return kafka_server.NewOutboxServer(outboxRepository, bookESRepository, bookChapterESRepository, db)
}
