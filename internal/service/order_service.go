package service

import (
	"context"
	"errors"
	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/constant"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/payment"
	"fmt"
	"strings"
	"time"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"go.uber.org/zap"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrOrderNotFound     = errors.New("order not found")
	ErrCreateAlipayOrder = errors.New("create alipay order fail")
)

const (
	StatusPending = 1  // 待付款
	StatusSending = 2  // 待发货
	StatusSuccess = 3  // 已发货
	StatusDeleted = -1 // 已删除
	StatusOutDate = -2 // 已超时
)

type OrderService interface {
	CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
	GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error)
	GetOrderListByUserID(ctx context.Context, userId string, req *dto.GetOrderListRequest) (*dto.GetOrderListResponse, error)
}

type orderService struct {
	userRepo       repository.UserRepository
	userAssetRepo  repository.UserAssetRepository
	orderRepo      repository.OrderRepository
	orderCache     cache.OrderCache
	db             *gorm.DB
	cfg            *config.Config
	alipayStrategy payment.Strategy
	pool           *agilePool.Pool
}

func NewOrderService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	orderRepo repository.OrderRepository,
	orderCache cache.OrderCache,
	db *gorm.DB,
	cfg *config.Config,
	alipayStrategy payment.Strategy,
	pool *agilePool.Pool,
) OrderService {
	return &orderService{
		userRepo:       userRepo,
		userAssetRepo:  userAssetRepo,
		orderRepo:      orderRepo,
		orderCache:     orderCache,
		db:             db,
		cfg:            cfg,
		alipayStrategy: alipayStrategy,
		pool:           pool,
	}
}

func (s orderService) CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
	var payUrl string
	var orderID string = uuid.New().String()
	var orderNo string = strings.ReplaceAll(orderID, "-", "_")
	var err error

	order := model.Order{
		ID:            orderID,
		UserId:        userId,
		PayPrice:      req.Price,
		OriginalPrice: req.Price,
		RewordGold:    req.RewardGold,
		Status:        StatusPending,
		PastAt:        time.Now().Add(time.Hour * 24),
		PaidAt:        nil,
		CreatedAt:     time.Now(),
	}

	// 货币转化率
	if int64(req.Price)*s.cfg.Purchased.CurrencyRatio != req.RewardGold {
		return nil, ErrParam
	}

	// 创建订单
	payUrl, _, err = s.alipayStrategy.CreateOrder(
		payment.Amount{
			Total:    int64(order.PayPrice * 100),
			Currency: "CNY",
		},
		orderNo,
		fmt.Sprintf("GSF充值账单: %d G币", order.RewordGold),
	)
	if err != nil {
		return nil, ErrCreateAlipayOrder
	}

	// 添加数据
	err = s.orderRepo.CreateOrder(ctx, s.db, &order)
	if err != nil {
		return nil, err
	}

	// 写入缓存
	if err := s.orderCache.Set(ctx, orderID, &order); err != nil {
		logger.Error("Failed to set order cache after creation", zap.String("orderId", orderID), zap.Error(err))
	}

	var submitSuccess bool = false
	if err := s.orderCache.LockGoroutine(ctx, orderID, time.Minute); err == nil {
		// 添加到线程池异步返回结果
		submitSuccess = s.pool.TrySubmit(s.taskCheckOrderTradeSuccess(
			constant.DoneServerCtx,
			userId,
			&order,
		))
	}

	if !submitSuccess {
		logger.Error("submit task error (order)", zap.Any("order", order))
		return nil, errors.New("submit task error")
	}

	return &dto.CreateOrderResponse{
		OrderID: orderID,
		PayURL:  payUrl,
		Price:   order.PayPrice,
	}, nil
}

func toOrderResponse(in *model.Order) *dto.OrderResponse {
	out := dto.OrderResponse{
		OrderID:       in.ID,
		UserID:        in.UserId,
		PayPrice:      in.PayPrice,
		OriginalPrice: in.OriginalPrice,
		RewordGold:    in.RewordGold,
		PastAt:        in.CreatedAt,
		CreatedAt:     in.CreatedAt,
	}
	if time.Now().Before(in.CreatedAt) {
		out.Status = StatusOutDate
	} else if in.Status == StatusDeleted {
		out.Status = StatusDeleted
	} else {
		out.Status = int(in.Status)
	}
	return &out
}

func (s orderService) GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error) {
	// 尝试从缓存获取
	order, exist, err := s.orderCache.Get(ctx, id)
	if err != nil {
		logger.Error("Failed to get order from cache", zap.String("orderId", id), zap.Error(err))
	}

	// 缓存未命中，从数据库获取
	if !exist || order == nil {
		order, err = s.orderRepo.GetOrderByID(ctx, s.db, id)
		if err != nil {
			return nil, err
		}
		if order == nil {
			return nil, ErrOrderNotFound
		}

		// 写入缓存
		if err := s.orderCache.Set(ctx, id, order); err != nil {
			logger.Error("Failed to set order cache", zap.String("orderId", id), zap.Error(err))
		}
	}

	if order.UserId != userId {
		return nil, ErrPermission
	}

	if order.Status == StatusPending {
		err := s.orderCache.LockGoroutine(ctx, order.ID, time.Minute)
		if err == nil {
			s.pool.Submit(s.taskCheckOrderTradeSuccess(
				constant.DoneServerCtx,
				userId,
				order,
			))
		}
	}

	return toOrderResponse(order), nil
}

