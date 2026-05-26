# 多模态编排路由补充

- 多模态路由应同时考虑用户意图与输入类型，不能只看文本关键词。
- “看图生成网页”推荐 `vision-agent -> web-agent`。
- “截图报错分析”推荐 `vision-agent -> test-agent/review-agent`。
- “资料生成 PPT”推荐 `file-agent/vision-agent -> ppt-agent`。
- “检查截图泄密”推荐 `vision-agent -> security-agent`。
