---
name: commit-security-review
description: "用于定义代码提交安全审查契约，包括 commit message 禁止模式、敏感信息泄露检测、文件级安全检查、pre-commit hook 规则、CI 安全门禁和 PR 安全 Review Checklist。"
---

# commit-security-review

## 1. Skill 目的

本 Skill 定义 AgentHub 代码提交的安全审查契约。

每次 `git commit` 和 PR merge 前，必须通过本契约定义的安全检查：commit message 不得包含禁止模式，提交内容不得泄露密钥、token、IP 地址、内部地址、证书、私钥或其他敏感信息。

本 Skill 的目标是让安全审查成为自动化门禁，而非事后补救。

## 2. 独立性原则

本 Skill 必须独立可读。读者不需要先阅读其他 Skill，也能理解什么内容禁止出现在 commit 中。

本 Skill 不定义：
- 业务代码安全（由 `security-boundary-contract` 负责）。
- 数据持久化安全（由 `data-persistence-contract` 负责）。
- 运行时安全边界（由 `security-boundary-contract` 负责）。
- 测试策略（由 `testing-review-contract` 负责）。

本 Skill 只负责 **commit 阶段** 的安全审查。

## 3. 当前阶段识别

```text
profile = v1.0-sprint
scope = all-commits
enforcement = pre-commit + CI gate + PR review
```

本 Skill 适用于所有分支、所有文件的 commit，不区分前后端、Agent 或 Demo 代码。

## 4. 本 Skill 负责什么

- Commit message 禁止模式检测。
- 敏感信息泄露检测（KEY、token、密码、证书、私钥、IP 地址、内部地址等）。
- 高危文件类型检测（.pem、.key、.p12、.pfx、credentials、.env 等）。
- Pre-commit hook 规则。
- CI 安全门禁规则。
- PR 安全 Review Checklist。
- 历史 commit 中敏感信息的处理指导。

## 5. 本 Skill 不负责什么

- 业务代码中的 SQL 注入、XSS、SSRF 等漏洞检测。
- 运行时 secret 管理（secret manager / vault 集成）。
- 数据库权限审查。
- 网络安全策略。
- 第三方依赖漏洞扫描。

## 6. 核心原则

1. **提交即审查**：每次 commit 必须经过安全检查，不得事后补救。
2. **敏感信息不入库**：任何形式的 KEY、token、密码、私钥、证书、IP 地址、内部地址、环境变量中的 secret 不得进入 git 历史。
3. **Message 可审计**：commit message 不得包含任何可能被外部解读为 AI 模型名称、内部代号或敏感上下文的模式。
4. **失败阻断**：安全检查失败时，commit 或 merge 必须被阻断，不得绕过。
5. **不依赖人工判断**：所有检测规则必须是确定性的、可脚本化的，不得依赖 "Reviewer 自己判断"。

## 7. Commit Message 禁止模式

### 7.1 禁止的字符串

Commit message（包括 subject 和 body）中，以下字符串（不区分大小写）必须被拒绝：

```text
claude
CLAUDE
Claude
anthropic
Anthropic
```

检测时必须使用不区分大小写的匹配。

### 7.2 规则

- 禁止出现在 subject 行。
- 禁止出现在 body 任意行。
- 禁止出现在 `Co-Authored-By` 等 trailer 行。
- 禁止以任何变体形式出现（如 `cl@ude`、`CLA UDE`、`ClaudeCode` 等）。
- 上述禁止模式不仅适用于完整单词，也适用于包含这些字符串的组合词。
- hook script 和 CI 脚本中的工具引用名必须使用中性替代名（如 `ai-assistant`、`code-helper`），不得直接使用禁止模式中的字符串作为脚本标识符。

### 7.3 允许的替代

如果需要引用 AI 辅助编写，使用以下中性表述：

```text
Co-Authored-By: AI Assistant <assistant@agent-hub.local>
```

或使用通用的 `pair-programming`、`automated-review` 等描述。

## 8. 敏感信息泄露检测

### 8.1 高置信度禁止模式

以下模式出现在任何提交文件中，必须阻断：