func (s orderService) GetOrderListByUserID(ctx context.Context, userId string, req *dto.GetOrderListRequest) (*dto.GetOrderListResponse, error) {
	orders, err := s.orderRepo.GetOrderListByUserID(ctx, s.db, userId, req.Page, req.Size)
	if err != nil {
		return nil, err
	}

	var outOrder []*dto.OrderResponse
	for _, order := range orders {
		outOrder = append(outOrder, toOrderResponse(order))
	}
	if len(outOrder) == 0 {
		outOrder = make([]*dto.OrderResponse, 0)
	}

	count, err := s.orderRepo.Count(ctx, s.db, userId)
	if err != nil {
		return nil, err
	}

	out := &dto.GetOrderListResponse{
		Orders: outOrder,
		PageListResponse: dto.PageListResponse{
			Page:  req.Page,
			Size:  req.Size,
			Count: count,
		},
	}

	return out, nil
}

func (s orderService) taskCheckOrderTradeSuccess(ctx context.Context, userId string, order *model.Order) agilePool.TaskFunc {
	// 默认等待 60s
	contextWithCancelTime, contextWithCancelTimeCancel := context.WithTimeout(context.Background(), time.Minute)
	defer contextWithCancelTimeCancel()

	var orderID string = order.ID
	var orderNo string = strings.ReplaceAll(orderID, "-", "_")
	return agilePool.TaskFunc(
		func() error {
			time.Sleep(time.Millisecond * 5 * 1000) // 5000ms
			for {
				time.Sleep(time.Millisecond * 1000) // 1000ms

				select {
				case <-ctx.Done():
					err := s.orderCache.UnlockGoroutine(context.Background(), orderID)
					if err != nil {
						logger.Error("Failed to Unlock order Goroutine", zap.String("orderId", orderID), zap.Error(err))
						return err
					}
					return nil
				case <-contextWithCancelTime.Done():
					err := s.orderCache.UnlockGoroutine(context.Background(), orderID)
					if err != nil {
						logger.Error("Failed to Unlock order Goroutine", zap.String("orderId", orderID), zap.Error(err))
						return err
					}
					return nil
				default:
				}

				status, paidTime, err := s.alipayStrategy.QueryOrder(orderNo)
				if err != nil {
					logger.Info("Error to query pay status", zap.String("orderNo", orderNo), zap.Error(err))
				}
				// 订单未支付
				if status == payment.StatusPending {
					continue
				}

				// 订单已失效
				if status == payment.StatusFail {
					logger.Info("Order already passed time", zap.String("orderNo", orderNo))

					err := s.db.Transaction(func(tx *gorm.DB) error {
						txOrder, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
						if err != nil {
							return err
						}
						if txOrder == nil {
							return ErrOrderNotFound
						}

						txOrder.Status = StatusOutDate

						err = s.orderRepo.UpdateOrder(ctx, tx, txOrder)
						return err
					})
					if err != nil {
						logger.Error("OrderRepo update error", zap.String("orderNo", orderNo), zap.Error(err))
					} else {
						// 删除缓存
						if err := s.orderCache.Delete(ctx, orderID); err != nil {
							logger.Error("Failed to delete order cache", zap.String("orderId", orderID), zap.Error(err))
						}
					}

					return nil
				}

				// 订单已支付
				if status == payment.StatusSuccess {
					err := s.db.Transaction(func(tx *gorm.DB) error {
						_, err := s.userAssetRepo.AddGoldByID(ctx, tx, userId, order.RewordGold)
						if err != nil {
							return err
						}

						txOrder, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
						if err != nil {
							return err
						}
						if txOrder == nil {
							return ErrOrderNotFound
						}

						txOrder.PaidAt = paidTime
						txOrder.Status = StatusSuccess

						err = s.orderRepo.UpdateOrder(ctx, tx, txOrder)
						return err
					})
					if err != nil {
						logger.Error("OrderRepo update error", zap.String("orderNo", orderNo), zap.Error(err))
						continue
					}

					// 删除缓存，强制下次查询从数据库获取最新数据
					if err := s.orderCache.Delete(ctx, orderID); err != nil {
						logger.Error("Failed to delete order cache", zap.String("orderId", orderID), zap.Error(err))
					}

					return nil
				}

				// 订单错误
				if status == payment.StatusError {
					logger.Info("Order query Error", zap.String("orderNo", orderNo))
				}
			}

		},
	)
}
