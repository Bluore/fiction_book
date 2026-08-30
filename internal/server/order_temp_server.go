package server

import (
	"context"
	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/payment"
	"fmt"
	"strings"
	"time"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type OrderTempServer struct {
	orderRepo      repository.OrderRepository
	orderTempRepo  repository.OrderTempRepository
	userAssetRepo  repository.UserAssetRepository
	orderCache     cache.OrderCache
	userAssetCache cache.UserAssetCache
	cfg            *config.Config
	db             *gorm.DB
	redis          *redis.Client
	pool           *agilePool.Pool
	payment        payment.Strategy
}

func NewOrderTempServer(
	orderRepo repository.OrderRepository,
	orderTempRepo repository.OrderTempRepository,
	userAssetRepo repository.UserAssetRepository,
	orderCache cache.OrderCache,
	userAssetCache cache.UserAssetCache,
	cfg *config.Config,
	db *gorm.DB,
	redis *redis.Client,
	pool *agilePool.Pool,
	payment payment.Strategy,
) *OrderTempServer {
	return &OrderTempServer{
		orderRepo:      orderRepo,
		orderTempRepo:  orderTempRepo,
		userAssetRepo:  userAssetRepo,
		orderCache:     orderCache,
		userAssetCache: userAssetCache,
		cfg:            cfg,
		db:             db,
		redis:          redis,
		pool:           pool,
		payment:        payment,
	}
}

func (s *OrderTempServer) OrderTempServerInit(ctx context.Context) {
	for i := 0; i < s.cfg.Setting.OrderConsumeNum; i++ {
		go s.OrderTempInitWithWorkspace(ctx, i)
	}
}

func (s *OrderTempServer) OrderTempInitWithWorkspace(ctx context.Context, workspace int) {
	ticker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ticker.C:
			s.consumeOrderTemp(ctx, workspace)
		case <-ctx.Done():
			return
		}
	}
}

func (s *OrderTempServer) consumeOrderTemp(ctx context.Context, workspace int) {
	orderDetails, err := s.orderTempRepo.GetAllOrderTemp(ctx, workspace)
	if err != nil {
		logger.Error("Get all order temp fail", zap.Error(err))
		return
	}

	for _, orderTemp := range orderDetails {
		s.pool.Submit(agilePool.TaskFunc(
			func() error {
				return s.consumeOrderTempOneTask(ctx, workspace, orderTemp)
			},
		))
	}
}

func (s *OrderTempServer) consumeOrderTempOneTask(ctx context.Context, workspace int, orderDetail *model.Order) error {
	var orderID string = orderDetail.ID
	var orderNo string = strings.ReplaceAll(orderID, "-", "_")
	select {
	case <-ctx.Done():
		return nil
	default:
	}

	status, paidTime, err := s.payment.QueryOrder(orderNo)
	if err != nil {
		logger.Info("Error to query pay status", zap.String("orderNo", orderNo), zap.Error(err))
	}
	// 订单未支付
	if status == payment.StatusPending {
		if orderDetail.ExpireAt.Before(time.Now()) {
			err := s.dealExpireOrderTemp(ctx, orderID)
			return err
		}

		return fmt.Errorf("order %s is pending", orderNo)
	}

	// 订单已失效
	if status == payment.StatusFail {
		logger.Info("query order is close", zap.String("orderNo", orderNo))
		if orderDetail.ExpireAt.Before(time.Now()) {
			err := s.dealExpireOrderTemp(ctx, orderID)
			return err
		}
		return nil
	}

	// 订单已支付
	if status == payment.StatusSuccess {
		return s.dealPaidOrderTemp(ctx, orderID, paidTime)
	}

	// 订单错误
	if status == payment.StatusError {
		logger.Info("Order query Error", zap.String("orderNo", orderNo))
		return fmt.Errorf("order %s is fail", orderNo)
	}

	logger.Info("Order query Error", zap.String("orderNo", orderNo))
	return fmt.Errorf("order %s is fail", orderNo)
}

func (s *OrderTempServer) closeOrderTemp(ctx context.Context, orderID string) error {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		orderDetail, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if orderDetail == nil {
			return fmt.Errorf("order %s not found", orderID)
		}

		if orderDetail.Status != model.OrderStatusPending {
			err := s.orderTempRepo.DeleteOrderTemp(ctx, orderID)
			if err != nil {
				logger.Warn("Order query fail", zap.String("orderId", orderID), zap.Error(err))
				return err
			}
			return fmt.Errorf("order %s is not pending", orderID)
		}

		orderDetail.PaidAt = nil
		orderDetail.Status = model.OrderStatusOutDate

		err = s.orderRepo.UpdateOrder(ctx, tx, orderDetail)
		if err != nil {
			return err
		}

		err = s.orderTempRepo.DeleteOrderTemp(ctx, orderID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error("OrderRepo update error", zap.String("order_id", orderID), zap.Error(err))
		return err
	}

	// 删除缓存，强制下次查询从数据库获取最新数据
	if err := s.orderCache.Delete(ctx, orderID); err != nil {
		logger.Error("Failed to delete order cache", zap.String("orderId", orderID), zap.Error(err))
	}

	return nil
}

func (s *OrderTempServer) dealPaidOrderTemp(ctx context.Context, orderID string, paidTime *time.Time) error {
	var userID string
	err := s.db.Transaction(func(tx *gorm.DB) error {
		orderDetail, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if orderDetail == nil {
			return fmt.Errorf("order %s not found", orderID)
		}

		userID = orderDetail.UserId

		if orderDetail.Status != model.OrderStatusPending {
			err := s.orderTempRepo.DeleteOrderTemp(ctx, orderID)
			if err != nil {
				logger.Warn("Order query fail", zap.String("orderId", orderID), zap.Error(err))
				return err
			}
			return fmt.Errorf("order %s is not pending", orderID)
		}

		_, err = s.userAssetRepo.AddGoldByID(ctx, tx, userID, orderDetail.RewordGold)
		if err != nil {
			return err
		}

		orderDetail.PaidAt = paidTime
		orderDetail.Status = model.OrderStatusSuccess

		err = s.orderRepo.UpdateOrder(ctx, tx, orderDetail)
		if err != nil {
			return err
		}

		err = s.orderTempRepo.DeleteOrderTemp(ctx, orderID)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		logger.Error("OrderRepo update error", zap.String("order_id", orderID), zap.Error(err))
		return err
	}

	// 删除缓存，强制下次查询从数据库获取最新数据
	if err := s.orderCache.Delete(ctx, orderID); err != nil {
		logger.Error("Failed to delete order cache", zap.String("orderId", orderID), zap.Error(err))
	}
	if _, err := s.userAssetCache.DeleteUserAsset(ctx, userID); err != nil {
		logger.Error("Failed to delete user asset cache", zap.String("orderId", orderID), zap.Error(err))
	}

	return nil
}

func (s *OrderTempServer) dealExpireOrderTemp(ctx context.Context, orderID string) error {
	var orderNo string = strings.ReplaceAll(orderID, "-", "_")

	success, _, err := s.payment.CloseOrder(orderNo)
	if err != nil {
		return err
	}
	if !success {
		return fmt.Errorf("order close %s is fail", orderNo)
	}

	err = s.closeOrderTemp(ctx, orderID)
	if err != nil {
		return err
	}
	return nil
}
