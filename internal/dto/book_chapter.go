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
