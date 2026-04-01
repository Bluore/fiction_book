package handler

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"

	"fiction-book/internal/api/middleware"
	"fiction-book/internal/dto"
	"fiction-book/internal/service"
	"fiction-book/pkg/config"
	"fiction-book/pkg/response"
)

// Handler 处理器结构
type Handler struct {
	userService service.UserService
	bookService service.BookService
	cfg         *config.Config
}

// NewHandler 创建处理器实例
func NewHandler(
	userService service.UserService,
	bookService service.BookService,
	cfg *config.Config,
) *Handler {
	return &Handler{
		userService: userService,
		bookService: bookService,
		cfg:         cfg,
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
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.CreateUserRequest](c)

	user, err := h.userService.Create(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrUserExists {
			response.BadRequest(c, "user already exists")
			return
		}
		if err == service.ErrEmailVerifySend {
			response.BadRequest(c, "please send email verify, firstly")
			return
		}
		if err == service.ErrVerification {
			response.BadRequest(c, "verification failed")
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
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.GetUserRequest](c)

	user, err := h.userService.GetByID(c.Request.Context(), req.ID)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.NotFound(c, "user not found")
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
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/{id} [put]
func (h *Handler) UpdateUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserRequest](c)

	user, err := h.userService.Update(c.Request.Context(), req.ID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.NotFound(c, "user not found")
			return
		}
		response.InternalError(c, err)
		return
	}

	response.Success(c, user)
}

// UpdateUserPassword 更新用户密码
// @Summary 更新用户密码
// @Description 更新用户信息（需要认证）
// @Tags 用户管理
// @Accept json
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param request body dto.UpdateUserPasswordRequest true "更新信息"
// @Success 200 {object} response.Response{data=dto.UserResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/{id}/password [put]
func (h *Handler) UpdateUserPassword(c *gin.Context) {
	req := middleware.MustGetRequest[dto.UpdateUserPasswordRequest](c)

	user, err := h.userService.UpdatePassword(c.Request.Context(), req.ID, req)
	if err != nil {
		if err == service.ErrUserNotFound {
			response.NotFound(c, "user not found")
			return
		}
		if err == service.ErrEmailVerifySend {
			response.BadRequest(c, "please send email verify, firstly")
			return
		}
		if err == service.ErrVerification {
			response.BadRequest(c, "verification failed")
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
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 404 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/{id} [delete]
func (h *Handler) DeleteUser(c *gin.Context) {
	req := middleware.MustGetRequest[dto.DeleteUserRequest](c)

	if err := h.userService.Delete(c.Request.Context(), req.ID); err != nil {
		if err == service.ErrUserNotFound {
			response.NotFound(c, "user not found")
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
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	req := middleware.MustGetRequest[dto.LoginRequest](c)

	loginResp, err := h.userService.Login(c.Request.Context(), req)
	if err != nil {
		if err == service.ErrUserNotFound || err == service.ErrInvalidPassword {
			response.BadRequest(c, "invalid username or password")
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
// @Failure 500 {object} response.Response
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
// @Success 200 {object} response.Response{data=dto.EmailVerifyResponse}
// @Failure 400 {object} response.Response
// @Failure 500 {object} response.Response
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
// @Description 上传用户头像（需要认证，只能修改自己的头像）
// @Tags 用户管理
// @Accept multipart/form-data
// @Produce json
// @Security Bearer
// @Param id path string true "用户ID"
// @Param avatar formData file true "头像文件（仅支持 jpeg/png，尺寸必须为 180x180）"
// @Success 200 {object} response.Response{data=dto.UploadAvatarResponse}
// @Failure 400 {object} response.Response
// @Failure 401 {object} response.Response
// @Failure 403 {object} response.Response
// @Failure 500 {object} response.Response
// @Router /api/v1/users/{id}/avatar [post]
func (h *Handler) UploadAvatar(c *gin.Context) {
	id := c.Param("id")
	userID, exists := c.Get("userID")
	if !exists {
		response.Unauthorized(c)
		return
	}
	if userID.(string) != id {
		response.Forbidden(c, "you can only update your own avatar")
		return
	}

	file, err := c.FormFile("avatar")
	if err != nil {
		response.BadRequest(c, "avatar file is required")
		return
	}

	// 校验文件类型
	contentType := file.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" {
		response.BadRequest(c, "avatar must be jpeg or png")
		return
	}

	// 读取文件内容
	src, err := file.Open()
	if err != nil {
		response.InternalError(c, err)
		return
	}
	data, err := io.ReadAll(src)
	src.Close()
	if err != nil {
		response.InternalError(c, err)
		return
	}

	// 校验图片尺寸
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		response.BadRequest(c, "invalid image format")
		return
	}
	if cfg.Width != 180 || cfg.Height != 180 {
		response.BadRequest(c, "avatar must be exactly 180x180 pixels")
		return
	}

	// 确定文件扩展名
	ext := ".jpg"
	switch contentType {
	case "image/png":
		ext = ".png"
	}

	filename := fmt.Sprintf("%s%s", id, ext)
	uploadDir := "upload/avatar"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		response.InternalError(c, err)
		return
	}
	dstPath := filepath.Join(uploadDir, filename)

	if err := os.WriteFile(dstPath, data, 0644); err != nil {
		response.InternalError(c, err)
		return
	}

	avatarURL := fmt.Sprintf("%s/upload/avatar/%s", h.cfg.Server.Website, filename)
	if _, err := h.userService.UpdateAvatar(c.Request.Context(), id, avatarURL); err != nil {
		response.InternalError(c, err)
		return
	}

	response.Success(c, dto.UploadAvatarResponse{URL: avatarURL})
}
