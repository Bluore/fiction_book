package cache

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"fiction-book/internal/model"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	keyFeedCard = "GSF:feed:cache:card:v1:%s"

	timeFeedCard = 60 * time.Second
)

// errCacheValueEmpty 缓存值为 EMPTY 墓碑，表示该实体确定不存在
var errCacheValueEmpty = errors.New("cache value empty")

// KeyFeedCard Feed 公共组合卡片缓存 key
func KeyFeedCard(feedID string) string {
	return fmt.Sprintf(keyFeedCard, feedID)
}

// TimeFeedCard 卡片缓存有效期，60s 加 0~10% 抖动
func TimeFeedCard() time.Duration {
	return timeFeedCard + TimeShakeDuration(timeFeedCard)
}

// FeedComposeCache Feed 公共卡片与查看者状态批量缓存
type FeedComposeCache interface {
	// GetCards 批量获取公共组合卡片，missed 为需要回源的 feedID
	GetCards(ctx context.Context, feedIDs []string) (cards map[string]*model.FeedCard, missed []string, err error)
	// GetViewerStates 批量获取当前查看者的点赞/书架状态。
	// map 中存在 key 表示命中：true 表示记录存在，false 表示 EMPTY（确定不存在）；
	// map 中不存在 key 表示缓存未命中，调用方需要回源。
	GetViewerStates(ctx context.Context, viewerID string, authorIDs []string, bookIDs []string) (likes map[string]bool, shelves map[string]bool, err error)
	// UpdateCards 批量写回公共组合卡片
	UpdateCards(ctx context.Context, cards []*model.FeedCard) error
	// DeleteCard 按 feedID 删除公共组合卡片
	DeleteCard(ctx context.Context, feedID string) error
}

type feedComposeCache struct {
	redis *redis.Client
}

// NewFeedComposeCache 创建 Feed 公共组合缓存
func NewFeedComposeCache(redis *redis.Client) FeedComposeCache {
	return &feedComposeCache{redis: redis}
}

// decodeCacheValue 通用缓存解码：识别 EMPTY 墓碑，否则按 JSON 解码
func decodeCacheValue(value string, target any) error {
	if value == constant.RedisValueEmpty {
		return errCacheValueEmpty
	}
	return json.Unmarshal([]byte(value), target)
}

// getStringCmds 用一条 pipeline 读取所有 key。
// pipeline 内单个 key miss 会让 Exec 返回 redis.Nil，这里只把网络/协议错误上抛，miss 交给调用方逐条判断。
func (c *feedComposeCache) getStringCmds(ctx context.Context, keys []string) ([]*redis.StringCmd, error) {
	if len(keys) == 0 {
		return nil, nil
	}

	pipe := c.redis.Pipeline()
	cmds := make([]*redis.StringCmd, 0, len(keys))
	for _, key := range keys {
		cmds = append(cmds, pipe.Get(ctx, key))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, err
	}
	return cmds, nil
}

