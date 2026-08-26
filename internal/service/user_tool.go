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

func queryUserDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userCache cache.UserCache,
	userRepo repository.UserRepository,
	userID string,
) (*model.User, error) {
	user, cacheHit, err := userCache.GetUser(ctx, userID)
	if err != nil {
		logger.Warn("get user cache fail", zap.Error(err))
	}
	if cacheHit {
		return user, nil
	}

	token, locked, err := userCache.LockUser(ctx, userID)
	if err != nil {
		logger.Warn("lock user cache fail", zap.Error(err))
		return userRepo.GetByID(ctx, db, userID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			user, cacheHit, _ = userCache.GetUser(ctx, userID)
			if cacheHit {
				return user, nil
			}
		}
		return nil, fmt.Errorf("user cache query too many fail")
	}
	defer func() {
		if err := userCache.UnlockUser(ctx, userID, token); err != nil {
			logger.Error("unlock user cache fail", zap.Error(err))
		}
	}()

	user, err = userRepo.GetByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if err := userCache.UpdateUser(ctx, userID, user); err != nil {
		logger.Warn("update user cache fail", zap.Error(err))
	}
	return user, nil
}

func createUserDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userCache cache.UserCache,
	userRepo repository.UserRepository,
	user *model.User,
) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userExist, err := queryUserDetailWithCache(ctx, db, userCache, userRepo, user.ID)
	if err != nil {
		return err
	}
	if userExist != nil {
		return nil
	}

	// FOCUS 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userCache.LockUser(ctx, user.ID)
		if err != nil {
			logger.Warn("lock user cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user cache lock too many fail")
	}
	defer func() {
		if err := userCache.UnlockUser(ctx, user.ID, token); err != nil {
			logger.Error("unlock user cache fail", zap.Error(err))
		}
	}()

	// FOCUS 删除记录
	_, err = userCache.DeleteUser(ctx, user.ID)
	if err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}

	// FOCUS 锁内二次查询，并发下另一请求可能已创建，保持幂等
	userExist, err = userRepo.GetByID(ctx, db, user.ID)
	if err != nil {
		return err
	}
	if userExist != nil {
		if err := userCache.UpdateUser(ctx, userExist.ID, userExist); err != nil {
			logger.Warn("update user cache fail", zap.Error(err))
		}
		return nil
	}

	// FOCUS 更新数据
	err = userRepo.Create(ctx, db, user)
	if err != nil {
		return err
	}

	if err := userCache.UpdateUser(ctx, user.ID, user); err != nil {
		logger.Warn("update user cache fail", zap.Error(err))
	}
	return nil
}

func updateUserDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userCache cache.UserCache,
	userRepo repository.UserRepository,
	user *model.User,
) error {
	if user == nil {
		return fmt.Errorf("user is nil")
	}

	var token string
	var locked bool
	var err error

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userCache.LockUser(ctx, user.ID)
		if err != nil {
			logger.Warn("lock user cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user cache lock too many fail")
	}
	defer func() {
		if err := userCache.UnlockUser(ctx, user.ID, token); err != nil {
			logger.Error("unlock user cache fail", zap.Error(err))
		}
	}()

	_, err = userCache.DeleteUser(ctx, user.ID)
	if err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}

	err = userRepo.Update(ctx, db, user)
	if err != nil {
		return err
	}

	if err := userCache.UpdateUser(ctx, user.ID, user); err != nil {
		logger.Warn("update user cache fail", zap.Error(err))
	}
	return nil
}

func deleteUserDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userCache cache.UserCache,
	userRepo repository.UserRepository,
	userID string,
) error {
	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userExist, err := queryUserDetailWithCache(ctx, db, userCache, userRepo, userID)
	if err != nil {
		return err
	}
	if userExist == nil {
		return nil
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userCache.LockUser(ctx, userID)
		if err != nil {
			logger.Warn("lock user cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user cache lock too many fail")
	}
	defer func() {
		if err := userCache.UnlockUser(ctx, userID, token); err != nil {
			logger.Error("unlock user cache fail", zap.Error(err))
		}
	}()

	_, err = userCache.DeleteUser(ctx, userID)
	if err != nil {
		logger.Warn("delete user cache fail", zap.Error(err))
	}

	err = userRepo.Delete(ctx, db, userID)
	if err != nil {
		return err
	}

	if err := userCache.UpdateUser(ctx, userID, nil); err != nil {
		logger.Warn("update user cache fail", zap.Error(err))
	}
	return nil
}
