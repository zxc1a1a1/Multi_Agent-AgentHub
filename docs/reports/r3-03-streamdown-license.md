# R3-03 Streamdown 许可与包体评估

- 日期：2026-11-03
- 包：`streamdown@2.7.1`
- 许可证：**Apache-2.0**（permissive；不是 AGPL/SSPL，不需要开源本仓库）
- 发布包体积：125,435 bytes（unpacked，`dist.unpackedSize`）
- 直接依赖：`clsx`、`marked`、`remend`、`unified`、`rehype-raw`、`remark-gfm`、
  `remark-parse`、`rehype-harden`、`remark-rehype`、`tailwind-merge`、
  `rehype-sanitize`、`unist-util-visit`、`html-url-attributes`、
  `hast-util-to-jsx-runtime`、`unist-util-visit-parents`
- Peer：React `^18 || ^19`；本仓库 React 18.3.1，兼容。
- 安全相关：使用 `rehype-harden` + `rehype-sanitize`；HTML/脚本注入测试通过
  （`StreamdownRenderer.test.tsx`）。
- 决策：**GO（仅 opt-in 试点）**。默认构建不加载；`--mode streamdown` 才启用。
  未满足“首屏不加载重组件”前不替换默认渲染器。

## 构建预算（2026-11-03，Windows / Node 20.12.2）

| 构建 | 入口 | 入口 gzip | Streamdown chunk |
|---|---|---|---|
| 默认（flag off） | `index-CCzS8niA.js` | 159,726 B（156.0 KiB） | 不存在 |
| `vite build --mode streamdown` | `index-BjRH7Qws.js` | 160.30 KiB | `index-DHUuYPXS.js` 200.33 KiB / gzip 64.11 KiB，动态 import，未进入 index.html 预加载 |

- 入口 gzip 均 ≤500KB。
- Streamdown chunk 只在 `STREAMDOWN_ENABLED=true` 且组件实际渲染时动态加载；
  加载/渲染失败回退旧渲染器。

## 启用方式

```bash
cd frontend
node node_modules/vite/bin/vite.js build --mode streamdown
# .env.streamdown: VITE_STREAMDOWN_ENABLED=true
```

默认 `npm run build` 不读取 `.env.streamdown`，行为与 R3-03 前一致。
