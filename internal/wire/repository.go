package wire

import (
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/google/wire"
	"gorm.io/gorm"

	"fiction-book/internal/repository"
)

// RepositorySet 仓储层 Provider 集合
var RepositorySet = wire.NewSet(
	ProvideUserRepository,
	ProvideBookRepository,
	ProvideBookTagRepository,
	ProvideBookChapterRepository,
	ProvideUserAssetRepository,
	ProvideUserPurchasedBookRepository,
	ProvideUserPurchasedChapterRepository,
	ProvideUserBookshelfRepository,
	ProvideUserHistoryRepository,
	ProvideOrderRepository,
	ProvideOutboxRepository,
	ProvideBookESRepository,
	ProvideBookChapterESRepository,
	ProvideFeedRepository,
	ProvideFeedCursorRepository,
	ProvideConsumeRecordRepository,
)

// ProvideUserRepository 提供用户仓储
func ProvideUserRepository(db *gorm.DB) repository.UserRepository {
	return repository.NewUserRepository(db)
}

// ProvideBookRepository 提供书籍仓储
func ProvideBookRepository(db *gorm.DB) repository.BookRepository {
	return repository.NewBookRepository(db)
}

func ProvideBookTagRepository(db *gorm.DB) repository.BookTagRepository {
	return repository.NewBookTagRepository(db)
}

// ProvideBookChapterRepository 提供书籍章节仓库
func ProvideBookChapterRepository(db *gorm.DB) repository.BookChapterRepository {
	return repository.NewBookChapterRepository(db)
}

// ProvideUserAssetRepository 提供用户资产仓库
func ProvideUserAssetRepository(db *gorm.DB) repository.UserAssetRepository {
	return repository.NewUserAssetRepository(db)
}

// ProvideUserPurchasedBookRepository 提供用户已购买的书籍仓库
func ProvideUserPurchasedBookRepository(db *gorm.DB) repository.UserPurchasedBookRepository {
	return repository.NewUserPurchasedBookRepository(db)
}

// ProvideUserPurchasedChapterRepository 提供用户已购买的章节仓库
func ProvideUserPurchasedChapterRepository(db *gorm.DB) repository.UserPurchasedChapterRepository {
	return repository.NewUserPurchasedChapterRepository(db)
}

// ProvideUserBookshelfRepository 提供用户书架仓库
func ProvideUserBookshelfRepository(db *gorm.DB) repository.UserBookshelfRepository {
	return repository.NewUserBookshelfRepository(db)
}

// ProvideUserHistoryRepository 提供用户历史仓库
func ProvideUserHistoryRepository(db *gorm.DB) repository.UserHistoryRepository {
	return repository.NewUserHistoryRepository(db)
}

// ProvideOrderRepository 提供订单仓库
func ProvideOrderRepository(db *gorm.DB) repository.OrderRepository {
	return repository.NewOrderRepository(db)
}

// ProvideOutboxRepository 提供事件发送仓库
func ProvideOutboxRepository(db *gorm.DB) repository.OutboxRepository {
	return repository.NewOutboxRepository(db)
}

// ProvideBookESRepository 提供书籍ES仓库
func ProvideBookESRepository(esClient *elasticsearch.Client) repository.BookESRepository {
	return repository.NewBookESRepository(esClient)
}

// ProvideBookChapterESRepository 提供书籍章节ES仓库
func ProvideBookChapterESRepository(esClient *elasticsearch.Client) repository.BookChapterESRepository {
	return repository.NewBookChapterESRepository(esClient)
}

// ProvideFeedRepository 提供 Feed 仓储
func ProvideFeedRepository(db *gorm.DB) repository.FeedRepository {
	return repository.NewFeedRepository(db)
}

// ProvideFeedCursorRepository 提供 Feed 游标仓储
func ProvideFeedCursorRepository(db *gorm.DB) repository.FeedCursorRepository {
	return repository.NewFeedCursorRepository(db)
}

// ProvideConsumeRecordRepository 提供 Kafka 消费记录仓储（用于幂等防重复消费）
func ProvideConsumeRecordRepository(db *gorm.DB) repository.ConsumeRecordRepository {
	return repository.NewConsumeRecordRepository(db)
}
