# Volume 与 Network 策略

## Volume

- MySQL 使用 named volume。
- 默认启动不删除数据。
- reset 命令可以删除数据，但必须明确危险。
- 默认配置不得使用本机绝对路径。
- dev profile 可以使用 bind mount。
- secret 文件不得作为 repo-tracked volume 挂载。

## Network

- 默认使用 Compose project network。
- 服务间通信使用 service name。
- 容器间不得使用 localhost 调其他服务。
- 只有需要宿主机访问的服务才映射 ports。
- Agent 服务优先 expose 给内部网络。
- 网络别名必须有明确理由。

## 常见错误

错误：

```text
gateway 容器访问 http://localhost:8081
```

正确：

```text
gateway 容器访问 http://agent-service-name:8081
```
