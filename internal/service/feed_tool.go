package service

import (
	"context"
	"fmt"
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
	var targetUserIDList []string
	var err error

	// FOCUS 获取用户关注列表
	targetUserIDList, err = queryUserLikeIDListWithCache(ctx, s.db, s.userLikeCache, s.userLikeRepo, userID)
	if err != nil {
		logger.Error("query user like detail failed", zap.Error(err))
		return nil, make([]string, 0), err
	}
	if len(targetUserIDList) == 0 {
		return nil, make([]string, 0), nil
	}

	// FOCUS 每次增加搜索范围，直到获取到100个Feed或者没有新的Feed
	baseTime := lastFeedTime
	findDeepDuration := time.Hour * 24 * 30
	lastFindTime := baseTime
	lastFindID := ""

	var feedDetailList = make([]*model.UserFeed, 0, 100)
GetFeedIDList:
	for len(feedDetailList) < 100 {
		if lastFindTime.Before(MINTime) {
			// FOCUS 超出时间范围进入下一步，防止死循环
			if len(feedDetailList) != 0 {
				break GetFeedIDList
			}
			return nil, make([]string, 0), nil
		}

		// FOCUS 获取读推送的Feed
		for _, targetUserID := range targetUserIDList {

			queryFeedDetailList, err := s.userFeedRepo.GetFeedListByUserIDWithStartCursorAndEndCursor(
				ctx,
				s.db,
				targetUserID,
				lastFindTime,
				lastFindID,
				lastFindTime.Add(-findDeepDuration),
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
			lastFeedTime,
			lastFeedID,
			lastFindTime,
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
				feedDetailList = append(feedDetailList, feedDetail)
			}
		}

		findDeepDuration *= 2
		lastFindTime = lastFindTime.Add(-findDeepDuration)
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
