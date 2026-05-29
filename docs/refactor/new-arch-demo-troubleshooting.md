# New Architecture Demo Troubleshooting

## 1. Docker daemon 未启动
症状：`docker version` 失败，或提示无法连接 daemon。  
处理：启动 Docker Desktop 后重试。

## 2. dockerDesktopLinuxEngine pipe missing
症状：Windows 上报错包含 `dockerDesktopLinuxEngine`。  
处理：重启 Docker Desktop；若仍失败，重启系统后先执行 `docker version` 再启动 compose。

## 3. docker compose config 失败
症状：`docker compose -f docker-compose.new-arch.yml config` 报错。  
处理：先修复 YAML 语法、缩进或字段拼写，再重试。

## 4. build 失败
症状：`docker compose ... build` 失败。  
处理：
- 先看详细日志。
- 检查网络与镜像拉取速度。
- 检查 Docker daemon 是否稳定。

## 5. 端口 8080/8081/8082/3000 被占用
症状：compose up 时端口绑定失败。  
处理：释放冲突端口，或修改端口映射后同步更新 smoke 参数。

## 6. code-agent health fail
症状：`http://localhost:8081/health` 无法返回 `status=ok`。  
处理：
- 查看 `code-agent-new` 容器日志。
- 确认容器已启动且未频繁重启。

## 7. web-agent health fail
症状：`http://localhost:8082/health` 检查失败。  
处理：
- 查看 `web-agent-new` 日志。
- 检查镜像是否构建成功。

## 8. gateway health fail
症状：`http://localhost:8080/health` 不可用。  
处理：
- 查看 `gateway-new` 日志。
- 确认 `AGENT_CODE_URL` / `AGENT_WEB_URL` 指向容器内 service name。

## 9. /api/agents 返回空
症状：前端 Agent 列表为空或接口返回异常。  
处理：确认 gateway 启动配置是否正确，检查静态注册表构建是否成功。

## 10. /api/chat SSE 无输出
症状：请求建立后无 `event: message` 或 `event: error`。  
处理：
- 先检查 gateway 和子 agent 健康。
- 用 smoke 脚本复现并看原始响应。

## 11. unknown agent error 不符合预期
症状：unknown agent 错误包含内部信息。  
处理：检查错误脱敏路径，确保不暴露 token/stack/internal URL。

## 12. frontend 无法访问 /api
症状：前端页面打开正常，但接口请求失败。  
处理：确认 `frontend/nginx.conf` 中 `/api` 代理目标为 `gateway-new:8080`。

## 13. nginx SSE buffering 问题
症状：SSE 延迟大或几乎不推送。  
处理：确认 nginx `/api` location 已设置：
- `proxy_buffering off;`
- `proxy_cache off;`
- `proxy_http_version 1.1;`

## 14. PowerShell 执行策略问题（npm.ps1）
症状：执行 `npm test`/`npm run build` 提示 `npm.ps1` 被策略阻止。  
处理：使用 `npm.cmd`：
```powershell
npm.cmd test -- --run
npm.cmd run build
```

## 15. sh 不存在
症状：Windows 环境执行 `sh ./doctor-new-arch.sh` 或 `sh ./smoke-new-arch.sh` 失败。  
处理：改用 PowerShell 脚本，或安装 Git Bash/WSL。

## 16. npm build chunk size warning 解释
症状：frontend build 出现 chunk size warning。  
处理：这是体积提示，不是构建失败；当前 Demo 可接受。

## 17. doctor 如何使用
```powershell
powershell -ExecutionPolicy Bypass -File ./doctor-new-arch.ps1
```
说明：
- doctor 不会启动服务。
- 服务未启动时 health 检查给 WARN，不视为脚本失败。
- 关键文件缺失或 compose config 失败会返回失败。

## 18. smoke 在服务未启动时 exit 1
说明：
- 若未执行 compose up，或 Docker daemon 不可用且服务未手动启动，smoke `exit 1` 是预期行为。
- 这不代表 smoke 脚本逻辑错误。

## 19. compose config 通过但 build/up 失败
说明：
- `compose config` 只验证配置可解析。
- `build/up` 还依赖 daemon、镜像拉取、端口和本机环境。

## 20. Docker daemon 不可用不是代码失败
说明：
- 常见是本机 Docker Desktop/引擎未运行。
- 先修复环境，再复验 compose build/up/smoke。

## 21. 如何收集日志
```powershell
docker compose -f docker-compose.new-arch.yml logs -f
```

## 22. 如何清理
```powershell
docker compose -f docker-compose.new-arch.yml down
```
