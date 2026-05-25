# Trust Boundary Model

## 目的

定义 AgentHub 中哪些主体、输入和输出默认不可信，以及在执行前必须经过哪些校验。

## 默认不可信对象

- 用户输入
- Frontend 请求字段
- LLM 输出
- Planner 输出
- Child Agent 输出
- 用户自建 Agent
- AgentCard 声明
- Artifact 内容
- Tool Call 参数
- 上传文件
- 外部 URL

## 核心规则

- 不可信输入不得直接执行。
- 不可信内容不得直接渲染到主应用 DOM。
- 不可信计划不得直接调度 Agent。
- 不可信 Tool 参数必须 schema validation。
- 高危动作必须 confirm_action。
- 所有用户资源必须对象级授权。

## 评审问题

- 这个输入来自哪里？
- 是否经过 schema validation？
- 是否经过权限校验？
- 是否会触发外部副作用？
- 是否需要用户确认？
- 是否已记录审计日志？
