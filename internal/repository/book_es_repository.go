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

type BookESRepository interface {
	Create(ctx context.Context, book *model.Book) error
	Update(ctx context.Context, book *model.Book) error
	Delete(ctx context.Context, id string) error
	Search(ctx context.Context, filter *model.BookFilterWithIDPage) ([]model.BookSearchResult, error)
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
func (r *bookESRepository) Search(ctx context.Context, filter *model.BookFilterWithIDPage) ([]model.BookSearchResult, error) {
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

	// 处理 LastID 筛选：ID 比 LastID 小的结果
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
				"name":        map[string]interface{}{},
				"description": map[string]interface{}{},
			},
		}
	}

	// 处理排序：先按用户指定的字段排序，最后固定以ID降序排序
	if filter.Order == "" {
		filter.Order = model.BookOrderUpdate
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
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(res.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("解析搜索响应失败: %w", err)
	}

	results := make([]model.BookSearchResult, 0, len(searchResponse.Hits.Hits))
	for _, hit := range searchResponse.Hits.Hits {
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

	return results, nil
}
