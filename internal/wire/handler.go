package wire

import (
	"fiction-book/internal/kafka"
	"fiction-book/internal/kafka/kafka_client"
	"fiction-book/internal/kafka/kafka_server"

	"github.com/google/wire"

	"fiction-book/internal/api/handler"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
)

// HandlerSet Handler 层 Provider 集合
var HandlerSet = wire.NewSet(
	ProvideHandler,
	ProvideKafkaHandler,
)

// ProvideHandler 提供主 Handler
func ProvideHandler(
	userService service.UserService,
	bookService service.BookService,
	bookChapterService service.BookChapterService,
	userAssetService service.UserAssetService,
	userPurchasedBookService service.UserPurchasedBookService,
	userPurchasedChapterService service.UserPurchasedChapterService,
	userBookshelfService service.UserBookshelfService,
	userHistoryService service.UserHistoryService,
	userLikeService service.UserLikeService,
	orderService service.OrderService,
	buyVIPService service.BuyVIP,
	feedService service.FeedService,
	cfg *config.Config,
) *handler.Handler {
	return handler.NewHandler(
		userService,
		bookService,
		bookChapterService,
		userAssetService,
		userPurchasedBookService,
		userPurchasedChapterService,
		userBookshelfService,
		userHistoryService,
		userLikeService,
		orderService,
		buyVIPService,
		feedService,
		cfg,
	)
}

// ProvideKafkaHandler 提供 Kafka Handler
func ProvideKafkaHandler(
	outboxClient kafka_client.OutboxClient,
	outboxServer kafka_server.OutboxServer,
	feedServer kafka_server.FeedServer,
) *kafka.Handler {
	return kafka.NewHandler(outboxClient, outboxServer, feedServer)
}
