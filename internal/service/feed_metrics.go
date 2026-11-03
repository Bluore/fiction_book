package service

import (
	"sync/atomic"

	"fiction-book/internal/model"
	"fiction-book/pkg/logger"

	"go.uber.org/zap"
)

// feedComposeMetrics Feed 组合缓存命中与回源计数
type feedComposeMetrics struct {
	cardHit  atomic.Int64
	cardMiss atomic.Int64

	sourceFallbackFeed      atomic.Int64
	sourceFallbackBook      atomic.Int64
	sourceFallbackUser      atomic.Int64
	sourceFallbackUserInfo  atomic.Int64
	sourceFallbackUserAsset atomic.Int64
	sourceFallbackLike      atomic.Int64
	sourceFallbackShelf     atomic.Int64
}

// feedComposeMetricsSnapshot 指标快照，供测试与日志使用
type feedComposeMetricsSnapshot struct {
	CardHit                 int64
	CardMiss                int64
	SourceFallbackFeed      int64
	SourceFallbackBook      int64
	SourceFallbackUser      int64
	SourceFallbackUserInfo  int64
	SourceFallbackUserAsset int64
	SourceFallbackLike      int64
	SourceFallbackShelf     int64
}

func newFeedComposeMetrics() *feedComposeMetrics {
	return &feedComposeMetrics{}
}

func (s *feedServiceV2) feedMetrics() *feedComposeMetrics {
	if s.composeMetrics == nil {
		s.composeMetrics = newFeedComposeMetrics()
	}
	return s.composeMetrics
}

func (m *feedComposeMetrics) addSourceFallback(kind model.FeedSourceKind) {
	switch kind {
	case model.FeedSourceFeed:
		m.sourceFallbackFeed.Add(1)
	case model.FeedSourceBook:
		m.sourceFallbackBook.Add(1)
	case model.FeedSourceUser:
		m.sourceFallbackUser.Add(1)
	case model.FeedSourceUserInfo:
		m.sourceFallbackUserInfo.Add(1)
	case model.FeedSourceUserAsset:
		m.sourceFallbackUserAsset.Add(1)
	case model.FeedSourceUserLike:
		m.sourceFallbackLike.Add(1)
	case model.FeedSourceUserBookshelf:
		m.sourceFallbackShelf.Add(1)
	}
}

func (m *feedComposeMetrics) snapshot() feedComposeMetricsSnapshot {
	return feedComposeMetricsSnapshot{
		CardHit:                 m.cardHit.Load(),
		CardMiss:                m.cardMiss.Load(),
		SourceFallbackFeed:      m.sourceFallbackFeed.Load(),
		SourceFallbackBook:      m.sourceFallbackBook.Load(),
		SourceFallbackUser:      m.sourceFallbackUser.Load(),
		SourceFallbackUserInfo:  m.sourceFallbackUserInfo.Load(),
		SourceFallbackUserAsset: m.sourceFallbackUserAsset.Load(),
		SourceFallbackLike:      m.sourceFallbackLike.Load(),
		SourceFallbackShelf:     m.sourceFallbackShelf.Load(),
	}
}

// logRequest 记录本次请求的命中与未命中数量，便于结合指标观察长尾卡片
func (m *feedComposeMetrics) logRequest(feedCount int, cardHit int, cardMiss int) {
	logger.Debug(
		"feed compose cache stats",
		zap.Int("feed_count", feedCount),
		zap.Int("card_hit", cardHit),
		zap.Int("card_miss", cardMiss),
	)
}
