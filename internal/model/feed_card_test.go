package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFeedCardJSONRoundTrip(t *testing.T) {
	card := FeedCard{
		FeedID:              "feed-1",
		UserID:              "author-1",
		BookID:              "book-1",
		Content:             "content",
		FeedCreatedAt:       time.Unix(100, 0).UTC(),
		FeedUpdatedAt:       time.Unix(200, 0).UTC(),
		BookName:            "book",
		BookDescription:     "description",
		BookCoverURL:        "cover",
		BookVIPLevel:        "v1",
		BookPrice:           100,
		BookBrowseAmount:    11,
		BookBookshelfAmount: 12,
		BookPushedAt:        time.Unix(300, 0).UTC(),
		BookCreatedAt:       time.Unix(400, 0).UTC(),
		BookUpdatedAt:       time.Unix(500, 0).UTC(),
		AuthorProfileURL:    "profile",
		AuthorFansNum:       66,
		AuthorVIPLevel:      "v2",
	}

	content, err := json.Marshal(card)
	require.NoError(t, err)

	var got FeedCard
	require.NoError(t, json.Unmarshal(content, &got))
	assert.Equal(t, card, got)
}
