package service

import (
	"context"
	"errors"
	"testing"
	"time"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/pkg/response"
)

// TestVIPStrToLevel 测试会员等级字符串到数值的映射
func TestVIPStrToLevel(t *testing.T) {
	cases := map[string]int{
		"":        -1,
		"vip_0":   0,
		"vip_1":   1,
		"vip_2":   2,
		"vip_3":   3,
		"off":     999,
		"unknown": 0,
	}
	for in, want := range cases {
		assert.Equal(t, want, VIPStrToLevel(in), "VIPStrToLevel(%q)", in)
	}
}

// TestVIPLevelToStr 测试会员等级数值到字符串的映射
func TestVIPLevelToStr(t *testing.T) {
	cases := map[int]string{
		-1:  "",
		0:   "vip_0",
		1:   "vip_1",
		2:   "vip_2",
		3:   "vip_3",
		4:   "off",
		999: "off",
	}
	for in, want := range cases {
		assert.Equal(t, want, VIPLevelToStr(in), "VIPLevelToStr(%d)", in)
	}
}

// TestVIPLevelRoundTrip 测试合法等级字符串经过两次转换后保持不变
func TestVIPLevelRoundTrip(t *testing.T) {
	for _, in := range []string{"", "vip_0", "vip_1", "vip_2", "vip_3", "off"} {
		assert.Equal(t, in, VIPLevelToStr(VIPStrToLevel(in)))
	}
}

// BookChapterServiceTestSuite 章节服务测试套件
type BookChapterServiceTestSuite struct {
	suite.Suite
	userRepo      *MockUserRepositoryForService
	bookRepo      *MockBookRepository
	chapterRepo   *MockBookChapterRepository
	esRepo        *MockBookChapterESRepository
	purchasedRepo *MockUserPurchasedChapterRepository
	assetRepo     *MockUserAssetRepository
	bookshelfRepo *MockUserBookshelfRepository
	outboxRepo    *MockOutboxRepository
	bookCache     *MockBookCache
	thermalCache  *MockBookThermalCache
	chapterCache  *MockBookChapterCache
	userStatCache *MockUserStatCache
	limitCache    *MockLimitCache
	feedRepo      *MockFeedRepository
	feedCache     *MockFeedCache
	feedClient    *MockFeedClient
	db            *gorm.DB
	pool          *agilePool.Pool
	svc           BookChapterService
}

func (suite *BookChapterServiceTestSuite) SetupTest() {
	suite.userRepo = new(MockUserRepositoryForService)
	suite.bookRepo = new(MockBookRepository)
	suite.chapterRepo = new(MockBookChapterRepository)
	suite.esRepo = new(MockBookChapterESRepository)
	suite.purchasedRepo = new(MockUserPurchasedChapterRepository)
	suite.assetRepo = new(MockUserAssetRepository)
	suite.bookshelfRepo = new(MockUserBookshelfRepository)
	suite.outboxRepo = new(MockOutboxRepository)
	suite.bookCache = new(MockBookCache)
	suite.thermalCache = new(MockBookThermalCache)
	suite.chapterCache = new(MockBookChapterCache)
	suite.userStatCache = new(MockUserStatCache)
	suite.limitCache = new(MockLimitCache)
	suite.feedRepo = new(MockFeedRepository)
	suite.feedCache = new(MockFeedCache)
	suite.feedClient = new(MockFeedClient)
	suite.db = newTestDB(suite.T())
	suite.pool = newTestPool()

	suite.svc = NewBookChapterService(
		suite.userRepo, suite.bookRepo, suite.chapterRepo, suite.esRepo,
		suite.purchasedRepo, suite.assetRepo, suite.bookshelfRepo,
		suite.feedRepo, suite.outboxRepo,
		suite.bookCache, suite.thermalCache, suite.chapterCache,
		suite.userStatCache, suite.limitCache, suite.feedCache, suite.feedClient,
		suite.db, newTestConfig(), suite.pool,
	)
}

// chapterSvc 返回具体类型，用于直接调用非接口方法与内部任务函数
func (suite *BookChapterServiceTestSuite) chapterSvc() *bookChapterService {
	return &bookChapterService{
		userRepo:                 suite.userRepo,
		bookRepo:                 suite.bookRepo,
		bookChapterRepo:          suite.chapterRepo,
		bookChapterESRepo:        suite.esRepo,
		userPurchasedChapterRepo: suite.purchasedRepo,
		userAssetRepo:            suite.assetRepo,
		userBookshelfRepo:        suite.bookshelfRepo,
		feedRepo:                 suite.feedRepo,
		outboxRepo:               suite.outboxRepo,
		bookCache:                suite.bookCache,
		bookThermalCache:         suite.thermalCache,
		bookChapterCache:         suite.chapterCache,
		userStatCache:            suite.userStatCache,
		limitCache:               suite.limitCache,
		feedCache:                suite.feedCache,
		feedClient:               suite.feedClient,
		db:                       suite.db,
		cfg:                      newTestConfig(),
		pool:                     suite.pool,
	}
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }
func i64Ptr(i int64) *int64   { return &i }

// mockChapterCacheHit 让章节信息与章节内容都命中缓存
func (suite *BookChapterServiceTestSuite) mockChapterCacheHit(ctx context.Context, chapterID string, info *model.BookChapterInfo, content *model.BookChapterContent) {
	suite.chapterCache.On("GetBookChapterInfo", ctx, chapterID).Return(info, true, nil)
	suite.chapterCache.On("GetBookChapterContent", ctx, chapterID).Return(content, true, nil)
}

