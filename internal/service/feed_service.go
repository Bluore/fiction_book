package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	MINTime = time.Date(2026, time.January, 1, 1, 0, 0, 0, time.UTC)
)

type FeedServiceV2 interface {
	// GetUserFeed 获取用户 Feed 流
	GetUserFeed(ctx context.Context, userID string, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error)
	// GuestGetUserFeed 获取用户 Feed 流 （未登录用户）
	GuestGetUserFeed(ctx context.Context, req *dto.GetUserFeedRequestV2) (*dto.GetGuestUserFeedResponseV2, error)
}

type feedServiceV2 struct {
	bookRepo           repository.BookRepository
	bookTagRepo        repository.BookTagRepository
	bookESRepo         repository.BookESRepository
	userRepo           repository.UserRepository
	userAssetRepo      repository.UserAssetRepository
	userLikeRepo       repository.UserLikeRepository
	purchasedBookRepo  repository.UserPurchasedBookRepository
	userFeedRepo       repository.UserFeedRepository
	userInboxRepo      repository.UserInboxRepository
	outboxRepo         repository.OutboxRepository
	userInfoRepo       repository.UserInfoRepository
	userBookshelfRepo  repository.UserBookshelfRepository
	userCache          cache.UserCache
	userAssetCache     cache.UserAssetCache
	bookCache          cache.BookCache
	userStatCache      cache.UserStatCache
	bookThermalCache   cache.BookThermalCache
	bookTagCache       cache.BookTagCache
	feedCache          cache.UserFeedCache
	feedComposeCache   cache.FeedComposeCache
	userLikeCache      cache.UserLikeCache
	userInfoCache      cache.UserInfoCache
	userBookshelfCache cache.UserBookshelfCache
	composeMetrics     *feedComposeMetrics
	db                 *gorm.DB
	cfg                *config.Config
}

// NewFeedServiceV2 创建 Feed 服务 V2 实例
func NewFeedServiceV2(
	bookRepo repository.BookRepository,
	bookTagRepo repository.BookTagRepository,
	bookESRepo repository.BookESRepository,
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	userLikeRepo repository.UserLikeRepository,
	purchasedBookRepo repository.UserPurchasedBookRepository,
	userCache cache.UserCache,
	userAssetCache cache.UserAssetCache,
	userFeedRepo repository.UserFeedRepository,
	userInboxRepo repository.UserInboxRepository,
	outboxRepo repository.OutboxRepository,
	userInfoRepo repository.UserInfoRepository,
	userBookshelfRepo repository.UserBookshelfRepository,
	bookCache cache.BookCache,
	userStatCache cache.UserStatCache,
	bookThermalCache cache.BookThermalCache,
	bookTagCache cache.BookTagCache,
	feedCache cache.UserFeedCache,
	feedComposeCache cache.FeedComposeCache,
	userLikeCache cache.UserLikeCache,
	userInfoCache cache.UserInfoCache,
	userBookshelfCache cache.UserBookshelfCache,
	db *gorm.DB,
	cfg *config.Config,
) FeedServiceV2 {
	return &feedServiceV2{
		bookRepo:           bookRepo,
		bookTagRepo:        bookTagRepo,
		bookESRepo:         bookESRepo,
		userRepo:           userRepo,
		userAssetRepo:      userAssetRepo,
		userLikeRepo:       userLikeRepo,
		purchasedBookRepo:  purchasedBookRepo,
		userCache:          userCache,
		userAssetCache:     userAssetCache,
		userFeedRepo:       userFeedRepo,
		userInboxRepo:      userInboxRepo,
		outboxRepo:         outboxRepo,
		userInfoRepo:       userInfoRepo,
		userBookshelfRepo:  userBookshelfRepo,
		bookCache:          bookCache,
		userStatCache:      userStatCache,
		bookThermalCache:   bookThermalCache,
		bookTagCache:       bookTagCache,
		feedCache:          feedCache,
		feedComposeCache:   feedComposeCache,
		userLikeCache:      userLikeCache,
		userInfoCache:      userInfoCache,
		userBookshelfCache: userBookshelfCache,
		composeMetrics:     newFeedComposeMetrics(),
		db:                 db,
		cfg:                cfg,
	}
}

// Feed 组合缓存失效约定：未来新增 Feed 删除或更新路径时，必须在同一流程中调用
// feedComposeCache.DeleteCard(feedID)，否则旧卡片会保留到 TTL 到期（60s + 抖动）。
func (s *feedServiceV2) GetUserFeed(ctx context.Context, userID string, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error) {
	nextCursor, feedIDList, err := s.queryFeedIDListWithCache(ctx, userID, req.LastCursorID, req.TargetCursorID, req.Offset)
	if err != nil {
		return nil, err
	}
	if nextCursor == nil {
		return &dto.GetUserFeedResponseV2{
			Feeds:        make([]*dto.FeedResponse, 0),
			NextCursorID: model.DefaultFeedCursor,
			NextOffset:   0,
		}, nil
	}

	cards, toWrite := s.loadFeedCards(ctx, feedIDList)
	likes, shelves := s.loadViewerStates(ctx, userID, cards)

	feedListResponse := make([]*dto.FeedResponse, 0, len(feedIDList))
	for _, feedID := range feedIDList {
		card := cards[feedID]
		if card == nil {
			continue
		}

		isUserLiker, isInBookShelf, ok := s.viewerState(ctx, userID, card, likes, shelves)
		if !ok {
			continue
		}
		feedListResponse = append(feedListResponse, feedResponseFromCard(feedID, card, isUserLiker, isInBookShelf))
	}

	s.writeBackCards(ctx, toWrite)

	return &dto.GetUserFeedResponseV2{
		Feeds:        feedListResponse,
		NextCursorID: nextCursor.nextCursorID,
		NextOffset:   nextCursor.nextOffset,
	}, nil
}

