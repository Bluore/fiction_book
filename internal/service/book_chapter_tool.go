package service

import (
	"context"
	"encoding/json"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	lockTryTimes = 8
	lockWaitTime = 100 * time.Millisecond
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
	if bookChapterInfo == nil {
		return nil, nil
	}

	// 获取章节内容
	bookChapterContent, err := queryBookChapterContentWithCache(ctx, db, bookChapterCache, bookChapterRepo, bookChapterID)
	if err != nil {
		return nil, err
	}
	if bookChapterContent == nil {
		return nil, nil
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

	// FOCUS 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookChapterCache.LockBookChapterInfo(ctx, bookChapterID)
	if lockErr != nil {
		// 降级查询
		logger.Warn("lock book_chapter_info cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(100 * time.Millisecond)
			bookChapterInfo, bookChapterCacheHit, _ = bookChapterCache.GetBookChapterInfo(ctx, bookChapterID)
			if bookChapterCacheHit {
				return bookChapterInfo, nil
			}
		}

		logger.Error("book_chapter_info cache query too many fail")
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

		return nil, nil
	}
	if err := bookChapterCache.UpdateBookChapterInfo(ctx, bookChapterID, bookChapterInfo); err != nil {
		logger.Warn("update book_chapter_info cache fail", zap.Error(err))
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

	// FOCUS 防止内存击穿：持锁查询数据库，未持锁轮询结果
	lockToken, lockOk, lockErr := bookChapterCache.LockBookChapterContent(ctx, bookChapterID)
	if lockErr != nil {
		// 降级查询
		logger.Warn("lock book_chapter_content cache fail", zap.Error(lockErr))
	}
	if !lockOk {
		// 轮询结果
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(100 * time.Millisecond)
			bookChapterContent, bookChapterCacheHit, _ = bookChapterCache.GetBookChapterContent(ctx, bookChapterID)
			if bookChapterCacheHit {
				return bookChapterContent, nil
			}
		}

		logger.Error("book_chapter_content cache query too many fail")
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

		return nil, nil
	}
	if err := bookChapterCache.UpdateBookChapterContent(ctx, bookChapterID, bookChapterContent); err != nil {
		logger.Warn("update book_chapter_content cache fail", zap.Error(err))
	}
	if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterID, lockToken); err != nil {
		logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
	}
	return bookChapterContent, nil
}

func updateBookChapterInfoWithCacheInTx(
	ctx context.Context,
	db *gorm.DB,
	bookChapterCache cache.BookChapterCache,
	bookChapterRepo repository.BookChapterRepository,
	outboxRepo repository.OutboxRepository,
	originBookChapter *model.BookChapter,
	bookChapterInfo *model.BookChapterInfo,
) error {
	if originBookChapter == nil {
		return fmt.Errorf("bookChapter is nil")
	}
	if bookChapterInfo == nil {
		return fmt.Errorf("bookChapterInfo is nil")
	}

	mergeBookChapter := &model.BookChapter{
		ID:        bookChapterInfo.ID,
		BookID:    bookChapterInfo.BookID,
		Title:     bookChapterInfo.Title,
		Content:   originBookChapter.Content,
		Order:     bookChapterInfo.Order,
		VIPLevel:  bookChapterInfo.VIPLevel,
		Price:     bookChapterInfo.Price,
		CreatedAt: bookChapterInfo.CreatedAt,
		UpdatedAt: bookChapterInfo.UpdatedAt,
		DeletedAt: bookChapterInfo.DeletedAt,
	}

	// FOCUS 更新数据也需要添加分布式锁
	var successLock bool = false
	var token string
	var err error
	for i := 0; i < lockTryTimes; i++ {
		token, successLock, err = bookChapterCache.LockBookChapterInfo(ctx, bookChapterInfo.ID)
		if err != nil {
			logger.Warn("lock book_chapter_info cache fail", zap.Error(err))
			time.Sleep(lockTryTimes)
		}
		if successLock {
			goto successLock
		}
		time.Sleep(lockWaitTime)
	}
	return fmt.Errorf("book_chapter_info cache lock too many fail")
successLock:

	mergeBookChapterJSON, err := json.Marshal(mergeBookChapter)
	if err != nil {
		if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterInfo.ID, token); err != nil {
			logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
		}
		return fmt.Errorf("merge book_chapter_info fail")
	}
	outboxDetail := &model.Outbox{
		ID:        uuid.New().String(),
		Type:      mergeBookChapter.TableName(),
		Operation: model.DatabaseOperationUpdate,
		Payload:   mergeBookChapterJSON,
	}

	// FOCUS 通过事务更新数据并添加到kafka
	tx := db.Begin()
	err = bookChapterRepo.BookChapterInfoUpdate(ctx, tx, bookChapterInfo.ID, bookChapterInfo)
	if err != nil {
		tx.Rollback()
		if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterInfo.ID, token); err != nil {
			logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
		}
		return err
	}

	err = outboxRepo.AddEvent(ctx, tx, outboxDetail)
	if err != nil {
		tx.Rollback()
		if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterInfo.ID, token); err != nil {
			logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
		}
		return err
	}

	tx.Commit()

	if err := bookChapterCache.UpdateBookChapterInfo(ctx, bookChapterInfo.ID, bookChapterInfo); err != nil {
		logger.Warn("error update book_chapter_info", zap.Error(err))
	}

	if err := bookChapterCache.UnlockBookChapterInfo(ctx, bookChapterInfo.ID, token); err != nil {
		logger.Error("unlock book_chapter_info cache fail", zap.Error(err))
	}
	return nil
}

