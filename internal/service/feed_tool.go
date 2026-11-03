package service

import (
	"context"
	"fmt"
	"slices"
	"sort"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/logger"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type nextQueryCursor struct {
	nextCursorID string
	nextOffset   int
}

func (s *feedServiceV2) queryFeedIDListWithCache(
	ctx context.Context,
	userID string,
	lastCursorID string,
	targetCursorID string,
	targetCursorOffset int,
) (*nextQueryCursor, []string, error) {
	if feedIDList, cacheHit, err := s.feedCache.GetUserFeedFeedIDs(ctx, userID, targetCursorID, targetCursorOffset, 10); err != nil {
		logger.Warn("get user feed id failed", zap.Error(err))
		return nil, make([]string, 0), err
	} else if cacheHit {
		if len(feedIDList) != 0 {
			// FOCUS 命中直接返回
			return &nextQueryCursor{
				nextCursorID: targetCursorID,
				nextOffset:   targetCursorOffset + len(feedIDList),
			}, feedIDList, nil
		}

		// FOCUS 如果表空了向下寻找
		curCursor, cacheHit, err := s.feedCache.GetUserFeedCursor(ctx, userID, targetCursorID)
		if err != nil {
			logger.Warn("get user feed cursor failed", zap.Error(err))
			return nil, make([]string, 0), err
		}
		if cacheHit {
			nextFeedIDList, cacheHit, err := s.feedCache.GetUserFeedFeedIDs(ctx, userID, curCursor.NextCursorID, 0, 10)
			if err != nil {
				logger.Warn("get user feed id failed", zap.Error(err))
				return nil, make([]string, 0), err
			} else if cacheHit {
				if len(nextFeedIDList) == 0 {
					return &nextQueryCursor{
						nextCursorID: model.DefaultFeedCursor,
						nextOffset:   0,
					}, make([]string, 0), nil
				}

				return &nextQueryCursor{
					nextCursorID: curCursor.NextCursorID,
					nextOffset:   10,
				}, nextFeedIDList, nil
			}

			// FOCUS 向下新建一个Cursor
			_, nextFeedIDList, err = s.refreshFeedIDListInCache(ctx, userID, curCursor.LastTime, curCursor.LastID, curCursor.NextCursorID)
			if err != nil {
				logger.Warn("get user feed id failed", zap.Error(err))
				return nil, make([]string, 0), err
			}
			if len(nextFeedIDList) == 0 {
				return &nextQueryCursor{
					nextCursorID: model.DefaultFeedCursor,
					nextOffset:   0,
				}, make([]string, 0), nil
			}

			return &nextQueryCursor{
				nextCursorID: curCursor.NextCursorID,
				nextOffset:   10,
			}, nextFeedIDList, nil
		}

		// FOCUS 缓存未命中直接下一步
	}

	// FOCUS 如果是起始节点，直接刷新当前节点
	if targetCursorID == model.DefaultFeedCursor {
		_, feedIDList, err := s.refreshFeedIDListInCache(ctx, userID, time.Now(), "", model.DefaultFeedCursor)
		if err != nil {
			logger.Error("get user feed id list failed", zap.Error(err))
			return nil, make([]string, 0), err
		}
		if len(feedIDList) == 0 {
			// FOCUS 没有任何的 Feed 流
			return nil, make([]string, 0), nil
		}
		return &nextQueryCursor{
			nextCursorID: model.DefaultFeedCursor,
			nextOffset:   len(feedIDList),
		}, feedIDList, err
	}

	// FOCUS 非起始节点先查找上一个节点
	if lastFeedCursorDetail, cacheHit, err := s.feedCache.GetUserFeedCursor(ctx, userID, lastCursorID); err != nil {
		logger.Warn("get user feed cursor failed", zap.Error(err))
		return nil, make([]string, 0), err
	} else if cacheHit && lastFeedCursorDetail != nil && lastFeedCursorDetail.NextCursorID == lastCursorID {
		_, feedIDList, err := s.refreshFeedIDListInCache(ctx, userID, lastFeedCursorDetail.LastTime, lastFeedCursorDetail.LastID, targetCursorID)
		return &nextQueryCursor{
			nextCursorID: targetCursorID,
			nextOffset:   len(feedIDList),
		}, feedIDList, err
	}

	// FOCUS 如果寻找父节点失败，返回根节点
	_, feedIDList, err := s.refreshFeedIDListInCache(ctx, userID, time.Now(), "", model.DefaultFeedCursor)
	if err != nil {
		logger.Error("get user feed id list failed", zap.Error(err))
		return nil, make([]string, 0), err
	}
	if len(feedIDList) == 0 {
		// FOCUS 没有任何的 Feed 流
		return nil, make([]string, 0), nil
	}
	return &nextQueryCursor{
		nextCursorID: model.DefaultFeedCursor,
		nextOffset:   len(feedIDList),
	}, feedIDList, err
}

