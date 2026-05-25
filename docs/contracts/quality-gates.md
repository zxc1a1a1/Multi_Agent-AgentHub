# AgentHub 质量门禁

## 1. 原则

优先执行仓库已有命令。命令不存在时，不得声称已通过。

## 2. Go 建议命令

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

如果项目配置了 lint：

```bash
golangci-lint run
```

## 3. 前端建议命令

```bash
npm run lint
npm run typecheck
npm run test
npm run build
```

## 4. 项目级建议命令

```bash
docker compose config
./smoke-test.sh
```

## 5. 文档 / Skill 包检查

- 路径正确。
- 内容中文。
- frontmatter 合法。
- Markdown 标题层级合理。
- 代码块标注语言。
- MANIFEST 完整。
- PATCH_NOTES 说明变更。

## 6. 输出报告

报告必须包含：

```text
已执行：...
未执行：...
未执行原因：...
建议本地执行：...
```
