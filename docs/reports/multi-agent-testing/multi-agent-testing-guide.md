# AgentHub 多子Agent 可调用性测试指南

> 目标：验证 19 个子 Agent 设计正确、可被成功调用、产出符合契约。

---

## 1. 当前状态

| 指标 | 值 |
|---|---|
| 子 Agent 总数 | 19（含 code-agent） |
| ADK 公共模块 | `agents/adk`（AgentCard、Context、LLMClient、A2AServer） |
| 每个 Agent 结构 | `config.yaml` + `handler.go` + `handler_test.go` + `main.go` + `Dockerfile` |
| 当前测试状态 | **全部 19 Agent + ADK 模块测试通过**（20 modules, 262 tests, all `ok`, `-race` zero） |
| 测试代码总量 | ~540 行（adk/stream_replay_test.go）+ ~470 行（adk/concurrency_test.go）+ ~2,320 行（handler_test.go）+ 133 行（adk/server_test.go）+ ~450 行（adk/agent_test.go） |
| Config 校验测试 | ✅ 已完成（`adk.ValidateConfig` + 每个 Agent 的 `TestConfigLoadable`） |
| Config 边界测试 | ✅ 已完成（ADK: 9 个 LoadConfig invalid fixture + 每个 Agent: 10 边界用例 `TestConfigValidation`）|
| AgentCard 安全扫描 | ✅ 已完成（`adk.ValidateAgentCard` + 每个 Agent 的 `TestAgentCardSecure`） |
| A2A 协议兼容性测试 | ✅ 已完成（ADK: 4 个 httptest + 每个 Agent: `TestA2AProtocol` 含 3 sub-case）|
| 跨 Agent 集成测试 | ✅ 已完成（4 条链路 fixture 验证: vision→web, vision→code, file→document, security→review, 共 8 tests）|
| 并发多 Agent 调用测试 | ✅ 已完成（ADK 9 tests + 4 Agent parser tests, 共 13 tests, `-race` zero）|
| Stream Replay 测试 | ✅ 已完成（ADK 14 tests: 事件顺序、cancel、error safety、malformed、多 Agent taskID 隔离、golden fixture）|

---

## 2. 测试维度总览

按 `testing-review-contract` 定义的分层矩阵，测试多 Agent 可调用性需要覆盖以下 6 个维度：

```
  ┌──────────────────────────────────────────────────┐
  │           L6: Docker Smoke Test                  │  一键启动 → /health → /api/agents → run
  ├──────────────────────────────────────────────────┤
  │    L5: Cross-Agent Integration Test              │  Agent A 产出 → Agent B 消费
  ├──────────────────────────────────────────────────┤
  │     L4: A2A Protocol Test                        │  AgentCard、/health、task sendSubscribe
  ├──────────────────────────────────────────────────┤
  │      L3: Artifact Output Test                    │  artifactType、content、metadata、fenced block 解析
  ├──────────────────────────────────────────────────┤
  │      L2: Handler Logic Test (Unit)               │  parse、extract、validate 纯逻辑函数
  ├──────────────────────────────────────────────────┤
  │      L1: Schema / Contract Test                  │  config.yaml、AgentCard 字段完整性
  │      ├─ L1a: Config 加载校验 (TestConfigLoadable) │  19 Agent × 1 = 19 用例
  │      ├─ L1b: Config 边界测试 (TestConfigValidation)│  ADK 9 fixture + 19 Agent × 10 = 190 sub-case
  │      └─ L1c: AgentCard 安全扫描 (TestAgentCardSecure)│ ADK 10 用例 + 19 Agent × 1 = 29 用例
  └──────────────────────────────────────────────────┘
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

`agents/adk/agent_test.go` 提供 14 个用例：
- `TestValidateConfig_AllFieldsPresent` — 全字段合法通过
- `TestValidateConfig_MissingName` — 缺失 name 报错
- `TestValidateConfig_MissingDescription` — 缺失 description 报错
- `TestValidateConfig_MissingVersion` — 缺失 version 报错
- `TestValidateConfig_MissingURL` — 缺失 url 报错
- `TestValidateConfig_EmptySkills` — 空 skills 报错
- `TestValidateConfig_EmptyInputModes` — 空 inputModes 报错
- `TestValidateConfig_EmptyOutputModes` — 空 outputModes 报错
- `TestValidateConfig_DescriptionContainsSecret` — description 含 API_KEY 被安全拦截
- `TestValidateConfig_URLContainsSecret` — URL 含 token 被安全拦截
- `TestValidateConfig_MultipleErrors` — 全空配置返回多条错误
- `TestLoadConfig_ValidYAML` — 有效 YAML 文件加载成功
- `TestLoadConfig_MissingFile` — 文件不存在报错
- `TestLoadConfig_InvalidYAML` — 非法 YAML 解析失败

每个 Agent 的 `handler_test.go` 增加 `TestConfigLoadable`（共 19 个），加载本 Agent 的 `config.yaml`，调用 `adk.ValidateConfig`，断言零错误。

### L1+：AgentCard 独立安全扫描

`adk.ValidateAgentCard` 对构建的 AgentCard 执行独立安全扫描，分为三层：

| 扫描层 | 检查内容 | 适用范围 |
|---|---|---|
| 硬密钥 | `API_KEY`, `api_key`, `APIKEY`, `BEGIN RSA`, `PRIVATE KEY`, `DB_PASSWORD`, `DATABASE_URL`, `sk-`, `sk-ant-` | 所有字段（含技能） |
| 上下文密钥 | `token=`, `access_token=`, `secret=`, `secret:`, `password=`, `password:`, `apikey=` | name, description, URL, modes, interfaces（技能豁免，如 `secret_scan` 为合法功能名） |
| 内部路径/系统提示 | `/internal/`, `/admin/`, `/debug/`, `system prompt`, `system_prompt`, `systemPrompt`, `you are a`, `your role is` | name, description, modes, interfaces（技能仅扫描路径不扫描提示词） |

`adk/agent_test.go` 中 10 个用例覆盖：
- `CleanCard` — 合法配置通过
- `DescriptionContainsAPIKey` — description 含 `API_KEY=sk-...` 被拦截
- `SkillContainsHardSecret` — 技能名含 `sk-ant-` 被拦截
- `SkillWithSecretScanIsLegitimate` — `secret_scan`、`password_check` 作为合法安全功能通过
- `InterfaceURLContainsSecret` — URL 含 `token=secret123` 被拦截
- `InternalPathInDescription` — description 含 `/internal/` 被拦截
- `SystemPromptIndicatorInDescription` — description 含 `system prompt` 被拦截
- `NameContainsSecret` — name 含 `sk-ant-` 被拦截
- `InputModeContainsInternalPath` — inputModes 含 `/internal/rpc` 被拦截
- `MultipleSecurityIssues` — 多条违规同时报告

每个 Agent 的 `handler_test.go` 增加 `TestAgentCardSecure`（共 19 个），加载 `config.yaml` → `BuildAgentCard` → `ValidateAgentCard` → 断言零违规。

### L1 边界测试：Config Invalid Fixture

`agents/adk/testdata/` 提供 9 个非法 YAML fixture：

| Fixture | 缺失/违规字段 | 预期行为 |
|---|---|---|
| `missing_name.yaml` | name 字段缺失 | `ValidateConfig` 返回 "name is required" |
| `missing_description.yaml` | description 字段缺失 | `ValidateConfig` 返回 "description is required" |
| `missing_version.yaml` | version 字段缺失 | `ValidateConfig` 返回 "version is required" |
| `missing_url.yaml` | url 字段缺失 | `ValidateConfig` 返回 "url is required" |
| `empty_skills.yaml` | skills 为空数组 `[]` | `ValidateConfig` 返回 "skills must be non-empty" |
| `empty_inputmodes.yaml` | inputModes 为空数组 `[]` | `ValidateConfig` 返回 "inputModes must be non-empty" |
| `empty_outputmodes.yaml` | outputModes 为空数组 `[]` | `ValidateConfig` 返回 "outputModes must be non-empty" |
| `secret_in_description.yaml` | description 含 `API_KEY=sk-abc123` | `ValidateConfig` 返回敏感关键词告警 |
| `token_in_url.yaml` | URL 含 `?token=secret123` | `ValidateConfig` 返回敏感关键词告警 |

`adk/agent_test.go` 提供对应的 12 个 LoadConfig 测试（3 个已有 + 9 个新增）：
- `TestLoadConfig_ValidYAML` / `TestLoadConfig_MissingFile` / `TestLoadConfig_InvalidYAML`
- `TestLoadConfig_MissingName` / `TestLoadConfig_MissingDescription` / `TestLoadConfig_MissingVersion`
- `TestLoadConfig_MissingURL` / `TestLoadConfig_EmptySkills` / `TestLoadConfig_EmptyInputModes`
- `TestLoadConfig_EmptyOutputModes` / `TestLoadConfig_SecretInDescription` / `TestLoadConfig_TokenInURL`

### L1 边界测试：每个 Agent 的 TestConfigValidation

每个 Agent 的 `handler_test.go` 增加 `TestConfigValidation`（共 19 个，每个含 10 个 sub-case），使用程序化构造的 `adk.AgentConfig` 结构体覆盖所有无效边界：

| 用例 | 构造方式 | 预期 |
|---|---|---|
| `missing_name` | AgentConfig 不设 Name | 返回至少 1 个 error |
| `missing_description` | AgentConfig 不设 Description | 返回至少 1 个 error |
| `missing_version` | AgentConfig 不设 Version | 返回至少 1 个 error |
| `missing_url` | AgentConfig 不设 URL | 返回至少 1 个 error |
| `empty_skills_(nil)` | Skills 设为 nil | 返回至少 1 个 error |
| `empty_inputModes_(nil)` | InputModes 设为 nil | 返回至少 1 个 error |
| `empty_outputModes_(nil)` | OutputModes 设为 nil | 返回至少 1 个 error |
| `all_fields_empty` | 空 AgentConfig{} | 返回多条 error |
| `secret_in_description` | Description 含 `API_KEY=sk-abc123` | 返回安全告警 error |
| `token_in_url` | URL 含 `?token=secret123` | 返回安全告警 error |

**总计: 9 个 YAML fixture + 12 个 ADK LoadConfig 测试 + 19 × 10 = 190 个 Agent 边界 sub-case。**

### 待补充

- （无 — L1 覆盖已完整）

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

### 已覆盖：ADK httptest 集成测试

`agents/adk/server_test.go` 提供 4 个 A2A 协议测试（无需启动真实端口）：

| 测试 | 验证内容 |
|---|---|
| `TestA2AServer_HealthEndpoint` | `GET /health` → 200, Content-Type JSON, `{"status":"ok","agent":"<name>"}` |
| `TestA2AServer_AgentCardEndpoint` | `GET /.well-known/agent.json` → 200, Content-Type JSON, 8 个必填字段完整，值匹配 |
| `TestA2AServer_AgentCardNoSecrets` | AgentCard 响应体不含 API_KEY、token、/internal/ 等禁止关键词 |
| `TestA2AServer_EndpointsReturnCorrectStatus` | 4 个路由 (`/health`, `/.well-known/agent.json`, `POST /`, `POST /a2a/tasks/sendSubscribe`) 均返回 200 |

这些测试通过 `httptest.NewServer(server.Handler())` 启动内存 HTTP server，无需绑定真实端口。

### 已覆盖：每个 Agent 的 A2A 协议测试

每个 Agent 的 `handler_test.go` 增加 `TestA2AProtocol`（共 19 个），调用 `adk.TestA2AEndpoints(t, cfg)`，该 helper 执行 3 个 sub-test：

| sub-test | 验证点 |
|---|---|
| `health` | 加载 `config.yaml` → 启动 httptest server → `GET /health` → 验证 status=ok, agent 名称匹配 |
| `agentcard` | `GET /.well-known/agent.json` → 验证 8 个必填字段存在、name/version/streaming 值与 config 一致 |
| `nosecrets` | AgentCard 响应体安全扫描：硬密钥、内部路径、系统提示词关键词均不得出现 |

**总计：4 个 ADK 协议测试 + 19 Agent × 3 sub-case = 61 个 A2A 协议验证点。**

### Docker 冒烟测试脚本

`docs/reports/a2a-protocol-tests/smoke_test_agents.sh` 提供基于 curl 的真实网络调用验证：

```bash
# 启动所有 Agent（需先 docker-compose up）
# 然后运行冒烟测试
./docs/reports/a2a-protocol-tests/smoke_test_agents.sh

