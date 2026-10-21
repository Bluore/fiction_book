package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AlipayNotify 支付宝异步通知
// @Summary 支付宝异步通知
// @Description 接收支付宝支付结果异步通知，验签成功后完成订单发货
// @Tags 支付管理
// @Accept application/x-www-form-urlencoded
// @Produce text/plain
// @Param out_trade_no formData string true "商户订单号"
// @Param trade_status formData string true "交易状态"
// @Param total_amount formData string true "订单金额"
// @Success 200 {string} string "success"
// @Failure 200 {string} string "failure"
// @Router /api/v1/pay/alipay [post]
func (h *Handler) AlipayNotify(c *gin.Context) {
	if err := h.payService.AlipayNotify(c.Request.Context(), c.Request); err != nil {
		// 支付宝约定：返回非 success 会触发重试，不能使用项目统一的 JSON 响应
		c.String(http.StatusOK, "failure")
		return
	}
	c.String(http.StatusOK, "success")
}
