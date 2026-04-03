package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"

	"github.com/gin-gonic/gin"
)

// AddBookChapter 添加章节
// @Summary 创建书籍章节
// @Description 为书籍创建一章章节
// @Tags 章节管理
// @Accept json
// @Produce json
// @Param request body dto.CreatBookChapterRequest true "创建章节请求"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Router /api/v1/chapters/create [post]
func (h *Handler) AddBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreatBookChapterRequest](c)

	bookChapterInfo, err := h.bookChapterService.AddByBookID(c.Request.Context(), req.BookID, req)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.BadRequest(c, "Book not found")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, bookChapterInfo)
}
