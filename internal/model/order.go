package model

import (
	"time"

	"gorm.io/gorm"
)

type Order struct {
	ID            string         `json:"id" gorm:"primary_key; type:varchar(36)"`
	UserId        string         `json:"user_id" gorm:"type:varchar(36)"`
	PayPrice      float64        `json:"pay_price" gorm:"type:decimal(10,2)"`
	OriginalPrice float64        `json:"original_price" gorm:"type:decimal(10,2)"`
	RewordGold    int64          `json:"reword_gold" gorm:"type:decimal(10,0)"`
	PastAt        time.Time      `json:"past_at"`
	Status        int64          `json:"status" gorm:"type:tinyint"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Order) TableName() string {
	return "order"
}
