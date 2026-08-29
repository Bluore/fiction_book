package repository

import (
	"context"
	"fiction-book/internal/model"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/utils"
	"fmt"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.uber.org/zap"
)

const (
	keyOrderTemp       = "GSF:order_temp:db:%d:order:%s"
	keyOrderTempPrefix = "GSF:order_temp:db:%d:order:*"
)

func KeyOrderTemp(workspace int, orderID string) string {
	return fmt.Sprintf(keyOrderTemp, workspace, orderID)
}

func KeyOrderTempPrefix(workspace int) string {
	return fmt.Sprintf(keyOrderTempPrefix, workspace)
}

type OrderTempRepository interface {
	CreateOrderTemp(ctx context.Context, orderDetail model.Order) (err error)
	GetAllOrderTemp(ctx context.Context, workspace int) (orderList []*model.Order, err error)
	DeleteOrderTemp(ctx context.Context, orderID string) (err error)
}

type orderTempRepository struct {
	cfg   *config.Config
	redis *redis.Client
}

func NewOrderTemp(cfg *config.Config, redis *redis.Client) OrderTempRepository {
	return &orderTempRepository{cfg: cfg, redis: redis}
}

func (r orderTempRepository) CreateOrderTemp(ctx context.Context, orderDetail model.Order) (err error) {
	var workspace = utils.HashToInt(orderDetail.ID, uint64(r.cfg.Setting.OrderConsumeNum))
	orderStore, err := bson.Marshal(orderDetail)
	if err != nil {
		return err
	}

	_, err = r.redis.SetArgs(ctx, KeyOrderTemp(workspace, orderDetail.ID), orderStore, redis.SetArgs{}).Result()
	if err != nil {
		return err
	}
	return nil
}

func (r orderTempRepository) GetAllOrderTemp(ctx context.Context, workspace int) (orderList []*model.Order, err error) {
	keys, err := r.redis.Keys(ctx, KeyOrderTempPrefix(workspace)).Result()
	if err != nil {
		return make([]*model.Order, 0), err
	}

	for _, key := range keys {
		result, err := r.redis.Get(ctx, key).Result()
		if err != nil {
			logger.Error("get all order temp from redis failed", zap.String("key", key), zap.Error(err))
			continue
		}

		var order model.Order
		err = bson.Unmarshal([]byte(result), &order)
		if err != nil {
			logger.Error("unmarshal order temp error", zap.String("key", key), zap.Error(err))
			continue
		}

		orderList = append(orderList, &order)
	}
	return orderList, nil
}

func (r orderTempRepository) DeleteOrderTemp(ctx context.Context, orderID string) (err error) {
	var workspace = utils.HashToInt(orderID, uint64(r.cfg.Setting.OrderConsumeNum))

	_, err = r.redis.Del(ctx, KeyOrderTemp(workspace, orderID)).Result()
	if err != nil {
		return err
	}
	return nil
}
