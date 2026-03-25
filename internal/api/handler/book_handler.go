package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"github.com/gin-gonic/gin"
)

// CreateBook 创建书籍
// @Summary 创建书籍
// @Description 创建一本新书籍
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param request body dto.CreateBookRequest true "创建书籍请求"
// @Success 200 {object} response.Response{data=dto.BookResponse}
// @Failure 400 {object} response.Response
// @Failure 409 {object} response.Response "书籍已存在"
// @Failure 500 {object} response.Response
// @Router /api/books [post]
func (h *Handler) CreateBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateBookRequest](c)

	book, err := h.bookService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrBookExists {
			response.BadRequest(c, "Book already exists")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, book)
}

// GetBook 获取书籍
// @Summary 获取书籍
// @Description 根据ID获取书籍详情
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param request body dto.GetBookRequest true "获取书籍请求"
// @Success 200 {object} response.Response{data=dto.BookResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/books [get]
func (h *Handler) GetBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookRequest](c)

	book, err := h.bookService.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.BadRequest(c, "Book not found")
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, book)
}
