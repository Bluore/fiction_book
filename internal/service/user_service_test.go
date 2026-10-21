package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/response"
)

type userServiceFixture struct {
	service       UserService
	userRepo      repository.UserRepository
	userCache     cache.UserCache
	db            *gorm.DB
	userAssetRepo *MockUserAssetRepository
}

func newUserServiceFixture(t *testing.T) *userServiceFixture {
	t.Helper()

	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserInfo{}))

	userRepo := repository.NewUserRepository(db)
	userAssetRepo := new(MockUserAssetRepository)
	userInfoRepo := repository.NewUserInfoRepository(db)

	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		require.NoError(t, redisClient.Close())
	})

	cfg := newTestConfig()
	cfg.Server.Mode = "debug"

	svc := NewUserService(
		userRepo,
		userAssetRepo,
		userInfoRepo,
		cache.NewUserCache(redisClient),
		cache.NewUserStatCache(redisClient),
		cache.NewUserAssetCache(redisClient),
		cache.NewUserInfoCache(redisClient),
		cache.NewAuthCache(redisClient),
		db,
		cfg,
	)

	return &userServiceFixture{
		service:       svc,
		userRepo:      userRepo,
		userCache:     cache.NewUserCache(redisClient),
		db:            db,
		userAssetRepo: userAssetRepo,
	}
}

// createFixtureUser 通过服务创建用户，并完成资产仓储的期望设置
func createFixtureUser(t *testing.T, fixture *userServiceFixture, username, email, password string) (*dto.UserResponse, error) {
	t.Helper()
	fixture.userAssetRepo.On("CreateUserAsset", mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	resp, err := fixture.service.Create(context.Background(), &dto.CreateUserRequest{
		Username: username,
		Email:    email,
		Password: password,
	})
	fixture.userAssetRepo.AssertExpectations(t)
	return resp, err
}

func TestUserService_Create(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "alice", "alice@example.com", "password123")
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "alice", resp.Username)
	assert.Equal(t, "alice@example.com", resp.Email)

	// 数据库中存在用户
	dbUser, err := fixture.userRepo.GetByID(ctx, fixture.db, resp.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	assert.True(t, checkPasswordHash("password123", dbUser.Password))

	// 用户详情同步创建
	dbUserInfo, err := repository.NewUserInfoRepository(fixture.db).GetByUserID(ctx, fixture.db, resp.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUserInfo)
	assert.Equal(t, "alice", dbUserInfo.Username)

	// 事务提交后缓存已回写
	cachedUser, hit, err := fixture.userCache.GetUser(ctx, resp.ID)
	require.NoError(t, err)
	assert.True(t, hit)
	require.NotNil(t, cachedUser)
	assert.Equal(t, "alice", cachedUser.Username)
}

func TestUserService_GetByID_UsesCache(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "bob", "bob@example.com", "password123")
	require.NoError(t, err)

	// 直接从数据库删除用户，缓存仍保留，GetByID 应命中缓存
	require.NoError(t, fixture.userRepo.Delete(ctx, fixture.db, resp.ID))

	got, err := fixture.service.GetByID(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "bob", got.Username)

	// 不存在的用户返回 response.ErrUserNotFound
	_, err = fixture.service.GetByID(ctx, "missing-user")
	assert.ErrorIs(t, err, response.ErrUserNotFound)
}

func TestUserService_Update_KeepsPasswordHash(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "carol", "carol@example.com", "password123")
	require.NoError(t, err)

	username := "carol-new"
	age := 30
	description := "hello world"
	updated, err := fixture.service.Update(ctx, resp.ID, &dto.UpdateUserRequest{
		Username:    &username,
		Age:         &age,
		Description: &description,
	})
	require.NoError(t, err)
	assert.Equal(t, "carol-new", updated.Username)
	assert.Equal(t, 30, updated.Age)

	// 更新后数据库密码哈希保持不变
	dbUser, err := fixture.userRepo.GetByID(ctx, fixture.db, resp.ID)
	require.NoError(t, err)
	require.NotNil(t, dbUser)
	assert.True(t, checkPasswordHash("password123", dbUser.Password))

	// 先更新数据库，再删除缓存：更新后缓存失效，需回源数据库
	cachedUser, hit, err := fixture.userCache.GetUser(ctx, resp.ID)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Nil(t, cachedUser)

	// 回源后缓存重建，且缓存中不包含密码
	got, err := fixture.service.GetByID(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "carol-new", got.Username)
	cachedUser, hit, err = fixture.userCache.GetUser(ctx, resp.ID)
	require.NoError(t, err)
	assert.True(t, hit)
	require.NotNil(t, cachedUser)
	assert.Equal(t, "carol-new", cachedUser.Username)
	assert.Equal(t, 30, cachedUser.Age)
	assert.Empty(t, cachedUser.Password)
}