func (s *feedServiceV2) refreshFeedIDListInCache(
	ctx context.Context,
	userID string,
	lastFeedTime time.Time,
	lastFeedID string,
	targetCursorID string,
) (*model.UserFeedCursor, []string, error) {
	// FOCUS 获取用户关注列表，拆分为大V/小V
	bigVUserIDList, smallVUserIDList, err := queryUserLikeIDListWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, userID)
	if err != nil {
		logger.Error("query user like detail failed", zap.Error(err))
		return nil, make([]string, 0), err
	}
	if len(bigVUserIDList) == 0 && len(smallVUserIDList) == 0 {
		return nil, make([]string, 0), nil
	}

	// FOCUS 每次增加搜索范围，直到获取到100个Feed或者没有新的Feed
	baseTime := lastFeedTime
	selectStepDuration := time.Hour * 24
	selectStartTime := baseTime

	var feedDetailList = make([]*model.UserFeed, 0, 100)
GetFeedIDList:
	for len(feedDetailList) < 100 {
		if selectStartTime.Before(MINTime) {
			// FOCUS 超出时间范围进入下一步，防止死循环
			if len(feedDetailList) != 0 {
				break GetFeedIDList
			}
			return nil, make([]string, 0), nil
		}

		// FOCUS 获取读推送的Feed（大V不写推送需拉取，小V已推送至收件箱）
		for _, targetUserID := range bigVUserIDList {

			queryFeedDetailList, err := s.userFeedRepo.GetFeedListByUserIDWithStartCursorAndEndCursor(
				ctx,
				s.db,
				targetUserID,
				selectStartTime,
				"",
				selectStartTime.Add(-selectStepDuration),
			)
			if err != nil {
				logger.Warn("get feed by id failed", zap.Error(err))
				return nil, make([]string, 0), err
			}

			if len(queryFeedDetailList) == 0 {
				continue
			}

			for _, feedDetail := range queryFeedDetailList {
				feedDetailList = append(feedDetailList, feedDetail)
			}
		}

		// FOCUS 获取写推送的Feed
		if queryFeedIDList, err := s.userInboxRepo.GetUserInboxIDListByUserIDWithStartCursorAndEndCursor(
			ctx,
			s.db,
			userID,
			selectStartTime,
			"",
			selectStartTime.Add(-selectStepDuration),
		); err != nil {
			logger.Warn("get user inbox by id failed", zap.Error(err))
			return nil, make([]string, 0), err
		} else if len(queryFeedIDList) > 0 {
			for _, feedID := range queryFeedIDList {
				feedDetail, err := queryUserFeedDetailWithCache(ctx, s.db, s.feedCache, s.userFeedRepo, feedID)
				if err != nil {
					logger.Warn("get user feed detail failed", zap.Error(err))
					continue
				}
				if feedDetail == nil {
					continue
				}

				feedDetailList = append(feedDetailList, feedDetail)
			}
		}

		selectStartTime = selectStartTime.Add(-selectStepDuration)
		selectStepDuration *= 2
	}
	if len(feedDetailList) > 100 {
		feedDetailList = feedDetailList[:100]
	}

	// FOCUS 排序ID结果
	sort.Slice(feedDetailList, func(i, j int) bool {
		itemA, itemB := feedDetailList[i], feedDetailList[j]
		if itemA.Rank != itemB.Rank {
			return itemA.Rank > itemB.Rank
		}
		if !itemA.CreatedAt.Equal(itemB.CreatedAt) {
			return itemA.CreatedAt.After(itemB.CreatedAt)
		}
		return itemA.FeedID > itemB.FeedID
	})

	var nextLastTime time.Time = time.Now()
	var nextLastFeedID string = ""
	var feedIDList []string = make([]string, 0, len(feedDetailList))

	for _, feedDetail := range feedDetailList {

		backFeedTime := feedDetail.CreatedAt
		backFeedID := feedDetail.FeedID

		if backFeedTime.Before(nextLastTime) || (backFeedTime.Equal(nextLastTime) && backFeedID < nextLastFeedID) {
			nextLastTime = backFeedTime
			nextLastFeedID = backFeedID
		}

		feedIDList = append(feedIDList, feedDetail.FeedID)
	}

	// FOCUS 更新数据
	err = s.feedCache.UpdateUserFeedFeedIDs(ctx, userID, targetCursorID, feedIDList)
	if err != nil {
		logger.Warn("update user feed id failed", zap.Error(err))
		return nil, make([]string, 0), err
	}

	targetCursor := &model.UserFeedCursor{
		CursorID:     targetCursorID,
		NextCursorID: uuid.New().String(),
		LastTime:     nextLastTime,
		LastID:       nextLastFeedID,
	}

	err = s.feedCache.UpdateUserFeedCursor(ctx, userID, targetCursorID, targetCursor)
	if err != nil {
		logger.Warn("update user feed cursor failed", zap.Error(err))
		return nil, make([]string, 0), err
	}

	feedIDListAfterCut := feedIDList
	if len(feedIDListAfterCut) > 10 {
		feedIDListAfterCut = feedIDListAfterCut[:10]
	}
	return targetCursor, feedIDListAfterCut, nil
}

