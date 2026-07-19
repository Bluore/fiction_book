package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fiction-book/internal/model"
	"fmt"

	"github.com/elastic/go-elasticsearch/v8"
)

type BookChapterESRepository interface {
	Create(ctx context.Context, chapter *model.BookChapter) error
	Update(ctx context.Context, chapter *model.BookChapter) error
	Delete(ctx context.Context, id string) error
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
