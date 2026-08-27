package service

import (
	"context"
	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/utils"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func bookThermalCacheAddByReadTask(ctx context.Context, s *bookChapterService, userID string, book *model.Book, bookChapter *model.BookChapter) error {
	var err error
	// 用户添加对应书籍数量
	_, _ = s.userStatCache.AddReadBook(ctx, userID, book.ID)

	// 检查锁：阅读频次需要大于1min
	exist, _ := s.limitCache.CheckUserBrowse1Min(ctx, userID)
	if exist {
		return nil
	}

	// 用户添加对应阅读量
	pageWords := int64(utils.CountPassageWord(bookChapter.Content))
	// todo perf:性能优化
	addReadWordsErr := s.userStatCache.IncreaseReadWordsNum(ctx, userID, pageWords)
	if addReadWordsErr != nil {
		logger.Fatal("increase read words fail: %v", zap.String("bookChapterId", bookChapter.ID))
	}

	// 检查锁：每阅读5min记录到书籍
	exist, _ = s.limitCache.CheckUserBrowseBook5Min(ctx, userID, book.ID)
	if exist {
		return nil
	}
	// 添加锁：热力值限流
	_ = s.limitCache.AddUserBrowseBook5Min(ctx, userID, book.ID)
	_ = s.limitCache.AddUserBrowse1Min(ctx, userID)

	// 数据库添加数据
	// todo perf:性能优化
	_, err = s.bookRepo.IncreaseOfBrowseAmount(ctx, s.db, book.ID)
	if err != nil {
		return err
	}

	// 更新数据
	s.pool.SubmitCtx(constant.DoneServerCtx, agilePool.TaskFunc(func() error {
		return bookThermalCacheAddValueTask(constant.DoneServerCtx, s.db, s.bookThermalCache, s.bookCache, s.bookRepo, s.pool, book.ID, 1)
	}))

	return nil
}

// Deprecated: addNewChapterFeedTask 旧版新章节 Feed 推送任务，将被移除
func addNewChapterFeedTask(
	ctx context.Context,
	s *bookChapterService,
	book *model.Book,
	chapter *model.BookChapter,
	db *gorm.DB,
) error {
	userIDs, err := s.userBookshelfRepo.ListUserIDsByBookID(ctx, s.db, book.ID)
	if err != nil {
		return err
	}
	if len(userIDs) == 0 {
		return nil
	}

	feeds := buildNewChapterFeeds(book, chapter, userIDs)
	if len(feeds) > 50 {
		return s.feedClient.PublishFeeds(ctx, s.cfg, feeds, userIDs)
	}

	if _, err := s.feedRepo.CreateBatchIgnoreConflict(ctx, db, feeds); err != nil {
		return err
	}
	return warmFeedCache(ctx, s.feedCache, feeds)
}