// mockChapterCacheMissNotFound 让章节信息缓存未命中且数据库也不存在
func (suite *BookChapterServiceTestSuite) mockChapterCacheMissNotFound(ctx context.Context, chapterID string) {
	suite.chapterCache.On("GetBookChapterInfo", ctx, chapterID).Return(nil, false, nil)
	suite.chapterCache.On("LockBookChapterInfo", ctx, chapterID).Return("token", true, nil)
	suite.chapterRepo.On("BookChapterInfoGetByID", ctx, suite.db, chapterID).Return(nil, nil)
	suite.chapterCache.On("UpdateBookChapterInfo", ctx, chapterID, (*model.BookChapterInfo)(nil)).Return(nil)
	suite.chapterCache.On("UnlockBookChapterInfo", ctx, chapterID, "token").Return(nil)
}

// mockBookCacheHit 让书籍命中缓存
func (suite *BookChapterServiceTestSuite) mockBookCacheHit(ctx context.Context, bookID string, book *model.Book) {
	suite.bookCache.On("GetBook", ctx, bookID).Return(book, true, nil)
}

// mockBookCacheMissNotFound 让书籍缓存未命中且数据库也不存在
func (suite *BookChapterServiceTestSuite) mockBookCacheMissNotFound(ctx context.Context, bookID string) {
	suite.bookCache.On("GetBook", ctx, bookID).Return(nil, false, nil)
	suite.bookCache.On("LockBook", ctx, bookID).Return("token", true, nil)
	suite.bookRepo.On("GetByID", ctx, bookID).Return(nil, nil)
	suite.bookCache.On("UpdateBook", ctx, bookID, (*model.Book)(nil)).Return(nil)
	suite.bookCache.On("UnlockBook", ctx, bookID, "token").Return(nil)
}

// ---------- GuestGetChapter ----------

// TestGuestGetChapterFree 测试游客可以读取无会员要求的章节
func (suite *BookChapterServiceTestSuite) TestGuestGetChapterFree() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "第一章"}
	chapterContent := &model.BookChapterContent{ID: "c1", Content: "内容"}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)

	res, err := suite.svc.GuestGetChapter(ctx, "c1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "c1", res.ChapterID)
	assert.Equal(suite.T(), "内容", res.Content)
	assert.Equal(suite.T(), "b1", res.Book.ID)
}

// TestGuestGetChapterVIPRequired 测试游客读取会员章节返回未登录错误
func (suite *BookChapterServiceTestSuite) TestGuestGetChapterVIPRequired() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", VIPLevel: "vip_1"}
	chapterContent := &model.BookChapterContent{ID: "c1"}
	book := &model.Book{ID: "b1"}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)

	res, err := suite.svc.GuestGetChapter(ctx, "c1")
	assert.ErrorIs(suite.T(), err, response.ErrUserNotLogin)
	assert.Nil(suite.T(), res)
}

