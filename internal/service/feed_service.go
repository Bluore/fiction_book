package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"time"

	"gorm.io/gorm"
)

const feedReadBatchSize = 10

// FeedService 提供用户 Feed 流读取能力。
type FeedService interface {
	// GetFeedList 按游标读取 Feed；本次返回的 Feed 会被标记为已读。
	GetFeedList(ctx context.Context, userID string, req *dto.GetFeedListRequest) (*dto.GetFeedListResponse, error)
}

type feedService struct {
	feedRepo       repository.FeedRepository
	feedCursorRepo repository.FeedCursorRepository
	feedCache      cache.FeedCache
	db             *gorm.DB
}

// feedPageCursor 是对外 cursor 的内部载荷。
// LastFeedID 和 LastCreatedAt 必须一起使用，才能和 DB 的 created_at DESC, id ASC 排序保持一致。
type feedPageCursor struct {
	LastFeedID    string    `json:"last_feed_id"`
	LastCreatedAt time.Time `json:"last_created_at"`
}

func NewFeedService(
	feedRepo repository.FeedRepository,
	feedCursorRepo repository.FeedCursorRepository,
	feedCache cache.FeedCache,
	db *gorm.DB,
) FeedService {
	return &feedService{
		feedRepo:       feedRepo,
		feedCursorRepo: feedCursorRepo,
		feedCache:      feedCache,
		db:             db,
	}
}

func (s *feedService) GetFeedList(
	ctx context.Context,
	userID string,
	req *dto.GetFeedListRequest,
) (*dto.GetFeedListResponse, error) {
	req.ToStandard()

	cursor, err := decodeFeedPageCursor(req.Cursor)
	if err != nil {
		return nil, ErrParam
	}

	pageLimit := req.Limit + 1
	feeds, err := s.loadFeedPage(ctx, userID, cursor, pageLimit)
	if err != nil {
		return nil, err
	}

	// 当前 cursor 已经没有更旧内容时，自动从最新 Feed 重新开始，形成循环 Feed 流。
	restarted := false
	if len(feeds) == 0 && cursor.hasValue() {
		restarted = true
		cursor = feedPageCursor{}
		feeds, err = s.loadFeedPage(ctx, userID, cursor, pageLimit)
		if err != nil {
			return nil, err
		}
	}

	hasMore := len(feeds) > req.Limit
	if hasMore {
		feeds = feeds[:req.Limit]
	}

	if err := s.markFeedsAsRead(ctx, userID, feeds); err != nil {
		return nil, err
	}

	nextCursor, err := encodeNextFeedCursor(feeds, hasMore)
	if err != nil {
		return nil, err
	}

	return &dto.GetFeedListResponse{
		Feeds:      toFeedItemResponses(feeds),
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Restarted:  restarted,
	}, nil
}

// loadFeedPage 优先从缓存读取；缓存缺失或缓存页不足时，再用 DB 补齐。
// 这样既保留热数据读取性能，也避免缓存只保存最近 N 条时把“缓存到底”误判为“全量到底”。
func (s *feedService) loadFeedPage(
	ctx context.Context,
	userID string,
	cursor feedPageCursor,
	limit int,
) ([]*model.FeedInbox, error) {
	feeds, hit, err := s.feedCache.GetFeeds(ctx, userID, cache.FeedListCursor{
		LastFeedID:    cursor.LastFeedID,
		LastCreatedAt: cursor.LastCreatedAt,
		Limit:         int64(limit),
	})
	if err != nil {
		return nil, err
	}
	if !hit {
		return s.loadFeedPageFromDB(ctx, userID, cursor, limit)
	}
	if len(feeds) == 0 {
		return s.loadFeedPageFromDB(ctx, userID, cursor, limit)
	}
	if len(feeds) >= limit {
		return feeds, nil
	}

	nextCursor := feedCursorFromFeed(feeds[len(feeds)-1])
	moreFeeds, err := s.loadFeedPageFromDB(ctx, userID, nextCursor, limit-len(feeds))
	if err != nil {
		return nil, err
	}
	return append(feeds, moreFeeds...), nil
}

