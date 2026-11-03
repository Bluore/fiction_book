package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// feedPipelineHook 统计 Redis pipeline 往返次数，用于验证热路径 RTT
type feedPipelineHook struct {
	pipelines atomic.Int64
}

func (h *feedPipelineHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *feedPipelineHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		return next(ctx, cmd)
	}
}

func (h *feedPipelineHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		h.pipelines.Add(1)
		return next(ctx, cmds)
	}
}

// feedServiceTestEnv Feed 服务测试依赖集合
type feedServiceTestEnv struct {
	svc       *feedServiceV2
	feedCache *MockUserFeedCache
	compose   *MockFeedComposeCache
	bookRepo  *MockBookRepository
}

func newFeedServiceTestEnv() *feedServiceTestEnv {
	feedCache := &MockUserFeedCache{}
	compose := &MockFeedComposeCache{}
	bookRepo := &MockBookRepository{}

	return &feedServiceTestEnv{
		svc: &feedServiceV2{
			bookRepo:         bookRepo,
			feedCache:        feedCache,
			feedComposeCache: compose,
			composeMetrics:   newFeedComposeMetrics(),
		},
		feedCache: feedCache,
		compose:   compose,
		bookRepo:  bookRepo,
	}
}

func testFeedCard(feedID string, authorID string, bookID string) *model.FeedCard {
	return &model.FeedCard{
		FeedID:              feedID,
		UserID:              authorID,
		BookID:              bookID,
		Content:             "content-" + feedID,
		FeedCreatedAt:       time.Unix(100, 0),
		FeedUpdatedAt:       time.Unix(200, 0),
		BookName:            "book-" + bookID,
		BookDescription:     "description-" + bookID,
		BookCoverURL:        "cover-" + bookID,
		BookVIPLevel:        "v1",
		BookPrice:           100,
		BookBrowseAmount:    11,
		BookBookshelfAmount: 12,
		BookPushedAt:        time.Unix(300, 0),
		BookCreatedAt:       time.Unix(400, 0),
		BookUpdatedAt:       time.Unix(500, 0),
		AuthorProfileURL:    "profile-" + authorID,
		AuthorFansNum:       66,
		AuthorVIPLevel:      "v2",
	}
}

func testFeedDetail(feedID string, authorID string, bookID string) *model.UserFeed {
	return &model.UserFeed{
		FeedID:    feedID,
		UserID:    authorID,
		BookID:    bookID,
		Content:   "content-" + feedID,
		CreatedAt: time.Unix(100, 0),
		UpdatedAt: time.Unix(200, 0),
	}
}

func newFeedRedisEnv(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() {
		_ = rdb.Close()
		mr.Close()
	})
	return mr, rdb
}

func setRedisJSON(t *testing.T, mr *miniredis.Miniredis, key string, value any) {
	t.Helper()
	content, err := json.Marshal(value)
	require.NoError(t, err)
	require.NoError(t, mr.Set(key, string(content)))
}

func attachEntityCaches(svc *feedServiceV2, rdb *redis.Client) {
	svc.bookCache = cache.NewBookCatch(rdb)
	svc.userCache = cache.NewUserCache(rdb)
	svc.userAssetCache = cache.NewUserAssetCache(rdb)
	svc.userInfoCache = cache.NewUserInfoCache(rdb)
	svc.userLikeCache = cache.NewUserLikeCache(rdb)
	svc.userBookshelfCache = cache.NewUserBookshelfCache(rdb)
}

func seedCardSources(t *testing.T, mr *miniredis.Miniredis, authorID string, bookID string) {
	t.Helper()
	setRedisJSON(t, mr, cache.KeyBook(bookID), &model.Book{
		ID:              bookID,
		Name:            "book-" + bookID,
		Description:     "description-" + bookID,
		CoverImage:      "cover-" + bookID,
		VIPLevel:        "v1",
		Price:           100,
		BrowseAmount:    11,
		BookshelfAmount: 12,
		PushedAt:        time.Unix(300, 0),
		CreatedAt:       time.Unix(400, 0),
		UpdatedAt:       time.Unix(500, 0),
	})
	setRedisJSON(t, mr, cache.KeyUser(authorID), &model.User{ID: authorID, ProfileImage: "profile-" + authorID})
	setRedisJSON(t, mr, cache.KeyUserInfo(authorID), &model.UserInfo{UserID: authorID, FansNum: 66})
	setRedisJSON(t, mr, cache.KeyUserAsset(authorID), &model.UserAsset{ID: authorID, VIPLevel: "v2"})
}

