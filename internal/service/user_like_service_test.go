package service

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/response"
)

type userLikeServiceFixture struct {
	service      UserLikeService
	userLikeRepo repository.UserLikeRepository
	userInfoRepo repository.UserInfoRepository
	db           *gorm.DB
}

func newUserLikeServiceFixture(t *testing.T) *userLikeServiceFixture {
	t.Helper()

	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserInfo{}, &model.UserLike{}))

	userRepo := repository.NewUserRepository(db)
	userInfoRepo := repository.NewUserInfoRepository(db)
	userLikeRepo := repository.NewUserLikeRepository(db)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	t.Cleanup(func() {
		require.NoError(t, redisClient.Close())
	})

	targetUser := &model.User{
		ID:       "target-user",
		Username: "target",
		Email:    "target@example.com",
		Password: "password",
	}
	require.NoError(t, userRepo.Create(context.Background(), db, targetUser))
	require.NoError(t, userInfoRepo.Create(context.Background(), db, &model.UserInfo{
		UserID:   targetUser.ID,
		Username: targetUser.Username,
	}))

	return &userLikeServiceFixture{
		service: NewUserLikeService(
			userRepo,
			userInfoRepo,
			userLikeRepo,
			cache.NewUserLikeCache(redisClient),
			cache.NewUserInfoCache(redisClient),
			db,
		),
		userLikeRepo: userLikeRepo,
		userInfoRepo: userInfoRepo,
		db:           db,
	}
}

func TestUserLikeService_UpdateUserLike(t *testing.T) {
	tests := []struct {
		name      string
		operators []string
		wantCount int
		wantErr   error
	}{
		{
			name:      "create like",
			operators: []string{userLikeOperatorCreate},
			wantCount: 1,
		},
		{
			name:      "duplicate create is idempotent",
			operators: []string{userLikeOperatorCreate, userLikeOperatorCreate},
			wantCount: 1,
		},
		{
			name:      "delete existing like",
			operators: []string{userLikeOperatorCreate, userLikeOperatorDelete},
			wantCount: 0,
		},
		{
			name:      "delete missing like is idempotent",
			operators: []string{userLikeOperatorDelete},
			wantCount: 0,
		},
		{
			name:      "reject invalid operator",
			operators: []string{"toggle"},
			wantCount: 0,
			wantErr:   response.ErrParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newUserLikeServiceFixture(t)
			ctx := context.Background()

			errors := make([]error, len(tt.operators))
			for i, operator := range tt.operators {
				_, errors[i] = fixture.service.UpdateUserLike(
					ctx,
					"liker-user",
					&dto.UpdateUserLikeRequest{
						TargetID: "target-user",
						Operator: operator,
					},
				)
			}

			for _, err := range errors {
				if tt.wantErr != nil {
					assert.ErrorIs(t, err, tt.wantErr)
					continue
				}
				assert.NoError(t, err)
			}

			count, err := fixture.userLikeRepo.CountByUserID(ctx, fixture.db, "target-user")
			require.NoError(t, err)
			assert.Equal(t, tt.wantCount, count)
		})
	}
}

func TestUserLikeService_UpdateUserLikeFansAndBigV(t *testing.T) {
	tests := []struct {
		name        string
		initialFans int64
		operators   []string
		wantFansNum int64
		wantIsV     int
	}{
		{
			name:        "first follower stays small v",
			initialFans: 0,
			operators:   []string{userLikeOperatorCreate},
			wantFansNum: 1,
			wantIsV:     0,
		},
		{
			name:        "reaching threshold marks big v",
			initialFans: 99,
			operators:   []string{userLikeOperatorCreate},
			wantFansNum: 100,
			wantIsV:     1,
		},
		{
			name:        "duplicate follow counts once",
			initialFans: 98,
			operators:   []string{userLikeOperatorCreate, userLikeOperatorCreate},
			wantFansNum: 99,
			wantIsV:     0,
		},
		{
			name:        "unfollow before threshold stays small v",
			initialFans: 98,
			operators:   []string{userLikeOperatorCreate, userLikeOperatorDelete},
			wantFansNum: 98,
			wantIsV:     0,
		},
		{
			name:        "unfollow after promotion keeps big v",
			initialFans: 99,
			operators:   []string{userLikeOperatorCreate, userLikeOperatorDelete},
			wantFansNum: 99,
			wantIsV:     1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newUserLikeServiceFixture(t)
			ctx := context.Background()

			if tt.initialFans > 0 {
				require.NoError(t, fixture.userInfoRepo.IncrementFansNum(ctx, fixture.db, "target-user", tt.initialFans))
			}

			for _, operator := range tt.operators {
				_, err := fixture.service.UpdateUserLike(
					ctx,
					"liker-user",
					&dto.UpdateUserLikeRequest{
						TargetID: "target-user",
						Operator: operator,
					},
				)
				require.NoError(t, err)
			}

			userInfo, err := fixture.userInfoRepo.GetByUserID(ctx, fixture.db, "target-user")
			require.NoError(t, err)
			require.NotNil(t, userInfo)
			assert.Equal(t, tt.wantFansNum, userInfo.FansNum)
			assert.Equal(t, tt.wantIsV, userInfo.IsV)
		})
	}
}

func TestUserLikeService_UpdateUserLikeTargetNotFound(t *testing.T) {
	fixture := newUserLikeServiceFixture(t)

	_, err := fixture.service.UpdateUserLike(
		context.Background(),
		"liker-user",
		&dto.UpdateUserLikeRequest{
			TargetID: "missing-user",
			Operator: userLikeOperatorCreate,
		},
	)

	assert.ErrorIs(t, err, response.ErrUserNotFound)
}

func TestUserLikeService_GetUserLike(t *testing.T) {
	fixture := newUserLikeServiceFixture(t)
	ctx := context.Background()
	req := &dto.GetUserLikeRequest{TargetID: "target-user"}

	response, err := fixture.service.GetUserLike(ctx, "liker-user", req)
	require.NoError(t, err)
	assert.Equal(t, "target-user", response.TargetID)
	assert.False(t, response.IsLike)

	_, err = fixture.service.UpdateUserLike(
		ctx,
		"liker-user",
		&dto.UpdateUserLikeRequest{
			TargetID: "target-user",
			Operator: userLikeOperatorCreate,
		},
	)
	require.NoError(t, err)

	response, err = fixture.service.GetUserLike(ctx, "liker-user", req)
	require.NoError(t, err)
	assert.True(t, response.IsLike)
}

func TestUserLikeService_GetUserLikeTargetNotFound(t *testing.T) {
	fixture := newUserLikeServiceFixture(t)

	_, err := fixture.service.GetUserLike(
		context.Background(),
		"liker-user",
		&dto.GetUserLikeRequest{TargetID: "missing-user"},
	)

	assert.ErrorIs(t, err, response.ErrUserNotFound)
}
