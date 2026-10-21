package payment

import (
	"net/http"
	"time"
)

// Amount 金额
type Amount struct {
	Total    int64  `json:"total"`    // 单位：分
	Currency string `json:"currency"` // 货币类型：CNY, USD, GBP
}

// OrderStatus 订单状态
type OrderStatus int

const (
	StatusSuccess OrderStatus = iota + 1
	StatusFail
	StatusPending
	StatusError
)

// CloseStatus 关闭订单状态
type CloseStatus int

const (
	CloseStatusSuccess CloseStatus = iota + 1
	CloseStatusFail
	CloseStatusError
	// CloseStatusPending 关闭状态不合法
	CloseStatusPending
)

// Strategy 支付策略接口
type Strategy interface {
	// CreateOrder 创建支付订单
	CreateOrder(amount Amount, orderNo string, subject string, notifyURL string) (payURL string, thirdOrderNo string, err error)
	// QueryOrder 查询订单状态
	QueryOrder(thirdOrderNo string) (status OrderStatus, paidTime *time.Time, err error)
	// PayType 支付类型标识
	PayType() string
	// CloseOrder 删除订单
	CloseOrder(orderNo string) (closeSuccess bool, status CloseStatus, err error)
	// VerifyNotify 解析并验签异步通知
	VerifyNotify(req *http.Request) (*NotifyResult, error)
}

// NotifyResult 支付异步通知解析结果
type NotifyResult struct {
	OutTradeNo  string
	TradeNo     string
	TradeStatus string
	TotalAmount int64 // 单位：分
	PaidAt      *time.Time
}

// Paid 通知是否为支付成功状态
func (n *NotifyResult) Paid() bool {
	return n.TradeStatus == TradeStatusSuccess || n.TradeStatus == TradeStatusFinished
}

// 支付宝交易状态
const (
	TradeStatusSuccess      = "TRADE_SUCCESS"
	TradeStatusFinished     = "TRADE_FINISHED"
	TradeStatusClosed       = "TRADE_CLOSED"
	TradeStatusWaitBuyerPay = "WAIT_BUYER_PAY"
)

// PaymentStrategy 支付策略接口别名 (保持向后兼容)
// Deprecated: 请使用 Strategy
//
//nolint:revive // keeping for backward compatibility
type PaymentStrategy = Strategy

// Config 支付配置
type Config struct {
	Alipay AlipayConfig `mapstructure:"alipay"`
	Wechat WechatConfig `mapstructure:"wechat"`
}

// AlipayConfig 支付宝配置
type AlipayConfig struct {
	AppID      string `mapstructure:"app_id"`
	PrivateKey string `mapstructure:"private_key"`
	PublicKey  string `mapstructure:"public_key"`
	// AlipayPublicKey 支付宝公钥，用于异步通知验签
	AlipayPublicKey string `mapstructure:"alipay_public_key"`
	NotifyURL       string `mapstructure:"notify_url"`
	ReturnURL       string `mapstructure:"return_url"`
	IsProd          bool   `mapstructure:"is_prod"`
	IsDebug         bool   `mapstructure:"is_debug"`
}

// WechatConfig 微信支付配置
type WechatConfig struct {
	AppID          string `mapstructure:"app_id"`
	MchID          string `mapstructure:"mch_id"`
	SerialNo       string `mapstructure:"serial_no"`
	APIv3Key       string `mapstructure:"api_v3_key"`
	PrivateKeyPath string `mapstructure:"private_key_path"`
	NotifyURL      string `mapstructure:"notify_url"`
}
