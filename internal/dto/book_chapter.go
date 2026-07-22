package dto

import (
	"fiction-book/internal/model"
	"time"
)

type CreatBookChapterRequest struct {
	BookID   string  `json:"book_id" binding:"required"`
	Title    *string `json:"title" binding:"required"`
	Content  *string `json:"content" binding:"required"`
	VIPLevel *string `json:"vip_level" binding:"required"`
	Price    *int    `json:"price" binding:"required"`
}

type BookChapterResponse struct {
	ChapterID string        `json:"chapter_id"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Order     int           `json:"order"`
	VIPLevel  string        `json:"vip_level"`
	Price     int64         `json:"price"`
	Book      *BookResponse `json:"Book"`
}

type BookChapterInfoResponse struct {
	ChapterID string        `json:"chapter_id"`
	Title     string        `json:"title"`
	Order     int           `json:"order"`
	VIPLevel  string        `json:"vip_level"`
	Price     int64         `json:"price"`
	Book      *BookResponse `json:"Book"`
}

type GetBookChapterRequest struct {
	ChapterID string `uri:"id" binding:"required"`
}

type UpdateBookChapterRequest struct {
	ID       string  `uri:"id" binding:"required"`
	Title    *string `json:"title" binding:""`
	Content  *string `json:"content" binding:""`
	VIPLevel *string `json:"vip_level" binding:""`
	Price    *int64  `json:"price" binding:""`
}

type DeleteBookChapterRequest struct {
	ChapterID string `uri:"id" binding:"required"`
}

type GetBookChapterListRequest struct {
	BookID string `uri:"id" binding:"required"`
}

type BookChapterCatalogueResponse struct {
	ChapterID string `json:"chapter_id"`
	Title     string `json:"title"`
	Order     int    `json:"order"`
	VIPLevel  string `json:"vip_level"`
	Price     int64  `json:"price"`
}

type GetBookChapterListResponse struct {
	Chapters []BookChapterCatalogueResponse `json:"Chapters"`
}

type BuyBookChapterRequest struct {
	BookID    string `json:"book_id" binding:"required"`
	ChapterID string `json:"chapter_id" binding:"required"`
	Price     int64  `json:"price" binding:"required"`
}

// SearchBookChaptersV2Request v2 版本的搜索章节请求
type SearchBookChaptersV2Request struct {
	Query          string                 `form:"query"`
	BookID         string                 `form:"book_id"`
	VIPLevel       []string               `form:"vip_level"`
	CreatedAtStart time.Time              `form:"created_at_start" time_format:"unix"`
	CreatedAtEnd   time.Time              `form:"created_at_end" time_format:"unix"`
	PriceStart     int64                  `form:"price_start"`
	PriceEnd       int64                  `form:"price_end"`
	Order          model.BookChapterOrder `form:"order"`
	OrderDirection model.SortOrder        `form:"order_direction"`
	LastID         *string                `form:"last_id"`
	Limit          int                    `form:"limit"`
}

func (p *SearchBookChaptersV2Request) ToStandard() *SearchBookChaptersV2Request {
	if p.Limit < 1 {
		p.Limit = 10
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Order == "" {
		p.Order = model.BookChapterOrderCreatedAt
	}
	if p.OrderDirection == "" {
		p.OrderDirection = model.SortOrderDesc
	}
	return p
}

// BookChapterSearchResult 搜索结果，包含章节和高亮信息
type BookChapterSearchResult struct {
	BookChapter BookChapterResponse `json:"book_chapter"`
	Highlight   map[string]string   `json:"highlight"`
}

// SearchBookChaptersV2Response v2 版本的搜索响应
type SearchBookChaptersV2Response struct {
	Chapters []*BookChapterSearchResult `json:"chapters"`
}