func seedViewerStates(t *testing.T, mr *miniredis.Miniredis, viewerID string, authorID string, bookID string, liked bool, inShelf bool) {
	t.Helper()
	if liked {
		setRedisJSON(t, mr, cache.KeyUserLike(authorID, viewerID), &model.UserLike{UserID: authorID, LikerID: viewerID})
	}
	if inShelf {
		setRedisJSON(t, mr, cache.KeyUserBookshelf(viewerID, bookID), &model.UserBookshelf{UserID: viewerID, BookID: bookID})
	}
}

// TestGetUserFeedAllCardsHit 卡片与查看者状态全命中：不回源、不写回
func TestGetUserFeedAllCardsHit(t *testing.T) {
	env := newFeedServiceTestEnv()
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 3, 10).
		Return([]string{"f1", "f2"}, true, nil).Once()

	cards := map[string]*model.FeedCard{
		"f1": testFeedCard("f1", "author-1", "book-1"),
		"f2": testFeedCard("f2", "author-2", "book-2"),
	}
	env.compose.On("GetCards", mock.Anything, []string{"f1", "f2"}).Return(cards, []string{}, nil).Once()
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(
			map[string]bool{"author-1": true, "author-2": false},
			map[string]bool{"book-1": false, "book-2": true},
			nil,
		).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1", Offset: 3})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 2)
	assert.Equal(t, "cursor-1", resp.NextCursorID)
	assert.Equal(t, 5, resp.NextOffset)

	assert.Equal(t, "book-book-1", resp.Feeds[0].BookName)
	assert.Equal(t, "profile-author-1", resp.Feeds[0].UserProfileUrl)
	assert.Equal(t, int64(66), resp.Feeds[0].UserFans)
	assert.Equal(t, "v2", resp.Feeds[0].UserVIPLevel)
	assert.True(t, resp.Feeds[0].IsUserLiker)
	assert.False(t, resp.Feeds[0].IsInBookShelf)
	assert.False(t, resp.Feeds[1].IsUserLiker)
	assert.True(t, resp.Feeds[1].IsInBookShelf)

	env.compose.AssertNumberOfCalls(t, "UpdateCards", 0)
	env.feedCache.AssertNotCalled(t, "GetFeedDetail", mock.Anything, mock.Anything)
	env.bookRepo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)

	snapshot := env.svc.feedMetrics().snapshot()
	assert.Equal(t, int64(2), snapshot.CardHit)
	assert.Equal(t, int64(0), snapshot.CardMiss)
}

// TestGetUserFeedPartialCardsHit 部分命中：只对 miss 的卡片走原有单条回源
func TestGetUserFeedPartialCardsHit(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1", "f2"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1", "f2"}).
		Return(map[string]*model.FeedCard{"f1": testFeedCard("f1", "author-1", "book-1")}, []string{"f2"}, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f2").
		Return(testFeedDetail("f2", "author-2", "book-2"), true, nil).Once()
	seedCardSources(t, mr, "author-2", "book-2")
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(
			map[string]bool{"author-1": true, "author-2": false},
			map[string]bool{"book-1": true, "book-2": false},
			nil,
		).Once()
	env.compose.On("UpdateCards", mock.Anything, mock.MatchedBy(func(cards []*model.FeedCard) bool {
		return len(cards) == 1 && cards[0].FeedID == "f2"
	})).Return(nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 2)
	assert.Equal(t, "content-f1", resp.Feeds[0].Content)
	assert.Equal(t, "content-f2", resp.Feeds[1].Content)
	assert.Equal(t, "book-book-2", resp.Feeds[1].BookName)

	snapshot := env.svc.feedMetrics().snapshot()
	assert.Equal(t, int64(1), snapshot.CardHit)
	assert.Equal(t, int64(1), snapshot.CardMiss)
	assert.Equal(t, int64(1), snapshot.SourceFallbackFeed)
	assert.Equal(t, int64(1), snapshot.SourceFallbackBook)
	assert.Equal(t, int64(1), snapshot.SourceFallbackUser)
	assert.Equal(t, int64(1), snapshot.SourceFallbackUserAsset)
	assert.Equal(t, int64(1), snapshot.SourceFallbackUserInfo)
}

