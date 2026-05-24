# Go Style

来源：`docs/contracts/code-style.md`、`docs/contracts/conventions.md`、`docs/skill/code-style-and-conventions/SKILL.md`。

- 使用 `gofmt`。
- 包名小写，文件名小写加下划线。
- 显式处理 error，向下传递 context。
- handler 保持薄层，不塞复杂编排。
- 正式代码至少 `go test` 通过。
