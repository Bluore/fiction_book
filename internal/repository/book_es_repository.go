package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fiction-book/pkg/logger"
	"fmt"
	"strconv"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

type BookESRepository interface {
	Create(ctx context.Context, book *model.Book) error
	Update(ctx context.Context, book *model.Book) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter *model.BookFilterWithIDPage) (*model.BookSearchPage, error)
}

type bookESRepository struct {
	esClient *elasticsearch.Client
}

// NewBookESRepository 创建用户仓储实例
func NewBookESRepository(esClient *elasticsearch.Client) BookESRepository {
	return &bookESRepository{esClient: esClient}
}

// Create 创建图书ES
func (r *bookESRepository) Create(ctx context.Context, book *model.Book) error {
	// 将图书模型序列化为JSON
	jsonData, err := json.Marshal(book)
	if err != nil {
		return fmt.Errorf("序列化图书数据失败: %w", err)
	}

	// 创建ES索引请求
	res, err := r.esClient.Index(
		model.TableNameBook,
		bytes.NewReader(jsonData),
		r.esClient.Index.WithDocumentID(book.ID),
		r.esClient.Index.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("创建图书ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	// 检查ES响应状态
	if res.IsError() {
		return fmt.Errorf("ES创建文档返回错误状态: %s", res.Status())
	}

	return nil
}

// Update 更新图书ES
func (r *bookESRepository) Update(ctx context.Context, book *model.Book) error {
	// 构造更新文档，使用doc和doc_as_upsert，实现如果文档不存在则创建
	updateDoc := map[string]interface{}{
		"doc":           book,
		"doc_as_upsert": true,
	}

	// 序列化更新数据
	jsonData, err := json.Marshal(updateDoc)
	if err != nil {
		return fmt.Errorf("序列化图书更新数据失败: %w", err)
	}

	// 创建ES更新请求
	res, err := r.esClient.Update(
		model.TableNameBook,
		book.ID,
		bytes.NewReader(jsonData),
		r.esClient.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("更新图书ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	// 检查ES响应状态
	if res.IsError() {
		return fmt.Errorf("ES更新文档返回错误状态: %s", res.Status())
	}

	return nil
}

// Delete 删除图书ES
func (r *bookESRepository) Delete(ctx context.Context, id string) error {
	// 创建ES删除请求
	res, err := r.esClient.Delete(
		model.TableNameBook,
		id,
		r.esClient.Delete.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("删除图书ES文档请求失败: %w", err)
	}
	defer res.Body.Close()

	// 检查ES响应状态
	if res.IsError() {
		// 如果是404错误（文档不存在），则忽略该错误，因为删除操作幂等
		if res.StatusCode == 404 {
			return nil
		}
		return fmt.Errorf("ES删除文档返回错误状态: %s", res.Status())
	}

	return nil
}

// Search 搜索图书ES
func (r *bookESRepository) Search(ctx context.Context, filter *model.BookFilterWithIDPage) (*model.BookSearchPage, error) {
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
					"fields": []string{"name", "description"},
					"type":   "best_fields",
				},
			},
		)
	}

	// 处理 VIP 等级过滤：多个 VIP 等级之间为或关系（vip_1 或 vip_2）
	if len(filter.VIPLevel) > 0 {
		vipShould := make([]interface{}, 0, len(filter.VIPLevel))
		for _, vipLevel := range filter.VIPLevel {
			vipShould = append(vipShould, map[string]interface{}{
				"term": map[string]interface{}{
					"vip_level": vipLevel,
				},
			})
		}

		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"bool": map[string]interface{}{
					"should":               vipShould,
					"minimum_should_match": 1,
				},
			},
		)
	}

	// 处理 PushedAt 范围过滤
	if !filter.PushedAtStart.IsZero() || !filter.PushedAtEnd.IsZero() {
		pushedAtRange := map[string]interface{}{}
		if !filter.PushedAtStart.IsZero() {
			pushedAtRange["gte"] = filter.PushedAtStart
		}
		if !filter.PushedAtEnd.IsZero() {
			pushedAtRange["lte"] = filter.PushedAtEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"pushed_at": pushedAtRange,
				},
			},
		)
	}

	// 处理 UpdatedAt 范围过滤
	if !filter.UpdatedAtStart.IsZero() || !filter.UpdatedAtEnd.IsZero() {
		updatedAtRange := map[string]interface{}{}
		if !filter.UpdatedAtStart.IsZero() {
			updatedAtRange["gte"] = filter.UpdatedAtStart
		}
		if !filter.UpdatedAtEnd.IsZero() {
			updatedAtRange["lte"] = filter.UpdatedAtEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"updated_at": updatedAtRange,
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

	// 处理 OwnerID 过滤
	if filter.OwnerID != "" {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"term": map[string]interface{}{
					"user_id": filter.OwnerID,
				},
			},
		)
	}

	// 处理 BookshelfAmount 范围过滤
	if filter.BookshelfAmountStart > 0 || filter.BookshelfAmountEnd > 0 {
		bookshelfRange := map[string]interface{}{}
		if filter.BookshelfAmountStart > 0 {
			bookshelfRange["gte"] = filter.BookshelfAmountStart
		}
		if filter.BookshelfAmountEnd > 0 {
			bookshelfRange["lte"] = filter.BookshelfAmountEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"bookshelf_amount": bookshelfRange,
				},
			},
		)
	}

	// 处理 ThermalValue 范围过滤
	if filter.ThermalValueStart > 0 || filter.ThermalValueEnd > 0 {
		thermalRange := map[string]interface{}{}
		if filter.ThermalValueStart > 0 {
			thermalRange["gte"] = filter.ThermalValueStart
		}
		if filter.ThermalValueEnd > 0 {
			thermalRange["lte"] = filter.ThermalValueEnd
		}
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"range": map[string]interface{}{
					"thermal_value": thermalRange,
				},
			},
		)
	}

	// 处理游标筛选：(pushed_at, id) 严格小于游标，与固定排序保持一致
	if filter.Cursor != nil {
		query["bool"].(map[string]interface{})["must"] = append(
			query["bool"].(map[string]interface{})["must"].([]interface{}),
			map[string]interface{}{
				"bool": map[string]interface{}{
					"should": []interface{}{
						map[string]interface{}{
							"range": map[string]interface{}{
								"pushed_at": map[string]interface{}{
									"lt": filter.Cursor.TimeMS,
								},
							},
						},
						map[string]interface{}{
							"bool": map[string]interface{}{
								"must": []interface{}{
									map[string]interface{}{
										"range": map[string]interface{}{
											"pushed_at": map[string]interface{}{
												"gte": filter.Cursor.TimeMS,
												"lte": filter.Cursor.TimeMS,
											},
										},
									},
									map[string]interface{}{
										"range": map[string]interface{}{
											"id": map[string]interface{}{
												"lt": filter.Cursor.ID,
											},
										},
									},
								},
							},
						},
					},
					"minimum_should_match": 1,
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
				"name":        map[string]interface{}{},
				"description": map[string]interface{}{},
			},
		}
	}

	// 固定排序：先按发布时间倒序，再以 ID 降序作为唯一 tie-breaker
	requestBody["sort"] = []map[string]interface{}{
		{
			"pushed_at": map[string]interface{}{
				"order": "desc",
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
		r.esClient.Search.WithIndex(model.TableNameBook),
		r.esClient.Search.WithBody(bytes.NewReader(jsonData)),
	)
	if err != nil {
		return nil, fmt.Errorf("执行搜索请求失败: %w", err)
	}
	defer res.Body.Close()

	logger.Info("Elasticsearch匹配语句: ", zap.Any("requestBody", requestBody))
	if res.IsError() {
		return nil, fmt.Errorf("ES搜索返回错误状态: %s", res.String())
	}

	var searchResponse struct {
		Hits struct {
			Hits []struct {
				Source    model.Book          `json:"_source"`
				Highlight map[string][]string `json:"highlight"`
				Sort      []json.RawMessage   `json:"sort"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}

	hits := searchResponse.Hits.Hits
	results := make([]model.BookSearchResult, 0, len(hits))
	for _, hit := range hits {
		// 处理高亮信息
		highlightMap := make(map[string]string)
		for field, fragments := range hit.Highlight {
			if len(fragments) > 0 {
				highlightMap[field] = fragments[0]
			}
		}

		results = append(results, model.BookSearchResult{
			Book:      hit.Source,
			Highlight: highlightMap,
		})
	}

	page := &model.BookSearchPage{Results: results}
	// 只请求 limit 条：满页时用最后一条命中记录的 hit.sort 作为下一页游标
	if filter.Limit > 0 && len(hits) == filter.Limit {
		cursor, err := parseSearchCursor(hits[len(hits)-1].Sort)
		if err != nil {
			return nil, fmt.Errorf("解析书籍搜索游标失败: %w", err)
		}
		page.Next = cursor
	}

	return page, nil
}

// parseSearchCursor 从 ES 命中结果的 sort 数组解析 (时间, id) 游标。
// 时间字段默认返回 epoch 毫秒（JSON number），同时兼容字符串毫秒和 RFC3339 时间。
func parseSearchCursor(sortValues []json.RawMessage) (*model.SearchCursor, error) {
	if len(sortValues) < 2 {
		return nil, fmt.Errorf("hit.sort 缺少排序值，实际长度 %d", len(sortValues))
	}

	timeMS, err := parseSortTimeMS(sortValues[0])
	if err != nil {
		return nil, err
	}

	var id string
	if err := json.Unmarshal(sortValues[1], &id); err != nil {
		return nil, fmt.Errorf("解析 hit.sort 的 id 失败: %w", err)
	}

	return &model.SearchCursor{TimeMS: timeMS, ID: id}, nil
}

func parseSortTimeMS(raw json.RawMessage) (int64, error) {
	var number float64
	if err := json.Unmarshal(raw, &number); err == nil {
		return int64(number), nil
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, fmt.Errorf("解析 hit.sort 的时间失败: %w", err)
	}

	if ms, err := strconv.ParseInt(text, 10, 64); err == nil {
		return ms, nil
	}

	parsed, err := time.Parse(time.RFC3339Nano, text)
	if err != nil {
		return 0, fmt.Errorf("解析 hit.sort 的时间失败: %w", err)
	}
	return parsed.UnixMilli(), nil
}