// TestGetUserFeedAllCardsMiss 全部未命中：逐条回源并批量写回
func TestGetUserFeedAllCardsMiss(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1", "f2"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1", "f2"}).
		Return(map[string]*model.FeedCard{}, []string{"f1", "f2"}, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f1").
		Return(testFeedDetail("f1", "author-1", "book-1"), true, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f2").
		Return(testFeedDetail("f2", "author-2", "book-2"), true, nil).Once()
	seedCardSources(t, mr, "author-1", "book-1")
	seedCardSources(t, mr, "author-2", "book-2")
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(
			map[string]bool{"author-1": true, "author-2": false},
			map[string]bool{"book-1": false, "book-2": true},
			nil,
		).Once()
	env.compose.On("UpdateCards", mock.Anything, mock.MatchedBy(func(cards []*model.FeedCard) bool {
		return len(cards) == 2
	})).Return(nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 2)

	snapshot := env.svc.feedMetrics().snapshot()
	assert.Equal(t, int64(0), snapshot.CardHit)
	assert.Equal(t, int64(2), snapshot.CardMiss)
}

// TestGetUserFeedCardEmpty 卡片为 EMPTY：跳过该条且不回源
func TestGetUserFeedCardEmpty(t *testing.T) {
	env := newFeedServiceTestEnv()
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{}, []string{}, nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
	env.feedCache.AssertNotCalled(t, "GetFeedDetail", mock.Anything, mock.Anything)
	env.compose.AssertNumberOfCalls(t, "UpdateCards", 0)
	env.compose.AssertNumberOfCalls(t, "GetViewerStates", 0)
}

// TestGetUserFeedGetCardsFailure pipeline 失败：全部按 miss 处理
func TestGetUserFeedGetCardsFailure(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(nil, nil, errors.New("pipeline down")).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f1").
		Return(testFeedDetail("f1", "author-1", "book-1"), true, nil).Once()
	seedCardSources(t, mr, "author-1", "book-1")
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(map[string]bool{"author-1": true}, map[string]bool{"book-1": true}, nil).Once()
	env.compose.On("UpdateCards", mock.Anything, mock.MatchedBy(func(cards []*model.FeedCard) bool {
		return len(cards) == 1
	})).Return(nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 1)
	assert.Equal(t, "book-book-1", resp.Feeds[0].BookName)
	assert.Equal(t, int64(1), env.svc.feedMetrics().snapshot().CardMiss)
}

// TestGetUserFeedMissingBook 必需书籍缺失：跳过且不写回
func TestGetUserFeedMissingBook(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{}, []string{"f1"}, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f1").
		Return(testFeedDetail("f1", "author-1", "book-1"), true, nil).Once()
	setRedisJSON(t, mr, cache.KeyUser("author-1"), &model.User{ID: "author-1", ProfileImage: "profile"})
	setRedisJSON(t, mr, cache.KeyUserInfo("author-1"), &model.UserInfo{UserID: "author-1", FansNum: 66})
	setRedisJSON(t, mr, cache.KeyUserAsset("author-1"), &model.UserAsset{ID: "author-1", VIPLevel: "v2"})
	env.bookRepo.On("GetByID", mock.Anything, "book-1").Return(nil, nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
	env.compose.AssertNumberOfCalls(t, "UpdateCards", 0)
}

// TestGetUserFeedMissingUser 必需作者缺失：跳过且不写回
func TestGetUserFeedMissingUser(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	userRepo := &MockUserRepositoryForService{}
	env.svc.userRepo = userRepo
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{}, []string{"f1"}, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f1").
		Return(testFeedDetail("f1", "author-1", "book-1"), true, nil).Once()
	setRedisJSON(t, mr, cache.KeyBook("book-1"), &model.Book{ID: "book-1", Name: "book"})
	setRedisJSON(t, mr, cache.KeyUserInfo("author-1"), &model.UserInfo{UserID: "author-1", FansNum: 66})
	setRedisJSON(t, mr, cache.KeyUserAsset("author-1"), &model.UserAsset{ID: "author-1", VIPLevel: "v2"})
	userRepo.On("GetByID", mock.Anything, mock.Anything, "author-1").Return(nil, nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
	env.compose.AssertNumberOfCalls(t, "UpdateCards", 0)
}

// TestGetUserFeedWriteBackFailure 写回失败不影响响应
func TestGetUserFeedWriteBackFailure(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{}, []string{"f1"}, nil).Once()
	env.feedCache.On("GetFeedDetail", mock.Anything, "f1").
		Return(testFeedDetail("f1", "author-1", "book-1"), true, nil).Once()
	seedCardSources(t, mr, "author-1", "book-1")
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(map[string]bool{"author-1": true}, map[string]bool{"book-1": true}, nil).Once()
	env.compose.On("UpdateCards", mock.Anything, mock.Anything).Return(errors.New("redis down")).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 1)
	assert.True(t, resp.Feeds[0].IsUserLiker)
	assert.True(t, resp.Feeds[0].IsInBookShelf)
}

// TestGetUserFeedViewerStateFallback 状态 map miss 时回退原有 helper
func TestGetUserFeedViewerStateFallback(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{"f1": testFeedCard("f1", "author-1", "book-1")}, []string{}, nil).Once()
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(map[string]bool{}, map[string]bool{}, nil).Once()
	seedViewerStates(t, mr, "viewer-1", "author-1", "book-1", true, true)

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 1)
	assert.True(t, resp.Feeds[0].IsUserLiker)
	assert.True(t, resp.Feeds[0].IsInBookShelf)

	snapshot := env.svc.feedMetrics().snapshot()
	assert.Equal(t, int64(1), snapshot.SourceFallbackLike)
	assert.Equal(t, int64(1), snapshot.SourceFallbackShelf)
}