| 模式 | 示例 | 说明 |
|---|---|---|
| API Key（高熵 base64） | `sk-[A-Za-z0-9]{32,}`、`AKIA[A-Z0-9]{16}` | OpenAI、AWS 等平台 key 前缀 |
| 私钥（PEM header） | `-----BEGIN (RSA\|EC\|DSA\|OPENSSH)? ?PRIVATE KEY-----` | PEM 格式私钥 |
| JWT Token | `eyJ[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}` | 结构化的 JWT |
| 数据库连接串 | `mysql://`、`postgresql://`、`mongodb://`、`redis://` 带密码 | 包含凭据的连接串 |
| 高熵 secret（>20 位 base64） | `[A-Za-z0-9+/=]{40,}` | 需要上下文判断 |
| GitHub Token | `ghp_[A-Za-z0-9]{36}`、`github_pat_[A-Za-z0-9]{22,}` | GitHub 个人访问令牌 |
| Slack Webhook | `https://hooks.slack.com/services/[A-Z0-9/]+` | Slack webhook URL |

### 8.2 中置信度禁止模式

以下模式出现在提交文件中，应触发 Review 阻断：

| 模式 | 说明 |
|---|---|
| `password\s*[:=]\s*[^\n'" ]+` | 明文密码赋值 |
| `secret\s*[:=]\s*[^\n'" ]+` | secret 字段赋值 |
| `token\s*[:=]\s*[^\n'" ]+` | token 字段赋值 |
| `api[_-]?key\s*[:=]\s*[^\n'" ]+` | API key 赋值 |
| `access[_-]?key\s*[:=]\s*[^\n'" ]+` | Access key 赋值 |
| `AUTH_TOKEN\s*[:=]\s*[^\n'" ]+` | Auth token 赋值 |
| `PRIVATE_KEY\s*[:=]` | 私钥变量赋值 |

### 8.3 IP 地址与内部地址

以下模式必须被检测并阻断：

| 模式 | 说明 |
|---|---|
| 公网 IPv4 地址 | `\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}` — 排除 `127.0.0.1`、`0.0.0.0`、`localhost` 和文档中的示例 IP（如 `192.0.2.x`、`198.51.100.x`、`203.0.113.x`） |
| 内网 IPv4 地址 | `10\.\d{1,3}\.\d{1,3}\.\d{1,3}`、`172\.(1[6-9]\|2\d\|3[01])\.\d{1,3}\.\d{1,3}`、`192\.168\.\d{1,3}\.\d{1,3}` |
| 内部服务名 | 包含真实内部 hostname 或 service name 的硬编码（如 `internal-api.prod.example.com`、`prod-db.internal`） |
| 内网 URL | 包含内网 IP 或内部 hostname 的完整 URL |

允许：
- `127.0.0.1`（本地回环）。
- `0.0.0.0`（绑定所有接口）。
- `localhost`。
- Docker Compose 中的 service name（如 `http://gateway:8080`、`http://code-agent:8081`）——这些是容器网络内部名称。
- 文档中明确标注的示例 IP（如 `192.0.2.1`、`198.51.100.1`、`203.0.113.1`）。

### 8.4 高危文件类型

以下文件名或扩展名的文件，除非在 `.gitignore` 或明确用于测试 fixture（且内容脱敏），否则不得提交：

```text
.env
.env.local
.env.production
.env.development
*.pem
*.key
*.p12
*.pfx
*.jks
*.keystore
*.pkcs12
credentials.json
credentials.yml
credentials.yaml
service-account.json
service-account.yaml
secret.env
secrets.yml
secrets.yaml
```

例外：
- 测试 fixture 中的 mock 凭据文件（如 `test/fixtures/mock-credentials.json`），必须使用占位符且文件名包含 `mock` 或 `fixture`。
- Docker Compose `.env.example` 文件（只允许示例值，不允许真实凭据）。

## 9. Pre-commit Hook 规则

推荐 pre-commit hook 执行以下检查：

1. **Commit message check**：使用正则检测第 7 节禁止的字符串（不区分大小写）。
2. **Secret scan**：使用第 8.1 节的高置信度模式扫描 staged 文件。
3. **File type check**：检测第 8.4 节的高危文件类型。
4. **Diff scan**：只扫描 `git diff --cached` 的新增行，避免旧代码噪音。

规则：

- 任一检查失败，阻断 commit。
- 阻断时必须输出明确信息和文件路径。
- 不得在 hook 中使用真实 secret 做对比。
- Hook script 本身不得包含禁止模式中的字符串作为函数名、变量名或注释中的标识符。

## 10. CI 安全门禁