// TestGuestGetChapterNotFound 测试章节不存在
func (suite *BookChapterServiceTestSuite) TestGuestGetChapterNotFound() {
	ctx := context.Background()
	suite.mockChapterCacheMissNotFound(ctx, "c1")

	res, err := suite.svc.GuestGetChapter(ctx, "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookChapterNotFound)
	assert.Nil(suite.T(), res)
}

// TestGuestGetChapterBookNotFound 测试章节存在但归属书籍不存在
func (suite *BookChapterServiceTestSuite) TestGuestGetChapterBookNotFound() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1"}
	chapterContent := &model.BookChapterContent{ID: "c1"}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheMissNotFound(ctx, "b1")

	res, err := suite.svc.GuestGetChapter(ctx, "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookNotFound)
	assert.Nil(suite.T(), res)
}

// ---------- GetChapter ----------

// TestGetChapterByVIPLevel 测试未购买但会员等级足够时可读，并提交热度任务
func (suite *BookChapterServiceTestSuite) TestGetChapterByVIPLevel() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", VIPLevel: "vip_1"}
	chapterContent := &model.BookChapterContent{ID: "c1", Content: "内容"}
	book := &model.Book{ID: "b1"}

	suite.userRepo.On("GetByID", ctx, suite.db, "u1").Return(&model.User{ID: "u1"}, nil)
	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").Return(nil, nil)
	suite.assetRepo.On("GetByID", ctx, suite.db, "u1").
		Return(&model.UserAsset{ID: "u1", VIPLevel: "vip_2"}, nil)

	// 异步热度任务：命中 1min 限流即结束
	suite.userStatCache.On("AddReadBook", mock.Anything, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", mock.Anything, "u1").Return(true, nil)

	res, err := suite.svc.GetChapter(ctx, "u1", "c1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "c1", res.ChapterID)
	suite.pool.Wait()
}

// TestGetChapterVIPInsufficient 测试未购买且会员等级不足时返回 response.ErrVIPLevel
func (suite *BookChapterServiceTestSuite) TestGetChapterVIPInsufficient() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", VIPLevel: "vip_3"}
	chapterContent := &model.BookChapterContent{ID: "c1"}
	book := &model.Book{ID: "b1"}

	suite.userRepo.On("GetByID", ctx, suite.db, "u1").Return(&model.User{ID: "u1"}, nil)
	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").Return(nil, nil)
	suite.assetRepo.On("GetByID", ctx, suite.db, "u1").
		Return(&model.UserAsset{ID: "u1", VIPLevel: "vip_1"}, nil)

	res, err := suite.svc.GetChapter(ctx, "u1", "c1")
	assert.ErrorIs(suite.T(), err, response.ErrVIPLevel)
	assert.Nil(suite.T(), res)
	// 无权访问时不应该产生热度
	suite.userStatCache.AssertNotCalled(suite.T(), "AddReadBook",
		mock.Anything, mock.Anything, mock.Anything)
}

// TestGetChapterPurchasedSkipVIPCheck 测试已购买章节时跳过会员等级校验
func (suite *BookChapterServiceTestSuite) TestGetChapterPurchasedSkipVIPCheck() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", VIPLevel: "vip_3"}
	chapterContent := &model.BookChapterContent{ID: "c1"}
	book := &model.Book{ID: "b1"}

	suite.userRepo.On("GetByID", ctx, suite.db, "u1").Return(&model.User{ID: "u1"}, nil)
	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").
		Return(&model.UserPurchasedChapter{ID: "p1"}, nil)

	suite.userStatCache.On("AddReadBook", mock.Anything, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", mock.Anything, "u1").Return(true, nil)

	res, err := suite.svc.GetChapter(ctx, "u1", "c1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "c1", res.ChapterID)
	suite.assetRepo.AssertNotCalled(suite.T(), "GetByID", mock.Anything, mock.Anything, mock.Anything)
	suite.pool.Wait()
}

// TestGetChapterUserNotFound 测试用户不存在
func (suite *BookChapterServiceTestSuite) TestGetChapterUserNotFound() {
	ctx := context.Background()
	suite.userRepo.On("GetByID", ctx, suite.db, "u1").Return(nil, nil)

	res, err := suite.svc.GetChapter(ctx, "u1", "c1")
	assert.ErrorIs(suite.T(), err, response.ErrUserNotFound)
	assert.Nil(suite.T(), res)
}

// TestGetChapterChapterNotFound 测试章节不存在
func (suite *BookChapterServiceTestSuite) TestGetChapterChapterNotFound() {
	ctx := context.Background()
	suite.userRepo.On("GetByID", ctx, suite.db, "u1").Return(&model.User{ID: "u1"}, nil)
	suite.mockChapterCacheMissNotFound(ctx, "c1")

	res, err := suite.svc.GetChapter(ctx, "u1", "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookChapterNotFound)
	assert.Nil(suite.T(), res)
}

// ---------- 阅读热度任务 ----------

// TestReadTaskLimitedBy1Min 测试 1min 阅读限流命中时不统计字数、不累加热度
func (suite *BookChapterServiceTestSuite) TestReadTaskLimitedBy1Min() {
	ctx := context.Background()
	book := &model.Book{ID: "b1"}
	chapter := &model.BookChapter{ID: "c1", BookID: "b1", Content: "hello world"}

	suite.userStatCache.On("AddReadBook", ctx, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", ctx, "u1").Return(true, nil)

	err := bookThermalCacheAddByReadTask(ctx, suite.chapterSvc(), "u1", book, chapter)
	suite.pool.Wait()

	assert.NoError(suite.T(), err)
	suite.userStatCache.AssertNotCalled(suite.T(), "IncreaseReadWordsNum",
		mock.Anything, mock.Anything, mock.Anything)
	suite.thermalCache.AssertNotCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, mock.Anything, mock.Anything)
}

// TestReadTaskLimitedBy5Min 测试 5min 书籍限流命中时仍统计字数，但不累加书籍热度
func (suite *BookChapterServiceTestSuite) TestReadTaskLimitedBy5Min() {
	ctx := context.Background()
	book := &model.Book{ID: "b1"}
	chapter := &model.BookChapter{ID: "c1", BookID: "b1", Content: "hello world"}

	suite.userStatCache.On("AddReadBook", ctx, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", ctx, "u1").Return(false, nil)
	suite.userStatCache.On("IncreaseReadWordsNum", ctx, "u1", mock.AnythingOfType("int64")).Return(nil)
	suite.limitCache.On("CheckUserBrowseBook5Min", ctx, "u1", "b1").Return(true, nil)

	err := bookThermalCacheAddByReadTask(ctx, suite.chapterSvc(), "u1", book, chapter)
	suite.pool.Wait()

	assert.NoError(suite.T(), err)
	suite.userStatCache.AssertCalled(suite.T(), "IncreaseReadWordsNum",
		ctx, "u1", mock.AnythingOfType("int64"))
	suite.bookCache.AssertNotCalled(suite.T(), "IncreaseBookThermalDaily",
		mock.Anything, mock.Anything, mock.Anything)
	suite.bookRepo.AssertNotCalled(suite.T(), "IncreaseOfBrowseAmount",
		mock.Anything, mock.Anything, mock.Anything)
}

// TestReadTaskFullFlow 测试未限流时写入限流标记、榜单、浏览量并提交热度累加任务
func (suite *BookChapterServiceTestSuite) TestReadTaskFullFlow() {
	ctx := context.Background()
	book := &model.Book{ID: "b1", ThermalValue: 50}
	chapter := &model.BookChapter{ID: "c1", BookID: "b1", Content: "hello world"}

	suite.userStatCache.On("AddReadBook", ctx, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", ctx, "u1").Return(false, nil)
	suite.userStatCache.On("IncreaseReadWordsNum", ctx, "u1", mock.AnythingOfType("int64")).Return(nil)
	suite.limitCache.On("CheckUserBrowseBook5Min", ctx, "u1", "b1").Return(false, nil)
	suite.limitCache.On("AddUserBrowseBook5Min", ctx, "u1", "b1").Return(nil)
	suite.limitCache.On("AddUserBrowse1Min", ctx, "u1").Return(nil)
	suite.bookCache.On("IncreaseBookThermalDaily", mock.Anything, "b1", 1).Return(nil)
	suite.bookRepo.On("IncreaseOfBrowseAmount", ctx, suite.db, "b1").Return(1, nil)

	// 异步热度累加任务：缓存已存在，走累加 + 加锁 + 同步
	suite.thermalCache.On("AddBookThermalValue", mock.Anything, "b1", cache.BookThermalValue(1)).Return(nil)
	suite.thermalCache.On("LockBookThermalValue", mock.Anything, "b1").Return(nil)
	suite.thermalCache.On("GetBookThermalValue", mock.Anything, "b1").Return(uint64(51), true, nil)
	suite.bookRepo.On("UpdateBookThermal", mock.Anything, mock.Anything, "b1", uint64(51)).Return(nil)
	suite.bookRepo.On("GetByID", mock.Anything, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", mock.Anything, "b1", book).Return(nil)

	err := bookThermalCacheAddByReadTask(ctx, suite.chapterSvc(), "u1", book, chapter)
	suite.pool.Wait()

	assert.NoError(suite.T(), err)
	suite.limitCache.AssertCalled(suite.T(), "AddUserBrowseBook5Min", ctx, "u1", "b1")
	suite.limitCache.AssertCalled(suite.T(), "AddUserBrowse1Min", ctx, "u1")
	suite.thermalCache.AssertCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, "b1", cache.BookThermalValue(1))
}

// TestReadTaskBrowseAmountError 测试浏览量自增失败时任务返回错误且不提交热度累加
func (suite *BookChapterServiceTestSuite) TestReadTaskBrowseAmountError() {
	ctx := context.Background()
	book := &model.Book{ID: "b1"}
	chapter := &model.BookChapter{ID: "c1", BookID: "b1", Content: "hello"}
	wantErr := errors.New("browse fail")

	suite.userStatCache.On("AddReadBook", ctx, "u1", "b1").Return(true, nil)
	suite.limitCache.On("CheckUserBrowse1Min", ctx, "u1").Return(false, nil)
	suite.userStatCache.On("IncreaseReadWordsNum", ctx, "u1", mock.AnythingOfType("int64")).Return(nil)
	suite.limitCache.On("CheckUserBrowseBook5Min", ctx, "u1", "b1").Return(false, nil)
	suite.limitCache.On("AddUserBrowseBook5Min", ctx, "u1", "b1").Return(nil)
	suite.limitCache.On("AddUserBrowse1Min", ctx, "u1").Return(nil)
	suite.bookCache.On("IncreaseBookThermalDaily", mock.Anything, "b1", 1).Return(nil)
	suite.bookRepo.On("IncreaseOfBrowseAmount", ctx, suite.db, "b1").Return(0, wantErr)

	err := bookThermalCacheAddByReadTask(ctx, suite.chapterSvc(), "u1", book, chapter)
	suite.pool.Wait()

	assert.ErrorIs(suite.T(), err, wantErr)
	suite.thermalCache.AssertNotCalled(suite.T(), "AddBookThermalValue",
		mock.Anything, mock.Anything, mock.Anything)
}

// ---------- AddByBookID ----------

// TestAddByBookIDPermissionDenied 测试非书籍作者添加章节被拒绝
func (suite *BookChapterServiceTestSuite) TestAddByBookIDPermissionDenied() {
	ctx := context.Background()
	req := &dto.CreatBookChapterRequest{
		BookID: "b1", Title: strPtr("第一章"), Content: strPtr("内容"),
		VIPLevel: strPtr("vip_0"), Price: intPtr(0),
	}

	suite.bookRepo.On("GetByID", ctx, "b1").Return(&model.Book{ID: "b1", OwnerID: "other"}, nil)

	res, err := suite.svc.AddByBookID(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, response.ErrPermission)
	assert.Nil(suite.T(), res)
	suite.chapterRepo.AssertNotCalled(suite.T(), "AddByBookID", mock.Anything, mock.Anything)
}

// TestAddByBookIDBookNotFound 测试书籍不存在
func (suite *BookChapterServiceTestSuite) TestAddByBookIDBookNotFound() {
	ctx := context.Background()
	req := &dto.CreatBookChapterRequest{
		BookID: "b1", Title: strPtr("第一章"), Content: strPtr("内容"),
		VIPLevel: strPtr("vip_0"), Price: intPtr(0),
	}

	suite.bookRepo.On("GetByID", ctx, "b1").Return(nil, nil)

	res, err := suite.svc.AddByBookID(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, response.ErrBookNotFound)
	assert.Nil(suite.T(), res)
}

// TestAddByBookID 测试作者添加章节时写入章节并投递 outbox 事件
func (suite *BookChapterServiceTestSuite) TestAddByBookID() {
	ctx := context.Background()
	req := &dto.CreatBookChapterRequest{
		BookID: "b1", Title: strPtr("第一章"), Content: strPtr("内容"),
		VIPLevel: strPtr("vip_1"), Price: intPtr(30),
	}
	book := &model.Book{ID: "b1", OwnerID: "u1"}
	created := &model.BookChapter{
		ID: "c1", BookID: "b1", Title: "第一章", Content: "内容", VIPLevel: "vip_1", Price: 30,
	}

	suite.bookRepo.On("GetByID", ctx, "b1").Return(book, nil)
	suite.chapterRepo.On("AddByBookID", ctx, mock.AnythingOfType("*model.BookChapter")).Return(created, nil)
	suite.outboxRepo.On("AddEvent", ctx, mock.Anything, mock.AnythingOfType("*model.Outbox")).Return(nil)
	suite.bookshelfRepo.On("ListUserIDsByBookID", mock.Anything, suite.db, "b1").Return([]string{}, nil)

	res, err := suite.svc.AddByBookID(ctx, "u1", req)
	suite.pool.Wait()

	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "第一章", res.Title)
	assert.Equal(suite.T(), "vip_1", res.VIPLevel)
	assert.Equal(suite.T(), int64(30), res.Price)

	// 入库前的章节应当带上归一化后的会员等级与价格
	passed := suite.chapterRepo.Calls[0].Arguments.Get(1).(*model.BookChapter)
	assert.Equal(suite.T(), "vip_1", passed.VIPLevel)
	assert.Equal(suite.T(), int64(30), passed.Price)
	assert.NotEmpty(suite.T(), passed.ID)

	// 校验 outbox 事件类型与操作
	outboxArg := suite.outboxRepo.Calls[0].Arguments.Get(2).(*model.Outbox)
	assert.Equal(suite.T(), model.TableNameBookChapter, outboxArg.Type)
	assert.Equal(suite.T(), model.DatabaseOperationCreate, outboxArg.Operation)
}

// ---------- UpdateChapter ----------

// TestUpdateChapterPartialFields 测试只更新传入的字段，未传字段保持原值
func (suite *BookChapterServiceTestSuite) TestUpdateChapterPartialFields() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "旧标题", VIPLevel: "vip_0", Price: 10}
	chapterContent := &model.BookChapterContent{ID: "c1", Content: "旧内容"}
	book := &model.Book{ID: "b1", OwnerID: "u1"}
	req := &dto.UpdateBookChapterRequest{ID: "c1", Title: strPtr("新标题")}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", book)
	suite.chapterCache.On("LockBookChapterInfo", ctx, "c1").Return("token", true, nil)
	suite.chapterRepo.On("BookChapterInfoUpdate", ctx, mock.Anything, "c1", mock.AnythingOfType("*model.BookChapterInfo")).Return(nil)
	suite.outboxRepo.On("AddEvent", ctx, mock.Anything, mock.AnythingOfType("*model.Outbox")).Return(nil)
	suite.chapterCache.On("UpdateBookChapterInfo", ctx, "c1", mock.AnythingOfType("*model.BookChapterInfo")).Return(nil)
	suite.chapterCache.On("UnlockBookChapterInfo", ctx, "c1", "token").Return(nil)

	res, err := suite.svc.UpdateChapter(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "新标题", res.Title)
	assert.Equal(suite.T(), "旧内容", res.Content)
	assert.Equal(suite.T(), int64(10), res.Price)
	suite.chapterCache.AssertCalled(suite.T(), "UpdateBookChapterInfo", ctx, "c1", mock.AnythingOfType("*model.BookChapterInfo"))
}

// TestUpdateChapterEmptyStringIgnored 测试传入空字符串时不覆盖原有标题与内容
func (suite *BookChapterServiceTestSuite) TestUpdateChapterEmptyStringIgnored() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "旧标题", VIPLevel: "vip_0"}
	chapterContent := &model.BookChapterContent{ID: "c1", Content: "旧内容"}
	req := &dto.UpdateBookChapterRequest{ID: "c1", Title: strPtr(""), Content: strPtr("")}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", &model.Book{ID: "b1", OwnerID: "u1"})

	res, err := suite.svc.UpdateChapter(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "旧标题", res.Title)
	assert.Equal(suite.T(), "旧内容", res.Content)
	suite.chapterCache.AssertNotCalled(suite.T(), "UpdateBookChapterInfo", mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateChapterVIPAndPrice 测试会员等级与价格被规范化后更新
func (suite *BookChapterServiceTestSuite) TestUpdateChapterVIPAndPrice() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "标题", VIPLevel: "vip_0", Price: 10}
	chapterContent := &model.BookChapterContent{ID: "c1"}
	req := &dto.UpdateBookChapterRequest{ID: "c1", VIPLevel: strPtr("weird"), Price: i64Ptr(99)}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", &model.Book{ID: "b1", OwnerID: "u1"})
	suite.chapterCache.On("LockBookChapterInfo", ctx, "c1").Return("token", true, nil)
	suite.chapterRepo.On("BookChapterInfoUpdate", ctx, mock.Anything, "c1", mock.AnythingOfType("*model.BookChapterInfo")).Return(nil)
	suite.outboxRepo.On("AddEvent", ctx, mock.Anything, mock.AnythingOfType("*model.Outbox")).Return(nil)
	suite.chapterCache.On("UpdateBookChapterInfo", ctx, "c1", mock.AnythingOfType("*model.BookChapterInfo")).Return(nil)
	suite.chapterCache.On("UnlockBookChapterInfo", ctx, "c1", "token").Return(nil)

	res, err := suite.svc.UpdateChapter(ctx, "u1", req)
	assert.NoError(suite.T(), err)
	// 未知等级字符串会被归一化为 vip_0
	assert.Equal(suite.T(), "vip_0", res.VIPLevel)
	assert.Equal(suite.T(), int64(99), res.Price)
}

