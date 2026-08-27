package service

import (
	"context"
	"errors"
	"testing"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
)

// UserBookshelfServiceTestSuite 书架服务测试套件
type UserBookshelfServiceTestSuite struct {
	suite.Suite
	userRepo      *MockUserRepositoryForService
	bookRepo      *MockBookRepository
	bookshelfRepo *MockUserBookshelfRepository
	bookCache     *MockBookCache
	thermalCache  *MockBookThermalCache
	limitCache    *MockLimitCache
	svc           UserBookshelfService
}

func (suite *UserBookshelfServiceTestSuite) SetupTest() {
	suite.userRepo = new(MockUserRepositoryForService)
	suite.bookRepo = new(MockBookRepository)
	suite.bookshelfRepo = new(MockUserBookshelfRepository)
	suite.bookCache = new(MockBookCache)
	suite.thermalCache = new(MockBookThermalCache)
	suite.limitCache = new(MockLimitCache)

	suite.svc = NewUserBookshelfService(
		suite.userRepo,
		suite.bookRepo,
		suite.bookshelfRepo,
		suite.bookCache,
		suite.thermalCache,
		suite.limitCache,
		newTestDB(suite.T()),
		newTestConfig(),
		newTestPool(),
	)
}

// TestGetUserBookshelfListCacheHit 测试书架列表命中缓存时不查询数据库
func (suite *UserBookshelfServiceTestSuite) TestGetUserBookshelfListCacheHit() {
	ctx := context.Background()
	req := &dto.PageListRequest{Page: 1, Size: 10}

	shelf := []*model.UserBookshelf{{ID: "s1", UserID: "u1", BookID: "b1"}}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.bookshelfRepo.On("ListByUserID", ctx, mock.Anything, "u1", 1, 10).Return(shelf, nil)
	suite.bookCache.On("GetBook", ctx, "b1").Return(book, true, nil)
	suite.bookshelfRepo.On("CountByUserID", ctx, mock.Anything, "u1").Return(1, nil)

	res, err := suite.svc.GetUserBookshelfList(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res.Books, 1)
	assert.Equal(suite.T(), "b1", res.Books[0].ID)
	assert.Equal(suite.T(), 1, res.Count)
	assert.Equal(suite.T(), 1, res.Page)
	assert.Equal(suite.T(), 10, res.Size)

	// 命中缓存时不应该回源
	suite.bookRepo.AssertNotCalled(suite.T(), "GetByID", mock.Anything, mock.Anything)
}

