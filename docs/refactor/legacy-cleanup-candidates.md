# Legacy Cleanup Candidates

## 1. 背景

AgentHub 仓库经过多轮重构（Phase 0 → Phase 8 → Phase 9 → Phase 10 → v1.0 Productization Stage），`docs/refactor/` 目录累积了大量历史阶段报告。当前有 27 个文件，其中很多是已完成阶段的产物，不再反映当前架构状态。

本文档记录清理候选，遵循最小化原则：**本轮不执行大规模删除**，仅记录候选供后续统一清理。

## 2. 当前新旧路径边界

| 路径 | 定位 | 操作 |
|------|------|------|
| `services/*` | 新架构主路径 | 保留 |
| `agents/` | 旧 Agent 能力池 | 不删，候选池 |
| `server/` | legacy 路径 | 不删 |
| `docker-compose.new-arch.yml` | 新架构 compose | 保留 |
| `docker-compose.yml` | legacy compose | 不删 |
| `docs/refactor/*` | 历史报告 | 候选归档 |

## 3. 清理原则

1. 仍被 `readme.md`、CI、Makefile、docker-compose、frontend、services 引用的文件 → **保留**
2. 已完成阶段的历史报告，仅在 `docs/refactor/` 内部交叉引用 → **归档候选**
3. 明显过时且无引用的临时文件 → **删除候选**
4. 当前活跃使用的架构/边界/guide 文档 → **保留**
5. 不执行大规模删除，不删除 `server/`、`agents/`、`docker-compose.yml`

## 4. 候选列表

### 4.1 保留（活跃引用）

| 文件 | 用途 | 保留原因 |
|------|------|----------|
| `productization-stage-guide.md` | v1.0 产品化阶段总指导 | 活跃引用 |
| `current-architecture-state.md` | 当前架构事实源 | 活跃引用 |
| `legacy-boundary.md` | Legacy 与新架构边界 | 活跃引用 |
| `frontend-multi-agent-sse-audit.md` | Step 1-D 审计产物 | Step 2-A/2-B 前置文档 |
| `frontend-multi-agent-ui-rendering-report.md` | Step 2-B/2-C 交付报告 | 当前活跃 |
| `current-to-target-module-map.md` | 模块迁移映射 | 迁移期参考 |
| `module-separation-migration-plan.md` | 模块分离计划 | 迁移期参考 |
| `refactor-risk-checklist.md` | 重构风险清单 | 历史参考 |
| `orchestrator-v1-implementation-audit-guide.md` | Orchestrator 审计指南 | 与当前架构相关 |

### 4.2 归档候选（历史阶段报告，已不再更新）

这些文件是过去 Phase 的产物，阶段已完成，内容不会继续更新。建议后续统一移动到 `docs/refactor/archive/` 目录：

| 文件 | 阶段 | 当前引用状态 |
|------|------|-------------|
| `phase-0-readiness-report.md` | Phase 0 | 未被其他文档引用 |
| `phase-8-preflight-report.md` | Phase 8 | 仅自引用 |
| `phase-8-code-agent-migration-report.md` | Phase 8 | 被 phase-8-* 同组文档引用 |
| `phase-8-web-agent-migration-report.md` | Phase 8 | 被 phase-8-* 同组文档引用 |
| `phase-8-code-agent-gateway-link-report.md` | Phase 8 | 被 phase-8-* 同组文档引用 |
| `phase-8-gateway-agent-routing-report.md` | Phase 8 | 被 phase-8-* 同组文档引用 |
| `phase-9-frontend-agent-routing-report.md` | Phase 9 | 被本报告交叉引用 |
| `phase-9-4-9-6-gateway-demo-report.md` | Phase 9 | 被本报告交叉引用 |
| `phase-10-demo-stabilization-report.md` | Phase 10 | 被 phase-10-* 同组文档引用 |
| `phase-10-demo-finalization-report.md` | Phase 10 | 被 phase-10-* 同组文档引用 |
| `phase-10-docker-new-arch-report.md` | Phase 10 | 被本报告交叉引用 |
| `new-arch-demo-handoff.md` | Demo | 被本报告交叉引用 |
| `new-arch-demo-troubleshooting.md` | Demo | 未被其他文档引用 |
| `new-arch-demo-acceptance-checklist.md` | Demo | 未被其他文档引用 |
| `new-arch-docker-demo-guide.md` | Demo | 未被其他文档引用 |
| `new-arch-final-acceptance-report.md` | Demo | 未被其他文档引用 |

### 4.3 审查候选人（需确认是否过时）

| 文件 | 当前内容 | 建议 |
|------|----------|------|
| `agent-capability-inventory.md` | Agent 能力清单 | 可能需要保留（迁移参考），但内容与当前 services/agents/* 可能不一致 |
| `agent-consolidation-plan.md` | Agent 合并计划 | 可能需要保留（迁移参考），但计划可能已过期 |
| `agent-reference-scan-report.md` | Agent 引用扫描 | 可能需要更新以反映当前状态 |

## 5. 暂不删除原因

1. **交叉引用链**：历史报告之间互相引用，删除一个会导致其他文档的链接失效
2. **不属于 Step 2-B/2-C 范围**：大规模 `docs/refactor/` 清理不在前端 multi-agent UI rendering 范围内
3. **需要统一归档策略**：应先确定 archive 策略（移动 vs 删除 vs Git 历史），再统一执行
4. **可能有调试/回溯价值**：部分历史报告包含当时的决策理由和 trade-off 记录
5. **CI/smoke/release 可能隐式引用**：虽未发现直接引用，但不能完全排除

## 6. 后续建议清理顺序

1. **Step 2-B/2-C 提交后**：将 phase-0 至 phase-10 的历史报告统一移动到 `docs/refactor/archive/`
2. **Demo 文档**：如果 new-arch demo 已稳定，可将 demo 相关文档归档
3. **Agent 能力文档**：在 Agent 服务化完成后，更新或删除旧的 agent-capability-inventory 等
4. **最终清理**：在 v1.0 发布前，删除所有 archive/ 中确认无用的文件

## 7. 版本

- 创建日期：2026-06-04
- 关联步骤：AgentHub v1.0 Productization Stage Step 2-C
- 状态：候选记录，不执行删除