// TestUpdateChapterPermissionDenied 测试非作者更新章节被拒绝
func (suite *BookChapterServiceTestSuite) TestUpdateChapterPermissionDenied() {
	ctx := context.Background()
	req := &dto.UpdateBookChapterRequest{ID: "c1", Title: strPtr("新标题")}

	suite.mockChapterCacheHit(ctx, "c1",
		&model.BookChapterInfo{ID: "c1", BookID: "b1"},
		&model.BookChapterContent{ID: "c1"})
	suite.mockBookCacheHit(ctx, "b1", &model.Book{ID: "b1", OwnerID: "other"})

	res, err := suite.svc.UpdateChapter(ctx, "u1", req)
	assert.ErrorIs(suite.T(), err, response.ErrPermission)
	assert.Nil(suite.T(), res)
	suite.chapterRepo.AssertNotCalled(suite.T(), "BookChapterInfoUpdate", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestUpdateChapterNotFound 测试章节不存在
func (suite *BookChapterServiceTestSuite) TestUpdateChapterNotFound() {
	ctx := context.Background()
	suite.mockChapterCacheMissNotFound(ctx, "c1")

	res, err := suite.svc.UpdateChapter(ctx, "u1", &dto.UpdateBookChapterRequest{ID: "c1"})
	assert.ErrorIs(suite.T(), err, response.ErrBookChapterNotFound)
	assert.Nil(suite.T(), res)
}

// TestUpdateChapterOutboxErrorRollback 测试 outbox 写入失败时整体失败且不刷新缓存
func (suite *BookChapterServiceTestSuite) TestUpdateChapterOutboxErrorRollback() {
	ctx := context.Background()
	wantErr := errors.New("outbox fail")
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "标题"}
	chapterContent := &model.BookChapterContent{ID: "c1"}

	suite.mockChapterCacheHit(ctx, "c1", chapterInfo, chapterContent)
	suite.mockBookCacheHit(ctx, "b1", &model.Book{ID: "b1", OwnerID: "u1"})
	suite.chapterCache.On("LockBookChapterInfo", ctx, "c1").Return("token", true, nil)
	suite.chapterRepo.On("BookChapterInfoUpdate", ctx, mock.Anything, "c1", mock.AnythingOfType("*model.BookChapterInfo")).Return(nil)
	suite.outboxRepo.On("AddEvent", ctx, mock.Anything, mock.AnythingOfType("*model.Outbox")).Return(wantErr)
	suite.chapterCache.On("UnlockBookChapterInfo", ctx, "c1", "token").Return(nil)

	res, err := suite.svc.UpdateChapter(ctx, "u1", &dto.UpdateBookChapterRequest{ID: "c1", Title: strPtr("新")})
	assert.ErrorIs(suite.T(), err, wantErr)
	assert.Nil(suite.T(), res)
	suite.chapterCache.AssertNotCalled(suite.T(), "UpdateBookChapterInfo", mock.Anything, mock.Anything, mock.Anything)
}

// ---------- DeleteChapter ----------

// TestDeleteChapter 测试作者删除章节并投递删除事件
func (suite *BookChapterServiceTestSuite) TestDeleteChapter() {
	ctx := context.Background()

	suite.chapterRepo.On("GetByID", ctx, "c1").Return(&model.BookChapter{ID: "c1", BookID: "b1"}, nil)
	suite.bookRepo.On("GetByID", ctx, "b1").Return(&model.Book{ID: "b1", OwnerID: "u1"}, nil)
	suite.chapterRepo.On("Delete", ctx, "c1").Return(nil)
	suite.outboxRepo.On("AddEvent", ctx, mock.Anything, mock.AnythingOfType("*model.Outbox")).Return(nil)

	err := suite.svc.DeleteChapter(ctx, "u1", "c1")
	assert.NoError(suite.T(), err)

	outboxArg := suite.outboxRepo.Calls[0].Arguments.Get(2).(*model.Outbox)
	assert.Equal(suite.T(), model.DatabaseOperationDelete, outboxArg.Operation)
	assert.JSONEq(suite.T(), `{"id":"c1"}`, string(outboxArg.Payload))
}

// TestDeleteChapterPermissionDenied 测试非作者删除章节被拒绝
func (suite *BookChapterServiceTestSuite) TestDeleteChapterPermissionDenied() {
	ctx := context.Background()

	suite.chapterRepo.On("GetByID", ctx, "c1").Return(&model.BookChapter{ID: "c1", BookID: "b1"}, nil)
	suite.bookRepo.On("GetByID", ctx, "b1").Return(&model.Book{ID: "b1", OwnerID: "other"}, nil)

	err := suite.svc.DeleteChapter(ctx, "u1", "c1")
	assert.ErrorIs(suite.T(), err, response.ErrPermission)
	suite.chapterRepo.AssertNotCalled(suite.T(), "Delete", mock.Anything, mock.Anything)
}

// TestDeleteChapterNotFound 测试删除不存在的章节
func (suite *BookChapterServiceTestSuite) TestDeleteChapterNotFound() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").Return(nil, nil)

	err := suite.svc.DeleteChapter(ctx, "u1", "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookChapterNotFound)
}

// ---------- GetChapterList ----------

// TestGetChapterList 测试返回章节目录
func (suite *BookChapterServiceTestSuite) TestGetChapterList() {
	ctx := context.Background()
	list := &[]model.BookChapterInfo{
		{ID: "c1", BookID: "b1", Title: "第一章", Order: 1, VIPLevel: "vip_0", Price: 0},
		{ID: "c2", BookID: "b1", Title: "第二章", Order: 2, VIPLevel: "vip_1", Price: 10},
	}

	suite.bookRepo.On("GetByID", ctx, "b1").Return(&model.Book{ID: "b1"}, nil)
	suite.chapterRepo.On("GetIDListByBookID", ctx, "b1").Return(list, nil)

	res, err := suite.svc.GetChapterList(ctx, &dto.GetBookChapterListRequest{BookID: "b1"})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res.Chapters, 2)
	assert.Equal(suite.T(), "c1", res.Chapters[0].ChapterID)
	assert.Equal(suite.T(), int64(10), res.Chapters[1].Price)
}

