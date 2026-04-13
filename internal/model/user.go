package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID           string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	Username     string         `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	Email        string         `json:"email" gorm:"uniqueIndex;type:varchar(100);not null"`
	Password     string         `json:"-" gorm:"type:varchar(255);not null"`
	Age          int            `json:"age" gorm:"type:int"`
	Description  string         `json:"description" gorm:"type:text"`
	ProfileImage string         `json:"profile_image" gorm:"type:varchar(255)"`
	CoverImage   string         `json:"cover_image" gorm:"type:varchar(255)"`
	Role         string         `json:"role" gorm:"type:varchar(20);default:'user'"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (User) TableName() string {
	return "user"
}
