package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	OrderStatusPending = 1 + iota // 待付款
	OrderStatusSending            // 待发货
	OrderStatusSuccess            // 已发货
	OrderStatusDeleted            // 已删除
	OrderStatusOutDate            // 已超时
)

type Order struct {
	ID            string         `json:"id" gorm:"primary_key; type:varchar(36)"`
	UserId        string         `json:"user_id" gorm:"type:varchar(36)"`
	PayPrice      float64        `json:"pay_price" gorm:"type:decimal(10,2)"`
	OriginalPrice float64        `json:"original_price" gorm:"type:decimal(10,2)"`
	RewordGold    int64          `json:"reword_gold" gorm:"type:decimal(10,0)"`
	ExpireAt      time.Time      `json:"expire_at" gorm:"type:datetime(3)"`
	Status        int            `json:"status" gorm:"type:tinyint"`
	PaidAt        *time.Time     `json:"paid_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Order) TableName() string {
	return "order"
}
