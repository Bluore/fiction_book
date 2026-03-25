package dto

import (
	"time"
)

type CreateBookRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	CoverImage  string `json:"cover_image" binding:"required"`
	OwnerId     string `json:"owner_id"`
}

type BookResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CoverImage  string    `json:"cover_image"`
	OwnerID     string    `json:"owner_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type GetBookRequest struct {
	ID string `uri:"id" binding:"required"`
}
