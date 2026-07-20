package dto

import (
	"fiction-book/internal/model"
	"mime/multipart"
	"time"
)

type CreateBookRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description" binding:"required"`
	CoverImage  string `json:"cover_image"`
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
	SortByThermal         BookSortKey = "thermal"
)

// SortOrder 排序方向
type SortOrder string

const (
	SortAsc  SortOrder = "asc"
	SortDesc SortOrder = "desc"
)

// ListBooksRequest 带排序的分页请求
type ListBooksRequest struct {
	Page          int         `form:"page"`
	Size          int         `form:"size"`
	SortKey       BookSortKey `form:"sort_key"`
	SortOrder     SortOrder   `form:"sort_order"`
	PushStartTime time.Time   `form:"push_start_time" time_format:"unix"`
	PushEndTime   time.Time   `form:"push_end_time" time_format:"unix"`
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

type GetBookListByThermalRequest struct {
	PageListRequest
}

type GetBookListByThermalDaily struct {
	PageListRequest
}

type GetBookListByOwnerIDRequest struct {
	PageListRequest
}

type UpdateBookRequest struct {
	BookID      string  `uri:"id" binding:"required"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Price       *int64  `json:"price"`
}

// SearchBooksV2Request v2 版本的搜索请求
type SearchBooksV2Request struct {
	Query                string          `form:"query"`
	VIPLevel             []string        `form:"vip_level"`
	PushStartTime        time.Time       `form:"push_start_time" time_format:"unix"`
	PushEndTime          time.Time       `form:"push_end_time" time_format:"unix"`
	UpdateStartTime      time.Time       `form:"update_start_time" time_format:"unix"`
	UpdateEndTime        time.Time       `form:"update_end_time" time_format:"unix"`
	PriceStart           int64           `form:"price_start"`
	PriceEnd             int64           `form:"price_end"`
	OwnerID              string          `form:"owner_id"`
	BookshelfAmountStart uint64          `form:"bookshelf_amount_start"`
	BookshelfAmountEnd   uint64          `form:"bookshelf_amount_end"`
	ThermalValueStart    uint64          `form:"thermal_value_start"`
	ThermalValueEnd      uint64          `form:"thermal_value_end"`
	Order                model.BookOrder `form:"order"`
	OrderDirection       model.SortOrder `form:"order_direction"`
	LastID               *string         `form:"last_id"`
	Limit                int             `form:"limit"`
}

// BookSearchResult 搜索结果，包含书籍和高亮信息
type BookSearchResult struct {
	Book      BookResponse      `json:"book"`
	Highlight map[string]string `json:"highlight"`
}

// SearchBooksV2Response v2 版本的搜索响应
type SearchBooksV2Response struct {
	Books []*BookSearchResult `json:"books"`
}

func (p *SearchBooksV2Request) ToStandard() *SearchBooksV2Request {
	if p.Limit < 1 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Order == "" {
		p.Order = model.BookOrderThermalValue
	}
	if p.OrderDirection == "" {
		p.OrderDirection = model.SortOrderDesc
	}
	return p
}