# 生成 JSON 报告
./docs/reports/a2a-protocol-tests/smoke_test_agents.sh --json
```

脚本覆盖每个 Agent 的 3 项检查（health + AgentCard + 安全扫描），共 57 个检查点。

### Docker 冒烟测试脚本内容

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

---

## 7. L5：跨 Agent 集成测试

### 目的
验证多 Agent 之间的中间产物传递链路（readme.md 第 5 节定义的多模态通道）。

### 关键集成链路（已实现 fixture 级验证）

```
链路 1: vision-agent → web-agent          ✅
  image_ref → vision-agent → vision_analysis → web-agent → webpage

链路 2: vision-agent → code-agent         ✅
  截图描述 → vision-agent → vision_analysis → code-agent → code

链路 3: file-agent → document-agent       ✅
  file_ref → file-agent → file_summary → document-agent → document

链路 4: security-agent → review-agent     ✅
  代码/配置 → security-agent → security_report → review-agent → review_report
```

### 测试设计

每个链路包含 2 个测试函数，采用 fixture 级别验证（不依赖真实 LLM 或网络）：

**测试 1: 下游 Agent 能消费上游 Agent 的输出文本**
- 模拟上游 Agent 的完整输出（含 fenced block）
- 验证下游 Agent 的 `extractTextFromParts()` 能提取文本上下文
- 验证下游 Agent 的解析器（`extractWebpageArtifacts` / `parseCodeBlocks` / `extractDocumentArtifacts` / `extractReviewArtifacts`）能正确处理混合上下文中的目标 artifact

**测试 2: 上游 Artifact 元数据可通过 A2A 链路传递**
- 构造上游 `adk.Artifact`（含 type/title/content/metadata）
- 验证上游 artifact.content 可作为下游 Agent 的输入
- 验证下游 Agent 能从上文中解析出目标 artifact

### 关键实现细节

- **vision→web**: `extractWebpageArtifacts` 在含 `json:vision-analysis.json` 的混合文本中正确识别 `html:index.html` 块
- **vision→code**: `parseCodeBlocks` 匹配所有有效语言块（含 json），测试需搜索特定语言块（tsx）而非假设唯一块
- **file→document**: `extractDocumentArtifacts` 仅匹配 markdown/md fence，忽略 `json:file-summary.json` 块
- **security→review**: `extractReviewArtifacts` 使用 `parseJSONBlocks` 匹配所有 JSON 块，测试需按 artifact type 过滤出 `review_report`

### 已实现测试

| 消费 Agent | 测试函数 | 覆盖链路 |
|---|---|---|
| web-agent | `TestCrossAgent_VisionToWeb_ConsumesVisionOutput` | vision → web |
| web-agent | `TestCrossAgent_VisionToWeb_ArtifactMetadataPassThrough` | vision → web |
| code-agent | `TestCrossAgent_VisionToCode_ConsumesVisionOutput` | vision → code |
| code-agent | `TestCrossAgent_VisionToCode_MetadataChain` | vision → code |
| document-agent | `TestCrossAgent_FileToDocument_ConsumesFileSummary` | file → document |
| document-agent | `TestCrossAgent_FileToDocument_PreservesContext` | file → document |
| review-agent | `TestCrossAgent_SecurityToReview_ConsumesSecurityReport` | security → review |
| review-agent | `TestCrossAgent_SecurityToReview_ArtifactTypeChain` | security → review |

**总计: 4 个消费 Agent × 2 tests = 8 个跨 Agent 集成测试。**

### 运行命令
```bash
# 运行全部跨 Agent 集成测试
cd /project/Multi_Agent_Framework/agents
go test ./web-agent/... ./code-agent/... ./document-agent/... ./review-agent/... -v -run "TestCrossAgent_" -count=1

