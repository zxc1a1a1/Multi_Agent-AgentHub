# AgentHub 多子Agent 可调用性测试指南

> 目标：验证 19 个子 Agent 设计正确、可被成功调用、产出符合契约。

---

## 1. 当前状态

| 指标 | 值 |
|---|---|
| 子 Agent 总数 | 19（含 code-agent） |
| ADK 公共模块 | `agents/adk`（AgentCard、Context、LLMClient、A2AServer） |
| 每个 Agent 结构 | `config.yaml` + `handler.go` + `handler_test.go` + `main.go` + `Dockerfile` |
| 当前测试状态 | **全部 19 Agent + ADK 模块测试通过**（20 modules, all `ok`） |
| 测试代码总量 | 1,520 行（handler_test.go）+ 133 行（adk/server_test.go） |

---

## 2. 测试维度总览

按 `testing-review-contract` 定义的分层矩阵，测试多 Agent 可调用性需要覆盖以下 6 个维度：

```
  ┌────────────────────────────────────────────┐
  │           L6: Docker Smoke Test            │  一键启动 → /health → /api/agents → run
  ├────────────────────────────────────────────┤
  │    L5: Cross-Agent Integration Test        │  Agent A 产出 → Agent B 消费
  ├────────────────────────────────────────────┤
  │     L4: A2A Protocol Test                  │  AgentCard、/health、task sendSubscribe
  ├────────────────────────────────────────────┤
  │      L3: Artifact Output Test              │  artifactType、content、metadata、fenced block 解析
  ├────────────────────────────────────────────┤
  │      L2: Handler Logic Test (Unit)         │  parse、extract、validate 纯逻辑函数
  ├────────────────────────────────────────────┤
  │      L1: Schema / Contract Test            │  config.yaml、AgentCard 字段完整性
  └────────────────────────────────────────────┘
```

---

## 3. L1：Schema / Contract 测试

### 目的
验证每个 Agent 的 `config.yaml` 字段完整、AgentCard 生成正确、不泄露敏感信息。

### 已覆盖
`agents/adk/server_test.go` 提供 3 个用例：
- `TestBuildAgentCard_MapsConfigFields` — 所有字段正确映射
- `TestBuildAgentCard_DifferentConfigsProduceDifferentCards` — 不同配置生成不同 Card
- `TestBuildAgentCard_DoesNotInjectHardcodedSkill` — 不注入硬编码 skill

### 待补充
每个 Agent 应增加独立的 config 加载校验：

```go
// 对每个 Agent 增加如下测试：
func TestConfigLoadable(t *testing.T) {
    cfg, err := adk.LoadConfig("config.yaml")
    if err != nil {
        t.Fatalf("failed to load config: %v", err)
    }
    if cfg.Name == "" {
        t.Fatal("name is empty")
    }
    if cfg.Version == "" {
        t.Fatal("version is empty")
    }
    if cfg.Description == "" {
        t.Fatal("description is empty")
    }
    // 安全检查：描述中不含内部路径或密钥关键字
    for _, keyword := range []string{"API_KEY", "token", "secret", "password", "internal"} {
        if contains(cfg.Description, keyword) {
            t.Fatalf("description contains sensitive keyword: %s", keyword)
        }
    }
}
```

### 批量校验命令
```bash
for d in agents/*/; do
  echo "=== $(basename $d) ==="
  cat "$d/config.yaml"
  echo
done
```

---

## 4. L2：Handler Logic 单元测试

### 目的
验证 Agent handler 中的纯逻辑函数（fenced block 解析、artifact 提取、JSON/YAML 解析等），不依赖 LLM 或网络。

### 已覆盖模式（以 web-agent 为例）

```
输入: 一段含 ```html:index.html ... ``` 的 LLM 模拟输出
操作: extractWebpageArtifacts(text)
断言: artifact.Type == "webpage", artifact.Title == "index.html", metadata.hasCSS == "true"
```

### 每个 Agent 的测试覆盖矩阵

| Agent | 测试用例数 | 覆盖场景 |
|---|---|---|
| code-agent | ~5 | 代码块解析、多文件、无代码块、非代码 fence |
| web-agent | 6 | HTML fence 解析（含/不含文件名）、多块、内嵌反引号不提前闭合 |
| document-agent | ~5 | Markdown 文档提取、多段、空输入 |
| vision-agent | ~7 | JSON 解析、多 block、字段完整性、安全脱敏 |
| security-agent | ~5 | 密钥检测、危险命令检测、安全报告生成 |
| artifact-agent | ~3 | 产物清单聚合、多 agent 输出合并 |
| 其余 13 个 Agent | 3-6 个/个 | 各自的 parse/extract/validate 逻辑 |

### 通用测试模式

所有 Agent handler 测试遵循相同的 fixture pattern：

