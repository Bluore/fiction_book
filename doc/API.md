# API 接口文档

小说阅读平台后端 HTTP 接口说明。本文档以当前路由、DTO、Handler 与 Service 实现为准。

- 在线调试：`/swagger/index.html`
- 健康检查：`GET /api/health`
- 接口前缀：`/api/v1`、`/api/v2`

---

## 1. 通用约定

### 1.1 Base URL

```text
http://localhost:8080
```

生产环境以实际域名与 `server.website` 配置为准。

### 1.2 认证

需要登录的接口使用 JWT：

```http
Authorization: Bearer <AccessToken>
```

- 登录成功后返回 `token`（AccessToken）与 `refresh_token`（RefreshToken）。
- AccessToken 过期时返回业务码 `4027`，客户端应调用 `POST /api/v1/auth/refresh` 换取新 Token。
- RefreshToken 仅能用于刷新，不能访问业务接口；同一用户只保留最新一条 RefreshToken（单点登录）。
- 缺失或非法 Token 返回业务码 `401`；管理员接口还要求 Token 中 `role=admin`，否则返回 `403`。

### 1.3 响应格式

除支付宝回调外，所有 JSON 接口均返回 **HTTP 200**，业务结果由响应体 `code` 判断。

成功：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

失败：

```json
{
  "code": 4001,
  "message": "书籍未找到"
}
```

- `data` 在无返回内容时省略。
- 参数校验失败返回 `code=400`，`message` 形如 `参数校验失败: 字段 Name required`。
- 认证失败返回 `code=401`，权限不足返回 `code=403`。
- 未归类的内部错误统一返回 `code=500`，不暴露内部细节。
- 支付宝异步通知为特例：`Content-Type: text/plain`，成功返回 `success`，失败返回 `failure`。

### 1.4 分页与时间

| 场景 | 参数 | 约定 |
| --- | --- | --- |
| 页码分页 | `page`、`size` | `page` 从 1 开始；`size` 默认 10，最大 100 |
| ID 游标分页 | `last_id`、`limit` | 传上一页最后一条记录的 ID；`limit` 默认 10，最大 100 |
| Feed 游标 | `last_cursor_id`、`target_cursor_id`、`offset` | 首次请求 `target_cursor_id=front`、`offset=0`；之后使用响应中的 `next_cursor_id`、`next_offset` |

时间格式：

- 查询过滤中的时间参数：Unix 秒。
- 书籍、Feed 响应中的时间戳：Unix 秒。
- 用户资料时间：`YYYY-MM-DD HH:mm:ss`。
- 订单、购买记录、VIP 到期时间：RFC3339。
- 创建书籍请求中的 `push_at`：RFC3339。

### 1.5 枚举

| 枚举 | 取值 | 说明 |
| --- | --- | --- |
| `vip_level` | `""`、`vip_0`、`vip_1`、`vip_2` | 空字符串表示免费；当前售卖 `vip_1`、`vip_2` |
| 书籍排序 `sort_key` | `browse_amount`、`bookshelf_amount`、`new_update`、`new_push`、`thermal` | 默认 `new_update` |
| 排序方向 | `asc`、`desc` | 默认 `desc` |
| 书籍搜索排序 `order` | `thermal_value`、`browse_amount`、`bookshelf_amount`、`updated_at`、`pushed_at` | 默认 `thermal_value` |
| 章节搜索排序 `order` | `created_at`、`price` | 默认 `created_at` |
| 订单状态 `status` | `1` 待支付、`2` 待发货、`3` 已成功、`4` 已删除、`5` 已超时 | 当前业务使用 1 / 3 / 5 |
| 关注操作 `operator` | `create`、`delete` | 关注 / 取消关注 |
| VIP 时长 `duration` | `month`、`year` | 月 / 年 |

### 1.6 业务错误码

| code | message | 说明 |
| --- | --- | --- |
| 4001 | 书籍未找到 | |
| 4002 | 用户未找到 | |
| 4003 | 验证码验证失败 | |
| 4004 | 请先发送邮箱验证码 | |
| 4005 | 文件大小超出预期 | |
| 4006 | 用户名或密码错误 | |
| 4007 | 用户已存在 | |
| 4008 | 书籍已存在 | |
| 4009 | 书籍已购买 | 书籍 / 章节重复购买 |
| 4010 | 章节未找到 | |
| 4011 | 文件类型出错 | |
| 4012 | 图片出错 | 尺寸或内容不符合要求 |
| 4013 | 邮箱已存在 | |
| 4014 | 请先完成登录 | 游客访问受限章节 |
| 4015 | 会员等级不足 | |
| 4016 | 没有购买此书籍的记录 | |
| 4017 | 参数错误 | 价格校验失败、重复参数等 |
| 4018 | 余额不足 | |
| 4020 | 缺失对应权限 | 非作者操作书籍 / 章节 |
| 4021 | 订单没找到 | |
| 4022 | 重复的请求 | `request_uuid` 重复 |
| 4023 | 没有购买此章节的记录 | |
| 4024 | 创建支付宝订单失败 | |
| 4025 | 支付宝回调验签失败 | |
| 4026 | 订单金额不匹配 | |
| 4027 | 登录已过期，请刷新 token | AccessToken 过期 |

---

## 2. 接口总览

