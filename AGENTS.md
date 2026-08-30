# Fiction Book 项目协作规范

本文件适用于仓库根目录及全部子目录。修改代码前先阅读相关模块和测试；现有实现、`go.mod`、CI 与 `Makefile` 是项目事实来源。若文档互相冲突，优先级为：可执行配置（`go.mod`、CI）> 当前代码与测试 > `Makefile` > `README.md` 和其他说明文档。

## 项目概览

- 项目是小说阅读平台后端，项目名称简称为`GSF`，模块路径为 `fiction-book`，使用 Gin、GORM、Redis、Elasticsearch、Kafka、Wire 和 Swagger。
- Go 版本以 `go.mod` 和 CI 为准：Go 1.24，工具链为 Go 1.24.11。不要采用仅在 Go 1.25+ 可用的 API。
- 服务入口是 `cmd/server/main.go`；本地启动需要 `config/config.yaml` 以及相应的 MySQL、Redis、Elasticsearch、Kafka 等基础设施。
- `config/config.yaml` 是本地敏感配置并被忽略。新增配置项时同步更新配置结构和 `config/config.example.yaml`；不得提交真实密钥、令牌、DSN、私钥或用户数据。

## 目录与职责

- `cmd/server/`：应用启动、基础设施装配、服务生命周期和优雅关闭。
- `cmd/init_data/`、`cmd/test/`：初始化数据和开发辅助脚本，不承载线上业务逻辑。
- `internal/api/router/`：路由、API 版本、认证/验证中间件的组合。
- `internal/api/handler/`：HTTP 边界。读取已验证 DTO、取得认证信息、调用 service，并通过 `pkg/response` 返回统一响应；不要在这里直接访问数据库或实现核心业务。
- `internal/api/middleware/`：认证、校验、日志、恢复、限流和可观测性等横切逻辑。
- `internal/dto/`：请求与响应结构、JSON/query/URI 标签和校验规则。不要直接将 GORM model 暴露为 API 响应。
- `internal/service/`：业务规则、跨仓储编排、事务边界及缓存/MQ 协调。业务错误不在本层定义，统一引用 `pkg/response/errors.go` 中的 `response.ErrXxx`。
- `internal/repository/`：MySQL/GORM 和 Elasticsearch 持久化。沿用“接口 + 私有实现 + `New...` 构造器”的结构，并传递 `context.Context` 和调用方提供的 `*gorm.DB`，以支持事务。
- `internal/cache/`：Redis 访问和缓存策略，同样通过接口和构造器提供。
- `internal/model/`：持久化模型和领域数据结构。
- `internal/kafka/`：Kafka 客户端、消费者及消息处理；需要保证消费幂等，涉及数据库写入与事件发布时沿用 transactional outbox 设计。
- `internal/wire/`：依赖注入 provider 集合。手工修改 provider 和 `wire.go`，生成的 `wire_gen.go` 不得手工编辑。
- `pkg/`：可复用的基础能力。只有确实跨模块复用的代码才放入这里，业务代码留在 `internal/`。
- `migrations/`：数据库迁移 SQL。已应用的迁移不可为新需求原地改写；新增向前迁移，并同步相关 model 和 `doc/MODEL.md`。
- `openapi/`：Swagger 生成产物，不得手工编辑。

## 实现约定

- 遵循现有分层和依赖方向：router -> handler -> service -> repository/cache。禁止 handler 绕过 service，也不要让 repository 依赖 API 层。
- `context.Context` 放在方法参数首位，并从 `c.Request.Context()` 一直传递到数据库、缓存、搜索和消息调用；不要无故替换为 `context.Background()`。
- 多仓储原子写入由 service 使用 `db.Transaction` 组织，并将事务 `tx` 传给 repository。不要在 repository 中隐藏跨业务事务。
- 请求参数在 DTO 上声明 `json`、`form`、`uri` 和 `binding` 标签，在路由中挂载 `middleware.Validation`，handler 使用 `middleware.MustGetRequest[T]` 读取。
- 认证路由显式使用 `middleware.Auth(cfg)`；需要管理员权限时继续使用 `middleware.AdminOnly()`。不要只依赖 handler 内部的零散检查。
- handler 使用 `response.Success`、`response.PushError`、`response.BadRequest` 或 `response.InternalError`，保持项目现有的统一 JSON 响应协议。不要直接返回另一套响应结构，也不要把内部错误细节暴露给客户端。
- 可预期的业务失败统一使用 `pkg/response/errors.go` 中定义的 `*Err` 错误（`type Err struct { Code int; Msg string }`）：service 返回 `response.ErrXxx`，handler 通过 `response.PushError` 输出 `code`/`msg`。不要在 service 或 handler 中自行定义业务错误。新增业务错误时在 `pkg/response/errors.go` 追加 `ErrXxx = &Err{Code: N, Msg: "..."}`；错误码是兼容性契约，不得复用或修改既有码值，也不得随意删除。错误判断使用 `errors.Is`/`errors.As`（`PushError` 通过 `errors.As` 识别 `*Err`），底层错误需要附加上下文时使用 `%w` 包装，保证 `errors.As` 仍能识别出 `*Err`。非 `Err` 的内部错误（如内联 `errors.New`）不暴露给客户端，统一走 500。
- 数据库“未找到”沿用 repository 返回 `(nil, nil)`、service 转换为领域错误的现有约定；同一模块内不要混用另一种语义。
- 对缓存、搜索索引和 Kafka 的写入，先确认项目已有的一致性策略。不要把数据库成功但外部状态失败的情况静默吞掉。
- 新增依赖必须有明确收益；优先使用标准库和仓库已有依赖。变更依赖后运行 `go mod tidy` 并提交匹配的 `go.mod`、`go.sum`。

