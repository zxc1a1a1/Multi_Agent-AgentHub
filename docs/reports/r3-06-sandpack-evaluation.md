# R3-06 Sandpack 交互预览评估

- 日期：2026-11-03
- 版本：`@codesandbox/sandpack-react@2.20.0` / `@codesandbox/sandpack-client@2.19.8`
- 结论：**NO-GO**（不接入生产交互预览；保留现有 code_preview / 沙箱 HTML）

## 1. 评估维度与证据

| 维度 | 事实 | 结论 |
|---|---|---|
| LICENSE | `@codesandbox/sandpack-react` Apache-2.0；client Apache-2.0 | 通过 |
| 包体 | 包 unpacked 1,206,437 B；20 个直接依赖（CodeMirror 6、react-devtools-inline、lz-string、stitches 等）。独立 esbuild（React external）bundle **976.0 KB min / gzip 280,734 B** | 只能懒加载；单块 gzip 280.7KB |
| 沙箱 | `sandpack-client/dist/clients/runtime/index.js:377`：`iframe.setAttribute("sandbox", "allow-forms allow-modals allow-popups allow-presentation allow-same-origin allow-scripts allow-downloads allow-pointer-lock")` | **FAIL**：包含 `allow-same-origin`，与本项目 R3-02 的 no-same-origin 策略冲突 |
| 离线 | 默认 runtime bundler 指向 `https://codesandbox.io/...` 与 `preview.sandpack-static-server.codesandbox.io`；自托管需单独部署 bundler | **FAIL**：默认依赖第三方在线服务；离线不可用 |
| 维护 | npm `time.modified` = 2025-04-29（评估时约 18 个月前） | 风险项 |
| 首屏 | 懒加载可避免进入首屏；但 chunk 280.7KB gzip，且需要 CodeMirror/devtools 依赖树 | 可接受但成本高 |

## 2. 复现命令

```bash
npm view @codesandbox/sandpack-react version license dist.unpackedSize dependencies time.modified --json
# 临时目录（不进入仓库）
npm install @codesandbox/sandpack-react@2.20.0 esbuild@0.25.0
echo "import { Sandpack } from '@codesandbox/sandpack-react'; console.log(Sandpack)" > entry.js
esbuild entry.js --bundle --minify --format=esm --external:react --external:react-dom --outfile=out.js
gzip -c out.js | wc -c     # 280734
grep -rn "allow-same-origin" node_modules/@codesandbox/sandpack-client/dist/clients/runtime/index.js
grep -rn "codesandbox.io" node_modules/@codesandbox/sandpack-client/dist/clients/runtime/index.js
```

## 3. 决策

计划门槛：首屏 ≤500KB、无同源逃逸、许可通过，三者缺一即 **NO-GO**。

- 许可：通过。
- 同源逃逸：**未满足**（官方 runtime 明确启用 `allow-same-origin`）。
- 离线：**未满足**（默认远程 bundler）。

因此 **NO-GO**：不引入 Sandpack，不修改生产依赖。现有：
- `code_preview`：只读代码文本；
- `html_sandbox`（R3-02）：`allow-scripts` only + 严格 CSP；
- RJSF 表单（R3-04）与声明式图表（R3-05）覆盖已评审的交互需求。

## 4. 后续重新评估条件

1. Sandpack 提供可自托管、无 `allow-same-origin` 的 runtime 配置；
2. 维护活跃（6 个月内有发布）；
3. 懒加载 chunk gzip 与首屏预算同时满足；
4. 有隔离的 bundler 部署方案与逃逸测试。