// TestGetUserBookshelfListCacheMiss 测试缓存未命中时回源数据库并回写缓存
func (suite *UserBookshelfServiceTestSuite) TestGetUserBookshelfListCacheMiss() {
	ctx := context.Background()
	req := &dto.PageListRequest{Page: 1, Size: 10}

	shelf := []*model.UserBookshelf{{ID: "s1", UserID: "u1", BookID: "b1"}}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.bookshelfRepo.On("ListByUserID", ctx, mock.Anything, "u1", 1, 10).Return(shelf, nil)
	suite.bookCache.On("GetBook", ctx, "b1").Return(nil, false, nil)
	suite.bookRepo.On("GetByID", ctx, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", ctx, "b1", book).Return(nil)
	suite.bookshelfRepo.On("CountByUserID", ctx, mock.Anything, "u1").Return(1, nil)

	res, err := suite.svc.GetUserBookshelfList(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res.Books, 1)
	suite.bookCache.AssertCalled(suite.T(), "UpdateBook", ctx, "b1", book)
}

// TestGetUserBookshelfListEmpty 测试书架为空时返回空数组而不是 nil
func (suite *UserBookshelfServiceTestSuite) TestGetUserBookshelfListEmpty() {
	ctx := context.Background()
	req := &dto.PageListRequest{Page: 1, Size: 10}

	suite.bookshelfRepo.On("ListByUserID", ctx, mock.Anything, "u1", 1, 10).
		Return([]*model.UserBookshelf{}, nil)
	suite.bookshelfRepo.On("CountByUserID", ctx, mock.Anything, "u1").Return(0, nil)

	res, err := suite.svc.GetUserBookshelfList(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res.Books)
	assert.Len(suite.T(), res.Books, 0)
}

// TestGetUserBookshelfListRepoError 测试列表查询失败时返回错误
func (suite *UserBookshelfServiceTestSuite) TestGetUserBookshelfListRepoError() {
	ctx := context.Background()
	req := &dto.PageListRequest{Page: 1, Size: 10}
	wantErr := errors.New("db down")

	suite.bookshelfRepo.On("ListByUserID", ctx, mock.Anything, "u1", 1, 10).Return(nil, wantErr)

	res, err := suite.svc.GetUserBookshelfList(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, wantErr)
	assert.Nil(suite.T(), res)
}

// TestGetUserBookshelfListCountError 测试统计失败时返回错误
func (suite *UserBookshelfServiceTestSuite) TestGetUserBookshelfListCountError() {
	ctx := context.Background()
	req := &dto.PageListRequest{Page: 1, Size: 10}
	wantErr := errors.New("count fail")

	suite.bookshelfRepo.On("ListByUserID", ctx, mock.Anything, "u1", 1, 10).
		Return([]*model.UserBookshelf{}, nil)
	suite.bookshelfRepo.On("CountByUserID", ctx, mock.Anything, "u1").Return(0, wantErr)

	res, err := suite.svc.GetUserBookshelfList(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, wantErr)
	assert.Nil(suite.T(), res)
}

// TestCreateUserBookshelfError 测试创建失败时直接返回错误且不触发热度任务
func (suite *UserBookshelfServiceTestSuite) TestCreateUserBookshelfError() {
	ctx := context.Background()
	req := &dto.CreateUserBookshelfRequest{BookID: "b1"}
	wantErr := errors.New("insert fail")

	suite.bookshelfRepo.On("CreateByUserIDAndBookID", ctx, mock.Anything, "u1", "b1").Return(wantErr)

	err := suite.svc.CreateUserBookshelf(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, wantErr)
	suite.limitCache.AssertNotCalled(suite.T(), "CheckUserBookshelfBook1Hour", mock.Anything, mock.Anything, mock.Anything)
}

// TestDeleteUserBookshelf 测试删除已存在的书架记录
func (suite *UserBookshelfServiceTestSuite) TestDeleteUserBookshelf() {
	ctx := context.Background()
	req := &dto.DeleteUserBookshelfRequest{BookID: "b1"}

	suite.bookshelfRepo.On("CheckByUserIDAndBookID", ctx, mock.Anything, "u1", "b1").
		Return(true, &model.UserBookshelf{ID: "s1"}, nil)
	suite.bookshelfRepo.On("DeleteByUserIDAndBookID", ctx, mock.Anything, "u1", "b1").Return(nil)

	err := suite.svc.DeleteUserBookshelf(ctx, "u1", req)
	assert.NoError(suite.T(), err)
}

// TestDeleteUserBookshelfNotFound 测试删除不存在的书架记录返回 ErrBookNotFound
func (suite *UserBookshelfServiceTestSuite) TestDeleteUserBookshelfNotFound() {
	ctx := context.Background()
	req := &dto.DeleteUserBookshelfRequest{BookID: "b1"}

	suite.bookshelfRepo.On("CheckByUserIDAndBookID", ctx, mock.Anything, "u1", "b1").
		Return(false, nil, nil)

	err := suite.svc.DeleteUserBookshelf(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, ErrBookNotFound)
	suite.bookshelfRepo.AssertNotCalled(suite.T(), "DeleteByUserIDAndBookID",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestDeleteUserBookshelfCheckError 测试检查阶段出错时不执行删除
func (suite *UserBookshelfServiceTestSuite) TestDeleteUserBookshelfCheckError() {
	ctx := context.Background()
	req := &dto.DeleteUserBookshelfRequest{BookID: "b1"}
	wantErr := errors.New("check fail")

	suite.bookshelfRepo.On("CheckByUserIDAndBookID", ctx, mock.Anything, "u1", "b1").
		Return(false, nil, wantErr)

	err := suite.svc.DeleteUserBookshelf(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, wantErr)
	suite.bookshelfRepo.AssertNotCalled(suite.T(), "DeleteByUserIDAndBookID",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// newBookshelfTaskService 构造一个带指定线程池的书架服务，用于直接驱动异步任务
func (suite *UserBookshelfServiceTestSuite) newBookshelfTaskService(pool *agilePool.Pool) *userBookshelfService {
	return &userBookshelfService{
		userRepo:          suite.userRepo,
		bookRepo:          suite.bookRepo,
		userBookshelfRepo: suite.bookshelfRepo,
		bookCache:         suite.bookCache,
		bookThermalCache:  suite.thermalCache,
		limitCatch:        suite.limitCache,
		db:                newTestDB(suite.T()),
		cfg:               newTestConfig(),
		pool:              pool,
	}
}

// TestBookshelfTaskLimited 测试 1 小时限流命中时不再累加热度
func (suite *UserBookshelfServiceTestSuite) TestBookshelfTaskLimited() {
	ctx := context.Background()
	req := &dto.CreateUserBookshelfRequest{BookID: "b1"}
	pool := newTestPool()

	suite.limitCache.On("CheckUserBookshelfBook1Hour", ctx, "u1", "b1").Return(true, nil)

	err := bookThermalCacheAddByBookshelfTask(ctx, suite.newBookshelfTaskService(pool), "u1", req)
	pool.Wait()

	assert.NoError(suite.T(), err)
	suite.limitCache.AssertNotCalled(suite.T(), "AddUserBookshelfBook1Hour",
		mock.Anything, mock.Anything, mock.Anything)
	suite.thermalCache.AssertNotCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, mock.Anything, mock.Anything)
}

// TestBookshelfTaskAddThermal 测试未限流时写入限流标记并按 10 点热度提交累加任务
func (suite *UserBookshelfServiceTestSuite) TestBookshelfTaskAddThermal() {
	ctx := context.Background()
	req := &dto.CreateUserBookshelfRequest{BookID: "b1"}
	pool := newTestPool()

	suite.limitCache.On("CheckUserBookshelfBook1Hour", ctx, "u1", "b1").Return(false, nil)
	suite.limitCache.On("AddUserBookshelfBook1Hour", ctx, "u1", "b1").Return(nil)
	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).
		Return(cache.ErrorBookThermalValueNotExist)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(nil, nil)

	err := bookThermalCacheAddByBookshelfTask(ctx, suite.newBookshelfTaskService(pool), "u1", req)
	pool.Wait()

	assert.NoError(suite.T(), err)
	suite.limitCache.AssertCalled(suite.T(), "AddUserBookshelfBook1Hour", ctx, "u1", "b1")
	// 热度必须记在书籍上，而不是用户上
	suite.thermalCache.AssertCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, "b1", cache.BookThermalValue(10))
	suite.thermalCache.AssertNotCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, "u1", mock.Anything)
}

func TestUserBookshelfServiceTestSuite(t *testing.T) {
	suite.Run(t, new(UserBookshelfServiceTestSuite))
}

// BookThermalTaskTestSuite 热度值异步任务测试套件，直接测试任务函数而不经过线程池
type BookThermalTaskTestSuite struct {
	suite.Suite
	bookRepo     *MockBookRepository
	bookCache    *MockBookCache
	thermalCache *MockBookThermalCache
	db           *gorm.DB
}

func (suite *BookThermalTaskTestSuite) SetupTest() {
	suite.bookRepo = new(MockBookRepository)
	suite.bookCache = new(MockBookCache)
	suite.thermalCache = new(MockBookThermalCache)
	suite.db = newTestDB(suite.T())
}

// runAddValueTask 执行热度累加任务，并等待其内部提交的异步任务完成
func (suite *BookThermalTaskTestSuite) runAddValueTask(bookID string, value cache.BookThermalValue) error {
	pool := newTestPool()
	err := bookThermalCacheAddValueTask(
		context.Background(), suite.db, suite.thermalCache, suite.bookCache,
		suite.bookRepo, pool, bookID, value,
	)
	pool.Wait()
	return err
}

// TestAddValueCacheHit 测试缓存存在时直接累加，并加锁后触发数据库同步与榜单更新
func (suite *BookThermalTaskTestSuite) TestAddValueCacheHit() {
	book := &model.Book{ID: "b1", Name: "book-1", ThermalValue: 100}

	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).Return(nil)
	suite.thermalCache.On("LockBookThermalValue", mock.Anything, "b1").Return(nil)
	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(110), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110)).Return(nil)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", mock.Anything, "b1", book).Return(nil)
	suite.bookCache.On("IncreaseBookThermalDaily", mock.Anything, "b1", 10).Return(nil)

	err := suite.runAddValueTask("b1", 10)
	assert.NoError(suite.T(), err)

	// 缓存已存在，不应该走 NX 初始化分支
	suite.thermalCache.AssertNotCalled(suite.T(), "UpdateBookThermalValue",
		mock.Anything, mock.Anything, mock.Anything)
	suite.bookRepo.AssertCalled(suite.T(), "UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110))
}