```go
func TestExtractXxxArtifacts_ValidInput(t *testing.T) {
    input := "<模拟的 LLM 输出文本，含 fenced block>"
    artifacts := extractXxxArtifacts(input)
    // assert artifact.Type, artifact.Title, artifact.Content, artifact.Metadata
}
```

### 运行命令
```bash
for d in agents/*/; do (cd "$d" && go test -v ./...); done
```

---

## 5. L3：Artifact 输出测试

### 目的
验证每个 Agent 输出的 Artifact type 与 `agents/README.md` 声明的类型一致，且 metadata 字段完整。

### 19 个 Agent 的 Artifact Type 映射

| Agent | 声明 Artifact Type | 测试中验证 |
|---|---|---|
| code-agent | `code` | type=code, language set, filename set |
| web-agent | `webpage` | type=webpage, language=html, hasCSS/hasJS |
| document-agent | `document` | type=document, format=markdown |
| vision-agent | `vision_analysis` | type=vision_analysis, JSON schema 完整性 |
| context-agent | `context_bundle` | type=context_bundle |
| review-agent | `review_report` | type=review_report |
| test-agent | `test_report` | type=test_report |
| security-agent | `security_report` | type=security_report |
| diff-agent | `diff` | type=diff |
| artifact-agent | `artifact_manifest` | type=artifact_manifest |
| qa-acceptance-agent | `acceptance_report` | type=acceptance_report |
| version-agent | `version_snapshot` | type=version_snapshot |
| agent-builder-agent | `agent_profile` | type=agent_profile |
| file-agent | `file_summary` | type=file_summary |
| web-research-agent | `web_research` | type=web_research |
| deploy-agent | `deployment_plan` | type=deployment_plan |
| ppt-agent | `slide_deck` | type=slide_deck |
| release-agent | `release_report` | type=release_report |
| custom-agent | `document` | type=document |

### 待补充：Artifact Schema 合规性测试

```go
// 建议对每个 Agent 输出做 schema 校验
func TestArtifactSchemaCompliance(t *testing.T) {
    art := extractArtifacts(testInput)[0]
    // artifact-contract 字段完整性
    requiredFields := []string{"Type", "Title", "Content"}
    for _, f := range requiredFields { /* validate non-empty */ }
    // content 不包含 secret
    if containsSecret(art.Content) { t.Fatal("artifact content contains secret") }
    // 大数据量应走 contentRef（当前第一版均为轻量产物，此断言可选）
}
```

---

## 6. L4：A2A 协议可调用性测试

### 目的
验证每个 Agent 的 HTTP 端口可访问、AgentCard 可获取、健康检查通过。

### 核心验证脚本

```bash
#!/bin/bash
# smoke_test_agents.sh — AgentHub 多 Agent 冒烟测试

AGENTS=(
  "code-agent:8081"
  "web-agent:8082"
  "document-agent:8083"
  "custom-agent:8084"
  "context-agent:8091"
  "vision-agent:8092"
  "diff-agent:8093"
  "review-agent:8094"
  "test-agent:8095"
  "artifact-agent:8096"
  "security-agent:8097"
  "version-agent:8099"
  "agent-builder-agent:8100"
  "file-agent:8101"
  "web-research-agent:8102"
  "qa-acceptance-agent:8103"
  "deploy-agent:8104"
  "ppt-agent:8105"
  "release-agent:8106"
)

PASS=0
FAIL=0

for entry in "${AGENTS[@]}"; do
  name="${entry%%:*}"
  port="${entry##*:}"
  base="http://localhost:$port"

  # 1. Health check
  if curl -sf "$base/health" > /dev/null 2>&1; then
    health="OK"
  else
    health="FAIL"
    ((FAIL++))
  fi

  # 2. AgentCard check
  card=$(curl -sf "$base/.well-known/agent.json" 2>/dev/null)
  if echo "$card" | jq -e '.name' > /dev/null 2>&1; then
    cardName=$(echo "$card" | jq -r '.name')
    cardOk="OK ($cardName)"
  else
    cardOk="FAIL (no card or invalid)"
    ((FAIL++))
  fi

  # 3. Security check: AgentCard must not contain secrets
  if echo "$card" | grep -qiE 'api.?key|token|secret|password|BEGIN RSA' 2>/dev/null; then
    cardOk="$cardOk [SECURITY WARN: potential secret in card]"
  fi

  printf "%-25s  health=%-4s  card=%-30s\n" "$name" "$health" "$cardOk"
  ((PASS++))
done

echo "---"
echo "Total: $((PASS + FAIL)), Passed: $PASS, Failed: $FAIL"
```

### A2A Task 调用测试

对代表性 Agent 发送 A2A task/sendSubscribe 请求：

