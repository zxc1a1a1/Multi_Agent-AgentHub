# A2A Artifact 输出补充

- 子 Agent 输出应通过 A2A Artifact 表达。
- Artifact 可包含 `text/code/json/html/markdown/diff` 或 `contentRef`。
- 大文件、图片、PPTX、PDF 应使用 `contentRef`。
- 子 Agent 不应伪造本地文件路径。
- `outputModes` 应和主要 artifact type 对齐。
