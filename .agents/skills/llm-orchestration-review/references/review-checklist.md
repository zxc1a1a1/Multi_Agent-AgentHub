# 审核清单

## 1. 路径边界

运行：

```bash
git diff --name-only
```

不允许出现：

```text
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**
```

## 2. 主路径检查

必须满足：

```text
LLMPlanner 是主路径
有效 LLM 输出不调用 RulePlanner
LLM 输出必须先 parse
LLM 输出必须 normalize
LLM 输出必须 validate
Executor 只执行已校验 plan
```

## 3. Parser 检查

必须支持：

```text
plain JSON
json fence
leading/trailing whitespace
extract first object from short text
```

必须拒绝：

```text
empty
non-JSON
array root
missing mode
missing steps
```

## 4. Normalizer 检查

必须满足：

```text
single -> internal single
parallel -> internal ordered_parallel
sequential 当前拒绝或 unsupported
不猜 unknown agent
不删除非法 step
不 fallback
```

## 5. Validator 检查

必须确定性校验：

```text
unknown agent
empty steps
single multiple steps
parallel depends_on
unsupported sequential
empty input
confidence range
internal URL
token/API key
DB DSN
system prompt leakage
```

## 6. Repairer 检查

必须满足：

```text
repair 最多一次
repair prompt 包含 raw output、errors、agents、schema
repair success 不 fallback
repair fail 当前 fallback RulePlanner
repair fail 记录 fallbackReason
```

## 7. RulePlanner 检查

必须满足：

```text
有 Deprecated 注释
没有新增关键词
没有新增路由规则
不是主路径
只在 transitional fallback 使用
```

## 8. 前端兼容检查

必须满足：

```text
/api/chat 未改
services/gateway 未改
event type 未改
SSE shape 未改
tool call 顺序未改
text message 顺序未改
只追加可选 metadata
metadata 不包含 raw prompt/secret/internal URL
```

## 9. 测试证据

必须看到：

```text
code-agent single
web-agent single
full-stack parallel
invalid JSON repair
unknown agent repair
repair fail fallback
sequential rejected
parallel depends_on rejected
```

## 10. 审核结论格式

输出：

```text
APPROVED
APPROVED WITH FOLLOW-UP
CHANGES REQUESTED
REJECTED
```

并列出：

```text
Blocking issues
Non-blocking issues
Evidence checked
Required fixes
Next phase suggestion
```
