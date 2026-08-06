package service

import (
	"context"
	"os"
	"testing"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
)

// TestMain 初始化 logger，服务层的异步任务里会直接调用 logger
func TestMain(m *testing.M) {
	if err := logger.Init("debug"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// newTestConfig 构造 toBookResponse 需要的最小配置
func newTestConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Server.Website = "https://example.com"
	return cfg
}

// newTestDB 提供一个内存 sqlite，用于需要真实事务的分支
func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	return db
}

// newTestPool 提供一个真实线程池，测试中通过 Wait 等待异步任务结束
func newTestPool() *agilePool.Pool {
	return agilePool.NewPool(
		agilePool.NewConfig(
			agilePool.WithBlockMode(agilePool.BLOCK),
			agilePool.WithTaskQueueSize(1000),
		),
	)
}

// ---------- BookRepository ----------

type MockBookRepository struct {
	mock.Mock
}

func (m *MockBookRepository) GetByID(ctx context.Context, id string) (*model.Book, error) {
	args := m.Called(ctx, id)
	book, _ := args.Get(0).(*model.Book)
	return book, args.Error(1)
}

func (m *MockBookRepository) GetByOwnerID(ctx context.Context, ownerID string, page, size int) ([]*model.Book, error) {
	args := m.Called(ctx, ownerID, page, size)
	books, _ := args.Get(0).([]*model.Book)
	return books, args.Error(1)
}

func (m *MockBookRepository) GetByName(ctx context.Context, name string) (*model.Book, error) {
	args := m.Called(ctx, name)
	book, _ := args.Get(0).(*model.Book)
	return book, args.Error(1)
}

func (m *MockBookRepository) List(ctx context.Context, page int, size int) ([]*model.Book, error) {
	args := m.Called(ctx, page, size)
	books, _ := args.Get(0).([]*model.Book)
	return books, args.Error(1)
}

func (m *MockBookRepository) ListWithOption(ctx context.Context, opt repository.BookListOption) ([]*model.Book, error) {
	args := m.Called(ctx, opt)
	books, _ := args.Get(0).([]*model.Book)
	return books, args.Error(1)
}

func (m *MockBookRepository) Create(ctx context.Context, db *gorm.DB, in *model.Book) error {
	return m.Called(ctx, db, in).Error(0)
}

func (m *MockBookRepository) Update(ctx context.Context, db *gorm.DB, in *model.Book) error {
	return m.Called(ctx, db, in).Error(0)
}

func (m *MockBookRepository) UpdateBookThermal(ctx context.Context, db *gorm.DB, id string, value uint64) error {
	return m.Called(ctx, db, id, value).Error(0)
}

func (m *MockBookRepository) Delete(ctx context.Context, db *gorm.DB, id string) error {
	return m.Called(ctx, db, id).Error(0)
}

func (m *MockBookRepository) Count(ctx context.Context) (int, error) {
	args := m.Called(ctx)
	return args.Int(0), args.Error(1)
}

func (m *MockBookRepository) CountByOwner(ctx context.Context, ownerID string) (int, error) {
	args := m.Called(ctx, ownerID)
	return args.Int(0), args.Error(1)
}

func (m *MockBookRepository) IncreaseOfBookShelf(ctx context.Context, db *gorm.DB, shelfID string) (int, error) {
	args := m.Called(ctx, db, shelfID)
	return args.Int(0), args.Error(1)
}

func (m *MockBookRepository) ReduceOfBookShelf(ctx context.Context, db *gorm.DB, shelfID string) (int, error) {
	args := m.Called(ctx, db, shelfID)
	return args.Int(0), args.Error(1)
}

func (m *MockBookRepository) IncreaseOfBrowseAmount(ctx context.Context, db *gorm.DB, shelfID string) (int, error) {
	args := m.Called(ctx, db, shelfID)
	return args.Int(0), args.Error(1)
}

// ---------- UserRepository ----------

type MockUserRepositoryForService struct {
	mock.Mock
}

func (m *MockUserRepositoryForService) Create(ctx context.Context, db *gorm.DB, user *model.User) error {
	return m.Called(ctx, db, user).Error(0)
}

func (m *MockUserRepositoryForService) GetByID(ctx context.Context, db *gorm.DB, id string) (*model.User, error) {
	args := m.Called(ctx, db, id)
	user, _ := args.Get(0).(*model.User)
	return user, args.Error(1)
}

func (m *MockUserRepositoryForService) GetByUsername(ctx context.Context, db *gorm.DB, username string) (*model.User, error) {
	args := m.Called(ctx, db, username)
	user, _ := args.Get(0).(*model.User)
	return user, args.Error(1)
}

