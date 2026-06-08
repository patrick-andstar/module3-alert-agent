# Module 3 Alert Agent

模块三是“终端敏感数据外发监控系统”的告警日志误报分析服务。它接收模块二上报的终端告警，按下面顺序处理：

```text
白名单过滤 -> 去重合并 -> 结构化召回 -> Eino ReAct Agent 精判 -> MySQL 入库 -> 查询 API / 前端展示
```

当前主流程不依赖 Embedding；误报召回使用结构化字段打分，Agent 仍是必选项。

## 0. 最短运行路径

如果只是想尽快确认“能跑通、能展示”，按这个顺序来：

```powershell
Set-Location C:\Users\11832\Desktop\方案规划\module3-alert-agent

$env:MYSQL_HOST="127.0.0.1"
$env:MYSQL_PORT="3306"
$env:MYSQL_USER="root"
$env:MYSQL_PASSWORD="<你的 MySQL 密码>"
$env:MYSQL_DATABASE="dlp_agent"

$env:ARK_CHAT_MODEL="<你的 Ark endpoint，例如 ep-xxxx>"
$env:ARK_API_KEY="<你的 Ark API Key>"

$env:MYSQL_PWD=$env:MYSQL_PASSWORD
mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --execute="CREATE DATABASE IF NOT EXISTS dlp_agent DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"

powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1
go run ./cmd/server
```

服务启动后打开：

```text
http://localhost:9090/
```

推荐演示路线：

```text
预置误报模式 -> 召回命中后 Agent 精判 -> 白名单命中丢弃 -> 去重窗口合并 -> 空召回 Agent 判断 -> 疑似误报 -> 非误报真实告警
```

一键验收脚本看到下面输出，就说明 MySQL、API、前端页面、白名单、去重、误报库、Agent 精判和查询 API 都通过了基础联调：

```text
E2E smoke passed against http://127.0.0.1:9090
```

## 1. 技术栈

- Go 1.25+
- CloudWeGo Hertz
- CloudWeGo Eino / ReAct Agent
- 火山方舟 Ark ChatModel
- MySQL 8.0+
- 内置静态前端控制台：`GET /`

## 2. 目录结构

```text
module3-alert-agent/
├── cmd/server/main.go
├── internal/
│   ├── agent/      # Eino Runtime、ReAct Agent、结构化召回、决策矩阵
│   ├── config/     # config.yaml + 环境变量覆盖
│   ├── model/      # Event、Whitelist、FalsePositive 等模型
│   ├── pipeline/   # 白名单过滤、去重合并
│   ├── router/     # Hertz 路由、API handler、前端静态资源
│   └── store/      # MySQL 访问、SQL 构造、扫描逻辑
├── internal/router/web/ # 前端控制台
├── sql/schema.sql       # 新库建表
├── sql/upgrade.sql      # 旧库补字段/索引
├── tools/e2e-smoke.ps1  # 端到端验收脚本
├── config.yaml
└── README.md
```

## 3. 环境准备

需要提前准备：

- Go：执行 `go version` 能看到 `go1.25.x` 或兼容版本。
- MySQL 8.0+：执行 `mysql --version` 能看到客户端版本。
- Ark ChatModel：需要可用的方舟 endpoint 和 API Key。
- PowerShell：后续命令默认在 Windows PowerShell 中执行。

### 3.1 启动 MySQL

Windows 管理员终端中执行：

```cmd
net start mysql
```

确认可以登录：

```cmd
mysql -uroot -p
```

进入 MySQL 后确认或创建数据库：

```sql
SHOW DATABASES;
CREATE DATABASE IF NOT EXISTS dlp_agent DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
```

### 3.2 设置 PowerShell 环境变量

在 `module3-alert-agent` 目录中打开 PowerShell，设置本次运行所需变量：

```powershell
Set-Location C:\Users\11832\Desktop\方案规划\module3-alert-agent

$env:MYSQL_HOST="127.0.0.1"
$env:MYSQL_PORT="3306"
$env:MYSQL_USER="root"
$env:MYSQL_PASSWORD="<你的 MySQL 密码>"
$env:MYSQL_DATABASE="dlp_agent"

$env:ARK_CHAT_MODEL="<你的 Ark endpoint，例如 ep-xxxx>"
$env:ARK_API_KEY="<你的 Ark API Key>"
```

