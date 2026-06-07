# 00 执行协议：防止 CC 乱操作

本协议优先级高于普通开发建议。执行 agent 必须遵守。

## 1. Phase Gate 规则

本任务分为 Phase 0 到 Phase 5。每个 Phase 都必须：

```text
先读要求
只做本 Phase 允许的事
跑本 Phase 指定测试
输出 Phase Report
停下来等待用户确认
```

禁止：

```text
未经确认自动进入下一 Phase
顺手修无关问题
因为看到 TODO 就扩展范围
因为测试失败就改 forbidden path
因为 skill 提到某模块就修改该模块
```

## 2. 最小改动原则

每次只做能满足本 Phase 验收的最小代码变更。

禁止：

```text
大规模重构
重命名无关文件
格式化全仓库
调整 import 顺序导致大量无关 diff
修改前端/Gateway/docker 来“顺手适配”
```

## 3. 证据优先原则

每个 Phase 完成必须给出证据：

```text
git diff --name-only
修改文件说明
测试命令
测试结果
是否触碰 forbidden paths
下一步建议
```

没有测试结果，不算完成。

## 4. 失败处理原则

测试失败时：

```text
先判断失败是否属于本 Phase 范围
只修本 Phase 范围内的问题
不能为了让测试过而改 forbidden paths
不能跳过测试
不能隐藏失败
```

如果失败来自无关模块，例如全仓库编译受 `SQLSessionService.GetOrCreate` 阻塞，本任务只记录，不处理。

## 5. 输出约束

所有报告必须使用模板：

```text
references/report-template.md
```

最终交付必须提供：

```text
git diff --stat
git diff --name-only
go test 输出
Phase Report
风险清单
```
