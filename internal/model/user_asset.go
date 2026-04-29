package model

import (
	"time"

	"gorm.io/gorm"
)

type UserAsset struct {
	ID          string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	Gold        int64          `json:"gold" gorm:"not null;type:int(25)"`
	VIPLevel    string         `json:"vip_level" gorm:"not null;type:varchar(20)"`
	VIPActiveAt *time.Time     `json:"vip_active_at" gorm:"type:datetime(3)"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

func (UserAsset) TableName() string {
	return "user_asset"
}