// TestAddValueCacheMissInitFromDB 测试缓存不存在时从数据库热度值加上增量后初始化缓存
func (suite *BookThermalTaskTestSuite) TestAddValueCacheMissInitFromDB() {
	book := &model.Book{ID: "b1", Name: "book-1", ThermalValue: 100}

	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).
		Return(cache.ErrorBookThermalValueNotExist)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(book, nil)
	suite.thermalCache.On("UpdateBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(110)).Return(nil)
	suite.thermalCache.On("LockBookThermalValue", mock.Anything, "b1").Return(nil)
	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(110), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110)).Return(nil)
	suite.bookCache.On("UpdateBook", mock.Anything, "b1", book).Return(nil)
	suite.bookCache.On("IncreaseBookThermalDaily", mock.Anything, "b1", 10).Return(nil)

	err := suite.runAddValueTask("b1", 10)
	assert.NoError(suite.T(), err)
	suite.thermalCache.AssertCalled(suite.T(), "UpdateBookThermalValue",
		mock.Anything, "b1", cache.BookThermalValue(110))
}

// TestAddValueBookNotFound 测试缓存不存在且数据库也查不到书籍时返回 ErrBookNotFound
func (suite *BookThermalTaskTestSuite) TestAddValueBookNotFound() {
	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).
		Return(cache.ErrorBookThermalValueNotExist)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(nil, nil)

	err := suite.runAddValueTask("b1", 10)
	assert.ErrorIs(suite.T(), err, ErrBookNotFound)
	suite.thermalCache.AssertNotCalled(suite.T(), "LockBookThermalValue", mock.Anything, mock.Anything)
}

