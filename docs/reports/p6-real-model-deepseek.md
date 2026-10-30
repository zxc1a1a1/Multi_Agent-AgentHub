# P6 真模型浏览器验收：DeepSeek（部分）

- 日期：2026-10-30
- 模型：用户指定的 DeepSeek 4.1 Flash；实际 API 接受 `deepseek-chat` 别名
- Provider：`openai` 兼容协议，`https://api.deepseek.com/v1`
- 密钥：来自用户已配置的 `.env`，通过 Compose 注入；未读取、未打印、未写入仓库

## 1. 配置

```text
LLM_PROVIDER=openai
LLM_MODEL=deepseek-chat
LLM_BASE_URL=https://api.deepseek.com/v1
ORCHESTRATOR_LLM_PROVIDER=openai
ORCHESTRATOR_LLM_MODEL=deepseek-chat
ORCHESTRATOR_LLM_BASE_URL=https://api.deepseek.com/v1
docker compose --env-file .env -f docker-compose.new-arch.yml up -d ...
```

说明：最初使用 `deepseek-4.1-flash`，Orchestrator 返回 `ORCHESTRATOR_PLANNER_FAILED`；改用该服务接受的 `deepseek-chat` 后规划与子 Agent 均成功。具体模型 ID 以供应商账号可用列表为准。

## 2. 浏览器主路径（真实模型）

环境：compose 最小栈（MySQL/Redis/code-agent/web-agent/Orchestrator/Gateway）+ Vite。

```text
1. 新建会话，发送“用 Go 写一个判断整数是否为偶数的函数，只输出代码。”
2. MainAgent 通过 DeepSeek 生成计划卡（code-agent / 1 task），页面显示 Approve & Execute
3. 点击批准 → code-agent 通过 DeepSeek 返回 Go 代码（func isEven）
4. MySQL：messages 存在 sender_type=assistant、sender_name=code-agent、run_id=run_59d3cdd4cfa17393 的记录
5. 刷新已完成的会话：历史从 MySQL 恢复，显示 Code Agent 与返回的代码（无活跃游标时直接读历史）
```

## 3. 刷新恢复 + 批准（真实模型）

```text
1. 新会话发送“真实模型刷新测试：用 Go 写一个判断字符串是否为空的函数，只输出代码。”
2. 计划卡出现后直接刷新页面
3. 页面自动恢复该会话、计划卡与 (runId,seq,subIndex) 游标
4. 点击批准 → 前端跟踪 run 至终态并刷新历史
5. 页面显示 code-agent 输出（func isEmpty）
6. MySQL：assistant 消息 sender_name=code-agent、run_id=run_f1d7fbe6b6dd299b
```

## 4. 本轮修复

真实 gRPC/AG-UI 路径的事件类型是 `AGENT_TURN_CONTENT`，Gateway 原先只收集 `TEXT_MESSAGE_CONTENT/message/message.delta`，导致：
- 浏览器能看到实时文本；
- MySQL 没有 assistant 消息，刷新后历史丢失。

修复：
- 收集 `AGENT_TURN_CONTENT`；
- 作者缺失时从事件 metadata 的 senderName 回退；
- 新增 `TestChatPersistsAssistantTextWithoutRole`（使用 `agent_turn_content`、无 role），旧实现会失败。

## 5. 未覆盖

- 仅单轮代码生成场景；未做多 Agent、工具调用、长上下文、并发/负载。
- 未验证 DeepSeek 4.1 Flash 的官方精确 model id；本报告只声明 `deepseek-chat` 别名成功。
- 未计费/未统计 token 成本；未做失败重试/限流矩阵。
