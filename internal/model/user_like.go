package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	UserLikeTableName = "user_like"
)

type UserLike struct {
	ID string `json:"id" gorm:"primaryKey;type:varchar(36)"`

	UserID  string `json:"user_id" gorm:"type:varchar(36);index:idx_userID_likerID;index"`
	LikerID string `json:"liker_id" gorm:"type:varchar(36);index:idx_userID_likerID;index"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (*UserLike) TableName() string {
	return UserLikeTableName
}
