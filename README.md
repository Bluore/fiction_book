> 快速启动于 github.com/d60-Lab/gin-template

go run cmd/server/main.go

swag init -g cmd/server/main.go -o openapi

go run github.com/google/wire/cmd/wire ./...