可选：如果设置了管理 token，前端顶部也要填同一个 token，否则查询、白名单和误报库管理接口会返回 `401 unauthorized`。

```powershell
$env:ADMIN_API_TOKEN="<本机演示用 token，可不设置>"
```

可选：如果 Ark 响应较慢，演示时可以提高 Agent 超时时间：

```powershell
$env:AGENT_ANALYSIS_TIMEOUT_SEC="90"
```

不要把真实密码或 API Key 写入 `config.yaml`、README、测试快照或提交记录。

## 4. 初始化或升级数据库

新库和旧库都推荐执行下面两步：

```powershell
$env:MYSQL_PWD=$env:MYSQL_PASSWORD
mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --execute="CREATE DATABASE IF NOT EXISTS dlp_agent DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;"
Get-Content -Raw .\sql\schema.sql | mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --database=$env:MYSQL_DATABASE
Get-Content -Raw .\sql\upgrade.sql | mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --database=$env:MYSQL_DATABASE
```

作用：

- `schema.sql`：创建 `alert_logs`、`whitelist_rules`、`false_positive_library`。
- `upgrade.sql`：如果库里已经有旧表，补齐 `agent_verdict`、`agent_confidence`、`agent_explanation`、`recall_score`、`scenario_key`、`hit_count`、`last_seen_at` 等字段和索引。

快速核验：

```powershell
mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --database=$env:MYSQL_DATABASE --execute="SHOW TABLES;"
```

应该能看到：

```text
alert_logs
false_positive_library
whitelist_rules
```

## 5. 一键端到端验收

推荐先跑自动验收脚本：

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1
```

只检查环境变量和 `mysql.exe` 是否能找到，不启动服务：

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1 -PreflightOnly
```

脚本会自动完成：

- 应用 `sql/schema.sql` 和 `sql/upgrade.sql`。
- 启动 `go run ./cmd/server`。
- 等待 `/healthz` 可用。
- 检查 `GET /` 和 `GET /app.js`，确认前端固定场景存在。
- 调用白名单 API，验证白名单命中会丢弃事件。
- 写入误报库模式，验证 `scenario_key` upsert 和 `hit_count`。
- 上报告警，调用 Ark ReAct Agent 精判。
- 查询 `/api/alerts/query`，确认返回 `agent_verdict`、`agent_confidence`、`recall_score`。
- 读取 `alert_logs`、`false_positive_library`、`whitelist_rules` 三类核心记录。

成功输出类似：

```text
E2E smoke passed against http://127.0.0.1:9090
```

