package dto

// UpdateUserLikeRequest 更新用户关注请求
type UpdateUserLikeRequest struct {
	TargetID string `json:"target_id" binding:"required"`
	Operator string `json:"operator" binding:"required,oneof=create delete"`
}

// UpdateUserLikeResponse 更新用户关注响应
type UpdateUserLikeResponse struct {
	TargetID string `json:"target_id"`
	IsLike   bool   `json:"is_like"`
}

// GetUserLikeRequest 获取用户关注状态请求
type GetUserLikeRequest struct {
	TargetID string `json:"target_id" form:"target_id" binding:"required"`
}

// GetUserLikeResponse 获取用户关注状态响应
type GetUserLikeResponse struct {
	TargetID string `json:"target_id"`
	IsLike   bool   `json:"is_like"`
}
