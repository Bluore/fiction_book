package dto

import (
	"mime/multipart"
	"time"
)

type CreateBookRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	CoverImage  string `json:"cover_image" binding:"required"`
	//OwnerId     string    `json:"owner_id"`
	VIPLevel string    `json:"vip_level"`
	Price    int64     `json:"price"`
	PushAt   time.Time `json:"push_at"`
}

type BookResponse struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	Description     string    `json:"description"`
	CoverImage      string    `json:"cover_image"`
	OwnerID         string    `json:"owner_id"`
	VIPLevel        string    `json:"vip_level"`
	Price           int64     `json:"price"`
	PushAt          time.Time `json:"push_at"`
	BrowseAmount    uint64    `json:"browse_amount"`
	BookshelfAmount uint64    `json:"bookshelf_amount"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type GetBookRequest struct {
	ID string `uri:"id" binding:"required"`
}

type BookListResponse struct {
	Books []*BookResponse `json:"books"`
	PageListResponse
}
type PageListResponse struct {
	Page  int `json:"page"`
	Size  int `json:"size"`
	Count int `json:"count"`
}

type PageListRequest struct {
	Page int `form:"page"`
	Size int `form:"size"`
}

func (p *PageListRequest) ToStandard() {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 {
		p.Size = 10
	}
	if p.Size > 100 {
		p.Size = 100
	}
}

// BookSortKey 排序字段枚举
type BookSortKey string

const (
	SortByBrowseAmount    BookSortKey = "browse_amount"
	SortByBookshelfAmount BookSortKey = "bookshelf_amount"
	SortByNewUpdate       BookSortKey = "new_update"
	SortByNewPush         BookSortKey = "new_push"
)

// SortOrder 排序方向
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ListBooksRequest 带排序的分页请求
type ListBooksRequest struct {
	Page      int         `form:"page"`
	Size      int         `form:"size"`
	SortKey   BookSortKey `form:"sort_key"`
	SortOrder SortOrder   `form:"sort_order"`
}

func (p *ListBooksRequest) ToStandard() *ListBooksRequest {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.Size < 1 {
		p.Size = 10
	}
	if p.Size > 100 {
		p.Size = 100
	}
	// 默认排序：按浏览量降序
	if p.SortKey == "" {
		p.SortKey = SortByNewUpdate
	}
	if p.SortOrder == "" {
		p.SortOrder = SortDesc
	}
	return p
}

type BuyBookRequest struct {
	BookID string `uri:"book_id"`
	Price  int64  `uri:"price"`
}

type UploadBookCoverRequest struct {
	BookID     string                `uri:"id" binding:"required"`
	CoverImage *multipart.FileHeader `form:"cover_image" binding:"required"`
}

type UploadBookCoverResponse struct {
	BookID string `json:"book_id"`
	Url    string `json:"url"`
}
