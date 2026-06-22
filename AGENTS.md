# 项目 AGENTS.md

本目录当前是“终端敏感数据外发监控系统”方案规划工作区，重点实施对象是 `excute-plan.md` 中的 **模块三：告警日志误报分析 Agent**。后续编码时默认在 `module3-alert-agent/` 子目录内创建 Go 服务，不要把实现文件散落在规划根目录。

## 构建、运行、测试

项目初始化后优先使用这些命令：

```powershell
cd module3-alert-agent
go mod tidy
go run ./cmd/server
go test ./...
```

前端源码在 `module3-alert-agent/frontend/`，演示页面构建到 `module3-alert-agent/internal/router/web/`，由 Go 服务通过 `GET /` 静态托管。修改前端展示层后优先运行：

```powershell
cd module3-alert-agent/frontend
npm run typecheck
npm run build
```

数据库初始化脚本应放在 `module3-alert-agent/sql/schema.sql`。需要连接 MySQL 时，先确认本机已有可用 MySQL 8.0+，再执行建表或测试。

服务默认端口是 `9090`。如果端口被占用，先说明原因，再临时改配置或使用环境变量覆盖。

## 资料来源与优先级

先读本文件，再读：

1. `excute-plan.md`：模块三实施方案，作为当前项目主规格。
2. `原始需求.md`：模块三原始需求与接口背景。
3. `dlpagent.md`：全系统背景，必要时用于理解模块二、模块四边界。

如果这些文件之间冲突，以 `excute-plan.md` 为准；如果用户当前消息更新了决策，以用户当前消息为准。资料里 OCR、编码或字段不清楚时，标记 `待确认`，不要猜。

## 核心架构约束

- 部署形态：独立 Go 服务。
- HTTP 框架：CloudWeGo Hertz。
- Agent 框架：必须使用 Eino ADK / ReAct，不要换成其他 Agent 框架。
- 数据库：MySQL，持久化 `alert_logs`、`whitelist_rules`、`false_positive_library`。
- 模块通信：HTTP + JSON。
- 主入口：`cmd/server/main.go`。
- 业务代码：放在 `internal/` 下，按 config、handler、pipeline、agent、model、store、router 分层。

规划中的目录结构：

```text
module3-alert-agent/
├── cmd/server/main.go
├── internal/config/
├── internal/handler/
├── internal/pipeline/
├── internal/agent/
├── internal/model/
├── internal/store/
├── internal/router/
├── frontend/
├── sql/schema.sql
├── config.yaml
├── go.mod
├── go.sum
└── README.md
```

## 处理流水线

必须保持顺序：

1. 白名单过滤
2. 去重合并
3. 结构化误报召回与 Agent 精判
4. 写入最终告警

白名单是确定性规则引擎，Agent 不参与白名单判断。白名单命中后直接丢弃日志，不写入 `alert_logs`。

去重合并使用精确匹配 + 滑动窗口。合并 Key 是：

```text
host_id + user_id + process_name + sensitive_type + operation
```

默认窗口：

| risk_level | window |
|---|---:|
| critical | 30s |
| high | 60s |
| medium | 180s |
| low | 300s |
| info | 600s |

## API 约束

必须实现：

```http
POST /api/client/events
POST /api/alerts/query
GET    /api/whitelist
POST   /api/whitelist
PUT    /api/whitelist/:id
DELETE /api/whitelist/:id
GET    /api/false-positives
POST   /api/false-positives
DELETE /api/false-positives/:id
```

`POST /api/client/events` 接收批量 `events` 数组。实现时可兼容单条输入，但规范输出仍以数组为准。

`POST /api/alerts/query` 面向模块四，必须支持条件过滤、分页、排序，并限制 `page_size` 上限。

## Agent 与误报召回

当前实现采用结构化召回 + ReAct Agent 两阶段：

1. 召回阶段：按 `sensitive_type`、`operation`、`process_name`、`target`、`user_id`、`process_path` 对 `false_positive_library` 做结构化打分，默认 Top-5。
2. C 阶段：ReAct Agent 精判。