CI Pipeline 必须包含以下安全门禁：

1. **Secret scan**：全量扫描 PR 所有变更文件。
2. **Commit message scan**：扫描 PR 中所有 commit message。
3. **高危文件检测**：检查是否有高危文件被意外添加。
4. **历史 leak 检测**：使用 `git log` + 正则检测历史 commit 中是否有敏感模式新增。

规则：

- CI 安全门禁失败时，PR 不得 merge。
- CI 输出不得包含检测到的敏感信息原文（只输出文件名和行号）。
- 安全门禁必须不可被 `/override` 或 label 绕过。

## 11. 发现敏感信息后的处理

如果 commit 或 PR 中发现敏感信息：

1. **立即阻断**：禁止 merge，标记为 security-blocked。
2. **撤销泄露**：
   - 如果敏感信息在最新 commit：使用 `git commit --amend` 或 `git reset` 移除后重新 commit。
   - 如果敏感信息在历史 commit：使用 `git filter-branch` 或 `BFG Repo-Cleaner` 清理。
3. **轮换凭据**：泄露的 KEY、token、密码必须在对应的外部系统立即撤销并重新生成。
4. **审计**：记录发现时间、文件路径、处理方式。

## 12. Contract Test 规则

本 Skill 的安全检测规则自身应可测试：

- 包含禁止模式的 commit message 被拒绝。
- 包含模拟私钥的文件被检测。
- 包含模拟 API key 的文件被检测。
- 包含真实 IP 地址的文件被检测。
- 合法 commit message 通过检查。
- 合法代码变更通过检查。
- 高危文件名被检测。

测试 fixture 必须使用明确的占位符（如 `sk-test-mock-key-not-real`、`TEST_API_KEY=test-only`），不得在测试中生成真实高熵 key。

## 13. Review Checklist

每次 commit 和 PR 的安全审查必须检查：

### Commit Message

- [ ] Message 中是否不含禁止模式（不区分大小写）？
- [ ] Subject 和 body 是否全部通过检测？
- [ ] Trailer 行是否安全？

### 敏感信息

- [ ] Diff 中是否无 API key、token、password、secret 明文？
- [ ] Diff 中是否无 PEM 格式私钥？
- [ ] Diff 中是否无数据库连接串含密码？
- [ ] Diff 中是否无 JWT token？
- [ ] Diff 中是否无公网真实 IP 地址？
- [ ] Diff 中是否无内网真实 IP 地址或内部 hostname？
- [ ] 环境变量文件是否只包含示例值（如 `.env.example`）？

### 文件检查

- [ ] 是否无 `.env`（非 example）文件？
- [ ] 是否无 `*.pem`、`*.key`、`*.p12` 等私钥文件？
- [ ] 是否无 `credentials.json` 等凭据文件？
- [ ] 例外文件是否明确标记为 mock/fixture 且内容脱敏？

### 自动化

- [ ] Pre-commit hook 是否已配置？
- [ ] CI 安全门禁是否已配置？
- [ ] 安全门禁是否不可被绕过？

## 14. 硬性规则

1. Commit message 不得包含 `claude`（不区分大小写及任何变体）。
2. 不得提交任何形式的真实 API key、token、密码。
3. 不得提交 PEM 格式私钥。
4. 不得提交包含真实凭据的数据库连接串。
5. 不得提交 JWT token。
6. 不得提交真实公网或内网 IP 地址（Docker Compose service name 和文档示例 IP 除外）。
7. 不得提交 `.env` 文件（`.env.example` 除外）。
8. 不得提交私钥文件（`.pem`、`.key`、`.p12` 等）。
9. Pre-commit hook 和 CI 安全门禁不得被绕过。
10. 发现敏感信息后必须立即轮换外部凭据。

## 15. 完成定义

本 Skill 视为完成，当且仅当：

- Commit message 禁止模式明确。
- 敏感信息检测模式明确（高置信度 + 中置信度）。
- IP 地址与内部地址检测规则明确。
- 高危文件类型清单明确。
- Pre-commit hook 规则可落地。
- CI 安全门禁规则明确。
- 敏感信息泄露后的处理流程明确。
- Review Checklist 可用于每次 PR。

---

## References

- `references/forbidden-patterns.md` — 完整的禁止模式正则清单（待创建）。
- `references/sensitive-file-types.md` — 高危文件类型详细说明（待创建）。