func (m *MockUserRepositoryForService) GetByEmail(ctx context.Context, db *gorm.DB, email string) (*model.User, error) {
	args := m.Called(ctx, db, email)
	user, _ := args.Get(0).(*model.User)
	return user, args.Error(1)
}

func (m *MockUserRepositoryForService) Update(ctx context.Context, db *gorm.DB, user *model.User) error {
	return m.Called(ctx, db, user).Error(0)
}

func (m *MockUserRepositoryForService) Delete(ctx context.Context, db *gorm.DB, id string) error {
	return m.Called(ctx, db, id).Error(0)
}

func (m *MockUserRepositoryForService) List(ctx context.Context, db *gorm.DB, offset, limit int) ([]*model.User, error) {
	args := m.Called(ctx, db, offset, limit)
	users, _ := args.Get(0).([]*model.User)
	return users, args.Error(1)
}

// ---------- UserBookshelfRepository ----------

type MockUserBookshelfRepository struct {
	mock.Mock
}

func (m *MockUserBookshelfRepository) ListByUserID(ctx context.Context, db *gorm.DB, userID string, page int, size int) ([]*model.UserBookshelf, error) {
	args := m.Called(ctx, db, userID, page, size)
	list, _ := args.Get(0).([]*model.UserBookshelf)
	return list, args.Error(1)
}

func (m *MockUserBookshelfRepository) CreateByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error {
	return m.Called(ctx, db, userID, bookID).Error(0)
}

func (m *MockUserBookshelfRepository) DeleteByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) error {
	return m.Called(ctx, db, userID, bookID).Error(0)
}

func (m *MockUserBookshelfRepository) CheckByUserIDAndBookID(ctx context.Context, db *gorm.DB, userID string, bookID string) (bool, *model.UserBookshelf, error) {
	args := m.Called(ctx, db, userID, bookID)
	item, _ := args.Get(1).(*model.UserBookshelf)
	return args.Bool(0), item, args.Error(2)
}

func (m *MockUserBookshelfRepository) CountByUserID(ctx context.Context, db *gorm.DB, userID string) (int, error) {
	args := m.Called(ctx, db, userID)
	return args.Int(0), args.Error(1)
}

// ---------- BookCache ----------

type MockBookCache struct {
	mock.Mock
}

func (m *MockBookCache) GetBook(ctx context.Context, id string) (*model.Book, bool, error) {
	args := m.Called(ctx, id)
	book, _ := args.Get(0).(*model.Book)
	return book, args.Bool(1), args.Error(2)
}

func (m *MockBookCache) UpdateBook(ctx context.Context, book *model.Book) error {
	return m.Called(ctx, book).Error(0)
}

func (m *MockBookCache) DeleteBook(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBookCache) GetBookThermalDaily(ctx context.Context, page int, size int) ([]string, error) {
	args := m.Called(ctx, page, size)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *MockBookCache) IncreaseBookThermalDaily(ctx context.Context, bookId string, score int) error {
	return m.Called(ctx, bookId, score).Error(0)
}

func (m *MockBookCache) GetBookThermalWeekly(ctx context.Context, page int, size int) ([]string, error) {
	args := m.Called(ctx, page, size)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *MockBookCache) GetBookThermalMonthly(ctx context.Context, page int, size int) ([]string, error) {
	args := m.Called(ctx, page, size)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

// ---------- BookThermalCache ----------

type MockBookThermalCache struct {
	mock.Mock
}

func (m *MockBookThermalCache) GetBookThermalValue(ctx context.Context, bookId string) (uint64, bool, error) {
	args := m.Called(ctx, bookId)
	value, _ := args.Get(0).(uint64)
	return value, args.Bool(1), args.Error(2)
}

func (m *MockBookThermalCache) UpdateBookThermalValue(ctx context.Context, bookId string, value uint64) error {
	return m.Called(ctx, bookId, value).Error(0)
}

func (m *MockBookThermalCache) AddBookThermalValue(ctx context.Context, bookId string, value uint64) error {
	return m.Called(ctx, bookId, value).Error(0)
}

func (m *MockBookThermalCache) LockBookThermalValue(ctx context.Context, bookId string) error {
	return m.Called(ctx, bookId).Error(0)
}

// ---------- LimitCache ----------

type MockLimitCache struct {
	mock.Mock
}

func (m *MockLimitCache) CheckUserBrowse1Min(ctx context.Context, userId string) (bool, error) {
	args := m.Called(ctx, userId)
	return args.Bool(0), args.Error(1)
}

func (m *MockLimitCache) AddUserBrowse1Min(ctx context.Context, userId string) error {
	return m.Called(ctx, userId).Error(0)
}

func (m *MockLimitCache) CheckUserBrowseBook5Min(ctx context.Context, userId string, bookId string) (bool, error) {
	args := m.Called(ctx, userId, bookId)
	return args.Bool(0), args.Error(1)
}

func (m *MockLimitCache) AddUserBrowseBook5Min(ctx context.Context, userId string, bookId string) error {
	return m.Called(ctx, userId, bookId).Error(0)
}

func (m *MockLimitCache) CheckUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) (bool, error) {
	args := m.Called(ctx, userId, bookId)
	return args.Bool(0), args.Error(1)
}