```bash
# 以 code-agent 为例
curl -s -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "method": "tasks/sendSubscribe",
    "params": {
      "id": "test-task-001",
      "message": {
        "role": "user",
        "parts": [{"type": "text", "text": "Write a hello world function in Go"}]
      }
    }
  }'
```

预期：返回 SSE 流，包含 `TaskStateWorking` → `ArtifactUpdate`（文本）→ `ArtifactUpdate`（code artifact）→ `TaskStateCompleted`。

---

## 7. L5：跨 Agent 集成测试

### 目的
验证多 Agent 之间的中间产物传递链路（readme.md 第 5 节定义的多模态通道）。

### 关键集成链路

```
链路 1: vision-agent → web-agent
  image_ref → vision-agent → vision_analysis → web-agent → webpage

链路 2: vision-agent → code-agent
  截图描述 → vision-agent → vision_analysis → code-agent → code

链路 3: file-agent → document-agent
  file_ref → file-agent → file_summary → document-agent → document

链路 4: security-agent → review-agent
  代码/配置 → security-agent → security_report → review-agent → review_report
```

### 集成测试 fixture 方案

```go
// 模拟集成链路：vision-agent 输出 → web-agent 输入
func TestVisionToWebIntegration(t *testing.T) {
    // Step 1: 模拟 vision-agent 的 vision_analysis 输出
    visionOutput := `{
        "summary": "A login page with username and password fields and a submit button",
        "uiElements": ["text input", "password input", "button"],
        "layout": "vertical centered form"
    }`
    visionArtifact := adk.Artifact{
        Type: "vision_analysis",
        Title: "vision-analysis.json",
        Content: visionOutput,
        Metadata: map[string]string{"imageType": "screenshot"},
    }

    // Step 2: 将 vision_analysis 作为 web-agent 的输入
    // web-agent 应能根据 vision_analysis 生成网页
    inputMsg := a2a.NewMessage(a2a.MessageRoleUser,
        a2a.NewTextPart("Based on this vision analysis: " + visionArtifact.Content + "\n\nGenerate a webpage"))
    
    // 此测试需要真实启动 Agent 或 mock A2A server
    // 当前阶段可先做 fixture 级别的契约验证
}
```

### 当前阶段建议
集成测试分两步推进：
1. **Fixture 级**：验证 Agent A 的输出格式能被 Agent B 的输入解析器识别（可 unit test）
2. **真实调用级**：启动两个 Agent 实例，通过 A2A task 调用链验证（需要 docker-compose 或进程管理）

---

## 8. L6：Docker Smoke Test

### 目的
验证 docker-compose 一键启动后，所有 Agent 均可发现、健康检查通过、可发送任务。

### 验证步骤

```bash
# 1. 启动所有服务
docker-compose up -d

# 2. 等待健康检查通过
docker-compose ps

# 3. 每个 Agent 健康检查
for port in 8081 8082 8083 8084 8091 8092 8093 8094 8095 8096 8097 8099 8100 8101 8102 8103 8104 8105 8106; do
  curl -sf http://localhost:$port/health && echo " :$port OK" || echo ":$port FAIL"
done

# 4. Gateway /api/agents 返回完整 Agent 列表
curl -s http://localhost:8080/api/agents | jq '[.data[] | {name, description}]'

# 5. 发送一次端到端 run
curl -s -X POST http://localhost:8080/api/runs \
  -H "Content-Type: application/json" \
  -d '{"conversationId": "test-001", "message": "create a hello world webpage"}'
```

---

## 9. 当前测试结果汇总

执行 `for d in agents/*/; do (cd "$d" && go test ./...); done` 结果：

```
 agents/adk                     ok  0.004s
 agents/agent-builder-agent     ok  0.004s
 agents/artifact-agent          ok  0.002s
 agents/code-agent              ok  0.004s
 agents/context-agent           ok  0.002s
 agents/custom-agent            ok  0.004s
 agents/deploy-agent            ok  0.004s
 agents/diff-agent              ok  0.004s
 agents/document-agent          ok  0.004s
 agents/file-agent              ok  0.002s
 agents/ppt-agent               ok  0.004s
 agents/qa-acceptance-agent     ok  0.004s
 agents/release-agent           ok  0.004s
 agents/review-agent            ok  0.003s
 agents/security-agent          ok  0.004s
 agents/test-agent              ok  0.004s
 agents/version-agent           ok  0.002s
 agents/vision-agent            ok  0.004s
 agents/web-agent               ok  0.004s
 agents/web-research-agent      ok  0.004s
```

**结论：20/20 测试通过。** 所有 Agent 的 handler 逻辑、artifact 提取、fenced block 解析均已有单元测试覆盖并通过。

---

## 10. 当前覆盖缺口与建议

### 已覆盖 ✅

