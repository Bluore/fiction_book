package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// TestMain 缓存层部分分支会写结构化日志，测试前需要初始化 logger
func TestMain(m *testing.M) {
	if err := logger.Init("debug"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// FeedComposeCacheTestSuite Feed 公共组合缓存测试套件
type FeedComposeCacheTestSuite struct {
	suite.Suite
	mr    *miniredis.Miniredis
	rdb   *redis.Client
	cache FeedComposeCache
}

func (suite *FeedComposeCacheTestSuite) SetupSuite() {
	mr, err := miniredis.Run()
	assert.NoError(suite.T(), err)

	suite.mr = mr
	suite.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	suite.cache = NewFeedComposeCache(suite.rdb)
}

func (suite *FeedComposeCacheTestSuite) TearDownSuite() {
	_ = suite.rdb.Close()
	suite.mr.Close()
}

func (suite *FeedComposeCacheTestSuite) TearDownTest() {
	suite.mr.FlushAll()
}

func (suite *FeedComposeCacheTestSuite) setJSON(key string, value any) {
	content, err := json.Marshal(value)
	suite.Require().NoError(err)
	suite.Require().NoError(suite.mr.Set(key, string(content)))
}

func newTestFeedCard(feedID string) *model.FeedCard {
	return &model.FeedCard{
		FeedID:           feedID,
		UserID:           "author-1",
		BookID:           "book-1",
		Content:          "content-" + feedID,
		BookName:         "book",
		AuthorProfileURL: "profile",
		AuthorFansNum:    10,
		AuthorVIPLevel:   "v1",
	}
}

// TestKeyAndTTL 测试 key 前缀与 TTL 抖动范围
func (suite *FeedComposeCacheTestSuite) TestKeyAndTTL() {
	assert.Equal(suite.T(), "GSF:feed:cache:card:v1:feed-1", KeyFeedCard("feed-1"))

	for i := 0; i < 20; i++ {
		ttl := TimeFeedCard()
		assert.GreaterOrEqual(suite.T(), ttl, 60*time.Second)
		assert.Less(suite.T(), ttl, 66*time.Second)
	}
}

// TestDecodeCacheValue 测试通用解码 helper 对普通值、EMPTY 与非法 JSON 的处理
func (suite *FeedComposeCacheTestSuite) TestDecodeCacheValue() {
	var card model.FeedCard
	assert.NoError(suite.T(), decodeCacheValue(`{"feed_id":"f1"}`, &card))
	assert.Equal(suite.T(), "f1", card.FeedID)

	err := decodeCacheValue(constant.RedisValueEmpty, &card)
	assert.ErrorIs(suite.T(), err, errCacheValueEmpty)

	err = decodeCacheValue("{not-json", &card)
	assert.Error(suite.T(), err)
	assert.False(suite.T(), errors.Is(err, errCacheValueEmpty))
}

// TestGetCardsAllHit 测试卡片全部命中
func (suite *FeedComposeCacheTestSuite) TestGetCardsAllHit() {
	ctx := context.Background()
	suite.setJSON(KeyFeedCard("f1"), newTestFeedCard("f1"))
	suite.setJSON(KeyFeedCard("f2"), newTestFeedCard("f2"))

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), cards, 2)
	assert.Empty(suite.T(), missed)
	assert.Equal(suite.T(), "content-f1", cards["f1"].Content)
}

// TestGetCardsPartialMiss 测试部分命中
func (suite *FeedComposeCacheTestSuite) TestGetCardsPartialMiss() {
	ctx := context.Background()
	suite.setJSON(KeyFeedCard("f1"), newTestFeedCard("f1"))

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), cards, 1)
	assert.Equal(suite.T(), []string{"f2"}, missed)
}

// TestGetCardsAllMiss 测试全部未命中
func (suite *FeedComposeCacheTestSuite) TestGetCardsAllMiss() {
	ctx := context.Background()

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), cards)
	assert.Equal(suite.T(), []string{"f1", "f2"}, missed)
}

// TestGetCardsEmptyTombstone 测试 EMPTY 表示确定不存在，不进入 missed
func (suite *FeedComposeCacheTestSuite) TestGetCardsEmptyTombstone() {
	ctx := context.Background()
	suite.Require().NoError(suite.mr.Set(KeyFeedCard("f1"), constant.RedisValueEmpty))

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), cards)
	assert.Equal(suite.T(), []string{"f2"}, missed)
}

// TestGetCardsInvalidJSON 测试单个 key 解码失败只影响该 key
func (suite *FeedComposeCacheTestSuite) TestGetCardsInvalidJSON() {
	ctx := context.Background()
	suite.Require().NoError(suite.mr.Set(KeyFeedCard("f1"), "{invalid"))
	suite.setJSON(KeyFeedCard("f2"), newTestFeedCard("f2"))

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), cards, 1)
	assert.NotContains(suite.T(), cards, "f1")
	assert.Equal(suite.T(), []string{"f1"}, missed)
}

// TestGetCardsDedupe 测试重复 feedID 不会产生重复 missed
func (suite *FeedComposeCacheTestSuite) TestGetCardsDedupe() {
	ctx := context.Background()

	_, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f1"})
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"f1"}, missed)
}

// TestGetCardsPipelineError 测试整条 pipeline 失败返回 error
func (suite *FeedComposeCacheTestSuite) TestGetCardsPipelineError() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = rdb.Close() }()
	c := NewFeedComposeCache(rdb)

	_, _, err := c.GetCards(context.Background(), []string{"f1"})
	assert.Error(suite.T(), err)
}

