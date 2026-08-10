package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/model"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyFeedItem   = "GSF:feed:cache:inbox:%s:items:%s"
	keyFeedCursor = "GSF:feed:cache:inbox:%s:cursor"

	defaultFeedCacheTTL = 7 * 24 * time.Hour
	defaultFeedMaxLen   = int64(500)
)

type FeedListCursor struct {
	LastFeedID    string    `json:"last_feed_id"`
	LastCreatedAt time.Time `json:"last_created_at"`
	Limit         int64     `json:"limit"`
}

type FeedCache interface {
	// DelFeeds 批量删除缓存
	DelFeeds(ctx context.Context, userID string, feedIDs []string) (successNum int64, err error)

	// WarmFeeds 缓存预热：将 DB 查出的 feeds 写入 item 缓存（首次回源后调用）
	WarmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) (successNum int64, err error)

	// GetFeeds 获取一组Feeds
	GetFeeds(ctx context.Context, userID string, feedIDs []string) (feeds []*model.FeedInbox, err error)

	// GetFeedSingle 获取单个Feeds
	GetFeedSingle(ctx context.Context, userID string, feedID string) (feeds *model.FeedInbox, hit bool, err error)

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

func KeyFeedCursor(userID string) string {
	return fmt.Sprintf(keyFeedCursor, userID)
}

func (c *feedCache) DelFeeds(ctx context.Context, userID string, feedIDs []string) (successNum int64, err error) {
	successNum = 0
	for _, feedID := range feedIDs {
		err = c.redis.Del(ctx, KeyFeedItem(userID, feedID)).Err()
		if err != nil {
			continue
		}
		successNum++
	}
	return
}
func (c *feedCache) WarmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) (int64, error) {
	if len(feeds) == 0 {
		return 0, nil
	}
	num, err := c.warmFeeds(ctx, userID, feeds)
	if err != nil {
		return num, err
	}
	return num, err
}
func (c *feedCache) GetFeeds(ctx context.Context, userID string, feedIDs []string) ([]*model.FeedInbox, error) {
	var feeds = make([]*model.FeedInbox, 0, len(feedIDs))
	var err error
	for _, feedID := range feedIDs {
		feedItem, hit, feedItemErr := c.GetFeedSingle(ctx, userID, feedID)
		if !hit {
			if feedItemErr == nil {
				err = feedItemErr
			}
			continue
		}
		feeds = append(feeds, feedItem)
	}
	return feeds, err
}

func (c *feedCache) GetFeedSingle(ctx context.Context, userID string, feedID string) (*model.FeedInbox, bool, error) {
	var result string
	var err error
	result, err = c.redis.Get(ctx, KeyFeedItem(userID, feedID)).Result()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var feedItem model.FeedInbox
	err = json.Unmarshal([]byte(result), &feedItem)
	if err != nil {
		return nil, false, err
	}
	return &feedItem, true, nil
}

func (c *feedCache) GetCursor(ctx context.Context, userID string) (*model.FeedCursor, bool, error) {
	cursorRes, err := c.redis.Get(ctx, KeyFeedCursor(userID)).Result()
	if errors.Is(err, redis.Nil) {
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

func (c *feedCache) warmFeeds(ctx context.Context, userID string, feeds []*model.FeedInbox) (successNum int64, err error) {
	successNum = 0
	for _, feed := range feeds {
		var feedEncode []byte
		feedEncode, err = json.Marshal(feed)
		if err != nil {
			continue
		}

		err = c.redis.Set(ctx, KeyFeedItem(userID, feed.ID), feedEncode, defaultFeedCacheTTL).Err()
		if err != nil {
			continue
		}
		successNum++
	}
	return
}
