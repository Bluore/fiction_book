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
	ID        int64             `json:"id" gorm:"primary_key;type:bigint(20)"`
	TableName string            `json:"table_name" gorm:"type:varchar(255)"`
	Operation DatabaseOperation `json:"operation" gorm:"type:varchar(255)"`

	Payload json.RawMessage `json:"payload" gorm:"type:JSON"`
	Status  OutboxStatus    `json:"status" gorm:"type:varchar(255)"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Outbox) tableName() string {
	return "outbox"
}