// loadFeedPageFromDB 使用稳定的时间+ID游标查询 DB，并把回源结果写回 Feed item 缓存。
func (s *feedService) loadFeedPageFromDB(
	ctx context.Context,
	userID string,
	cursor feedPageCursor,
	limit int,
) ([]*model.FeedInbox, error) {
	filter := model.FeedFilterWithTimePage{
		UserID: userID,
		TimePage: model.TimePage{
			LastTime: cursor.LastCreatedAt,
			Limit:    limit,
		},
	}
	if cursor.LastFeedID != "" {
		filter.LastID = &cursor.LastFeedID
	}

	feeds, err := s.feedRepo.GetUserFeeds(ctx, s.db, filter)
	if err != nil {
		return nil, err
	}
	if len(feeds) > 0 {
		_ = s.feedCache.WarmFeeds(ctx, userID, feeds)
	}
	return feeds, nil
}

// markFeedsAsRead 在 Feed 被返回给客户端后同步标记已读。
// 标记更新按 10 条一组执行，并只用实际从未读变已读的行数扣减未读计数。
func (s *feedService) markFeedsAsRead(ctx context.Context, userID string, feeds []*model.FeedInbox) error {
	if len(feeds) == 0 {
		return nil
	}

	cursor, err := s.feedCursorRepo.GetOrCreate(ctx, userID)
	if err != nil {
		return err
	}

	var markedCount int64
	for _, feedIDs := range chunkFeedIDs(feedIDsFromFeeds(feeds), feedReadBatchSize) {
		count, err := s.feedRepo.MarkAsRead(ctx, s.db, userID, feedIDs)
		if err != nil {
			return err
		}
		markedCount += count
	}

	if markedCount > 0 {
		if err := s.feedCursorRepo.UpdateUnreadCount(ctx, userID, -int(markedCount)); err != nil {
			return err
		}
		_ = s.feedCache.DecrUnreadCount(ctx, userID, markedCount)
		cursor.UnreadCount -= int(markedCount)
		if cursor.UnreadCount < 0 {
			cursor.UnreadCount = 0
		}
	}

	lastFeed := feeds[len(feeds)-1]
	if err := s.feedCursorRepo.UpdateLastRead(ctx, userID, lastFeed.ID); err != nil {
		return err
	}

	now := time.Now()
	cursor.LastReadAt = &now
	cursor.LastFeedID = &lastFeed.ID
	cursor.UpdatedAt = now
	_ = s.feedCache.SetCursor(ctx, userID, cursor)

	for _, feed := range feeds {
		feed.IsRead = true
	}
	_ = s.feedCache.WarmFeeds(ctx, userID, feeds)

	return nil
}

func decodeFeedPageCursor(cursor string) (feedPageCursor, error) {
	if cursor == "" {
		return feedPageCursor{}, nil
	}

	cursorJSON, err := base64.RawURLEncoding.DecodeString(cursor)
	if err != nil {
		return feedPageCursor{}, err
	}

	var decoded feedPageCursor
	if err := json.Unmarshal(cursorJSON, &decoded); err != nil {
		return feedPageCursor{}, err
	}
	hasFeedID := decoded.LastFeedID != ""
	hasCreatedAt := !decoded.LastCreatedAt.IsZero()
	if hasFeedID != hasCreatedAt {
		return feedPageCursor{}, errors.New("invalid feed cursor")
	}
	return decoded, nil
}

func encodeNextFeedCursor(feeds []*model.FeedInbox, hasMore bool) (string, error) {
	if len(feeds) == 0 || !hasMore {
		return "", nil
	}

	cursor := feedCursorFromFeed(feeds[len(feeds)-1])
	cursorJSON, err := json.Marshal(cursor)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(cursorJSON), nil
}

func feedCursorFromFeed(feed *model.FeedInbox) feedPageCursor {
	return feedPageCursor{
		LastFeedID:    feed.ID,
		LastCreatedAt: feed.CreatedAt,
	}
}

func (c feedPageCursor) hasValue() bool {
	return c.LastFeedID != "" && !c.LastCreatedAt.IsZero()
}

func feedIDsFromFeeds(feeds []*model.FeedInbox) []string {
	feedIDs := make([]string, 0, len(feeds))
	for _, feed := range feeds {
		feedIDs = append(feedIDs, feed.ID)
	}
	return feedIDs
}

func chunkFeedIDs(feedIDs []string, size int) [][]string {
	if len(feedIDs) == 0 {
		return [][]string{}
	}

	chunks := make([][]string, 0, (len(feedIDs)+size-1)/size)
	for start := 0; start < len(feedIDs); start += size {
		end := start + size
		if end > len(feedIDs) {
			end = len(feedIDs)
		}
		chunks = append(chunks, feedIDs[start:end])
	}
	return chunks
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