// TestGetChapterListEmpty 测试无章节时返回空数组而不是 nil
func (suite *BookChapterServiceTestSuite) TestGetChapterListEmpty() {
	ctx := context.Background()

	suite.bookRepo.On("GetByID", ctx, "b1").Return(&model.Book{ID: "b1"}, nil)
	suite.chapterRepo.On("GetIDListByBookID", ctx, "b1").Return(&[]model.BookChapterInfo{}, nil)

	res, err := suite.svc.GetChapterList(ctx, &dto.GetBookChapterListRequest{BookID: "b1"})
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res.Chapters)
	assert.Len(suite.T(), res.Chapters, 0)
}

// TestGetChapterListBookNotFound 测试书籍不存在
func (suite *BookChapterServiceTestSuite) TestGetChapterListBookNotFound() {
	ctx := context.Background()
	suite.bookRepo.On("GetByID", ctx, "b1").Return(nil, nil)

	res, err := suite.svc.GetChapterList(ctx, &dto.GetBookChapterListRequest{BookID: "b1"})
	assert.ErrorIs(suite.T(), err, response.ErrBookNotFound)
	assert.Nil(suite.T(), res)
}

// ---------- BuyBookChapter ----------

// TestBuyBookChapterPriceMismatch 测试请求价格与章节价格不一致时返回参数错误
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterPriceMismatch() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "b1", Price: 20}, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrParam)
}

