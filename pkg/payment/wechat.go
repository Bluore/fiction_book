package payment

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/wechat/v3"
)

// WechatStrategy 微信支付策略
type WechatStrategy struct {
	client *wechat.ClientV3
	config WechatConfig
}

// NewWechatStrategy 创建微信支付策略
func NewWechatStrategy(cfg WechatConfig) (*WechatStrategy, error) {
	privateKey, err := os.ReadFile(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read wechat private key: %w", err)
	}

	client, err := wechat.NewClientV3(cfg.MchID, cfg.SerialNo, cfg.APIv3Key, string(privateKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create wechat client: %w", err)
	}

	if err = client.AutoVerifySign(); err != nil {
		return nil, fmt.Errorf("failed to auto verify sign: %w", err)
	}

	return &WechatStrategy{
		client: client,
		config: cfg,
	}, nil
}

// CreateOrder 创建微信支付订单（Native 扫码支付）
func (s *WechatStrategy) CreateOrder(amount Amount, orderNo, subject string) (string, string, error) {
	bm := make(gopay.BodyMap)
	bm.Set("appid", s.config.AppID).
		Set("mchid", s.config.MchID).
		Set("notify_url", s.config.NotifyURL).
		Set("description", subject).
		Set("out_trade_no", orderNo).
		SetBodyMap("amount", func(bm gopay.BodyMap) {
			bm.Set("total", amount.Total)
		})

	resp, err := s.client.V3TransactionNative(context.Background(), bm)
	if err != nil {
		return "", "", fmt.Errorf("wechat create order failed: %w", err)
	}

	if resp.Code != wechat.Success {
		return "", "", fmt.Errorf("wechat error: %s", resp.Error)
	}

	return resp.Response.CodeUrl, orderNo, nil
}

// QueryOrder 查询订单状态
func (s *WechatStrategy) QueryOrder(thirdOrderNo string) (OrderStatus, *time.Time, error) {
	resp, err := s.client.V3TransactionQueryOrder(context.Background(), wechat.OutTradeNo, thirdOrderNo)
	if err != nil {
		return StatusError, nil, err
	}

	if resp.Code != wechat.Success {
		return StatusError, nil, fmt.Errorf("wechat error: %s", resp.Error)
	}

	switch resp.Response.TradeState {
	case wechat.TradeStateSuccess:
		t, err := time.Parse(time.RFC3339, resp.Response.SuccessTime)
		if err != nil {
			return StatusSuccess, nil, nil
		}
		return StatusSuccess, &t, nil
	case wechat.TradeStateRefund, wechat.TradeStateClosed, wechat.TradeStateRevoked, wechat.TradeStatePayError:
		return StatusFail, nil, nil
	case wechat.TradeStateNoPay, wechat.TradeStatePaying:
		return StatusPending, nil, nil
	default:
		return StatusError, nil, nil
	}
}

// PayType 返回支付类型
func (s *WechatStrategy) PayType() string {
	return "wechat"
}

var _ PaymentStrategy = (*WechatStrategy)(nil)
