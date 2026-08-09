package model

import (
	"time"
)

type FeedEventType string

const (
	FeedEventBookUpdate FeedEventType = "book_update"
	FeedEventNewChapter FeedEventType = "new_chapter"
)

// FeedInbox 用户动态流
type FeedInbox struct {
	ID           string        `json:"id" gorm:"primaryKey;type:varchar(36)"`
	UserID       string        `json:"user_id" gorm:"type:varchar(36);not null;index:idx_feed_user_created;index:idx_feed_user_read_created"`
	EventType    FeedEventType `json:"event_type" gorm:"type:varchar(50);not null"`
	BookID       string        `json:"book_id" gorm:"type:varchar(36);not null;index:idx_feed_book_id"`
	ChapterID    *string       `json:"chapter_id" gorm:"type:varchar(36)"`
	BookName     string        `json:"book_name" gorm:"type:varchar(50);not null"`
	BookCover    string        `json:"book_cover" gorm:"type:varchar(255);not null"`
	ChapterTitle *string       `json:"chapter_title" gorm:"type:varchar(255)"`
	Content      string        `json:"content" gorm:"type:text"`
	IsRead       bool          `json:"is_read" gorm:"type:tinyint;default:0;not null;index:idx_feed_user_read_created"`
	CreatedAt    time.Time     `json:"created_at" gorm:"not null;index:idx_feed_user_created,priority:2;index:idx_feed_user_read_created,priority:3;index:idx_feed_created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
}

func (FeedInbox) TableName() string {
	return "feed_inbox"
}

// FeedCursor 用户 feed 游标
type FeedCursor struct {
	UserID      string     `json:"user_id" gorm:"primaryKey;type:varchar(36)"`
	LastReadAt  *time.Time `json:"last_read_at"`
	LastFeedID  *string    `json:"last_feed_id" gorm:"type:varchar(36)"`
	UnreadCount int        `json:"unread_count" gorm:"type:integer;default:0;not null"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"index:idx_feed_cursor_updated"`
}

func (FeedCursor) TableName() string {
	return "feed_cursor"
}

// FeedFilterWithIDPage FeedInbox 查询过滤器
type FeedFilterWithTimePage struct {
	UserID     string          `json:"user_id"`
	EventTypes []FeedEventType `json:"event_types"`
	BookID     string          `json:"book_id"`
	IsRead     *bool           `json:"is_read"`
	TimePage
}
