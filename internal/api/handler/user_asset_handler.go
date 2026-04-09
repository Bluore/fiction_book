package handler

import (
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// GetUserAsset 获取用户资产
// @Summary 获取用户资产（认证）
// @Description 获取指定用户的资产信息，包括金币、VIP标识和VIP等级
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=dto.UserAssetResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4011 {object} response.Response "用户不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/assets [get]
func (h *Handler) GetUserAsset(c *gin.Context) {
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	userAsset, err := h.userAssetRepository.GetByID(c.Request.Context(), userID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, userAsset)
}
