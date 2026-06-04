# Legacy Cleanup Final Review

## 1. 审计范围

对 AgentHub 仓库进行最终 legacy cleanup 审计，目标：识别可安全删除的临时文件、重复文档、过期引用。

## 2. 审计方法

- 检查根目录 `*.md` 文件
- 检查 `docs/refactor/` 文件的交叉引用链
- 检查 `server/` 和 `agents/` 目录使用状态
- 用 `rg` 验证删除候选无引用

## 3. 本轮删除

**无文件删除。**

审计结论：不满足安全删除条件的文件。

## 4. 保留文件及原因

### 4.1 根目录

| 文件 | 原因 |
|------|------|
| `readme.md` | 项目主 README，当前使用中 |
| `AGENTS.md` | 项目 AI 协作配置 |
| `AgentHub_重构与模块解耦阶段报告.md` | 用户自有文档（未跟踪），不碰 |

### 4.2 server/ 目录

**保留，不删除。**

原因：
- MVP v0.1 legacy Gateway + 内嵌 Orchestrator
- 作为历史参考和回归测试基线
- README 中已明确标注为 Legacy 路径
- 一次性大规模搬迁工作量大、风险高

### 4.3 agents/ 目录

**保留，不删除。**

原因：
- 旧 Agent 能力池和未来服务化候选池
- vision-agent、file-agent、document-agent 等尚未服务化
- README 中已明确标注边界
- 按能力价值和契约成熟度分批搬迁是正确策略

### 4.4 docs/refactor/ 目录

**保留所有文件。**

原因：
- phase-*.md 系列文件通过 `legacy-cleanup-candidates.md` 和 `frontend-multi-agent-ui-rendering-report.md` 交叉引用
- 每个 phase 报告是实施历史记录，有追溯价值
- persistence-implementation-plan.md 是当前主计划文件

### 4.5 docker-compose.yml

**保留。**

原因：
- Legacy compose（单 Agent + MySQL + 真实 LLM）
- 与 `docker-compose.new-arch.yml` 平行存在，用途不同
- 保留作为历史参考

### 4.6 Makefile

**保留。**

原因：
- 包含 legacy dev/install/build 命令
- 可能有开发者依赖

### 4.7 init.sql

**保留（列入 future candidate）。**

原因：
- 旧 MySQL 初始化脚本
- 需要确认无引用后再删除

## 5. Future Cleanup Candidates

| 候选 | 条件 |
|------|------|
| `init.sql` | 确认无脚本/CI/README 引用后删除 |
| `docs/refactor/phase-*.md` | v1.1 后可统一归档为 archive/ 目录 |
| `server/` 整体 | 当 services/gateway + services/orchestrator 完全覆盖 server/ 功能后 |
| `agents/` 中已服务化部分 | 当对应 services/agents/* 完成服务化后移除旧实现 |

## 6. 边界说明

在当前 v1.0 交付阶段：

- **主路径**: `services/gateway` → `services/orchestrator` → `services/agents/*`
- **Legacy 历史参考**: `server/`、`agents/`（未服务化部分）、`docker-compose.yml`
- **新架构 compose**: `docker-compose.new-arch.yml`
- **新架构 smoke**: `smoke-new-arch.sh`
- **新架构 CI**: `.github/workflows/new-arch-smoke.yml`

所有新功能开发必须在主路径进行。Legacy 路径保留但不扩展。

## 7. 审计日期

2026-06-04

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Legacy Cleanup Final Review
