package cache

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

// BookThermalCacheTestSuite 书籍热度缓存测试套件
type BookThermalCacheTestSuite struct {
	suite.Suite
	mr    *miniredis.Miniredis
	rdb   *redis.Client
	cache BookThermalCache
}

func (suite *BookThermalCacheTestSuite) SetupSuite() {
	mr, err := miniredis.Run()
	assert.NoError(suite.T(), err)

	suite.mr = mr
	suite.rdb = redis.NewClient(&redis.Options{Addr: mr.Addr()})
	suite.cache = NewBookThermalCache(suite.rdb)
}

func (suite *BookThermalCacheTestSuite) TearDownSuite() {
	_ = suite.rdb.Close()
	suite.mr.Close()
}

// TearDownTest 每个测试后清空 redis 数据
func (suite *BookThermalCacheTestSuite) TearDownTest() {
	suite.mr.FlushAll()
}

// TestKeyFormat 测试 key 的拼接格式
func (suite *BookThermalCacheTestSuite) TestKeyFormat() {
	assert.Equal(suite.T(), "GSF:book:cache:book:book-1:thermal", KeyBookThermalValue("book-1"))
	assert.Equal(suite.T(), "GSF:book:lock:book:book-1:thermal", BookThermalTimeLockKey("book-1"))
}

// TestGetBookThermalValueNotExist 测试 key 不存在时返回 exist=false 且不报错
func (suite *BookThermalCacheTestSuite) TestGetBookThermalValueNotExist() {
	ctx := context.Background()

	value, exist, err := suite.cache.GetBookThermalValue(ctx, "not-exist-book")
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exist)
	assert.Equal(suite.T(), BookThermalValue(0), value)
}

// TestUpdateBookThermalValueCreate 测试 key 不存在时创建热度值并带上过期时间
func (suite *BookThermalCacheTestSuite) TestUpdateBookThermalValueCreate() {
	ctx := context.Background()
	bookID := "book-create"

	err := suite.cache.UpdateBookThermalValue(ctx, bookID, 100)
	assert.NoError(suite.T(), err)

	value, exist, err := suite.cache.GetBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exist)
	assert.Equal(suite.T(), BookThermalValue(100), value)

	// NX 写入时应当设置过期时间
	assert.Equal(suite.T(), bookThermalValueCacheTime, suite.mr.TTL(KeyBookThermalValue(bookID)))
}

// TestUpdateBookThermalValueNotOverwrite 测试 NX 语义：key 已存在时不覆盖已有热度值
func (suite *BookThermalCacheTestSuite) TestUpdateBookThermalValueNotOverwrite() {
	ctx := context.Background()
	bookID := "book-nx"

	assert.NoError(suite.T(), suite.cache.UpdateBookThermalValue(ctx, bookID, 100))

	// 第二次写入不应该覆盖，也不应该返回错误
	err := suite.cache.UpdateBookThermalValue(ctx, bookID, 999)
	assert.NoError(suite.T(), err)

	value, exist, err := suite.cache.GetBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exist)
	assert.Equal(suite.T(), BookThermalValue(100), value)
}

// TestAddBookThermalValueNotExist 测试 key 不存在时自增返回 ErrorBookThermalValueNotExist
func (suite *BookThermalCacheTestSuite) TestAddBookThermalValueNotExist() {
	ctx := context.Background()

	err := suite.cache.AddBookThermalValue(ctx, "book-add-miss", 5)
	assert.ErrorIs(suite.T(), err, ErrorBookThermalValueNotExist)

	// 不应该因为自增而创建 key
	assert.False(suite.T(), suite.mr.Exists(KeyBookThermalValue("book-add-miss")))
}

// TestAddBookThermalValue 测试 key 存在时热度值累加
func (suite *BookThermalCacheTestSuite) TestAddBookThermalValue() {
	ctx := context.Background()
	bookID := "book-add"

	assert.NoError(suite.T(), suite.cache.UpdateBookThermalValue(ctx, bookID, 10))

	assert.NoError(suite.T(), suite.cache.AddBookThermalValue(ctx, bookID, 5))
	assert.NoError(suite.T(), suite.cache.AddBookThermalValue(ctx, bookID, 7))

	value, exist, err := suite.cache.GetBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exist)
	assert.Equal(suite.T(), BookThermalValue(22), value)
}

