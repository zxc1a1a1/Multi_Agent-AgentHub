# Commit Message 规范

## 格式

使用 Conventional Commits 风格：

```text
<type>(<scope>): <summary>
```

## type

```text
feat      新功能
fix       修复问题
docs      文档变更
refactor  重构，不改变行为
test      测试
chore     工程配置或杂项
style     格式、命名，不改变行为
ci        CI/CD
build     构建系统或依赖
```

## scope

scope 应描述影响区域：

```text
server
frontend
agents
skills
docs
contracts
docker
tests
```

## 示例

```text
feat(agents): support generic child agent config
fix(frontend): parse SSE blocks by blank line
docs(skills): rewrite code style conventions in Chinese
refactor(server): extract reusable HTTP client
test(orchestrator): cover fallback planning path
```

## 禁止

```text
update
fix bug
wip
final
misc
随便改一下
```

## AI 生成提交说明

AI 生成提交说明必须：

- 准确描述实际变更。
- 不夸大完成功能。
- 不声明未执行的测试通过。
- 不把多个无关改动混成一个 summary。
