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
