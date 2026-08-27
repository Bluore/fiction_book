package service

import (
	"context"
	"sync"
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
)

type userLikeServiceFixture struct {
	service      UserLikeService
	userLikeRepo repository.UserLikeRepository
	db           *gorm.DB
}

func newUserLikeServiceFixture(t *testing.T) *userLikeServiceFixture {
	t.Helper()

	db := newTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserLike{}))

	userRepo := repository.NewUserRepository(db)
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

	return &userLikeServiceFixture{
		service: NewUserLikeService(
			userRepo,
			userLikeRepo,
			cache.NewUserLikeCache(redisClient),
			db,
		),
		userLikeRepo: userLikeRepo,
		db:           db,
	}
}

func TestUserLikeService_UpdateUserLike(t *testing.T) {
	tests := []struct {
		name       string
		operators  []string
		wantCount  int
		wantErr    error
		concurrent bool
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
			name:       "concurrent create is idempotent",
			operators:  []string{userLikeOperatorCreate, userLikeOperatorCreate},
			wantCount:  1,
			concurrent: true,
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
			wantErr:   ErrParam,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newUserLikeServiceFixture(t)
			ctx := context.Background()

			errors := make([]error, len(tt.operators))
			if tt.concurrent {
				var group sync.WaitGroup
				for i, operator := range tt.operators {
					group.Add(1)
					go func(index int, operation string) {
						defer group.Done()
						_, errors[index] = fixture.service.UpdateUserLike(
							ctx,
							"liker-user",
							&dto.UpdateUserLikeRequest{
								TargetID: "target-user",
								Operator: operation,
							},
						)
					}(i, operator)
				}
				group.Wait()
			} else {
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

	assert.ErrorIs(t, err, ErrUserNotFound)
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

	assert.ErrorIs(t, err, ErrUserNotFound)
}