// TestBuyBookChapterBookIDMismatch 测试章节不属于请求书籍时返回参数错误
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterBookIDMismatch() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "other", Price: 10}, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrParam)
}

// TestBuyBookChapterNotFound 测试章节不存在
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterNotFound() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").Return(nil, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrBookNotFound)
}

// TestBuyBookChapterAlreadyBuy 测试重复购买返回 response.ErrBookAlreadyBuy
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterAlreadyBuy() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "b1", Price: 10}, nil)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").
		Return(&model.UserPurchasedChapter{ID: "p1"}, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrBookAlreadyBuy)
	suite.assetRepo.AssertNotCalled(suite.T(), "SubGoldByID",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

// TestBuyBookChapterInsufficientFunds 测试余额不足时不扣款、不写购买记录
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterInsufficientFunds() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "b1", Price: 10}, nil)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").Return(nil, nil)
	suite.assetRepo.On("GetByID", ctx, mock.Anything, "u1").
		Return(&model.UserAsset{ID: "u1", Gold: 5}, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrSufficientFunds)
	suite.assetRepo.AssertNotCalled(suite.T(), "SubGoldByID",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	suite.purchasedRepo.AssertNotCalled(suite.T(), "AddPurchasedChapter", mock.Anything, mock.Anything)
}

// TestBuyBookChapterUserAssetNotFound 测试用户资产不存在
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterUserAssetNotFound() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "b1", Price: 10}, nil)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").Return(nil, nil)
	suite.assetRepo.On("GetByID", ctx, mock.Anything, "u1").Return(nil, nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.ErrorIs(suite.T(), err, response.ErrUserNotFound)
}