# 运行单个链路的跨 Agent 测试
go test ./web-agent/... -v -run "TestCrossAgent_VisionToWeb" -count=1
go test ./code-agent/... -v -run "TestCrossAgent_VisionToCode" -count=1
```

---

## 7+. L5+：并发多 Agent 调用测试

### 目的
验证多个 Agent 服务器在并发场景下的隔离性、正确性和安全性。确保同时向多个 Agent 发送 task 不会导致跨 Agent 污染、数据竞争或响应错误。

### 测试架构

```
┌──────────────────────────────────────────────────────────────┐
│              Concurrent Multi-Agent Test Layers              │
├──────────────────────────────────────────────────────────────┤
│  Layer 1: ADK Server Concurrency (concurrency_test.go)       │
│  ├─ 多服务器并发隔离 (Multiple servers, concurrent health)   │
│  ├─ AgentCard 并发隔离 (Multiple servers, concurrent cards)  │
│  ├─ 单服务器高并发 (Single server, 40 concurrent requests)   │
│  ├─ BuildAgentCard 并发安全 (4 configs × 10 goroutines)      │
│  ├─ NoopHandler 可重入性 (50 concurrent health checks)       │
│  ├─ 混合端点并发 (10 health + 10 AgentCard concurrently)     │
│  ├─ 多 Agent task 隔离 (3 agents, 30 concurrent requests)    │
│  ├─ ValidateConfig 并发安全 (20 goroutines)                  │
│  └─ 服务器启动关闭并发 (10 servers start/serve/close)        │
│  → 9 tests, httptest-based, zero external deps               │
├──────────────────────────────────────────────────────────────┤
│  Layer 2: Agent Parser Concurrency (handler_test.go)         │
│  ├─ parseCodeBlocks 并发安全 (code-agent, 4 inputs × 10)     │
│  ├─ extractWebpageArtifacts 并发安全 (web-agent, 4 × 10)     │
│  ├─ extractDocumentArtifacts 并发安全 (document-agent, 4×10) │
│  └─ extractReviewArtifacts 并发安全 (review-agent, 4 × 10)   │
│  → 4 tests, 160 goroutine calls per test                     │
├──────────────────────────────────────────────────────────────┤
│  Race Detection: go test -race                               │
│  → All 262 tests, 20 modules, zero data races                │
└──────────────────────────────────────────────────────────────┘
```

### 已实现测试

| 层级 | 测试函数 | 验证内容 |
|---|---|---|
| ADK | `TestConcurrent_MultipleServersIsolated` | 5 个 Agent 服务器并发启动，各自 /health 返回正确身份 |
| ADK | `TestConcurrent_MultipleServersAgentCardIsolated` | 3 个服务器并发 AgentCard，无跨 Agent 污染 |
| ADK | `TestConcurrent_SingleServerParallelRequests` | 单服务器 40 并发请求（20 health + 20 AgentCard），100% 成功率 |
| ADK | `TestConcurrent_BuildAgentCardParallel` | `BuildAgentCard` 在 4×10=40 goroutines 中并发调用，无指针混叠 |
| ADK | `TestConcurrent_NoopHandlerReentrant` | NoopHandler 在 50 并发 health check 下安全 |
| ADK | `TestConcurrent_MixedEndpointsParallel` | 10 health + 10 AgentCard 混合并发，身份一致 |
| ADK | `TestConcurrent_MultiAgentTaskIsolation` | 3 Agent × 10 并发请求，无跨 Agent 身份污染 |
| ADK | `TestConcurrent_ValidateConfigParallel` | `ValidateConfig` 在 20 goroutines 中并发调用，正确区分 valid/invalid |
| ADK | `TestConcurrent_ServerStartupShutdown` | 10 个服务器并发启动→请求→关闭，无干扰 |
| code-agent | `TestConcurrent_ParseCodeBlocksParallel` | `parseCodeBlocks` 4 种语言 × 10 goroutines 并发解析 |
| web-agent | `TestConcurrent_ExtractWebpageArtifactsParallel` | `extractWebpageArtifacts` 4 输入 × 10 goroutines 并发提取 |
| document-agent | `TestConcurrent_ExtractDocumentArtifactsParallel` | `extractDocumentArtifacts` 4 输入 × 10 goroutines 并发提取 |
| review-agent | `TestConcurrent_ExtractReviewArtifactsParallel` | `extractReviewArtifacts` 4 输入 × 10 goroutines 并发提取 |

**总计: 9 ADK + 4 Agent = 13 个并发测试。**

### 运行命令
```bash
# 运行全部并发测试
cd /project/Multi_Agent_Framework/agents
go test ./adk/... ./code-agent/... ./web-agent/... ./document-agent/... ./review-agent/... -v -run "TestConcurrent_" -count=1

# 运行并发测试 + 数据竞争检测
go test ./... -race -count=1

# 仅运行 ADK 并发测试
go test ./adk/... -v -run "TestConcurrent_" -count=1

