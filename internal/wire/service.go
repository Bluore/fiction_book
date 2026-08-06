package wire

import (
	"fiction-book/internal/cache"

	agilepool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/wire"
	"gorm.io/gorm"

	"fiction-book/internal/repository"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
	"fiction-book/pkg/payment"
)

// ServiceSet 服务层 Provider 集合
var ServiceSet = wire.NewSet(
	ProvideUserService,
	ProvideBookService,
	ProvideBookChapterService,
	ProvideUserAssetService,
	ProvideUserPurchasedBookService,
	ProvideUserPurchasedChapterService,
	ProvideBuyVIPService,
	ProvideUserBookshelfService,
	ProvideUserHistoryService,
	ProvideOrderService,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	userStatCache cache.UserStatCache,
	userAssetCache cache.UserAssetCache,
	authCache cache.AuthCache,
	db *gorm.DB,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(
		userRepo,
		userAssetRepo,
		userStatCache,
		userAssetCache,
		authCache,
		db,
		cfg,
	)
}

// ProvideBookService 提供书籍服务
func ProvideBookService(
	bookRepo repository.BookRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	outboxRepo repository.OutboxRepository,
	bookCatch cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	db *gorm.DB,
	cfg *config.Config,
) service.BookService {
	return service.NewBookService(
		bookRepo,
		bookESRepo,
		userRepo,
		userAssetRepo,
		purchasedBookRepo,
		outboxRepo,
		bookCatch,
		userStatCache,
		bookThermalCache,
		db,
		cfg,
	)
}

// ProvideBookChapterService 提供章节仓库
func ProvideBookChapterService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	bookChapterRepo repository.BookChapterRepository,
	bookChapterESRepo repository.BookChapterESRepository,
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository,
	userAssetRepo repository.UserAssetRepository,
	outboxRepo repository.OutboxRepository,
	bookCatch cache.BookCache,
	bookChapterCatch cache.BookChapterCache,
	userStatCache cache.UserStatCache,
	limitCatch cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
) service.BookChapterService {
	return service.NewBookChapterService(
		userRepo,
		bookRepo,
		bookChapterRepo,
		bookChapterESRepo,
		userPurchasedChapterRepo,
		userAssetRepo,
		outboxRepo,
		bookCatch,
		bookChapterCatch,
		userStatCache,
		limitCatch,
		db,
		cfg,
	)
}

// ProvideUserAssetService  提供用户资产仓库
func ProvideUserAssetService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	db *gorm.DB,
	cfg *config.Config,
) service.UserAssetService {
	return service.NewUserAssetService(
		userRepo,
		userAssetRepo,
		db,
		cfg,
	)
}

// ProvideUserPurchasedBookService  提供用户已购买的书籍服务
func ProvideUserPurchasedBookService(
	userRepo repository.UserRepository,
	userPurchasedBookRepo repository.UserPurchasedBookRepository,
	cfg *config.Config,
) service.UserPurchasedBookService {
	return service.NewUserPurchasedBookService(userRepo, userPurchasedBookRepo, cfg)

}

// ProvideUserPurchasedChapterService 提供用户已购买的章节服务
func ProvideUserPurchasedChapterService(
	userRepo repository.UserRepository,
	userPurchasedChapterRepo repository.UserPurchasedChapterRepository,
	cfg *config.Config,
) service.UserPurchasedChapterService {
	return service.NewUserPurchasedChapterService(userRepo, userPurchasedChapterRepo, cfg)
}

// ProvideBuyVIPService 提供VIP购买服务
func ProvideBuyVIPService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	db *gorm.DB,
	cfg *config.Config,
) service.BuyVIP {
	return service.NewBuyVIPService(userRepo, userAssetRepo, db, cfg)
}

// ProvideUserBookshelfService 提供用户书架服务
func ProvideUserBookshelfService(
	userRepo repository.UserRepository,
	bookRepo repository.BookRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	bookCatch cache.BookCache,
	limitCatch cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
) service.UserBookshelfService {
	return service.NewUserBookshelfService(
		userRepo,
		bookRepo,
		userBookshelfRepo,
		bookCatch,
		limitCatch,
		db,
		cfg,
	)
}

// ProvideUserHistoryService 提供用户访问书籍历史服务
func ProvideUserHistoryService(
	userRepo repository.UserRepository,
	userHistoryRepo repository.UserHistoryRepository,
	bookRepo repository.BookRepository,
	db *gorm.DB,
	cfg *config.Config,
) service.UserHistoryService {
	return service.NewUserHistoryService(userRepo, userHistoryRepo, bookRepo, db, cfg)
}

// ProvideOrderService 提供订单服务
func ProvideOrderService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	orderRepo repository.OrderRepository,
	orderCache cache.OrderCache,
	db *gorm.DB,
	cfg *config.Config,
	alipayStrategy payment.Strategy,
	pool *agilepool.Pool,
) service.OrderService {
	return service.NewOrderService(userRepo, userAssetRepo, orderRepo, orderCache, db, cfg, alipayStrategy, pool)
}