// guestFeedCacheUserID 游客 Feed 复用 UserFeedCache 时的固定键，真实用户 ID 为 UUID，不会冲突
const guestFeedCacheUserID = "guest"

// 游客 Feed 每个游标节点缓存的Feed数量与单页返回数量
const (
	guestFeedListBatchSize = 100
	guestFeedListPageSize  = 10
)

// queryGuestFeedIDListWithCache 获取游客 Feed 的FeedID列表（全局 Feed 按创建时间倒序）
// 游标机制与 queryFeedIDListWithCache 保持一致，仅数据源不同：全局 Feed 表
func (s *feedServiceV2) queryGuestFeedIDListWithCache(
	ctx context.Context,
	lastCursorID string,
	targetCursorID string,
	targetCursorOffset int,
) (*nextQueryCursor, []string, error) {
	if feedIDList, cacheHit, err := s.feedCache.GetUserFeedFeedIDs(ctx, guestFeedCacheUserID, targetCursorID, targetCursorOffset, guestFeedListPageSize); err != nil {
		logger.Warn("get guest feed id failed", zap.Error(err))
		return nil, make([]string, 0), err
	} else if cacheHit {
		if len(feedIDList) != 0 {
			// FOCUS 命中直接返回
			return &nextQueryCursor{
				nextCursorID: targetCursorID,
				nextOffset:   targetCursorOffset + len(feedIDList),
			}, feedIDList, nil
		}

		// FOCUS 当前节点列表已空，向下寻找下一个节点
		curCursor, cacheHit, err := s.feedCache.GetUserFeedCursor(ctx, guestFeedCacheUserID, targetCursorID)
		if err != nil {
			logger.Warn("get guest feed cursor failed", zap.Error(err))
			return nil, make([]string, 0), err
		}
		if cacheHit {
			nextFeedIDList, cacheHit, err := s.feedCache.GetUserFeedFeedIDs(ctx, guestFeedCacheUserID, curCursor.NextCursorID, 0, guestFeedListPageSize)
			if err != nil {
				logger.Warn("get guest feed id failed", zap.Error(err))
				return nil, make([]string, 0), err
			} else if cacheHit {
				if len(nextFeedIDList) == 0 {
					return &nextQueryCursor{
						nextCursorID: model.DefaultFeedCursor,
						nextOffset:   0,
					}, make([]string, 0), nil
				}

				return &nextQueryCursor{
					nextCursorID: curCursor.NextCursorID,
					nextOffset:   guestFeedListPageSize,
				}, nextFeedIDList, nil
			}

			// FOCUS 下一个节点尚未建立，向下新建一个 Cursor
			_, nextFeedIDList, err = s.refreshGuestFeedIDListInCache(ctx, curCursor.LastTime, curCursor.LastID, curCursor.NextCursorID)
			if err != nil {
				logger.Warn("get guest feed id failed", zap.Error(err))
				return nil, make([]string, 0), err
			}
			if len(nextFeedIDList) == 0 {
				return &nextQueryCursor{
					nextCursorID: model.DefaultFeedCursor,
					nextOffset:   0,
				}, make([]string, 0), nil
			}

			return &nextQueryCursor{
				nextCursorID: curCursor.NextCursorID,
				nextOffset:   guestFeedListPageSize,
			}, nextFeedIDList, nil
		}

		// FOCUS 缓存未命中直接下一步
	}

	// FOCUS 如果是起始节点，直接刷新当前节点
	if targetCursorID == model.DefaultFeedCursor {
		_, feedIDList, err := s.refreshGuestFeedIDListInCache(ctx, time.Time{}, "", model.DefaultFeedCursor)
		if err != nil {
			logger.Error("get guest feed id list failed", zap.Error(err))
			return nil, make([]string, 0), err
		}
		if len(feedIDList) == 0 {
			// FOCUS 没有任何 Feed
			return nil, make([]string, 0), nil
		}
		return &nextQueryCursor{
			nextCursorID: model.DefaultFeedCursor,
			nextOffset:   len(feedIDList),
		}, feedIDList, nil
	}

	// FOCUS 非起始节点先查找上一个节点，确认目标节点是链上的下一个节点
	if lastFeedCursorDetail, cacheHit, err := s.feedCache.GetUserFeedCursor(ctx, guestFeedCacheUserID, lastCursorID); err != nil {
		logger.Warn("get guest feed cursor failed", zap.Error(err))
		return nil, make([]string, 0), err
	} else if cacheHit && lastFeedCursorDetail != nil && lastFeedCursorDetail.NextCursorID == targetCursorID {
		_, feedIDList, err := s.refreshGuestFeedIDListInCache(ctx, lastFeedCursorDetail.LastTime, lastFeedCursorDetail.LastID, targetCursorID)
		return &nextQueryCursor{
			nextCursorID: targetCursorID,
			nextOffset:   len(feedIDList),
		}, feedIDList, err
	}

	// FOCUS 如果寻找父节点失败，返回根节点
	_, feedIDList, err := s.refreshGuestFeedIDListInCache(ctx, time.Time{}, "", model.DefaultFeedCursor)
	if err != nil {
		logger.Error("get guest feed id list failed", zap.Error(err))
		return nil, make([]string, 0), err
	}
	if len(feedIDList) == 0 {
		// FOCUS 没有任何 Feed
		return nil, make([]string, 0), nil
	}
	return &nextQueryCursor{
		nextCursorID: model.DefaultFeedCursor,
		nextOffset:   len(feedIDList),
	}, feedIDList, nil
}

