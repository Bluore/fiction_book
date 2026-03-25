package dto

import (
	"time"
)

type CreateBookRequest struct {
	Name        string `json:"name" gorm:"uniqueIndex;type:varchar(50);not null"`
	Description string `json:"description" gorm:"type:text;not null"`
	CoverImage  string `json:"cover_image" gorm:"type:varchar(255);not null"`
	OwnerId     string `json:"owner_id" gorm:"type:varchar(36)"`
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
