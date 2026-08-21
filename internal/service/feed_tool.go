package service

import (
	"context"
	"fiction-book/internal/model"
	"fiction-book/pkg/logger"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
		targetCursor, cacheHit, err := s.feedCache.GetUserFeedCursor(ctx, userID, targetCursorID)
		if err != nil {
			logger.Warn("get user feed cursor failed", zap.Error(err))
			return nil, make([]string, 0), err
		}
		if cacheHit {
			if nextCursor, feedIDList, err := s.refreshFeedIDListInCache(ctx, userID, targetCursor.LastTime, targetCursor.NextCursorID, targetCursor.CursorID); err != nil {
				logger.Warn("get user feed id list failed", zap.Error(err))
				return nil, make([]string, 0), err
			} else if nextCursor != nil {
				return &nextQueryCursor{
					nextCursorID: nextCursor.CursorID,
					nextOffset:   len(feedIDList),
				}, feedIDList, nil
			} else {
				// FOCUS 如果没有内容了就直接返回空
				return nil, make([]string, 0), nil
			}
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
	var feedIDList []string
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
	findDeepDuration := time.Hour
	lastFindTime := baseTime.Add(-findDeepDuration)

	var nextLastTime time.Time
	var nextLastFeedID string

GetFeedIDList:
	for len(feedIDList) >= 100 {
		var queryFeedIDListResult = make([]string, 100)

		if lastFeedTime.Before(MINTime) {
			// FOCUS 超出时间范围进入下一步，防止死循环
			if len(queryFeedIDListResult) != 0 {
				break GetFeedIDList
			}
			return nil, make([]string, 0), nil
		}

		// FOCUS 获取读推送的Feed
		for _, targetUserID := range targetUserIDList {

			queryFeedDetailList, err := s.feedRepo.GetFeedListByUserIDWithStartCursorAndEndCursor(ctx, s.db, targetUserID, lastFeedTime, lastFeedID, lastFindTime)
			if err != nil {
				logger.Warn("get feed by id failed", zap.Error(err))
				continue
			}

			if len(queryFeedDetailList) == 0 {
				continue
			}

			for _, feedDetail := range queryFeedDetailList {
				queryFeedIDListResult = append(queryFeedIDListResult, feedDetail.FeedID)
			}

			nextLastTime = queryFeedDetailList[len(queryFeedDetailList)-1].CreatedAt
			nextLastFeedID = queryFeedDetailList[len(queryFeedDetailList)-1].FeedID
		}

		// todo 获取写推送的Feed

		findDeepDuration *= 2
		lastFindTime.Add(-findDeepDuration)
	}
	feedIDList = feedIDList[:100]

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

	return targetCursor, feedIDList, nil
}