// refreshGuestFeedIDListInCache 按创建时间倒序拉取一批全局 FeedID 并缓存到指定游标节点
// 尾部 Feed 的 (创建时间, FeedID) 作为下一个游标节点的起点，保证分页不重不漏
func (s *feedServiceV2) refreshGuestFeedIDListInCache(
	ctx context.Context,
	lastFeedTime time.Time,
	lastFeedID string,
	targetCursorID string,
) (*model.UserFeedCursor, []string, error) {
	feedDetailList, err := s.userFeedRepo.GetFeedListWithStartCursorAndLimit(ctx, s.db, lastFeedTime, lastFeedID, guestFeedListBatchSize)
	if err != nil {
		logger.Error("query guest feed list failed", zap.Error(err))
		return nil, make([]string, 0), err
	}
	if len(feedDetailList) == 0 {
		return nil, make([]string, 0), nil
	}

	feedIDList := make([]string, 0, len(feedDetailList))
	for _, feedDetail := range feedDetailList {
		feedIDList = append(feedIDList, feedDetail.FeedID)
	}

	// FOCUS 尾部的 (创建时间, FeedID) 作为下一页的游标起点
	lastFeed := feedDetailList[len(feedDetailList)-1]

	err = s.feedCache.UpdateUserFeedFeedIDs(ctx, guestFeedCacheUserID, targetCursorID, feedIDList)
	if err != nil {
		logger.Warn("update guest feed id failed", zap.Error(err))
		return nil, make([]string, 0), err
	}

	targetCursor := &model.UserFeedCursor{
		CursorID:     targetCursorID,
		NextCursorID: uuid.New().String(),
		LastTime:     lastFeed.CreatedAt,
		LastID:       lastFeed.FeedID,
	}

	err = s.feedCache.UpdateUserFeedCursor(ctx, guestFeedCacheUserID, targetCursorID, targetCursor)
	if err != nil {
		logger.Warn("update guest feed cursor failed", zap.Error(err))
		return nil, make([]string, 0), err
	}

	feedIDListAfterCut := feedIDList
	if len(feedIDListAfterCut) > guestFeedListPageSize {
		feedIDListAfterCut = feedIDListAfterCut[:guestFeedListPageSize]
	}
	return targetCursor, feedIDListAfterCut, nil
}

