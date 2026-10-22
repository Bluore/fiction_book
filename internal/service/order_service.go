package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"fiction-book/internal/cache"
	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/internal/repository"
	"fiction-book/pkg/config"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/payment"
	"fiction-book/pkg/response"

	agilePool "github.com/Yiming1997/agilePool/v2"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	// orderLongPollTimeout 订单长轮询最长保持时间，需小于 HTTP WriteTimeout
	orderLongPollTimeout = 30 * time.Second
	// orderLongPollInterval 订单长轮询检查间隔
	orderLongPollInterval = time.Second
	// orderTimeout 订单固定超时时间
	orderTimeout = 5 * time.Minute
	// orderTimeoutExpress 支付宝下单超时参数，需与 orderTimeout 保持一致
	orderTimeoutExpress = "5m"
)

type OrderService interface {
	CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error)
	GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error)
	GetOrderListByUserID(ctx context.Context, userId string, req *dto.GetOrderListRequest) (*dto.GetOrderListResponse, error)
}

type orderService struct {
	userRepo          repository.UserRepository
	userAssetRepo     repository.UserAssetRepository
	orderRepo         repository.OrderRepository
	orderTempRepo     repository.OrderTempRepository
	consumeRecordRepo repository.ConsumeRecordRepository
	orderCache        cache.OrderCache
	db                *gorm.DB
	cfg               *config.Config
	alipayStrategy    payment.Strategy
	pool              *agilePool.Pool
}

func NewOrderService(
	userRepo repository.UserRepository,
	userAssetRepo repository.UserAssetRepository,
	orderRepo repository.OrderRepository,
	orderTempRepo repository.OrderTempRepository,
	consumeRecordRepo repository.ConsumeRecordRepository,
	orderCache cache.OrderCache,
	db *gorm.DB,
	cfg *config.Config,
	alipayStrategy payment.Strategy,
	pool *agilePool.Pool,
) OrderService {
	return &orderService{
		userRepo:          userRepo,
		userAssetRepo:     userAssetRepo,
		orderRepo:         orderRepo,
		orderTempRepo:     orderTempRepo,
		consumeRecordRepo: consumeRecordRepo,
		orderCache:        orderCache,
		db:                db,
		cfg:               cfg,
		alipayStrategy:    alipayStrategy,
		pool:              pool,
	}
}

func (s orderService) CreateOrder(ctx context.Context, userId string, req *dto.CreateOrderRequest) (*dto.CreateOrderResponse, error) {
	var payUrl string
	var orderID string = uuid.New().String()
	var orderNo string = strings.ReplaceAll(orderID, "-", "_")
	var err error
	var reqUUID = uuid.NewSHA1(uuid.NameSpaceDNS, []byte(req.RequestUUID)).String()

	orderDetail := model.Order{
		ID:            orderID,
		UserId:        userId,
		PayPrice:      req.Price,
		OriginalPrice: req.Price,
		RewordGold:    req.RewardGold,
		Status:        model.OrderStatusPending,
		ExpireAt:      time.Now().Add(orderTimeout),
		PaidAt:        nil,
		CreatedAt:     time.Now(),
	}

	// 货币转化率
	if int64(req.Price*float64(s.cfg.Purchased.CurrencyRatio)) != req.RewardGold {
		return nil, response.ErrParam
	}

	// 添加数据
	err = s.db.Transaction(func(tx *gorm.DB) error {
		err = s.orderRepo.CreateOrder(ctx, tx, &orderDetail)
		if err != nil {
			return err
		}

		err = s.consumeRecordRepo.MarkConsumed(ctx, tx, reqUUID)
		if err != nil {
			return response.ErrRepetitionRequest
		}

		// 创建订单
		payUrl, _, err = s.alipayStrategy.CreateOrder(
			payment.Amount{
				Total:    int64(orderDetail.PayPrice * 100),
				Currency: "CNY",
			},
			orderNo,
			fmt.Sprintf("GSF充值账单: %d G币", orderDetail.RewordGold),
			orderTimeoutExpress,
			s.cfg.Payment.Alipay.NotifyURL,
		)
		if err != nil {
			return response.ErrCreateAlipayOrder
		}

		// 写入redis数据
		for i := 1; i <= 3; i++ {
			err := s.orderTempRepo.CreateOrderTemp(ctx, orderDetail)
			if err != nil {
				logger.Error("create order temp fail", zap.Error(err))
				if i == 3 {
					return fmt.Errorf("create order temp fail")
				}
				continue
			}
			break
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &dto.CreateOrderResponse{
		OrderID: orderID,
		PayURL:  payUrl,
		Price:   orderDetail.PayPrice,
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
		Status:        in.Status,
		CreatedAt:     in.CreatedAt,
	}
	if in.Status == model.OrderStatusPending && !time.Now().Before(in.ExpireAt) {
		out.Status = model.OrderStatusOutDate
	}
	return &out
}

// isOrderPending 订单是否仍在等待支付
func isOrderPending(order *model.Order) bool {
	return order.Status == model.OrderStatusPending && time.Now().Before(order.ExpireAt)
}

func (s orderService) GetOrder(ctx context.Context, userId string, id string) (*dto.OrderResponse, error) {
	order, err := s.getOrder(ctx, id)
	if err != nil {
		return nil, err
	}
	if order.UserId != userId {
		return nil, response.ErrPermission
	}

	// 已支付、已超时等终态直接返回，不占用长连接
	if !isOrderPending(order) {
		return toOrderResponse(order), nil
	}

	waitCtx, cancel := context.WithTimeout(ctx, orderLongPollTimeout)
	defer cancel()

	ticker := time.NewTicker(orderLongPollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-waitCtx.Done():
			// 客户端主动断开时不再写响应
			if errors.Is(waitCtx.Err(), context.Canceled) {
				return nil, waitCtx.Err()
			}
			return toOrderResponse(order), nil
		case <-ticker.C:
			latest, err := s.getOrder(waitCtx, id)
			if err != nil {
				return nil, err
			}
			order = latest
			if !isOrderPending(order) {
				return toOrderResponse(order), nil
			}
		}
	}
}

func (s orderService) getOrder(ctx context.Context, id string) (*model.Order, error) {
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
			return nil, response.ErrOrderNotFound
		}

		// 写入缓存
		if err := s.orderCache.Set(ctx, id, order); err != nil {
			logger.Error("Failed to set order cache", zap.String("orderId", id), zap.Error(err))
		}
	}

	return order, nil
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

// Deprecated: 通过order_temp中创建订单
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
				if status == payment.StatusClose {
					logger.Info("Order already passed time", zap.String("orderNo", orderNo))

					err := s.db.Transaction(func(tx *gorm.DB) error {
						txOrder, err := s.orderRepo.GetOrderByID(ctx, tx, orderID)
						if err != nil {
							return err
						}
						if txOrder == nil {
							return response.ErrOrderNotFound
						}

						txOrder.Status = model.OrderStatusOutDate

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
							return response.ErrOrderNotFound
						}

						txOrder.PaidAt = paidTime
						txOrder.Status = model.OrderStatusSuccess

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