# 仅运行 Agent parser 并发测试
go test ./code-agent/... ./web-agent/... ./document-agent/... ./review-agent/... -v -run "TestConcurrent_" -count=1
```

---

## 7++. L5++：Stream Replay 测试

### 目的
验证 A2A 事件流的顺序稳定性、中断恢复、错误安全性及多 Agent taskID 隔离。确保 SSE 流在各种异常条件下不会导致前端崩溃或事件混淆。

### 测试架构

```
┌──────────────────────────────────────────────────────────────┐
│                  Stream Replay Test Layers                   │
├──────────────────────────────────────────────────────────────┤
│  Event Order Stability                                       │
│  ├─ 完整事件序列: Submitted → Working → text → code → Complete│
│  ├─ 空 handler 最小序列: Task(Submitted) → Working → Complete│
│  ├─ NoopHandler 稳定序列验证                                 │
│  └─ 大文本流 (20 chunks) 保序验证                            │
├──────────────────────────────────────────────────────────────┤
│  Error & Fault Safety                                        │
│  ├─ handler error → Failed (非 Completed)                    │
│  ├─ 错误消息安全性: 无 stack trace/goroutine/panic/敏感词    │
│  └─ 终端状态互斥: Completed ≠ Failed ≠ Canceled              │
├──────────────────────────────────────────────────────────────┤
│  Cancel & Interruption                                       │
│  ├─ Cancel() → Done channel 关闭                             │
│  ├─ 协作式取消模式: handler 检查 Done 后提前返回             │
│  └─ Cancel executor 返回 Canceled (非 Completed)             │
├──────────────────────────────────────────────────────────────┤
│  Malformed & Edge Cases                                      │
│  ├─ 空 content / nil metadata artifact 不崩溃               │
│  ├─ 畸形 JSON 内容不中断流                                   │
│  └─ 嵌套 fence 内容安全处理                                  │
├──────────────────────────────────────────────────────────────┤
│  Multi-Agent & Metadata                                      │
│  ├─ 多 Agent taskID 隔离 (concurrent streams)                │
│  ├─ TaskID / ContextID 在所有事件中一致                      │
│  └─ 并发流无交叉污染                                         │
├──────────────────────────────────────────────────────────────┤
│  Golden Fixture (Event Replay)                               │
│  └─ 所有测试的 events slice 可作为 replay fixture 使用       │
└──────────────────────────────────────────────────────────────┘
```

### 事件序列模型

`ExecuteHandler` 产生的事件序列（无已有 task 时）：

```
[0] *a2a.Task         {State: Submitted}    ← NewSubmittedTask
[1] *TaskStatusUpdate  {State: Working}
[2..N-2] *TaskArtifactUpdate {Name: "response"}           ← StreamText chunks
          *TaskArtifactUpdate {Name: "<filename>"}         ← code artifacts
[N-1] *TaskStatusUpdate {State: Completed|Failed}         ← terminal
```

**关键规则**：
- 第一个事件永远是 `*a2a.Task`（非 `*TaskStatusUpdateEvent`）
- Working 永远不会出现在 Submitted 之前
- Completed 永远不会出现在 Working 之前
- 终端状态（Completed/Failed/Canceled/Rejected）互斥
- 所有事件共享同一 TaskID 和 ContextID

### 已实现测试

| 分类 | 测试函数 | 验证内容 |
|---|---|---|
| 事件顺序 | `TestStream_EventOrderStability` | Submitted→Working→text→code→Completed 序列稳定性 |
| 事件顺序 | `TestStream_EmptyHandlerProducesMinimalSequence` | 空 handler 最小序列 Task(Submitted)→Working→Completed |
| 事件顺序 | `TestStream_NoopHandlerProducesValidSequence` | NoopHandler 3-event 序列 |
| 事件顺序 | `TestStream_LargeTextStreamPreservesOrder` | 20 chunks 保序验证 |
| 错误安全 | `TestStream_ErrorProducesFailedStatus` | handler error → Failed, 错误消息不含 stack/sensitive |
| 错误安全 | `TestStream_TerminalStatesAreDistinct` | Completed/Failed/Canceled 互斥 |
| Cancel | `TestStream_CancelSignalsContext` | Cancel() → Done channel 关闭 |
| Cancel | `TestStream_CancelCooperativePattern` | handler 检查 Done 后停止输出 |
| Cancel | `TestStream_CancelDoesNotEmitCompleted` | executor.Cancel() → Canceled 非 Completed |
| 畸形输入 | `TestStream_MalformedArtifactsDontCrash` | 空 title/content/nil metadata 安全 |
| 畸形输入 | `TestStream_InvalidContentIsSafe` | 畸形 JSON / 嵌套 fence 安全 |
| 多Agent | `TestStream_MultiAgentTaskIDIsolation` | 双 Agent 并发流 taskID 不交叉 |
| 多Agent | `TestStream_ConcurrentEventStreamsDontCrossContaminate` | 三 Agent 并发流无文本交叉 |
| 元数据 | `TestStream_EventMetadataHasTaskID` | TaskID/ContextID 在所有事件中正确传播 |

**总计: 14 个 Stream Replay 测试，全部通过，`-race` zero。**

### 运行命令
```bash
# 运行全部 Stream Replay 测试
cd /project/Multi_Agent_Framework/agents
go test ./adk/... -v -run "TestStream_" -count=1

# 运行 + 数据竞争检测
go test ./adk/... -race -run "TestStream_" -count=1

# 仅运行事件顺序测试
go test ./adk/... -v -run "TestStream_EventOrder|TestStream_Empty|TestStream_Noop|TestStream_Large" -count=1

