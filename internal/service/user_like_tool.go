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

// Deprecated: createUserLikeDetailWithCache 旧版创建点赞带缓存，将被移除
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

	// FOCUS 删除记录
	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

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

// Deprecated: updateUserLikeDetailWithCache 旧版更新点赞带缓存，将被移除
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

	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

	err = userLikeRepo.Update(ctx, db, userLike)
	if err != nil {
		return err
	}

	if err := userLikeCache.UpdateUserLike(ctx, userLike.UserID, userLike.LikerID, userLike); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return nil
}

// Deprecated: deleteUserLikeDetailWithCache 旧版删除点赞带缓存，将被移除
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

	_, err = userLikeCache.DeleteUserLike(ctx, userLike.UserID, userLike.LikerID)
	if err != nil {
		logger.Warn("delete user_like cache fail", zap.Error(err))
	}

	err = userLikeRepo.DeleteByUserIDAndLikerID(ctx, db, userLike.UserID, userLike.LikerID)
	if err != nil {
		return err
	}

	if err := userLikeCache.UpdateUserLike(ctx, userLike.UserID, userLike.LikerID, nil); err != nil {
		logger.Warn("update user_like cache fail", zap.Error(err))
	}
	return nil
}

// queryUserLikeIDListWithCache 获取用户关注的大V/小V用户ID列表
func queryUserLikeIDListWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	likerID string,
) (bigVUserIDList []string, smallVUserIDList []string, err error) {
	bigVUserIDList, err = queryUserLikeBigVUserIDListWithCache(ctx, db, userLikeCache, userLikeRepo, likerID)
	if err != nil {
		return make([]string, 0), make([]string, 0), err
	}
	smallVUserIDList, err = queryUserLikeSmallVUserIDListWithCache(ctx, db, userLikeCache, userLikeRepo, likerID)
	if err != nil {
		return make([]string, 0), make([]string, 0), err
	}
	return bigVUserIDList, smallVUserIDList, nil
}

func queryUserLikeBigVUserIDListWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	likerID string,
) ([]string, error) {
	userIDList, cacheHit, err := userLikeCache.GetUserLikeBigVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("get user_like_big_v_user_id_list cache fail", zap.Error(err))
	}
	if cacheHit {
		return userIDList, nil
	}

	token, locked, err := userLikeCache.LockUserLikeBigVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("lock user_like_big_v_user_id_list cache fail", zap.Error(err))
		return userLikeRepo.GetUserLikeBigVListByLikerID(ctx, db, likerID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userIDList, cacheHit, _ = userLikeCache.GetUserLikeBigVUserIDList(ctx, likerID)
			if cacheHit {
				return userIDList, nil
			}
		}
		return make([]string, 0), fmt.Errorf("user_like_big_v_user_id_list cache query too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLikeBigVUserIDList(ctx, likerID, token); err != nil {
			logger.Error("unlock user_like_big_v_user_id_list cache fail", zap.Error(err))
		}
	}()

	userIDList, err = userLikeRepo.GetUserLikeBigVListByLikerID(ctx, db, likerID)
	if err != nil {
		return make([]string, 0), err
	}
	if err := userLikeCache.UpdateUserLikeBigVUserIDList(ctx, likerID, userIDList); err != nil {
		logger.Warn("update user_like_big_v_user_id_list cache fail", zap.Error(err))
	}
	return userIDList, nil
}

func queryUserLikeSmallVUserIDListWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	likerID string,
) ([]string, error) {
	userIDList, cacheHit, err := userLikeCache.GetUserLikeSmallVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("get user_like_small_v_user_id_list cache fail", zap.Error(err))
	}
	if cacheHit {
		return userIDList, nil
	}

	token, locked, err := userLikeCache.LockUserLikeSmallVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("lock user_like_small_v_user_id_list cache fail", zap.Error(err))
		return userLikeRepo.GetUserLikeSmallVListByLikerID(ctx, db, likerID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userIDList, cacheHit, _ = userLikeCache.GetUserLikeSmallVUserIDList(ctx, likerID)
			if cacheHit {
				return userIDList, nil
			}
		}
		return make([]string, 0), fmt.Errorf("user_like_small_v_user_id_list cache query too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLikeSmallVUserIDList(ctx, likerID, token); err != nil {
			logger.Error("unlock user_like_small_v_user_id_list cache fail", zap.Error(err))
		}
	}()

	userIDList, err = userLikeRepo.GetUserLikeSmallVListByLikerID(ctx, db, likerID)
	if err != nil {
		return make([]string, 0), err
	}
	if err := userLikeCache.UpdateUserLikeSmallVUserIDList(ctx, likerID, userIDList); err != nil {
		logger.Warn("update user_like_small_v_user_id_list cache fail", zap.Error(err))
	}
	return userIDList, nil
}

// Deprecated: deleteUserLikeIDListWithCache 旧版删除点赞ID列表带缓存，将被移除
func deleteUserLikeIDListWithCache(
	ctx context.Context,
	db *gorm.DB,
	userLikeCache cache.UserLikeCache,
	userLikeRepo repository.UserLikeRepository,
	likerID string,
) error {
	var bigVToken string
	var smallVToken string
	var bigVLocked bool
	var smallVLocked bool
	var err error

	// 查询当前关注的用户列表，用于清理详情缓存，空列表直接返回，保持幂等
	userIDList, err := userLikeRepo.GetUserLikeListByLikerID(ctx, db, likerID)
	if err != nil {
		return err
	}
	if len(userIDList) == 0 {
		return nil
	}

	// 抢锁
	for i := 0; i < lockTryTimes; i++ {
		bigVToken, bigVLocked, err = userLikeCache.LockUserLikeBigVUserIDList(ctx, likerID)
		if err != nil {
			logger.Warn("lock user_like_big_v_user_id_list cache fail", zap.Error(err))
		}
		if bigVLocked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !bigVLocked {
		return fmt.Errorf("user_like_big_v_user_id_list cache lock too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLikeBigVUserIDList(ctx, likerID, bigVToken); err != nil {
			logger.Error("unlock user_like_big_v_user_id_list cache fail", zap.Error(err))
		}
	}()

	for i := 0; i < lockTryTimes; i++ {
		smallVToken, smallVLocked, err = userLikeCache.LockUserLikeSmallVUserIDList(ctx, likerID)
		if err != nil {
			logger.Warn("lock user_like_small_v_user_id_list cache fail", zap.Error(err))
		}
		if smallVLocked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !smallVLocked {
		return fmt.Errorf("user_like_small_v_user_id_list cache lock too many fail")
	}
	defer func() {
		if err := userLikeCache.UnlockUserLikeSmallVUserIDList(ctx, likerID, smallVToken); err != nil {
			logger.Error("unlock user_like_small_v_user_id_list cache fail", zap.Error(err))
		}
	}()

	// 删除列表缓存，避免读到旧数据
	err = userLikeCache.DeleteUserLikeBigVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("delete user_like_big_v_user_id_list cache fail", zap.Error(err))
	}
	err = userLikeCache.DeleteUserLikeSmallVUserIDList(ctx, likerID)
	if err != nil {
		logger.Warn("delete user_like_small_v_user_id_list cache fail", zap.Error(err))
	}

	// 删除记录
	err = userLikeRepo.DeleteAllUserByLikerID(ctx, db, likerID)
	if err != nil {
		return err
	}

	// 清理该 liker 对应的详情缓存，避免读到已删除的关注状态
	for _, userID := range userIDList {
		if _, err := userLikeCache.DeleteUserLike(ctx, userID, likerID); err != nil {
			logger.Warn("delete user_like cache fail", zap.Error(err))
		}
	}

	return nil
}
