package payment

import (
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

// Strategy 支付策略接口
type Strategy interface {
	// CreateOrder 创建支付订单
	CreateOrder(amount Amount, orderNo string, subject string) (payURL string, thirdOrderNo string, err error)
	// QueryOrder 查询订单状态
	QueryOrder(thirdOrderNo string) (status OrderStatus, paidTime *time.Time, err error)
	// PayType 支付类型标识
	PayType() string
}

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
	NotifyURL  string `mapstructure:"notify_url"`
	ReturnURL  string `mapstructure:"return_url"`
	IsProd     bool   `mapstructure:"is_prod"`
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
