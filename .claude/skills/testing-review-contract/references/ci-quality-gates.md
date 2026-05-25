# CI Quality Gates

## 必过 Gates

- lint / format。
- backend unit。
- frontend unit。
- schema validation。
- contract tests。
- security regression。
- minimal smoke。

## 条件必过 Gates

- docker compose smoke。
- demo E2E。
- migration check。
- coverage threshold。

## 禁止

测试失败仍合并、普通 CI 使用真实 LLM key、跳过 contract test、CI 输出 secret。
