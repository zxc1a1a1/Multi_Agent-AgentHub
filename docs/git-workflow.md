# Git 团队协作流程

## 分支设计

本项目保留 `feature/xxx` 功能分支，同时增加 3 个长期成员分支：`c`、`y`、`g`。

```text
master/main：稳定分支，只放可发布版本，不直接开发
dev：团队开发集成分支，所有成员分支最终合并到这里
c：成员 c 的长期工作分支
y：成员 y 的长期工作分支
g：成员 g 的长期工作分支
feature/<成员>/<任务>：具体功能分支，短期存在
bugfix/<成员>/<问题>：修复分支，短期存在
hotfix/<问题>：紧急修复分支
```

推荐结构：

```text
master
  ↑
dev
  ↑
├── c
│   └── feature/c/adk-base
├── y
│   └── feature/y/frontend-layout
└── g
    └── feature/g/a2a-server
```

## 合并流程

推荐主流程：

```text
feature/<成员>/<任务> → 成员分支 c/y/g → dev → master/main
```

例如：

```text
feature/c/adk-base → c → dev → master
feature/y/web-console → y → dev → master
feature/g/a2a-server → g → dev → master
```

这样做的好处：

- `master` 始终稳定。
- `dev` 是团队统一集成分支。
- `c/y/g` 可以承载每个人未完全完成的阶段性代码。
- `feature/xxx` 仍然用来隔离具体任务，避免一个人的多个任务混在一起。

## 负责人初始化远程仓库

负责人创建远程空仓库后，在项目根目录执行：

```bash
bash scripts/init-repo.sh <remote-url>
```

脚本会创建并推送这些分支：

```text
master
dev
c
y
g
```

建议远程仓库设置：

```text
master：保护分支，只能 PR/MR 合并
dev：保护分支，只能 PR/MR 合并
c/y/g：对应成员维护，可以允许对应成员 push，也可以要求 PR/MR
```

## 成员第一次拉取项目

```bash
git clone <remote-url>
cd multi-agent-framework-go-react
```

成员 c：

```bash
git checkout c
git pull origin c
```

成员 y：

```bash
git checkout y
git pull origin y
```

成员 g：

```bash
git checkout g
git pull origin g
```

## 成员开发新功能

推荐从自己的成员分支创建功能分支。

成员 c 开发 ADK 基础功能：

```bash
bash scripts/create-branch.sh feature/c/adk-base c
```

成员 y 开发前端布局：

```bash
bash scripts/create-branch.sh feature/y/frontend-layout y
```

成员 g 开发 A2A 服务：

```bash
bash scripts/create-branch.sh feature/g/a2a-server g
```

开发完成后：

```bash
git add .
git commit -m "feat: describe your change"
git push -u origin feature/c/adk-base
```

然后在 GitHub / GitLab / Gitee 上提交 PR/MR：

```text
feature/c/adk-base → c
```

成员分支阶段性稳定后，再提交 PR/MR：

```text
c → dev
```

## 同步 dev 到成员分支

当 `dev` 有别人合入的新代码时，成员分支要定期同步，避免冲突越来越大。

```bash
bash scripts/sync-member-branch.sh c
bash scripts/sync-member-branch.sh y
bash scripts/sync-member-branch.sh g
```

脚本会执行：

```text
dev → 成员分支
```

## 版本发布

当 `dev` 测试通过后，由负责人发起 PR/MR：

```text
dev → master/main
```

## commit 命名建议

```text
feat: 新功能
fix: 修复 bug
docs: 文档
refactor: 重构
test: 测试
chore: 工程杂项
```

## 分支命名建议

```text
feature/c/adk-base
feature/y/web-console
feature/g/a2a-server
bugfix/c/fix-agent-run
bugfix/y/fix-ui-renderer
bugfix/g/fix-stream-timeout
```

不要使用含糊的分支名：

```text
my-code
test
new-branch
update
final
```
