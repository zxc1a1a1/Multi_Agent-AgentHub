# Phase Report 模板

## Phase

```text
Phase:
Status: PASS / FAIL / PARTIAL
是否停止等待确认：是 / 否
```

## 修改文件

```text
- 
```

## 禁止路径确认

请逐项确认：

```text
frontend/**：未修改 / 已修改
services/gateway/**：未修改 / 已修改
docker-compose*：未修改 / 已修改
pkg/adk/**：未修改 / 已修改
pkg/runtime/agui/**：未修改 / 已修改
server/**：未修改 / 已修改
agents/**：未修改 / 已修改
```

如有已修改，必须说明原因并建议 revert。

## 本 Phase 实现内容

```text
1.
2.
3.
```

## 执行命令

```bash

```

## 测试结果

```text
PASS / FAIL / NOT RUN
```

关键输出：

```text

```

## 行为证据

按本 Phase 填写：

```text
Parser：
Normalizer：
Validator：
LLMPlanner：
Repairer：
RulePlanner fallback：
State metadata：
```

## git diff 检查

```bash
git diff --name-only
```

输出：

```text

```

## 风险

```text
-
```

## 下一步建议

```text
继续下一 Phase / 停止等待审核 / 需要返工
```
