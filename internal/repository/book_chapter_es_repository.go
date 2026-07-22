package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/pkg/logger"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

type BookChapterESRepository interface {
	Create(ctx context.Context, chapter *model.BookChapter) error
	Update(ctx context.Context, chapter *model.BookChapter) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter model.BookChapterFilterWithIDPage) ([]model.BookChapterSearchResult, error)
}

type bookChapterESRepository struct {
	esClient *elasticsearch.Client
}

func NewBookChapterESRepository(esClient *elasticsearch.Client) BookChapterESRepository {
	return &bookChapterESRepository{esClient: esClient}
}

func (r *bookChapterESRepository) Create(ctx context.Context, chapter *model.BookChapter) error {
	jsonData, err := json.Marshal(chapter)
	if err != nil {
		return fmt.Errorf("序列化章节数据失败: %w", err)
	}

	res, err := r.esClient.Index(
		model.TableNameBookChapter,
		bytes.NewReader(jsonData),
		r.esClient.Index.WithDocumentID(chapter.ID),
		r.esClient.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("创建章节ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES创建文档返回错误状态: %s", res.Status())
	}

	return nil
}

func (r *bookChapterESRepository) Update(ctx context.Context, chapter *model.BookChapter) error {
	updateDoc := map[string]interface{}{
		"doc":           chapter,
		"doc_as_upsert": true,
	}

	jsonData, err := json.Marshal(updateDoc)
	if err != nil {
		return fmt.Errorf("序列化章节更新数据失败: %w", err)
	}

	res, err := r.esClient.Update(
		model.TableNameBookChapter,
		chapter.ID,
		bytes.NewReader(jsonData),
		r.esClient.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("更新章节ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("ES更新文档返回错误状态: %s", res.Status())
	}

	return nil
}

func (r *bookChapterESRepository) Delete(ctx context.Context, id string) error {
	res, err := r.esClient.Delete(
		model.TableNameBookChapter,
		id,
		r.esClient.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("删除章节ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("ES删除文档返回错误状态: %s", res.Status())
	}

	return nil
}

// Search 搜索图书章节ES
func (r *bookChapterESRepository) Search(ctx context.Context, filter model.BookChapterFilterWithIDPage) ([]model.BookChapterSearchResult, error) {
	query := map[string]interface{}{
		"bool": map[string]interface{}{
			"must": []interface{}{},
		},
	}

	// 处理查询关键词
	if filter.Query != "" {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"multi_match": map[string]interface{}{
					"query":  filter.Query,
					"fields": []string{"title", "content"},
					"type":   "best_fields",
				},
			},
		)
	}

	// 处理 BookID 过滤
	if filter.BookID != "" {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"term": map[string]interface{}{
					"book_id": filter.BookID,
				},
			},
		)
	}

	// 处理 CreatedAt 范围过滤
	if !filter.CreatedAtStart.IsZero() || !filter.CreatedAtEnd.IsZero() {
		createdAtRange := map[string]interface{}{}
		if !filter.CreatedAtStart.IsZero() {
			createdAtRange["gte"] = filter.CreatedAtStart
		}
		if !filter.CreatedAtEnd.IsZero() {
			createdAtRange["lte"] = filter.CreatedAtEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"created_at": createdAtRange,
				},
			},
		)
	}

	// 处理 VIP 等级过滤
	if len(filter.VIPLevel) > 0 {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"terms": map[string]interface{}{
					"vip_level": filter.VIPLevel,
				},
			},
		)
	}

	// 处理 Price 范围过滤
	if filter.PriceStart > 0 || filter.PriceEnd > 0 {
		priceRange := map[string]interface{}{}
		if filter.PriceStart > 0 {
			priceRange["gte"] = filter.PriceStart
		}
		if filter.PriceEnd > 0 {
			priceRange["lte"] = filter.PriceEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"price": priceRange,
				},
			},
		)
	}

	// 处理 LastID 筛选：ID 比 LastID 小的结果 (游标查询)
	if filter.LastID != nil && *filter.LastID != "" {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"id": map[string]interface{}{
						"lt": *filter.LastID,
					},
				},
			},
		)
	}

	// 构建请求体
	requestBody := map[string]interface{}{
		"query": query,
		"size":  filter.Limit,
	}

	// 添加高亮配置
	if filter.Query != "" {
		requestBody["highlight"] = map[string]interface{}{
			"fields": map[string]interface{}{
				"title":   map[string]interface{}{},
				"content": map[string]interface{}{},
			},
		}
	}

	// 处理排序：先按用户指定的字段排序，最后固定以ID降序排序
	if filter.Order == "" {
		filter.Order = model.BookChapterOrderCreatedAt
	}
	var userSortOrder string
	if filter.OrderDirection == model.SortOrderAsc {
		userSortOrder = "asc"
	} else {
		userSortOrder = "desc"
	}

	requestBody["sort"] = []map[string]interface{}{
		{
			string(filter.Order): map[string]interface{}{
				"order": userSortOrder,
			},
		},
		{
			"id": map[string]interface{}{
				"order": "desc",
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化搜索请求失败: %w", err)
	}

	res, err := r.esClient.Search(
		r.esClient.Search.WithContext(ctx),
		r.esClient.Search.WithIndex(model.TableNameBookChapter),
		r.esClient.Search.WithBody(bytes.NewReader(jsonData)),
	)
	if err != nil {
		return nil, fmt.Errorf("执行搜索请求失败: %w", err)
	}
	defer res.Body.Close()

	// Note: The logger import is missing. Add it if needed.
	// logger.Info("Elasticsearch匹配语句: ", zap.Any("requestBody", requestBody))
	if res.IsError() {
		return nil, fmt.Errorf("ES搜索返回错误状态: %s", res.String())
	}

	var searchResponse struct {
		Hits struct {
			Hits []struct {
				Source    model.BookChapter   `json:"_source"`
				Highlight map[string][]string `json:"highlight"`
			} `json:"hits"`
		} `json:"hits"`
	}

	logger.Info("BookChapterElasticsearchSearch: ", zap.Any("requestBody", requestBody))
	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}

	results := make([]model.BookChapterSearchResult, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
		// 处理高亮信息
		highlightMap := make(map[string]string)
		for field, fragments := range hit.Highlight {
			if len(fragments) > 0 {
				highlightMap[field] = fragments[0]
			}
		}

		results = append(results, model.BookChapterSearchResult{
			BookChapter: hit.Source,
			Highlight:   highlightMap,
		})
	}

	return results, nil
}