// TestAddBookThermalValueSetTTLWhenMissing 测试已存在但无过期时间的 key 会被补上过期时间
func (suite *BookThermalCacheTestSuite) TestAddBookThermalValueSetTTLWhenMissing() {
	ctx := context.Background()
	bookID := "book-add-no-ttl"
	key := KeyBookThermalValue(bookID)

	// 手动写入一个没有过期时间的 key
	assert.NoError(suite.T(), suite.mr.Set(key, "10"))
	assert.Equal(suite.T(), time.Duration(0), suite.mr.TTL(key))

	assert.NoError(suite.T(), suite.cache.AddBookThermalValue(ctx, bookID, 5))

	value, exist, err := suite.cache.GetBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.True(suite.T(), exist)
	assert.Equal(suite.T(), BookThermalValue(15), value)
	assert.Equal(suite.T(), bookThermalValueCacheTime, suite.mr.TTL(key))
}

// TestAddBookThermalValueKeepTTL 测试已有过期时间的 key 不会被重置过期时间
func (suite *BookThermalCacheTestSuite) TestAddBookThermalValueKeepTTL() {
	ctx := context.Background()
	bookID := "book-keep-ttl"
	key := KeyBookThermalValue(bookID)

	assert.NoError(suite.T(), suite.mr.Set(key, "10"))
	suite.mr.SetTTL(key, time.Minute)

	assert.NoError(suite.T(), suite.cache.AddBookThermalValue(ctx, bookID, 1))
	assert.Equal(suite.T(), time.Minute, suite.mr.TTL(key))
}

// TestLockBookThermalValue 测试加锁成功后重复加锁返回 ErrorBookThermalLockExist
func (suite *BookThermalCacheTestSuite) TestLockBookThermalValue() {
	ctx := context.Background()
	bookID := "book-lock"

	err := suite.cache.LockBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), bookThermalLockCacheTime, suite.mr.TTL(BookThermalTimeLockKey(bookID)))

	// 锁有效期内重复加锁失败
	err = suite.cache.LockBookThermalValue(ctx, bookID)
	assert.ErrorIs(suite.T(), err, ErrorBookThermalLockExist)
}

// TestLockBookThermalValueAfterExpire 测试锁过期后可以重新加锁
func (suite *BookThermalCacheTestSuite) TestLockBookThermalValueAfterExpire() {
	ctx := context.Background()
	bookID := "book-lock-expire"

	assert.NoError(suite.T(), suite.cache.LockBookThermalValue(ctx, bookID))

	// 推进 miniredis 时钟使锁过期
	suite.mr.FastForward(bookThermalLockCacheTime + time.Second)

	assert.NoError(suite.T(), suite.cache.LockBookThermalValue(ctx, bookID))
}

// TestBookThermalValueExpire 测试热度值缓存到期后失效
func (suite *BookThermalCacheTestSuite) TestBookThermalValueExpire() {
	ctx := context.Background()
	bookID := "book-expire"

	assert.NoError(suite.T(), suite.cache.UpdateBookThermalValue(ctx, bookID, 42))

	suite.mr.FastForward(bookThermalValueCacheTime + time.Second)

	_, exist, err := suite.cache.GetBookThermalValue(ctx, bookID)
	assert.NoError(suite.T(), err)
	assert.False(suite.T(), exist)
}

// TestRedisUnavailable 测试 redis 不可用时各方法返回错误
func (suite *BookThermalCacheTestSuite) TestRedisUnavailable() {
	ctx := context.Background()

	// 指向一个没有服务监听的地址
	rdb := redis.NewClient(&redis.Options{Addr: "127.0.0.1:1"})
	defer func() { _ = rdb.Close() }()
	c := NewBookThermalCache(rdb)

	_, exist, err := c.GetBookThermalValue(ctx, "book")
	assert.Error(suite.T(), err)
	assert.False(suite.T(), exist)

	assert.Error(suite.T(), c.UpdateBookThermalValue(ctx, "book", 1))

	err = c.AddBookThermalValue(ctx, "book", 1)
	assert.Error(suite.T(), err)
	// 连接错误不能被误判为「缓存不存在」
	assert.NotErrorIs(suite.T(), err, ErrorBookThermalValueNotExist)

	err = c.LockBookThermalValue(ctx, "book")
	assert.Error(suite.T(), err)
	assert.NotErrorIs(suite.T(), err, ErrorBookThermalLockExist)
}

func TestBookThermalCacheTestSuite(t *testing.T) {
	suite.Run(t, new(BookThermalCacheTestSuite))
}
