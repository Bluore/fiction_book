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

func queryUserAssetDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userAssetCache cache.UserAssetCache,
	userAssetRepo repository.UserAssetRepository,
	userID string,
) (*model.UserAsset, error) {
	userAsset, cacheHit, err := userAssetCache.GetUserAsset(ctx, userID)
	if err != nil {
		logger.Warn("get user_asset cache fail", zap.Error(err))
	}
	if cacheHit {
		return userAsset, nil
	}

	token, locked, err := userAssetCache.LockUserAsset(ctx, userID)
	if err != nil {
		logger.Warn("lock user_asset cache fail", zap.Error(err))
		return userAssetRepo.GetByID(ctx, db, userID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userAsset, cacheHit, _ = userAssetCache.GetUserAsset(ctx, userID)
			if cacheHit {
				return userAsset, nil
			}
		}
		return nil, fmt.Errorf("user_asset cache query too many fail")
	}
	defer func() {
		if err := userAssetCache.UnlockUserAsset(ctx, userID, token); err != nil {
			logger.Error("unlock user_asset cache fail", zap.Error(err))
		}
	}()

	userAsset, err = userAssetRepo.GetByID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if err := userAssetCache.UpdateUserAsset(ctx, userID, userAsset); err != nil {
		logger.Warn("update user_asset cache fail", zap.Error(err))
	}
	return userAsset, nil
}

// Deprecated: createUserAssetDetailWithCache 旧版创建用户资产带缓存，将被移除
func createUserAssetDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userAssetCache cache.UserAssetCache,
	userAssetRepo repository.UserAssetRepository,
	userID string,
) error {
	var token string
	var locked bool
	var err error

	// 查询是否已存在，保持幂等
	userAssetExist, err := queryUserAssetDetailWithCache(ctx, db, userAssetCache, userAssetRepo, userID)
	if err != nil {
		return err
	}
	if userAssetExist != nil {
		return nil
	}

	// 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userAssetCache.LockUserAsset(ctx, userID)
		if err != nil {
			logger.Warn("lock user_asset cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_asset cache lock too many fail")
	}
	defer func() {
		if err := userAssetCache.UnlockUserAsset(ctx, userID, token); err != nil {
			logger.Error("unlock user_asset cache fail", zap.Error(err))
		}
	}()

	// 删除记录
	_, err = userAssetCache.DeleteUserAsset(ctx, userID)
	if err != nil {
		logger.Warn("delete user_asset cache fail", zap.Error(err))
	}

	// 锁内二次查询，并发下另一请求可能已创建，保持幂等
	userAssetExist, err = userAssetRepo.GetByID(ctx, db, userID)
	if err != nil {
		return err
	}
	if userAssetExist != nil {
		if err := userAssetCache.UpdateUserAsset(ctx, userAssetExist.ID, userAssetExist); err != nil {
			logger.Warn("update user_asset cache fail", zap.Error(err))
		}
		return nil
	}

	// 更新数据
	err = userAssetRepo.CreateUserAsset(ctx, db, userID)
	if err != nil {
		return err
	}

	// 回读完整记录并写入缓存
	userAsset, err := userAssetRepo.GetByID(ctx, db, userID)
	if err != nil {
		return err
	}
	if err := userAssetCache.UpdateUserAsset(ctx, userID, userAsset); err != nil {
		logger.Warn("update user_asset cache fail", zap.Error(err))
	}
	return nil
}

// Deprecated: updateUserAssetDetailWithCache 旧版更新用户资产带缓存，将被移除
func updateUserAssetDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userAssetCache cache.UserAssetCache,
	userAssetRepo repository.UserAssetRepository,
	userAsset *model.UserAsset,
) error {
	if userAsset == nil {
		return fmt.Errorf("userAsset is nil")
	}

	var token string
	var locked bool
	var err error

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userAssetCache.LockUserAsset(ctx, userAsset.ID)
		if err != nil {
			logger.Warn("lock user_asset cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_asset cache lock too many fail")
	}
	defer func() {
		if err := userAssetCache.UnlockUserAsset(ctx, userAsset.ID, token); err != nil {
			logger.Error("unlock user_asset cache fail", zap.Error(err))
		}
	}()

	_, err = userAssetCache.DeleteUserAsset(ctx, userAsset.ID)
	if err != nil {
		logger.Warn("delete user_asset cache fail", zap.Error(err))
	}

	err = userAssetRepo.UpdateByID(ctx, db, userAsset.ID, userAsset)
	if err != nil {
		return err
	}

	if err := userAssetCache.UpdateUserAsset(ctx, userAsset.ID, userAsset); err != nil {
		logger.Warn("update user_asset cache fail", zap.Error(err))
	}
	return nil
}

// Deprecated: deleteUserAssetDetailWithCache 旧版删除用户资产带缓存，将被移除
func deleteUserAssetDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userAssetCache cache.UserAssetCache,
	userAssetRepo repository.UserAssetRepository,
	userID string,
) error {
	var token string
	var locked bool
	var err error

	// 查询是否已存在，保持幂等
	userAssetExist, err := queryUserAssetDetailWithCache(ctx, db, userAssetCache, userAssetRepo, userID)
	if err != nil {
		return err
	}
	if userAssetExist == nil {
		return nil
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userAssetCache.LockUserAsset(ctx, userID)
		if err != nil {
			logger.Warn("lock user_asset cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_asset cache lock too many fail")
	}
	defer func() {
		if err := userAssetCache.UnlockUserAsset(ctx, userID, token); err != nil {
			logger.Error("unlock user_asset cache fail", zap.Error(err))
		}
	}()

	_, err = userAssetCache.DeleteUserAsset(ctx, userID)
	if err != nil {
		logger.Warn("delete user_asset cache fail", zap.Error(err))
	}

	err = userAssetRepo.DeleteByID(ctx, db, userID)
	if err != nil {
		return err
	}

	if err := userAssetCache.UpdateUserAsset(ctx, userID, nil); err != nil {
		logger.Warn("update user_asset cache fail", zap.Error(err))
	}
	return nil
}