| 模块 | 方法 | 路径 | 认证 |
| --- | --- | --- | --- |
| 系统 | GET | `/api/health` | 否 |
| 认证 | POST | `/api/v1/auth/login` | 否 |
| 认证 | POST | `/api/v1/auth/refresh` | 否 |
| 认证 | POST | `/api/v1/auth/email/verify` | 否 |
| 认证 | POST | `/api/v1/auth/forget/password` | 否 |
| 用户 | POST | `/api/v1/users` | 否 |
| 用户 | GET | `/api/v1/users` | 否 |
| 用户 | GET | `/api/v1/users/{id}` | 否 |
| 用户 | GET | `/api/v1/users/info` | 是 |
| 用户 | PUT | `/api/v1/users/information` | 是 |
| 用户 | PUT | `/api/v1/users/password` | 见 5.7 说明 |
| 用户 | PUT | `/api/v1/users/email` | 是 |
| 用户 | POST | `/api/v1/users/avatar` | 是 |
| 用户 | DELETE | `/api/v1/users/delete` | 是 + 管理员 |
| 书籍 | POST | `/api/v1/books/create` | 是 |
| 书籍 | GET | `/api/v1/books/list` | 否 |
| 书籍 | GET | `/api/v1/books/{id}` | 否 |
| 书籍 | PUT | `/api/v1/books/{id}/info` | 是 |
| 书籍 | POST | `/api/v1/books/{id}/cover` | 是 |
| 书籍 | GET | `/api/v1/books/{id}/chapters/list` | 否 |
| 书籍 | GET | `/api/v1/books/own/list` | 是 |
| 书籍 | GET | `/api/v1/books/history/list` | 是 |
| 书籍 | GET | `/api/v1/books/thermal/daily` | 否 |
| 书籍 | GET | `/api/v1/books/thermal/weekly` | 否 |
| 书籍 | GET | `/api/v1/books/thermal/monthly` | 否 |
| 书籍 | GET | `/api/v2/books/search` | 否 |
| 书籍 | GET | `/api/v2/books/{id}/tags` | 否 |
| 书籍 | PUT | `/api/v2/books/{id}/tags` | 是 |
| 章节 | POST | `/api/v1/chapters/create` | 是 |
| 章节 | GET | `/api/v1/chapters/{id}/guest` | 否 |
| 章节 | GET | `/api/v1/chapters/{id}` | 是 |
| 章节 | GET | `/api/v1/chapters/{id}/info` | 否 |
| 章节 | PUT | `/api/v1/chapters/{id}` | 是 |
| 章节 | DELETE | `/api/v1/chapters/{id}` | 是 |
| 章节 | GET | `/api/v2/chapters/search` | 否 |
| 消费 | POST | `/api/v1/buy/book` | 是 |
| 消费 | POST | `/api/v1/buy/chapter` | 是 |
| 资产 | GET | `/api/v1/assets` | 是 |
| 资产 | GET | `/api/v1/purchases/book` | 是 |
| 资产 | GET | `/api/v1/purchases/book/{book_id}` | 是 |
| 资产 | GET | `/api/v1/purchases/chapters` | 是 |
| 资产 | GET | `/api/v1/purchases/chapters/{chapter_id}` | 是 |
| 消费 | GET | `/api/v2/purchases/vip/shop` | 是 |
| 消费 | POST | `/api/v2/purchases/vip` | 是 |
| 消费 | POST | `/api/v2/purchases/vip/upgrade` | 是 |
| 书架 | GET | `/api/v1/bookshelf/list` | 是 |
| 书架 | GET | `/api/v1/bookshelf/{user_id}` | 否 |
| 书架 | POST | `/api/v1/bookshelf/add` | 是 |
| 书架 | POST | `/api/v1/bookshelf/delete` | 是 |
| 历史 | GET | `/api/v1/history/{user_id}` | 否（已弃用） |
| 关注 | GET | `/api/v1/likes` | 是 |
| 关注 | PUT | `/api/v1/likes` | 是 |
| Feed | GET | `/api/v1/feed/list` | 是（已弃用） |
| Feed | POST | `/api/v2/feed` | 是 |
| Feed | POST | `/api/v2/feed/guest` | 否 |
| 订单 | GET | `/api/v1/orders/{id}` | 是 |
| 订单 | GET | `/api/v1/orders` | 是 |
| 订单 | POST | `/api/v1/orders/create/gold` | 是 |
| 支付 | POST | `/api/v1/pay/alipay` | 否（支付宝服务端） |

---

## 3. 系统

### 3.1 健康检查

```http
GET /api/health
```

**响应 `data`**

```json
{ "status": "ok" }
```

---

## 4. 认证

### 4.1 登录

```http
POST /api/v1/auth/login
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `username` | string | 是 | 用户名 |
| `password` | string | 是 | 密码 |

```json
{
  "username": "reader",
  "password": "123456"
}
```

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `token` | string | AccessToken |
| `refresh_token` | string | RefreshToken |
| `expires_in` | int | AccessToken 有效期（秒） |
| `user_id` | string | 用户 ID |
| `username` | string | 用户名 |

**说明**

- 用户名不存在与密码错误统一返回 `4006`。
- 每次登录都会覆盖该用户的 RefreshToken 白名单，旧 RefreshToken 立即失效。

### 4.2 刷新 Token

```http
POST /api/v1/auth/refresh
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `refresh_token` | string | 是 | 登录返回的 RefreshToken |

