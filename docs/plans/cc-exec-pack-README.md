# AgentHub LLM 编排 CC 执行包 v2

这是重新梳理后的 Claude Code / CC 执行包，重点是**防止 agent 乱操作**。

## 包含内容

```text
1. 当前项目已有 .claude/skills 快照
2. 当前项目已有 .agents/skills 快照
3. 新增 .claude/skills/llm-orchestration-dev
4. 新增 .claude/skills/llm-orchestration-review
5. 新增 .agents/skills/llm-orchestration-dev
6. 新增 .agents/skills/llm-orchestration-review
7. docs/plans/llm-orchestration-development-plan.md
8. docs/plans/llm-orchestration-cc-master-prompt.md
9. docs/plans/llm-orchestration-review-handoff.md
10. scripts 安装、检查、审核证据收集脚本
```

## v2 相比上一版加强点

```text
Phase Gate 更硬：每个 Phase 必须停下来报告
禁止路径更明确：frontend/Gateway/docker/ADK/runtime AG-UI/旧目录全部禁止
每个 Phase 都有允许路径、禁止行为、测试命令、报告要求
新增设计标准说明：single/parallel/sequential 与 internal ordered_parallel 的关系
新增 forbidden path 检查脚本
新增审核证据收集脚本
审核 skill 更细，能直接判定 APPROVED / CHANGES REQUESTED / REJECTED
```

## 安装

推荐在项目根目录执行：

```bash
unzip agenthub-llm-orchestration-cc-execution-pack-v2.zip
cd agenthub-llm-orchestration-cc-execution-pack-v2
./scripts/install-cc-execution-pack.sh new-only
```

`new-only` 只安装新增两个 skill 和 docs，不覆盖你已有 skill。

如需同步全部 skill 快照：

```bash
./scripts/install-cc-execution-pack.sh all-skills
```

## CC 执行

在 Claude Code 中输入：

```text
/llm-orchestration-dev

按 docs/plans/llm-orchestration-development-plan.md 执行。
从 Phase 0 开始。
每个 Phase 完成后停止并输出报告，不要自动进入下一 Phase。
```

## 审核

开发完成后运行：

```bash
./scripts/collect-review-evidence.sh
```

然后把：

```text
llm-orchestration-review-evidence.txt
Phase Reports
go test 输出
```

交给审核方，使用：

```text
/llm-orchestration-review
```