func queryUserFeedDetailWithCache(
	ctx context.Context,
	db *gorm.DB,
	feedCache cache.UserFeedCache,
	userFeedRepo repository.UserFeedRepository,
	feedID string,
) (*model.UserFeed, error) {
	feedDetail, cacheHit, err := feedCache.GetFeedDetail(ctx, feedID)
	if err != nil {
		logger.Warn("get user_feed cache fail", zap.Error(err))
	}
	if cacheHit {
		return feedDetail, nil
	}

	token, locked, err := feedCache.LockFeedDetail(ctx, feedID)
	if err != nil {
		logger.Warn("lock user_feed cache fail", zap.Error(err))
		return userFeedRepo.GetFeedByFeedID(ctx, db, feedID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			feedDetail, cacheHit, _ = feedCache.GetFeedDetail(ctx, feedID)
			if cacheHit {
				return feedDetail, nil
			}
		}
		return nil, fmt.Errorf("user_feed cache query too many fail")
	}
	defer func() {
		if err := feedCache.UnlockFeedDetail(ctx, feedID, token); err != nil {
			logger.Error("unlock user_feed cache fail", zap.Error(err))
		}
	}()

	feedDetail, err = userFeedRepo.GetFeedByFeedID(ctx, db, feedID)
	if err != nil {
		return nil, err
	}
	if err := feedCache.UpdateFeedDetail(ctx, feedID, feedDetail); err != nil {
		logger.Warn("update user_feed cache fail", zap.Error(err))
	}
	return feedDetail, nil
}

// loadFeedCards 批量读取公共卡片；命中直接复用，miss 时按原有单条 helper 组装。
// 返回可用的卡片 map，以及本轮新建、需要写回的完整卡片。
func (s *feedServiceV2) loadFeedCards(ctx context.Context, feedIDs []string) (map[string]*model.FeedCard, []*model.FeedCard) {
	cards := make(map[string]*model.FeedCard, len(feedIDs))
	if len(feedIDs) == 0 {
		return cards, nil
	}

	var cached map[string]*model.FeedCard
	var missed []string
	if s.feedComposeCache == nil {
		missed = feedIDs
	} else {
		var err error
		cached, missed, err = s.feedComposeCache.GetCards(ctx, feedIDs)
		if err != nil {
			logger.Warn("get feed cards cache fail, fallback to per-feed query", zap.Error(err))
			cached = nil
			missed = feedIDs
		}
	}

	metrics := s.feedMetrics()
	metrics.cardHit.Add(int64(len(cached)))
	metrics.cardMiss.Add(int64(len(missed)))
	metrics.logRequest(len(feedIDs), len(cached), len(missed))

	toWrite := make([]*model.FeedCard, 0, len(missed))
	for _, feedID := range feedIDs {
		if card, ok := cached[feedID]; ok {
			cards[feedID] = card
			continue
		}
		if !slices.Contains(missed, feedID) {
			// EMPTY：确定不存在，跳过该条 Feed
			continue
		}

		card := s.buildCard(ctx, feedID)
		if card == nil {
			continue
		}
		cards[feedID] = card
		toWrite = append(toWrite, card)
	}
	return cards, toWrite
}