func (s *feedServiceV2) GuestGetUserFeed(ctx context.Context, req *dto.GetUserFeedRequestV2) (*dto.GetGuestUserFeedResponseV2, error) {
	nextCursor, feedIDList, err := s.queryGuestFeedIDListWithCache(ctx, req.LastCursorID, req.TargetCursorID, req.Offset)
	if err != nil {
		return nil, err
	}
	if nextCursor == nil {
		return &dto.GetGuestUserFeedResponseV2{
			Feeds:        make([]*dto.GuestFeedResponse, 0),
			NextCursorID: model.DefaultFeedCursor,
			NextOffset:   0,
		}, nil
	}

	// 游客与登录用户共用公共卡片，但不读取 like/shelf 状态
	cards, toWrite := s.loadFeedCards(ctx, feedIDList)
	feedListResponse := make([]*dto.GuestFeedResponse, 0, len(feedIDList))
	for _, feedID := range feedIDList {
		card := cards[feedID]
		if card == nil {
			continue
		}
		feedListResponse = append(feedListResponse, guestFeedResponseFromCard(feedID, card))
	}

	s.writeBackCards(ctx, toWrite)

	return &dto.GetGuestUserFeedResponseV2{
		Feeds:        feedListResponse,
		NextCursorID: nextCursor.nextCursorID,
		NextOffset:   nextCursor.nextOffset,
	}, nil
}

// viewerState 优先使用批量状态 map；map 中没有对应 key 时回退原有单条 helper。
// 返回值第三个字段为 false 表示 shelf 查询错误，调用方应跳过该条 Feed。
func (s *feedServiceV2) viewerState(
	ctx context.Context,
	viewerID string,
	card *model.FeedCard,
	likes map[string]bool,
	shelves map[string]bool,
) (bool, bool, bool) {
	isUserLiker, ok := likes[card.UserID]
	if !ok {
		s.feedMetrics().addSourceFallback(model.FeedSourceUserLike)
		userLikeDetail, err := queryUserLikeDetailWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, card.UserID, viewerID)
		if err != nil {
			// 保持现有行为：点赞查询失败只记录日志并视为未点赞
			logger.Warn("query user like detail fail", zap.Error(err))
		}
		isUserLiker = userLikeDetail != nil
	}

	isInBookShelf, ok := shelves[card.BookID]
	if !ok {
		s.feedMetrics().addSourceFallback(model.FeedSourceUserBookshelf)
		userBookshelfDetail, err := queryUserBookshelfDetailWithCache(ctx, s.db, s.userBookshelfCache, s.userBookshelfRepo, viewerID, card.BookID)
		if err != nil {
			// 保持现有行为：书架查询失败跳过该条 Feed
			logger.Warn("query user bookshelf detail fail", zap.Error(err))
			return false, false, false
		}
		isInBookShelf = userBookshelfDetail != nil
	}

	return isUserLiker, isInBookShelf, true
}

func feedResponseFromCard(feedID string, card *model.FeedCard, isUserLiker bool, isInBookShelf bool) *dto.FeedResponse {
	return &dto.FeedResponse{
		FeedID:              feedID,
		BookID:              card.BookID,
		UserID:              card.UserID,
		Content:             card.Content,
		FeedCreatedAt:       dto.UnixTime(card.FeedCreatedAt),
		FeedUpdatedAt:       dto.UnixTime(card.FeedUpdatedAt),
		BookName:            card.BookName,
		BookDescription:     card.BookDescription,
		BookCoverUrl:        card.BookCoverURL,
		BookVIPLevel:        card.BookVIPLevel,
		BookPrice:           card.BookPrice,
		BookBrowseAmount:    card.BookBrowseAmount,
		BookBookshelfAmount: card.BookBookshelfAmount,
		BookPushAt:          dto.UnixTime(card.BookPushedAt),
		BookCreatedAt:       dto.UnixTime(card.BookCreatedAt),
		BookUpdatedAt:       dto.UnixTime(card.BookUpdatedAt),
		UserProfileUrl:      card.AuthorProfileURL,
		UserFans:            card.AuthorFansNum,
		UserVIPLevel:        card.AuthorVIPLevel,
		IsInBookShelf:       isInBookShelf,
		IsUserLiker:         isUserLiker,
	}
}

func guestFeedResponseFromCard(feedID string, card *model.FeedCard) *dto.GuestFeedResponse {
	return &dto.GuestFeedResponse{
		FeedID:              feedID,
		BookID:              card.BookID,
		UserID:              card.UserID,
		Content:             card.Content,
		FeedCreatedAt:       dto.UnixTime(card.FeedCreatedAt),
		FeedUpdatedAt:       dto.UnixTime(card.FeedUpdatedAt),
		BookName:            card.BookName,
		BookDescription:     card.BookDescription,
		BookCoverUrl:        card.BookCoverURL,
		BookVIPLevel:        card.BookVIPLevel,
		BookPrice:           card.BookPrice,
		BookBrowseAmount:    card.BookBrowseAmount,
		BookBookshelfAmount: card.BookBookshelfAmount,
		BookPushAt:          dto.UnixTime(card.BookPushedAt),
		BookCreatedAt:       dto.UnixTime(card.BookCreatedAt),
		BookUpdatedAt:       dto.UnixTime(card.BookUpdatedAt),
		UserProfileUrl:      card.AuthorProfileURL,
		UserFans:            card.AuthorFansNum,
		UserVIPLevel:        card.AuthorVIPLevel,
	}
}
