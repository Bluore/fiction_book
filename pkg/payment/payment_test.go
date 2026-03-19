package payment

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAmount(t *testing.T) {
	amount := Amount{
		Total:    10000,
		Currency: "CNY",
	}

	assert.Equal(t, int64(10000), amount.Total)
	assert.Equal(t, "CNY", amount.Currency)
}

func TestOrderStatus(t *testing.T) {
	assert.Equal(t, OrderStatus(1), StatusSuccess)
	assert.Equal(t, OrderStatus(2), StatusFail)
	assert.Equal(t, OrderStatus(3), StatusPending)
	assert.Equal(t, OrderStatus(4), StatusError)
}

func TestAlipayConfig(t *testing.T) {
	cfg := AlipayConfig{
		AppID:      "test_app_id",
		PrivateKey: "test_private_key", // pragma: allowlist secret
		NotifyURL:  "https://example.com/notify",
		ReturnURL:  "https://example.com/return",
		IsProd:     false,
	}

	assert.Equal(t, "test_app_id", cfg.AppID)
	assert.False(t, cfg.IsProd)
}

func TestWechatConfig(t *testing.T) {
	cfg := WechatConfig{
		AppID:          "wx_app_id",
		MchID:          "mch_id",
		SerialNo:       "serial_no",
		APIv3Key:       "api_v3_key",
		PrivateKeyPath: "/path/to/key.pem",
		NotifyURL:      "https://example.com/notify",
	}

	assert.Equal(t, "wx_app_id", cfg.AppID)
	assert.Equal(t, "mch_id", cfg.MchID)
}

// MockPaymentStrategy 用于测试的 mock 策略
type MockPaymentStrategy struct {
	payType     string
	createErr   error
	queryStatus OrderStatus
	queryPaidAt *time.Time
}

func (m *MockPaymentStrategy) CreateOrder(_ Amount, orderNo, _ string) (string, string, error) {
	if m.createErr != nil {
		return "", "", m.createErr
	}
	return "https://pay.example.com/order/" + orderNo, orderNo, nil
}

func (m *MockPaymentStrategy) QueryOrder(_ string) (OrderStatus, *time.Time, error) {
	return m.queryStatus, m.queryPaidAt, nil
}

func (m *MockPaymentStrategy) PayType() string {
	return m.payType
}

func TestMockPaymentStrategy(t *testing.T) {
	now := time.Now()
	mock := &MockPaymentStrategy{
		payType:     "mock",
		queryStatus: StatusSuccess,
		queryPaidAt: &now,
	}

	// 验证接口实现
	var _ PaymentStrategy = mock

	// 测试创建订单
	payURL, thirdOrderNo, err := mock.CreateOrder(Amount{Total: 100}, "order123", "Test Order")
	assert.NoError(t, err)
	assert.Contains(t, payURL, "order123")
	assert.Equal(t, "order123", thirdOrderNo)

	// 测试查询订单
	status, paidAt, err := mock.QueryOrder("order123")
	assert.NoError(t, err)
	assert.Equal(t, StatusSuccess, status)
	assert.NotNil(t, paidAt)

	// 测试支付类型
	assert.Equal(t, "mock", mock.PayType())
}

func TestPaymentStrategyInterface(_ *testing.T) {
	// 验证 AlipayStrategy 实现了接口（编译时检查）
	var _ Strategy = (*AlipayStrategy)(nil)

	// 验证 WechatStrategy 实现了接口（编译时检查）
	var _ Strategy = (*WechatStrategy)(nil)
}
