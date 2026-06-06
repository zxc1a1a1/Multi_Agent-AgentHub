# Phase 0：只读侦察，不改代码

## 目标

确认当前 `services/orchestrator` 的真实结构、已有 Planner、已有策略、已有 validator、已有 HTTP event 输出点。

## 严格规则

```text
本 Phase 禁止修改任何文件。
本 Phase 只读。
本 Phase 结束必须停下来汇报。
```

## 必须执行命令

从仓库根目录执行：

```bash
pwd
git status --short
find services/orchestrator -maxdepth 4 -type f | sort
grep -R "type .*Planner" -n services/orchestrator || true
grep -R "RulePlanner\|LLMPlanner\|ordered_parallel\|sequential" -n services/orchestrator || true
grep -R "run_started\|state_update\|message_delta\|run_error" -n services/orchestrator || true
grep -R "type .*Validation\|func .*Validate" -n services/orchestrator || true
cd services/orchestrator && go list ./...
```

## 必须阅读文件

```text
services/orchestrator/planner/planner.go
services/orchestrator/planner/llm_planner.go
services/orchestrator/planner/planner_llm.go
services/orchestrator/planner/rule_planner.go
services/orchestrator/plan/types.go
services/orchestrator/validator/validator.go
services/orchestrator/httpapi/handler_run_stream.go
services/orchestrator/cmd/orchestrator/main.go
services/orchestrator/registry/static_registry.go
```

如果某文件不存在，记录为“不存在”，不要创建。

## 输出报告必须包含

```text
1. 当前 planner 文件清单
2. 当前 Planner 接口/类型
3. 当前 LLMPlanner 是否已存在
4. 当前 RulePlanner fallback 是否已存在
5. 当前 plan strategy 常量
6. 当前 executor/httpapi 支持 single / ordered_parallel / sequential 的情况
7. 当前 validator 已校验规则
8. 当前 run_started/state_update 输出位置
9. git status --short 结果
10. 确认没有修改文件
```

## 停止条件

完成后必须停止，不进入 Phase 1，等待用户确认。
