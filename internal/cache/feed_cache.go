package cache

import (
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyFeedItem        = "GSF:feed:cache:inbox:%s:items:%s"
	keyFeedUnreadCount = "GSF:feed:cache:inbox:%s:unread_count"
	keyFeedCursor      = "GSF:feed:cache:inbox:%s:cursor"

	defaultFeedCacheTTL = 7 * 24 * time.Hour
	defaultFeedMaxLen   = int64(500)
)

var decrUnreadCountScript = redis.NewScript(`
local current = redis.call("GET", KEYS[1])
if not current then
	current = 0
else
	current = tonumber(current)
end

local next = current - tonumber(ARGV[1])
if next < 0 then
	next = 0
end

redis.call("SET", KEYS[1], next, "EX", ARGV[2])
return next
`)

type FeedListCursor struct {
	LastFeedID    string    `json:"last_feed_id"`
	LastCreatedAt time.Time `json:"last_created_at"`
	Limit         int64     `json:"limit"`
}

type FeedCache interface {
	// PushFeeds 写推送：将新 feed 写入用户 feed item 缓存；同时裁剪至 MaxLen
	PushFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) error

	// GetFeeds 游标分页读取缓存 feed；hit=false 表示 key 不存在（需回源）
	GetFeeds(ctx context.Context, userID string, cursor FeedListCursor) (feeds []*model.FeedInbox, hit bool, err error)

	// TrimFeeds 裁剪用户 feed item 缓存，仅保留最新 maxLen 条（PushFeeds 内部调用，也可手动触发）
	TrimFeeds(ctx context.Context, userID string, maxLen int64) error

	// WarmFeeds 缓存预热：将 DB 查出的 feeds 写入 item 缓存（首次回源后调用）
	WarmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) error

	// GetUnreadCount 读取未读数；hit=false 表示缓存未命中
	GetUnreadCount(ctx context.Context, userID string) (count int64, hit bool, err error)

	// IncrUnreadCount 单用户未读数 +delta（写推送时调用）
	IncrUnreadCount(ctx context.Context, userID string, delta int64) error

	// BatchIncrUnreadCount 批量 +delta，Pipeline 减少 RTT（写推送扇出场景）
	BatchIncrUnreadCount(ctx context.Context, userIDs []string, delta int64) error

	// DecrUnreadCount 部分已读时-delta（下界为 0，不允许负数）
	DecrUnreadCount(ctx context.Context, userID string, delta int64) error

	// ResetUnreadCount 全部已读时清零
	ResetUnreadCount(ctx context.Context, userID string) error

	// GetCursor 读取游标；hit=false 表示缓存未命中
	GetCursor(ctx context.Context, userID string) (cursor *model.FeedCursor, hit bool, err error)

	// SetCursor 写入/更新游标缓存
	SetCursor(ctx context.Context, userID string, cursor *model.FeedCursor) error

	// DeleteCursor 主动失效（游标被DB 直接修改后调用）
	DeleteCursor(ctx context.Context, userID string) error
}

type feedCache struct {
	redis *redis.Client
}

func NewFeedCache(redis *redis.Client) FeedCache {
	return &feedCache{redis: redis}
}

func KeyFeedItem(userID string, feedID string) string {
	return fmt.Sprintf(keyFeedItem, userID, feedID)
}

func KeyFeedUnreadCount(userID string) string {
	return fmt.Sprintf(keyFeedUnreadCount, userID)
}

func KeyFeedCursor(userID string) string {
	return fmt.Sprintf(keyFeedCursor, userID)
}

func (c *feedCache) PushFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) error {
	if len(feeds) == 0 {
		return nil
	}

	if err := c.warmFeeds(ctx, userID, feeds); err != nil {
		return err
	}
	return c.TrimFeeds(ctx, userID, defaultFeedMaxLen)
}

func (c *feedCache) GetFeeds(
	ctx context.Context,
	userID string,
	cursor FeedListCursor,
) ([]*model.FeedInbox, bool, error) {
	limit := cursor.Limit
	if limit <= 0 {
		limit = 20
	}

	feeds, hit, err := c.getUserFeedItems(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if !hit {
		return nil, false, nil
	}

	start := feedPageStart(feeds, cursor.LastFeedID)
	if start < 0 {
		return nil, false, nil
	}
	if start >= int64(len(feeds)) {
		return []*model.FeedInbox{}, true, nil
	}

	end := start + limit
	if end > int64(len(feeds)) {
		end = int64(len(feeds))
	}

	return feeds[start:end], true, nil
}

func (c *feedCache) TrimFeeds(ctx context.Context, userID string, maxLen int64) error {
	if maxLen <= 0 {
		maxLen = defaultFeedMaxLen
	}

	feeds, hit, err := c.getUserFeedItems(ctx, userID)
	if err != nil {
		return err
	}
	if !hit || int64(len(feeds)) <= maxLen {
		return nil
	}

	staleFeeds := feeds[maxLen:]
	staleKeys := make([]string, 0, len(staleFeeds))
	for _, feed := range staleFeeds {
		staleKeys = append(staleKeys, KeyFeedItem(userID, feed.ID))
	}

	pipe := c.redis.Pipeline()
	pipe.Del(ctx, staleKeys...)
	_, err = pipe.Exec(ctx)
	return err
}

func (c *feedCache) WarmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) error {
	if len(feeds) == 0 {
		return nil
	}

	if err := c.warmFeeds(ctx, userID, feeds); err != nil {
		return err
	}
	return c.TrimFeeds(ctx, userID, defaultFeedMaxLen)
}

