package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// UserFeedCacheTestSuite 用户 Feed 缓存测试套件
type UserFeedCacheTestSuite struct {
	suite.Suite
	mr    *miniredis.Miniredis
	rdb   *redis.Client
	cache UserFeedCache
}

func (suite *UserFeedCacheTestSuite) SetupSuite() {
	mr, err := miniredis.Run()
	assert.NoError(suite.T(), err)

	suite.mr = mr
	suite.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	suite.cache = NewUserFeedCache(suite.rdb)
}

func (suite *UserFeedCacheTestSuite) TearDownSuite() {
	_ = suite.rdb.Close()
	suite.mr.Close()
}

func (suite *UserFeedCacheTestSuite) TearDownTest() {
	suite.mr.FlushAll()
}

// TestGetUserFeedFeedIDsKeyNotExist 测试 key 不存在时返回 cacheHit=false
func (suite *UserFeedCacheTestSuite) TestGetUserFeedFeedIDsKeyNotExist() {
	feedIDs, cacheHit, err := suite.cache.GetUserFeedFeedIDs(context.Background(), "user-1", "cursor-1", 0, 10)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), cacheHit)
	assert.Nil(suite.T(), feedIDs)
}

// TestGetUserFeedFeedIDsEmpty 测试 key 存在但 limit<=0 时返回空列表与 cacheHit=true
func (suite *UserFeedCacheTestSuite) TestGetUserFeedFeedIDsEmpty() {
	ctx := context.Background()
	assert.NoError(suite.T(), suite.cache.UpdateUserFeedFeedIDs(ctx, "user-1", "cursor-1", []string{"f1", "f2"}))

	feedIDs, cacheHit, err := suite.cache.GetUserFeedFeedIDs(ctx, "user-1", "cursor-1", 0, 0)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), cacheHit)
	assert.Empty(suite.T(), feedIDs)

	feedIDs, cacheHit, err = suite.cache.GetUserFeedFeedIDs(ctx, "user-1", "cursor-1", 0, -1)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), cacheHit)
	assert.Empty(suite.T(), feedIDs)
}

// TestGetUserFeedFeedIDsPaging 测试 LRANGE 分页与 pipeline 结果
func (suite *UserFeedCacheTestSuite) TestGetUserFeedFeedIDsPaging() {
	ctx := context.Background()
	assert.NoError(suite.T(), suite.cache.UpdateUserFeedFeedIDs(ctx, "user-1", "cursor-1", []string{"f1", "f2", "f3", "f4"}))

	feedIDs, cacheHit, err := suite.cache.GetUserFeedFeedIDs(ctx, "user-1", "cursor-1", 1, 2)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), cacheHit)
	assert.Equal(suite.T(), []string{"f2", "f3"}, feedIDs)

	feedIDs, cacheHit, err = suite.cache.GetUserFeedFeedIDs(ctx, "user-1", "cursor-1", 10, 2)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), cacheHit)
	assert.Empty(suite.T(), feedIDs)
}

// TestGetUserFeedFeedIDsRedisUnavailable 测试 Redis 不可用时返回错误
func (suite *UserFeedCacheTestSuite) TestGetUserFeedFeedIDsRedisUnavailable() {
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = rdb.Close() }()
	c := NewUserFeedCache(rdb)

	_, cacheHit, err := c.GetUserFeedFeedIDs(context.Background(), "user-1", "cursor-1", 0, 10)
	assert.Error(suite.T(), err)
	assert.False(suite.T(), cacheHit)
}

func TestUserFeedCacheTestSuite(t *testing.T) {
	suite.Run(t, new(UserFeedCacheTestSuite))
}
