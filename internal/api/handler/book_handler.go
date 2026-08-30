package handler

import (
	"errors"
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/pkg/logger"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CreateBook 创建书籍
// @Summary 创建书籍
// @Description 创建一本新书籍（需要认证）
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Security Bearer
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
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	book, err := h.bookService.Create(c.Request.Context(), userID, req)
	if err != nil {
		if err == response.ErrBookExists {
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
// @Param id path string true "书籍ID"
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
		if err == response.ErrBookNotFound {
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
// @Param push_start_time query string false "书籍创建开始时间  (Unix格式)"
// @Param push_end_time query string false "书籍创建结束时间"
// @Success 200 {object} response.Response{data=dto.BookListResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/list [get]
func (h *Handler) BookList(c *gin.Context) {
	req := middleware.MustGetRequest[dto.ListBooksRequest](c).ToStandard()
	req.ToStandard()

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
// @Param request body dto.BuyBookRequest true "购买书籍请求"
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

// UploadBookCoverImage 上传书籍封面
// @Summary 上传书籍封面（验证）
// @Description 用户上传指定书籍的封面图片，支持jpg格式
// @Tags 书籍管理
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param id path string true "书籍ID"
// @Param cover_image formData file true "封面图片文件（仅支持 jpg，尺寸必须为 368X460）"
// @Success 200 {object} dto.UploadBookCoverResponse "封面URL"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "用户未登录"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4005 {object} response.Response "文件大小超出"
// @Failure 4011 {object} response.Response "文件类型出错"
// @Failure 4012 {object} response.Response "图片不合规"
// @Failure 4020 {object} response.Response "你没有该书籍的所有权"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/{id}/cover [post]
func (h *Handler) UploadBookCoverImage(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UploadBookCoverRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, errors.New("user id is empty"))
		return
	}

	res, err := h.bookService.UploadBookCover(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetBookListThermalDaily 获取每日热度榜单
// @Summary 获取每日热度榜单
// @Description 榜单每5分钟刷新，缓存优化
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.UploadBookCoverResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/thermal/daily [get]
func (h *Handler) GetBookListThermalDaily(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookListByThermalDaily](c)
	req.ToStandard()

	res, err := h.bookService.GetBookListByThermalDaily(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetBookListByOwner 获取用户创作的书籍
// @Summary 获取用户创作的书籍
// @Description 获取书籍owner_id为该用户的书籍（认证）
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.UploadBookCoverResponse "封面URL"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "用户登录失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/own/list [get]
func (h *Handler) GetBookListByOwner(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookListByOwnerIDRequest](c)
	req.ToStandard()
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.bookService.GetListByOwnerID(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetUserReadBookHistoryRecently 获取用户最近读书的历史记录
// @Summary 获取用户最近读书的历史记录
// @Description 获取用户最近阅读的书籍，返回每本书籍的详细字段，但是不返回阅读的时间（认证）
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.UploadBookCoverResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "用户登录失效"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/history/list [get]
func (h *Handler) GetUserReadBookHistoryRecently(c *gin.Context) {
	req := middleware.MustGetRequest[dto.PageListRequest](c)
	req.ToStandard()

	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.bookService.GetReadHistoryRecently(c.Request.Context(), userID, req.Page, req.Size)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// UpdateBookInformation 更新书籍信息请求
// @Summary 更新书籍信息请求
// @Description 更新用户书籍的请求，支持更新名称、价格、描述
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "书籍ID"
// @Param request body dto.UpdateBookRequest true "更新信息"
// @Success 200 {object} dto.BookResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "用户登录失效"
// @Failure 4008 {object} response.Response "书籍名称已存在"
// @Failure 4020 {object} response.Response "你没有该书籍的所有权"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/{id}/info [put]
func (h *Handler) UpdateBookInformation(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateBookRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.bookService.UpdateBook(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetBookTag 获取书籍标签
// @Summary 获取书籍标签
// @Description 获取指定书籍的标签列表，缓存优先
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param id path string true "书籍ID"
// @Success 200 {object} response.Response{data=dto.GetBookTagResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4017 {object} response.Response "参数错误"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/books/{id}/tags [get]
func (h *Handler) GetBookTag(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookTagRequest](c)

	res, err := h.bookService.GetBookTags(c.Request.Context(), req.BookID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// UpdateBookTag 更新书籍标签
// @Summary 更新书籍标签
// @Description 更新用户书籍的标签，会先删除旧标签，再添加新标签；每本书标签数量为 1~3 个
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "书籍ID"
// @Param request body dto.UpdateBookTagRequest true "更新标签"
// @Success 200 {object} dto.UpdateBookTagResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 401 {object} response.Response "用户登录失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4017 {object} response.Response "参数错误"
// @Failure 4020 {object} response.Response "你没有该书籍的所有权"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/books/{id}/tags [put]
func (h *Handler) UpdateBookTag(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateBookTagRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.bookService.UpdateBookTag(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetBookListThermalWeekly 获取每周热度榜单
// @Summary 获取每周热度榜单
// @Description 榜单每5分钟刷新，缓存优化，获取今天到前6天的排行总汇
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.UploadBookCoverResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/thermal/weekly [get]
func (h *Handler) GetBookListThermalWeekly(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookListByThermalDaily](c)
	req.ToStandard()

	res, err := h.bookService.GetBookListByThermalWeekly(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// GetBookListThermalMonthly 获取每月热度榜单
// @Summary 获取每月热度榜单
// @Description 榜单每5分钟刷新，缓存优化，获取今天到前30天的排行总汇
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(10)
// @Success 200 {object} dto.UploadBookCoverResponse
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/thermal/monthly [get]
func (h *Handler) GetBookListThermalMonthly(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookListByThermalDaily](c)
	req.ToStandard()

	res, err := h.bookService.GetBookListByThermalMonthly(c.Request.Context(), req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// SearchBooksV2 搜索书籍
// @Summary 搜索书籍
// @Description 搜索书籍，支持多条件过滤、排序和高亮
// @Tags 书籍管理
// @Accept json
// @Produce json
// @Param query query string false "搜索关键词"
// @Param vip_level query string false "VIP等级，多个等级使用逗号分隔 (例如 vip_0,vip_1)"
// @Param push_start_time query int false "推送开始时间 (Unix)"
// @Param push_end_time query int false "推送结束时间 (Unix)"
// @Param update_start_time query int false "更新开始时间 (Unix)"
// @Param update_end_time query int false "更新结束时间 (Unix)"
// @Param price_start query int false "价格最小值 (分)"
// @Param price_end query int false "价格最大值 (分)"
// @Param owner_id query string false "作者ID"
// @Param bookshelf_amount_start query int false "书架数最小值"
// @Param bookshelf_amount_end query int false "书架数最大值"
// @Param thermal_value_start query int false "热度最小值"
// @Param thermal_value_end query int false "热度最大值"
// @Param order query string false "排序字段 (thermal_value|browse_amount|bookshelf_amount|updated_at|pushed_at)" default(thermal_value)
// @Param order_direction query string false "排序方向 (asc|desc)" default(desc)
// @Param last_id query string false "用于向后分页的最后一个ID"
// @Param limit query int false "每页数量 (1-100)" default(10)
// @Success 200 {object} response.Response{data=dto.SearchBooksV2Response}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v2/books/search [get]
func (h *Handler) SearchBooksV2(c *gin.Context) {
	req := middleware.MustGetRequest[dto.SearchBooksV2Request](c)
	req.ToStandard()

	books, err := h.bookService.SearchBooksV2(c.Request.Context(), req)
	if err != nil {
		response.InternalError(c, err)
		logger.Info("HTTP Handler: SearchBooksV2 err", zap.Error(err))
		return
	}
	response.Success(c, books)
}
