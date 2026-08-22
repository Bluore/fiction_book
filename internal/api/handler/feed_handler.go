package handler

import (
	"github.com/gin-gonic/gin"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"
)

// GetUserFeedV2 获取当前登录用户的 Feed 流
// @Summary 获取用户 Feed 流
// @Description 获取当前登录用户的 Feed 流，通过游标分页
// @Tags Feed
// @Accept json
// @Produce json
// @Security Bearer
// @Param last_cursor_id query string false "上一页游标ID"
// @Param target_cursor_id query string false "目标游标ID"
// @Param offset query int false "游标内偏移量"
// @Success 200 {object} response.Response{data=dto.GetUserFeedResponseV2}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/feed [get]
func (h *Handler) GetUserFeedV2(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserFeedRequestV2](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	feeds, err := h.feedServiceV2.GetUserFeed(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, feeds)
}

// GuestGetUserFeedV2 获取用户 Feed 流（未登录用户）
// @Summary 获取用户 Feed 流（游客）
// @Description 未登录用户获取 Feed 流，通过游标分页
// @Tags Feed
// @Accept json
// @Produce json
// @Param last_cursor_id query string false "上一页游标ID"
// @Param target_cursor_id query string false "目标游标ID"
// @Param offset query int false "游标内偏移量"
// @Success 200 {object} response.Response{data=dto.GetUserFeedResponseV2}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/feed/guest [get]
func (h *Handler) GuestGetUserFeedV2(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserFeedRequestV2](c)

	feeds, err := h.feedServiceV2.GuestGetUserFeed(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, feeds)
}