func (c *feedCache) GetUnreadCount(ctx context.Context, userID string) (int64, bool, error) {
	count, err := c.redis.Get(ctx, KeyFeedUnreadCount(userID)).Result()
	if err == redis.Nil {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}

	res, err := strconv.ParseInt(count, 10, 64)
	if err != nil {
		return 0, false, err
	}
	return res, true, nil
}

func (c *feedCache) IncrUnreadCount(ctx context.Context, userID string, delta int64) error {
	pipe := c.redis.Pipeline()
	pipe.IncrBy(ctx, KeyFeedUnreadCount(userID), delta)
	pipe.Expire(ctx, KeyFeedUnreadCount(userID), defaultFeedCacheTTL)
	_, err := pipe.Exec(ctx)
	return err
}

func (c *feedCache) BatchIncrUnreadCount(ctx context.Context, userIDs []string, delta int64) error {
	if len(userIDs) == 0 {
		return nil
	}

	pipe := c.redis.Pipeline()
	for _, userID := range userIDs {
		pipe.IncrBy(ctx, KeyFeedUnreadCount(userID), delta)
		pipe.Expire(ctx, KeyFeedUnreadCount(userID), defaultFeedCacheTTL)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (c *feedCache) DecrUnreadCount(ctx context.Context, userID string, delta int64) error {
	return decrUnreadCountScript.Run(
		ctx,
		c.redis,
		[]string{KeyFeedUnreadCount(userID)},
		delta,
		int64(defaultFeedCacheTTL.Seconds()),
	).Err()
}

func (c *feedCache) ResetUnreadCount(ctx context.Context, userID string) error {
	return c.redis.Set(ctx, KeyFeedUnreadCount(userID), 0, defaultFeedCacheTTL).Err()
}

func (c *feedCache) GetCursor(ctx context.Context, userID string) (*model.FeedCursor, bool, error) {
	cursorRes, err := c.redis.Get(ctx, KeyFeedCursor(userID)).Result()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var cursor model.FeedCursor
	if err := json.Unmarshal([]byte(cursorRes), &cursor); err != nil {
		return nil, false, err
	}
	return &cursor, true, nil
}

func (c *feedCache) SetCursor(ctx context.Context, userID string, cursor *model.FeedCursor) error {
	cursorEncode, err := json.Marshal(cursor)
	if err != nil {
		return err
	}
	return c.redis.Set(ctx, KeyFeedCursor(userID), cursorEncode, defaultFeedCacheTTL).Err()
}

func (c *feedCache) DeleteCursor(ctx context.Context, userID string) error {
	return c.redis.Del(ctx, KeyFeedCursor(userID)).Err()
}

func (c *feedCache) warmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) error {
	pipe := c.redis.Pipeline()
	for _, feed := range feeds {
		feedEncode, err := json.Marshal(feed)
		if err != nil {
			return err
		}

		pipe.Set(ctx, KeyFeedItem(userID, feed.ID), feedEncode, defaultFeedCacheTTL)
	}

	_, err := pipe.Exec(ctx)
	return err
}

func (c *feedCache) getUserFeedItems(ctx context.Context, userID string) ([]*model.FeedInbox, bool, error) {
	feedKeys, err := c.scanFeedItemKeys(ctx, userID)
	if err != nil {
		return nil, false, err
	}
	if len(feedKeys) == 0 {
		return nil, false, nil
	}

	feedItems, err := c.redis.MGet(ctx, feedKeys...).Result()
	if err != nil {
		return nil, false, err
	}

	feeds := make([]*model.FeedInbox, 0, len(feedItems))
	for _, feedItem := range feedItems {
		if feedItem == nil {
			return nil, false, nil
		}

		feedJSON, ok := feedItem.(string)
		if !ok {
			return nil, false, nil
		}

		var feed model.FeedInbox
		if err := json.Unmarshal([]byte(feedJSON), &feed); err != nil {
			return nil, false, err
		}
		feeds = append(feeds, &feed)
	}

	sortFeeds(feeds)
	return feeds, true, nil
}

func (c *feedCache) scanFeedItemKeys(ctx context.Context, userID string) ([]string, error) {
	keys := make([]string, 0)
	var cursor uint64

	for {
		var batch []string
		var err error
		batch, cursor, err = c.redis.Scan(ctx, cursor, KeyFeedItem(userID, "*"), 100).Result()
		if err != nil {
			return nil, err
		}

		keys = append(keys, batch...)
		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func sortFeeds(feeds []*model.FeedInbox) {
	sort.SliceStable(feeds, func(i int, j int) bool {
		if feeds[i].CreatedAt.Equal(feeds[j].CreatedAt) {
			return feeds[i].ID < feeds[j].ID
		}
		return feeds[i].CreatedAt.After(feeds[j].CreatedAt)
	})
}

func feedPageStart(feeds []*model.FeedInbox, lastFeedID string) int64 {
	if lastFeedID == "" {
		return 0
	}

	for i, feed := range feeds {
		if feed.ID == lastFeedID {
			return int64(i + 1)
		}
	}
	return -1
}
