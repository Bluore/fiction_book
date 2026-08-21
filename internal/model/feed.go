package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	UserFeedTableName  = "user_feed"
	UserInboxTableName = "user_inbox"
)

type UserFeed struct {
	FeedID string `json:"feed_id" gorm:"primaryKey;type:varchar(36)"`
	UserID string `json:"user_id" gorm:"type:varchar(36);index:idx_userFeed,priority:1"`

	feedContent

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_userFeed,priority:2"`
}

type feedContent struct {
	BookID  string `json:"book_id" gorm:"primaryKey;type:varchar(36)"`
	Content string `json:"content" gorm:"type:text"`
}

func (*UserFeed) TableName() string {
	return UserFeedTableName
}

type UserInbox struct {
	ID string `json:"id" gorm:"primaryKey;type:varchar(36)"`

	UserID string `json:"user_id" gorm:"type:varchar(36);index:idx_userInbox,priority:1"`
	FeedID string `json:"feed_id" gorm:"type:varchar(36)"`

	CreatedAt time.Time      `json:"created_at" gorm:"index:idx_userInbox,priority:2"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index;index:idx_userInbox,priority:3"`
}

func (*UserInbox) TableName() string {
	return UserInboxTableName
}
