package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"fiction-book/internal/model"
	"fiction-book/pkg/logger"
)

func TestMain(m *testing.M) {
	if err := logger.Init("debug"); err != nil {
		panic(err)
	}
	os.Exit(m.Run())
}

// fakeES 启动一个假 ES，记录收到的请求体并按给定响应体返回。
func fakeES(t *testing.T, responseBody string) (*elasticsearch.Client, *map[string]interface{}) {
	t.Helper()

	captured := &map[string]interface{}{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		decoded := map[string]interface{}{}
		if err := json.Unmarshal(body, &decoded); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		*captured = decoded

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Elastic-Product", "Elasticsearch")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(responseBody))
	}))
	t.Cleanup(server.Close)

	client, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{server.URL}})
	require.NoError(t, err)
	return client, captured
}

func mustObject(t *testing.T, value interface{}) map[string]interface{} {
	t.Helper()
	obj, ok := value.(map[string]interface{})
	require.True(t, ok, "expected object, got %T", value)
	return obj
}

func mustSlice(t *testing.T, value interface{}) []interface{} {
	t.Helper()
	items, ok := value.([]interface{})
	require.True(t, ok, "expected array, got %T", value)
	return items
}

// findCursorCondition 在 bool.must 中查找游标复合条件
func findCursorCondition(t *testing.T, requestBody map[string]interface{}) map[string]interface{} {
	t.Helper()

	query := mustObject(t, requestBody["query"])
	boolQuery := mustObject(t, query["bool"])
	must := mustSlice(t, boolQuery["must"])

	for _, item := range must {
		condition := mustObject(t, item)
		nested, ok := condition["bool"]
		if !ok {
			continue
		}
		nestedBool := mustObject(t, nested)
		if _, ok := nestedBool["should"]; ok {
			return nestedBool
		}
	}
	require.FailNow(t, "未找到游标复合条件")
	return nil
}

// TestBookESSearchSortAndCursor 验证固定排序、复合游标条件、size 和下一页游标
func TestBookESSearchSortAndCursor(t *testing.T) {
	client, captured := fakeES(t, `{
		"hits": {
			"hits": [
				{"_source": {"id": "book-a", "name": "A"}, "sort": [1730707200123, "book-a"]},
				{"_source": {"id": "book-b", "name": "B"}, "sort": [1730707200123, "book-b"]}
			]
		}
	}`)
	repo := NewBookESRepository(client)

	page, err := repo.Search(context.Background(), &model.BookFilterWithIDPage{
		Cursor: &model.SearchCursor{TimeMS: 1730707200456, ID: "book-cursor"},
		Limit:  2,
	})
	require.NoError(t, err)

	// 固定排序：pushed_at desc + id desc
	sortFields := mustSlice(t, (*captured)["sort"])
	require.Len(t, sortFields, 2)
	assert.Equal(t, map[string]interface{}{"pushed_at": map[string]interface{}{"order": "desc"}}, sortFields[0])
	assert.Equal(t, map[string]interface{}{"id": map[string]interface{}{"order": "desc"}}, sortFields[1])

	// size 等于 limit，不额外多查一条
	assert.Equal(t, float64(2), (*captured)["size"])

	// 游标条件必须是 (pushed_at, id) 复合条件，而不是顶层单一 id range
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
	pushedAtRange := mustObject(t, mustObject(t, firstBranch["range"])["pushed_at"])
	assert.Equal(t, float64(1730707200456), pushedAtRange["lt"])

	secondBranch := mustObject(t, should[1])
	secondMust := mustSlice(t, mustObject(t, secondBranch["bool"])["must"])
	require.Len(t, secondMust, 2)
	equalRange := mustObject(t, mustObject(t, mustObject(t, secondMust[0])["range"])["pushed_at"])
	assert.Equal(t, float64(1730707200456), equalRange["gte"])
	assert.Equal(t, float64(1730707200456), equalRange["lte"])
	idRange := mustObject(t, mustObject(t, mustObject(t, secondMust[1])["range"])["id"])
	assert.Equal(t, "book-cursor", idRange["lt"])

	// 满页时 Next 来自最后一条原始 hit 的 hit.sort
	require.Len(t, page.Results, 2)
	require.NotNil(t, page.Next)
	assert.Equal(t, int64(1730707200123), page.Next.TimeMS)
	assert.Equal(t, "book-b", page.Next.ID)
}

// TestBookESSearchShortPage 验证短页不返回下一页游标
func TestBookESSearchShortPage(t *testing.T) {
	client, _ := fakeES(t, `{
		"hits": {
			"hits": [
				{"_source": {"id": "book-a", "name": "A"}, "sort": [1730707200123, "book-a"]}
			]
		}
	}`)
	repo := NewBookESRepository(client)

	page, err := repo.Search(context.Background(), &model.BookFilterWithIDPage{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, page.Results, 1)
	assert.Nil(t, page.Next)
}

// TestBookESSearchNoCursor 验证第一页不追加游标条件
func TestBookESSearchNoCursor(t *testing.T) {
	client, captured := fakeES(t, `{"hits": {"hits": []}}`)
	repo := NewBookESRepository(client)

	page, err := repo.Search(context.Background(), &model.BookFilterWithIDPage{Limit: 10})
	require.NoError(t, err)
	assert.Nil(t, page.Next)
	assert.Empty(t, page.Results)

	query := mustObject(t, (*captured)["query"])
	must := mustSlice(t, mustObject(t, query["bool"])["must"])
	for _, item := range must {
		condition := mustObject(t, item)
		_, hasNested := condition["bool"]
		assert.False(t, hasNested, "无游标时不应出现嵌套 bool 条件")
	}
}
