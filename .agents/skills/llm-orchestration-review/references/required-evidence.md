# 审核所需证据

执行 agent 必须提供：

```text
git diff --stat
git diff --name-only
Phase Reports
go test 输出
修改文件说明
forbidden path 确认
```

必须提供行为证据：

```text
valid LLM plan 不走 RulePlanner
invalid JSON repair success
unknown agent repair success
repair fail fallback RulePlanner
RulePlanner 未新增关键词
sequential 当前被拒绝
parallel depends_on 被拒绝
state metadata 只追加
```

如缺少证据，结论应为 CHANGES REQUESTED。
