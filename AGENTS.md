# Fiction Book 项目协作规范

本文件适用于仓库根目录及全部子目录。事实来源优先级：`go.mod`、`.github/workflows/ci.yml` > 当前代码与测试 > `Makefile` > `README.md`/`doc/`。修改前先读相关模块和测试。

## 快速上手

- 小说阅读平台后端，模块路径 `fiction-book`，入口 `cmd/server/main.go`；运行依赖 MySQL、Redis、Elasticsearch、Kafka。
- 版本：`go.mod` 和 CI 是 Go 1.24（toolchain 1.24.11）。当前开发机 go 更新（1.26.1），但不要使用 Go 1.25+ 才有的 API；README 写的 `>=1.25.0` 已过时。
- 配置由 viper 读取 `./config/config.yaml`（或当前目录），并合并 `./config/data.yaml` 的 `book_tags`。`config/config.yaml` 被忽略且含密钥，不得提交；修改配置结构时同步 `config/config.example.yaml`。
- 环境变量覆盖需要匹配 viper 键名（如 `MYSQL.HOST`）；`docker-compose.yml` 里的 `DB_HOST`/`REDIS_HOST` 等不会被读取。容器镜像实际是把 `config/config.docker.yaml` 复制成 `config/config.yaml`。
- 缺少 `config/config.yaml` 或基础设施时启动会失败/panic，不要当成代码问题。

常用命令：

```bash
go run cmd/server/main.go
go test ./internal/service -run TestName
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
go vet ./...
go build ./...
gofmt -l <changed files>
go run github.com/google/wire/cmd/wire ./...
swag init -g cmd/server/main.go -o openapi --parseDependency --parseInternal
```

- CI 只执行 `go build ./...`、`go test -race ...`、`gofmt -l`、`go vet ./...`，不执行 golangci-lint。当前开发机未安装 `golangci-lint`、`goimports`，不要假设 `make lint`/`make fmt` 可用。
- 不要自行设置 `GOCACHE`、`GOPATH`。

## 架构与目录

- 请求链路：`internal/api/router` -> `internal/api/handler` -> `internal/service` -> `internal/repository`/`internal/cache`。handler 不直接访问数据库，repository 不依赖 API 层。
- `cmd/server/main.go` 依次初始化配置/日志/Sentry/Tracing、`wire.InitializeApp()`、`app.Kafka.KafkaInit`、`app.Server.Start` 和 HTTP server，并处理优雅关闭。`cmd/init_data/`、`cmd/test/` 是初始化/辅助脚本，不承载线上逻辑。
- `internal/server`：定时轮询支付宝订单状态，处理超时、关单和补单；`internal/kafka`：outbox producer/consumer 与 feed consumer，均由 `main` 启动。改这部分注意 context 取消与消费幂等。
- 路由前缀为 `/api/v1`、`/api/v2`；健康检查 `/api/health`，Swagger `/swagger/index.html`，上传文件由 `/upload` 静态提供。
- `internal/dto`：请求/响应结构与校验标签；不要把 GORM model 直接作为 API 响应。
- `internal/wire/*.go`：手工维护 provider set（`InfraSet`、`RepositorySet`、`CatchSet`、`MassageQueueSet`、`ServiceSet`、`ServerSet`、`HandlerSet`）；`wire_gen.go` 由生成器产生。`Catch`/`Massage` 是历史拼写，勿顺手改名。
- `migrations/` 存放向前 SQL 迁移；`pkg/database` 里的 `AutoMigrate` 列表是显式维护的，受 `mysql.enable_auto_migrate` 控制。新增 model 时同时处理 migration 和该列表。
- `pkg/` 放跨模块基础能力（response/config/jwt/logger/payment/im/oss 等），业务代码留在 `internal/`。
- `openapi/` 是 Swagger 生成产物，不手工编辑。

## 实现约定

