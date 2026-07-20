package model

import (
	"time"

	"gorm.io/gorm"
)

const TableNameBookChapter = "book_chapter"

type BookChapterOrder string

const (
	BookChapterOrderCreatedAt = "created_at"
	BookChapterOrderPrice     = "price"
)

type BookChapter struct {
	ID        string         `json:"id" gorm:"primary_key;type:varchar(36)"`
	BookID    string         `json:"book_id" gorm:"type:varchar(36)"`
	Title     string         `json:"title" gorm:"type:varchar(255)"`
	Content   string         `json:"content" gorm:"type:text"`
	Order     int            `json:"order" gorm:"type:integer;not null;default:1"`
	VIPLevel  string         `json:"vip_level" gorm:"type:varchar(255)"`
	Price     int64          `json:"price" gorm:"type:decimal(10,0)"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (BookChapter) TableName() string {
	return TableNameBookChapter
}

type BookChapterInfo struct {
	ID       string `json:"id"`
	BookID   string `json:"book_id"`
	Title    string `json:"title"`
	Order    int    `json:"order"`
	VIPLevel string `json:"vip_level"`
	Price    int64  `json:"price"`
}

func (BookChapterInfo) TableName() string {
	return TableNameBookChapter
}

type BookChapterFilterWithIDPage struct {
	Query          string           `json:"query"`
	BookID         string           `json:"book_id"`
	CreatedAtStart time.Time        `json:"created_at_start"`
	CreatedAtEnd   time.Time        `json:"created_at_end"`
	VIPLevel       []string         `json:"vip_level"`
	PriceStart     int64            `json:"price_start"`
	PriceEnd       int64            `json:"price_end"`
	Order          BookChapterOrder `json:"order"`
	OrderDirection SortOrder        `json:"order_direction"`
	IDPage
}
