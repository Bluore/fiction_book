package handler

import (
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"time"

	"github.com/gin-gonic/gin"
)

// CreateBook 创建书籍
// @Summary 创建书籍
// @Description 创建一本新书籍（需要认证）
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param request body dto.CreateBookRequest true "创建书籍请求"
// @Success 200 {object} response.Response{data=dto.BookResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4008 {object} response.Response "书籍已存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/create [post]
func (h *Handler) CreateBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateBookRequest](c)
	if req.PushAt.IsZero() {
		req.PushAt = time.Now()
	}

	book, err := h.bookService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrBookExists {
			response.PushError(c, err)
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
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/{id} [get]
func (h *Handler) GetBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookRequest](c)

	book, err := h.bookService.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.PushError(c, err)
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
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
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

// BuyBook 购买书籍
// @Summary 购买书籍
// @Description 购买书籍（认证）
// @Tags 消费管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param request body dto.BuyBookRequest true "更新信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4009 {object} response.Response "书籍已购买"
// @Failure 4016 {object} response.Response "服务器内部错误"
// @Failure 4017 {object} response.Response "价格校验出错"
// @Failure 4018 {object} response.Response "余额不足"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/buy/book [post]
func (h *Handler) BuyBook(c *gin.Context) {
	req := middleware.MustGetRequest[dto.BuyBookRequest](c)
	userID, _ := c.Get("userID")

	purchasedBook, err := h.bookService.BuyBook(c.Request.Context(), userID.(string), req.BookID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, purchasedBook)
}