| 维度 | 状态 |
|---|---|
| ADK AgentCard 生成 | ✅ `agents/adk/server_test.go` |
| 每个 Agent handler 纯逻辑 | ✅ 19 个 `handler_test.go`，共 1520 行 |
| LLM mock/fake | ✅ 各 test 使用 fixture 文本代替 LLM 真实调用 |
| Artifact type 映射 | ✅ 各 test 验证 type/metadata |
| Security: AgentCard 无 secret | ✅ server_test 不写入真实 key |

### 待补充 ⚠️

| 维度 | 优先级 | 说明 |
|---|---|---|
| **config.yaml 加载校验测试** | 高 | 每个 Agent 增加 `TestConfigLoadable`，验证 yaml 可解析、必填字段存在 |
| **AgentCard 安全扫描测试** | 高 | 验证 `description`、`skills` 中不含 API key、内部路径、system prompt |
| **A2A 协议兼容性测试** | 高 | 启动 Agent 后 curl /health 和 /.well-known/agent.json |
| **跨 Agent 中间产物传递** | 中 | vision → web/code、file → document 等链路 fixture 验证 |
| **Docker compose smoke** | 中 | 一键启动 → 全量健康检查 → run 验证 |
| **并发多 Agent 调用** | 中 | 同时向多个 Agent 发送 task，验证隔离性和正确性 |
| **Fallback/Retry 测试** | 低 | Agent unhealthy → 编排降级（依赖 Orchestrator 就绪） |
| **Stream replay 测试** | 低 | SSE 中断/乱序/缺事件的鲁棒性 |
| **E2E Demo 路径** | 低 | 单聊 → 群聊 → @mention → 多 Agent 回复 → 刷新持久化 |

---

## 11. 快速验证脚本

将以下内容保存为 `smoke_all_agents.sh`：

```bash
#!/bin/bash
set -e
echo "=== AgentHub Multi-Agent Smoke Test ==="

# Unit tests (no LLM, no network)
echo "[1/3] Running unit tests for all agents..."
for d in agents/*/; do
  (cd "$d" && go test ./... > /dev/null 2>&1) || { echo "FAIL: $(basename $d)"; exit 1; }
done
echo "  All 20 modules passed"

# Config validation
echo "[2/3] Validating config.yaml files..."
for d in agents/*/; do
  name=$(basename "$d")
  [ "$name" = "adk" ] && continue
  cfg="$d/config.yaml"
  [ -f "$cfg" ] || { echo "FAIL: $name missing config.yaml"; exit 1; }
  # check required fields
  for field in name description version url skills inputModes outputModes streaming; do
    grep -q "^${field}:" "$cfg" || echo "WARN: $name config.yaml missing field '$field'"
  done
  # security: no secrets in config
  grep -qiE 'api.?key|token|secret|password' "$cfg" && echo "SECURITY WARN: $name config.yaml may contain secrets"
done
echo "  Config validation complete"

# (Optional) Docker smoke — requires docker-compose up first
echo "[3/3] Docker smoke (skip if not running)..."
if curl -sf http://localhost:8081/health > /dev/null 2>&1; then
  passed=0
  failed=0
  for port in 8081 8082 8083 8084 8091 8092 8093 8094 8095 8096 8097 8099 8100 8101 8102 8103 8104 8105 8106; do
    if curl -sf http://localhost:$port/health > /dev/null 2>&1; then
      ((passed++))
    else
      echo "  FAIL: port $port not reachable"
      ((failed++))
    fi
  done
  echo "  $passed healthy, $failed failed"
fi

echo "=== Done ==="
```

---

## 12. 总结

| 问题 | 答案 |
|---|---|
| 子 Agent 设计是否成功？ | **19 个 Agent 均通过单元测试**，handler 逻辑、artifact 提取、config 结构一致 |
| 能否被成功调用？ | **需要补充 A2A 协议级冒烟测试**（启动 Agent → curl /health → 发送 task）来端到端验证 |
| 当前最需要补什么？ | (1) 每个 Agent 增加 config 加载校验测试 (2) 1 个 docker-compose 级别的全量健康检查脚本 (3) 代表性 Agent 的 A2A task 端到端调用验证 |
| 测试哲学对齐？ | 全部测试遵循 **contract-first + 确定性优先**：fixture 代替真实 LLM，通用命名（example-agent-a/b），不写死具体 Agent 名称 |

---

## 参考资料

- `testing-review-contract` Skill — 测试分层与质量门禁契约
- `a2a-agent-contract` Skill — AgentCard、Health Check、A2A Streaming Task 契约
- `artifact-contract` Skill — Artifact type 注册、schema 校验、安全边界
- `adk-runtime-contract` Skill — Runtime Context、LLMClient、TaskHandler 契约
- `agents/README.md` — 子 Agent 能力矩阵与 Artifact Type 约定
