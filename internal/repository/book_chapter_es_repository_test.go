package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fiction-book/internal/model"
)

// TestBookChapterESSearchSortAndCursor 验证章节搜索固定排序、复合游标条件和下一页游标
func TestBookChapterESSearchSortAndCursor(t *testing.T) {
	client, captured := fakeES(t, `{
		"hits": {
			"hits": [
				{"_source": {"id": "chapter-a", "title": "A"}, "sort": [1730707200789, "chapter-a"]},
				{"_source": {"id": "chapter-b", "title": "B"}, "sort": [1730707200789, "chapter-b"]}
			]
		}
	}`)
	repo := NewBookChapterESRepository(client)

	page, err := repo.Search(context.Background(), model.BookChapterFilterWithIDPage{
		Cursor: &model.SearchCursor{TimeMS: 1730707200999, ID: "chapter-cursor"},
		Limit:  2,
	})
	require.NoError(t, err)

	// 固定排序：created_at desc + id desc
	sortFields := mustSlice(t, (*captured)["sort"])
	require.Len(t, sortFields, 2)
	assert.Equal(t, map[string]interface{}{"created_at": map[string]interface{}{"order": "desc"}}, sortFields[0])
	assert.Equal(t, map[string]interface{}{"id": map[string]interface{}{"order": "desc"}}, sortFields[1])

	// size 等于 limit，不额外多查一条
	assert.Equal(t, float64(2), (*captured)["size"])

	// 游标条件必须是 (created_at, id) 复合条件，而不是顶层单一 id range
	query := mustObject(t, (*captured)["query"])
	must := mustSlice(t, mustObject(t, query["bool"])["must"])
	for _, item := range must {
		condition := mustObject(t, item)
		if rangeValue, ok := condition["range"]; ok {
			_, hasIDRange := mustObject(t, rangeValue)["id"]
			assert.False(t, hasIDRange, "游标条件不应是顶层单一 id range")
		}
	}

	cursorCondition := findCursorCondition(t, *captured)
	should := mustSlice(t, cursorCondition["should"])
	require.Len(t, should, 2)

	firstBranch := mustObject(t, should[0])
	createdAtRange := mustObject(t, mustObject(t, firstBranch["range"])["created_at"])
	assert.Equal(t, float64(1730707200999), createdAtRange["lt"])

	secondBranch := mustObject(t, should[1])
	secondMust := mustSlice(t, mustObject(t, secondBranch["bool"])["must"])
	require.Len(t, secondMust, 2)
	equalRange := mustObject(t, mustObject(t, mustObject(t, secondMust[0])["range"])["created_at"])
	assert.Equal(t, float64(1730707200999), equalRange["gte"])
	assert.Equal(t, float64(1730707200999), equalRange["lte"])
	idRange := mustObject(t, mustObject(t, mustObject(t, secondMust[1])["range"])["id"])
	assert.Equal(t, "chapter-cursor", idRange["lt"])

	// 满页时 Next 来自最后一条原始 hit 的 hit.sort
	require.Len(t, page.Results, 2)
	require.NotNil(t, page.Next)
	assert.Equal(t, int64(1730707200789), page.Next.TimeMS)
	assert.Equal(t, "chapter-b", page.Next.ID)
}

// TestBookChapterESSearchShortPage 验证短页不返回下一页游标
func TestBookChapterESSearchShortPage(t *testing.T) {
	client, _ := fakeES(t, `{
		"hits": {
			"hits": [
				{"_source": {"id": "chapter-a", "title": "A"}, "sort": [1730707200789, "chapter-a"]}
			]
		}
	}`)
	repo := NewBookChapterESRepository(client)

	page, err := repo.Search(context.Background(), model.BookChapterFilterWithIDPage{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page.Results, 1)
	assert.Nil(t, page.Next)
}