如果服务已经手动启动，可以让脚本复用正在运行的服务：

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1 -UseRunningServer
```

如果你已经手动初始化过数据库，只想复用当前表结构：

```powershell
powershell -ExecutionPolicy Bypass -File .\tools\e2e-smoke.ps1 -UseRunningServer -SkipSchema
```

## 6. 手动启动服务

手动演示时建议开两个 PowerShell：

- 窗口 A：启动服务，保持不要关闭。
- 窗口 B：发 API 请求、跑 SQL、执行测试命令。

窗口 A：

```powershell
go mod tidy
go run ./cmd/server
```

启动成功后默认监听：

```text
http://localhost:9090/
```

健康检查：

```powershell
Invoke-RestMethod http://localhost:9090/healthz
```

期望返回：

```json
{"status":"ok"}
```

窗口 B 可以用下面命令确认前端资源也能访问：

```powershell
Invoke-WebRequest http://localhost:9090/ -UseBasicParsing | Select-Object -ExpandProperty StatusCode
Invoke-WebRequest http://localhost:9090/app.js -UseBasicParsing | Select-Object -ExpandProperty StatusCode
```

两个命令都返回 `200`，说明浏览器页面所需资源已经就绪。

## 7. 前端演示流程

浏览器打开：

```text
http://localhost:9090/
```

前端是一个固定场景控制台，左侧按钮内置 JSON 输入，点击后会直接调用真实 API，并在右侧展示：

- 场景输入 JSON
- 每一步请求与响应
- `alert_logs` 查询结果
- `false_positive_library` 误报模式记录
- `whitelist_rules` 白名单记录

推荐展示顺序：

1. **预置误报模式**
   - 点击后写入 `false_positive_library`。
   - 观察 `scenario_key`、`hit_count`、`last_seen_at`。

2. **召回命中后 Agent 精判**
   - 先自动写入误报模式，再上报告警。
   - 观察 `agent_verdict=false_positive`、`agent_confidence`、`recall_score`。
   - 观察风险等级按规则降级。

3. **白名单命中丢弃**
   - 先写入白名单，再上报匹配事件。
   - 观察响应中的 `dropped` 增加。
   - 说明白名单命中不进入 `alert_logs`。

4. **去重窗口合并**
   - 批量上报相同 key 的告警。
   - 观察合并后的 `is_merge_event`、`file_count`、`files`。

5. **空召回 Agent 判断**
   - 不预置误报模式，仍调用 Agent。
   - 观察 `agent_verdict` 和解释字段。
   - 说明空召回不会写入误报库，最多降 2 级且不能降到 `info`。

6. **疑似误报**
   - 部分字段相似但证据不足。
   - 观察最多降 1 级，不能降到 `info`。

7. **非误报真实告警**
   - 外发目标明显风险。
   - 观察原风险等级保持，不自动升级。

### 7.1 展示时重点看哪里

页面右侧有三块结果，建议边点按钮边解释：

- `alert_logs`：看最终入库告警，重点字段是 `event_id`、`risk_level`、`old_risk_level`、`is_merge_event`、`file_count`、`agent_verdict`、`agent_confidence`、`agent_explanation`、`recall_score`。
- `false_positive_library`：看误报模式是否按 `scenario_key` 去重，重点字段是 `scenario_key`、`hit_count`、`last_seen_at`、`expired_at`。
- `whitelist_rules`：看白名单规则是否创建成功，重点字段是 `rule_name`、`logic`、`process_name`、`enabled`。

### 7.2 每个场景的讲解口径

| 场景 | 要证明什么 | 预期现象 |
|---|---|---|
| 预置误报模式 | 误报库支持主动写入和 `scenario_key` 去重 | `false_positive_library` 新增或更新同一个 `scenario_key`，重复点击会增加 `hit_count` |
| 召回命中后 Agent 精判 | 结构化召回 + ReAct Agent 是主判断链路 | `recall_score` 较高，`agent_verdict` 有值，误报时风险等级按规则下降 |
| 白名单命中丢弃 | 白名单是确定性过滤，优先级最高 | `/api/client/events` 返回 `dropped > 0`，对应事件不进入 `alert_logs` |
| 去重窗口合并 | 相同 key 在窗口内会聚合 | `is_merge_event=true`，`file_count` 大于 1，`files` 中有多个文件 |
| 空召回 Agent 判断 | 没有历史误报也必须走 Agent | `recall_score` 接近 0，但仍返回 `agent_verdict` 和解释 |
| 疑似误报 | 证据不足时不能直接写误报库 | 一般为 `uncertain`，不新增误报模式，最多有限降级 |
| 非误报真实告警 | 真实告警保持原风险等级 | `agent_verdict=true_alert`，`risk_level` 保持原等级，不自动升级 |

### 7.3 演示前清理或保留数据

默认不需要清理数据。前端场景里的 `event_id` 会带当前时间戳，重复点击也能生成新事件。

如果你想让页面更干净，可以只在演示前查看最近记录，而不是删除旧数据：

```sql
SELECT event_id, risk_level, agent_verdict, agent_confidence, recall_score, timestamp
FROM alert_logs
ORDER BY id DESC
LIMIT 20;
```

## 8. API 清单

```http
GET    /
GET    /healthz
POST   /api/client/events
POST   /api/alerts/query
GET    /api/whitelist
POST   /api/whitelist
PUT    /api/whitelist/:id
DELETE /api/whitelist/:id
GET    /api/false-positives
POST   /api/false-positives
DELETE /api/false-positives/:id
```

如果设置了 `ADMIN_API_TOKEN`，除 `/healthz` 和 `/api/client/events` 外，管理和查询接口需要请求头：

```http
Authorization: Bearer <ADMIN_API_TOKEN>
```

前端顶部的 `Admin Token` 输入框用于填这个 token。

### 8.1 手工调用 API 示例

创建白名单：

```powershell
$headers = @{}
if ($env:ADMIN_API_TOKEN) { $headers["Authorization"] = "Bearer $env:ADMIN_API_TOKEN" }

