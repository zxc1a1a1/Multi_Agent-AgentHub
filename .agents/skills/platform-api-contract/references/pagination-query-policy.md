# 分页、排序与搜索策略

所有列表接口必须有分页策略。

默认分页参数：`page`、`pageSize`。

可扩展分页参数：`cursor`、`limit`。

排序和搜索参数：`sortBy`、`sortOrder`、`keyword`。

`pageSize` 必须有最大值，大型列表不得无分页返回。