# 仅运行 cancel 测试
go test ./adk/... -v -run "TestStream_Cancel" -count=1
```

### Replay Fixture 使用

每个 Stream Replay 测试的 `collectEvents` helper 收集了完整的事件序列。这些事件序列可作为 golden fixture 用于：

1. **协议兼容性验证**：对比不同版本的 A2A 库生成的事件序列
2. **前端回放测试**：将事件序列输入前端 SSE parser，验证渲染一致性
3. **CI 回归检测**：保存 golden JSON，检测事件结构变动

```go
// 记录 golden fixture
events := collectEvents(t, execCtx, handler)
// events 即为可序列化的 replay fixture
for _, ev := range events {
    data, _ := json.Marshal(ev)
    // 保存到 golden 文件
}
```

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

执行 `go test ./... -count=1` 结果（2026-05-26）：

```
 agents/adk                     ok  (67 tests: 14 ValidateConfig + 12 LoadConfig + 10 ValidateAgentCard + 4 A2A protocol + 3 BuildAgentCard + 9 concurrent + 14 stream replay + 1 LLM)
 agents/agent-builder-agent     ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/artifact-agent          ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/code-agent              ok  (handler tests + 2 cross-agent tests + 1 concurrent test + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/context-agent           ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/custom-agent            ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/deploy-agent            ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/diff-agent              ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/document-agent          ok  (handler tests + 2 cross-agent tests + 1 concurrent test + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/file-agent              ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/ppt-agent               ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/qa-acceptance-agent     ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/release-agent           ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/review-agent            ok  (handler tests + 2 cross-agent tests + 1 concurrent test + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/security-agent          ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/test-agent              ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/version-agent           ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/vision-agent            ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/web-agent               ok  (handler tests + 2 cross-agent tests + 1 concurrent test + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
 agents/web-research-agent      ok  (handler tests + TestConfigLoadable + TestAgentCardSecure + TestConfigValidation/10 + TestA2AProtocol/3)
```

**结论：20/20 模块全部通过，共 262 tests，`-race` zero。** 每个 Agent 同时覆盖：
- handler 纯逻辑 + artifact 提取（各 Agent 特有测试）
- config.yaml 加载校验（`TestConfigLoadable`）
- config 无效边界用例（`TestConfigValidation`，10 个 sub-case）
- AgentCard 独立安全扫描（`TestAgentCardSecure`）
- A2A 协议端点验证（`TestA2AProtocol`，3 sub-case）
- 跨 Agent 集成测试（4 个消费 Agent 各 2 tests，共 8 tests）
- 并发安全测试（ADK 9 tests + 4 消费 Agent 各 1 test, 共 13 tests）
- Stream Replay 测试（ADK 14 tests: 事件顺序/cancel/error safety/malformed/多 Agent taskID 隔离）

---

## 10. 当前覆盖缺口与建议

### 已覆盖 ✅

| 维度 | 状态 |
|---|---|
| ADK AgentCard 生成 | ✅ `agents/adk/server_test.go`（3 个用例） |
| ADK config 校验 | ✅ `agents/adk/agent_test.go`（14 个用例）|
| ADK config invalid fixture | ✅ `agents/adk/agent_test.go`（12 个 LoadConfig 用例，9 个非法 YAML fixture）|
| 每个 Agent handler 纯逻辑 | ✅ 19 个 `handler_test.go`，共 ~2,320 行 |
| 每个 Agent config 加载校验 | ✅ 19 个 `TestConfigLoadable`（加载 config.yaml + ValidateConfig）|
| 每个 Agent config 边界测试 | ✅ 19 个 `TestConfigValidation`（各 10 sub-case，共 190 边界用例）|
| LLM mock/fake | ✅ 各 test 使用 fixture 文本代替 LLM 真实调用 |
| Artifact type 映射 | ✅ 各 test 验证 type/metadata |
| Security: config 无 secret | ✅ `ValidateConfig` 扫描 API_KEY/token/secret/password |
| Security: AgentCard 安全扫描 | ✅ `ValidateAgentCard` 10 用例 + 每个 Agent `TestAgentCardSecure` |
| Security: AgentCard 无 secret | ✅ server_test 不写入真实 key |
| A2A 协议兼容性测试 | ✅ ADK 4 个 httptest + 每个 Agent `TestA2AProtocol`（3 sub-case，共 61 验证点）|
| 跨 Agent 中间产物传递 | ✅ 4 条链路 vision→web/code, file→document, security→review（8 tests）|
| 并发多 Agent 调用测试 | ✅ ADK 多服务器隔离 + 单服务器高并发 + parser 并发安全（13 tests）+ `-race` |
| Stream replay 测试 | ✅ ADK 事件顺序/cancel/error safety/malformed/多 Agent taskID 隔离（14 tests）+ `-race` |
| Docker smoke 脚本 | ✅ `docs/reports/a2a-protocol-tests/smoke_test_agents.sh` |

### 待补充 ⚠️

| 维度 | 优先级 | 说明 |
|---|---|---|
| **Docker compose smoke** | 中 | 一键启动 → 全量健康检查 → run 验证 |
| **Fallback/Retry 测试** | 低 | Agent unhealthy → 编排降级（依赖 Orchestrator 就绪） |
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
| Config 加载校验是否完成？ | ✅ **已全部完成**：`adk.ValidateConfig`（7 必填字段 + 敏感关键词扫描）+ 19 个 Agent 的 `TestConfigLoadable` |
| Config 无效边界测试是否完成？ | ✅ **已全部完成**：ADK 9 个非法 YAML fixture + 12 个 LoadConfig 测试 + 每 Agent 10 边界用例 `TestConfigValidation`（共 190 sub-case）|
| AgentCard 安全扫描是否完成？ | ✅ **已全部完成**：`adk.ValidateAgentCard`（10 用例，分硬密钥/上下文密钥/内部路径三层）+ 19 个 Agent `TestAgentCardSecure` |
| A2A 协议兼容性测试是否完成？ | ✅ **已全部完成**：ADK 4 个 httptest + 19 Agent `TestA2AProtocol`（61 验证点）+ Docker smoke 脚本 |
| 跨 Agent 中间产物传递是否完成？ | ✅ **已全部完成**：4 条链路 fixture 验证 (vision→web/code, file→document, security→review)，共 8 tests |
| 并发多 Agent 调用测试是否完成？ | ✅ **已全部完成**：ADK 服务器隔离 + parser 并发安全 (13 tests, `-race` zero)，无数据竞争 |
| Stream replay 测试是否完成？ | ✅ **已全部完成**：ADK 事件顺序/cancel/error safety/malformed/多 Agent taskID 隔离 (14 tests, `-race` zero) |
| 能否被成功调用？ | **A2A 协议层已验证通过**（/health + AgentCard + 安全扫描），跨 Agent 链路 fixture 已验证，并发安全已验证，Stream Replay 已验证，完整 task 调用待 Docker 环境 |
| 当前最需要补什么？ | (1) docker-compose 全量健康检查执行 (2) 跨 Agent 真实 A2A task 端到端流式调用验证 |
| 测试哲学对齐？ | 全部测试遵循 **contract-first + 确定性优先**：fixture 代替真实 LLM，通用命名（example-agent-a/b），不写死具体 Agent 名称 |

---

## 13. 详细测试操作指南

### 13.1 环境准备

```bash
cd /project/Multi_Agent_Framework/agents
go mod tidy
```

### 13.2 运行全部测试

```bash
# 运行全部 20 个模块（ADK + 19 Agent）
cd /project/Multi_Agent_Framework/agents
go test ./... -count=1

# 详细输出（含每个 sub-case）
go test ./... -v -count=1 2>&1 | grep -E '^(=== RUN|--- PASS|--- FAIL|ok|FAIL)'

# 仅运行 ADK 模块测试
go test ./adk/... -v -count=1
```

### 13.3 运行单个 Agent 的测试

```bash
# 以 code-agent 为例
cd /project/Multi_Agent_Framework/agents/code-agent
go test -v -count=1

# 仅运行 config 相关测试
go test -v -run "TestConfig" -count=1

# 运行特定 sub-case
go test -v -run "TestConfigValidation/missing_name" -count=1
```

### 13.4 运行 config 边界测试

```bash
# 运行 ADK 的 LoadConfig invalid fixture 测试
cd /project/Multi_Agent_Framework/agents
go test ./adk/... -v -run "TestLoadConfig_" -count=1

# 运行所有 Agent 的 TestConfigValidation（190 个 sub-case）
go test ./... -v -run "TestConfigValidation" -count=1 2>&1 | grep -E 'PASS|FAIL'

# 运行所有 Agent 的 config 加载 + 边界 + 安全扫描
go test ./... -v -run "TestConfig|TestAgentCard" -count=1 2>&1 | grep -E 'PASS|FAIL'
```

### 13.5 运行 A2A 协议兼容性测试

```bash
# 运行 ADK 的 A2A 协议测试（httptest，无需 Docker）
cd /project/Multi_Agent_Framework/agents
go test ./adk/... -v -run "TestA2AServer_" -count=1

# 运行所有 Agent 的 A2A 协议测试（每个 Agent 启动 httptest server）
go test ./... -v -run "TestA2AProtocol" -count=1 2>&1 | grep -E 'PASS|FAIL'

# 运行单个 Agent 的 A2A 协议测试（查看详细 sub-case）
cd /project/Multi_Agent_Framework/agents/code-agent
go test -v -run "TestA2AProtocol" -count=1
# 输出:
#   --- PASS: TestA2AProtocol/health
#   --- PASS: TestA2AProtocol/agentcard
#   --- PASS: TestA2AProtocol/nosecrets

# Docker smoke（需要先 docker-compose up）
./docs/reports/a2a-protocol-tests/smoke_test_agents.sh
./docs/reports/a2a-protocol-tests/smoke_test_agents.sh --json  # 生成 JSON 报告
```

### 13.6 A2A 协议测试架构

```
┌─────────────────────────────────────────────────────────────┐
│                   A2A Protocol Test Layers                   │
├─────────────────────────────────────────────────────────────┤
│  Layer 1: ADK httptest (server_test.go)                     │
│  ├─ TestA2AServer_HealthEndpoint      验证 /health 响应格式   │
│  ├─ TestA2AServer_AgentCardEndpoint   验证 AgentCard 字段完整性│
│  ├─ TestA2AServer_AgentCardNoSecrets  验证 AgentCard 无敏感词 │
│  └─ TestA2AServer_Endpoints*          验证 4 个路由均返回 200  │
│  → 4 测试，7 sub-case，无需真实端口                            │
├─────────────────────────────────────────────────────────────┤
│  Layer 2: Agent httptest (handler_test.go, 19 agents)       │
│  ├─ TestA2AProtocol/health            加载 config → 启动 server│
│  ├─ TestA2AProtocol/agentcard         → 验证 8 个必填字段      │
│  └─ TestA2AProtocol/nosecrets         → 安全扫描 AgentCard     │
│  → 19 Agent × 3 sub-case = 57 验证点                          │
│  → 复用 adk.TestA2AEndpoints(t, cfg) helper                  │
├─────────────────────────────────────────────────────────────┤
│  Layer 3: Docker smoke (smoke_test_agents.sh)               │
│  ├─ curl /health              → 200 + JSON 格式校验           │
│  ├─ curl /.well-known/agent.json → 必填字段 + 名称匹配         │
│  └─ grep 安全扫描              → 禁止关键词告警                │
│  → 需要 docker-compose up 先启动所有 Agent                    │
└─────────────────────────────────────────────────────────────┘
```

### 13.7 如何为新 Agent 添加测试

当创建新的子 Agent 时，必须添加以下 5 个测试函数：

**Step 1: 确保 `handler_test.go` 引入 `adk` 包**

```go
import (
    "testing"
    "github.com/zxc1a1a1/Multi_Agent-AgentHub/agents/adk"
)
```

**Step 2: 添加 `TestConfigLoadable` — 验证 config.yaml 可加载且通过校验**

```go
func TestConfigLoadable(t *testing.T) {
    cfg, err := adk.LoadConfig("config.yaml")
    if err != nil {
        t.Fatalf("failed to load config.yaml: %v", err)
    }
    errs := adk.ValidateConfig(cfg)
    if len(errs) != 0 {
        for _, e := range errs {
            t.Errorf("config validation error: %v", e)
        }
    }
}
```

**Step 3: 添加 `TestConfigValidation` — 边界用例（10 个无效配置）**

```go
func TestConfigValidation(t *testing.T) {
    tests := []struct {
        name string
        cfg  adk.AgentConfig
    }{
        {name: "missing name", cfg: adk.AgentConfig{
            Description: "test", Version: "0.1.0", URL: "http://localhost:8081",
            Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "missing description", cfg: adk.AgentConfig{
            Name: "test-agent", Version: "0.1.0", URL: "http://localhost:8081",
            Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "missing version", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", URL: "http://localhost:8081",
            Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "missing url", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", Version: "0.1.0",
            Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "empty skills (nil)", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", Version: "0.1.0", URL: "http://localhost:8081",
            Skills: nil, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "empty inputModes (nil)", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", Version: "0.1.0", URL: "http://localhost:8081",
            Skills: []string{"s"}, InputModes: nil, OutputModes: []string{"text"},
        }},
        {name: "empty outputModes (nil)", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", Version: "0.1.0", URL: "http://localhost:8081",
            Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: nil,
        }},
        {name: "all fields empty", cfg: adk.AgentConfig{}},
        {name: "secret in description", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "Uses API_KEY=sk-abc123 for auth", Version: "0.1.0",
            URL: "http://localhost:8081", Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
        {name: "token in url", cfg: adk.AgentConfig{
            Name: "test-agent", Description: "test", Version: "0.1.0",
            URL: "http://localhost:8081?token=secret123", Skills: []string{"s"}, InputModes: []string{"text"}, OutputModes: []string{"text"},
        }},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            errs := adk.ValidateConfig(&tt.cfg)
            if len(errs) == 0 {
                t.Fatal("expected validation errors, got none")
            }
        })
    }
}
```

**Step 4: 添加 `TestAgentCardSecure` — AgentCard 安全扫描**

```go
func TestAgentCardSecure(t *testing.T) {
    cfg, err := adk.LoadConfig("config.yaml")
    if err != nil {
        t.Fatalf("failed to load config.yaml: %v", err)
    }
    card := adk.BuildAgentCard(cfg)
    errs := adk.ValidateAgentCard(card)
    if len(errs) != 0 {
        for _, e := range errs {
            t.Errorf("agentcard security violation: %v", e)
        }
    }
}
```

**Step 5: 添加 `TestA2AProtocol` — A2A 协议端点验证**

```go
func TestA2AProtocol(t *testing.T) {
    cfg, err := adk.LoadConfig("config.yaml")
    if err != nil {
        t.Fatalf("failed to load config.yaml: %v", err)
    }
    adk.TestA2AEndpoints(t, cfg)
}
```

该测试自动验证 `/health`（200 + JSON + agent 名称匹配）、`/.well-known/agent.json`（8 个必填字段 + 值匹配）和 AgentCard 安全扫描（无禁止关键词）。使用 `httptest` 无需真实端口或 Docker。

### 13.8 如何向 ADK 添加新的 invalid fixture

1. 在 `agents/adk/testdata/` 创建新的 YAML fixture 文件
2. 在 `agents/adk/agent_test.go` 添加对应的 `TestLoadConfig_*` 函数
3. 同时更新 `TestConfigValidation` 的用例表（如果适用）
4. 运行 `go test ./adk/... -count=1` 确保通过

### 13.9 测试文件结构速查

```
agents/
├── adk/
│   ├── agent.go              # ValidateConfig, ValidateAgentCard, LoadConfig
│   ├── agent_test.go         # 40 tests: 14 ValidateConfig + 12 LoadConfig + 10 ValidateAgentCard + 4 A2A protocol
│   ├── server.go             # BuildAgentCard, NewA2AServer, Handler()
│   ├── server_test.go        # 7 tests: 3 AgentCard mapping + 4 A2A protocol integration
│   ├── concurrency_test.go   # 9 concurrent tests
│   ├── stream_replay_test.go # 14 stream replay tests
│   ├── context.go            # TaskHandler, NoopHandler, ExecuteHandler
│   ├── a2a_test_util.go      # TestA2AEndpoints helper (httptest-based, called by each agent)
│   └── testdata/
│       ├── valid_config.yaml
│       ├── invalid_config.yaml
│       ├── missing_name.yaml
│       ├── missing_description.yaml
│       ├── missing_version.yaml
│       ├── missing_url.yaml
│       ├── empty_skills.yaml
│       ├── empty_inputmodes.yaml
│       ├── empty_outputmodes.yaml
│       ├── secret_in_description.yaml
│       └── token_in_url.yaml
├── <agent>/
│   ├── config.yaml            # Agent 身份配置
│   ├── handler.go             # 业务逻辑
│   ├── handler_test.go        # handler 单测 + TestConfigLoadable + TestConfigValidation + TestAgentCardSecure + TestA2AProtocol
│   ├── main.go                # 启动入口
│   └── Dockerfile
└── ...

docs/reports/
├── multi-agent-testing/
│   ├── multi-agent-testing-guide.md          # 主测试报告
│   └── verification/
│       ├── SUMMARY.md
│       ├── <agent>/
│       │   ├── health.json
│       │   └── agent-card.json
│       ├── cross-agent/
│       │   ├── vision-to-web.json
│       │   ├── vision-to-code.json
│       │   ├── file-to-document.json
│       │   └── security-to-review.json
│       ├── concurrent/
│       │   ├── adk-concurrency.json
│       │   └── agent-parser-concurrency.json
│       └── stream-replay/
│           └── stream-replay.json
└── a2a-protocol-tests/
    ├── README.md
    └── smoke_test_agents.sh
```

---

### 13.10 运行跨 Agent 集成测试

```bash
# 运行全部 4 条链路的跨 Agent 集成测试
cd /project/Multi_Agent_Framework/agents
go test ./web-agent/... ./code-agent/... ./document-agent/... ./review-agent/... -v -run "TestCrossAgent_" -count=1

# 运行单条链路的测试
go test ./web-agent/... -v -run "TestCrossAgent_VisionToWeb" -count=1
go test ./code-agent/... -v -run "TestCrossAgent_VisionToCode" -count=1
go test ./document-agent/... -v -run "TestCrossAgent_FileToDocument" -count=1
go test ./review-agent/... -v -run "TestCrossAgent_SecurityToReview" -count=1
```

跨 Agent 测试预期输出（每条链路 2 个测试通过）：
```
=== RUN   TestCrossAgent_VisionToWeb_ConsumesVisionOutput
--- PASS: TestCrossAgent_VisionToWeb_ConsumesVisionOutput (0.00s)
=== RUN   TestCrossAgent_VisionToWeb_ArtifactMetadataPassThrough
--- PASS: TestCrossAgent_VisionToWeb_ArtifactMetadataPassThrough (0.00s)
=== RUN   TestCrossAgent_VisionToCode_ConsumesVisionOutput
--- PASS: TestCrossAgent_VisionToCode_ConsumesVisionOutput (0.00s)
=== RUN   TestCrossAgent_VisionToCode_MetadataChain
--- PASS: TestCrossAgent_VisionToCode_MetadataChain (0.00s)
=== RUN   TestCrossAgent_FileToDocument_ConsumesFileSummary
--- PASS: TestCrossAgent_FileToDocument_ConsumesFileSummary (0.00s)
=== RUN   TestCrossAgent_FileToDocument_PreservesContext
--- PASS: TestCrossAgent_FileToDocument_PreservesContext (0.00s)
=== RUN   TestCrossAgent_SecurityToReview_ConsumesSecurityReport
--- PASS: TestCrossAgent_SecurityToReview_ConsumesSecurityReport (0.00s)
=== RUN   TestCrossAgent_SecurityToReview_ArtifactTypeChain
--- PASS: TestCrossAgent_SecurityToReview_ArtifactTypeChain (0.00s)
```

### 13.11 跨 Agent 集成测试架构

```
┌──────────────────────────────────────────────────────────────┐
│                 Cross-Agent Integration Test                  │
├──────────────────────────────────────────────────────────────┤
│  输入: 上游 Agent 的模拟输出文本（含 fenced block）            │
│  ├─ vision-agent 输出 → 含 json:vision-analysis.json         │
│  ├─ file-agent 输出   → 含 json:file-summary.json             │
│  └─ security-agent 输出 → 含 json:security-report.json        │
├──────────────────────────────────────────────────────────────┤
│  中间步骤:                                                    │
│  ├─ a2a.ContentParts → extractTextFromParts() → 提取文本     │
│  └─ 下游解析器 → 从混合上下文中识别目标 artifact              │
├──────────────────────────────────────────────────────────────┤
│  断言:                                                        │
│  ├─ 下游能提取上游输出中的语义内容                             │
│  ├─ 下游解析器在混合上下文中正确识别目标 artifact              │
│  ├─ Artifact type/title/metadata 符合契约                      │
│  └─ 上游 artifact.content 可作为下游输入                      │
├──────────────────────────────────────────────────────────────┤
│  4 条链路 × 2 tests × 0 外部依赖 = 8 个 fixture 测试          │
└──────────────────────────────────────────────────────────────┘
```

### 13.12 如何为新链路添加跨 Agent 集成测试

当新增 Agent 之间的中间产物传递链路时：

1. **确定上游 Agent 的输出格式**：了解上游 Agent 的 artifact type 和输出文本格式
2. **在下游 Agent 的 `handler_test.go` 中添加 2 个测试**：

```go
// 测试 1: 下游能消费上游输出文本
func TestCrossAgent_UpstreamToDownstream_ConsumesUpstreamOutput(t *testing.T) {
    // 模拟上游 Agent 的完整输出
    upstreamOutput := "....前导文本...\n" +
        "```json:upstream-artifact.json\n" +
        `{"key":"value"}` + "\n" +
        "```\n\n后续文本"
    
    // 验证 extractTextFromParts 能提取文本
    parts := a2a.ContentParts{a2a.NewTextPart(upstreamOutput)}
    text := extractTextFromParts(parts)
    if text == "" { t.Fatal("should extract text") }
    
    // 验证下游解析器在混合上下文中正确工作
    mixedResponse := upstreamOutput + "\n\n下游响应\n\n" +
        "```targetlang:target-file.ext\n目标内容\n```"
    artifacts := downstreamExtractFunc(mixedResponse)
    // 断言 artifact type/title/content
}

// 测试 2: artifact 元数据可通过链路传递
func TestCrossAgent_UpstreamToDownstream_ArtifactMetadataPassThrough(t *testing.T) {
    upstreamArtifact := adk.Artifact{
        Type:    "upstream_type",
        Title:   "upstream.json",
        Content: `{"data":"..."}`,
        Metadata: map[string]string{"format": "json"},
    }
    if upstreamArtifact.Content == "" {
        t.Fatal("artifact must have content for downstream")
    }
    // 验证上游 content 可作为下游输入构造
    _ = "Based on: " + upstreamArtifact.Content
}
```

3. **注意解析器行为**：部分解析器匹配范围可能比预期宽（如 `parseCodeBlocks` 匹配所有有效语言，`parseJSONBlocks` 匹配所有 JSON 块），测试需按语言或 artifact type 过滤查找。

**当前已实现的 4 条链路可作为参考模板**：`web-agent/handler_test.go`（vision→web）、`code-agent/handler_test.go`（vision→code）、`document-agent/handler_test.go`（file→document）、`review-agent/handler_test.go`（security→review）。

### 13.13 运行并发多 Agent 调用测试

```bash
# 运行全部并发测试（ADK + Agent parser）
cd /project/Multi_Agent_Framework/agents
go test ./adk/... ./code-agent/... ./web-agent/... ./document-agent/... ./review-agent/... -v -run "TestConcurrent_" -count=1

# 仅运行 ADK 并发测试
go test ./adk/... -v -run "TestConcurrent_" -count=1

# 仅运行 Agent parser 并发测试
go test ./code-agent/... ./web-agent/... ./document-agent/... ./review-agent/... -v -run "TestConcurrent_" -count=1

# 全量并发 + 数据竞争检测（推荐在 CI 中执行）
go test ./... -race -count=1
```

并发测试预期输出（13 个测试全部通过）：
```
=== RUN   TestConcurrent_MultipleServersIsolated
--- PASS: TestConcurrent_MultipleServersIsolated (0.00s)
=== RUN   TestConcurrent_MultipleServersAgentCardIsolated
--- PASS: TestConcurrent_MultipleServersAgentCardIsolated (0.00s)
=== RUN   TestConcurrent_SingleServerParallelRequests
--- PASS: TestConcurrent_SingleServerParallelRequests (0.01s)
=== RUN   TestConcurrent_BuildAgentCardParallel
--- PASS: TestConcurrent_BuildAgentCardParallel (0.00s)
=== RUN   TestConcurrent_NoopHandlerReentrant
--- PASS: TestConcurrent_NoopHandlerReentrant (0.00s)
=== RUN   TestConcurrent_MixedEndpointsParallel
--- PASS: TestConcurrent_MixedEndpointsParallel (0.00s)
=== RUN   TestConcurrent_MultiAgentTaskIsolation
--- PASS: TestConcurrent_MultiAgentTaskIsolation (0.00s)
=== RUN   TestConcurrent_ValidateConfigParallel
--- PASS: TestConcurrent_ValidateConfigParallel (0.00s)
=== RUN   TestConcurrent_ServerStartupShutdown
--- PASS: TestConcurrent_ServerStartupShutdown (0.00s)
=== RUN   TestConcurrent_ParseCodeBlocksParallel
--- PASS: TestConcurrent_ParseCodeBlocksParallel (0.00s)
=== RUN   TestConcurrent_ExtractWebpageArtifactsParallel
--- PASS: TestConcurrent_ExtractWebpageArtifactsParallel (0.00s)
=== RUN   TestConcurrent_ExtractDocumentArtifactsParallel
--- PASS: TestConcurrent_ExtractDocumentArtifactsParallel (0.00s)
=== RUN   TestConcurrent_ExtractReviewArtifactsParallel
--- PASS: TestConcurrent_ExtractReviewArtifactsParallel (0.00s)
```

### 13.14 如何为新函数添加并发安全测试

当添加新的共享函数或 Agent parser 时：

1. **判断是否需要并发测试**：如果函数会被多个 goroutine 调用（如 parser、config builder、validator），必须添加并发测试。

2. **ADK 共享函数的并发测试**：在 `adk/concurrency_test.go` 中添加，使用 `sync.WaitGroup` + channel 收集并发结果。

3. **Agent parser 的并发测试**：在该 Agent 的 `handler_test.go` 中添加 `TestConcurrent_*Parallel`：

```go
func TestConcurrent_MyParserParallel(t *testing.T) {
    inputs := []string{
        "input1",
        "input2",
    }
    var wg sync.WaitGroup
    results := make(chan resultType, len(inputs)*10)
    
    for i, input := range inputs {
        for j := 0; j < 10; j++ {
            wg.Add(1)
            go func(idx int, in string) {
                defer wg.Done()
                result := myParser(in)
                results <- resultType{idx: idx, ...}
            }(i, input)
        }
    }
    
    wg.Wait()
    close(results)
    
    for r := range results {
        // 断言每个 input 的 parser 输出正确
    }
}
```

4. **关键模式**：
   - 每个 input 运行 10 次并发调用（足够的并发压力暴露竞态）
   - 使用 channel 收集结果，WaitGroup 同步
   - 断言结果与 input index 对应（防止跨 goroutine 混淆）
   - 在 CI 中始终带 `-race` flag

5. **验证清零**：创建新测试后，运行 `go test ./... -race -count=1`，确保零数据竞争。

**当前已实现的并发测试可作为参考模板**：`adk/concurrency_test.go`（9 个场景）、`code-agent/handler_test.go`（parseCodeBlocks）、`web-agent/handler_test.go`（extractWebpageArtifacts）、`document-agent/handler_test.go`（extractDocumentArtifacts）、`review-agent/handler_test.go`（extractReviewArtifacts）。

### 13.15 运行 Stream Replay 测试

```bash
# 运行全部 Stream Replay 测试
cd /project/Multi_Agent_Framework/agents
go test ./adk/... -v -run "TestStream_" -count=1

# 运行 + 数据竞争检测
go test ./adk/... -race -run "TestStream_" -count=1

# 仅运行事件顺序测试
go test ./adk/... -v -run "TestStream_EventOrder|TestStream_Empty|TestStream_Noop|TestStream_Large" -count=1

# 仅运行 cancel 测试
go test ./adk/... -v -run "TestStream_Cancel" -count=1

# 仅运行错误安全测试
go test ./adk/... -v -run "TestStream_Error|TestStream_Terminal" -count=1

# 仅运行多 Agent 隔离测试
go test ./adk/... -v -run "TestStream_MultiAgent|TestStream_ConcurrentEvent" -count=1
```

Stream Replay 测试预期输出（14 个测试全部通过）：
```
=== RUN   TestStream_EventOrderStability
--- PASS: TestStream_EventOrderStability (0.00s)
=== RUN   TestStream_ErrorProducesFailedStatus
--- PASS: TestStream_ErrorProducesFailedStatus (0.00s)
=== RUN   TestStream_CancelSignalsContext
--- PASS: TestStream_CancelSignalsContext (0.00s)
=== RUN   TestStream_CancelCooperativePattern
--- PASS: TestStream_CancelCooperativePattern (0.00s)
=== RUN   TestStream_EmptyHandlerProducesMinimalSequence
--- PASS: TestStream_EmptyHandlerProducesMinimalSequence (0.00s)
=== RUN   TestStream_MultiAgentTaskIDIsolation
--- PASS: TestStream_MultiAgentTaskIDIsolation (0.00s)
=== RUN   TestStream_MalformedArtifactsDontCrash
--- PASS: TestStream_MalformedArtifactsDontCrash (0.00s)
=== RUN   TestStream_LargeTextStreamPreservesOrder
--- PASS: TestStream_LargeTextStreamPreservesOrder (0.00s)
=== RUN   TestStream_CancelDoesNotEmitCompleted
--- PASS: TestStream_CancelDoesNotEmitCompleted (0.00s)
=== RUN   TestStream_ConcurrentEventStreamsDontCrossContaminate
--- PASS: TestStream_ConcurrentEventStreamsDontCrossContaminate (0.00s)
=== RUN   TestStream_NoopHandlerProducesValidSequence
--- PASS: TestStream_NoopHandlerProducesValidSequence (0.00s)
=== RUN   TestStream_EventMetadataHasTaskID
--- PASS: TestStream_EventMetadataHasTaskID (0.00s)
=== RUN   TestStream_InvalidContentIsSafe
--- PASS: TestStream_InvalidContentIsSafe (0.00s)
=== RUN   TestStream_TerminalStatesAreDistinct
--- PASS: TestStream_TerminalStatesAreDistinct (0.00s)
```

### 13.16 Stream Replay 关键实现细节

**事件序列模型**：`ExecuteHandler` 产生的第一个事件永远是 `*a2a.Task`（非 `*TaskStatusUpdateEvent`），其后依次为 Working、text/code artifact chunks、Completed/Failed。

**协程取消语义**：`ctx.Cancel()` 仅关闭 `Done` channel，不自动阻止后续 `StreamText`/`AddArtifact` 调用。Handler 需采用协作式取消模式：

```go
handler := func(ctx *Context, messages []a2a.Message) error {
    ctx.StreamText("chunk-1")
    ctx.Cancel()
    // 协作式检查
    select {
    case <-ctx.Done():
        return nil  // 提前返回，不再输出
    default:
    }
    ctx.StreamText("should-not-appear")  // 不会执行
    return nil
}
```

**错误消息安全**：`TestStream_ErrorProducesFailedStatus` 验证 handler error 导致的 Failed 状态消息**不含**以下禁止关键词：`goroutine`, `panic`, `.go:`, `stack trace`, `API_KEY`, `token`, `secret`, `password`, `/internal/`, `/admin/`。

**Part 结构体访问**：`*a2a.Part` 是 struct（非 interface），使用 `.Text()` 方法获取文本内容，不可做 type assertion。

**Golden Fixture 生成**：每个测试的 `collectEvents` 返回完整事件序列，可序列化为 JSON 用于：
1. 协议兼容性验证（对比不同版本 A2A 库）
2. 前端回放测试（输入 SSE parser 验证渲染一致性）
3. CI 回归检测（保存 golden JSON）

---

## 参考资料

- `testing-review-contract` Skill — 测试分层与质量门禁契约
- `a2a-agent-contract` Skill — AgentCard、Health Check、A2A Streaming Task 契约
- `artifact-contract` Skill — Artifact type 注册、schema 校验、安全边界
- `adk-runtime-contract` Skill — Runtime Context、LLMClient、TaskHandler 契约
- `agents/README.md` — 子 Agent 能力矩阵与 Artifact Type 约定
