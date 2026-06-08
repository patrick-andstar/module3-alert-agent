# 模块三：告警日志误报分析 Agent — 实施方案

> **状态：** 待实施  
> **负责人：** [你的小组]  
> **最后更新：** 2026-06-05

---

## 一、架构决策总览

| # | 决策项 | 选择 | 简述 |
|---|--------|------|------|
| 1 | 部署形态 | **独立 Go 服务** | 模块二通过 HTTP 上报日志 |
| 2 | 处理流水线顺序 | **白名单 → 去重合并 → 误报检索(Agent)** | 先规则后 AI |
| 3 | 模块间通信 | **HTTP + JSON** | `POST /api/client/events` 批量上报 |
| 4 | 数据库 | **MySQL** | 持久化白名单、误报库、告警记录 |
| 5 | 白名单匹配策略 | **可配置 AND/OR** | 每条规则独立配置逻辑 |
| 6 | Agent 参与白名单 | **不参与** | 白名单是确定性规则引擎 |
| 7 | 白名单管理 | **API + 内存热缓存** | 写 DB 同步刷新缓存，查时命中缓存 |
| 8 | 去重合并算法 | **精确匹配 + 滑动窗口** | 维度完全一致才合并 |
| 9 | 合并窗口 | **按风险等级可配置** | 默认值见 §3.2 |
| 10 | 误报检测策略 | **结构化召回(B) + Agent 精判(C)** | MySQL 误报模式库打分召回 Top-5 → Agent 推理 |
| 11 | 误报模式库 | **MySQL + scenario_key 去重聚合** | 误报库沉淀可复用场景，不堆积重复事件 |
| 12 | Agent 框架 | **Eino ADK** | CloudWeGo 全家桶 |
| 13 | LLM | **火山方舟 deepseekv4** | `set-via-env` |
| 14 | Embedding | **可选增强，当前不依赖** | 无可用 Embedding 模型时，主流程仍必须跑通 |
| 15 | 结构化召回字段 | **多字段打分** | sensitive_type + operation + process_name + target，user_id/process_path 辅助打分 |
| 16 | Agent 模式 | **单条单次 ReAct + goroutine 并发** | 先测并发上限再定池大小 |
| 17 | Agent 并发控制 | **固定并发池** | 先测试得出上限 x，池大小 = x/2 |
| 18 | Agent 误报写入 | **服务端统一控制** | Agent 只输出判断；服务端按 recall_score、confidence、scenario_key 去重后写入/更新误报库 |
| 19 | 误报 TTL | **30 天** | 过期自动失效 |
| 20 | 模块四对接 | **查询 API** | 通用查询接口，JSON 条件 → 分页结果 |
| 21 | HTTP 框架 | **Hertz** | CloudWeGo 生态 |
| 22 | 项目结构 | **标准 Go 布局** | `cmd/server/` + `internal/` |
| 23 | 服务端口 | **9090** | |

---

## 二、系统架构

### 2.1 整体数据流

```
┌──────────────┐     HTTP POST /api/client/events      ┌─────────────────────────────┐
│   模块二      │ ───────────────────────────────────> │  模块三：告警误报分析 Agent    │
│  监控客户端   │        (JSON, 批量上报)               │                              │
└──────────────┘                                       │  ┌──────────────────────┐   │
                                                        │  │ ① 白名单过滤         │   │
                                                        │  │   (内存热缓存命中)    │   │
                                                        │  │   → 命中: 直接丢弃    │   │
                                                        │  ├──────────────────────┤   │
                                                        │  │ ② 去重合并           │   │
                                                        │  │   (精确匹配+滑动窗口) │   │
                                                        │  │   → 合并为聚合告警    │   │
                                                        │  ├──────────────────────┤   │
                                                        │  │ ③ 误报检索(Agent)    │   │
                                                        │  │   结构化召回 → ReAct │   │
                                                        │  │   → 降级/维持/标记   │   │
                                                        │  └──────────────────────┘   │
                                                        │              │               │
                                                        │    MySQL     │               │
                                                        │  ┌───────────┴──────────┐   │
                                                        │  │ alert_logs           │   │
                                                        │  │ whitelist_rules      │   │
                                                        │  │ false_positive_lib   │   │
                                                        │  └──────────────────────┘   │
                                                        │              │               │
                                                        │    HTTP API (查询)          │
                                                        └──────────────┬──────────────┘
                                                                       │
                                                       GET/POST /api/alerts/query
                                                                       │
                                                              ┌────────┴───────┐
                                                              │   模块四        │
                                                              │  统计分析Agent  │
                                                              └────────────────┘
```

