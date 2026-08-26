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

func queryUserInfoDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userInfoCache cache.UserInfoCache,
	userInfoRepo repository.UserInfoRepository,
	userID string,
) (*model.UserInfo, error) {
	userInfo, cacheHit, err := userInfoCache.GetUserInfo(ctx, userID)
	if err != nil {
		logger.Warn("get user_info cache fail", zap.Error(err))
	}
	if cacheHit {
		return userInfo, nil
	}

	token, locked, err := userInfoCache.LockUserInfo(ctx, userID)
	if err != nil {
		logger.Warn("lock user_info cache fail", zap.Error(err))
		return userInfoRepo.GetByUserID(ctx, db, userID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userInfo, cacheHit, _ = userInfoCache.GetUserInfo(ctx, userID)
			if cacheHit {
				return userInfo, nil
			}
		}
		return nil, fmt.Errorf("user_info cache query too many fail")
	}
	defer func() {
		if err := userInfoCache.UnlockUserInfo(ctx, userID, token); err != nil {
			logger.Error("unlock user_info cache fail", zap.Error(err))
		}
	}()

	userInfo, err = userInfoRepo.GetByUserID(ctx, db, userID)
	if err != nil {
		return nil, err
	}
	if err := userInfoCache.UpdateUserInfo(ctx, userID, userInfo); err != nil {
		logger.Warn("update user_info cache fail", zap.Error(err))
	}
	return userInfo, nil
}

func createUserInfoDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userInfoCache cache.UserInfoCache,
	userInfoRepo repository.UserInfoRepository,
	userInfo *model.UserInfo,
) error {
	if userInfo == nil {
		return fmt.Errorf("userInfo is nil")
	}

	var token string
	var locked bool
	var err error

	// 查询是否已存在，保持幂等
	userInfoExist, err := queryUserInfoDetailWithCache(ctx, db, userInfoCache, userInfoRepo, userInfo.UserID)
	if err != nil {
		return err
	}
	if userInfoExist != nil {
		return nil
	}

	// 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userInfoCache.LockUserInfo(ctx, userInfo.UserID)
		if err != nil {
			logger.Warn("lock user_info cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_info cache lock too many fail")
	}
	defer func() {
		if err := userInfoCache.UnlockUserInfo(ctx, userInfo.UserID, token); err != nil {
			logger.Error("unlock user_info cache fail", zap.Error(err))
		}
	}()

	// 删除记录
	_, err = userInfoCache.DeleteUserInfo(ctx, userInfo.UserID)
	if err != nil {
		logger.Warn("delete user_info cache fail", zap.Error(err))
	}

	// 锁内二次查询，并发下另一请求可能已创建，保持幂等
	userInfoExist, err = userInfoRepo.GetByUserID(ctx, db, userInfo.UserID)
	if err != nil {
		return err
	}
	if userInfoExist != nil {
		if err := userInfoCache.UpdateUserInfo(ctx, userInfoExist.UserID, userInfoExist); err != nil {
			logger.Warn("update user_info cache fail", zap.Error(err))
		}
		return nil
	}

	// 更新数据
	err = userInfoRepo.Create(ctx, db, userInfo)
	if err != nil {
		return err
	}

	if err := userInfoCache.UpdateUserInfo(ctx, userInfo.UserID, userInfo); err != nil {
		logger.Warn("update user_info cache fail", zap.Error(err))
	}
	return nil
}

func updateUserInfoDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userInfoCache cache.UserInfoCache,
	userInfoRepo repository.UserInfoRepository,
	userInfo *model.UserInfo,
) error {
	if userInfo == nil {
		return fmt.Errorf("userInfo is nil")
	}

	var token string
	var locked bool
	var err error

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userInfoCache.LockUserInfo(ctx, userInfo.UserID)
		if err != nil {
			logger.Warn("lock user_info cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_info cache lock too many fail")
	}
	defer func() {
		if err := userInfoCache.UnlockUserInfo(ctx, userInfo.UserID, token); err != nil {
			logger.Error("unlock user_info cache fail", zap.Error(err))
		}
	}()

	_, err = userInfoCache.DeleteUserInfo(ctx, userInfo.UserID)
	if err != nil {
		logger.Warn("delete user_info cache fail", zap.Error(err))
	}

	err = userInfoRepo.Update(ctx, db, userInfo)
	if err != nil {
		return err
	}

	if err := userInfoCache.UpdateUserInfo(ctx, userInfo.UserID, userInfo); err != nil {
		logger.Warn("update user_info cache fail", zap.Error(err))
	}
	return nil
}

func deleteUserInfoDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userInfoCache cache.UserInfoCache,
	userInfoRepo repository.UserInfoRepository,
	userID string,
) error {
	var token string
	var locked bool
	var err error

	// 查询是否已存在，保持幂等
	userInfoExist, err := queryUserInfoDetailWithCache(ctx, db, userInfoCache, userInfoRepo, userID)
	if err != nil {
		return err
	}
	if userInfoExist == nil {
		return nil
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userInfoCache.LockUserInfo(ctx, userID)
		if err != nil {
			logger.Warn("lock user_info cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_info cache lock too many fail")
	}
	defer func() {
		if err := userInfoCache.UnlockUserInfo(ctx, userID, token); err != nil {
			logger.Error("unlock user_info cache fail", zap.Error(err))
		}
	}()

	_, err = userInfoCache.DeleteUserInfo(ctx, userID)
	if err != nil {
		logger.Warn("delete user_info cache fail", zap.Error(err))
	}

	err = userInfoRepo.Delete(ctx, db, userID)
	if err != nil {
		return err
	}

	if err := userInfoCache.UpdateUserInfo(ctx, userID, nil); err != nil {
		logger.Warn("update user_info cache fail", zap.Error(err))
	}
	return nil
}