- 参数在 DTO 上声明 `json`/`form`/`uri`/`binding` 标签；路由挂 `middleware.Validation`，handler 用 `middleware.MustGetRequest[T]` 读取。缺少 Validation 时 `MustGetRequest` 会 panic。
- 认证路由显式挂 `middleware.Auth(cfg)`，管理员再挂 `middleware.AdminOnly()`。`/api/v1/pay/alipay` 是支付宝服务端回调，故意不挂 Auth，不要“修复”它。
- `context.Context` 放参数首位，从 `c.Request.Context()` 传到数据库/缓存/搜索/MQ；不要换成 `context.Background()`。
- 跨仓储原子写入在 service 用 `db.Transaction` 组织，并把 `tx` 传给 repository。写图书/章节时，outbox 事件与业务写入在同一事务内；不要先写库再单独发 Kafka。
- repository 查询未找到返回 `(nil, nil)`（现有约定），service 转成 `response.ErrXxx`。
- 统一响应：成功为 `{code:200, message:"success", data}`；错误也返回 HTTP 200，body 为 `{code:<业务码>, message:<错误文本>}`。字段是 `message`，不是 `msg`。测试必须解析 body 的 `code`，不能只看 HTTP 200。
- 业务错误只在 `pkg/response/errors.go` 定义 `ErrXxx = &Err{Code, Msg}`；service 返回或用 `%w` 包装，handler 用 `response.PushError`。错误码是兼容性契约，不得复用、修改或随意删除；非 `Err` 内部错误统一走 body code 500，且不暴露细节。
- 不随意改变既有 API 路径、JSON 字段、数据库列、Kafka topic 或缓存 key（缓存 key 使用 `GSF:` 前缀）。
- 延续现有命名和写法（包括 `Catch`/`Massage`/`OrderTemp` 等历史拼写）；改动模仿相邻同类型模块。
- 所有 Go 文件通过 `gofmt`；本模块导入 `fiction-book/...` 单独归组。只格式化本次涉及的文件。

## 测试

仅在任务明确要求时补充或运行测试。

- repository/service/cache 测试使用内存 SQLite（`gorm.io/driver/sqlite`）和 `miniredis`，沿用现有 testify suite 风格。
- service/handler 的 mock 是手写的 `testify/mock`（见 `internal/service/service_mocks_test.go`、handler 测试），没有 mock 生成器。
- handler 测试使用 `gin.TestMode` + `httptest`；需在 `TestMain`/`init` 中调用 `logger.Init` 和 `validator.Init`，否则全局日志/校验不可用。
- 修复缺陷时补一个修复前会失败的回归测试；涉及 goroutine/缓存/MQ 时跑 `-race`，不要依赖 `time.Sleep` 的时序断言。

## 新增或修改 API

1. `internal/dto/` 加请求、响应和校验标签。
2. repository/cache 扩最小接口，service 实现业务与事务。
3. handler 加方法及完整 Swagger 注解。
4. `internal/api/router/router.go` 在对应 v1/v2 分组注册，按需挂 Validation/Auth/Admin。
5. 新依赖时更新 `internal/wire/` provider set，再生成 Wire。
6. 重新生成 Swagger；接口目录变化更新 `doc/API.md`，表结构变化加 migration 并更新 `doc/MODEL.md`。
7. 明确要求测试时，覆盖成功、校验失败、权限失败、未找到和关键业务错误。

生成命令见上文。生成后检查 diff，确认 `internal/wire/wire_gen.go` 和 `openapi/` 的变化只来自本次源码。

## 改动纪律

- 保持改动聚焦，不重构无关代码，不覆盖工作区已有的用户改动。
- 不手工编辑 `internal/wire/wire_gen.go`、`openapi/docs.go`、`openapi/swagger.json`、`openapi/swagger.yaml`。
- 不提交 `config/config.yaml`、真实密钥/DSN、日志、覆盖率报告、`data/` 或运行产生的上传文件。
- 未经明确要求不要提交 Git commit。需要提交时遵循 `.trae/rules/git-commit-message.md`：首行 `type(scope): description (agent assist)`，正文用 `- ` 列主要改动，尽量使用代码中的字段名。
