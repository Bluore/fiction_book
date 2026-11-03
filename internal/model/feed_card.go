package model

import "time"

// FeedSourceKind 标识 Feed 组装依赖的数据来源
type FeedSourceKind string

const (
	FeedSourceFeed          FeedSourceKind = "feed"
	FeedSourceBook          FeedSourceKind = "book"
	FeedSourceUser          FeedSourceKind = "user"
	FeedSourceUserInfo      FeedSourceKind = "user_info"
	FeedSourceUserAsset     FeedSourceKind = "user_asset"
	FeedSourceUserLike      FeedSourceKind = "like"
	FeedSourceUserBookshelf FeedSourceKind = "shelf"
)

// FeedCard 公共组合卡片，仅包含与查看者无关的 Feed 数据
type FeedCard struct {
	FeedID        string    `json:"feed_id"`
	UserID        string    `json:"user_id"`
	BookID        string    `json:"book_id"`
	Content       string    `json:"content"`
	FeedCreatedAt time.Time `json:"feed_created_at"`
	FeedUpdatedAt time.Time `json:"feed_updated_at"`

	BookName            string    `json:"book_name"`
	BookDescription     string    `json:"book_description"`
	BookCoverURL        string    `json:"book_cover_url"`
	BookVIPLevel        string    `json:"book_vip_level"`
	BookPrice           int64     `json:"book_price"`
	BookBrowseAmount    uint64    `json:"book_browse_amount"`
	BookBookshelfAmount uint64    `json:"book_bookshelf_amount"`
	BookPushedAt        time.Time `json:"book_pushed_at"`
	BookCreatedAt       time.Time `json:"book_created_at"`
	BookUpdatedAt       time.Time `json:"book_updated_at"`

	AuthorProfileURL string `json:"author_profile_url"`
	AuthorFansNum    int64  `json:"author_fans_num"`
	AuthorVIPLevel   string `json:"author_vip_level"`
}
