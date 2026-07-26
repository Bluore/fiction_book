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
