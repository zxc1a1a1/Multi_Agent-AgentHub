# Artifact / Runtime Security

## 风险分级

| 等级 | 示例 | 要求 |
|---|---|---|
| 低 | markdown_render、只读 code_preview | 防 XSS、只读展示 |
| 中 | file_download、document preview | 鉴权、对象级授权、短期 URL |
| 高 | web_preview、run_command、deploy、file_upload | sandbox、权限、confirm_action、审计 |

## 规则

- Artifact 内容不可信。
- HTML 预览必须 iframe sandbox。
- Markdown 不默认允许危险 HTML。
- Code preview 默认不执行。
- 大 Artifact 通过 contentRef 或受控下载。
