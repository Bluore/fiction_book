package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// ListUserBookshelf 获取用户书架
// @Summary 获取用户书架
// @Description 获取指定用户的书架列表（分页）
// @Tags 书架管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param user_id path string true "用户ID"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=dto.ListUserBookshelfResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/bookshelf/{user_id} [get]
func (h *Handler) ListUserBookshelf(c *gin.Context) {
	req := middleware.MustGetRequest[dto.ListUserBookshelfRequest](c)
	req.ToStandard()

	books, err := h.userBookshelfService.GetUserBookshelfList(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, books)
}

// CreateUserBookshelf 添加到书架
// @Summary 添加到书架
// @Description 将书籍添加到用户书架（需要认证）
// @Tags 书架管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.CreateUserBookshelfRequest true "书籍信息"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/bookshelf/add [post]
func (h *Handler) CreateUserBookshelf(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateUserBookshelfRequest](c)
	userId := transmit.GetStr(c, "userID")
	if userId == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	err := h.userBookshelfService.CreateUserBookshelf(c.Request.Context(), userId, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, nil)
}
