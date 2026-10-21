# streaming-record

B 站直播开播/下播记录服务（Go 重写版，SQLite 本地存储）。

## 功能

- 每分钟轮询 B 站 API，自动记录开播（`start_time`）和下播（`end_time`），UTC 毫秒时间戳
- `GET /getInfo?room_id=` 透传 B 站 `get_info` API
- `GET /query?room_id=&startTime=&endTime=`（unix 秒）查询时间区间内的直播 sessions
- `/admin/streamers` `/admin/sessions` 完整 CRUD，Bearer token 鉴权
- CORS 仅放行 `loli.cn.com` 及其子域

## 运行

```sh
go build -o streaming-record .
./streaming-record -addr :8787 -db streaming.db -admin-token "$(openssl rand -hex 32)"
```

或用环境变量传 token：`ADMIN_TOKEN=xxx ./streaming-record`

## 参数

| Flag | 默认 | 说明 |
|---|---|---|
| `-addr` | `:8787` | 监听地址 |
| `-db` | `streaming.db` | SQLite 文件路径（自动创建，WAL 模式） |
| `-admin-token` | `$ADMIN_TOKEN` | admin API 的 Bearer token；为空时所有 `/admin/*` 返回 401 |

## 数据库

启动时自动建表（`streamers`、`sessions`），字段与 Workers 版本一致：

- `streamers(id, name, room_id, uuid UNIQUE)`
- `sessions(id, streamer_id FK, start_time INTEGER, end_time INTEGER NULL)` — unix 毫秒 UTC；`end_time IS NULL` 表示在播

## 反爬说明

B 站 API 有 412 风控，本实现会自动先请求 `https://www.bilibili.com/` 拿 `buvid3` cookie（缓存 6h），再调 `get_info`。如果仍然 412，可能需要用户提供登录态 cookie，或考虑直接抓直播间 HTML 的 `__INITIAL_STATE__`。

## API 示例

```sh
# 添加主播
curl -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name":"主播名","room_id":"1962432178","uuid":"任意uuid"}' \
  http://localhost:8787/admin/streamers

# 查询某房间 2026-10-21 全天的直播
curl "http://localhost:8787/query?room_id=1962432178&startTime=1792540800&endTime=1792627200"
```
