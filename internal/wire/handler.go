package wire

import (
	"github.com/google/wire"

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
		cfg,
	)
}
