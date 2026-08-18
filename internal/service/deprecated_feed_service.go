package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Deprecated: 旧版 feed 相关常量
const (
	feedListPageSize  = 10
	feedReadBatchSize = 10
)

// Deprecated: FeedService 旧版用户 Feed 流读取服务，将被新的 feed 架构替代
type FeedService interface {
	// Deprecated: GetFeedList 按服务端游标读取 Feed；本次返回的 Feed 会被标记为已读
	Deprecate_GetFeedList(ctx context.Context, userID string) (*dto.GetFeedListResponse, error)
}

type feedService struct {
	feedRepo  repository.FeedRepository
	feedCache cache.FeedCache
	db        *gorm.DB
}

// Deprecated: feedPageCursor 旧版对外 cursor 的内部载荷
// LastFeedID 和 LastCreatedAt 必须一起使用，才能和 DB 的 created_at DESC, id ASC 排序保持一致。
type feedPageCursor struct {
	LastFeedID    string    `json:"last_feed_id"`
	LastCreatedAt time.Time `json:"last_created_at"`
}

func NewFeedService(
	feedRepo repository.FeedRepository,
	feedCache cache.FeedCache,
	db *gorm.DB,
) FeedService {
	return &feedService{
		feedRepo:  feedRepo,
		feedCache: feedCache,
		db:        db,
	}
}

func (s *feedService) Deprecate_GetFeedList(
	ctx context.Context,
	userID string,
) (*dto.GetFeedListResponse, error) {
	var cursor *model.FeedCursor
	// 缓存获取 Cursor
	cursor, _, _ = s.feedCache.GetCursor(ctx, userID)
	if !cursor.HasValue() {
		cursor = &model.FeedCursor{}
	}

	// 获取 Feed内容
	feedFilter := &model.FeedFilterWithTimePage{
		UserID:     &userID,
		EventTypes: []model.FeedEventType{},
		BookID:     nil,
		IsRead:     nil,
		TimePage: model.TimePage{
			LastTime: cursor.LastReadAt,
			LastID:   &cursor.LastFeedID,
			Page:     0,
			Limit:    feedListPageSize,
		},
	}

	feedsIDs, err := s.feedRepo.GetUserFeedsIDs(ctx, s.db, feedFilter)
	if err != nil {
		return nil, err
	}

	// 当前 cursor 已经没有更旧内容时，自动从最新 Feed 重新开始，形成循环 Feed 流。
	cursorRestarted := false
	if len(feedsIDs) == 0 {
		cursorRestarted = true
		feedFilter.TimePage = model.TimePage{
			Page:  0,
			Limit: feedListPageSize,
		}
		feedsIDs, err = s.feedRepo.GetUserFeedsIDs(ctx, s.db, feedFilter)
		if err != nil {
			return nil, err
		}
	}

	// 标记已读
	if _, err := s.feedRepo.MarkAsRead(ctx, s.db, userID, feedsIDs); err != nil {
		return nil, err
	}

	// 获取 Feeds
	var feeds []*model.FeedInbox = make([]*model.FeedInbox, 0, len(feedsIDs))
	for _, feedID := range feedsIDs {
		var feedItem *model.FeedInbox
		feedItem, feedItemHit, err := s.feedCache.GetFeedSingle(ctx, userID, feedID)
		if feedItemHit {
			feeds = append(feeds, feedItem)
			continue
		}
		if err != nil {
			logger.Warn("get feeds by cache failed", zap.Error(err))
		}

		feedItem, _, err = s.feedRepo.GetUserFeedSingle(ctx, s.db, feedID)
		if err != nil {
			logger.Warn("get feeds by cache failed", zap.Error(err))
			continue
		}
		feeds = append(feeds, feedItem)
	}

	if len(feeds) == 0 {
		return &dto.GetFeedListResponse{
			Feeds:     make([]*dto.FeedItemResponse, 0),
			Restarted: false,
		}, nil
	}

	// 设置游标 Cursor
	lastFeedItem := feeds[len(feeds)-1]
	nextCursor := &model.FeedCursor{
		UserID:     userID,
		LastFeedID: lastFeedItem.ID,
		LastReadAt: lastFeedItem.CreatedAt,
	}
	err = s.feedCache.SetCursor(ctx, userID, nextCursor)
	if err != nil {
		logger.Warn("set cursor failed", zap.Error(err))
	}

	return &dto.GetFeedListResponse{
		Feeds:     toFeedItemResponses(feeds),
		Restarted: cursorRestarted,
	}, nil
}

func toFeedItemResponses(feeds []*model.FeedInbox) []*dto.FeedItemResponse {
	responses := make([]*dto.FeedItemResponse, 0, len(feeds))
	for _, feed := range feeds {
		responses = append(responses, &dto.FeedItemResponse{
			FeedID:       feed.ID,
			EventType:    feed.EventType,
			BookID:       feed.BookID,
			ChapterID:    feed.ChapterID,
			BookName:     feed.BookName,
			BookCover:    feed.BookCover,
			ChapterTitle: feed.ChapterTitle,
			Content:      feed.Content,
			CreatedAt:    feed.CreatedAt,
		})
	}
	return responses
}
