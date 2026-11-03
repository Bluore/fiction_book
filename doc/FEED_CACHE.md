# Feed 公共组合缓存

本文记录 Feed 时间线读取链路中的公共组合卡片缓存（`FeedComposeCache`）的 key、TTL、数据边界、读取顺序、失效方式、降级路径与回滚方式。

## 1. 缓存内容

一张卡片只包含与查看者无关的公共数据，使用扁平结构：

```text
FeedCard
  FeedID, UserID, BookID, Content, FeedCreatedAt, FeedUpdatedAt
  BookName, BookDescription, BookCoverURL, BookVIPLevel, BookPrice,
  BookBrowseAmount, BookBookshelfAmount, BookPushedAt, BookCreatedAt, BookUpdatedAt
  AuthorProfileURL, AuthorFansNum, AuthorVIPLevel
```

以下字段**不进入**公共卡片，每次请求按当前查看者实时读取：

- `is_user_liker`：来自 `user_like`，key 为 `KeyUserLike(authorID, viewerID)`
- `is_in_book_shelf`：来自 `user_bookshelf`，key 为 `KeyUserBookshelf(viewerID, bookID)`

游客与登录用户共用同一份公共卡片；游客只跳过 like/shelf 查询，因此不会出现查看者状态串号。

## 2. Key 与 TTL

代码位置：`internal/cache/feed_compose_cache.go`。

```text
GSF:feed:cache:card:v1:<feedID>
```

- 由 `keyFeedCard` 常量与 `KeyFeedCard(feedID)` 生成，修改时必须保持 `card:v1` 前缀或升级版本号。
- 有效期：`TimeFeedCard()` = `60s + TimeShakeDuration(60s)`，即 60–66 秒随机抖动，避免热门卡片同时过期回源。
- 缓存值：扁平 `model.FeedCard` 的 JSON；值为 `constant.RedisValueEmpty`（`"EMPTY"`）时视为确定不存在，跳过该条 Feed。
- 源数据（`book`、`user`、`userInfo`、`userAsset`、`feed`、`user_like`、`user_bookshelf`）继续复用各自现有 key 与 TTL（约 1 小时）。

## 3. cache-aside 读取顺序

```text
R0  pipeline(EXISTS, LRANGE) 取 feed ID 列表 ................ 1 RTT
R1  GetCards(feedIDs) 读取卡片 .............................. 1 RTT
      命中  -> 直接使用
      miss  -> 对该 feedID 调用 buildCard，走原有单条 helper
      EMPTY -> 跳过该条 Feed
R2  GetViewerStates(authorIDs, bookIDs) 读取 like/shelf ..... 1 RTT
      map 命中 -> 直接使用
      map 缺失 -> 对单个 key 调用原有 helper
R3  UpdateCards(toWrite) 写回本轮新建卡片 ................... 1 RTT（仅冷路径）
```

`buildCard` 的顺序与旧逻辑一致：

```text
queryUserFeedDetailWithCache
  -> queryBookDetailWithCache
  -> queryUserDetailWithCache
  -> queryUserAssetDetailWithCache
  -> queryUserInfoDetailWithCache
任意一步返回 nil -> 跳过该条 Feed，不写卡片
```

热路径 RTT：

| 场景 | RTT | 说明 |
|---|---:|---|
| 登录用户，卡片和查看者状态全命中 | 3 | R0 + R1 + R2 |
| 游客，卡片全命中 | 2 | R0 + R1 |
| 卡片未命中，源缓存命中 | 3 + 5×miss | 每个 miss 的 Feed 走原有五条 helper；写回再 +1 pipeline |
| 源缓存也未命中 | 3 + 5×miss + DB | DB 回源只针对 miss 的实体 |

card miss 不再走批量源数据 pipeline，而是按 Feed 顺序回源；这是用冷路径 RTT 换读取链路的顺序化和类型收敛。

## 4. `GetViewerStates` 语义

返回两个 `map[string]bool`，用“键是否存在”表达三种状态：