### 2.2 流水线详细流程

```
Raw Events (from 模块二)
    │
    ▼
[Step 1: 白名单过滤] ──命中──> 丢弃 (不写入 DB)
    │ 未命中
    ▼
[Step 2: 去重合并]
    │ 基于 Key: host_id + user_id + process_name + sensitive_type + operation
    │ 滑动窗口: 按 risk_level 配置不同窗口
    │
    ▼
[Step 3: 误报检索 Agent]
    │
    ├── [B 阶段: 结构化召回]
    │   │ 当前日志字段 → 与误报模式库打分匹配 → Top-5 候选
    │   │
    │   ▼
    └── [C 阶段: ReAct Agent 精判]
        │ 输入: 当前日志 + Top-5 候选误报
        │ Tools: SearchFalsePositiveHistory, GetEventDetail, QueryWhitelist, MarkAsFalsePositive
        │ 输出: agent_verdict / is_false_positive / new_risk_level / reason / confidence / explanation
        │
        ▼
    最终告警 (写入 alert_logs)
```

---

## 三、核心逻辑规格

### 3.1 白名单过滤

**匹配逻辑**：每条规则配置 `logic: AND | OR`

```json
// OR 示例：任一条件命中即丢弃
{
  "rule_name": "备份程序豁免",
  "logic": "OR",
  "conditions": {
    "process_name": "backup.exe",
    "file_path_pattern": "C:/Backup/*"
  }
}

// AND 示例：必须全部满足才丢弃
{
  "rule_name": "管理员CRM操作",
  "logic": "AND",
  "conditions": {
    "user_id": "admin",
    "process_name": "chrome.exe",
    "target": "internal-crm.company.com"
  }
}
```

**热缓存策略**：
- 服务启动时从 MySQL 加载白名单到内存
- 白名单 API (增/删/改) → 更新 MySQL + 同步刷新内存缓存
- 日志过滤时直接读内存，零 DB 查询

### 3.2 去重合并

**合并 Key**：`host_id + user_id + process_name + sensitive_type + operation`

**滑动窗口默认值**：

| 原风险等级 | 合并窗口 | 说明 |
|-----------|---------|------|
| critical | 30 秒 | 高危快速放行 |
| high | 60 秒 | 需求文档示例值 |
| medium | 180 秒 | 中等缓冲 |
| low | 300 秒 | 充分合并 |
| info | 600 秒 | 最大化合并 |

**合并输出**：聚合事件，包含 `file_count`、`files[]`、`time_range`、`duration`、`is_merge_event: true`

### 3.3 结构化召回（B 阶段）

当前阶段不依赖 Embedding。误报召回使用 `false_positive_library` 中的误报模式记录，按结构化字段计算相似分数，再取 Top-5 交给 Agent 精判。

**召回字段与默认权重**：

| 字段 | 匹配方式 | 权重 |
|---|---|---:|
| `sensitive_type` | 精确匹配 | 0.25 |
| `operation` | 精确匹配 | 0.20 |
| `process_name` | 忽略大小写精确匹配 | 0.20 |
| `target` | 规范化后匹配 | 0.20 |
| `user_id` | 精确匹配 | 0.10 |
| `process_path` | 忽略大小写精确匹配 | 0.05 |

**召回流程**：
1. 从 `false_positive_library` 查询 `expired_at > now()` 的未过期误报模式。
2. 对当前日志和每条误报模式计算结构化召回分数 `recall_score`。
3. 仅保留 `recall_score >= structured_recall_threshold` 的候选。
4. 按 `recall_score` 降序取 Top-5。
5. 将当前日志和 Top-5 候选误报模式交给 ReAct Agent。

**默认参数**：

- `top_k_recall`: 5
- `structured_recall_threshold`: 0.60
- `strong_recall_threshold`: 0.75

`scenario_key` 用于误报库去重，默认由以下字段生成：

```text
lower(sensitive_type) + "|" + lower(operation) + "|" + lower(process_name) + "|" + normalize_target(target)
```

`user_id` 不进入默认 `scenario_key`，避免同一业务场景被不同用户拆成大量重复模式；但 `user_id` 仍参与召回打分。

### 3.4 ReAct Agent 精判（C 阶段）

**Tools（4 个）**：

