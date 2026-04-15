package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"

	"github.com/gin-gonic/gin"
)

// ListUserHistory 获取用户浏览历史
// @Summary 获取用户浏览历史
// @Description 获取指定用户的浏览历史（分页）
// @Tags 历史管理
// @Accept application/json
// @Produce application/json
// @Security Bearer
// @Param user_id path string true "用户ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=dto.ListUserHistoryResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/history/{user_id} [get]
func (h *Handler) ListUserHistory(c *gin.Context) {
	req := middleware.MustGetRequest[dto.ListUserHistoryRequest](c)
	books, err := h.userHistoryService.GetUserHistoryList(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, books)
}
