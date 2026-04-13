package dto

type AddUserBookshelfRequest struct {
	BookID string `json:"book_id" binding:"required"`
}

type UserBookshelfResponse struct {
	ID        string `json:"id"`
	UserID    string `json:"user_id"`
	BookID    string `json:"book_id"`
	CreatedAt string `json:"created_at"`
}

type ListUserBookshelfRequest struct {
	PageListRequest
}

type ListUserBookshelfResponse struct {
	Books []*UserBookshelfResponse `json:"books"`
	PageListResponse
}