// TestAddValueCacheError 测试非「缓存不存在」的缓存错误会直接向上返回
func (suite *BookThermalTaskTestSuite) TestAddValueCacheError() {
	wantErr := errors.New("redis down")

	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).Return(wantErr)

	err := suite.runAddValueTask("b1", 10)
	assert.ErrorIs(suite.T(), err, wantErr)
	suite.thermalCache.AssertNotCalled(suite.T(), "LockBookThermalValue", mock.Anything, mock.Anything)
	suite.bookRepo.AssertNotCalled(suite.T(), "UpdateBookThermal",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestAddValueLockExist 测试锁已存在（锁期内已同步过）时跳过数据库同步与榜单更新
func (suite *BookThermalTaskTestSuite) TestAddValueLockExist() {
	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).Return(nil)
	suite.thermalCache.On("LockBookThermalValue", mock.Anything, "b1").
		Return(cache.ErrorBookThermalLockExist)

	err := suite.runAddValueTask("b1", 10)
	assert.ErrorIs(suite.T(), err, cache.ErrorBookThermalLockExist)

	suite.bookRepo.AssertNotCalled(suite.T(), "UpdateBookThermal",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	suite.bookCache.AssertNotCalled(suite.T(), "IncreaseBookThermalDaily",
		mock.Anything, mock.Anything, mock.Anything)
}

// TestAddValueRankingErrorIgnored 测试榜单更新失败只记录日志，不影响任务返回
func (suite *BookThermalTaskTestSuite) TestAddValueRankingErrorIgnored() {
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(10)).Return(nil)
	suite.thermalCache.On("LockBookThermalValue", mock.Anything, "b1").Return(nil)
	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(110), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110)).Return(nil)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", mock.Anything, "b1", book).Return(nil)
	suite.bookCache.On("IncreaseBookThermalDaily", mock.Anything, "b1", 10).
		Return(errors.New("zset fail"))

	err := suite.runAddValueTask("b1", 10)
	assert.NoError(suite.T(), err)
}

// TestSyncToDatabaseCacheMiss 测试缓存缺失时同步任务直接返回，不写数据库
func (suite *BookThermalTaskTestSuite) TestSyncToDatabaseCacheMiss() {
	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(0), false, nil)

	err := bookThermalCacheSyncToDatabaseTask(
		context.Background(), suite.db, suite.thermalCache, suite.bookCache, suite.bookRepo, "b1",
	)
	assert.NoError(suite.T(), err)
	suite.bookRepo.AssertNotCalled(suite.T(), "UpdateBookThermal",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestSyncToDatabaseUpdateError 测试数据库写入失败时返回错误且不刷新书籍缓存
func (suite *BookThermalTaskTestSuite) TestSyncToDatabaseUpdateError() {
	wantErr := errors.New("update fail")

	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(110), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110)).Return(wantErr)

	err := bookThermalCacheSyncToDatabaseTask(
		context.Background(), suite.db, suite.thermalCache, suite.bookCache, suite.bookRepo, "b1",
	)
	assert.ErrorIs(suite.T(), err, wantErr)
	suite.bookCache.AssertNotCalled(suite.T(), "UpdateBook", mock.Anything, mock.Anything)
}

// TestSyncToDatabaseCacheUpdateErrorIgnored 测试书籍缓存刷新失败只记录日志
func (suite *BookThermalTaskTestSuite) TestSyncToDatabaseCacheUpdateErrorIgnored() {
	book := &model.Book{ID: "b1", ThermalValue: 110}

	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(110), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(110)).Return(nil)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", mock.Anything, "b1", book).Return(errors.New("cache fail"))

	err := bookThermalCacheSyncToDatabaseTask(
		context.Background(), suite.db, suite.thermalCache, suite.bookCache, suite.bookRepo, "b1",
	)
	assert.NoError(suite.T(), err)
}

func TestBookThermalTaskTestSuite(t *testing.T) {
	suite.Run(t, new(BookThermalTaskTestSuite))
}
