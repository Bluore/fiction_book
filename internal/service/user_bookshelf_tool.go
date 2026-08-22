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

func queryUserBookshelfDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userBookshelfCache cache.UserBookshelfCache,
	userBookshelfRepo repository.UserBookshelfRepository,
	userID string,
	bookID string,
) (*model.UserBookshelf, error) {
	userBookshelf, cacheHit, err := userBookshelfCache.GetUserBookshelf(ctx, userID, bookID)
	if err != nil {
		logger.Warn("get user_bookshelf cache fail", zap.Error(err))
	}
	if cacheHit {
		return userBookshelf, nil
	}

	token, locked, err := userBookshelfCache.LockUserBookshelf(ctx, userID, bookID)
	if err != nil {
		logger.Warn("lock user_bookshelf cache fail", zap.Error(err))
		return userBookshelfRepo.GetByUserIDAndBookID(ctx, db, userID, bookID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			userBookshelf, cacheHit, _ = userBookshelfCache.GetUserBookshelf(ctx, userID, bookID)
			if cacheHit {
				return userBookshelf, nil
			}
		}
		return nil, fmt.Errorf("user_bookshelf cache query too many fail")
	}
	defer func() {
		if err := userBookshelfCache.UnlockUserBookshelf(ctx, userID, bookID, token); err != nil {
			logger.Error("unlock user_bookshelf cache fail", zap.Error(err))
		}
	}()

	userBookshelf, err = userBookshelfRepo.GetByUserIDAndBookID(ctx, db, userID, bookID)
	if err != nil {
		return nil, err
	}
	if err := userBookshelfCache.UpdateUserBookshelf(ctx, userID, bookID, userBookshelf); err != nil {
		logger.Warn("update user_bookshelf cache fail", zap.Error(err))
	}
	return userBookshelf, nil
}

func createUserBookshelfDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userBookshelfCache cache.UserBookshelfCache,
	userBookshelfRepo repository.UserBookshelfRepository,
	userID string,
	bookID string,
) error {
	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userBookshelfExist, err := queryUserBookshelfDetailWithCache(ctx, db, userBookshelfCache, userBookshelfRepo, userID, bookID)
	if err != nil {
		return err
	}
	if userBookshelfExist != nil {
		return nil
	}

	// FOCUS 删除记录
	_, err = userBookshelfCache.DeleteUserBookshelf(ctx, userID, bookID)
	if err != nil {
		logger.Warn("delete user_bookshelf cache fail", zap.Error(err))
	}

	// FOCUS 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userBookshelfCache.LockUserBookshelf(ctx, userID, bookID)
		if err != nil {
			logger.Warn("lock user_bookshelf cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_bookshelf cache lock too many fail")
	}
	defer func() {
		if err := userBookshelfCache.UnlockUserBookshelf(ctx, userID, bookID, token); err != nil {
			logger.Error("unlock user_bookshelf cache fail", zap.Error(err))
		}
	}()

	// FOCUS 锁内二次查询，并发下另一请求可能已创建，保持幂等
	userBookshelfExist, err = userBookshelfRepo.GetByUserIDAndBookID(ctx, db, userID, bookID)
	if err != nil {
		return err
	}
	if userBookshelfExist != nil {
		if err := userBookshelfCache.UpdateUserBookshelf(ctx, userID, bookID, userBookshelfExist); err != nil {
			logger.Warn("update user_bookshelf cache fail", zap.Error(err))
		}
		return nil
	}

	// FOCUS 更新数据
	err = userBookshelfRepo.CreateByUserIDAndBookID(ctx, db, userID, bookID)
	if err != nil {
		return err
	}

	userBookshelf, err := userBookshelfRepo.GetByUserIDAndBookID(ctx, db, userID, bookID)
	if err != nil {
		return err
	}
	if err := userBookshelfCache.UpdateUserBookshelf(ctx, userID, bookID, userBookshelf); err != nil {
		logger.Warn("update user_bookshelf cache fail", zap.Error(err))
	}
	return nil
}

func deleteUserBookshelfDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	userBookshelfCache cache.UserBookshelfCache,
	userBookshelfRepo repository.UserBookshelfRepository,
	userID string,
	bookID string,
) error {
	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	userBookshelfExist, err := queryUserBookshelfDetailWithCache(ctx, db, userBookshelfCache, userBookshelfRepo, userID, bookID)
	if err != nil {
		return err
	}
	if userBookshelfExist == nil {
		return nil
	}

	_, err = userBookshelfCache.DeleteUserBookshelf(ctx, userID, bookID)
	if err != nil {
		logger.Warn("delete user_bookshelf cache fail", zap.Error(err))
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = userBookshelfCache.LockUserBookshelf(ctx, userID, bookID)
		if err != nil {
			logger.Warn("lock user_bookshelf cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("user_bookshelf cache lock too many fail")
	}
	defer func() {
		if err := userBookshelfCache.UnlockUserBookshelf(ctx, userID, bookID, token); err != nil {
			logger.Error("unlock user_bookshelf cache fail", zap.Error(err))
		}
	}()

	err = userBookshelfRepo.DeleteByUserIDAndBookID(ctx, db, userID, bookID)
	if err != nil {
		return err
	}

	if err := userBookshelfCache.UpdateUserBookshelf(ctx, userID, bookID, nil); err != nil {
		logger.Warn("update user_bookshelf cache fail", zap.Error(err))
	}
	return nil
}