// TestBuyBookChapterSuccess 测试余额充足时扣款并写入购买记录
func (suite *BookChapterServiceTestSuite) TestBuyBookChapterSuccess() {
	ctx := context.Background()
	suite.chapterRepo.On("GetByID", ctx, "c1").
		Return(&model.BookChapter{ID: "c1", BookID: "b1", Price: 10}, nil)
	suite.purchasedRepo.On("GetByChapterIDAndUserID", ctx, "c1", "u1").Return(nil, nil)
	suite.assetRepo.On("GetByID", ctx, mock.Anything, "u1").
		Return(&model.UserAsset{ID: "u1", Gold: 100}, nil)
	suite.assetRepo.On("SubGoldByID", ctx, mock.Anything, "u1", int64(10)).
		Return(&model.UserAsset{ID: "u1", Gold: 90}, nil)
	suite.purchasedRepo.On("AddPurchasedChapter", ctx, mock.AnythingOfType("*model.UserPurchasedChapter")).
		Return(nil)

	err := suite.svc.BuyBookChapter(ctx, "u1", &dto.BuyBookChapterRequest{
		BookID: "b1", ChapterID: "c1", Price: 10,
	})
	assert.NoError(suite.T(), err)

	purchased := suite.purchasedRepo.Calls[len(suite.purchasedRepo.Calls)-1].
		Arguments.Get(1).(*model.UserPurchasedChapter)
	assert.Equal(suite.T(), "u1", purchased.UserID)
	assert.Equal(suite.T(), "c1", purchased.ChapterID)
	assert.Equal(suite.T(), "b1", purchased.BookID)
	// 购买有效期约为 3 年
	assert.True(suite.T(), purchased.ActiveAt.After(time.Now().AddDate(2, 0, 0)))
}

// ---------- GetChapterInfo ----------

// TestGetChapterInfoAllCacheHit 测试章节与书籍均命中缓存时不查询数据库
func (suite *BookChapterServiceTestSuite) TestGetChapterInfoAllCacheHit() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "第一章", Order: 1, VIPLevel: "vip_0"}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.chapterCache.On("GetBookChapterInfo", ctx, "c1").Return(chapterInfo, true, nil)
	suite.mockBookCacheHit(ctx, "b1", book)

	res, err := suite.svc.GetChapterInfo(ctx, "c1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "c1", res.ChapterID)
	assert.Equal(suite.T(), "b1", res.Book.ID)
	suite.chapterRepo.AssertNotCalled(suite.T(), "GetByID", mock.Anything, mock.Anything)
	suite.bookRepo.AssertNotCalled(suite.T(), "GetByID", mock.Anything, mock.Anything)
}

// TestGetChapterInfoCacheMiss 测试缓存全部未命中时回源并回写缓存
func (suite *BookChapterServiceTestSuite) TestGetChapterInfoCacheMiss() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1", Title: "第一章"}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.chapterCache.On("GetBookChapterInfo", ctx, "c1").Return(nil, false, nil)
	suite.chapterCache.On("LockBookChapterInfo", ctx, "c1").Return("token", true, nil)
	suite.chapterRepo.On("BookChapterInfoGetByID", ctx, suite.db, "c1").Return(chapterInfo, nil)
	suite.chapterCache.On("UpdateBookChapterInfo", ctx, "c1", chapterInfo).Return(nil)
	suite.chapterCache.On("UnlockBookChapterInfo", ctx, "c1", "token").Return(nil)

	suite.bookCache.On("GetBook", ctx, "b1").Return(nil, false, nil)
	suite.bookCache.On("LockBook", ctx, "b1").Return("token", true, nil)
	suite.bookRepo.On("GetByID", ctx, "b1").Return(book, nil)
	suite.bookCache.On("UpdateBook", ctx, "b1", book).Return(nil)
	suite.bookCache.On("UnlockBook", ctx, "b1", "token").Return(nil)

	res, err := suite.svc.GetChapterInfo(ctx, "c1")
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "c1", res.ChapterID)
	suite.chapterCache.AssertCalled(suite.T(), "UpdateBookChapterInfo", ctx, "c1", chapterInfo)
	suite.bookCache.AssertCalled(suite.T(), "UpdateBook", ctx, "b1", book)
}

