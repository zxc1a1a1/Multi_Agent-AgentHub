# project-architecture 执行清单

## 1. 当前阶段目标

围绕 `project-architecture` Skill，当前阶段只完成：

1. 架构文档。
2. 服务边界文档。
3. Contract 文件清单。
4. 第一阶段 Mock-first 任务拆分。
5. Review checklist。

暂时不要写业务实现代码。

---

## 2. 本阶段应创建的文件

```text
docs/architecture/overview.md
docs/architecture/service-boundaries.md
docs/architecture/frontend.md
docs/architecture/gateway.md
docs/architecture/orchestrator.md
docs/architecture/child-agents.md
docs/contracts/README.md
```

---

## 3. 本阶段不做什么

暂时不做：

- 不初始化 React 项目。
- 不初始化 Go 服务。
- 不写数据库模型。
- 不写真实 API handler。
- 不写真实 A2A Client。
- 不接真实 LLM。
- 不写 Docker Compose。

这些属于后续阶段。

---

## 4. 下一步应进入的 Skill

完成本阶段后，下一步应开始：

```text
platform-api-contract
```

原因：REST API 和 OpenAPI 是 Gateway 与 Frontend 的持久化资源 contract，必须在写前后端代码之前固定。

---

## 5. 给 Coding Agent 的推荐 Prompt

```text
使用 project-architecture。
根据 PDR 生成 AgentHub 的架构文档、服务边界文档、Contract 文件清单和第一阶段 Mock-first 开发任务。
先不要写业务实现代码。
必须保证 Frontend 只连 Gateway，Gateway 不写意图编排，Orchestrator 负责 A2A 调度，Child Agents 暴露 AgentCard。
```

---

## 6. Review Checklist

- 是否明确 Frontend 只连接 Gateway？
- 是否明确 Gateway 不做编排？
- 是否明确 Orchestrator 独立于 Gateway？
- 是否明确 Orchestrator 通过 A2A 调用 Child Agents？
- 是否明确 Child Agents 暴露 AgentCard？
- 是否明确 Artifact 映射 Frontend Skills？
- 是否明确 AG-UI、REST API、A2A 的职责？
- 是否没有提前写业务代码？

---

## MVP v0.1 架构检查项

- [ ] Orchestrator 可以嵌入 Gateway 进程，但 handler / orchestrator / a2a / converter 模块边界清晰。
- [ ] MVP 只实现 `code-agent`。
- [ ] MVP 只实现 `code_preview`。
- [ ] MVP 使用 MySQL 8。
- [ ] MVP 暂不使用 Redis。
- [ ] MVP 暂不使用对象存储。
- [ ] MVP 使用固定 Token 或环境变量 Token。
- [ ] MVP Agent 注册可以使用配置文件。
- [ ] Frontend 仍然只连接 Gateway。
- [ ] Gateway 仍然不直接调用 Child Agent，必须通过 Orchestrator 模块。
- [ ] Orchestrator 仍然通过 A2A 调用 `code-agent`。
- [ ] Artifact 仍然通过 AG-UI Tool Call 映射到 Frontend Skill。
- [ ] 未提前实现群聊、自建 Agent、web-agent、doc-agent 或复杂多 Agent 编排。
