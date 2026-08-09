package dto

type UpdateBookTagRequest struct {
	BookID string   `uri:"id" binding:"required"`
	Tags   []string `json:"tags" binding:"required,min=1,max=3,dive,required"`
}

type UpdateBookTagResponse struct {
	BookID string   `json:"book_id"`
	Tags   []string `json:"tags"`
}

type GetBookTagRequest struct {
	BookID string `uri:"id" binding:"required"`
}

type GetBookTagResponse struct {
	BookID string   `json:"book_id"`
	Tags   []string `json:"tags"`
}
