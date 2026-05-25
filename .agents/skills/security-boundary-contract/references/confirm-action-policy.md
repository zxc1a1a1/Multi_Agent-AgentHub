# confirm_action Policy

## 目的

为高危动作提供统一用户确认安全门。

## 规则

- 用户确认前不得执行动作。
- 确认内容必须包含动作类型、目标资源、影响范围、风险等级。
- 确认不能只依赖 LLM 自然语言。
- 确认后仍需最终权限校验。
- 超时或取消视为拒绝。
- 确认记录必须审计。

## 必须确认的动作

- run_command
- deploy
- file_overwrite
- external_publish
- credential_update
- workspace_delete
