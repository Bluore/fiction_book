package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"github.com/gin-gonic/gin"
	"time"
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
// @Router /api/v1/books/create [post]
func (h *Handler) CreateBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateBookRequest](c)
	if req.PushAt.IsZero() {
		req.PushAt = time.Now()
	}

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
// @Router /api/v1/books/{id} [get]
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

// BookList 获取书籍列表
// @Summary 获取书籍列表
// @Description 分页获取书籍列表
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Param sort_key query string false "排序字段 (browse_amount|bookshelf_amount|new_update|new_push)" default(browse_amount)
// @Param sort_order query string false "排序方向 (asc|desc)" default(desc)
// @Success 200 {object} response.Response{data=dto.BookListResponse}
// @Failure 500 {object} response.Response
// @Router /api/v1/books/list [get]
func (h *Handler) BookList(c *gin.Context) {
	req := middleware.MustGetRequest[dto.ListBooksRequest](c).ToStandard()

	booklist, err := h.bookService.ListBooks(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err)
		return
	}
	response.Success(c, booklist)
}
