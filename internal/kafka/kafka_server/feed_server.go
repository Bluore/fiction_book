package kafka_server

import (
	"context"
	"encoding/json"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	feedConsumerGroupID = "feed_syncer"

	// 缓存抢锁重试参数，与 service 层保持一致
	lockTryTimes = 8
	lockWaitTime = 100 * time.Millisecond
)

// FeedServer Feed 推送服务端接口
type FeedServer interface {
	// FeedConsumerClient 消费 Feed 消息，扇出到关注者收件箱
	FeedConsumerClient(ctx context.Context, cfg *config.Config)
}

type feedServer struct {
	userInboxRepo     repository.UserInboxRepository
	userLikeRepo      repository.UserLikeRepository
	userLikeCache     cache.UserLikeCache
	feedCache         cache.UserFeedCache
	consumeRecordRepo repository.ConsumeRecordRepository
	db                *gorm.DB
}

// NewFeedServer 创建 Feed 推送服务端
func NewFeedServer(
	userInboxRepo repository.UserInboxRepository,
	userLikeRepo repository.UserLikeRepository,
	userLikeCache cache.UserLikeCache,
	feedCache cache.UserFeedCache,
	consumeRecordRepo repository.ConsumeRecordRepository,
	db *gorm.DB,
) FeedServer {
	return &feedServer{
		userInboxRepo:     userInboxRepo,
		userLikeRepo:      userLikeRepo,
		userLikeCache:     userLikeCache,
		feedCache:         feedCache,
		consumeRecordRepo: consumeRecordRepo,
		db:                db,
	}
}

func (s *feedServer) FeedConsumerClient(ctx context.Context, cfg *config.Config) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.Kafka.Addr,
		Topic:    cfg.Kafka.Topic.Feed,
		GroupID:  feedConsumerGroupID,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			if err := s.consumeMessage(ctx, reader); err != nil {
				if errors.Is(err, context.Canceled) {
					logger.Info("exit feed consumer")
					return
				}
				logger.Error(fmt.Sprintf("feed consumer error: %v", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (s *feedServer) consumeMessage(ctx context.Context, reader *kafka.Reader) error {
	message, err := reader.FetchMessage(ctx)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err != nil {
		return err
	}

	if err := s.handleFeedMessage(ctx, message.Value); err != nil {
		return err
	}
	return reader.CommitMessages(ctx, message)
}

func (s *feedServer) handleFeedMessage(ctx context.Context, messageValue []byte) error {
	var message model.UserFeedMassageV2
	if err := json.Unmarshal(messageValue, &message); err != nil {
		return err
	}
	if message.Operator != model.UserFeedMassageOperatorPush {
		// 历史遗留消息（旧版 FeedPushMessage 等）不处理，跳过并提交，避免阻塞分区
		logger.Warn("skip feed massage with invalid operator", zap.String("operator", message.Operator))
		return nil
	}
	if message.ID == "" {
		logger.Warn("skip feed massage with empty id")
		return nil
	}
	if message.UserFeed.FeedID == "" {
		return nil
	}

	if err := s.consumeFeedPushInTx(ctx, &message); err != nil {
		return err
	}
	s.warmUserFeed(ctx, &message.UserFeed)
	return nil
}

// consumeFeedPushInTx 消费推送消息，将 Feed 扇出到作者关注者的收件箱。
// 单条消息处理失败时整个事务回滚，消费端重试，消费幂等由 consume_record 保证。
func (s *feedServer) consumeFeedPushInTx(ctx context.Context, message *model.UserFeedMassageV2) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// FOCUS 幂等标记，防止重复消费
		firstConsume, err := s.consumeRecordRepo.TryMarkConsumed(ctx, tx, message.ID)
		if err != nil {
			return err
		}
		if !firstConsume {
			return nil
		}

		// FOCUS 将 Feed 扇出到关注者收件箱
		feed := &message.UserFeed
		if feed.UserID == "" {
			return nil
		}

		followerIDList, err := s.queryFollowerIDListWithCache(ctx, feed.UserID)
		if err != nil {
			logger.Warn("query follower id list failed", zap.Error(err))
			return err
		}
		if len(followerIDList) == 0 {
			return nil
		}

		userInboxList := make([]*model.UserInbox, 0, len(followerIDList))
		for _, followerID := range followerIDList {
			if followerID == "" {
				continue
			}
			userInboxList = append(userInboxList, &model.UserInbox{
				ID:     uuid.NewString(),
				UserID: followerID,
				FeedID: feed.FeedID,
			})
		}
		if len(userInboxList) == 0 {
			return nil
		}

		if err := s.userInboxRepo.CreateUserInboxBatch(ctx, tx, userInboxList); err != nil {
			return err
		}
		return nil
	})
}

// queryFollowerIDListWithCache 获取关注某用户的全部粉丝，缓存优先
func (s *feedServer) queryFollowerIDListWithCache(ctx context.Context, userID string) ([]string, error) {
	followerIDList, cacheHit, err := s.userLikeCache.GetUserFollowerIDList(ctx, userID)
	if err != nil {
		logger.Warn("get user_follower_id_list cache fail", zap.Error(err))
	}
	if cacheHit {
		return followerIDList, nil
	}

	token, locked, err := s.userLikeCache.LockUserFollowerIDList(ctx, userID)
	if err != nil {
		logger.Warn("lock user_follower_id_list cache fail", zap.Error(err))
		return s.userLikeRepo.GetUserFollowerIDListByUserID(ctx, s.db, userID)
	}
	if !locked {
		for i := 0; i < lockTryTimes; i++ {
			time.Sleep(lockWaitTime)
			followerIDList, cacheHit, _ = s.userLikeCache.GetUserFollowerIDList(ctx, userID)
			if cacheHit {
				return followerIDList, nil
			}
		}
		return make([]string, 0), fmt.Errorf("user_follower_id_list cache query too many fail")
	}
	defer func() {
		if err := s.userLikeCache.UnlockUserFollowerIDList(ctx, userID, token); err != nil {
			logger.Error("unlock user_follower_id_list cache fail", zap.Error(err))
		}
	}()

	followerIDList, err = s.userLikeRepo.GetUserFollowerIDListByUserID(ctx, s.db, userID)
	if err != nil {
		return make([]string, 0), err
	}
	if err := s.userLikeCache.UpdateUserFollowerIDList(ctx, userID, followerIDList); err != nil {
		logger.Warn("update user_follower_id_list cache fail", zap.Error(err))
	}
	return followerIDList, nil
}

// warmUserFeed 预热 Feed 详情缓存
func (s *feedServer) warmUserFeed(ctx context.Context, feed *model.UserFeed) {
	if feed == nil || feed.FeedID == "" {
		return
	}
	if err := s.feedCache.UpdateFeedDetail(ctx, feed.FeedID, feed); err != nil {
		logger.Warn("warm user feed fail", zap.Error(err))
	}
}