| Tool | 功能 | 输入 | 输出 |
|------|------|------|------|
| `SearchFalsePositiveHistory` | 结构化召回误报候选 | event_id | Top-5 候选误报模式及 `recall_score` |
| `GetEventDetail` | 获取事件完整上下文 | event_id | 事件完整 JSON |
| `QueryWhitelist` | 查询白名单（兜底） | event 字段 | 是否命中白名单 |
| `MarkAsFalsePositive` | 表达“建议标记为误报” | event + reason + confidence | 标记建议，不直接写库 |

**Agent 输出 JSON**：

```json
{
  "event_id": "evt_001",
  "is_false_positive": true,
  "new_risk_level": "info",
  "false_positive_reason": "用户上传客户资料到内部CRM系统，属于正常业务操作",
  "confidence": 0.92,
  "agent_verdict": "false_positive",
  "explanation": "该事件与历史误报模式 #42 高度相似（结构化召回分数 0.85），均为员工使用 Chrome 上传客户数据到内部 CRM。"
}
```

Agent 不直接写入 `false_positive_library`。服务端根据 `recall_score`、`confidence`、`false_positive_reason` 和 `scenario_key` 统一决定是否写入或更新误报模式库。

### 3.5 风险等级转换

模块三只负责误报分析，不负责风险升级。Agent 判断不误报时保持原风险等级。

| 判断结果 | risk_level 处理 | 是否写入 `alert_logs` | 是否写入 `false_positive_library` |
|----------|--------------|---:|---:|
| 白名单命中 | 丢弃 | 否 | 否 |
| 非误报 (`agent_verdict=true_alert`) | 保持原等级，不升级 | 是 | 否 |
| 疑似误报 (`confidence < 0.8`) | 最多降 1 级，不能降到 `info` | 是 | 否 |
| 空召回但 Agent 判断误报 | 最多降 2 级，不能降到 `info` | 是 | 否 |
| 召回命中且确认误报 (`recall_score >= 0.75` 且 `confidence >= 0.8`) | 可降到 `low` 或 `info` | 是 | 是，按 `scenario_key` 去重写入/更新 |

写入误报库必须同时满足：

1. 结构化召回命中，且 Top-1 `recall_score >= strong_recall_threshold`。
2. Agent `confidence >= confidence_threshold`。
3. `false_positive_reason` 非空。
4. `scenario_key` 生成成功。

若 `scenario_key` 已存在，则不新增重复记录，只更新 `hit_count`、`last_seen_at`、`expired_at` 和必要的 `reason`。

---

## 四、数据库表结构

### 4.1 `alert_logs` — 处理后的告警日志

```sql
CREATE TABLE alert_logs (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  event_id VARCHAR(64) NOT NULL UNIQUE,
  host_id VARCHAR(64),
  user_id VARCHAR(64),
  file_path VARCHAR(512),
  file_hash VARCHAR(128),
  sensitive BOOLEAN DEFAULT FALSE,
  sensitive_type VARCHAR(128),
  risk_level ENUM('critical','high','medium','low','info'),
  old_risk_level ENUM('critical','high','medium','low','info'),
  process_name VARCHAR(256),
  process_path VARCHAR(512),
  target VARCHAR(512),
  operation VARCHAR(64),
  sensitive_file_id VARCHAR(64),
  is_merge_event BOOLEAN DEFAULT FALSE,
  file_count INT DEFAULT 1,
  false_positive_reason TEXT,
  agent_verdict ENUM('false_positive','true_alert','uncertain'),
  agent_confidence DECIMAL(4,3),
  agent_explanation TEXT,
  recall_score DECIMAL(4,3) DEFAULT 0,
  timestamp BIGINT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_timestamp (timestamp),
  INDEX idx_user (user_id),
  INDEX idx_risk (risk_level),
  INDEX idx_agent_verdict (agent_verdict),
  INDEX idx_created (created_at)
);
```

### 4.2 `whitelist_rules` — 白名单规则

```sql
CREATE TABLE whitelist_rules (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  rule_name VARCHAR(256) NOT NULL,
  logic ENUM('AND','OR') DEFAULT 'OR',
  process_name VARCHAR(256),
  user_id VARCHAR(64),
  file_path_pattern VARCHAR(512),
  time_window_start TIME,
  time_window_end TIME,
  enabled BOOLEAN DEFAULT TRUE,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
```

### 4.3 `false_positive_library` — 误报库

