package model

import (
	"time"

	"gorm.io/gorm"
)

// UserPurchasedBook 用户已购买书籍记录
type UserPurchasedBook struct {
	ID         string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	UserID     string         `json:"userId" gorm:"index;type:varchar(36)"`
	BookID     string         `json:"book_id" gorm:"index;type:varchar(36)"`
	ActiveTime time.Time      `json:"activeTime" gorm:"index"`
	CreatedAt  time.Time      `json:"created_at" gorm:"index"`
	UpdatedAt  time.Time      `json:"updated_at" gorm:"index"`
	DeletedAt  gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserPurchasedBook) TableName() string {
	return "user_purchased_book"
}
