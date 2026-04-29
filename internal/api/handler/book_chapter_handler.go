package handler

import (
	"errors"
	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/response"
	"fiction-book/pkg/transmit"

	"github.com/gin-gonic/gin"
)

// AddBookChapter 添加章节
// @Summary 创建书籍章节
// @Description 为书籍创建一章章节
// @Tags 章节管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.CreatBookChapterRequest true "创建章节请求"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4020 {object} response.Response "缺失对应权限"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/create [post]
func (h *Handler) AddBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreatBookChapterRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	bookChapterInfo, err := h.bookChapterService.AddByBookID(c.Request.Context(), userID, req)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, bookChapterInfo)
}

// GuestGetBookChapter 获取章节内容(游客)
// @Summary 获取章节内容（游客）
// @Description 获取章节内容（游客）
// @Tags 章节管理
// @Accept json
// @Produce json
// @Param request path dto.GetBookChapterRequest true "获取章节请求"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4010 {object} response.Response "章节未找到"
// @Failure 4014 {object} response.Response "请先完成登录"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/{id}/guest [get]
func (h *Handler) GuestGetBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookChapterRequest](c)

	bookChapter, err := h.bookChapterService.GuestGetChapter(c.Request.Context(), req.ChapterID)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.InternalError(c, err)
			return
		}
		response.PushError(c, err)
		return
	}
	response.Success(c, bookChapter)
}

// GetBookChapter 获取章节内容(认证)
// @Summary 获取章节内容（认证）
// @Description 获取章节内容（认证）
// @Tags 章节管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request path dto.GetBookChapterRequest true "获取章节请求"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4010 {object} response.Response "章节未找到"
// @Failure 4015 {object} response.Response "该章节需要付费"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/{id} [get]
func (h *Handler) GetBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookChapterRequest](c)
	userID, exist := c.Get("userID")
	if !exist {
		response.InternalError(c, errors.New("登录失败"))
		return
	}

	bookChapter, err := h.bookChapterService.GetChapter(c.Request.Context(), userID.(string), req.ChapterID)
	if err != nil {
		if err == service.ErrBookNotFound || err == service.ErrUserNotFound {
			response.InternalError(c, err)
			return
		}
		response.PushError(c, err)
		return
	}
	response.Success(c, bookChapter)
}

// UpdateBookChapter 修改编辑小说章节
// @Summary 修改编辑小说章节 (认证)
// @Description 修改编辑小说章节(认证)
// @Tags 章节管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "章节id"
// @Param request body dto.UpdateBookChapterRequest true "修改章节请求"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4010 {object} response.Response "章节未找到"
// @Failure 4020 {object} response.Response "你没有编辑权限"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/{id} [put]
func (h *Handler) UpdateBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateBookChapterRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	curChapter, err := h.bookChapterService.UpdateChapter(c.Request.Context(), userID, req)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.InternalError(c, err)
		}
		response.PushError(c, err)
		return
	}
	response.Success(c, curChapter)
}

// DeleteBookChapter 删除章节
// @Summary 删除章节 (认证)
// @Description 删除指定章节(认证，需要管理员权限)
// @Tags 章节管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "章节id"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 403 {object} response.Response "禁止访问"
// @Failure 4010 {object} response.Response "章节未找到"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/{id} [delete]
func (h *Handler) DeleteBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.DeleteBookChapterRequest](c)
	userID, exists := c.Get("userID")
	if !exists {
		response.BadRequest(c, "中间件获取数据失败")
		return
	}
	userIDStr, ok := userID.(string)
	if !ok {
		response.BadRequest(c, "中间件获取数据失败")
		return
	}

	err := h.bookChapterService.DeleteChapter(c.Request.Context(), userIDStr, req.ChapterID)
	if err != nil {
		if err == service.ErrBookNotFound {
			response.InternalError(c, err)
			return
		}
		response.PushError(c, err)
		return
	}

	response.Success(c, nil)
}

// GetBookChapterList 获取书籍章节列表
// @Summary 获取书籍章节列表
// @Description 获取书籍章节列表
// @Tags 章节管理
// @Accept json
// @Produce json
// @Param id path string true "书籍id"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/books/{id}/chapters/list [get]
func (h *Handler) GetBookChapterList(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetBookChapterListRequest](c)

	res, err := h.bookChapterService.GetChapterList(c, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}

// BuyBookChapter 购买章节
// @Summary 购买章节
// @Description 购买章节（认证）
// @Tags 消费管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param request body dto.BuyBookChapterRequest true "购买章节请求"
// @Success 200 {object} response.Response{data=string} "购买成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4009 {object} response.Response "书籍已购买"
// @Failure 4017 {object} response.Response "参数错误"
// @Failure 4018 {object} response.Response "余额不足"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/buy/chapter [post]
func (h *Handler) BuyBookChapter(c *gin.Context) {
	req := middleware.MustGetRequest[dto.BuyBookChapterRequest](c)
	userID := transmit.GetStr(c, "userID")
	if userID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	err := h.bookChapterService.BuyBookChapter(c.Request.Context(), userID, req)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, "ok")
}

// GetBookChapterInfo 获取章节信息
// @Summary 获取章节信息
// @Description 获取章节信息
// @Tags 章节管理
// @Accept json
// @Produce json
// @Param id path string true "章节ID"
// @Success 200 {object} response.Response{data=dto.BookChapterResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4001 {object} response.Response "书籍未找到"
// @Failure 4010 {object} response.Response "章节未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/chapters/{id}/info [get]
func (h *Handler) GetBookChapterInfo(c *gin.Context) {
	bookID := c.Param("id")
	if bookID == "" {
		response.PushError(c, ErrGetParam)
		return
	}

	res, err := h.bookChapterService.GetChapterInfo(c.Request.Context(), bookID)
	if err != nil {
		response.PushError(c, err)
		return
	}
	response.Success(c, res)
}