// TestGetUserFeedShelfErrorSkipFeed 书架查询错误时跳过该条
func TestGetUserFeedShelfErrorSkipFeed(t *testing.T) {
	env := newFeedServiceTestEnv()
	ctx := context.Background()

	badRdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = badRdb.Close() }()
	env.svc.userLikeCache = cache.NewUserLikeCache(badRdb)
	env.svc.userBookshelfCache = cache.NewUserBookshelfCache(badRdb)
	userLikeRepo := &MockUserLikeRepository{}
	userLikeRepo.On("GetByUserIDAndLikerID", mock.Anything, mock.Anything, "author-1", "viewer-1").Return(nil, nil).Once()
	env.svc.userLikeRepo = userLikeRepo
	userBookshelfRepo := &MockUserBookshelfRepository{}
	userBookshelfRepo.On("GetByUserIDAndBookID", mock.Anything, mock.Anything, "viewer-1", "book-1").Return(nil, errors.New("db down")).Once()
	env.svc.userBookshelfRepo = userBookshelfRepo

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{"f1": testFeedCard("f1", "author-1", "book-1")}, []string{}, nil).Once()
	env.compose.On("GetViewerStates", mock.Anything, "viewer-1", mock.Anything, mock.Anything).
		Return(map[string]bool{}, map[string]bool{}, nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
}

// TestGuestGetUserFeedSharedCards 游客复用卡片且不读取 like/shelf
func TestGuestGetUserFeedSharedCards(t *testing.T) {
	env := newFeedServiceTestEnv()
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, guestFeedCacheUserID, "cursor-g", 0, 10).
		Return([]string{"f1"}, true, nil).Once()
	env.compose.On("GetCards", mock.Anything, []string{"f1"}).
		Return(map[string]*model.FeedCard{"f1": testFeedCard("f1", "author-1", "book-1")}, []string{}, nil).Once()

	resp, err := env.svc.GuestGetUserFeed(ctx, &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-g"})
	require.NoError(t, err)
	require.Len(t, resp.Feeds, 1)
	assert.Equal(t, "f1", resp.Feeds[0].FeedID)
	assert.Equal(t, "book-book-1", resp.Feeds[0].BookName)
	assert.Equal(t, "cursor-g", resp.NextCursorID)
	assert.Equal(t, 1, resp.NextOffset)

	env.compose.AssertNumberOfCalls(t, "GetViewerStates", 0)
	env.compose.AssertNumberOfCalls(t, "UpdateCards", 0)
}

