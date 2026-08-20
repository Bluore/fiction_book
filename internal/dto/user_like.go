package dto

type UpdateUserLikeRequest struct {
	TargetID string `json:"target_id" binding:"required"`
	Operator string `json:"operator" binding:"required,oneof=create delete"`
}

type UpdateUserLikeResponse struct {
}

type GetUserLikeRequest struct {
	TargetID string `json:"target_id" form:"target_id" binding:"required"`
}

type GetUserLikeResponse struct {
	TargetID string `json:"target_id"`
	IsLike   bool   `json:"is_like"`
}
