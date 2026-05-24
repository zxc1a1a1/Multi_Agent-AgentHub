# Frontend Security

来源：`docs/contracts/security-boundaries.md`、`docs/contracts/sandbox-policy.md`。

- 前端不直接接触后端密钥。
- 前端不直连 Orchestrator / A2A。
- `code_preview` 只展示不执行。
- 预览类能力需隔离渲染（Post-MVP `web_preview` 用 sandbox iframe）。
