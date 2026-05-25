# Public API Auth Policy

## 范围

适用于 Frontend → Gateway 的公开 Platform API。

## 规则

- 公开 API 默认需要用户鉴权。
- 访问令牌只能放在 Authorization header。
- 禁止把 token 放入 query string、Artifact、日志、错误、debug dump。
- 所有 conversation、message、artifact、run、agent config 必须对象级授权。
- 公开 API 不得暴露 Orchestrator 内部路径、Child Agent endpoint、service token。
- 返回数据必须字段裁剪。

## 错误处理

- 未鉴权返回 401。
- 无权限返回 403 或安全 404。
- 不得通过错误文案泄露他人资源是否存在。
