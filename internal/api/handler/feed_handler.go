package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// GetFeedList 获取当前用户 Feed 流
// @Summary 获取当前用户 Feed 流
// @Description 游标分页获取 Feed，获取后自动标记为已读；cursor 到底后自动从头开始
// @Tags Feed
// @Accept json
// @Produce json
// @Security Bearer
// @Param cursor query string false "游标"
// @Param limit query int false "每次获取数量，默认20，最大50"
// @Success 200 {object} response.Response{data=dto.GetFeedListResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/feed/list [get]
func (h *Handler) GetFeedList(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetFeedListRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	feeds, err := h.feedService.GetFeedList(c.Request.Context(), userID, req)
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
