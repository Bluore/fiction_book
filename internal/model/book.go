package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameBook = "book"

type BookOrder string

const (
	BookOrderThermalValue    BookOrder = "thermal_value"
	BookOrderBrowseAmount    BookOrder = "browse_amount"
	BookOrderBookshelfAmount BookOrder = "bookshelf_amount"
	BookOrderUpdate          BookOrder = "updated_at"
	BookOrderPush            BookOrder = "pushed_at"
)

// Book 模型
type Book struct {
	ID              string         `json:"id" gorm:"primaryKey;type:varchar(36)"`
	OwnerID         string         `json:"user_id" gorm:"type:varchar(255)"`
	Name            string         `json:"username" gorm:"uniqueIndex;type:varchar(50);not null"`
	Description     string         `json:"description" gorm:"type:text;not null"`
	VIPLevel        string         `json:"vip_level" gorm:"type:varchar(255)"`
	Price           int64          `json:"price" gorm:"type:decimal(10,0)"`
	CoverImage      string         `json:"cover_image" gorm:"type:varchar(255);not null"`
	BrowseAmount    uint64         `json:"browse_amount" gorm:"type:integer;not null;default:0"`
	BookshelfAmount uint64         `json:"bookshelf_amount" gorm:"type:integer;not null;default:0"`
	PushedAt        time.Time      `json:"pushed_at" gorm:"type:timestamp;not null"`
	ThermalValue    uint64         `json:"thermal_value" gorm:"type:integer;not null;default:0"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName 指定表名
func (Book) TableName() string {
	return TableNameBook
}

func (m *Book) CalculateThermal() {
	m.ThermalValue = m.BookshelfAmount*10 + m.BrowseAmount
}

type BookFilterWithIDPage struct {
	Query                string    `json:"query"`
	VIPLevel             []string  `json:"vip_level"`
	PushedAtStart        time.Time `json:"pushed_at_start"`
	PushedAtEnd          time.Time `json:"pushed_at_end"`
	UpdatedAtStart       time.Time `json:"updated_at_start"`
	UpdatedAtEnd         time.Time `json:"updated_at_end"`
	PriceStart           int64     `json:"price_start"`
	PriceEnd             int64     `json:"price_end"`
	OwnerID              string    `json:"owner_id"`
	BookshelfAmountStart uint64    `json:"bookshelf_amount_start"`
	BookshelfAmountEnd   uint64    `json:"bookshelf_amount_end"`
	ThermalValueStart    uint64    `json:"thermal_value_start"`
	ThermalValueEnd      uint64    `json:"thermal_value_end"`
	Order                BookOrder `json:"order"`
	OrderDirection       SortOrder `json:"order_direction"`
	IDPage
}

type BookSearchResult struct {
	Book      Book              `json:"book"`
	Highlight map[string]string `json:"highlight"`
}
