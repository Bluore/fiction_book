package model

import (
	"encoding/json"
	"time"
)

type DatabaseOperation string
type OutboxStatus string

var (
	DatabaseOperationCreate DatabaseOperation = "create"
	DatabaseOperationUpdate DatabaseOperation = "update"
	DatabaseOperationDelete DatabaseOperation = "delete"

	OutboxStatusCreated    OutboxStatus = "created"
	OutboxStatusInProgress OutboxStatus = "progress"
	OutboxStatusFinish     OutboxStatus = "finished"
)

type Outbox struct {
	ID        string            `json:"id" gorm:"primary_key;type:varchar(36)"`
	Type      string            `json:"type" gorm:"type:varchar(255)"`
	Operation DatabaseOperation `json:"operation" gorm:"type:varchar(255)"`

	Payload json.RawMessage `json:"payload" gorm:"type:JSON"`
	Status  OutboxStatus    `json:"status" gorm:"type:varchar(255)"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Outbox) TableName() string {
	return "outbox"
}

type OutboxEvent struct {
	TableName string            `json:"table_name"`
	Operation DatabaseOperation `json:"operation"`
	Payload   json.RawMessage   `json:"payload"`
}
