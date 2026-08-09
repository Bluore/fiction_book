package model

import (
	"time"
)

type ConsumeRecord struct {
	ID        string    `json:"id" gorm:"primary_key; type:varchar(36)"`
	CreatedAt time.Time `json:"created_at"`
}

func (ConsumeRecord) TableName() string {
	return "kafka_consume"
}