**响应 `data`**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `token` | string | 新的 AccessToken |
| `refresh_token` | string | 新的 RefreshToken |
| `expires_in` | int | AccessToken 有效期（秒） |

**说明**

- RefreshToken 不在 Redis 白名单中时返回 `4014`；已过期返回 `4027`。
- 刷新后旧 RefreshToken 失效。

### 4.3 发送邮箱验证码

```http
POST /api/v1/auth/email/verify
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `email` | string | 是 | 接收验证码的邮箱 |

**响应 `data`**

```json
{ "email": "user@example.com" }
```

**说明**

- 验证码为 6 位数字，Redis 保存 5 分钟，一次性使用。
- 需要配置 `resend.from` 与 `resend.api_key`。

### 4.4 忘记密码

```http
POST /api/v1/auth/forget/password
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `password` | string | 是 | 新密码，至少 6 位 |
| `email` | string | 是 | 账号邮箱 |
| `email_verify` | string | 是 | 6 位邮箱验证码 |

**响应 `data`**：`null`

**说明**

- 需先调用 4.3 获取验证码。
- 验证码错误返回 `4003`，未发送验证码返回 `4004`，用户不存在返回 `4002`。

---

## 5. 用户

### 5.1 注册

```http
POST /api/v1/users
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `username` | string | 是 | 至少 3 个字符，不能包含空格 |
| `email` | string | 是 | 邮箱 |
| `password` | string | 是 | 至少 6 位 |
| `age` | int | 否 | 0~130 |
| `verify` | string | 否 | 6 位邮箱验证码；非 `debug` 模式实际必填 |

**响应 `data`**：[User](#a1-user)

**说明**

- 注册成功会在同一事务内创建 `user`、`user_info`、`user_asset`。
- `debug` 模式下用户名为 `admin` 时角色自动设为 `admin`。
- 用户名已存在返回 `4007`，邮箱已存在返回 `4013`，验证码错误返回 `4003` / `4004`。

### 5.2 用户列表

```http
GET /api/v1/users?page=1&page_size=10
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 否 | 页码，默认 1 |
| `page_size` | int | 否 | 每页数量，默认 10，最大 100 |

**响应 `data`**

```json
{
  "page": 1,
  "page_size": 10,
  "list": [ /* User */ ]
}
```

### 5.3 获取用户信息

```http
GET /api/v1/users/{id}
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 用户 ID |

**响应 `data`**：[User](#a1-user)

### 5.4 获取当前用户详情

```http
GET /api/v1/users/info
Authorization: Bearer <token>
```

**响应 `data`**：[UserStat](#a2-userstat)

包含用户基础资料、金币、VIP 状态与阅读统计。

### 5.5 更新当前用户资料

```http
PUT /api/v1/users/information
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**（均为可选，只更新传入字段）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `username` | string | 至少 3 个字符，不能包含空格 |
| `age` | int | 0~130 |
| `description` | string | 个人简介，非空 |

