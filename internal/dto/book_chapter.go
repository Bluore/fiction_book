package dto

type CreatBookChapterRequest struct {
	BookID   string  `json:"book_id" binding:"required"`
	Title    *string `json:"title" binding:"required"`
	Content  *string `json:"content" binding:"required"`
	VIPLevel string  `json:"vip_level" binding:"required"`
	Price    int     `json:"price" binding:"required"`
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
	ID      string  `uri:"id" binding:"required"`
	Title   *string `json:"title" binding:""`
	Content *string `json:"content" binding:""`
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
