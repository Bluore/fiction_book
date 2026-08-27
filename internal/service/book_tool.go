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

	// 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookCache.LockBook(ctx, bookID)
	if lockErr != nil {
		logger.Warn("lock book cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
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

		return nil, nil
	}

	if err := bookCache.UpdateBook(ctx, bookID, book); err != nil {
		logger.Warn("update book cache fail", zap.Error(err))
	}
	if err := bookCache.UnlockBook(ctx, bookID, lockToken); err != nil {
		logger.Error("unlock book cache fail", zap.Error(err))
	}
	return book, nil
}

func createBookDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookCache cache.BookCache,
	bookRepo repository.BookRepository,
	book *model.Book,
) error {
	if book == nil {
		return fmt.Errorf("book is nil")
	}

	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	bookExist, err := queryBookDetailWithCache(ctx, db, bookCache, bookRepo, book.ID)
	if err != nil {
		return err
	}
	if bookExist != nil {
		return nil
	}

	// FOCUS 抢锁
	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = bookCache.LockBook(ctx, book.ID)
		if err != nil {
			logger.Warn("lock book cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("book cache lock too many fail")
	}
	defer func() {
		if err := bookCache.UnlockBook(ctx, book.ID, token); err != nil {
			logger.Error("unlock book cache fail", zap.Error(err))
		}
	}()

	// FOCUS 删除记录
	err = bookCache.DeleteBook(ctx, book.ID)
	if err != nil {
		logger.Warn("delete book cache fail", zap.Error(err))
	}

	// FOCUS 锁内二次查询，并发下另一请求可能已创建，保持幂等
	bookExist, err = bookRepo.GetByID(ctx, book.ID)
	if err != nil {
		return err
	}
	if bookExist != nil {
		if err := bookCache.UpdateBook(ctx, bookExist.ID, bookExist); err != nil {
			logger.Warn("update book cache fail", zap.Error(err))
		}
		return nil
	}

	// FOCUS 更新数据
	err = bookRepo.Create(ctx, db, book)
	if err != nil {
		return err
	}

	if err := bookCache.UpdateBook(ctx, book.ID, book); err != nil {
		logger.Warn("update book cache fail", zap.Error(err))
	}
	return nil
}

func updateBookDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookCache cache.BookCache,
	bookRepo repository.BookRepository,
	book *model.Book,
) error {
	if book == nil {
		return fmt.Errorf("book is nil")
	}

	var token string
	var locked bool
	var err error

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = bookCache.LockBook(ctx, book.ID)
		if err != nil {
			logger.Warn("lock book cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("book cache lock too many fail")
	}
	defer func() {
		if err := bookCache.UnlockBook(ctx, book.ID, token); err != nil {
			logger.Error("unlock book cache fail", zap.Error(err))
		}
	}()

	err = bookCache.DeleteBook(ctx, book.ID)
	if err != nil {
		logger.Warn("delete book cache fail", zap.Error(err))
	}

	err = bookRepo.Update(ctx, db, book)
	if err != nil {
		return err
	}

	if err := bookCache.UpdateBook(ctx, book.ID, book); err != nil {
		logger.Warn("update book cache fail", zap.Error(err))
	}
	return nil
}

func deleteBookDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookCache cache.BookCache,
	bookRepo repository.BookRepository,
	bookID string,
) error {
	var token string
	var locked bool
	var err error

	// FOCUS 查询是否已存在，保持幂等
	bookExist, err := queryBookDetailWithCache(ctx, db, bookCache, bookRepo, bookID)
	if err != nil {
		return err
	}
	if bookExist == nil {
		return nil
	}

	for i := 0; i < lockTryTimes; i++ {
		token, locked, err = bookCache.LockBook(ctx, bookID)
		if err != nil {
			logger.Warn("lock book cache fail", zap.Error(err))
		}
		if locked {
			break
		}
		time.Sleep(lockWaitTime)
	}
	if !locked {
		return fmt.Errorf("book cache lock too many fail")
	}
	defer func() {
		if err := bookCache.UnlockBook(ctx, bookID, token); err != nil {
			logger.Error("unlock book cache fail", zap.Error(err))
		}
	}()

	err = bookCache.DeleteBook(ctx, bookID)
	if err != nil {
		logger.Warn("delete book cache fail", zap.Error(err))
	}

	err = bookRepo.Delete(ctx, db, bookID)
	if err != nil {
		return err
	}

	if err := bookCache.UpdateBook(ctx, bookID, nil); err != nil {
		logger.Warn("update book cache fail", zap.Error(err))
	}
	return nil
}