```sql
CREATE TABLE false_positive_library (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  scenario_key VARCHAR(512) NOT NULL UNIQUE,
  host_id VARCHAR(64),
  user_id VARCHAR(64),
  sensitive_type VARCHAR(128),
  risk_level VARCHAR(32),
  process_name VARCHAR(256),
  process_path VARCHAR(512),
  target VARCHAR(512),
  operation VARCHAR(64),
  reason TEXT,
  embedding_json LONGTEXT,
  hit_count INT DEFAULT 1,
  last_seen_at DATETIME,
  expired_at DATETIME NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_expired (expired_at),
  INDEX idx_user (user_id),
  INDEX idx_process (process_name),
  INDEX idx_last_seen (last_seen_at)
);
```

`embedding_json` 仅作为后续可选增强字段保留，当前结构化召回主流程不依赖该字段。

---

## 五、API 接口设计

### 5.1 日志接收接口（模块二 → 模块三）

```http
POST /api/client/events
Content-Type: application/json
```

**请求**：
```json
{
  "host_id": "host_001",
  "events": [
    {
      "event_id": "evt_001",
      "host_id": "host_001",
      "user_id": "user_001",
      "file_path": "C:/Users/A/Desktop/customer.xlsx",
      "file_hash": "xxx",
      "sensitive": true,
      "sensitive_type": "客户资料",
      "risk_level": "high",
      "process_name": "chrome.exe",
      "process_path": "C:/Program Files/Google/Chrome/Application/chrome.exe",
      "target": "mail.qq.com",
      "operation": "upload",
      "timestamp": 177777934,
      "sensitive_file_id": "file_001"
    }
  ]
}
```

**响应**：
```json
{
  "accepted": 1,
  "status": "ok"
}
```

### 5.2 告警查询接口（模块四调用）

```http
POST /api/alerts/query
Content-Type: application/json
```

**请求**：
```json
{
  "start_time": "2026-06-01T00:00:00Z",
  "end_time": "2026-06-05T23:59:59Z",
  "risk_level": "high",
  "user_id": "user_001",
  "sensitive_type": "客户资料",
  "process_name": "chrome.exe",
  "operation": "upload",
  "page": 1,
  "page_size": 20,
  "order_by": "timestamp",
  "order": "desc"
}
```

**响应**：
```json
{
  "total": 150,
  "page": 1,
  "page_size": 20,
  "data": [
    {
      "event_id": "evt_001",
      "risk_level": "medium",
      "old_risk_level": "high",
      "user_id": "user_001",
      "sensitive_type": "客户资料",
      "process_name": "chrome.exe",
      "operation": "upload",
      "target": "internal-crm.company.com",
      "agent_verdict": "uncertain",
      "agent_confidence": 0.73,
      "recall_score": 0.68,
      "agent_explanation": "与历史 CRM 上传场景相似，但目标字段存在差异，因此仅降一级且不写入误报库。",
      "false_positive_reason": "",
      "timestamp": 177777934
    }
  ]
}
```

查询响应必须返回 `agent_verdict`、`agent_confidence`、`agent_explanation`、`recall_score`，方便模块四展示误报分析依据。

### 5.3 白名单管理接口

```http
GET    /api/whitelist          # 列表（支持分页）
POST   /api/whitelist          # 新增规则
PUT    /api/whitelist/:id      # 更新规则
DELETE /api/whitelist/:id      # 删除规则
```

### 5.4 误报库管理接口

```http
GET    /api/false-positives          # 列表
POST   /api/false-positives          # 新增/更新误报模式（按 scenario_key 去重）
DELETE /api/false-positives/:id      # 手动删除误报记录
```

---

## 六、项目结构

```
module3-alert-agent/
├── cmd/
│   └── server/
│       └── main.go              # 服务入口
├── internal/
│   ├── config/
│   │   └── config.go            # 配置（DB、方舟API、端口等）
│   ├── handler/
│   │   ├── event.go             # /api/client/events 处理
│   │   ├── query.go             # /api/alerts/query 处理
│   │   └── whitelist.go         # 白名单 CRUD
│   ├── pipeline/
│   │   ├── pipeline.go          # 流水线编排（白名单→去重→Agent）
│   │   ├── whitelist.go         # 白名单过滤 + 热缓存
│   │   └── dedup.go             # 去重合并逻辑
│   ├── agent/
│   │   ├── agent.go             # Eino ADK Agent 初始化
│   │   ├── tools.go             # 4 个 Tool 实现
│   │   ├── prompt.go            # System Prompt
│   │   └── recall.go            # 结构化召回逻辑
│   ├── model/
│   │   ├── event.go             # 日志事件结构体
│   │   ├── whitelist.go         # 白名单结构体
│   │   └── falsepositive.go     # 误报库结构体
│   ├── store/
│   │   ├── mysql.go             # MySQL 连接 & 初始化
│   │   ├── alert_store.go       # alert_logs 表操作
│   │   ├── whitelist_store.go   # whitelist_rules 表操作
│   │   └── fp_store.go          # false_positive_library 表操作
│   └── router/
│       └── router.go            # Hertz 路由注册
├── sql/
│   └── schema.sql               # 建表 DDL
├── config.yaml                  # 配置文件
├── go.mod
├── go.sum
└── README.md
```