Invoke-RestMethod -Method POST -Uri http://localhost:9090/api/whitelist -Headers $headers -ContentType "application/json" -Body '{
  "rule_name": "demo-backup-whitelist",
  "logic": "OR",
  "process_name": "backup.exe",
  "enabled": true
}'
```

批量上报告警：

```powershell
$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
Invoke-RestMethod -Method POST -Uri http://localhost:9090/api/client/events -ContentType "application/json" -Body "{
  `"host_id`": `"host-demo-01`",
  `"events`": [
    {
      `"event_id`": `"evt-manual-$timestamp`",
      `"host_id`": `"host-demo-01`",
      `"user_id`": `"alice`",
      `"file_path`": `"C:/Users/Alice/Desktop/customer.xlsx`",
      `"file_hash`": `"hash-demo-customer`",
      `"sensitive`": true,
      `"sensitive_type`": `"customer`",
      `"risk_level`": `"high`",
      `"process_name`": `"dlp-demo-browser.exe`",
      `"process_path`": `"C:/DLPDemo/dlp-demo-browser.exe`",
      `"target`": `"internal-crm.company.com`",
      `"operation`": `"upload`",
      `"timestamp`": $timestamp,
      `"sensitive_file_id`": `"file-demo-001`"
    }
  ]
}"
```

查询告警：

```powershell
Invoke-RestMethod -Method POST -Uri http://localhost:9090/api/alerts/query -Headers $headers -ContentType "application/json" -Body '{
  "page": 1,
  "page_size": 20,
  "order_by": "timestamp",
  "order": "desc"
}'
```

写入误报模式：

```powershell
Invoke-RestMethod -Method POST -Uri http://localhost:9090/api/false-positives -Headers $headers -ContentType "application/json" -Body '{
  "scenario_key": "customer|upload|dlp-demo-browser.exe|internal-crm.company.com",
  "user_id": "alice",
  "sensitive_type": "customer",
  "risk_level": "low",
  "process_name": "dlp-demo-browser.exe",
  "process_path": "C:/DLPDemo/dlp-demo-browser.exe",
  "target": "internal-crm.company.com",
  "operation": "upload",
  "reason": "normal crm upload from trusted internal workflow",
  "hit_count": 1
}'
```

### 8.2 `/api/client/events` 入参要点

`events` 是数组，演示和模块二对接都建议按批量格式传。每条事件至少需要：

```text
event_id, host_id, user_id, process_name, sensitive_type, operation, risk_level, timestamp
```

`risk_level` 只能是：

```text
critical, high, medium, low, info
```

`timestamp` 使用 Unix 秒级时间戳，不能是未来超过 24 小时的时间。

### 8.3 `/api/alerts/query` 查询条件

支持字段：

```text
event_id, start_timestamp, end_timestamp, risk_level, user_id, sensitive_type, process_name, operation, page, page_size, order_by, order
```

分页规则：

- `page` 默认 `1`。
- `page_size` 默认 `20`，最大 `100`。
- `order_by` 支持 `timestamp`、`event_id`、`created_at`。
- `order` 支持 `asc`，其他值会按 `desc` 处理。

## 9. 数据库核验 SQL

查看最近告警：

```sql
SELECT event_id, process_name, risk_level, agent_verdict, agent_confidence, recall_score
FROM alert_logs
ORDER BY id DESC
LIMIT 10;
```

查看误报库模式：

```sql
SELECT scenario_key, hit_count, last_seen_at, expired_at
FROM false_positive_library
ORDER BY id DESC
LIMIT 10;
```

查看白名单：

```sql
SELECT rule_name, process_name, user_id, enabled
FROM whitelist_rules
ORDER BY id DESC
LIMIT 10;
```

确认旧库升级字段：

```sql
SELECT column_name
FROM information_schema.columns
WHERE table_schema='dlp_agent'
  AND table_name='alert_logs'
  AND column_name IN ('agent_verdict','agent_confidence','agent_explanation','recall_score');

SELECT column_name
FROM information_schema.columns
WHERE table_schema='dlp_agent'
  AND table_name='false_positive_library'
  AND column_name IN ('scenario_key','hit_count','last_seen_at');
```

## 10. 自动化测试

```powershell
go test ./... -count=1
go test -race ./internal/pipeline -run TestDeduperCanAcceptConcurrentEvents -count=1
git diff --check
```

如果要跑真实 MySQL 集成校验，需要先设置数据库环境变量，然后额外打开：

```powershell
$env:RUN_MYSQL_INTEGRATION="1"
go test ./internal/store -run TestMySQLIntegration -count=1
```

## 11. 当前误报判断规则

- 白名单是确定性规则，命中后直接丢弃，不写入 `alert_logs`。
- 去重 key：

```text
host_id + user_id + process_name + sensitive_type + operation
```

- 结构化召回字段：

```text
sensitive_type, operation, process_name, target, user_id, process_path
```

- `agent_verdict=true_alert`：
  - 保持原风险等级。
  - 不自动升级。
  - 不写误报库。

- 低置信度疑似误报：
  - `agent_verdict=uncertain`。
  - 最多降 1 级。
  - 不能降到 `info`。
  - 不写误报库。

- 空召回但 Agent 判断疑似误报：
  - `agent_verdict=uncertain`。
  - 最多降 2 级。
  - 不能降到 `info`。
  - 不写误报库。

- 强召回 + 高置信度误报：
  - 可降到 `low` 或 `info`。
  - 按 `scenario_key` upsert `false_positive_library`。
  - 重复场景累加 `hit_count`，更新 `last_seen_at`。

## 12. 常见问题

### MySQL 报 Access denied

确认环境变量里的用户名、密码和数据库名正确：

```powershell
mysql --host=$env:MYSQL_HOST --port=$env:MYSQL_PORT --user=$env:MYSQL_USER --database=$env:MYSQL_DATABASE -p
```

### 提示 `ark.chat_model_endpoint must be set via ARK_CHAT_MODEL`

设置 Ark endpoint：

```powershell
$env:ARK_CHAT_MODEL="<你的 ep-xxxx>"
```

### Agent 超时

先确认 Ark 直连可用，再提高本次演示超时：

```powershell
$env:AGENT_ANALYSIS_TIMEOUT_SEC="90"
```

当前服务已将 Ark ChatModel 配置为低温、限制输出 token、禁用长 thinking，以减少 ReAct 超时概率。

### 端口 9090 被占用

查询占用进程：

```powershell
netstat -ano | Select-String ':9090'
```

停止旧服务后再启动，或临时修改 `config.yaml` 的 `server.port`。

### 前端按钮点击后没有记录变化

优先看右侧“执行输出”里的请求响应。常见原因：

- Admin Token 未填写。
- 事件被白名单命中丢弃。
- Ark Agent 调用失败。
- MySQL 表是旧结构但未执行 `sql/upgrade.sql`。

## 13. 演示建议话术

可以按这个顺序讲：

1. “模块三不是简单入库，它先做白名单确定性过滤，再做去重合并。”
2. “Embedding 当前可去掉，误报召回用结构化打分，但 Agent 精判保留。”
3. “Agent 不直接写误报库，最终由服务端根据 recall score、confidence、reason 和 scenario_key 控制写入。”
4. “真实告警保持原风险等级，不自动升级；疑似误报最多降级，且不能随便降到 info。”
5. “误报库通过 scenario_key 去重，重复场景累加 hit_count，避免相似数据膨胀。”
6. “模块四可以通过 `/api/alerts/query` 拿到告警和 Agent 分析字段。”
