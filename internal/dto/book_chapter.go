package dto

type CreatBookChapterRequest struct {
	BookID  string  `json:"book_id" binding:"required"`
	Title   *string `json:"title" binding:"required"`
	Content *string `json:"content" binding:"required"`
}

type BookChapterResponse struct {
	ChapterID string        `json:"chapter_id"`
	Title     string        `json:"title"`
	Content   string        `json:"content"`
	Order     int           `json:"order"`
	Book      *BookResponse `json:"Book"`
}

type BookChapterInfoResponse struct {
	ChapterID string        `json:"chapter_id"`
	Title     string        `json:"title"`
	Order     int           `json:"order"`
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
}

type GetBookChapterListResponse struct {
	Chapters []BookChapterCatalogueResponse `json:"Chapters"`
}
