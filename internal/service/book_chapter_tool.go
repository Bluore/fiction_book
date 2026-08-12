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
	// 获取章节信息
	bookChapterInfo, err := queryBookChapterInfoWithCache(ctx, db, bookChapterCache, bookChapterRepo, bookChapterID)
	if err != nil {
		return nil, err
	}

	// 获取章节内容
	bookChapterContent, err := queryBookChapterContentWithCache(ctx, db, bookChapterCache, bookChapterRepo, bookChapterID)
	if err != nil {
		return nil, err
	}

	// 组合成完整的 BookChapter
	bookChapter := &model.BookChapter{
		ID:       bookChapterInfo.ID,
		BookID:   bookChapterInfo.BookID,
		Title:    bookChapterInfo.Title,
		Content:  bookChapterContent.Content,
		Order:    bookChapterInfo.Order,
		VIPLevel: bookChapterInfo.VIPLevel,
		Price:    bookChapterInfo.Price,
	}

	return bookChapter, nil
}

func queryBookChapterInfoWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookChapterCache cache.BookChapterCache,
	bookChapterRepo repository.BookChapterRepository,
	bookChapterID string,
) (*model.BookChapterInfo, error) {
	// 缓存查询
	var bookChapterInfo *model.BookChapterInfo
	var err error
	var bookChapterCacheHit bool
	bookChapterInfo, bookChapterCacheHit, err = bookChapterCache.GetBookChapterInfo(ctx, bookChapterID)
	if err != nil {
		logger.Warn("get book_chapter_info cache fail", zap.Error(err))
	}
	if bookChapterCacheHit {
		return bookChapterInfo, nil
	}
	if bookChapterInfo == nil {
		return nil, ErrBookChapterNotFound
	}

	// FOCUS 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookChapterCache.LockBookChapterInfo(ctx, bookChapterID)
	if lockErr != nil {
		// 降级查询
		logger.Warn("lock book_chapter_info cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < 8; i++ {
			time.Sleep(100 * time.Millisecond)
			bookChapterInfo, bookChapterCacheHit, _ = bookChapterCache.GetBookChapterInfo(ctx, bookChapterID)
			if bookChapterCacheHit {
				return bookChapterInfo, nil
			}
		}

		return nil, fmt.Errorf("book_chapter_info cache query too many fail")
	}

	bookChapterInfo, err = bookChapterRepo.BookChapterInfoGetByID(ctx, db, bookChapterID)
	if err != nil {
		if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
		}
		return nil, err
	}
	if bookChapterInfo == nil {
		// FOCUS 防止内存穿透：存储不存在的值
		updateChapterErr := bookChapterCache.UpdateBookChapterInfo(ctx, bookChapterID, nil)
		if updateChapterErr != nil {
			logger.Warn("update book_chapter_info cache fail", zap.Error(updateChapterErr))
		}

		if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
		}

		return nil, ErrBookChapterNotFound
	}
	if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterID, lockToken); err != nil {
		logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
	}
	return bookChapterInfo, nil
}

func queryBookChapterContentWithCache(
	ctx context.Context,
	db *gorm.DB,
	bookChapterCache cache.BookChapterCache,
	bookChapterRepo repository.BookChapterRepository,
	bookChapterID string,
) (*model.BookChapterContent, error) {
	// 缓存查询
	var bookChapterContent *model.BookChapterContent
	var err error
	var bookChapterCacheHit bool
	bookChapterContent, bookChapterCacheHit, err = bookChapterCache.GetBookChapterContent(ctx, bookChapterID)
	if err != nil {
		logger.Warn("get book_chapter_content cache fail", zap.Error(err))
	}
	if bookChapterCacheHit {
		return bookChapterContent, nil
	}
	if bookChapterContent == nil {
		return nil, ErrBookChapterNotFound
	}

	// FOCUS 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookChapterCache.LockBookChapterContent(ctx, bookChapterID)
	if lockErr != nil {
		// 降级查询
		logger.Warn("lock book_chapter_content cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < 8; i++ {
			time.Sleep(100 * time.Millisecond)
			bookChapterContent, bookChapterCacheHit, _ = bookChapterCache.GetBookChapterContent(ctx, bookChapterID)
			if bookChapterCacheHit {
				return bookChapterContent, nil
			}
		}

		return nil, fmt.Errorf("book_chapter_content cache query too many fail")
	}

	bookChapterContent, err = bookChapterRepo.BookChapterContentGetByID(ctx, db, bookChapterID)
	if err != nil {
		if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
		}
		return nil, err
	}
	if bookChapterContent == nil {
		// FOCUS 防止内存穿透：存储不存在的值
		updateChapterErr := bookChapterCache.UpdateBookChapterContent(ctx, bookChapterID, nil)
		if updateChapterErr != nil {
			logger.Warn("update book_chapter_content cache fail", zap.Error(updateChapterErr))
		}

		if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterID, lockToken); err != nil {
			logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
		}

		return nil, ErrBookChapterNotFound
	}
	if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterID, lockToken); err != nil {
		logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
	}
	return bookChapterContent, nil
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
