const nowSeconds = () => Math.floor(Date.now() / 1000);

const eventBase = {
  host_id: "host-demo-01",
  user_id: "alice",
  file_path: "C:/Users/Alice/Desktop/customer.xlsx",
  file_hash: "hash-demo-customer",
  sensitive: true,
  sensitive_type: "customer",
  risk_level: "high",
  process_name: "dlp-demo-browser.exe",
  process_path: "C:/DLPDemo/dlp-demo-browser.exe",
  target: "internal-crm.company.com",
  operation: "upload",
  sensitive_file_id: "file-demo-001"
};

const scenarios = [
  {
    id: "whitelist_drop",
    title: "白名单命中丢弃",
    summary: "先写入白名单规则，再上报 backup.exe 告警，验证不进入 alert_logs。",
    steps: () => [
      {
        label: "创建白名单",
        method: "POST",
        path: "/api/whitelist",
        body: {
          rule_name: "demo-backup-whitelist",
          logic: "OR",
          process_name: "backup.exe",
          enabled: true
        }
      },
      {
        label: "上报告警",
        method: "POST",
        path: "/api/client/events",
        body: {
          host_id: "host-demo-01",
          events: [
            {
              ...eventBase,
              event_id: `evt-whitelist-${Date.now()}`,
              process_name: "backup.exe",
              process_path: "C:/Backup/backup.exe",
              target: "D:/Backup/customer.xlsx",
              timestamp: nowSeconds()
            }
          ]
        }
      }
    ]
  },
  {
    id: "dedup_merge",
    title: "去重窗口合并",
    summary: "同一 host/user/process/type/operation 在窗口内重复上报，验证聚合输出。",
    steps: () => {
      const stamp = nowSeconds();
      const group = `evt-dedup-${Date.now()}`;
      return [
        {
          label: "批量上报重复告警",
          method: "POST",
          path: "/api/client/events",
          body: {
            host_id: "host-demo-01",
            events: [
              {
                ...eventBase,
                event_id: `${group}-a`,
                file_path: "C:/Users/Alice/Desktop/customer-a.xlsx",
                file_hash: "hash-a",
                timestamp: stamp
              },
              {
                ...eventBase,
                event_id: `${group}-b`,
                file_path: "C:/Users/Alice/Desktop/customer-b.xlsx",
                file_hash: "hash-b",
                timestamp: stamp + 20
              }
            ]
          }
        }
      ];
    }
  },
  {
    id: "seed_false_positive",
    title: "预置误报模式",
    summary: "写入 false_positive_library，供后续结构化召回命中。",
    steps: () => [
      {
        label: "写入误报模式",
        method: "POST",
        path: "/api/false-positives",
        body: {
          scenario_key: "customer|upload|dlp-demo-browser.exe|internal-crm.company.com",
          user_id: "alice",
          sensitive_type: "customer",
          risk_level: "low",
          process_name: "dlp-demo-browser.exe",
          process_path: "C:/DLPDemo/dlp-demo-browser.exe",
          target: "internal-crm.company.com",
          operation: "upload",
          reason: "normal crm upload from trusted internal workflow",
          hit_count: 1
        }
      }
    ]
  },
  {
    id: "confirmed_false_positive",
    title: "召回命中后 Agent 精判",
    summary: "先预置误报模式，再上报完全匹配事件，验证 recall_score 与 Agent 输出。",
    steps: () => {
      const timestamp = nowSeconds();
      return [
        ...scenarios.find((item) => item.id === "seed_false_positive").steps(),
        {
          label: "上报匹配误报模式的告警",
          method: "POST",
          path: "/api/client/events",
          body: {
            host_id: "host-demo-01",
            events: [
              {
                ...eventBase,
                event_id: `evt-fp-${Date.now()}`,
                timestamp
              }
            ]
          }
        }
      ];
    }
  },
  {
    id: "uncertain_candidate",
    title: "疑似误报",
    summary: "字段部分相似但 target 不同，验证低/中证据场景的展示。",
    steps: () => [
      {
        label: "上报部分相似告警",
        method: "POST",
        path: "/api/client/events",
        body: {
          host_id: "host-demo-01",
          events: [
            {
              ...eventBase,
              event_id: `evt-uncertain-${Date.now()}`,
              target: "partner-crm.example.com",
              timestamp: nowSeconds()
            }
          ]
        }
      }
    ]
  },
  {
    id: "empty_recall_agent_judgement",
    title: "空召回 Agent 判断",
    summary: "使用全新业务字段，不预置误报模式，验证空召回仍调用 Agent 且不写误报库。",
    steps: () => [
      {
        label: "上报无历史模式告警",
        method: "POST",
        path: "/api/client/events",
        body: {
          host_id: "host-demo-02",
          events: [
            {
              ...eventBase,
              event_id: `evt-empty-recall-${Date.now()}`,
              host_id: "host-demo-02",
              user_id: "bob",
              file_path: "C:/Users/Bob/Desktop/legal-contract.pdf",
              file_hash: "hash-legal-contract",
              sensitive_type: "legal_contract",
              process_name: "edge.exe",
              process_path: "C:/Program Files/Microsoft/Edge/Application/msedge.exe",
              target: "legal-review.internal",
              operation: "upload",
              risk_level: "critical",
              sensitive_file_id: "file-demo-legal",
              timestamp: nowSeconds()
            }
          ]
        }
      }
    ]
  },
  {
    id: "true_alert",
    title: "非误报真实告警",
    summary: "敏感文件上传外部邮箱，验证真实告警进入 alert_logs 且保留风险等级。",
    steps: () => [
      {
        label: "上报外发告警",
        method: "POST",
        path: "/api/client/events",
        body: {
          host_id: "host-demo-01",
          events: [
            {
              ...eventBase,
              event_id: `evt-true-${Date.now()}`,
              target: "mail.qq.com",
              operation: "upload",
              risk_level: "critical",
              timestamp: nowSeconds()
            }
          ]
        }
      }
    ]
  }
];

