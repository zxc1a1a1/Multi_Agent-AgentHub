# Tool Use Adapter Policy

Provider 可能支持 tool use / function calling，但 Provider Adapter 不执行业务工具。

## 规则

- Provider tool call 必须归一化为内部 tool intent。
- Provider Adapter 不直接调用外部工具。
- Provider Adapter 不直接写数据库。
- Tool 参数必须 schema validation。
- 不支持 tool use 的模型不得处理 tool-use 请求。
- 工具执行属于调用方业务层，不属于 Provider Adapter。
