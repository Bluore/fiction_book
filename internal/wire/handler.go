package wire

import (
	mq "fiction-book/internal/kafka"
	"fiction-book/internal/repository"

	"github.com/google/wire"
	"gorm.io/gorm"

	"fiction-book/internal/api/handler"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
)

// HandlerSet Handler 层 Provider 集合
var HandlerSet = wire.NewSet(
	ProvideHandler,
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
	orderService service.OrderService,
	buyVIPService service.BuyVIP,
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
		orderService,
		buyVIPService,
		cfg,
	)
}

var MassageHandlerSet = wire.NewSet(
	ProvideMassageHandlerSer,
)

func ProvideMassageHandlerSer(
	outboxRepository repository.OutboxRepository,
	bookESRepository repository.BookESRepository,
	bookChapterESRepository repository.BookChapterESRepository,
	db *gorm.DB,
) *mq.Handler {
	return mq.NewHandler(outboxRepository, bookESRepository, bookChapterESRepository, db)
}
