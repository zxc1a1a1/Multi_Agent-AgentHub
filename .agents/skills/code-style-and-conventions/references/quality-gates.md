# 质量门禁

## 原则

优先执行仓库已有脚本。

如果命令不存在，不得声称已执行或已通过。

## Go

建议命令：

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

如果项目配置了 lint：

```bash
golangci-lint run
```

## 前端

建议命令：

```bash
npm run lint
npm run typecheck
npm run test
npm run build
```

如果项目使用 pnpm / yarn，应按实际 lockfile 和脚本执行。

## 项目级

建议命令：

```bash
docker compose config
./smoke-test.sh
```

## 文档任务

文档或 Skill 文件包至少检查：

- 路径是否正确。
- frontmatter 是否合法。
- 是否中文。
- Markdown 标题层级是否合理。
- 代码块语言是否标注。
- 是否没有未授权生成业务代码。
- MANIFEST 是否列出全部文件。
- PATCH_NOTES 是否说明替换和新增内容。

## 报告格式

输出结果时应说明：

```text
已执行：...
未执行：...
未执行原因：...
建议本地执行：...
```
