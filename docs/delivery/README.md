# AgentHub v1.0 比赛交付包

本目录是 AgentHub v1.0 比赛提交的最终交付文档集合。

## 推荐阅读顺序（评委）

1. **[final-architecture-overview.md](./final-architecture-overview.md)** — 架构总览（含 Mermaid 图）
2. **[final-feature-checklist.md](./final-feature-checklist.md)** — 功能清单与完成状态
3. **[demo-walkthrough.md](./demo-walkthrough.md)** — 演示脚本与讲解词
4. **[test-and-ci-evidence.md](./test-and-ci-evidence.md)** — 测试与 CI 证据
5. **[security-and-risk-review.md](./security-and-risk-review.md)** — 安全与风险审计
6. **[v1.1-roadmap.md](./v1.1-roadmap.md)** — 后续规划
7. **[submission-package-index.md](./submission-package-index.md)** — 提交包索引

## 核心入口

| 入口 | 位置 |
|------|------|
| 项目 README | `../README.md` |
| 架构文档 | `../refactor/current-architecture-state.md` |
| Legacy 边界 | `../refactor/legacy-boundary.md` |
| 一键启动 | `docker compose -f docker-compose.new-arch.yml up --build` |
| 验收脚本 | `bash ./smoke-new-arch.sh` |

## 文档地图

```
docs/
├── delivery/                          ← 本目录（比赛交付包）
│   ├── README.md                      ← 你在这里
│   ├── final-architecture-overview.md
│   ├── final-feature-checklist.md
│   ├── test-and-ci-evidence.md
│   ├── demo-walkthrough.md
│   ├── security-and-risk-review.md
│   ├── v1.1-roadmap.md
│   └── submission-package-index.md
├── refactor/                          ← 实施过程文档
│   ├── productization-stage-guide.md
│   ├── current-architecture-state.md
│   ├── legacy-boundary.md
│   ├── persistence-implementation-plan.md
│   ├── release-checklist-v1.0.md
│   ├── demo-script-v1.0.md
│   └── ... (phase reports)
└── contracts/                         ← 协议契约
```

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Delivery Package