// TestGetChapterInfoChapterNotFound 测试缓存与数据库均无章节
func (suite *BookChapterServiceTestSuite) TestGetChapterInfoChapterNotFound() {
	ctx := context.Background()

	suite.mockChapterCacheMissNotFound(ctx, "c1")

	res, err := suite.svc.GetChapterInfo(ctx, "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookChapterNotFound)
	assert.Nil(suite.T(), res)
}

// TestGetChapterInfoBookNotFound 测试章节存在但书籍已不存在
func (suite *BookChapterServiceTestSuite) TestGetChapterInfoBookNotFound() {
	ctx := context.Background()
	chapterInfo := &model.BookChapterInfo{ID: "c1", BookID: "b1"}

	suite.chapterCache.On("GetBookChapterInfo", ctx, "c1").Return(chapterInfo, true, nil)
	suite.mockBookCacheMissNotFound(ctx, "b1")

	res, err := suite.svc.GetChapterInfo(ctx, "c1")
	assert.ErrorIs(suite.T(), err, response.ErrBookNotFound)
	assert.Nil(suite.T(), res)
}

// ---------- SearchBookChaptersV2 ----------

// TestSearchBookChaptersV2 测试搜索结果组合书籍信息与高亮
func (suite *BookChapterServiceTestSuite) TestSearchBookChaptersV2() {
	ctx := context.Background()
	esResults := []model.BookChapterSearchResult{
		{
			BookChapter: model.BookChapter{ID: "c1", BookID: "b1", Title: "第一章"},
			Highlight:   map[string]string{"title": "<em>第一</em>章"},
		},
	}
	book := &model.Book{ID: "b1", Name: "book-1"}

	suite.esRepo.On("Search", ctx, mock.AnythingOfType("model.BookChapterFilterWithIDPage")).
		Return(esResults, nil)
	suite.bookCache.On("GetBook", ctx, "b1").Return(book, true, nil)

	res, err := suite.svc.SearchBookChaptersV2(ctx, &dto.SearchBookChaptersV2Request{
		Query: "第一", BookID: "b1", Limit: 10,
	})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res.Chapters, 1)
	assert.Equal(suite.T(), "c1", res.Chapters[0].BookChapter.ChapterID)
	assert.Equal(suite.T(), "<em>第一</em>章", res.Chapters[0].Highlight["title"])
}

// TestSearchBookChaptersV2VIPLevelFilter 测试逗号分隔的会员等级被拆分后传给 ES
func (suite *BookChapterServiceTestSuite) TestSearchBookChaptersV2VIPLevelFilter() {
	ctx := context.Background()

	suite.esRepo.On("Search", ctx, mock.AnythingOfType("model.BookChapterFilterWithIDPage")).
		Return([]model.BookChapterSearchResult{}, nil)

	_, err := suite.svc.SearchBookChaptersV2(ctx, &dto.SearchBookChaptersV2Request{
		VIPLevel: "vip_0,vip_1", Limit: 10,
	})
	assert.NoError(suite.T(), err)

	filter := suite.esRepo.Calls[0].Arguments.Get(1).(model.BookChapterFilterWithIDPage)
	assert.Equal(suite.T(), []string{"vip_0", "vip_1"}, filter.VIPLevel)
}

// TestSearchBookChaptersV2SkipMissingBook 测试书籍缺失的搜索结果被跳过
func (suite *BookChapterServiceTestSuite) TestSearchBookChaptersV2SkipMissingBook() {
	ctx := context.Background()
	esResults := []model.BookChapterSearchResult{
		{BookChapter: model.BookChapter{ID: "c1", BookID: "b-missing"}},
		{BookChapter: model.BookChapter{ID: "c2", BookID: "b1"}},
	}

	suite.esRepo.On("Search", ctx, mock.AnythingOfType("model.BookChapterFilterWithIDPage")).
		Return(esResults, nil)
	suite.mockBookCacheMissNotFound(ctx, "b-missing")
	suite.mockBookCacheHit(ctx, "b1", &model.Book{ID: "b1"})

	res, err := suite.svc.SearchBookChaptersV2(ctx, &dto.SearchBookChaptersV2Request{Limit: 10})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), res.Chapters, 1)
	assert.Equal(suite.T(), "c2", res.Chapters[0].BookChapter.ChapterID)
}

// TestSearchBookChaptersV2ESError 测试 ES 查询失败时返回错误
func (suite *BookChapterServiceTestSuite) TestSearchBookChaptersV2ESError() {
	ctx := context.Background()
	wantErr := errors.New("es down")

	suite.esRepo.On("Search", ctx, mock.AnythingOfType("model.BookChapterFilterWithIDPage")).
		Return(nil, wantErr)

	res, err := suite.svc.SearchBookChaptersV2(ctx, &dto.SearchBookChaptersV2Request{Limit: 10})
	assert.ErrorIs(suite.T(), err, wantErr)
	assert.Nil(suite.T(), res)
}

// TestSearchBookChaptersV2Empty 测试无结果时返回空数组而不是 nil
func (suite *BookChapterServiceTestSuite) TestSearchBookChaptersV2Empty() {
	ctx := context.Background()

	suite.esRepo.On("Search", ctx, mock.AnythingOfType("model.BookChapterFilterWithIDPage")).
		Return([]model.BookChapterSearchResult{}, nil)

	res, err := suite.svc.SearchBookChaptersV2(ctx, &dto.SearchBookChaptersV2Request{Limit: 10})
	assert.NoError(suite.T(), err)
	assert.NotNil(suite.T(), res.Chapters)
	assert.Len(suite.T(), res.Chapters, 0)
}

func TestBookChapterServiceTestSuite(t *testing.T) {
	suite.Run(t, new(BookChapterServiceTestSuite))
}