// TestGetUserFeedEmptyCursorList 目标游标为空：返回空列表与默认游标
func TestGetUserFeedEmptyCursorList(t *testing.T) {
	env := newFeedServiceTestEnv()
	ctx := context.Background()

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-1", 0, 10).
		Return([]string{}, true, nil).Once()
	env.feedCache.On("GetUserFeedCursor", mock.Anything, "viewer-1", "cursor-1").
		Return(&model.UserFeedCursor{CursorID: "cursor-1", NextCursorID: "cursor-2"}, true, nil).Once()
	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", "cursor-2", 0, 10).
		Return([]string{}, true, nil).Once()

	resp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
	assert.Equal(t, model.DefaultFeedCursor, resp.NextCursorID)
	assert.Equal(t, 0, resp.NextOffset)
	env.compose.AssertNumberOfCalls(t, "GetCards", 0)
}

// TestGetUserFeedRootFallback 关注列表为空导致 nextCursor 为 nil：返回默认游标
func TestGetUserFeedRootFallback(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	env := newFeedServiceTestEnv()
	env.svc.userLikeCache = cache.NewUserLikeCache(rdb)
	userLikeRepo := &MockUserLikeRepository{}
	userLikeRepo.On("GetUserLikeBigVListByLikerID", mock.Anything, mock.Anything, "viewer-1").
		Return([]string{}, nil).Once()
	userLikeRepo.On("GetUserLikeSmallVListByLikerID", mock.Anything, mock.Anything, "viewer-1").
		Return([]string{}, nil).Once()
	env.svc.userLikeRepo = userLikeRepo

	env.feedCache.On("GetUserFeedFeedIDs", mock.Anything, "viewer-1", model.DefaultFeedCursor, 0, 10).
		Return([]string{}, false, nil).Once()

	resp, err := env.svc.GetUserFeed(context.Background(), "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: model.DefaultFeedCursor})
	require.NoError(t, err)
	assert.Empty(t, resp.Feeds)
	assert.Equal(t, model.DefaultFeedCursor, resp.NextCursorID)
	assert.Equal(t, 0, resp.NextOffset)
}

// TestFeedCardSharedByUserAndGuest 登录用户写回的卡片可被游客复用，并验证热路径 RTT
func TestFeedCardSharedByUserAndGuest(t *testing.T) {
	mr, rdb := newFeedRedisEnv(t)
	hook := &feedPipelineHook{}
	rdb.AddHook(hook)

	env := newFeedServiceTestEnv()
	attachEntityCaches(env.svc, rdb)
	env.svc.feedCache = cache.NewUserFeedCache(rdb)
	env.svc.feedComposeCache = cache.NewFeedComposeCache(rdb)
	ctx := context.Background()

	require.NoError(t, rdb.RPush(ctx, cache.KeyUserFeedFeedIDs("viewer-1", "cursor-1"), "f1").Err())
	require.NoError(t, rdb.RPush(ctx, cache.KeyUserFeedFeedIDs(guestFeedCacheUserID, "cursor-g"), "f1").Err())
	setRedisJSON(t, mr, cache.KeyFeedDetail("f1"), testFeedDetail("f1", "author-1", "book-1"))
	seedCardSources(t, mr, "author-1", "book-1")
	seedViewerStates(t, mr, "viewer-1", "author-1", "book-1", true, true)

	// 冷路径：卡片 miss，走 buildCard 并写回
	userResp, err := env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, userResp.Feeds, 1)
	assert.True(t, userResp.Feeds[0].IsUserLiker)
	assert.True(t, userResp.Feeds[0].IsInBookShelf)
	assert.True(t, mr.Exists(cache.KeyFeedCard("f1")))

	// 热路径：登录用户 feed IDs + cards + viewer states = 3 次 pipeline
	pipelineBefore := hook.pipelines.Load()
	userResp, err = env.svc.GetUserFeed(ctx, "viewer-1", &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-1"})
	require.NoError(t, err)
	require.Len(t, userResp.Feeds, 1)
	assert.Equal(t, int64(3), hook.pipelines.Load()-pipelineBefore)

	// 游客：feed IDs + cards = 2 次 pipeline，不读取查看者状态
	pipelineBefore = hook.pipelines.Load()
	guestResp, err := env.svc.GuestGetUserFeed(ctx, &dto.GetUserFeedRequestV2{TargetCursorID: "cursor-g"})
	require.NoError(t, err)
	require.Len(t, guestResp.Feeds, 1)
	assert.Equal(t, "book-book-1", guestResp.Feeds[0].BookName)
	assert.Equal(t, int64(2), hook.pipelines.Load()-pipelineBefore)
}
