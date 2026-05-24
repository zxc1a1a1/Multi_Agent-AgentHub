# Sandbox Policy

来源：`docs/contracts/sandbox-policy.md`。

- MVP 不默认开放 `run_command` / `file_upload`。
- Post-MVP 若开放，需白名单、超时、路径限制与确认机制。
- 高危操作必须 `confirm_action`。
- 预览内容需与主应用隔离。
