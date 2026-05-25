# 允许修改文件策略

## 1. 默认允许

在用户要求输出某个 Skill 文件包时，默认允许生成：

```text
.claude/skills/{skill-name}/SKILL.md
.claude/skills/{skill-name}/references/*.md
docs/contracts/{related-contract}.md
docs/contracts/{related-review-checklist}.md
PATCH_NOTES.md
MANIFEST.md
```

## 2. 默认不允许

除非用户明确要求，不生成或修改：

```text
server/
frontend/
agents/
docker-compose.yml
Dockerfile
init.sql
Makefile
package.json
go.mod
.github/
```

## 3. 文件包要求

文件包必须使用仓库相对路径。

不得把文件直接放在用户本地仓库路径里。

不得包含：

- `.DS_Store`
- `node_modules`
- `.git`
- 构建产物
- 临时缓存
- API key
- 私密配置

## 4. 覆盖提醒

如果生成的文件用于覆盖已有文件，应在 `PATCH_NOTES.md` 中明确：

- 哪些文件是替换；
- 哪些文件是新增；
- 哪些文件是说明文件；
- 是否包含业务代码。
