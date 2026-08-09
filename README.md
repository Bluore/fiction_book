# fiction_book

## 项目运行环境

- Golang >= 1.25.0
- MySQL >= 8.4.9
- Redis >= 8.6.3
- Elasticsearch >= 9.4.1
- Kafka >= 4.0.0

## 启动项目
go run cmd/server/main.go

## 生成swagger文档
swag init -g cmd/server/main.go -o openapi

## wire依赖注入
go run github.com/google/wire/cmd/wire ./...

## 文档

[API 文档](./doc/API.md)

[数据库模型文档](./doc/MODEL.md)

> 致谢 由[github.com/d60-Lab/gin-template](http://github.com/d60-Lab/gin-template)提供的项目模板
