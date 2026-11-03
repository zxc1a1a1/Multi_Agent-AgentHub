# R3-06 Sandpack 评估（NO-GO）

- task: R3-06
- 对应缺口：GAP-11；UI-03
- 类型：**评估报告，无生产变更**
- 结论：**NO-GO**

## 证据

| 维度 | 结果 |
|---|---|
| 许可 | Apache-2.0，通过 |
| 包体 | 976.0KB min / 280,734 B gzip（React external） |
| 沙箱 | runtime iframe 含 `allow-same-origin` + `allow-scripts` + downloads/popups → 违反 R3-02 策略 |
| 离线 | 默认 bundler 指向 codesandbox.io；自托管需额外部署 |
| 维护 | 最后发布 2025-04-29 |

完整报告：`docs/reports/r3-06-sandpack-evaluation.md`。

## 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 评估证据 | events.jsonl | 4 | yes |
| UI/SSE/HITL/记忆/MQ | - | 0 | not_applicable: 无生产接入 |

## 回滚

- 无生产变更。
