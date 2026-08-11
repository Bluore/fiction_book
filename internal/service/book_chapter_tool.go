package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

func queryBookChapterDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookChapterCache cache.BookChapterCache,
	bookChapterRepo repository.BookChapterRepository,
	bookChapterID string,
) (*model.BookChapter, error) {
	// 缓存查询
	var bookChapter *model.BookChapter
	var err error
	var bookChapterCacheHit bool
	bookChapter, bookChapterCacheHit, err = bookChapterCache.GetBookChapter(ctx, bookChapterID)
	if err != nil {
		logger.Warn("get book chapter cache fail", zap.Error(err))
	}
	if bookChapterCacheHit {
		return bookChapter, nil
	}
	if bookChapter == nil {
		return nil, ErrBookChapterNotFound
	}

	// FOCUS 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookChapterCache.LockBookChapter(ctx, bookChapterID)
	if lockErr != nil {
		// 降级查询
		logger.Warn("lock book chapter cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < 8; i++ {
			time.Sleep(100 * time.Millisecond)
			bookChapter, bookChapterCacheHit, _ = bookChapterCache.GetBookChapter(ctx, bookChapterID)
			if bookChapterCacheHit {
				return bookChapter, nil
			}
		}

		return nil, fmt.Errorf("book chapter cache query too many fail")
	}

	bookChapter, err = bookChapterRepo.GetByID(ctx, bookChapterID)
	if err != nil {
		if err := bookChapterCache.UnlockBookChapter(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book chapter cache fail", zap.Error(err))
		}
		return nil, err
	}
	if bookChapter == nil {
		// FOCUS 防止内存穿透：存储不存在的值
		updateChapterErr := bookChapterCache.UpdateBookChapter(ctx, bookChapterID, nil)
		if updateChapterErr != nil {
			logger.Warn("update book chapter cache fail", zap.Error(updateChapterErr))
		}

		if err := bookChapterCache.UnlockBookChapter(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book chapter cache fail", zap.Error(err))
		}

		return nil, ErrBookChapterNotFound
	}
	if err := bookChapterCache.UnlockBookChapter(ctx, bookChapterID, lockToken); err != nil {
		logger.Error("unlock book chapter cache fail", zap.Error(err))
	}
	return bookChapter, nil
}

func toBookChapterInfoResponse(cfg *config.Config, info *model.BookChapterInfo, book *model.Book) *dto.BookChapterInfoResponse {
	return &dto.BookChapterInfoResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Order:     info.Order,
		VIPLevel:  info.VIPLevel,
		Price:     info.Price,
		Book:      toBookResponse(cfg, book),
	}
}

func toBookChapterResponse(cfg *config.Config, info *model.BookChapter, book *model.Book) *dto.BookChapterResponse {
	return &dto.BookChapterResponse{
		ChapterID: info.ID,
		Title:     info.Title,
		Content:   info.Content,
		Order:     info.Order,
		VIPLevel:  info.VIPLevel,
		Price:     info.Price,
		Book:      toBookResponse(cfg, book),
	}
}
