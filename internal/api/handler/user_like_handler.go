package handler

import (
	"github.com/gin-gonic/gin"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"
)

// UpdateUserLike 关注或取消关注用户
// @Summary 关注或取消关注用户
// @Description 当前登录用户对目标用户执行关注或取消关注操作
// @Tags 关注管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.UpdateUserLikeRequest true "关注操作"
// @Success 200 {object} response.Response{data=dto.UpdateUserLikeResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/likes [put]
func (h *Handler) UpdateUserLike(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserLikeRequest](c)
	likerID := transmit.GetStr(c, "userID")
	if likerID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	result, err := h.userLikeService.UpdateUserLike(c.Request.Context(), likerID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, result)
}

// GetUserLike 查询当前用户是否已关注目标用户
// @Summary 查询用户关注状态
// @Description 查询当前登录用户是否已关注目标用户
// @Tags 关注管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param target_id query string true "目标用户ID"
// @Success 200 {object} response.Response{data=dto.GetUserLikeResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/likes [get]
func (h *Handler) GetUserLike(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserLikeRequest](c)
	likerID := transmit.GetStr(c, "userID")
	if likerID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	result, err := h.userLikeService.GetUserLike(c.Request.Context(), likerID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}

	response.Success(c, result)
}
