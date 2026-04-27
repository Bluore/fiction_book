package dto

import "time"

type CreateUserBookshelfRequest struct {
	BookID string `json:"book_id" binding:"required"`
}

type UserBookshelfResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ListUserBookshelfRequest struct {
	UserID string `uri:"user_id"`
	PageListRequest
}

type ListUserBookshelfResponse struct {
	Books []*UserBookshelfResponse `json:"books"`
	PageListResponse
}

type DeleteUserBookshelfRequest struct {
	BookID string `json:"book_id" binding:"required"`
}
