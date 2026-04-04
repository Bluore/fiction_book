package handler

import (
	_ "image/jpeg"
	_ "image/png"

	"github.com/gin-gonic/gin"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
	"fiction-book/pkg/response"
)

// Handler 处理器结构
type Handler struct {
	userService        service.UserService
	bookService        service.BookService
	bookChapterService service.BookChapterService
	cfg                *config.Config
}

// NewHandler 创建处理器实例
func NewHandler(
	userService service.UserService,
	bookService service.BookService,
	bookChapterService service.BookChapterService,
	cfg *config.Config,
) *Handler {
	return &Handler{
		userService:        userService,
		bookService:        bookService,
		bookChapterService: bookChapterService,
		cfg:                cfg,
	}
}

// HealthCheck 健康检查
// @Summary 健康检查
// @Description 检查服务是否正常运行
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=map[string]string}
// @Router /health [get]
func (h *Handler) HealthCheck(c *gin.Context) {
	response.Success(c, gin.H{
		"status": "ok",
	})
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 注册新用户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param request body dto.CreateUserRequest true "用户信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4003 {object} response.Response "验证码验证失败"
// @Failure 4004 {object} response.Response "验证码未发送"
// @Failure 4007 {object} response.Response "用户已存在"
// @Failure 4013 {object} response.Response "邮箱已存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateUserRequest](c)

	user, err := h.userService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrUserExists {
			response.PushError(c, err)
			return
		}
		if err == service.ErrEmailExists {
			response.PushError(c, err)
			return
		}
		if err == service.ErrEmailVerifySend {
			response.PushError(c, err)
			return
		}
		if err == service.ErrVerification {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, user)
}

// GetUser 获取用户
// @Summary 获取用户信息
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserRequest](c)

	user, err := h.userService.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, user)
}

// UpdateUser 更新用户
// @Summary 更新用户信息
// @Description 更新用户信息（需要认证）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param request body dto.UpdateUserRequest true "更新信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserRequest](c)

	user, err := h.userService.Update(c.Request.Context(), req.ID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, user)
}

// UpdateUserPassword 更新用户密码
// @Summary 更新用户密码
// @Description 更新用户信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param request body dto.UpdateUserPasswordRequest true "更新信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4003 {object} response.Response "验证码验证失败"
// @Failure 4004 {object} response.Response "验证码未发送"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/password [put]
func (h *Handler) UpdateUserPassword(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserPasswordRequest](c)

	user, err := h.userService.UpdatePassword(c.Request.Context(), req.ID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		if err == service.ErrEmailVerifySend {
			response.PushError(c, err)
			return
		}
		if err == service.ErrVerification {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, user)
}

// UpdateUserEmail 更新用户邮箱
// @Summary 更新用户邮箱
// @Description 更新用户信息（需要认证新的邮箱）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param request body dto.UpdateUserEmailRequest true "更新信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4003 {object} response.Response "验证码验证失败"
// @Failure 4004 {object} response.Response "验证码未发送"
// @Failure 4013 {object} response.Response "邮箱已存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/email [put]
func (h *Handler) UpdateUserEmail(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserEmailRequest](c)

	user, err := h.userService.UpdateEmail(c.Request.Context(), req.ID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		if err == service.ErrEmailVerifySend {
			response.PushError(c, err)
			return
		}
		if err == service.ErrVerification {
			response.PushError(c, err)
			return
		}
		if err == service.ErrEmailExists {
			response.PushError(c, err)
			return
		}
		if err == service.ErrInvalidPassword {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, user)
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除用户（需要认证和管理员权限）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Success 200 {object} response.Response
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.DeleteUserRequest](c)

	if err := h.userService.Delete(c.Request.Context(), req.ID); err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, nil)
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录并获取JWT token
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.LoginRequest true "登录信息"
// @Success 200 {object} response.Response{data=dto.LoginResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 4006 {object} response.Response "用户名或密码错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	req := middleware.MustGetRequest[dto.LoginRequest](c)

	loginResp, err := h.userService.Login(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrUserNotFound || err == service.ErrInvalidPassword {
			response.PushError(c, service.ErrInvalidPassword)
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, loginResp)
}

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取用户列表（分页）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} response.Response{data=map[string]interface{}}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	req := middleware.MustGetRequest[dto.ListUsersRequest](c)
	req.SetDefaults()

	users, err := h.userService.List(c.Request.Context(), req.Page, req.PageSize)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, gin.H{
		"page":      req.Page,
		"page_size": req.PageSize,
		"list":      users,
	})
}

// EmailVerify 获取邮箱验证码
// @Summary 获取邮箱验证码
// @Description 向指定邮箱发送验证码
// @Tags 认证
// @Accept json
// @Produce json
// @Param request body dto.EmailVerifyRequest true "邮箱信息"
// @Success 200 {object} response.Response{data=dto.EmailVerifyResponse} "发送成功"
// @Failure 400 {object} response.Response "参数错误"
// @Failure 500 {object} response.Response "服务器内部出错"
// @Router /api/v1/auth/email/verify [post]
func (h *Handler) EmailVerify(c *gin.Context) {
	req := middleware.MustGetRequest[dto.EmailVerifyRequest](c)

	err := h.userService.SendEmailVerification(c.Request.Context(), req.Email)
	if err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, dto.EmailVerifyResponse{
		Email: req.Email,
	})
}

// UploadAvatar 上传头像
// @Summary 上传头像
// @Description 上传并修改用户头像（需要认证，只能修改自己的头像）
// @Tags 用户管理
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param avatar formData file true "头像文件（仅支持 jpeg/png，尺寸必须为 180x180）"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response "参数错误"
// @Failure 401 {object} response.Response "登录已失效"
// @Failure 4002 {object} response.Response "用户未找到"
// @Failure 4005 {object} response.Response "文件大小超出"
// @Failure 4011 {object} response.Response "文件类型出错"
// @Failure 4012 {object} response.Response "图片不合规"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/users/{id}/avatar [post]
func (h *Handler) UploadAvatar(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UploadAvatarRequest](c)

	user, err := h.userService.UpdateAvatar(c.Request.Context(), req.ID, req.Avatar)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.PushError(c, err)
			return
		}
		if err == service.ErrFileType {
			response.PushError(c, err)
			return
		}
		if err == service.ErrImage {
			response.PushError(c, err)
			return
		}
		if err == service.ErrSizeExceed {
			response.PushError(c, err)
		}
		response.InternalError(c, err)
		return
	}
	response.Success(c, user)
}
