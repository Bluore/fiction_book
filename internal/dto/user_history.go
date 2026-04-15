package dto

import "time"

type UserHistoryResponse struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	BookID    string    `json:"book_id"`
	CreatedAt time.Time `json:"created_at"`
}

type ListUserHistoryRequest struct {
	UserID string `url:"user_id"`
	PageListResponse
}

type ListUserHistoryResponse struct {
	Books []*UserHistoryResponse `json:"books"`
	PageListResponse
}
