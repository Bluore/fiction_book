package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

type BookESRepository interface {
	Create(ctx context.Context, book *model.Book) error
	Update(ctx context.Context, book *model.Book) error
	Delete(ctx context.Context, id string) error
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