// dedupeStrings 去重并保持顺序，忽略空字符串
func dedupeStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// GetCards 批量获取公共组合卡片
func (c *feedComposeCache) GetCards(ctx context.Context, feedIDs []string) (map[string]*model.FeedCard, []string, error) {
	feedIDs = dedupeStrings(feedIDs)
	cards := make(map[string]*model.FeedCard, len(feedIDs))
	if len(feedIDs) == 0 {
		return cards, make([]string, 0), nil
	}

	keys := make([]string, 0, len(feedIDs))
	for _, feedID := range feedIDs {
		keys = append(keys, KeyFeedCard(feedID))
	}
	cmds, err := c.getStringCmds(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	missed := make([]string, 0)
	for i, feedID := range feedIDs {
		cmd := cmds[i]
		if errors.Is(cmd.Err(), redis.Nil) {
			missed = append(missed, feedID)
			continue
		}
		if cmd.Err() != nil {
			logger.Warn("get feed card cache fail", zap.String("feed_id", feedID), zap.Error(cmd.Err()))
			missed = append(missed, feedID)
			continue
		}

		var card model.FeedCard
		if err := decodeCacheValue(cmd.Val(), &card); err != nil {
			if errors.Is(err, errCacheValueEmpty) {
				// EMPTY 表示确定不存在，不加入 missed，避免重复回源
				continue
			}
			logger.Warn("decode feed card cache fail", zap.String("feed_id", feedID), zap.Error(err))
			missed = append(missed, feedID)
			continue
		}
		cards[feedID] = &card
	}
	return cards, missed, nil
}

// GetViewerStates 用一条 pipeline 批量读取去重后的点赞和书架状态
func (c *feedComposeCache) GetViewerStates(
	ctx context.Context,
	viewerID string,
	authorIDs []string,
	bookIDs []string,
) (map[string]bool, map[string]bool, error) {
	likes := make(map[string]bool)
	shelves := make(map[string]bool)

	authorIDs = dedupeStrings(authorIDs)
	bookIDs = dedupeStrings(bookIDs)
	if len(authorIDs) == 0 && len(bookIDs) == 0 {
		return likes, shelves, nil
	}

	keys := make([]string, 0, len(authorIDs)+len(bookIDs))
	for _, authorID := range authorIDs {
		keys = append(keys, KeyUserLike(authorID, viewerID))
	}
	for _, bookID := range bookIDs {
		keys = append(keys, KeyUserBookshelf(viewerID, bookID))
	}

	cmds, err := c.getStringCmds(ctx, keys)
	if err != nil {
		return nil, nil, err
	}

	index := 0
	for _, authorID := range authorIDs {
		cmd := cmds[index]
		index++
		if errors.Is(cmd.Err(), redis.Nil) {
			continue
		}
		if cmd.Err() != nil {
			logger.Warn("get user_like cache fail", zap.String("author_id", authorID), zap.Error(cmd.Err()))
			continue
		}

		var userLike model.UserLike
		if err := decodeCacheValue(cmd.Val(), &userLike); err != nil {
			if errors.Is(err, errCacheValueEmpty) {
				likes[authorID] = false
				continue
			}
			logger.Warn("decode user_like cache fail", zap.String("author_id", authorID), zap.Error(err))
			continue
		}
		likes[authorID] = true
	}

	for _, bookID := range bookIDs {
		cmd := cmds[index]
		index++
		if errors.Is(cmd.Err(), redis.Nil) {
			continue
		}
		if cmd.Err() != nil {
			logger.Warn("get user_bookshelf cache fail", zap.String("book_id", bookID), zap.Error(cmd.Err()))
			continue
		}

		var userBookshelf model.UserBookshelf
		if err := decodeCacheValue(cmd.Val(), &userBookshelf); err != nil {
			if errors.Is(err, errCacheValueEmpty) {
				shelves[bookID] = false
				continue
			}
			logger.Warn("decode user_bookshelf cache fail", zap.String("book_id", bookID), zap.Error(err))
			continue
		}
		shelves[bookID] = true
	}

	return likes, shelves, nil
}

// UpdateCards 批量写回公共组合卡片，使用带抖动的短 TTL
func (c *feedComposeCache) UpdateCards(ctx context.Context, cards []*model.FeedCard) error {
	if len(cards) == 0 {
		return nil
	}

	pipe := c.redis.Pipeline()
	for _, card := range cards {
		if card == nil {
			continue
		}
		content, err := json.Marshal(card)
		if err != nil {
			return err
		}
		pipe.Set(ctx, KeyFeedCard(card.FeedID), content, TimeFeedCard())
	}
	_, err := pipe.Exec(ctx)
	return err
}

// DeleteCard 按 feedID 删除公共组合卡片
func (c *feedComposeCache) DeleteCard(ctx context.Context, feedID string) error {
	return c.redis.Del(ctx, KeyFeedCard(feedID)).Err()
}
