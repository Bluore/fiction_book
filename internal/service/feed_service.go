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
	GuestGetUserFeed(ctx context.Context, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error)
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
	userLikeCache      cache.UserLikeCache
	userInfoCache      cache.UserInfoCache
	userBookshelfCache cache.UserBookshelfCache
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
		userLikeCache:      userLikeCache,
		userInfoCache:      userInfoCache,
		userBookshelfCache: userBookshelfCache,
		db:                 db,
		cfg:                cfg,
	}
}

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

	var feedListResponse = make([]*dto.FeedResponse, 0, len(feedIDList))
	// 组装 Feed 信息
	for _, feedID := range feedIDList {
		feedDetail, err := queryUserFeedDetailWithCache(ctx, s.db, s.feedCache, s.userFeedRepo, feedID)
		if err != nil {
			logger.Warn("query user feed detail fail", zap.Error(err))
			continue
		}
		if feedDetail == nil {
			continue
		}

		bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, feedDetail.BookID)
		if err != nil {
			logger.Warn("query book detail fail", zap.Error(err))
			continue
		}
		if bookDetail == nil {
			continue
		}

		userDetail, err := queryUserDetailWithCache(ctx, s.db, s.userCache, s.userRepo, feedDetail.UserID)
		if err != nil {
			logger.Warn("query user detail fail", zap.Error(err))
			continue
		}
		if userDetail == nil {
			continue
		}

		userAssetDetail, err := queryUserAssetDetailWithCache(ctx, s.db, s.userAssetCache, s.userAssetRepo, feedDetail.UserID)
		if err != nil {
			logger.Warn("query user asset detail fail", zap.Error(err))
			continue
		}
		if userAssetDetail == nil {
			continue
		}

		userInfoDetail, err := queryUserInfoDetailWithCache(ctx, s.db, s.userInfoCache, s.userInfoRepo, feedDetail.UserID)
		if err != nil {
			logger.Warn("query user info detail fail", zap.Error(err))
			continue
		}
		if userInfoDetail == nil {
			continue
		}

		userLikeDetail, err := queryUserLikeDetailWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, userID, feedDetail.UserID)
		if err != nil {
			logger.Warn("query user like detail fail", zap.Error(err))
		}
		var isUserLiker = true
		if userLikeDetail == nil {
			isUserLiker = false
		}

		userBookshelfDetail, err := queryUserBookshelfDetailWithCache(ctx, s.db, s.userBookshelfCache, s.userBookshelfRepo, userID, feedDetail.BookID)
		if err != nil {
			logger.Warn("query user bookshelf detail fail", zap.Error(err))
			continue
		}
		var isInBookshelf = true
		if userBookshelfDetail == nil {
			isInBookshelf = false
		}

		feedDto := &dto.FeedResponse{
			FeedID:              feedID,
			BookID:              feedDetail.BookID,
			UserID:              feedDetail.UserID,
			Content:             feedDetail.Content,
			FeedCreatedAt:       dto.UnixTime(feedDetail.CreatedAt),
			FeedUpdatedAt:       dto.UnixTime(feedDetail.UpdatedAt),
			BookName:            bookDetail.Name,
			BookDescription:     bookDetail.Description,
			BookCoverUrl:        bookDetail.CoverImage,
			BookVIPLevel:        bookDetail.VIPLevel,
			BookPrice:           bookDetail.Price,
			BookBrowseAmount:    bookDetail.BrowseAmount,
			BookBookshelfAmount: bookDetail.BookshelfAmount,
			BookPushAt:          dto.UnixTime(bookDetail.PushedAt),
			BookCreatedAt:       dto.UnixTime(bookDetail.CreatedAt),
			BookUpdatedAt:       dto.UnixTime(bookDetail.UpdatedAt),
			UserProfileUrl:      userDetail.ProfileImage,
			UserFans:            userInfoDetail.FansNum,
			UserVIPLevel:        userAssetDetail.VIPLevel,
			IsInBookShelf:       isInBookshelf,
			IsUserLiker:         isUserLiker,
		}

		feedListResponse = append(feedListResponse, feedDto)
	}

	userFeedResponse := &dto.GetUserFeedResponseV2{
		Feeds:        feedListResponse,
		NextCursorID: nextCursor.nextCursorID,
		NextOffset:   nextCursor.nextOffset,
	}
	return userFeedResponse, nil
}

func (s *feedServiceV2) GuestGetUserFeed(ctx context.Context, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error) {
	//TODO implement me
	panic("implement me")
}
