# 安装说明

## 推荐安装：只安装新增 skill

```bash
unzip agenthub-llm-orchestration-cc-execution-pack-v2.zip
cd agenthub-llm-orchestration-cc-execution-pack-v2
./scripts/install-cc-execution-pack.sh new-only
```

该模式复制：

```text
.claude/skills/llm-orchestration-dev
.claude/skills/llm-orchestration-review
.agents/skills/llm-orchestration-dev
.agents/skills/llm-orchestration-review
docs/plans/*
```

## 可选安装：同步全部 skill 快照

```bash
./scripts/install-cc-execution-pack.sh all-skills
```

该模式会复制包内所有 `.claude/skills` 和 `.agents/skills`。

## 执行

在 Claude Code 输入：

```text
/llm-orchestration-dev
```

再输入：

```text
按 docs/plans/llm-orchestration-development-plan.md 执行。从 Phase 0 开始，每个 Phase 完成后停止并输出报告。
```

## 审核

```bash
./scripts/collect-review-evidence.sh
```

把输出文件和 phase reports 给审核方。
