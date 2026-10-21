package service

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strings"

	"fiction-book/internal/cache"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/payment"
	"fiction-book/pkg/response"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PayService interface {
	AlipayNotify(ctx context.Context, req *http.Request) error
}

type payService struct {
	orderRepo      repository.OrderRepository
	orderTempRepo  repository.OrderTempRepository
	userAssetRepo  repository.UserAssetRepository
	orderCache     cache.OrderCache
	userAssetCache cache.UserAssetCache
	alipayStrategy payment.Strategy
	db             *gorm.DB
}

func NewPayService(
	orderRepo repository.OrderRepository,
	orderTempRepo repository.OrderTempRepository,
	userAssetRepo repository.UserAssetRepository,
	orderCache cache.OrderCache,
	userAssetCache cache.UserAssetCache,
	alipayStrategy payment.Strategy,
	db *gorm.DB,
) PayService {
	return &payService{
		orderRepo:      orderRepo,
		orderTempRepo:  orderTempRepo,
		userAssetRepo:  userAssetRepo,
		orderCache:     orderCache,
		userAssetCache: userAssetCache,
		alipayStrategy: alipayStrategy,
		db:             db,
	}
}

// AlipayNotify 处理支付宝异步通知，验签成功后完成订单并发货
func (s payService) AlipayNotify(ctx context.Context, req *http.Request) error {
	notify, err := s.alipayStrategy.VerifyNotify(req)
	if err != nil {
		logger.Error("Alipay notify verify failed", zap.Error(err))
		return response.ErrAlipayNotifyVerify
	}

	// 仅处理支付成功通知，其余状态直接确认收到，避免支付宝重复通知
	if !notify.Paid() {
		logger.Info("Alipay notify ignored",
			zap.String("orderNo", notify.OutTradeNo),
			zap.String("tradeStatus", notify.TradeStatus),
		)
		return nil
	}

	// 创建订单时将 UUID 中的 "-" 替换为 "_" 作为 out_trade_no
	orderID := strings.ReplaceAll(notify.OutTradeNo, "_", "-")
	var userID string

	err = s.db.Transaction(func(tx *gorm.DB) error {
		orderDetail, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
		if err != nil {
			return err
		}
		if orderDetail == nil {
			return response.ErrOrderNotFound
		}
		userID = orderDetail.UserId

		// 已处理或已超时的订单不再重复发放
		if orderDetail.Status != model.OrderStatusPending {
			return fmt.Errorf("order status error: %d", orderDetail.Status)
		}

		if int64(math.Round(orderDetail.PayPrice*100)) != notify.TotalAmount {
			return response.ErrOrderAmountMismatch
		}

		_, err = s.userAssetRepo.AddGoldByID(ctx, tx, userID, orderDetail.RewordGold)
		if err != nil {
			return err
		}

		orderDetail.PaidAt = notify.PaidAt
		orderDetail.Status = model.OrderStatusSuccess
		if err := s.orderRepo.UpdateOrder(ctx, tx, orderDetail); err != nil {
			return err
		}

		// 删除轮询 key，停止后台对账任务继续查询该订单
		return s.orderTempRepo.DeleteOrderTemp(ctx, orderID)
	})
	if err != nil {
		logger.Error("Alipay notify deal order failed",
			zap.String("orderId", orderID),
			zap.String("tradeNo", notify.TradeNo),
			zap.Error(err),
		)
		return err
	}

	// 删除缓存，强制下次查询从数据库获取最新数据
	if err := s.orderCache.Delete(ctx, orderID); err != nil {
		logger.Error("Failed to delete order cache", zap.String("orderId", orderID), zap.Error(err))
	}
	if userID != "" {
		if _, err := s.userAssetCache.DeleteUserAsset(ctx, userID); err != nil {
			logger.Error("Failed to delete user asset cache", zap.String("userId", userID), zap.Error(err))
		}
	}

	return nil
}

var _ PayService = (*payService)(nil)