func updateBookChapterContentWithCacheInTx(
	ctx context.Context,
	db *gorm.DB,
	bookChapterCache cache.BookChapterCache,
	bookChapterRepo repository.BookChapterRepository,
	outboxRepo repository.OutboxRepository,
	originBookChapter *model.BookChapter,
	bookChapterContent *model.BookChapterContent,
) error {
	if originBookChapter == nil {
		return fmt.Errorf("bookChapter is nil")
	}
	if bookChapterContent == nil {
		return fmt.Errorf("bookChapterContent is nil")
	}

	mergeBookChapter := &model.BookChapter{
		ID:        bookChapterContent.ID,
		BookID:    originBookChapter.BookID,
		Title:     originBookChapter.Title,
		Content:   bookChapterContent.Content,
		Order:     originBookChapter.Order,
		VIPLevel:  originBookChapter.VIPLevel,
		Price:     originBookChapter.Price,
		CreatedAt: originBookChapter.CreatedAt,
		UpdatedAt: originBookChapter.UpdatedAt,
		DeletedAt: originBookChapter.DeletedAt,
	}

	// FOCUS 更新数据也需要添加分布式锁
	var successLock bool = false
	var token string
	var err error
	for i := 0; i < lockTryTimes; i++ {
		token, successLock, err = bookChapterCache.LockBookChapterContent(ctx, bookChapterContent.ID)
		if err != nil {
			logger.Warn("lock book_chapter_content cache fail", zap.Error(err))
			time.Sleep(lockWaitTime)
		}
		if successLock {
			goto successLock
		}
		time.Sleep(lockWaitTime)
	}
	return fmt.Errorf("book_chapter_content cache lock too many fail")
successLock:

	mergeBookChapterJSON, err := json.Marshal(mergeBookChapter)
	if err != nil {
		if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterContent.ID, token); err != nil {
			logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
		}
		return fmt.Errorf("merge book_chapter_content fail")
	}
	outboxDetail := &model.Outbox{
		ID:        uuid.New().String(),
		Type:      mergeBookChapter.TableName(),
		Operation: model.DatabaseOperationUpdate,
		Payload:   mergeBookChapterJSON,
	}

	// FOCUS 通过事务更新数据并添加到kafka
	tx := db.Begin()
	err = bookChapterRepo.BookChapterContentUpdate(ctx, tx, bookChapterContent.ID, bookChapterContent)
	if err != nil {
		tx.Rollback()
		if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterContent.ID, token); err != nil {
			logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
		}
		return err
	}

	err = outboxRepo.AddEvent(ctx, tx, outboxDetail)
	if err != nil {
		tx.Rollback()
		if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterContent.ID, token); err != nil {
			logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
		}
		return err
	}

	tx.Commit()

	if err := bookChapterCache.UpdateBookChapterContent(ctx, bookChapterContent.ID, bookChapterContent); err != nil {
		logger.Warn("error update book_chapter_content", zap.Error(err))
	}

	if err := bookChapterCache.UnlockBookChapterContent(ctx, bookChapterContent.ID, token); err != nil {
		logger.Error("unlock book_chapter_content cache fail", zap.Error(err))
	}
	return nil
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

// Deprecated: buildNewChapterFeeds 旧版构建新章节 feed，将被移除
func buildNewChapterFeeds(
	book *model.Book,
	chapter *model.BookChapter,
	userIDs []string,
) []*model.FeedInbox {
	now := time.Now()
	chapterID := chapter.ID
	chapterTitle := chapter.Title
	feeds := make([]*model.FeedInbox, 0, len(userIDs))

	for _, userID := range userIDs {
		feeds = append(feeds, &model.FeedInbox{
			ID:           uuid.New().String(),
			UserID:       userID,
			EventType:    model.FeedEventNewChapter,
			BookID:       book.ID,
			ChapterID:    &chapterID,
			BookName:     book.Name,
			BookCover:    book.CoverImage,
			ChapterTitle: &chapterTitle,
			Content:      fmt.Sprintf("《%s》更新了章节：%s", book.Name, chapter.Title),
			IsRead:       false,
			CreatedAt:    now,
			UpdatedAt:    now,
		})
	}

	return feeds
}

// Deprecated: warmFeedCache 旧版预热 feed 缓存，将被移除
func warmFeedCache(ctx context.Context, feedCache cache.FeedCache, feeds []*model.FeedInbox) (err error) {
	userFeeds := make(map[string][]*model.FeedInbox)
	for _, feed := range feeds {
		userFeeds[feed.UserID] = append(userFeeds[feed.UserID], feed)
	}
	for userID, feed := range userFeeds {
		if _, feedWarmErr := feedCache.WarmFeeds(ctx, userID, feed); feedWarmErr != nil {
			logger.Warn("warm feed fail", zap.Error(feedWarmErr))
			err = feedWarmErr
		}
	}
	return
}
func VIPStrToLevel(vipStr string) int {
	switch vipStr {
	case "":
		return -1
	case "vip_0":
		return 0
	case "vip_1":
		return 1
	case "vip_2":
		return 2
	case "vip_3":
		return 3
	case "off":
		return 999
	default:
		return 0
	}
}

func VIPLevelToStr(vipLevel int) string {
	switch vipLevel {
	case -1:
		return ""
	case 0:
		return "vip_0"
	case 1:
		return "vip_1"
	case 2:
		return "vip_2"
	case 3:
		return "vip_3"
	default:
		return "off"
	}
}
