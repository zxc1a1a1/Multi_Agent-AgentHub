# 模块拆分重构风险清单

## 1. 构建风险

- `go.work` 可能影响现有 `agents/server` 的 `go test` 解析路径与依赖选择。
- 新 module path 若与当前仓库路径不一致，可能导致 import 断裂。
- 多 module 并存后，import path 需要统一规划，避免交叉引用混乱。
- Windows 路径分隔符与 CRLF 差异可能触发格式或脚本告警。
- `node_modules` 不应被扫描为构建产物或被纳入 Go 构建分析范围。

## 2. 功能风险

- Gateway/Orchestrator 拆分若越界，可能破坏现有聊天主链路。
- A2A 转 AG-UI 事件映射不一致，可能影响 artifact 预览与前端消费。
- Session/Conversation 持久化职责拆分不当，可能影响历史消息完整性。
- 多 Agent streaming 聚合策略变化，可能改变前端展示顺序与归属。

## 3. 安全风险

- `.env` 被误纳入版本控制。
- API Key / Token / `DATABASE_URL` / 私钥被写入配置、文档或日志。
- `contentRef` / `artifactRef` 使用真实授权 URL 造成凭据泄露。
- 前端误执行 Agent 输出 HTML/JS，触发 XSS 或沙箱逃逸风险。
- `docker-compose` 暴露敏感环境变量导致泄露风险。

## 4. 协作风险

- 同事前端/API 联调期间若破坏主链路，会导致并行任务阻塞。
- 大规模移动文件会导致 PR 难以审查和回滚。
- `docs/skill` 已在远端删除，误恢复会制造无效冲突。
- 模块重构与统一 Agent 编排任务并行时，容易产生交叉冲突。

## 5. 每轮提交前检查

```powershell
git status --short
git diff --stat
git diff --cached --name-status
git ls-files .env
git diff --cached | Select-String "sk-[A-Za-z0-9_-]{20,}"
git diff --cached | Select-String "OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DB_PASSWORD|MYSQL_ROOT_PASSWORD|DATABASE_URL|PRIVATE KEY|BEGIN RSA"
Get-ChildItem . -Recurse -Filter "*.exe" | Where-Object { $_.FullName -notmatch "\\node_modules\\" -and $_.FullName -notmatch "\\.git\\" }
```
