package wire

import (
	"fiction-book/internal/cache"
	"fiction-book/internal/kafka/kafka_client"

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
	ProvideUserLikeService,
	ProvideOrderService,
	ProvideFeedService,
	ProvideFeedServiceV2,
)

// ProvideUserService 提供用户服务
func ProvideUserService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	userCache cache.UserCache,
	userStatCache cache.UserStatCache,
	userAssetCache cache.UserAssetCache,
	authCache cache.AuthCache,
	db *gorm.DB,
	cfg *config.Config,
) service.UserService {
	return service.NewUserService(
		userRepo,
		userAssetRepo,
		userCache,
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
	bookTagRepo repository.BookTagRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	outboxRepo repository.OutboxRepository,
	userInfoRepo repository.UserInfoRepository,
	userFeedRepo repository.UserFeedRepository,
	bookCatch cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	bookTagCache cache.BookTagCache,
	userInfoCache cache.UserInfoCache,
	userFeedCache cache.UserFeedCache,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilepool.Pool,
) service.BookService {
	return service.NewBookService(
		bookRepo,
		bookTagRepo,
		bookESRepo,
		userRepo,
		userAssetRepo,
		purchasedBookRepo,
		outboxRepo,
		userInfoRepo,
		userFeedRepo,
		bookCatch,
		userStatCache,
		bookThermalCache,
		bookTagCache,
		userInfoCache,
		userFeedCache,
		db,
		cfg,
		pool,
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
	userBookshelfRepo repository.UserBookshelfRepository,
	feedRepo repository.FeedRepository,
	outboxRepo repository.OutboxRepository,
	bookCatch cache.BookCache,
	bookThermalCache cache.BookThermalCache,
	bookChapterCatch cache.BookChapterCache,
	userStatCache cache.UserStatCache,
	limitCatch cache.LimitCache,
	feedCache cache.FeedCache,
	feedClient kafka_client.FeedClient,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilepool.Pool,
) service.BookChapterService {
	return service.NewBookChapterService(
		userRepo,
		bookRepo,
		bookChapterRepo,
		bookChapterESRepo,
		userPurchasedChapterRepo,
		userAssetRepo,
		userBookshelfRepo,
		feedRepo,
		outboxRepo,
		bookCatch,
		bookThermalCache,
		bookChapterCatch,
		userStatCache,
		limitCatch,
		feedCache,
		feedClient,
		db,
		cfg,
		pool,
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
	bookThermalCache cache.BookThermalCache,
	limitCatch cache.LimitCache,
	db *gorm.DB,
	cfg *config.Config,
	pool *agilepool.Pool,
) service.UserBookshelfService {
	return service.NewUserBookshelfService(
		userRepo,
		bookRepo,
		userBookshelfRepo,
		bookCatch,
		bookThermalCache,
		limitCatch,
		db,
		cfg,
		pool,
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

// ProvideUserLikeService 提供用户关注服务
func ProvideUserLikeService(
	userRepo repository.UserRepository,
	userLikeRepo repository.UserLikeRepository,
	userLikeCache cache.UserLikeCache,
	db *gorm.DB,
) service.UserLikeService {
	return service.NewUserLikeService(userRepo, userLikeRepo, userLikeCache, db)
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

// Deprecated: ProvideFeedService 旧版提供 Feed 服务的依赖注入
func ProvideFeedService(
	feedRepo repository.FeedRepository,
	feedCache cache.FeedCache,
	db *gorm.DB,
) service.FeedService {
	return service.NewFeedService(feedRepo, feedCache, db)
}

// ProvideFeedServiceV2 提供新版 Feed 服务
func ProvideFeedServiceV2(
	bookRepo repository.BookRepository,
	bookTagRepo repository.BookTagRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	userLikeRepo repository.UserLikeRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	userCache cache.UserCache,
	userAssetCache cache.UserAssetCache,
	userFeedRepo repository.UserFeedRepository,
	userInboxRepo repository.UserInboxRepository,
	outboxRepo repository.OutboxRepository,
	userInfoRepo repository.UserInfoRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	bookCache cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	bookTagCache cache.BookTagCache,
	feedCache cache.UserFeedCache,
	userLikeCache cache.UserLikeCache,
	userInfoCache cache.UserInfoCache,
	userBookshelfCache cache.UserBookshelfCache,
	db *gorm.DB,
	cfg *config.Config,
) service.FeedServiceV2 {
	return service.NewFeedServiceV2(
		bookRepo,
		bookTagRepo,
		bookESRepo,
		userRepo,
		userAssetRepo,
		userLikeRepo,
		purchasedBookRepo,
		userCache,
		userAssetCache,
		userFeedRepo,
		userInboxRepo,
		outboxRepo,
		userInfoRepo,
		userBookshelfRepo,
		bookCache,
		userStatCache,
		bookThermalCache,
		bookTagCache,
		feedCache,
		userLikeCache,
		userInfoCache,
		userBookshelfCache,
		db,
		cfg,
	)
}