func (m *MockLimitCache) AddUserBookshelfBook1Hour(ctx context.Context, userId string, bookId string) error {
	return m.Called(ctx, userId, bookId).Error(0)
}

// ---------- UserStatCache ----------

type MockUserStatCache struct {
	mock.Mock
}

func (m *MockUserStatCache) AddReadBook(ctx context.Context, userId string, bookId string) (bool, error) {
	args := m.Called(ctx, userId, bookId)
	return args.Bool(0), args.Error(1)
}

func (m *MockUserStatCache) GetReadBookNum(ctx context.Context, userId string) (int64, bool, error) {
	args := m.Called(ctx, userId)
	num, _ := args.Get(0).(int64)
	return num, args.Bool(1), args.Error(2)
}

func (m *MockUserStatCache) IncreaseReadWordsNum(ctx context.Context, userId string, increaseNum int64) error {
	return m.Called(ctx, userId, increaseNum).Error(0)
}

func (m *MockUserStatCache) GetReadWordsNum(ctx context.Context, userId string) (int64, bool, error) {
	args := m.Called(ctx, userId)
	num, _ := args.Get(0).(int64)
	return num, args.Bool(1), args.Error(2)
}

func (m *MockUserStatCache) GetReadBookList(ctx context.Context, userId string) ([]string, error) {
	args := m.Called(ctx, userId)
	ids, _ := args.Get(0).([]string)
	return ids, args.Error(1)
}

func (m *MockUserStatCache) CountReadBookList(ctx context.Context, userId string) (int64, error) {
	args := m.Called(ctx, userId)
	num, _ := args.Get(0).(int64)
	return num, args.Error(1)
}

// ---------- BookChapterRepository ----------

type MockBookChapterRepository struct {
	mock.Mock
}

func (m *MockBookChapterRepository) GetByID(ctx context.Context, id string) (*model.BookChapter, error) {
	args := m.Called(ctx, id)
	chapter, _ := args.Get(0).(*model.BookChapter)
	return chapter, args.Error(1)
}

func (m *MockBookChapterRepository) GetByBookIDAndOrder(ctx context.Context, id string, order int) (*model.BookChapter, error) {
	args := m.Called(ctx, id, order)
	chapter, _ := args.Get(0).(*model.BookChapter)
	return chapter, args.Error(1)
}

func (m *MockBookChapterRepository) GetIDListByBookID(ctx context.Context, id string) (*[]model.BookChapterInfo, error) {
	args := m.Called(ctx, id)
	list, _ := args.Get(0).(*[]model.BookChapterInfo)
	return list, args.Error(1)
}

func (m *MockBookChapterRepository) AddByBookID(ctx context.Context, bookChapter *model.BookChapter) (*model.BookChapter, error) {
	args := m.Called(ctx, bookChapter)
	chapter, _ := args.Get(0).(*model.BookChapter)
	return chapter, args.Error(1)
}

func (m *MockBookChapterRepository) CountByBookID(ctx context.Context, bookID string) (int, error) {
	args := m.Called(ctx, bookID)
	return args.Int(0), args.Error(1)
}

func (m *MockBookChapterRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBookChapterRepository) Update(ctx context.Context, id string, bookChapter *model.BookChapter) error {
	return m.Called(ctx, id, bookChapter).Error(0)
}

// ---------- BookChapterESRepository ----------

type MockBookChapterESRepository struct {
	mock.Mock
}

func (m *MockBookChapterESRepository) Create(ctx context.Context, chapter *model.BookChapter) error {
	return m.Called(ctx, chapter).Error(0)
}

func (m *MockBookChapterESRepository) Update(ctx context.Context, chapter *model.BookChapter) error {
	return m.Called(ctx, chapter).Error(0)
}

func (m *MockBookChapterESRepository) Delete(ctx context.Context, id string) error {
	return m.Called(ctx, id).Error(0)
}

