package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"fiction-book/internal/model"
	"fiction-book/pkg/config"
	"fiction-book/pkg/database"
	"fiction-book/pkg/logger"

	"github.com/elastic/go-elasticsearch/v8"
	"go.uber.org/zap"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 初始化日志
	if err := logger.Init(cfg.Server.Mode); err != nil {
		panic(fmt.Sprintf("Failed to init logger: %v", err))
	}
	defer logger.Sync()

	logger.Info("Starting index initialization...")

	// 初始化 Elasticsearch 连接
	esClient, err := database.InitES(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize Elasticsearch client", zap.Error(err))
	}

	// 创建 Book 索引
	if err := createBookIndex(esClient); err != nil {
		logger.Fatal("Failed to create book index", zap.Error(err))
	}

	// 创建 BookChapter 索引
	if err := createBookChapterIndex(esClient); err != nil {
		logger.Fatal("Failed to create book chapter index", zap.Error(err))
	}

	logger.Info("Index initialization completed successfully")
}

// createBookIndex 创建 Book 索引
func createBookIndex(esClient *elasticsearch.Client) error {
	// 检查索引是否已存在
	existsRes, err := esClient.Indices.Exists([]string{model.TableNameBook})
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode != 404 {
		logger.Info("Index already exists, skipping creation", zap.String("index", model.TableNameBook))
		return nil
	}

	// 定义索引映射
	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   3,
			"number_of_replicas": 1,
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"user_id": map[string]interface{}{
					"type": "keyword",
				},
				"username": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max_word",
					"search_analyzer": "ik_smart",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type":         "keyword",
							"ignore_above": 256,
						},
					},
				},
				"description": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max_word",
					"search_analyzer": "ik_smart",
				},
				"vip_level": map[string]interface{}{
					"type": "keyword",
				},
				"price": map[string]interface{}{
					"type": "long",
				},
				"cover_image": map[string]interface{}{
					"type":  "keyword",
					"index": false,
				},
				"browse_amount": map[string]interface{}{
					"type": "long",
				},
				"bookshelf_amount": map[string]interface{}{
					"type": "long",
				},
				"pushed_at": map[string]interface{}{
					"type": "date",
				},
				"thermal_value": map[string]interface{}{
					"type": "long",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	// 将映射序列化为 JSON
	jsonBody, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// 创建索引
	createRes, err := esClient.Indices.Create(
		model.TableNameBook,
		esClient.Indices.Create.WithBody(bytes.NewReader(jsonBody)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("elasticsearch error: %s", createRes.String())
	}

	logger.Info("Index created successfully", zap.String("index", model.TableNameBook))
	return nil
}

// createBookChapterIndex 创建 BookChapter 索引
func createBookChapterIndex(esClient *elasticsearch.Client) error {
	// 检查索引是否已存在
	existsRes, err := esClient.Indices.Exists([]string{model.TableNameBookChapter})
	if err != nil {
		return fmt.Errorf("failed to check if index exists: %w", err)
	}
	defer existsRes.Body.Close()

	if existsRes.StatusCode != 404 {
		logger.Info("Index already exists, skipping creation", zap.String("index", model.TableNameBookChapter))
		return nil
	}

	// 定义索引映射
	mapping := map[string]interface{}{
		"settings": map[string]interface{}{
			"number_of_shards":   3,
			"number_of_replicas": 1,
		},
		"mappings": map[string]interface{}{
			"properties": map[string]interface{}{
				"id": map[string]interface{}{
					"type": "keyword",
				},
				"book_id": map[string]interface{}{
					"type": "keyword",
				},
				"title": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max_word",
					"search_analyzer": "ik_smart",
					"fields": map[string]interface{}{
						"keyword": map[string]interface{}{
							"type":         "keyword",
							"ignore_above": 256,
						},
					},
				},
				"content": map[string]interface{}{
					"type":            "text",
					"analyzer":        "ik_max_word",
					"search_analyzer": "ik_smart",
				},
				"order": map[string]interface{}{
					"type": "integer",
				},
				"vip_level": map[string]interface{}{
					"type": "keyword",
				},
				"price": map[string]interface{}{
					"type": "long",
				},
				"created_at": map[string]interface{}{
					"type": "date",
				},
				"updated_at": map[string]interface{}{
					"type": "date",
				},
			},
		},
	}

	// 将映射序列化为 JSON
	jsonBody, err := json.Marshal(mapping)
	if err != nil {
		return fmt.Errorf("failed to marshal mapping: %w", err)
	}

	// 创建索引
	createRes, err := esClient.Indices.Create(
		model.TableNameBookChapter,
		esClient.Indices.Create.WithBody(bytes.NewReader(jsonBody)),
	)
	if err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}
	defer createRes.Body.Close()

	if createRes.IsError() {
		return fmt.Errorf("elasticsearch error: %s", createRes.String())
	}

	logger.Info("Index created successfully", zap.String("index", model.TableNameBookChapter))
	return nil
}
