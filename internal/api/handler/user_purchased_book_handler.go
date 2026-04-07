package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetUserPurchasedBook 获取用户已购买的书籍列表
// @Summary 获取用户已购买的书籍列表
// @Description 获取指定用户已购买的所有书籍列表（认证）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response{data=[]dto.GetUserPurchasedBookResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/purchases [get]
func (h *Handler) GetUserPurchasedBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserPurchasedBookRequest](c)

	purchasedBook, err := h.userPurchasedBookService.GetListPurchasedBookByUserID(c.Request.Context(), req.UserID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, purchasedBook)
}

// CheckUserPurchasedBook 判断用户是否已购买本书籍
// @Summary 判断用户是否已购买本书籍
// @Description 检查指定用户是否已购买指定书籍（认证）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param book_id path string true "书籍ID"
// @Success 200 {object} response.Response{data=dto.GetUserPurchasedBookResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 4016 {object} response.Response "没有购买这本书籍的记录"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/purchases/{book_id} [get]
func (h *Handler) CheckUserPurchasedBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CheckUserPurchasedBookRequest](c)

	purchasedBook, err := h.userPurchasedBookService.GetPurchasedBookByUserIDANDBookID(c.Request.Context(), req.UserID, req.BookID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, purchasedBook)
}
