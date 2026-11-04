package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"fiction-book/internal/dto"
	"fiction-book/internal/model"
	"fiction-book/pkg/response"
)

func newBookSearchService(repo *MockBookESRepository) *bookService {
	return &bookService{bookESRepo: repo, cfg: newTestConfig()}
}

// TestSearchBooksV2FirstPage 测试不传游标时返回第一页且没有下一页游标
func TestSearchBooksV2FirstPage(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	repo.On("Search", ctx, mock.AnythingOfType("*model.BookFilterWithIDPage")).
		Return(&model.BookSearchPage{Results: []model.BookSearchResult{{
			Book: model.Book{ID: "b1", Name: "book-1"},
		}}}, nil)

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{Limit: 10})
	require.NoError(t, err)
	require.Len(t, res.Books, 1)
	assert.Equal(t, "b1", res.Books[0].Book.ID)
	assert.Zero(t, res.NextTimeMS)
	assert.Empty(t, res.NextID)

	filter := repo.Calls[0].Arguments.Get(1).(*model.BookFilterWithIDPage)
	assert.Nil(t, filter.Cursor)
	assert.Equal(t, 10, filter.Limit)
}

// TestSearchBooksV2CursorPassthrough 测试完整游标被透传给搜索过滤条件
func TestSearchBooksV2CursorPassthrough(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	repo.On("Search", ctx, mock.AnythingOfType("*model.BookFilterWithIDPage")).
		Return(&model.BookSearchPage{}, nil)

	lastTime := int64(1730707200123)
	lastID := "book-cursor"
	svc := newBookSearchService(repo)
	_, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{
		LastTimeMS: &lastTime,
		LastID:     &lastID,
		Limit:      20,
	})
	require.NoError(t, err)

	filter := repo.Calls[0].Arguments.Get(1).(*model.BookFilterWithIDPage)
	require.NotNil(t, filter.Cursor)
	assert.Equal(t, lastTime, filter.Cursor.TimeMS)
	assert.Equal(t, lastID, filter.Cursor.ID)
	assert.Equal(t, 20, filter.Limit)
}

// TestSearchBooksV2OnlyLastID 测试只传 last_id 时返回参数错误
func TestSearchBooksV2OnlyLastID(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	lastID := "book-cursor"

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{LastID: &lastID, Limit: 10})
	assert.ErrorIs(t, err, response.ErrParam)
	assert.Nil(t, res)
	repo.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
}

// TestSearchBooksV2OnlyLastTime 测试只传 last_time_ms 时返回参数错误
func TestSearchBooksV2OnlyLastTime(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	lastTime := int64(1730707200123)

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{LastTimeMS: &lastTime, Limit: 10})
	assert.ErrorIs(t, err, response.ErrParam)
	assert.Nil(t, res)
	repo.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
}

// TestSearchBooksV2NegativeLastTime 测试负数时间游标返回参数错误
func TestSearchBooksV2NegativeLastTime(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	lastTime := int64(-1)
	lastID := "book-cursor"

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{LastTimeMS: &lastTime, LastID: &lastID, Limit: 10})
	assert.ErrorIs(t, err, response.ErrParam)
	assert.Nil(t, res)
	repo.AssertNotCalled(t, "Search", mock.Anything, mock.Anything)
}

// TestSearchBooksV2FullPage 测试满页时下一页游标映射到响应
func TestSearchBooksV2FullPage(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	repo.On("Search", ctx, mock.AnythingOfType("*model.BookFilterWithIDPage")).
		Return(&model.BookSearchPage{
			Results: []model.BookSearchResult{
				{Book: model.Book{ID: "b1", Name: "book-1"}},
				{Book: model.Book{ID: "b2", Name: "book-2"}},
			},
			Next: &model.SearchCursor{TimeMS: 1730707200123, ID: "b2"},
		}, nil)

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Books, 2)
	assert.Equal(t, int64(1730707200123), res.NextTimeMS)
	assert.Equal(t, "b2", res.NextID)
}

// TestSearchBooksV2ShortPage 测试短页时下一页游标为空
func TestSearchBooksV2ShortPage(t *testing.T) {
	ctx := context.Background()
	repo := new(MockBookESRepository)
	repo.On("Search", ctx, mock.AnythingOfType("*model.BookFilterWithIDPage")).
		Return(&model.BookSearchPage{
			Results: []model.BookSearchResult{
				{Book: model.Book{ID: "b1", Name: "book-1"}},
			},
		}, nil)

	svc := newBookSearchService(repo)
	res, err := svc.SearchBooksV2(ctx, &dto.SearchBooksV2Request{Limit: 2})
	require.NoError(t, err)
	require.Len(t, res.Books, 1)
	assert.Zero(t, res.NextTimeMS)
	assert.Empty(t, res.NextID)
}
