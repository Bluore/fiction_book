package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func queryBookDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookCache cache.BookCache,
	bookRepo repository.BookRepository,
	bookID string,
) (*model.Book, error) {
	var book *model.Book
	var err error
	// 缓存查询
	var bookCacheHit bool
	book, bookCacheHit, err = bookCache.GetBook(ctx, bookID)
	if err != nil {
		logger.Warn("get book cache fail", zap.Error(err))
	}
	if bookCacheHit {
		return book, nil
	}
	if book == nil {
		return nil, ErrBookNotFound
	}

	// 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookCache.LockBook(ctx, bookID)
	if lockErr != nil {
		logger.Warn("lock book cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < 8; i++ {
			time.Sleep(100 * time.Millisecond)
			book, bookCacheHit, _ = bookCache.GetBook(ctx, bookID)
			if bookCacheHit {
				return book, nil
			}
		}

		return nil, fmt.Errorf("book chapter cache query too many fail")
	}

	// 降级查询
	book, err = bookRepo.GetByID(ctx, bookID)
	if err != nil {
		if err := bookCache.UnlockBook(ctx, bookID, lockToken); err != nil {
			logger.Error("unlock book cache fail", zap.Error(err))
		}
		return nil, err
	}
	if book == nil {
		if err := bookCache.UpdateBook(ctx, bookID, nil); err != nil {
			logger.Warn("update book cache fail", zap.Error(err))
		}

		if err := bookCache.UnlockBook(ctx, bookID, lockToken); err != nil {
			logger.Error("unlock book cache fail", zap.Error(err))
		}

		return nil, ErrBookNotFound
	}
	if err := bookCache.UnlockBook(ctx, bookID, lockToken); err != nil {
		logger.Error("unlock book cache fail", zap.Error(err))
	}
	return book, nil
}