## Go 代码风格

- 所有 Go 文件必须通过 `gofmt`；使用 `goimports` 统一导入并将本模块导入归组。只格式化本次涉及的文件，避免无关改动。
- 保持函数职责单一，先处理错误和边界条件并尽早返回；避免无必要的 `else` 和深层嵌套。
- 非零初始化优先使用 `:=`，零值语义清晰时使用 `var`。复合字面量必须写字段名。
- 多行调用或签名按语义换行；参数较多时每行一个参数。不要为了行数压缩可读性，也不要为一次使用过早抽象。
- 导出范围保持最小。注释解释业务原因、并发/一致性约束或不直观行为，不复述代码。
- 延续已有命名，即使历史代码中存在 `Catch`/`MassageQueue` 等拼写。除非任务就是兼容性重命名，不要顺手做大范围更名。

## 新增或修改 API

一次完整的 API 变更按实际需要检查以下联动点：

1. 在 `internal/dto/` 定义请求、响应和校验标签。
2. 在 repository/cache 中扩展最小接口，并在 service 中实现业务和事务。
3. 在 handler 中添加方法及完整 Swagger 注解，包括路由、认证、参数、成功和失败响应。
4. 在 `internal/api/router/router.go` 的正确版本组中注册路由，并按需挂载 Validation/Auth/Admin 中间件。
5. 新增构造依赖时更新 `internal/wire/` 对应 provider set，然后重新生成 Wire。
6. 重新生成 Swagger，并在接口目录变化时更新 `doc/API.md`；数据库结构变化时新增 migration 并更新 `doc/MODEL.md`。
7. 如果明确要求测试接口：添加覆盖成功、校验失败、权限失败、未找到和关键业务错误的测试。

生成命令：

```bash
go run github.com/google/wire/cmd/wire ./...
swag init -g cmd/server/main.go -o openapi --parseDependency --parseInternal
```

生成后检查 diff，确保 `internal/wire/wire_gen.go` 与 `openapi/` 的变化确实来自本次源代码修改。

## 测试规范

如果明确要求进行测试才需要执行以下测试要求，没有要求不要测试

- 测试文件与源文件对应命名为 `foo_test.go`，放在被测包旁边。测试可观察行为和接口契约，不绑定私有实现细节。
- 多场景测试使用带 `name` 字段的表驱动测试和 `t.Run`。只有测试完全隔离、不共享全局状态、端口或数据库时才使用 `t.Parallel()`。
- handler 测试使用 `httptest`、Gin test mode 和 service mock；断言 HTTP 状态之外，还要验证统一响应中的业务 `code`、数据或错误消息。
- repository 测试沿用内存 SQLite；Redis 测试优先使用 `miniredis`。测试必须可独立运行、结果确定，并清理自己创建的状态。
- 使用 testify 时，`require` 负责无法继续的前置条件，`assert` 负责结果比较；mock 后调用 `AssertExpectations`。不要在父测试创建绑定父 `*testing.T` 的断言对象后复用于子测试。
- 修复缺陷时必须增加能在修复前失败的回归测试。涉及 goroutine、缓存并发或消息处理的代码至少运行 race detector，并避免依赖 `time.Sleep` 的脆弱时序断言。

## 验证命令

先运行最小范围测试，再运行与改动风险匹配的全量检查：

```bash
# 示例：聚焦单个包或测试
go test ./internal/service -run TestName

# CI 的核心检查
go test -race -coverprofile=coverage.txt -covermode=atomic ./...
go vet ./...
go build ./...
```

提交前至少保证改动文件已格式化、相关测试通过。若因为缺少外部服务、工具或平台限制未能运行某项检查，交付说明中必须明确列出未运行项和原因，不得声称已经验证。

## 依赖注入

先修改 `/internal/wire/` 里面的内容，不要直接修改 `wire_gen.go`

```bash
wire ./...
```

修改完成后执行命令用于完成依赖注入

## 改动纪律

- 保持改动聚焦，不重构无关代码，不覆盖工作区中已有的用户改动，不提交运行产生的上传文件、日志、覆盖率报告或本地配置。
- 不手工编辑 `internal/wire/wire_gen.go`、`openapi/docs.go`、`openapi/swagger.json` 或 `openapi/swagger.yaml`；修改其源定义后使用生成命令更新。
- 不随意改变既有 API 路径、JSON 字段、业务错误码、数据库列、Kafka topic 或缓存 key；这些都是兼容性契约。必要变更须同时处理迁移和兼容策略。
- 未经明确要求不要提交 Git commit。需要提交时，遵循仓库现有格式 `type(scope): description (agent assist)`；scope 可省略，正文只列主要改动。

## CheckList
- 开发中一定不要自行设置 $env:GOCACHE 和 $env:GOPATH
- 你的更改需要依照相似模块的更改，仿照相似模块的书写风格