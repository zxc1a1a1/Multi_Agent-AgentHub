# 中文指导文件：如何使用 v2 包

## 1. 安装

在项目根目录执行：

```bash
unzip agenthub-llm-orchestration-cc-execution-pack-v2.zip
cd agenthub-llm-orchestration-cc-execution-pack-v2
./scripts/install-cc-execution-pack.sh new-only
```

默认 `new-only` 不覆盖你原来的 18 个 skill，只新增：

```text
llm-orchestration-dev
llm-orchestration-review
```

## 2. 让 CC 执行

在 Claude Code 中输入：

```text
/llm-orchestration-dev

按 docs/plans/llm-orchestration-development-plan.md 执行。
从 Phase 0 开始。
每个 Phase 完成后停止并输出报告，不要自动进入下一 Phase。
```

## 3. 每个 Phase 必须停

如果 CC 自动想继续，让它停止。  
每个 Phase 都要先给你报告。

## 4. 防止乱改的关键

本包已经把禁止路径写进 skill、references、计划书和脚本：

```text
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**
```

执行中可以随时运行：

```bash
./scripts/verify-forbidden-paths.sh
```

## 5. 审核时给我什么

让 agent 输出：

```text
Phase Reports
Final Report
go test 输出
```

你再运行：

```bash
./scripts/collect-review-evidence.sh
```

把生成的 `llm-orchestration-review-evidence.txt` 和 agent 报告贴给我。

我会按 `/llm-orchestration-review` 检查。
