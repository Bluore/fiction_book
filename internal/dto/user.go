package dto

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string `json:"username" binding:"required,username"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Age      int    `json:"age" binding:"omitempty,gte=0,lte=130"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	ID       string  `uri:"id" binding:"required"`
	Username *string `json:"username" binding:"omitempty,username"`
	Email    *string `json:"email" binding:"omitempty,email"`
	Age      *int    `json:"age" binding:"omitempty,gte=0,lte=130"`
}

// GetUserRequest 获取用户请求
type GetUserRequest struct {
	ID string `uri:"id" binding:"required"`
}

// DeleteUserRequest 删除用户请求
type DeleteUserRequest struct {
	ID string `uri:"id" binding:"required"`
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
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Age       int    `json:"age"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