const elements = {
  list: document.querySelector("#scenarioList"),
  input: document.querySelector("#scenarioInput"),
  output: document.querySelector("#runOutput"),
  state: document.querySelector("#runState"),
  badge: document.querySelector("#scenarioBadge"),
  title: document.querySelector("#activeTitle"),
  token: document.querySelector("#adminToken"),
  alerts: document.querySelector("#alertsOutput"),
  fps: document.querySelector("#fpOutput"),
  whitelist: document.querySelector("#whitelistOutput"),
  alertCount: document.querySelector("#alertCount"),
  fpCount: document.querySelector("#fpCount"),
  whitelistCount: document.querySelector("#whitelistCount"),
  refresh: document.querySelector("#refreshButton")
};

function pretty(value) {
  return JSON.stringify(value, null, 2);
}

function headers() {
  const result = { "Content-Type": "application/json" };
  const token = elements.token.value.trim();
  if (token) {
    result.Authorization = `Bearer ${token}`;
  }
  return result;
}

async function request(step) {
  const response = await fetch(step.path, {
    method: step.method,
    headers: headers(),
    body: step.body ? JSON.stringify(step.body) : undefined
  });
  const text = await response.text();
  let parsed = text;
  try {
    parsed = JSON.parse(text);
  } catch (_) {
    parsed = text;
  }
  return {
    label: step.label,
    request: {
      method: step.method,
      path: step.path,
      body: step.body || null
    },
    status: response.status,
    response: parsed
  };
}

async function runScenario(scenario) {
  document.body.classList.remove("is-error", "is-ok");
  document.body.classList.add("is-running");
  elements.state.textContent = "running";
  elements.badge.textContent = scenario.id;
  elements.title.textContent = scenario.title;
  const steps = scenario.steps();
  elements.input.textContent = pretty(steps.map((step) => ({
    label: step.label,
    method: step.method,
    path: step.path,
    body: step.body
  })));
  elements.output.textContent = "执行中...";

  try {
    const results = [];
    for (const step of steps) {
      results.push(await request(step));
    }
    elements.output.textContent = pretty(results);
    document.body.classList.remove("is-running");
    document.body.classList.add("is-ok");
    elements.state.textContent = "ok";
    await refreshRecords();
  } catch (error) {
    document.body.classList.remove("is-running");
    document.body.classList.add("is-error");
    elements.state.textContent = "error";
    elements.output.textContent = error.stack || String(error);
  }
}

async function getJSON(path, fallback) {
  const response = await fetch(path, { headers: headers() });
  if (!response.ok) {
    return fallback;
  }
  return response.json();
}

async function refreshRecords() {
  const [alerts, fps, whitelist] = await Promise.all([
    request({
      label: "查询告警",
      method: "POST",
      path: "/api/alerts/query",
      body: { page: 1, page_size: 20, order_by: "timestamp", order: "desc" }
    }).then((item) => item.response),
    getJSON("/api/false-positives", []),
    getJSON("/api/whitelist", [])
  ]);

  const alertRows = Array.isArray(alerts.data) ? alerts.data : [];
  elements.alerts.textContent = pretty(alerts);
  elements.fps.textContent = pretty(fps);
  elements.whitelist.textContent = pretty(whitelist);
  elements.alertCount.textContent = String(alerts.total || alertRows.length || 0);
  elements.fpCount.textContent = String(Array.isArray(fps) ? fps.length : 0);
  elements.whitelistCount.textContent = String(Array.isArray(whitelist) ? whitelist.length : 0);
}

function renderScenarios() {
  elements.list.innerHTML = "";
  for (const scenario of scenarios) {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "scenario";
    button.innerHTML = `<strong>${scenario.title}</strong><small>${scenario.summary}</small>`;
    button.addEventListener("click", () => runScenario(scenario));
    elements.list.appendChild(button);
  }
}

elements.refresh.addEventListener("click", refreshRecords);
renderScenarios();
refreshRecords();
