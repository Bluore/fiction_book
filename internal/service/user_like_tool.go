package service

import (
	"context"
	"fmt"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func queryUserLikeDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	userID string,
	likerID string,
) (*model.UserLike, error) {
	userLike, cacheHit, err := userLikeCache.GetUserLike(ctx, userID, likerID)
	if err != nil {
		logger.Warn("get user_like cache fail", zap.Error(err))
	}
	if cacheHit {
		return userLike, nil
	}

	token, locked, err := userLikeCache.LockUserLike(ctx, userID, likerID)
	if err != nil {
		logger.Warn("lock user_like cache fail", zap.Error(err))
		return userLikeRepo.GetByUserIDAndLikerID(ctx, db, userID, likerID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userLike, cacheHit, _ = userLikeCache.GetUserLike(ctx, userID, likerID)
			if cacheHit {
				return userLike, nil
			}
		}
		return nil, fmt.Errorf("user_like cache query too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLike(ctx, userID, likerID, token); err != nil {
			logger.Error("unlock user_like cache fail", zap.Error(err))
		}
	}()

	userLike, err = userLikeRepo.GetByUserIDAndLikerID(ctx, db, userID, likerID)
	if err != nil {
		return nil, err
	}
	if err := userLikeCache.UpdateUserLike(ctx, userID, likerID, userLike); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return userLike, nil
}

func createUserLikeDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	userLike *model.UserLike,
) error {
	if userLike == nil {
		return fmt.Errorf("userLike is nil")
	}

	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userLikeExist, err := queryUserLikeDetailWithCache(ctx, db, userLikeCache, userLikeRepo, userLike.UserID, userLike.LikerID)
	if err != nil {
		return err
	}
	if userLikeExist != nil {
		return nil
	}

	// FOCUS 删除记录
	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

	// FOCUS 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userLikeCache.LockUserLike(ctx, userLike.UserID, userLike.LikerID)
		if err != nil {
			logger.Warn("lock user_like cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_like cache lock too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLike(ctx, userLike.UserID, userLike.LikerID, token); err != nil {
			logger.Error("unlock user_like cache fail", zap.Error(err))
		}
	}()

	// FOCUS 锁内二次查询，并发下另一请求可能已创建，保持幂等
	userLikeExist, err = userLikeRepo.GetByUserIDAndLikerID(ctx, db, userLike.UserID, userLike.LikerID)
	if err != nil {
		return err
	}
	if userLikeExist != nil {
		if err := userLikeCache.UpdateUserLike(ctx, userLikeExist.UserID, userLikeExist.LikerID, userLikeExist); err != nil {
			logger.Warn("update user_like cache fail", zap.Error(err))
		}
		return nil
	}

	// FOCUS 更新数据
	err = userLikeRepo.Create(ctx, db, userLike)
	if err != nil {
		return err
	}

	if err := userLikeCache.UpdateUserLike(ctx, userLike.UserID, userLike.LikerID, userLike); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return nil
}

func updateUserLikeDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	userLike *model.UserLike,
) error {
	if userLike == nil {
		return fmt.Errorf("userLike is nil")
	}

	var token string
	var locked bool
	var err error

	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userLikeCache.LockUserLike(ctx, userLike.UserID, userLike.LikerID)
		if err != nil {
			logger.Warn("lock user_like cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_like cache lock too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLike(ctx, userLike.UserID, userLike.LikerID, token); err != nil {
			logger.Error("unlock user_like cache fail", zap.Error(err))
		}
	}()

	err = userLikeRepo.Update(ctx, db, userLike)
	if err != nil {
		return err
	}

	if err := userLikeCache.UpdateUserLike(ctx, userLike.UserID, userLike.LikerID, userLike); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return nil
}

func deleteUserLikeDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	userLike *model.UserLike,
) error {
	if userLike == nil {
		return fmt.Errorf("userLike is nil")
	}

	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userLikeExist, err := queryUserLikeDetailWithCache(ctx, db, userLikeCache, userLikeRepo, userLike.UserID, userLike.LikerID)
	if err != nil {
		return err
	}
	if userLikeExist == nil {
		return nil
	}

	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userLikeCache.LockUserLike(ctx, userLike.UserID, userLike.LikerID)
		if err != nil {
			logger.Warn("lock user_like cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_like cache lock too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLike(ctx, userLike.UserID, userLike.LikerID, token); err != nil {
			logger.Error("unlock user_like cache fail", zap.Error(err))
		}
	}()

	err = userLikeRepo.DeleteByUserIDAndLikerID(ctx, db, userLike.UserID, userLike.LikerID)
	if err != nil {
		return err
	}

	if err := userLikeCache.UpdateUserLike(ctx, userLike.UserID, userLike.LikerID, nil); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return nil
}
