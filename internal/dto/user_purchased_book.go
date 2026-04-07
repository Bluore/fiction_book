package dto

import "time"

type GetUserPurchasedBookRequest struct {
	UserID string `uri:"id"`
}

type CheckUserPurchasedBookRequest struct {
	UserID string `uri:"id"`
	BookID string `uri:"book_id"`
}

type GetUserPurchasedBookResponse struct {
	PurchasedID string    `json:"purchased_id"`
	UserID      string    `json:"user_id"`
	BookID      string    `json:"book_id"`
	PurchasedAt time.Time `json:"purchased_at"`
	ActiveAt    time.Time `json:"active_at"`
}
