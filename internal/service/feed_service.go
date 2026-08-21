package service

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"time"

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
	bookRepo          repository.BookRepository
	bookTagRepo       repository.BookTagRepository
	bookESRepo        repository.BookESRepository
	userRepo          repository.UserRepository
	userAssetRepo     repository.UserAssetRepository
	userLikeRepo      repository.UserLikeRepository
	purchasedBookRepo repository.UserPurchasedBookRepository
	feedRepo          repository.UserFeedRepository
	outboxRepo        repository.OutboxRepository
	bookCache         cache.BookCache
	userStatCache     cache.UserStatCache
	bookThermalCache  cache.BookThermalCache
	bookTagCache      cache.BookTagCache
	feedCache         cache.UserFeedCache
	userLikeCache     cache.UserLikeCache
	db                *gorm.DB
	cfg               *config.Config
}

func (s *feedServiceV2) GetUserFeed(ctx context.Context, userID string, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error) {
	nextCursor, feedIDList, err := s.queryFeedIDListWithCache(ctx, userID, req.LastCursorID, req.TargetCursorID, req.Offset)
	if err != nil {
		return nil, err
	}
	if nextCursor == nil || len(feedIDList) == 0 {
		return &dto.GetUserFeedResponseV2{
			Feeds:        make([]*dto.FeedResponse, 0),
			NextCursorID: model.DefaultFeedCursor,
			NextOffset:   0,
		}, nil
	}

	//TODO implement me
	panic("implement me")
}

func (s *feedServiceV2) GuestGetUserFeed(ctx context.Context, req *dto.GetUserFeedRequestV2) (*dto.GetUserFeedResponseV2, error) {
	//TODO implement me
	panic("implement me")
}
