package dto

import (
	"fiction-book/internal/model"
	"time"
)

// Deprecated: FeedItemResponse 旧版 feed 项响应结构，将被新的 feed 架构替代
type FeedItemResponse struct {
	FeedID       string              `json:"feed_id"`
	EventType    model.FeedEventType `json:"event_type"`
	BookID       string              `json:"book_id"`
	ChapterID    *string             `json:"chapter_id,omitempty"`
	BookName     string              `json:"book_name"`
	BookCover    string              `json:"book_cover"`
	ChapterTitle *string             `json:"chapter_title,omitempty"`
	Content      string              `json:"content"`
	CreatedAt    time.Time           `json:"created_at"`
}

// Deprecated: GetFeedListResponse 旧版 feed 列表响应结构，将被新的 feed 架构替代
type GetFeedListResponse struct {
	Feeds     []*FeedItemResponse `json:"feeds"`
	Restarted bool                `json:"restarted"`
}
