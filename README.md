# 圣魔社 SeiMa 概念站（非官方）

基于 [Astro](https://astro.build) 的静态概念网页，展示虚拟主播公会「圣魔社 SeiMa」的主播阵容、直播历史与企划动态。**非官方项目**：内容基于公开资料（B 站官方动态、投稿页、招募文章及各主播公开主页）整理，不能代表公会对成员名单、活动排期或商用素材的正式确认。调研详情见 [`info.md`](./info.md)。

## 项目结构

```
seima-unofficial/
├── src/
│   ├── pages/index.astro        # 单页入口，按序组装各区块
│   ├── layouts/Layout.astro     # HTML 骨架、导航、页脚、字体与 meta
│   ├── components/              # Hero / About / Talents / StreamingHistory / Updates / Join
│   │   └── icons/               # 内联 SVG 图标组件
│   └── styles/                  # 按 Layout 导入顺序层叠的 CSS
├── streaming-record/            # B 站直播开播/下播记录服务（Go + SQLite），详见其 README
├── public/assets/               # favicon 与主播头像（webp）
├── info.md                      # 调研资料：主播身份、动态时间线、宣传文案
└── astro.config.mjs             # dev 代理 /api → streaming-record 服务
```

## 快速开始

需 Node.js 22.12+（或 24+）与 pnpm 10.33.0。

```sh
pnpm install
pnpm dev      # 开发服务器，Astro 默认 http://localhost:4321/
```

`pnpm dev` 会把 `/api/*` 代理到 `https://streaming-time.rotcool.top`（见 `astro.config.mjs`），供首页「直播历史」区块在本地拉取开播/下播记录；生产构建则直连该地址。

```sh
pnpm build    # 产物输出到 dist/
pnpm preview  # 预览生产产物
```

## streaming-record 服务

首页的直播历史数据来自同仓库的 Go 服务（每分钟轮询 B 站 API，SQLite 记录 sessions，提供 `/query`、`/getInfo` 与带 Bearer 鉴权的 `/admin/*` CRUD）。本地运行与 API 说明见 [`streaming-record/README.md`](./streaming-record/README.md)。

每周预约按后端约定的 UTC+8（北京时间）计算下一次开播时刻，再按用户浏览器的本地时区显示日期与时间，与直播历史记录保持一致；跨日和夏令时偏移由浏览器处理。

## 版权说明

- 页面代码与视觉元素为独立制作，未复制参考站点素材。
- 头像（`public/assets/*.webp`）为各主播公开 B 站主页头像的本地副本，版权归原权利人所有；正式对外发布前应取得使用许可。
- 网页字体使用 Google Fonts（LXGW WenKai TC / Playball / Shippori Mincho），无法访问时回退系统字体。
- 主播名单与身份描述基于调研时点（2026-10-15）的公开页面内容，可能随时间变化。
