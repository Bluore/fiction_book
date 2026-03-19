package payment

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
)

// AlipayStrategy 支付宝支付策略
type AlipayStrategy struct {
	client *alipay.Client
	config AlipayConfig
}

// NewAlipayStrategy 创建支付宝策略
func NewAlipayStrategy(cfg AlipayConfig) (*AlipayStrategy, error) {
	client, err := alipay.NewClient(cfg.AppID, cfg.PrivateKey, cfg.IsProd)
	if err != nil {
		return nil, fmt.Errorf("failed to create alipay client: %w", err)
	}

	client.SetLocation(alipay.LocationShanghai).
		SetCharset(alipay.UTF8).
		SetSignType(alipay.RSA2).
		SetReturnUrl(cfg.ReturnURL).
		SetNotifyUrl(cfg.NotifyURL)

	return &AlipayStrategy{
		client: client,
		config: cfg,
	}, nil
}

// CreateOrder 创建支付宝订单
func (s *AlipayStrategy) CreateOrder(amount Amount, orderNo, subject string) (string, string, error) {
	bm := make(gopay.BodyMap)
	bm.Set("subject", subject).
		Set("out_trade_no", orderNo).
		Set("total_amount", fmt.Sprintf("%.2f", float64(amount.Total)/100)).
		Set("qr_pay_mode", 2)

	payURL, err := s.client.TradePagePay(context.Background(), bm)
	if err != nil {
		return "", "", fmt.Errorf("alipay create order failed: %w", err)
	}

	return payURL, orderNo, nil
}

// QueryOrder 查询订单状态
func (s *AlipayStrategy) QueryOrder(thirdOrderNo string) (OrderStatus, *time.Time, error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", thirdOrderNo)

	resp, err := s.client.TradeQuery(context.Background(), bm)
	if err != nil {
		return StatusError, nil, err
	}

	switch resp.Response.TradeStatus {
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		t := time.Now()
		return StatusSuccess, &t, nil
	case "TRADE_CLOSED":
		return StatusFail, nil, nil
	case "WAIT_BUYER_PAY":
		return StatusPending, nil, nil
	default:
		return StatusPending, nil, nil
	}
}

// PayType 返回支付类型
func (s *AlipayStrategy) PayType() string {
	return "alipay"
}

var _ PaymentStrategy = (*AlipayStrategy)(nil)