func TestUserService_UpdateEmail(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "dave", "dave@example.com", "password123")
	require.NoError(t, err)

	// 密码错误
	_, err = fixture.service.UpdateEmail(ctx, resp.ID, &dto.UpdateUserEmailRequest{
		Password: "wrong-password",
		Email:    "dave-new@example.com",
		Verify:   "123456",
	})
	assert.ErrorIs(t, err, response.ErrInvalidPassword)

	// 密码正确，邮箱更新成功，缓存失效需回源
	updated, err := fixture.service.UpdateEmail(ctx, resp.ID, &dto.UpdateUserEmailRequest{
		Password: "password123",
		Email:    "dave-new@example.com",
		Verify:   "123456",
	})
	require.NoError(t, err)
	assert.Equal(t, "dave-new@example.com", updated.Email)

	cachedUser, hit, err := fixture.userCache.GetUser(ctx, resp.ID)
	require.NoError(t, err)
	assert.False(t, hit)
	assert.Nil(t, cachedUser)

	got, err := fixture.service.GetByID(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, "dave-new@example.com", got.Email)

	// 邮箱已存在
	_, err = createFixtureUser(t, fixture, "eve", "eve@example.com", "password123")
	require.NoError(t, err)
	_, err = fixture.service.UpdateEmail(ctx, resp.ID, &dto.UpdateUserEmailRequest{
		Password: "password123",
		Email:    "eve@example.com",
		Verify:   "123456",
	})
	assert.ErrorIs(t, err, response.ErrEmailExists)
}

func TestUserService_Delete(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "frank", "frank@example.com", "password123")
	require.NoError(t, err)

	require.NoError(t, fixture.service.Delete(ctx, resp.ID))

	// 数据库已删除，缓存已失效
	dbUser, err := fixture.userRepo.GetByID(ctx, fixture.db, resp.ID)
	require.NoError(t, err)
	assert.Nil(t, dbUser)

	_, hit, err := fixture.userCache.GetUser(ctx, resp.ID)
	require.NoError(t, err)
	assert.False(t, hit)

	_, err = fixture.service.GetByID(ctx, resp.ID)
	assert.ErrorIs(t, err, response.ErrUserNotFound)

	// 再次删除保持幂等语义（用户不存在返回 response.ErrUserNotFound）
	err = fixture.service.Delete(ctx, resp.ID)
	assert.ErrorIs(t, err, response.ErrUserNotFound)
}

func TestUserService_GetStatByID(t *testing.T) {
	fixture := newUserServiceFixture(t)
	ctx := context.Background()

	resp, err := createFixtureUser(t, fixture, "grace", "grace@example.com", "password123")
	require.NoError(t, err)

	fixture.userAssetRepo.On("GetByID", mock.Anything, mock.Anything, resp.ID).Return(&model.UserAsset{
		ID:       resp.ID,
		Gold:     100,
		VIPLevel: "vip_0",
	}, nil).Once()

	stat, err := fixture.service.GetStatByID(ctx, resp.ID)
	require.NoError(t, err)
	assert.Equal(t, resp.ID, stat.ID)
	assert.Equal(t, int64(100), stat.Gold)
	assert.Equal(t, 0, stat.VIPLevel)

	fixture.userAssetRepo.AssertExpectations(t)

	// 用户不存在
	_, err = fixture.service.GetStatByID(ctx, "missing-user")
	assert.ErrorIs(t, err, response.ErrUserNotFound)
}
