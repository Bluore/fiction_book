package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	UserFeedTableName  = "user_feed"
	UserInboxTableName = "user_inbox"
)

// UserFeed Table
type UserFeed struct {
	FeedID string `json:"feed_id" gorm:"primaryKey;type:varchar(36);index:idx_userID_createAt_feedID_deleteAt,priority:3"`
	UserID string `json:"user_id" gorm:"type:varchar(36);index:idx_userID_createAt_feedID_deleteAt,priority:1"`

	feedContent

	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_userID_createAt_feedID_deleteAt,priority:2"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_userID_createAt_feedID_deleteAt,priority:4"`
}

type feedContent struct {
	BookID  string `json:"book_id" gorm:"type:varchar(36)"`
	Content string `json:"content" gorm:"type:text"`
	Rank    int64  `json:"rank" gorm:"type:bigint"`
}

func (*UserFeed) TableName() string {
	return UserFeedTableName
}

// UserInbox Table
type UserInbox struct {
	ID string `json:"id" gorm:"primaryKey;type:varchar(36);index:idx_userID_createAt_ID_deleteAt,priority:3"`

	UserID string `json:"user_id" gorm:"type:varchar(36);index:idx_userID_createAt_ID_deleteAt,priority:1"`
	FeedID string `json:"feed_id" gorm:"type:varchar(36)"`

	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_userID_createAt_ID_deleteAt,priority:2"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_userID_createAt_ID_deleteAt,priority:4"`
}

func (*UserInbox) TableName() string {
	return UserInboxTableName
}

// UserFeedCursor Feed游标
type UserFeedCursor struct {
	CursorID     string `json:"cursor_id"`
	NextCursorID string `json:"next_cursor_id"`

	LastTime time.Time `json:"last_time"`
	LastID   string    `json:"last_id"`
}
