package dto

import (
	"fiction-book/internal/model"
	"time"
)

type GetFeedListRequest struct {
	Cursor string `form:"cursor"`
	Limit  int    `form:"limit"`
}

func (r *GetFeedListRequest) ToStandard() {
	if r.Limit < 1 {
		r.Limit = 20
	}
	if r.Limit > 50 {
		r.Limit = 50
	}
}

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

type GetFeedListResponse struct {
	Feeds      []*FeedItemResponse `json:"feeds"`
	NextCursor string              `json:"next_cursor"`
	HasMore    bool                `json:"has_more"`
	Restarted  bool                `json:"restarted"`
}