---

## 七、关键技术风险与注意事项

### 7.1 当前已知风险

| 风险 | 等级 | 应对 |
|------|------|------|
| LLM API 限流 | 中 | 先测试并发上限，设固定并发池 |
| 结构化召回误匹配 | 中 | 使用 `recall_score` 阈值、Top-5 候选和 Agent 二次精判 |
| Agent 判断不稳定 | 中 | `confidence < 0.8` 不写误报库；空召回不写误报库；定期人工抽检 |
| 误报库重复膨胀 | 中 | 使用 `scenario_key` 唯一键，重复场景只更新 `hit_count`、`last_seen_at`、`expired_at` |
| 模块二接口格式未知 | 低 | `/api/client/events` events 是数组，兼容单条和批量 |
| 白名单规则膨胀 | 低 | TBD后续迭代加规则审计和去重 |

### 7.2 System Prompt 调优

System Prompt 中有 `[TODO]` 标记的部分需要在测试后调整：
- 误报判断标准可能需要增删
- confidence 阈值可能需要调高/调低
- 需要根据实际误报案例补充 few-shot 示例

---

## 八、实施步骤

### Phase 1：基础骨架（2-3 天）

- [ ] 初始化 Go 项目 + go mod
- [ ] MySQL 建表（`sql/schema.sql`）
- [ ] Hertz 服务启动（端口 9090）
- [ ] 配置文件加载（DB、方舟 API Key）
- [ ] 路由注册

### Phase 2：规则引擎（2-3 天）

- [ ] 白名单 CRUD API
- [ ] 白名单热缓存（启动加载 + API 同步刷新）
- [ ] 白名单过滤逻辑
- [ ] 去重合并（精确匹配 + 滑动窗口）
- [ ] `POST /api/client/events` → 流水线（白名单 → 去重）

### Phase 3：AI Agent（3-4 天）

- [ ] Eino ADK 集成 + 方舟 ChatModel 配置
- [ ] 结构化召回组件配置
- [ ] 4 个 Tool 实现
- [ ] System Prompt 加载
- [ ] Agent 精判逻辑（结构化召回 → ReAct）
- [ ] 服务端按 `scenario_key` 去重写入/更新误报库 + TTL

### Phase 4：查询 API + 联调（2-3 天）

- [ ] `POST /api/alerts/query` 通用查询 API
- [ ] 流水线端到端集成
- [ ] 压测并发上限 → 设定并发池大小
- [ ] 与模块二接口对齐（格式确认）
- [ ] 与模块四接口对齐（查询能力确认）

### Phase 5：测试与文档（2 天）

- [ ] 单元测试（白名单、去重核心逻辑）
- [ ] Agent 效果测试（构造误报样例验证）
- [ ] API 文档（README.md）
- [ ] 答辩材料准备

---

## 九、配置项清单

```yaml
# config.yaml
server:
  port: 9090

mysql:
  host: "127.0.0.1"
  port: 3306
  user: "root"
  password: "your_password"
  database: "dlp_agent"

ark:
  chat_model_endpoint: "set-via-env"
  api_key: "rotate-this-outside-code"

pipeline:
  dedup_windows:
    critical: 30
    high: 60
    medium: 180
    low: 300
    info: 600

agent:
  max_concurrency: 5          # 先设5，测试后调整
  recall_strategy: "structured"
  top_k_recall: 5
  structured_recall_threshold: 0.60
  strong_recall_threshold: 0.75
  confidence_threshold: 0.8
  false_positive_ttl_days: 30
```

---

## 十、依赖清单

| 依赖 | 用途 | 版本 |
|------|------|------|
| Go | 语言 | ≥ 1.21 |
| MySQL | 数据库 | 8.0+ |
| github.com/cloudwego/eino | Agent 框架 | latest |
| github.com/cloudwego/hertz | HTTP 框架 | latest |
| github.com/cloudwego/eino-ext/components/model/ark | 方舟 ChatModel | latest |
| github.com/go-sql-driver/mysql | MySQL 驱动 | latest |

---

> **下一步**：请分配 Phase 1 任务开始编码，或根据需要调整任何决策。