```text
likes[authorID] = true    命中，已点赞
likes[authorID] = false   EMPTY，确定未点赞
likes[authorID] 不存在     cache miss，调用 queryUserLikeDetailWithCache

shelves[bookID] 同理
```

边界：

| 情况 | 行为 |
| --- | --- |
| 整个 pipeline 失败 | 返回 nil maps + error；service 对每个 Feed 回退原 helper |
| 单个 key 为 `redis.Nil` | 不放入 map，service 回退 helper |
| 单个 key 为 `EMPTY` | 放入 map，值为 `false`，不查 DB |
| 单个 key 解码失败 | 不放入 map，记录 warning，回退 helper |
| like 查询错误 | 记录 warning，视为未点赞 |
| shelf 查询错误 | 记录 warning，跳过该条 Feed |
| 同一 author/book 多个 Feed | 按 ID 去重，一次 pipeline |

## 5. 未命中与缺失语义

- `book`、`user`、`userInfo`、`userAsset` 任一缺失：跳过该条 Feed，且**不写回**不完整卡片。
- `feed` 缺失：跳过该条 Feed。
- 卡片值为 `EMPTY`：跳过该条 Feed，不进入 `buildCard`。
- 卡片解码失败：按 miss 处理，走 `buildCard`。
- `GetCards` 或 `GetViewerStates` 整条 pipeline 失败：记录 warning，按 miss 逐条回退原有 helper。
- 游客不调用 `GetViewerStates`，也不读取 like/shelf。

## 6. 写回策略

- 只写回本轮卡片 miss、且公共实体全部组装成功的条目。
- 命中的卡片不写回，避免无意义续 TTL。
- 响应组装完成后统一执行一次 `UpdateCards(toWrite)`；写失败只记录 warning，不影响本次响应。
- 不写 `EMPTY` 墓碑，避免把临时缺失固化；缺失条目后续请求会再次回源。
- 写回发生在返回响应前，保证下一次请求（包括游客请求）可以命中同一张卡片。

## 7. 失效

- `FeedComposeCache.DeleteCard(ctx, feedID)` 删除单张卡片。
- 当前没有 Feed 更新/删除 API，只有 Kafka 推送创建路径；创建不会产生旧卡片，因此暂不需要主动删除。
- 未来新增 Feed 删除/更新路径时，必须在同一业务流程中调用 `DeleteCard`，否则旧卡片最多保留一个 TTL（60–66 秒）。
- `book`、`user`、`userInfo`、`userAsset` 没有反向索引，跨实体变更依赖短 TTL 收敛，不做精确失效。

## 8. 可观测性

`internal/service/feed_metrics.go` 维护服务级原子计数：

- `card_hit` / `card_miss`
- `source_fallback`，按 `book` / `user` / `user_info` / `user_asset` / `feed` / `like` / `shelf` 区分

每次请求输出一条 debug 日志：

```text
feed compose cache stats  {"feed_count": N, "card_hit": N, "card_miss": N}
```

测试通过 `feedMetrics().snapshot()` 断言命中、未命中与回源计数；`TestFeedCardSharedByUserAndGuest` 用 Redis pipeline hook 验证登录用户热路径为 3 次 pipeline、游客为 2 次，并且 card miss 只回源对应 Feed。

## 9. 降级与回滚

降级顺序：

1. `GetCards` pipeline 失败 -> 所有 Feed 走 `buildCard` 原有单条 helper。
2. `GetViewerStates` pipeline 失败 -> 所有 Feed 的 like/shelf 走原有 helper。
3. 单个源 key 未命中 -> 只对该 key 调用现有 `query*WithCache`，保持原缓存与分布式锁行为。

回滚：

- 回退二进制后旧代码不会读取 `GSF:feed:cache:card:v1:*`，无需删除 Redis key。
- 卡片 TTL 为 60–66 秒，回退后最迟一个 TTL 自然清理，不需要数据库或 Redis 清理动作。
- 本次变更不涉及数据库表、migration、API 路径、DTO 或 Swagger。