// TestUpdateAndDeleteCards 测试批量写回、TTL 与删除
func (suite *FeedComposeCacheTestSuite) TestUpdateAndDeleteCards() {
	ctx := context.Background()

	err := suite.cache.UpdateCards(ctx, []*model.FeedCard{newTestFeedCard("f1"), newTestFeedCard("f2")})
	assert.NoError(suite.T(), err)

	ttl := suite.mr.TTL(KeyFeedCard("f1"))
	assert.GreaterOrEqual(suite.T(), ttl, 60*time.Second)
	assert.Less(suite.T(), ttl, 66*time.Second)

	cards, missed, err := suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Len(suite.T(), cards, 2)
	assert.Empty(suite.T(), missed)

	assert.NoError(suite.T(), suite.cache.DeleteCard(ctx, "f1"))
	_, missed, err = suite.cache.GetCards(ctx, []string{"f1", "f2"})
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), []string{"f1"}, missed)
}

// TestUpdateCardsEmpty 测试空列表直接返回成功
func (suite *FeedComposeCacheTestSuite) TestUpdateCardsEmpty() {
	assert.NoError(suite.T(), suite.cache.UpdateCards(context.Background(), nil))
}

// TestUpdateCardsRedisUnavailable 测试写失败返回错误
func (suite *FeedComposeCacheTestSuite) TestUpdateCardsRedisUnavailable() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = rdb.Close() }()
	c := NewFeedComposeCache(rdb)

	err := c.UpdateCards(context.Background(), []*model.FeedCard{newTestFeedCard("f1")})
	assert.Error(suite.T(), err)
}

// TestGetViewerStates 测试点赞/书架命中、EMPTY、miss 与去重
func (suite *FeedComposeCacheTestSuite) TestGetViewerStates() {
	ctx := context.Background()
	suite.setJSON(KeyUserLike("author-1", "viewer-1"), &model.UserLike{UserID: "author-1", LikerID: "viewer-1"})
	suite.setJSON(KeyUserBookshelf("viewer-1", "book-1"), &model.UserBookshelf{UserID: "viewer-1", BookID: "book-1"})
	suite.Require().NoError(suite.mr.Set(KeyUserLike("author-empty", "viewer-1"), constant.RedisValueEmpty))
	suite.Require().NoError(suite.mr.Set(KeyUserBookshelf("viewer-1", "book-empty"), constant.RedisValueEmpty))

	likes, shelves, err := suite.cache.GetViewerStates(
		ctx,
		"viewer-1",
		[]string{"author-1", "author-1", "author-empty", "author-missing"},
		[]string{"book-1", "book-1", "book-empty", "book-missing"},
	)
	assert.NoError(suite.T(), err)

	assert.Equal(suite.T(), map[string]bool{
		"author-1":     true,
		"author-empty": false,
	}, likes)
	assert.Equal(suite.T(), map[string]bool{
		"book-1":     true,
		"book-empty": false,
	}, shelves)
}

// TestGetViewerStatesDecodeError 测试单个 key 解码失败按 miss 处理
func (suite *FeedComposeCacheTestSuite) TestGetViewerStatesDecodeError() {
	ctx := context.Background()
	suite.Require().NoError(suite.mr.Set(KeyUserLike("author-bad", "viewer-1"), "{invalid"))
	suite.Require().NoError(suite.mr.Set(KeyUserBookshelf("viewer-1", "book-bad"), "{invalid"))

	likes, shelves, err := suite.cache.GetViewerStates(ctx, "viewer-1", []string{"author-bad"}, []string{"book-bad"})
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), likes)
	assert.Empty(suite.T(), shelves)
}

// TestGetViewerStatesPipelineError 测试整条 pipeline 失败返回 error
func (suite *FeedComposeCacheTestSuite) TestGetViewerStatesPipelineError() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = rdb.Close() }()
	c := NewFeedComposeCache(rdb)

	_, _, err := c.GetViewerStates(context.Background(), "viewer-1", []string{"author-1"}, []string{"book-1"})
	assert.Error(suite.T(), err)
}

// TestGetViewerStatesViewerIsolation 测试点赞/书架状态按查看者隔离
func (suite *FeedComposeCacheTestSuite) TestGetViewerStatesViewerIsolation() {
	ctx := context.Background()
	suite.setJSON(KeyUserLike("author-1", "viewer-a"), &model.UserLike{UserID: "author-1", LikerID: "viewer-a"})
	suite.setJSON(KeyUserBookshelf("viewer-a", "book-1"), &model.UserBookshelf{UserID: "viewer-a", BookID: "book-1"})

	likesA, shelvesA, err := suite.cache.GetViewerStates(ctx, "viewer-a", []string{"author-1"}, []string{"book-1"})
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), likesA["author-1"])
	assert.True(suite.T(), shelvesA["book-1"])

	likesB, shelvesB, err := suite.cache.GetViewerStates(ctx, "viewer-b", []string{"author-1"}, []string{"book-1"})
	assert.NoError(suite.T(), err)
	assert.Empty(suite.T(), likesB)
	assert.Empty(suite.T(), shelvesB)
}

func TestFeedComposeCacheTestSuite(t *testing.T) {
	suite.Run(t, new(FeedComposeCacheTestSuite))
}