func (m *MockBookChapterESRepository) Search(ctx context.Context, filter model.BookChapterFilterWithIDPage) ([]model.BookChapterSearchResult, error) {
	args := m.Called(ctx, filter)
	results, _ := args.Get(0).([]model.BookChapterSearchResult)
	return results, args.Error(1)
}

// ---------- UserPurchasedChapterRepository ----------

type MockUserPurchasedChapterRepository struct {
	mock.Mock
}

func (m *MockUserPurchasedChapterRepository) GetListByUserID(ctx context.Context, userID string) (*[]model.UserPurchasedChapter, error) {
	args := m.Called(ctx, userID)
	list, _ := args.Get(0).(*[]model.UserPurchasedChapter)
	return list, args.Error(1)
}

func (m *MockUserPurchasedChapterRepository) CheckPurchasedChapter(ctx context.Context, userID, chapterID string) (bool, *model.UserPurchasedChapter, error) {
	args := m.Called(ctx, userID, chapterID)
	item, _ := args.Get(1).(*model.UserPurchasedChapter)
	return args.Bool(0), item, args.Error(2)
}

func (m *MockUserPurchasedChapterRepository) AddPurchasedChapter(ctx context.Context, in *model.UserPurchasedChapter) error {
	return m.Called(ctx, in).Error(0)
}

func (m *MockUserPurchasedChapterRepository) GetByChapterIDAndUserID(ctx context.Context, chapterID string, userID string) (*model.UserPurchasedChapter, error) {
	args := m.Called(ctx, chapterID, userID)
	item, _ := args.Get(0).(*model.UserPurchasedChapter)
	return item, args.Error(1)
}

// ---------- UserAssetRepository ----------

type MockUserAssetRepository struct {
	mock.Mock
}

func (m *MockUserAssetRepository) GetByID(ctx context.Context, db *gorm.DB, id string) (*model.UserAsset, error) {
	args := m.Called(ctx, db, id)
	asset, _ := args.Get(0).(*model.UserAsset)
	return asset, args.Error(1)
}

func (m *MockUserAssetRepository) UpdateByID(ctx context.Context, db *gorm.DB, id string, data *model.UserAsset) error {
	return m.Called(ctx, db, id, data).Error(0)
}

func (m *MockUserAssetRepository) AddGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error) {
	args := m.Called(ctx, db, id, gold)
	asset, _ := args.Get(0).(*model.UserAsset)
	return asset, args.Error(1)
}

func (m *MockUserAssetRepository) SubGoldByID(ctx context.Context, db *gorm.DB, id string, gold int64) (*model.UserAsset, error) {
	args := m.Called(ctx, db, id, gold)
	asset, _ := args.Get(0).(*model.UserAsset)
	return asset, args.Error(1)
}

func (m *MockUserAssetRepository) GetGoldByID(ctx context.Context, db *gorm.DB, id string) (int64, error) {
	args := m.Called(ctx, db, id)
	gold, _ := args.Get(0).(int64)
	return gold, args.Error(1)
}

func (m *MockUserAssetRepository) CreateUserAsset(ctx context.Context, db *gorm.DB, id string) error {
	return m.Called(ctx, db, id).Error(0)
}

// ---------- OutboxRepository ----------

type MockOutboxRepository struct {
	mock.Mock
}

func (m *MockOutboxRepository) AddEvent(ctx context.Context, db *gorm.DB, outbox *model.Outbox) error {
	return m.Called(ctx, db, outbox).Error(0)
}

func (m *MockOutboxRepository) GetEventByTableName(ctx context.Context, db *gorm.DB, table interface{}) (*model.Outbox, error) {
	args := m.Called(ctx, db, table)
	outbox, _ := args.Get(0).(*model.Outbox)
	return outbox, args.Error(1)
}

func (m *MockOutboxRepository) FinishEventByTableNameAndID(ctx context.Context, db *gorm.DB, table interface{}, id string) error {
	return m.Called(ctx, db, table, id).Error(0)
}

// ---------- BookChapterCache ----------

type MockBookChapterCache struct {
	mock.Mock
}

func (m *MockBookChapterCache) UpdateBookChapter(ctx context.Context, chapter *model.BookChapter) error {
	return m.Called(ctx, chapter).Error(0)
}

func (m *MockBookChapterCache) GetBookChapter(ctx context.Context, chapter string) (*model.BookChapter, bool, error) {
	args := m.Called(ctx, chapter)
	item, _ := args.Get(0).(*model.BookChapter)
	return item, args.Bool(1), args.Error(2)
}
