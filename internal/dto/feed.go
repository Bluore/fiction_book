package dto

type GetUserFeedRequestV2 struct {
	LastCursorID   string `json:"last_cursor_id"`
	TargetCursorID string `json:"target_cursor_id"`
	Offset         int    `json:"offset"`
}

type GetUserFeedResponseV2 struct {
	Feeds        []*FeedResponse `json:"feeds"`
	NextCursorID string          `json:"next_cursor_id"`
	NextOffset   int             `json:"next_offset"`
}

type FeedResponse struct {
	BookID        string   `json:"book_id"`
	UserID        string   `json:"user_id"`
	Content       string   `json:"content"`
	FeedCreatedAt UnixTime `json:"feed_created_at"`
	FeedUpdatedAt UnixTime `json:"feed_updated_at"`

	BookName            string   `json:"book_name"`
	BookDescription     string   `json:"book_description"`
	BookCoverUrl        string   `json:"book_cover_url"`
	BookVIPLevel        string   `json:"book_vip_level"`
	BookPrice           int64    `json:"book_price"`
	BookBrowseAmount    uint64   `json:"book_browse_amount"`
	BookBookshelfAmount uint64   `json:"book_bookshelf_amount"`
	BookPushAt          UnixTime `json:"book_push_at"`
	BookCreatedAt       UnixTime `json:"book_created_at"`
	BookUpdatedAt       UnixTime `json:"book_updated_at"`

	UserProfileUrl string `json:"user_profile_url"`
	UserFans       int64  `json:"user_fans"`
	UserVIPLevel   string `json:"user_vip_level"`

	IsInBookShelf bool `json:"is_in_book_shelf"`
	IsUserLiker   bool `json:"is_user_liker"`
}
