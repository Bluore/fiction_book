package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"fiction-book/internal/cache"
	"fiction-book/internal/model"
)

func TestFeedService_GetFeedList_UsesCachedCursorAndFixedLimit(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	userID := "u1"
	cursorTime := time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC)
	cursorFeedID := "f2"

	feedRepo := new(MockFeedRepository)
	feedCache := new(MockFeedCache)
	svc := NewFeedService(feedRepo, feedCache, db)

	feeds := []*model.FeedInbox{
		newFeedInbox(userID, "f3", cursorTime.Add(-time.Minute)),
		newFeedInbox(userID, "f4", cursorTime.Add(-2*time.Minute)),
	}
	cachedCursor := &model.FeedCursor{
		UserID:     userID,
		LastReadAt: cursorTime,
		LastFeedID: cursorFeedID,
	}

	feedCache.On("GetCursor", ctx, userID).Return(cachedCursor, true, nil).Twice()
	feedCache.On("GetFeeds", ctx, userID, cache.FeedListCursor{
		LastFeedID:    cursorFeedID,
		LastCreatedAt: cursorTime,
		Limit:         int64(feedListPageSize + 1),
	}).Return([]*model.FeedInbox(nil), false, nil).Once()
	feedRepo.On(
		"GetUserFeeds",
		ctx,
		db,
		mock.MatchedBy(func(filter model.FeedFilterWithTimePage) bool {
			return filter.UserID == userID &&
				filter.LastID != nil &&
				*filter.LastID == cursorFeedID &&
				filter.LastTime.Equal(cursorTime) &&
				filter.Limit == feedListPageSize+1
		}),
	).Return(feeds, nil).Once()
	feedCache.On("WarmFeeds", ctx, userID, mock.Anything).Return(nil)
	feedRepo.On(
		"MarkAsRead",
		ctx,
		db,
		userID,
		[]string{"f3", "f4"},
	).Return(int64(2), nil).Once()
	feedCache.On("SetCursor", ctx, userID, mock.MatchedBy(func(cursor *model.FeedCursor) bool {
		return cursor.LastFeedID == "f4" &&
			cursor.LastReadAt.Equal(feeds[1].CreatedAt)
	})).Return(nil).Once()

	res, err := svc.GetFeedList(ctx, userID)

	require.NoError(t, err)
	require.Len(t, res.Feeds, 2)
	require.False(t, res.HasMore)
	feedRepo.AssertExpectations(t)
	feedCache.AssertExpectations(t)
}

func TestFeedService_GetFeedList_UsesEmptyCursorOnCacheMiss(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	userID := "u1"

	feedRepo := new(MockFeedRepository)
	feedCache := new(MockFeedCache)
	svc := NewFeedService(feedRepo, feedCache, db)

	feed := newFeedInbox(userID, "f1", time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC))

	feedCache.On("GetCursor", ctx, userID).Return((*model.FeedCursor)(nil), false, nil).Once()
	feedCache.On("GetFeeds", ctx, userID, cache.FeedListCursor{
		Limit: int64(feedListPageSize + 1),
	}).Return([]*model.FeedInbox(nil), false, nil).Once()
	feedRepo.On(
		"GetUserFeeds",
		ctx,
		db,
		mock.MatchedBy(func(filter model.FeedFilterWithTimePage) bool {
			return filter.UserID == userID &&
				filter.LastID == nil &&
				filter.LastTime.IsZero() &&
				filter.Limit == feedListPageSize+1
		}),
	).Return([]*model.FeedInbox{feed}, nil).Once()
	feedCache.On("WarmFeeds", ctx, userID, mock.Anything).Return(nil)
	feedCache.On("GetCursor", ctx, userID).Return((*model.FeedCursor)(nil), false, nil).Once()
	feedRepo.On(
		"MarkAsRead",
		ctx,
		db,
		userID,
		[]string{"f1"},
	).Return(int64(1), nil).Once()
	feedCache.On("SetCursor", ctx, userID, mock.MatchedBy(func(cursor *model.FeedCursor) bool {
		return cursor.LastFeedID == "f1" &&
			cursor.LastReadAt.Equal(feed.CreatedAt)
	})).Return(nil).Once()

	res, err := svc.GetFeedList(ctx, userID)

	require.NoError(t, err)
	require.Len(t, res.Feeds, 1)
	require.False(t, res.HasMore)
	feedRepo.AssertExpectations(t)
	feedCache.AssertExpectations(t)
}

func newFeedInbox(userID string, feedID string, createdAt time.Time) *model.FeedInbox {
	return &model.FeedInbox{
		ID:        feedID,
		UserID:    userID,
		EventType: model.FeedEventBookUpdate,
		BookID:    "b1",
		BookName:  "book",
		BookCover: "cover",
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}
