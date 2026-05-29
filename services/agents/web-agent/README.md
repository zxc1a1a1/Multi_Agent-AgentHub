# services/agents/web-agent

这是新架构下的 `web-agent v0.1` 最小迁移骨架（Phase 8.5~8.7）。

## 当前范围

- 基于 `pkg/adk` 实现最小 `WebAgent`（mock/minimal）
- 基于 `pkg/adk/a2a` 暴露最小 A2A Server：
  - `GET /health`
  - `GET /.well-known/agent.json`
  - `POST /`
  - `POST /a2a/tasks/sendSubscribe`
- 提供最小 web tools：
  - `generate_html_snippet`
  - `summarize_ui_request`
- 提供 `httptest` round-trip 集成测试（`a2a.Client` / `a2a.RemoteAgent`）

## 关键说明

- 当前不接真实 LLM。
- 不读取 `.env` 文件，不依赖真实密钥。
- 不访问真实外网，不连接真实数据库。
- 旧 `agents/web-agent` 仍保留，尚未删除。
- 当前 HTML 输出是安全模板化片段，不包含 `script`、`iframe`、外链和事件处理属性。

## 当前声明 skills

- `web_generation`
- `html_generation`
- `ui_summarization`

## 当前声明 inputModes

- `text`
- `image_ref`
- `extracted_text`
- `vision_analysis`

## 当前声明 outputModes

- `text`
- `webpage`
- `html`
- `artifact_ref`

## 后续工作（不在本轮）

- 接入真实 provider 与模型调用
- 对接正式 artifact 归一化与存储
- 对接 Docker Compose 运行拓扑
- 对接 Gateway/Orchestrator 完整链路与前端 `web_preview`