`false_positive_library.embedding_json` 是兼容字段，当前主流程不依赖 Embedding；不要为了演示前端重新引入向量数据库或强制外部 Embedding 服务。

默认参数：

- `top_k_recall`: 5
- `structured_recall_threshold`: 0.60
- `strong_recall_threshold`: 0.75
- `confidence_threshold`: 0.8
- `false_positive_ttl_days`: 30
- `max_concurrency`: 先设 5，压测后调整

Agent 工具应包含：

- `SearchFalsePositiveHistory`
- `GetEventDetail`
- `QueryWhitelist`
- `MarkAsFalsePositive`

`confidence < 0.8` 时不要调用 `MarkAsFalsePositive` 写入误报库，只允许降级风险等级。

## 配置与密钥

不要把真实 API Key、数据库密码或其他凭据写进代码、日志、测试快照或提交说明。

`config.yaml` 只能放占位值或本地示例值；真实值优先从环境变量读取，例如：

```powershell
$env:ARK_CHAT_MODEL="ep-20260602133449-kmp8g"
$env:ARK_EMBEDDING_MODEL="ep-m-20260410103316-w6v55"
$env:ARK_API_KEY="..."
```

如果规划文档里出现真实密钥，后续实现时也不要复制到源码中。

## 数据模型与存储

以 `excute-plan.md` 的三张表为基准：

- `alert_logs`
- `whitelist_rules`
- `false_positive_library`

表结构变更前先说明影响。除非用户确认，不新增数据库迁移体系或更换数据库。

`false_positive_library.embedding_json` 保留为兼容字段；当前召回逻辑以结构化字段打分为准。不要引入向量数据库，除非用户重新决策。

## 编码约定

- 使用 Go 1.21+。
- 优先沿用标准 Go 布局和小而清晰的 `internal` 包。
- Handler 只做 HTTP 入参、出参和错误码处理；业务流程放 `pipeline`；数据库访问放 `store`；Agent 相关逻辑放 `agent`。
- 对外 JSON 字段名保持与方案一致，例如 `event_id`、`risk_level`、`false_positive_reason`。
- 风险等级只允许 `critical`、`high`、`medium`、`low`、`info`。
- 时间字段要统一处理，避免同时混用秒、毫秒和 RFC3339 而不标注。
- 不为白名单、去重、Agent 判断写模糊的字符串拼接逻辑；能用结构体和明确字段就用结构体。

## 测试与验收

优先补这些聚焦测试：

- 白名单 AND/OR 命中与未命中。
- 白名单 API 更新后内存缓存同步刷新。
- 去重合并 Key 完全一致才合并。
- 不同 `risk_level` 使用不同滑动窗口。
- `confidence < 0.8` 不写入误报库。
- `/api/alerts/query` 的分页、排序和条件过滤。

完成前至少运行：

```powershell
go test ./...
```

如果数据库或外部 LLM/Embedding 服务不可用，说明未验证项，并尽量用 mock 或本地假数据覆盖规则、去重和查询逻辑。

## 实施阶段

按 `excute-plan.md` 的 Phase 顺序推进：

1. Phase 1：Go 骨架、MySQL DDL、Hertz 启动、配置加载、路由注册。
2. Phase 2：白名单 CRUD、热缓存、过滤、去重合并、事件接收流水线。
3. Phase 3：Eino ADK、方舟 ChatModel/Embedding、4 个 Tool、ReAct 精判、误报库 TTL。
4. Phase 4：查询 API、端到端联调、并发压测、模块二/四接口对齐。
5. Phase 5：单元测试、Agent 样例测试、README/API 文档、答辩材料，以及内置 Demo Theater 前端演示页面。

跨 Phase 做大改前先说明范围。新增计划外依赖、修改锁文件、接入外部服务或调整数据库结构前，先向用户确认。

## 交付说明

每次完成后简短说明：

- 改了哪些文件。
- 跑了哪些检查。
- 哪些内容仍是 `待确认`。
- 是否建议做备份提交或进入下一 Phase。
