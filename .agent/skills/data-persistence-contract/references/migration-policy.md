# Migration 规则

## 目标

所有数据结构变化必须可追溯、可 review、可回滚或可解释。

## 命名

推荐：

```text
migrations/0001_init.sql
migrations/0002_add_conversation_participants.sql
migrations/0003_add_agent_registry.sql
```

或使用项目现有 migration 工具约定。

## expand / migrate / contract

### expand

添加兼容字段、表、索引。

示例：

```sql
ALTER TABLE conversations ADD COLUMN conversation_type VARCHAR(32) NOT NULL DEFAULT 'single';
```

### migrate

回填数据、双写、切换读取路径。

示例：

```sql
UPDATE conversations SET conversation_type = 'single' WHERE conversation_type IS NULL;
```

### contract

确认新路径稳定后删除旧字段或旧逻辑。

## 破坏性变更

以下变更必须分阶段：

- 删除列。
- 重命名列。
- 改字段类型。
- 改 nullable。
- 改主键。
- 改唯一约束。
- 大规模拆表。

## 禁止

- 直接手工改库。
- 只改 `init.sql` 不写 migration。
- 生产数据表上直接执行长时间锁表操作。
- 没有备份策略就删除字段。
- migration 中写入 secret。

## Review 要求

每个 migration 必须说明：

- 为什么改。
- 影响哪些表。
- 是否兼容旧代码。
- 是否需要回填。
- 是否需要索引。
- 是否有数据安全风险。
