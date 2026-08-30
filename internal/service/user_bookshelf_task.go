package service

import (
	"context"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/repository"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/response"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func bookThermalCacheAddByBookshelfTask(ctx context.Context, s *userBookshelfService, userID string, req *dto.CreateUserBookshelfRequest) error {
	exist, _ := s.limitCatch.CheckUserBookshelfBook1Hour(ctx, userID, req.BookID)
	if exist {
		return nil
	}

	// 热力值限流
	_ = s.limitCatch.AddUserBookshelfBook1Hour(ctx, userID, req.BookID)

	// 更新数据
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddValueTask(constant.DoneServerCtx, s.db, s.bookThermalCache, s.bookCache, s.bookRepo, s.pool, req.BookID, 10)
	}))

	return nil
}

// 1. 添加热力值到Redis和Mysql（自动侧载读取和持久化）
// 2. 添加热力值到排行榜
func bookThermalCacheAddValueTask(ctx context.Context, db *gorm.DB, bookThermalCache cache.BookThermalCache, bookCache cache.BookCache, d repository.BookRepository, p *agilePool.Pool, bookID string, value cache.BookThermalValue) error {
	err := bookThermalCache.AddBookThermalValue(ctx, bookID, value)
	updateBookThermalValueFunc := func() error {
		bookDetail, err := d.GetByID(ctx, bookID)
		if err != nil {
			return err
		}
		if bookDetail == nil {
			return response.ErrBookNotFound
		}
		bookThermalValue := cache.BookThermalValue(bookDetail.ThermalValue)
		bookThermalValue += value

		err = bookThermalCache.UpdateBookThermalValue(ctx, bookID, bookThermalValue)
		if err != nil {
			logger.Error("UpdateBookThermalValue", zap.Error(err))
			return err
		}
		return nil
	}
	if err != nil {
		if errors.Is(err, cache.ErrorBookThermalValueNotExist) {
			err := updateBookThermalValueFunc()
			if err != nil {
				return err
			}

			goto breakBookCacheError
		}
		return err
	}
breakBookCacheError:

	if err := bookThermalCache.LockBookThermalValue(ctx, bookID); err != nil {
		if errors.Is(err, cache.ErrorBookThermalValueNotExist) {
			return nil
		}
		return err
	}

	// 异步：数据库同步
	p.SubmitCtx(ctx, agilePool.TaskFunc(func() error {
		return bookThermalCacheSyncToDatabaseTask(ctx, db, bookThermalCache, bookCache, d, bookID)
	}))

	// 更新榜单
	err = bookCache.IncreaseBookThermalDaily(ctx, bookID, int(value))
	if err != nil {
		logger.Error("IncreaseBookThermalDaily", zap.Error(err))
	}

	return nil
}

func bookThermalCacheSyncToDatabaseTask(ctx context.Context, db *gorm.DB, bookThermalCache cache.BookThermalCache, bookCache cache.BookCache, d repository.BookRepository, bookID string) error {
	bookThermalValue, ok, _ := bookThermalCache.GetBookThermalValue(ctx, bookID)
	if !ok {
		return nil
	}

	err := d.UpdateBookThermal(ctx, db, bookID, bookThermalValue)
	if err != nil {
		return err
	}

	bookDetail, err := d.GetByID(ctx, bookID)
	if err != nil {
		return err
	}
	err = bookCache.UpdateBook(ctx, bookDetail.ID, bookDetail)
	if err != nil {
		logger.Error("UpdateBookCache", zap.Error(err))
	}

	return nil
}
