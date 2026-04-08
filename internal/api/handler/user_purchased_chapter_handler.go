package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/response"

	"github.com/gin-gonic/gin"
)

// GetUserPurchasedChapter 获取用户已购买的章节列表
// @Summary 获取用户已购买的章节列表
// @Description 获取指定用户已购买的所有章节列表（认证）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response{data=[]dto.GetUserPurchasedChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/chapters/purchases [get]
func (h *Handler) GetUserPurchasedChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserPurchasedChapterRequest](c)

	purchasedChapter, err := h.userPurchasedChapterService.GetListPurchasedChapterByUserID(c.Request.Context(), req.UserID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, purchasedChapter)
}

// CheckUserPurchasedChapter 判断用户是否已购买本章节
// @Summary 判断用户是否已购买本章节
// @Description 检查指定用户是否已购买指定章节（认证）
// @Tags 资产管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param chapter_id path string true "章节ID"
// @Success 200 {object} response.Response{data=dto.GetUserPurchasedChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "未授权"
// @Failure 4016 {object} response.Response "没有购买本章节的记录"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/chapters/purchases/{chapter_id} [get]
func (h *Handler) CheckUserPurchasedChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CheckUserPurchasedChapterRequest](c)

	purchasedChapter, err := h.userPurchasedChapterService.GetPurchasedChapterByUserIDAndChapterID(c.Request.Context(), req.UserID, req.ChapterID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, purchasedChapter)
}
