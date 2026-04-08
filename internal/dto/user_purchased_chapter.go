package dto

import "time"

type GetUserPurchasedChapterRequest struct {
	UserID string `uri:"id"`
}

type CheckUserPurchasedChapterRequest struct {
	UserID    string `uri:"id"`
	ChapterID string `uri:"chapter_id"`
}

type GetUserPurchasedChapterResponse struct {
	PurchasedID string    `json:"purchased_id"`
	UserID      string    `json:"user_id"`
	ChapterID   string    `json:"chapter_id"`
	BookID      string    `json:"book_id"`
	PurchasedAt time.Time `json:"purchased_at"`
	ActiveAt    time.Time `json:"active_at"`
}
