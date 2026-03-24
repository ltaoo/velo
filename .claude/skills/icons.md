---
description: "Look up Timeless Icons usage, createIcon factory, icon list, and naming conventions. Trigger when user asks about icon, 图标, XxxOutlined, @timeless/icons, or createIcon."
---

# @timeless/icons 图标包查阅

用户询问图标用法、图标列表、或如何添加新图标时，**读取子文件**后再回答。

## 子文件索引

| 用户提到 | 读取文件 |
|---------|---------|
| 图标用法, createIcon, 图标列表, 添加图标, XxxOutlined | `.claude/skills/icons/usage.md` |

## 查阅流程

1. 读取 `.claude/skills/icons/usage.md`
2. 如需查看某个图标的具体 SVG → 读取 `packages/icons/src/<kebab-name>.ts`
3. 如需查看 barrel export → 读取 `packages/icons/src/index.ts`
