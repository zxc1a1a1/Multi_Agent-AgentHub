# 02 设计标准：高质量 LLM 编排做法

本项目本阶段采用通用高质量工程模式：

```text
LLM structured plan
+ Agent registry / AgentCard capability input
+ Parser
+ Normalizer
+ deterministic Validator
+ one-shot Repairer
+ controlled Executor
+ Trace / metadata
```

## 1. LLM 不直接执行

错误做法：

```text
LLM 返回 plan → Executor 执行
```

正确做法：

```text
LLM 返回 raw output
→ Parser 解析
→ Normalizer 转内部 plan
→ Validator 校验
→ Repairer 修复一次
→ Executor 执行已校验 plan
```

## 2. LLM-facing schema 使用通用模式

LLM 输出层使用通用编排模式：

```text
single
parallel
sequential
```

含义：

```text
single：一个 Agent 完成
parallel：多个 Agent 独立完成
sequential：多个 Agent 按依赖顺序完成
```

## 3. 当前内部兼容映射

当前 AgentHub 内部可能使用已有 strategy：

```text
single
ordered_parallel
```

本阶段 normalizer 做兼容映射：

```text
LLM mode=single    → internal StrategySingle
LLM mode=parallel  → internal StrategyOrderedParallel
LLM mode=sequential → 当前拒绝，除非 executor/httpapi 明确完整支持
```

## 4. 为什么暂时拒绝 sequential

只有满足以下条件，才允许启用 sequential：

```text
Executor 能按 depends_on 顺序执行
后一步能拿到前一步输出
SSE/state_update 能报告每一步状态
失败时能停止后续依赖步骤
Validator 能检查 depends_on 合法性和环
测试覆盖 sequential 成功和失败
```

如果这些不完整，validator 必须拒绝 sequential 并触发 repair。

## 5. RulePlanner 定位

RulePlanner 只允许作为临时兜底：

```text
Deprecated transitional fallback only
```

禁止：

```text
新增关键词
新增 agent 匹配规则
扩大 RulePlanner 职责
把 RulePlanner 当主路径
```

后续目标是删除 RulePlanner。
