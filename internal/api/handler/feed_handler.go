package handler

import (
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// GetFeedList 获取当前用户 Feed 流
// @Summary 获取当前用户 Feed 流
// @Description 服务端自动维护游标获取 Feed，获取后自动标记为已读；每次固定返回 10 条
// @Tags Feed
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} response.Response{data=dto.GetFeedListResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/feed/list [get]
func (h *Handler) GetFeedList(c *gin.Context) {
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	feeds, err := h.feedService.GetFeedList(c.Request.Context(), userID)
	if err != nil {
		if err == service.ErrParam {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, feeds)
}