// buildCard 按原有单条 helper 顺序组装公共卡片；任一必需实体缺失时返回 nil。
func (s *feedServiceV2) buildCard(ctx context.Context, feedID string) *model.FeedCard {
	metrics := s.feedMetrics()

	metrics.addSourceFallback(model.FeedSourceFeed)
	feedDetail, err := queryUserFeedDetailWithCache(ctx, s.db, s.feedCache, s.userFeedRepo, feedID)
	if err != nil {
		logger.Warn("query user feed detail fail", zap.Error(err))
		return nil
	}
	if feedDetail == nil {
		return nil
	}

	metrics.addSourceFallback(model.FeedSourceBook)
	bookDetail, err := queryBookDetailWithCache(ctx, s.db, s.bookCache, s.bookRepo, feedDetail.BookID)
	if err != nil {
		logger.Warn("query book detail fail", zap.Error(err))
		return nil
	}
	if bookDetail == nil {
		return nil
	}

	metrics.addSourceFallback(model.FeedSourceUser)
	userDetail, err := queryUserDetailWithCache(ctx, s.db, s.userCache, s.userRepo, feedDetail.UserID)
	if err != nil {
		logger.Warn("query user detail fail", zap.Error(err))
		return nil
	}
	if userDetail == nil {
		return nil
	}

	metrics.addSourceFallback(model.FeedSourceUserAsset)
	userAssetDetail, err := queryUserAssetDetailWithCache(ctx, s.db, s.userAssetCache, s.userAssetRepo, feedDetail.UserID)
	if err != nil {
		logger.Warn("query user asset detail fail", zap.Error(err))
		return nil
	}
	if userAssetDetail == nil {
		return nil
	}

	metrics.addSourceFallback(model.FeedSourceUserInfo)
	userInfoDetail, err := queryUserInfoDetailWithCache(ctx, s.db, s.userInfoCache, s.userInfoRepo, feedDetail.UserID)
	if err != nil {
		logger.Warn("query user info detail fail", zap.Error(err))
		return nil
	}
	if userInfoDetail == nil {
		return nil
	}

	return &model.FeedCard{
		FeedID:              feedDetail.FeedID,
		UserID:              feedDetail.UserID,
		BookID:              feedDetail.BookID,
		Content:             feedDetail.Content,
		FeedCreatedAt:       feedDetail.CreatedAt,
		FeedUpdatedAt:       feedDetail.UpdatedAt,
		BookName:            bookDetail.Name,
		BookDescription:     bookDetail.Description,
		BookCoverURL:        bookDetail.CoverImage,
		BookVIPLevel:        bookDetail.VIPLevel,
		BookPrice:           bookDetail.Price,
		BookBrowseAmount:    bookDetail.BrowseAmount,
		BookBookshelfAmount: bookDetail.BookshelfAmount,
		BookPushedAt:        bookDetail.PushedAt,
		BookCreatedAt:       bookDetail.CreatedAt,
		BookUpdatedAt:       bookDetail.UpdatedAt,
		AuthorProfileURL:    userDetail.ProfileImage,
		AuthorFansNum:       userInfoDetail.FansNum,
		AuthorVIPLevel:      userAssetDetail.VIPLevel,
	}
}

// loadViewerStates 收集当前页卡片涉及的 author/book ID，用一条 pipeline 读取点赞和书架状态。
// 返回 nil map 表示批量读取失败，调用方按单条 helper 回源。
func (s *feedServiceV2) loadViewerStates(ctx context.Context, viewerID string, cards map[string]*model.FeedCard) (map[string]bool, map[string]bool) {
	if s.feedComposeCache == nil || len(cards) == 0 {
		return nil, nil
	}

	authorIDs := make([]string, 0, len(cards))
	bookIDs := make([]string, 0, len(cards))
	seenAuthors := make(map[string]struct{}, len(cards))
	seenBooks := make(map[string]struct{}, len(cards))
	for _, card := range cards {
		if card == nil {
			continue
		}
		if _, ok := seenAuthors[card.UserID]; !ok && card.UserID != "" {
			seenAuthors[card.UserID] = struct{}{}
			authorIDs = append(authorIDs, card.UserID)
		}
		if _, ok := seenBooks[card.BookID]; !ok && card.BookID != "" {
			seenBooks[card.BookID] = struct{}{}
			bookIDs = append(bookIDs, card.BookID)
		}
	}

	likes, shelves, err := s.feedComposeCache.GetViewerStates(ctx, viewerID, authorIDs, bookIDs)
	if err != nil {
		logger.Warn("get feed viewer states cache fail, fallback to per-feed query", zap.Error(err))
		return nil, nil
	}
	return likes, shelves
}

// writeBackCards 在响应组装完成后写回本轮新建的完整卡片，写失败只记录日志。
func (s *feedServiceV2) writeBackCards(ctx context.Context, cards []*model.FeedCard) {
	if len(cards) == 0 || s.feedComposeCache == nil {
		return
	}
	if err := s.feedComposeCache.UpdateCards(ctx, cards); err != nil {
		logger.Warn("update feed cards cache fail", zap.Error(err))
	}
}
