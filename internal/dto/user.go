package dto

import "mime/multipart"

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Age      int    `json:"age" binding:"omitempty,gte=0,lte=130"`
	Verify   string `json:"verify" binding:"omitempty,len=6"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Username    *string `json:"username" binding:"omitempty,username"`
	Age         *int    `json:"age" binding:"omitempty,gte=0,lte=130"`
	Description *string `json:"description" binding:"omitempty,min=1"`
}

// UpdateUserPasswordRequest 更新用户密码请求
type UpdateUserPasswordRequest struct {
	Password string `json:"password" binding:"required,min=6"`
	Verify   string `json:"verify" binding:"required,len=6"`
}

// UpdateUserEmailRequest 更新用户邮箱请求
type UpdateUserEmailRequest struct {
	Password string `json:"password" binding:"required,min=6"`
	Email    string `json:"email" binding:"required,email"`
	Verify   string `json:"verify" binding:"required,len=6"`
}

// GetUserRequest 获取用户请求
type GetUserRequest struct {
	ID string `uri:"id" binding:"required"`
}

// DeleteUserRequest 删除用户请求
type DeleteUserRequest struct {
}

// ListUsersRequest 用户列表请求
type ListUsersRequest struct {
	Page     int `form:"page" binding:"omitempty,gte=1"`
	PageSize int `form:"page_size" binding:"omitempty,gte=1,lte=100"`
}

// SetDefaults 设置默认值
func (r *ListUsersRequest) SetDefaults() {
	if r.Page < 1 {
		r.Page = 1
	}
	if r.PageSize < 1 || r.PageSize > 100 {
		r.PageSize = 10
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token    string `json:"token"`
	UserID   string `json:"user_id"`
	Username string `json:"username"`
}

// UserResponse 用户响应
type UserResponse struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	Age          int    `json:"age"`
	Description  string `json:"description"`
	ProfileImage string `json:"profile_image"`
	CoverImage   string `json:"cover_image"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// EmailVerifyRequest 邮箱验证请求
type EmailVerifyRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// EmailVerifyResponse 邮箱验证响应
type EmailVerifyResponse struct {
	Email string `json:"email"`
}

// UploadAvatarRequest 上传头像请求
type UploadAvatarRequest struct {
	Avatar *multipart.FileHeader `form:"avatar" binding:"required"`
}

// UserStatResponse 用户详情响应
type UserStatResponse struct {
	UserResponse
	UserAssetResponse
	ReadBooksNum int64 `json:"read_books_num"`
	ReadWordsNum int64 `json:"read_words_num"`
}

// ForgetPasswordRequest 忘记密码请求
type ForgetPasswordRequest struct {
	Password    string `json:"password" binding:"required,min=6"`
	Email       string `json:"email" binding:"required,email"`
	EmailVerify string `json:"email_verify" binding:"required,len=6"`
}
