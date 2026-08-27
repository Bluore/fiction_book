package wire

import (
	"fiction-book/internal/cache"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/kafka/kafka_server"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"

	"github.com/google/wire"
	"gorm.io/gorm"
)

// MassageQueueSet 消息队列层 Provider 集合
var MassageQueueSet = wire.NewSet(
	ProvideOutboxClient,
	ProvideOutboxServer,
	ProvideFeedClient,
	ProvideFeedClientV1,
	ProvideFeedServer,
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

// ProvideFeedClient 提供 Feed 推送生产者
func ProvideFeedClient(cfg *config.Config) kafka_client.FeedClient {
	return kafka_client.NewFeedClient(cfg)
}

// ProvideFeedClientV1 提供旧版 Feed 推送生产者（章节推送仍在使用，待迁移）
func ProvideFeedClientV1(cfg *config.Config) kafka_client.FeedClientV1 {
	return kafka_client.NewFeedClientV1(cfg)
}

// ProvideFeedServer 提供 Feed 推送消费者
func ProvideFeedServer(
	userInboxRepository repository.UserInboxRepository,
	userLikeRepository repository.UserLikeRepository,
	userLikeCache cache.UserLikeCache,
	feedCache cache.UserFeedCache,
	consumeRecordRepository repository.ConsumeRecordRepository,
	db *gorm.DB,
) kafka_server.FeedServer {
	return kafka_server.NewFeedServer(userInboxRepository, userLikeRepository, userLikeCache, feedCache, consumeRecordRepository, db)
}
