# R0-07 S3 兼容对象存储运行认证

- task: R0-07（`docs/plans/v3-gap-atomic-plan.md` §4.1）
- 对应缺口：GAP-06；F-12
- commit baseline: `0f7a210`（dev，R0-06 之后）
- 环境：Windows 10.0.19045；Docker 29.5.2
- 运行时：S3 兼容网关 **SeaweedFS 4.50**（镜像 `chrislusf/seaweedfs@sha256:c2658820e763...`，见 logs/image-digest.txt）
- overall: **PASS**（两阶段：正常跨实例 + 停机失败）

## 0. MinIO 替代说明（重要）

计划要求 MinIO，但本环境无法从 Docker Hub/Quay 拉取任何 MinIO 镜像（`minio/minio:latest` 与固定 tag 均 `not found`，Quay 返回 401）。
因此使用 **S3 API 兼容**的 SeaweedFS S3 网关做运行认证；契约面（bucket、Put/Get、Presign、List、Delete、10MB、key 安全）
与 MinIO 相同。compose 新增 `objectstore` profile 的 `s3-objectstore` 服务（固定 digest、healthy、端口 9000）。
这不是“MinIO 已认证”，而是“S3 兼容对象存储已认证”。

## 命名说明

测试函数为 `TestS3IntegrationCrossInstanceArtifact`（不以 `TestIntegration` 开头），
因此 R0-01 的通用 `/^TestIntegration/` 门禁不会把它当作集成用例；S3 需要可选的
`AGENTHUB_S3_TEST_ENDPOINT`，由 R0-07 单独启动对象存储后执行。

## 1. 结果

| 阶段 | 测试 | 结果 |
|---|---|---|
| 正常 | TestS3IntegrationCrossInstanceArtifact | PASS（0.28s） |
| 停机 | 同一测试 `AGENTHUB_S3_TEST_EXPECT_DOWN=1` | PASS（读取明确失败） |
| compose profile | `s3-objectstore` | Up (healthy)，`GET / -> 200` |

## 2. 覆盖

- 两个独立 `S3ObjectStore` 实例（模拟两个 Orchestrator）共享同一 bucket；
- A 写入 → B 读取，字节一致，SHA-256 一致；
- B 生成 Presign URL，HTTP GET 返回 200 且内容一致；8 天 TTL 被拒绝；
- 元数据（`S3Store`）由 A 写、B 读，字段一致；
- key 安全：`../escape`、`/etc/passwd`、`a\b`、空 key 全部拒绝；
- 10MB+1 在到达网络前被拒绝；
- 停机后读取明确失败，不存在静默回落本地卷的代码路径。

## 3. 未认证 / 边界

- **HTTP `/internal/orchestrator/artifacts/{id}/content` 的跨进程读取未在本任务单独起两个 Orchestrator 进程验证**；
  本次验证到 storage/metadata 层。handler 的 StorageRef/checksum 逻辑由既有 httpapi 单测覆盖。
- “错误不泄露 endpoint/凭据”未做专门断言；S3 客户端错误文本未纳入证据。
- MinIO 本体不可得，见 §0。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| Artifact 生成/读取 | events.jsonl | 2（正常/停机） | yes（storage 层） |
| API 请求/响应 | - | 0 | not_applicable: 本测试直接调用存储接口 |
| 用户操作/SSE/HITL/记忆 | - | 0 | not_applicable: 无浏览器/无 run |
| 多实例 | events.jsonl | 2 | yes（两个 store 实例） |
| MQ | - | 0 | not_applicable: R2 未接线 |
| 故障注入 | events.jsonl | 1 | yes（停机读取失败） |

## 5. 回滚

- 关闭 `objectstore` profile，清空 `ARTIFACT_S3_ENDPOINT` 即回退本地卷；默认启动路径不变。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R0-07/`（commands.txt、events.jsonl、summary.md、logs/）
