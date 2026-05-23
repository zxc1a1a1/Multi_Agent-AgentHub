# Prompt 模板规则

Prompt 模板必须可审查。

正式开发阶段建议支持 prompt version。

Prompt 可以来自：

- Agent handler 内部模板。
- prompt template 文件。
- prompt registry。
- config 中的 template name。

建议集中管理：

```text
prompts/{agent}/{name}.md
```

Prompt 不得包含：

- API key。
- access token。
- refresh token。
- 数据库连接字符串。
- 对象存储私有地址。
- system prompt secret。
- 用户无权访问的文件内容。
- 未脱敏日志。
- 内部服务地址。

如果 prompt 用于 structured output，必须引用对应 schema 名称和用途，但 schema validation 必须在代码层执行。
