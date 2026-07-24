# 多模态预览策略

- 前端可渲染 `file_upload`、`image_preview`、`file_card`、`vision_analysis_card`、`slide_deck_preview`。
- 前端不执行 Agent 输出中的任意脚本。
- `webpage` 预览必须使用沙箱策略（如 iframe sandbox）隔离执行环境。
- 文件下载与预览必须带权限控制，禁止匿名长期公开访问。