**响应 `data`**：[User](#a1-user)

### 5.6 更新当前用户邮箱

```http
PUT /api/v1/users/email
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `password` | string | 是 | 当前密码 |
| `email` | string | 是 | 新邮箱 |
| `verify` | string | 是 | 新邮箱的 6 位验证码 |

**响应 `data`**：[User](#a1-user)

**说明**

- 新邮箱已存在返回 `4013`；密码错误返回 `4006`；验证码错误返回 `4003` / `4004`。

### 5.7 更新当前用户密码

```http
PUT /api/v1/users/password
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `password` | string | 是 | 新密码，至少 6 位 |
| `verify` | string | 是 | 邮箱验证码 |

**响应 `data`**：[User](#a1-user)

> 已知问题：该路由当前未挂载 `Auth` 中间件，但处理器从登录上下文读取 `userID`。未经认证链路调用会返回 `4017`。修复认证配置前该接口不可正常使用。

### 5.8 上传头像

```http
POST /api/v1/users/avatar
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**表单字段**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `avatar` | file | 是 | PNG / JPG，尺寸必须为 180×180，≤ 2 MB |

**响应 `data`**：[User](#a1-user)，`profile_image` 为绝对 URL。

**相关错误**

- `4005` 文件过大
- `4011` 文件类型不符
- `4012` 图片尺寸不符

### 5.9 删除当前用户

```http
DELETE /api/v1/users/delete
Authorization: Bearer <token>   # role 必须为 admin
```

**请求体**：无（请求体不参与删除目标选择）

**响应 `data`**：`null`

**说明**

- 删除的是 **当前 Token 对应的用户**，不是请求体中的用户。
- 仅管理员可调用；非管理员返回 `403`。

---

## 6. 书籍

### 6.1 创建书籍

```http
POST /api/v1/books/create
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `name` | string | 是 | 书名，全局唯一，最长 50 |
| `description` | string | 是 | 简介 |
| `cover_image` | string | 否 | 封面路径，可后续上传 |
| `vip_level` | string | 否 | 阅读所需 VIP 等级 |
| `price` | int64 | 否 | 书籍价格（金币） |
| `push_at` | string | 否 | 发布时间，RFC3339；不传则取当前时间 |

**响应 `data`**：[Book](#a3-book)

**说明**

- 同名书籍存在返回 `4008`。
- 创建成功后写入 Feed 与 Outbox，并累加作者的 `read_book_num`。

### 6.2 书籍列表

```http
GET /api/v1/books/list?page=1&size=10&sort_key=new_update&sort_order=desc
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 否 | 页码，默认 1 |
| `size` | int | 否 | 每页数量，默认 10，最大 100 |
| `sort_key` | string | 否 | `browse_amount` / `bookshelf_amount` / `new_update` / `new_push` / `thermal`，默认 `new_update` |
| `sort_order` | string | 否 | `asc` / `desc`，默认 `desc` |
| `push_start_time` | int | 否 | 发布时间起（Unix 秒） |
| `push_end_time` | int | 否 | 发布时间止（Unix 秒） |

**响应 `data`**：[BookList](#a4-booklist)

### 6.3 获取书籍详情

```http
GET /api/v1/books/{id}
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 书籍 ID |

**响应 `data`**：[Book](#a3-book)

书籍不存在返回 `4001`。

### 6.4 更新书籍信息

```http
PUT /api/v1/books/{id}/info
Authorization: Bearer <token>
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 书籍 ID |

**请求体**（均为可选）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `name` | string | 新书名，唯一 |
| `description` | string | 新简介 |
| `price` | int64 | 新价格（金币） |

**响应 `data`**：[Book](#a3-book)

**说明**

- 只有书籍作者可操作，否则返回 `4020`。
- 名称冲突返回 `4008`；更新会写入 Outbox 并清理缓存。

### 6.5 上传书籍封面

```http
POST /api/v1/books/{id}/cover
Authorization: Bearer <token>
Content-Type: multipart/form-data
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 书籍 ID |

**表单字段**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `cover_image` | file | 是 | JPG，尺寸必须为 368×460，≤ 10 MB |

**响应 `data`**

```json
{
  "book_id": "book-uuid",
  "url": "http://localhost:8080/upload/book_cover/xxx.jpg"
}
```

**说明**

- 仅作者可上传，否则返回 `4020`。
- 文件过大 `4005`，类型错误 `4011`，尺寸错误 `4012`。

### 6.6 获取书籍章节目录

```http
GET /api/v1/books/{id}/chapters/list
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 书籍 ID |

**响应 `data`**

```json
{
  "Chapters": [
    {
      "chapter_id": "chapter-uuid",
      "title": "第一章",
      "order": 1,
      "vip_level": "",
      "price": 0
    }
  ]
}
```

> 响应字段 `Chapters` 为大写开头，与实现保持一致。

### 6.7 获取自己创作的书籍

```http
GET /api/v1/books/own/list?page=1&size=10
Authorization: Bearer <token>
```

**查询参数**：`page`、`size`，同 6.2。

**响应 `data`**：[BookList](#a4-booklist)

### 6.8 获取最近阅读的书籍

```http
GET /api/v1/books/history/list?page=1&size=10
Authorization: Bearer <token>
```

**查询参数**：`page`、`size`。

**响应 `data`**：[BookList](#a4-booklist)

**说明**

- 数据来自 Redis 阅读统计，不返回阅读时间。
- 仅返回当前登录用户最近阅读的书籍。

### 6.9 热度榜单

```http
GET /api/v1/books/thermal/daily?page=1&size=10
GET /api/v1/books/thermal/weekly?page=1&size=10
GET /api/v1/books/thermal/monthly?page=1&size=10
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `page` | int | 否 | 页码，默认 1 |
| `size` | int | 否 | 每页数量，默认 10，最大 100 |

**响应 `data`**：[BookList](#a4-booklist)

**说明**

- 榜单由 Redis ZSet 聚合：日榜取当天与前一天，周榜取近 7 天，月榜取近 31 天。
- 日榜聚合缓存 5 分钟；周 / 月榜聚合缓存 30 天。
- 阅读会带来热度，收藏权重更高（`thermal_value = bookshelf_amount × 10 + browse_amount`）。

### 6.10 搜索书籍

```http
GET /api/v2/books/search?query=剑&order=thermal_value&order_direction=desc&limit=10
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `query` | string | 否 | 关键字，匹配书名 / 简介 |
| `vip_level` | string | 否 | VIP 等级，多个用逗号分隔，如 `vip_0,vip_1` |
| `push_start_time` | int | 否 | 推送开始时间（Unix 秒） |
| `push_end_time` | int | 否 | 推送结束时间（Unix 秒） |
| `update_start_time` | int | 否 | 更新开始时间（Unix 秒） |
| `update_end_time` | int | 否 | 更新结束时间（Unix 秒） |
| `price_start` | int | 否 | 最低价格（金币） |
| `price_end` | int | 否 | 最高价格（金币） |
| `owner_id` | string | 否 | 作者 ID |
| `bookshelf_amount_start` | int | 否 | 收藏量下限 |
| `bookshelf_amount_end` | int | 否 | 收藏量上限 |
| `thermal_value_start` | int | 否 | 热度下限 |
| `thermal_value_end` | int | 否 | 热度上限 |
| `order` | string | 否 | 排序字段，默认 `thermal_value` |
| `order_direction` | string | 否 | `asc` / `desc`，默认 `desc` |
| `last_id` | string | 否 | 上一页最后一条书籍 ID |
| `limit` | int | 否 | 每页数量，默认 10，最大 100 |

**响应 `data`**

```json
{
  "books": [
    {
      "book": { /* Book */ },
      "highlight": {
        "name": "<em>剑</em>来"
      }
    }
  ]
}
```

**说明**

- 搜索由 Elasticsearch 完成，`highlight` 的 key 为命中字段。
- 游标分页不返回总数，下一页请把最后一条书籍的 `id` 作为 `last_id`。

### 6.11 获取书籍标签

```http
GET /api/v2/books/{id}/tags
```

**响应 `data`**

```json
{
  "book_id": "book-uuid",
  "tags": ["玄幻", "升级流"]
}
```

### 6.12 更新书籍标签

```http
PUT /api/v2/books/{id}/tags
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `tags` | string[] | 是 | 1~3 个标签，必须属于 `config/data.yaml` 白名单 |

```json
{ "tags": ["玄幻", "升级流"] }
```

**响应 `data`**

```json
{
  "book_id": "book-uuid",
  "tags": ["玄幻", "升级流"]
}
```

**说明**

- 仅作者可操作，否则返回 `4020`。
- 标签先删后插；非法或超出数量返回 `4017`。

---

## 7. 章节

### 7.1 创建章节

```http
POST /api/v1/chapters/create
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `book_id` | string | 是 | 所属书籍 |
| `title` | string | 是 | 章节标题 |
| `content` | string | 是 | 章节正文 |
| `vip_level` | string | 是 | 所需 VIP 等级，空字符串表示免费 |
| `price` | int | 是 | 单章价格（金币） |

**响应 `data`**：[Chapter](#a5-chapter)

**说明**

- 仅书籍作者可创建，否则返回 `4020`。
- `order` 自动取当前书籍最大序号 + 1，无需传入。
- 未知 VIP 等级会被归一化为 `vip_0`。
- 创建后写入 Outbox，并触发新章节通知任务。

### 7.2 游客获取章节正文

```http
GET /api/v1/chapters/{id}/guest
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 章节 ID |

**响应 `data`**：[Chapter](#a5-chapter)

**说明**

- 仅 `vip_level` 为空的免费章节可游客阅读。
- 需要登录的章节返回 `4014`。

### 7.3 获取章节正文（登录）

```http
GET /api/v1/chapters/{id}
Authorization: Bearer <token>
```

**响应 `data`**：[Chapter](#a5-chapter)

**说明**

- 已购买该章节，或当前 VIP 等级 ≥ 章节要求时返回正文。
- VIP 不足返回 `4015`。
- 成功读取会异步累计阅读字数、阅读记录、书籍浏览量并更新热度（带频控）。

### 7.4 获取章节信息

```http
GET /api/v1/chapters/{id}/info
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 章节 ID |

**响应 `data`**：[ChapterInfo](#a6-chapterinfo)

不返回章节正文。

### 7.5 更新章节

```http
PUT /api/v1/chapters/{id}
Authorization: Bearer <token>
Content-Type: application/json
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 章节 ID |

**请求体**（均为可选）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `title` | string | 新标题；空字符串不更新 |
| `content` | string | 新正文；空字符串不更新 |
| `vip_level` | string | 新 VIP 等级 |
| `price` | int64 | 新价格 |

**响应 `data`**：[Chapter](#a5-chapter)

**说明**

- 仅书籍作者可操作，否则返回 `4020`。
- 信息字段与正文字段分别走独立事务与 Outbox 事件。

### 7.6 删除章节

```http
DELETE /api/v1/chapters/{id}
Authorization: Bearer <token>
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 章节 ID |

**响应 `data`**：`null`

**说明**

- 仅书籍作者可操作，否则返回 `4020`。
- 使用硬删除，并写入 Outbox 删除事件以同步 Elasticsearch。

### 7.7 搜索章节

```http
GET /api/v2/chapters/search?query=剑&book_id=xxx&limit=10
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `query` | string | 否 | 关键字，匹配标题 / 正文 |
| `book_id` | string | 否 | 限定书籍 |
| `vip_level` | string | 否 | VIP 等级，多个用逗号分隔 |
| `created_at_start` | int | 否 | 创建开始时间（Unix 秒） |
| `created_at_end` | int | 否 | 创建结束时间（Unix 秒） |
| `price_start` | int | 否 | 最低价格（金币） |
| `price_end` | int | 否 | 最高价格（金币） |
| `order` | string | 否 | `created_at` / `price`，默认 `created_at` |
| `order_direction` | string | 否 | `asc` / `desc`，默认 `desc` |
| `last_id` | string | 否 | 上一页最后一条章节 ID |
| `limit` | int | 否 | 每页数量，默认 10，最大 100 |

**响应 `data`**

```json
{
  "chapters": [
    {
      "book_chapter": { /* Chapter */ },
      "highlight": {
        "content": "……<em>剑</em>……"
      }
    }
  ]
}
```

---

## 8. 消费与资产

### 8.1 购买书籍

```http
POST /api/v1/buy/book
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体（设计约定）**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `book_id` | string | 是 | 书籍 ID |
| `price` | int64 | 是 | 客户端展示价格，必须与书籍当前价格一致 |

**响应 `data`**：[PurchasedBook](#a7-purchasedbook)

**说明**

- 校验书籍存在、价格一致、未重复购买、金币充足。
- 扣费成功后把 50% 金币（向上取整）结算给作者，购买有效期约 3 年。
- 重复购买返回 `4009`，余额不足返回 `4018`。
- 已知问题：当前 DTO 中 `book_id` 使用 `uri` 标签，JSON 请求体无法填充该字段，接口待修复。

### 8.2 购买章节

```http
POST /api/v1/buy/chapter
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `book_id` | string | 是 | 书籍 ID |
| `chapter_id` | string | 是 | 章节 ID |
| `price` | int64 | 是 | 客户端展示价格，必须与章节当前价格一致 |

**响应 `data`**：`"ok"`

**说明**

- 章节必须属于指定书籍，价格必须一致。
- 重复购买返回 `4009`，余额不足返回 `4018`。
- 作者获得 50% 金币收益（向上取整）。

### 8.3 获取用户资产

```http
GET /api/v1/assets
Authorization: Bearer <token>
```

**响应 `data`**：[UserAsset](#a8-userasset)

### 8.4 已购买书籍列表

```http
GET /api/v1/purchases/book
Authorization: Bearer <token>
```

**响应 `data`**：`PurchasedBook[]`

### 8.5 检查是否已购买书籍

```http
GET /api/v1/purchases/book/{book_id}
Authorization: Bearer <token>
```

**响应 `data`**：[PurchasedBook](#a7-purchasedbook)

未购买返回 `4016`。

### 8.6 已购买章节列表

```http
GET /api/v1/purchases/chapters
Authorization: Bearer <token>
```

**响应 `data`**：`PurchasedChapter[]`

### 8.7 检查是否已购买章节

```http
GET /api/v1/purchases/chapters/{chapter_id}
Authorization: Bearer <token>
```

**响应 `data`**：[PurchasedChapter](#a9-purchasedchapter)

未购买返回 `4023`。

### 8.8 VIP 商品列表

```http
GET /api/v2/purchases/vip/shop
Authorization: Bearer <token>
```

**响应 `data`**

```json
{
  "shop": [
    {
      "type": "buy",
      "vip_level": "vip_1",
      "price": 1000,
      "duration": "month",
      "active_at": "2026-11-22T10:00:00+08:00"
    },
    {
      "type": "buy",
      "vip_level": "vip_2",
      "price": 3000,
      "duration": "month",
      "active_at": "2026-11-22T10:00:00+08:00"
    }
  ]
}
```

**说明**

- `type` 取值：`buy` 新购、`renew` 续费、`upgrade` 升级。
- 未开通 VIP 时返回 `vip_1` / `vip_2` 的月付与年付购买项。
- 当前为 `vip_1` 时返回续费项与 `vip_2` 升级项；当前为 `vip_2` 时返回续费项。
- 价格为金币，与 `purchased.vip_1/vip_2` 配置一致。

### 8.9 购买 / 续费 VIP

```http
POST /api/v2/purchases/vip
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `vip_level` | string | 是 | `vip_1` 或 `vip_2` |
| `duration` | string | 是 | `month` 或 `year` |
| `price` | int64 | 是 | 金币价格，必须与服务端配置一致 |

**响应 `data`**

```json
{
  "user_id": "user-uuid",
  "gold": 9000,
  "vip_mark": "vip_1",
  "vip_level": 1,
  "vip_active_at": "2026-11-22T10:00:00+08:00",
  "price": 1000,
  "duration": "month"
}
```

**说明**

- 价格由 `purchased.vip_1/vip_2` 配置决定，不一致返回 `4017`。
- 金币不足返回 `4018`；同级续费会在原到期时间上顺延。
- 已开通 `vip_1` 时不能通过本接口直接购买 `vip_2`，请使用 8.10 升级；已开通 `vip_2` 不支持降级。

### 8.10 升级 VIP

```http
POST /api/v2/purchases/vip/upgrade
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `original_vip_level` | string | 是 | 当前等级，必须为 `vip_1` |
| `target_vip_level` | string | 是 | 目标等级，必须为 `vip_2` |
| `price` | int64 | 是 | 升级价格，必须与服务端按剩余有效期计算的价格一致 |

**响应 `data`**

```json
{
  "user_id": "user-uuid",
  "gold": 7000,
  "original_vip_level": "vip_1",
  "target_vip_level": "vip_2",
  "vip_mark": "vip_2",
  "vip_level": 2,
  "vip_active_at": "2026-12-22T10:00:00+08:00",
  "price": 2000
}
```

**说明**

- 当前 VIP 不是 `vip_1`、目标不是 `vip_2` 或价格不一致时返回 `4017`。
- 升级后保持原 VIP 到期时间不变。

---

## 9. 书架、历史与关注

### 9.1 获取自己的书架

```http
GET /api/v1/bookshelf/list?page=1&size=10
Authorization: Bearer <token>
```

**响应 `data`**：[BookList](#a4-booklist)

### 9.2 获取指定用户书架

```http
GET /api/v1/bookshelf/{user_id}?page=1&size=10
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `user_id` | string | 用户 ID |

**查询参数**：`page`、`size`。

**响应 `data`**：[BookList](#a4-booklist)

### 9.3 添加到书架

```http
POST /api/v1/bookshelf/add
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `book_id` | string | 是 | 书籍 ID |

**响应 `data`**：`null`

**说明**

- 加入书架会创建收藏记录，并异步为书籍热度 +10；热度更新有 1 小时频控。
- 重复添加当前会返回内部错误，请先查询书架再添加。

### 9.4 从书架删除

```http
POST /api/v1/bookshelf/delete
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `book_id` | string | 是 | 书籍 ID |

**响应 `data`**：`null`

### 9.5 获取用户浏览历史（已弃用）

```http
GET /api/v1/history/{user_id}?page=1&size=10
```

**响应 `data`**

```json
{
  "books": [
    {
      "id": "history-uuid",
      "user_id": "user-uuid",
      "book_id": "book-uuid",
      "created_at": "2026-10-22T10:00:00+08:00"
    }
  ],
  "page": 1,
  "size": 10,
  "count": 1
}
```

> 该接口与 `user_history` 表为旧版链路保留；新客户端请使用 `GET /api/v1/books/history/list`。

### 9.6 查询关注状态

```http
GET /api/v1/likes?target_id=user-uuid
Authorization: Bearer <token>
```

**查询参数**

| 参数 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `target_id` | string | 是 | 目标用户 ID |

**响应 `data`**

```json
{
  "target_id": "user-uuid",
  "is_like": true
}
```

### 9.7 关注 / 取消关注

```http
PUT /api/v1/likes
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `target_id` | string | 是 | 目标用户 ID |
| `operator` | string | 是 | `create` 关注 / `delete` 取消关注 |

**响应 `data`**

```json
{
  "target_id": "user-uuid",
  "is_like": true
}
```

**说明**

- 关注会累加目标用户粉丝数；粉丝数达到阈值后标记为大 V。
- 该关系决定新书 Feed 是否扇出到当前用户。

---

## 10. Feed

### 10.1 获取当前用户 Feed 流

```http
POST /api/v2/feed
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `last_cursor_id` | string | 否 | 上一页游标 ID |
| `target_cursor_id` | string | 是 | 目标游标，首次传 `front` |
| `offset` | int | 否 | 游标内偏移，默认 0 |

```json
{
  "last_cursor_id": "",
  "target_cursor_id": "front",
  "offset": 0
}
```

**响应 `data`**

```json
{
  "feeds": [ /* Feed */ ],
  "next_cursor_id": "cursor-uuid",
  "next_offset": 10
}
```

**说明**

- 只返回当前用户关注作者产生的 Feed。
- 下一页把响应中的 `next_cursor_id`、`next_offset` 原样回传。

### 10.2 游客 Feed 流

```http
POST /api/v2/feed/guest
Content-Type: application/json
```

**请求体**：同 10.1。

**响应 `data`**

```json
{
  "feeds": [ /* GuestFeed */ ],
  "next_cursor_id": "cursor-uuid",
  "next_offset": 10
}
```

**说明**

- 返回全局 Feed 时间线，不包含 `is_in_book_shelf`、`is_user_liker` 等登录态字段。

### 10.3 旧版 Feed 列表（已弃用）

```http
GET /api/v1/feed/list
Authorization: Bearer <token>
```

**响应 `data`**

```json
{
  "feeds": [
    {
      "feed_id": "feed-uuid",
      "event_type": "new_chapter",
      "book_id": "book-uuid",
      "chapter_id": "chapter-uuid",
      "book_name": "书名",
      "book_cover": "http://.../cover.jpg",
      "chapter_title": "第一章",
      "content": "更新了章节",
      "created_at": "2026-10-22T10:00:00+08:00"
    }
  ],
  "restarted": false
}
```

> 该接口固定返回 10 条并自动推进游标，已由 V2 取代，仅作兼容保留。

---

## 11. 订单与支付

### 11.1 创建金币充值订单

```http
POST /api/v1/orders/create/gold
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `price` | float64 | 是 | 支付金额（元） |
| `reward_gold` | int64 | 是 | 到账金币数，必须等于 `price × currency_ratio`（默认 100） |
| `request_uuid` | string | 是 | 客户端幂等 ID |

**响应 `data`**

```json
{
  "order_id": "order-uuid",
  "pay_url": "https://openapi.alipay.com/...",
  "price": 10
}
```

**说明**

- 金额与金币比例不一致返回 `4017`。
- 同一 `request_uuid` 重复提交返回 `4022`。
- 支付宝下单失败返回 `4024`。
- 订单 5 分钟内有效。

### 11.2 查询订单

```http
GET /api/v1/orders/{id}
Authorization: Bearer <token>
```

**路径参数**

| 参数 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 订单 ID |

**响应 `data`**：[Order](#a10-order)

**说明**

- 订单待支付时使用长轮询：服务端最多保持 30 秒，每秒检查一次状态，状态变化或超时后返回。
- 只能查询自己的订单，否则返回 `4020`；订单不存在返回 `4021`。
- 客户端断开连接时服务端直接结束，不返回响应。

### 11.3 订单列表

```http
GET /api/v1/orders?page=1&size=10
Authorization: Bearer <token>
```

**查询参数**：`page`、`size`。

**响应 `data`**

```json
{
  "orders": [ /* Order */ ],
  "page": 1,
  "size": 10,
  "count": 1
}
```

### 11.4 支付宝异步通知

```http
POST /api/v1/pay/alipay
Content-Type: application/x-www-form-urlencoded
```

**表单参数（由支付宝发送）**

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `out_trade_no` | string | 商户订单号（订单 UUID 中的 `-` 替换为 `_`） |
| `trade_status` | string | 交易状态 |
| `total_amount` | string | 订单金额（元） |

**响应**：`text/plain`

- 处理成功：`success`
- 验签失败 / 处理失败：`failure`（支付宝会重试）

**说明**

- 该接口由支付宝服务端调用，不挂载登录认证。
- 仅处理支付成功通知；验签失败返回 `failure`，金额与订单不一致时不发放金币。
- 支付成功后在事务内发放金币、更新订单状态、删除临时轮询记录与缓存。

---

## 附录 A：公共响应结构

### A.1 User

```json
{
  "id": "user-uuid",
  "username": "reader",
  "email": "reader@example.com",
  "age": 20,
  "description": "简介",
  "profile_image": "http://localhost:8080/upload/avatar/xxx.png",
  "cover_image": "http://localhost:8080/upload/cover_image/default.jpg",
  "created_at": "2026-10-22 10:00:00",
  "updated_at": "2026-10-22 10:00:00"
}
```

### A.2 UserStat

在 User 全部字段基础上追加：

```json
{
  "gold": 1000,
  "vip_mark": "vip_1",
  "vip_level": 1,
  "vip_active_at": "2026-11-22T10:00:00+08:00",
  "read_books_num": 12,
  "read_words_num": 345678
}
```

### A.3 Book

```json
{
  "id": "book-uuid",
  "name": "书名",
  "description": "简介",
  "cover_image": "http://localhost:8080/upload/book_cover/xxx.jpg",
  "owner_id": "author-uuid",
  "vip_level": "",
  "price": 0,
  "push_at": 1761098400,
  "browse_amount": 100,
  "bookshelf_amount": 20,
  "created_at": 1761098400,
  "updated_at": 1761098400
}
```

### A.4 BookList

```json
{
  "books": [ /* Book */ ],
  "page": 1,
  "size": 10,
  "count": 35
}
```

### A.5 Chapter

```json
{
  "chapter_id": "chapter-uuid",
  "title": "第一章",
  "content": "正文……",
  "order": 1,
  "vip_level": "",
  "price": 0,
  "Book": { /* Book */ }
}
```

> `Book` 字段首字母大写，与实现一致。

### A.6 ChapterInfo

```json
{
  "chapter_id": "chapter-uuid",
  "title": "第一章",
  "order": 1,
  "vip_level": "",
  "price": 0,
  "Book": { /* Book */ }
}
```

### A.7 PurchasedBook

```json
{
  "purchased_id": "purchase-uuid",
  "user_id": "user-uuid",
  "book_id": "book-uuid",
  "purchased_at": "2026-10-22T10:00:00+08:00",
  "active_at": "2029-10-22T10:00:00+08:00"
}
```

### A.8 UserAsset

```json
{
  "id": "user-uuid",
  "gold": 1000,
  "vip_mark": "vip_1",
  "vip_level": 1,
  "vip_active_at": "2026-11-22T10:00:00+08:00"
}
```

### A.9 PurchasedChapter

```json
{
  "purchased_id": "purchase-uuid",
  "user_id": "user-uuid",
  "chapter_id": "chapter-uuid",
  "book_id": "book-uuid",
  "purchased_at": "2026-10-22T10:00:00+08:00",
  "active_at": "2029-10-22T10:00:00+08:00"
}
```

### A.10 Order

```json
{
  "order_id": "order-uuid",
  "user_id": "user-uuid",
  "pay_price": 10,
  "original_price": 10,
  "reword_gold": 1000,
  "past_at": "2026-10-22T10:05:00+08:00",
  "status": 1,
  "created_at": "2026-10-22T10:00:00+08:00"
}
```

> 响应字段 `past_at` 当前实现取订单创建时间（`created_at`），并非订单过期时间；订单真实过期时间不对外返回。底层模型字段为 `expire_at`。

### A.11 Feed

```json
{
  "feed_id": "feed-uuid",
  "book_id": "book-uuid",
  "user_id": "author-uuid",
  "content": "发布了书籍《书名》",
  "feed_created_at": 1761098400,
  "feed_updated_at": 1761098400,
  "book_name": "书名",
  "book_description": "简介",
  "book_cover_url": "/upload/book_cover/xxx.jpg",
  "book_vip_level": "",
  "book_price": 0,
  "book_browse_amount": 100,
  "book_bookshelf_amount": 20,
  "book_push_at": 1761098400,
  "book_created_at": 1761098400,
  "book_updated_at": 1761098400,
  "user_profile_url": "/upload/avatar/xxx.png",
  "user_fans": 1000,
  "user_vip_level": "vip_1",
  "is_in_book_shelf": false,
  "is_user_liker": true
}
```

游客 Feed（`GuestFeed`）包含除 `is_in_book_shelf`、`is_user_liker` 外的全部字段。

> Feed 响应中的 `book_cover_url`、`user_profile_url` 为数据库存储路径，客户端需拼接 `server.website` 后访问。

---

## 附录 B：接口现状说明

1. `PUT /api/v1/users/password` 未挂载认证中间件，详见 5.7。
2. `POST /api/v1/buy/book` 请求 DTO 的 `BookID` 使用 `uri` 标签，JSON body 无法绑定，详见 8.1。
3. `GET /api/v1/feed/list`、`GET /api/v1/history/{user_id}` 为兼容旧客户端保留，新功能请使用 V2 Feed 与 `/api/v1/books/history/list`。
4. 所有接口的路径、字段名、错误码均属于兼容性契约，调整前需评估客户端影响。
