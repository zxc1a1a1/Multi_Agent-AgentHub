# sequential 依赖规则

`sequential` 用于后续任务依赖前置任务输出的场景。

## 规则

- `dependsOn` 必须引用同计划内已有 taskId。
- `dependsOn` 不得成环。
- 前置任务失败后，不得盲目执行依赖任务。
- 依赖任务使用前置输出时必须使用摘要或受控引用。
- sequential 不得被 LLM 随意生成，必须通过校验。
