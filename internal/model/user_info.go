package model

import (
	"gorm.io/gorm"
)

const UserInfoTableName = "user_info"

type UserInfo struct {
	UserID         string `json:"user_id" gorm:"primaryKey;type:varchar(36)"`
	Username       string `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	UserProfileUrl string `json:"user_profile_url" gorm:"type:varchar(255);not null"`
	ReadBookNum    int64  `json:"read_book_num" gorm:"type:bigint;not null;default:0"`
	ReadWordsNum   int64  `json:"read_words_num" gorm:"type:bigint;not null;default:0"`
	FansNum        int64  `json:"fans_num" gorm:"type:bigint;not null;default:0"`
	LikeNum        int64  `json:"like_num" gorm:"type:bigint;not null;default:0"`
	BookShelfNum   int64  `json:"book_shelf_num" gorm:"type:bigint;not null;default:0"`
	LastLoginIP    string `json:"last_login_ip" gorm:"type:varchar(50);null"`
	IsV            int    `json:"is_v" gorm:"type:tinyint;not null;default:0"`

	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserInfo) TableName() string {
	return UserInfoTableName
}
