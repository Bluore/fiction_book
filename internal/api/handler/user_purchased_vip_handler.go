package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// BuyVIP 购买VIP
// @Summary 购买VIP（认证）
// @Description 使用金币购买 VIP
// @Tags 消费管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.BuyVIPRequest true "购买VIP请求"
// @Success 200 {object} response.Response{data=dto.BuyVIPResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4018 {object} response.Response "余额不足"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/purchases/vip [post]
func (h *Handler) BuyVIP(c *gin.Context) {
	req := middleware.MustGetRequest[dto.BuyVIPRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.buyVIPService.BuyVIP(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, res)
}

// UpgradeVIP 升级VIP
// @Summary 升级VIP（认证）
// @Description 使用金币将当前有效 vip_1 升级为 vip_2，并保持原有效期不变
// @Tags 消费管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.UpgradeVIPRequest true "升级VIP请求"
// @Success 200 {object} response.Response{data=dto.UpgradeVIPResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4018 {object} response.Response "余额不足"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/purchases/vip/upgrade [post]
func (h *Handler) UpgradeVIP(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpgradeVIPRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.buyVIPService.UpgradeVIP(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, res)
}

// ShopListOfVIP 获取VIP商品列表
// @Summary 获取VIP商品列表（认证）
// @Description 获取当前用户所有可购买、可续费和可升级的VIP计划
// @Tags 消费管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=dto.ShopListOfVIPResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/purchases/vip/shop [get]
func (h *Handler) ShopListOfVIP(c *gin.Context) {
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.buyVIPService.ShopListOfVIP(c.Request.Context(), userID)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, res)
}
