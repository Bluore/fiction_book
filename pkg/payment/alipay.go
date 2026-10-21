package payment

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/go-pay/gopay"
	"github.com/go-pay/gopay/alipay"
)

// alipayLocation 支付宝异步通知时间所在时区
var alipayLocation = time.FixedZone("CST", 8*60*60)

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

	if cfg.IsDebug {
		// 开启调试日志
		client.DebugSwitch = gopay.DebugOn
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
func (s *AlipayStrategy) CreateOrder(amount Amount, orderNo, subject, timeoutExpress, notifyURL string) (string, string, error) {
	bm := make(gopay.BodyMap)
	bm.Set("subject", subject).
		Set("out_trade_no", orderNo).
		Set("total_amount", fmt.Sprintf("%.2f", float64(amount.Total)/100)).
		Set("qr_pay_mode", 2)
	if timeoutExpress != "" {
		bm.Set("timeout_express", timeoutExpress)
	}
	if notifyURL != "" {
		bm.Set("notify_url", notifyURL)
	}

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
		if resp != nil && resp.Response.Code == "40004" {
			return StatusFail, nil, nil
		}
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

// VerifyNotify 解析并验签支付宝异步通知
func (s *AlipayStrategy) VerifyNotify(req *http.Request) (*NotifyResult, error) {
	bm, err := alipay.ParseNotifyToBodyMap(req)
	if err != nil {
		return nil, fmt.Errorf("alipay parse notify failed: %w", err)
	}

	if bm.GetString("app_id") != s.config.AppID {
		return nil, fmt.Errorf("alipay notify app_id mismatch")
	}
	if bm.GetString("out_trade_no") == "" {
		return nil, fmt.Errorf("alipay notify out_trade_no is empty")
	}

	ok, err := alipay.VerifySign(s.config.AlipayPublicKey, bm)
	if err != nil {
		return nil, fmt.Errorf("alipay notify verify sign failed: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("alipay notify verify sign failed")
	}

	totalAmount, err := yuanToFen(bm.GetString("total_amount"))
	if err != nil {
		return nil, fmt.Errorf("alipay notify total_amount invalid: %w", err)
	}

	paidAt := time.Now()
	if gmtPayment := bm.GetString("gmt_payment"); gmtPayment != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", gmtPayment, alipayLocation); err == nil {
			paidAt = t
		}
	}

	return &NotifyResult{
		OutTradeNo:  bm.GetString("out_trade_no"),
		TradeNo:     bm.GetString("trade_no"),
		TradeStatus: bm.GetString("trade_status"),
		TotalAmount: totalAmount,
		PaidAt:      &paidAt,
	}, nil
}

// yuanToFen 将元为单位的金额转换为分
func yuanToFen(yuan string) (int64, error) {
	amount, err := strconv.ParseFloat(yuan, 64)
	if err != nil {
		return 0, err
	}
	return int64(math.Round(amount * 100)), nil
}

// PayType 返回支付类型
func (s *AlipayStrategy) PayType() string {
	return "alipay"
}

// CloseOrder 关闭订单
func (s *AlipayStrategy) CloseOrder(orderNo string) (closeSuccess bool, status CloseStatus, err error) {
	bm := make(gopay.BodyMap)
	bm.Set("out_trade_no", orderNo)

	result, err := s.client.TradeClose(context.Background(), bm)
	if err != nil {
		return false, CloseStatusError, err
	}

	if result.Response.Code == "10000" {
		return true, CloseStatusSuccess, nil
	}

	switch result.Response.SubCode {
	case "ACQ.REASON_ILLEGAL_STATUS", "ACQ.REASON_TRADE_STATUS_INVALID", "ACQ.TRADE_STATUS_ERROR":
		return false, CloseStatusPending, nil
	case "ACQ.TRADE_NOT_EXIST":
		return true, CloseStatusFail, nil
	case "ACQ.SYSTEM_ERROR", "ACQ.INVALID_PARAMETER":
		return false, CloseStatusError, nil
	default:
		return false, CloseStatusError, nil
	}
}

var _ PaymentStrategy = (*AlipayStrategy)(nil)
