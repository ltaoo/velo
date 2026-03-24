# @timeless/icons 使用指南

## 安装与导入

```ts
// 全量导入
import { CalendarOutlined, SearchOutlined } from "@timeless/icons";

// 按需导入（tree-shakeable，推荐）
import { CalendarOutlined } from "@timeless/icons/calendar";
import { SearchOutlined } from "@timeless/icons/search";
```

## createIcon 工厂函数

位于 `packages/icons/src/utils.ts`。

```ts
export function createIcon(svg: string) {
  return function (props?: {
    class?: string;
    style?: string;
    id?: string;
    onClick?: (e: MouseEvent) => void;
    onMounted?: (el: HTMLSpanElement) => void;
    beforeUnmounted?: () => void;
    onUnmounted?: () => void;
  }) => ViewLike;
}
```

返回值是一个符合 Timeless View 协议的对象（`{ t: "view", $elm, render(), onMounted(), beforeUnmounted(), onUnmounted() }`），可直接用在 `View` children 中。

### Props

| prop | 类型 | 说明 |
|------|------|------|
| `class` | `string` | 添加到外层 `<span>` 的 className |
| `style` | `string` | 追加的内联样式 |
| `id` | `string` | 设置元素 id |
| `onClick` | `(e) => void` | 点击事件 |
| `onMounted` | `(el) => void` | 挂载后回调，参数为外层 span |
| `beforeUnmounted` | `() => void` | 卸载前回调 |
| `onUnmounted` | `() => void` | 卸载后回调 |

## 使用示例

```ts
import { View } from "@timeless/headless";
import { SearchOutlined } from "@timeless/icons/search";

View(() => [
  SearchOutlined({ class: "text-muted-foreground", style: "font-size: 16px" }),
]);
```

## SVG 特性

- 尺寸：`width="1em" height="1em"` — 随文字大小缩放
- 颜色：`stroke="currentColor"` — 继承父元素文字颜色
- 外层包裹 `<span style="display:inline-flex;align-items:center;justify-content:center">`
- 图标来源：Lucide Icons

## 命名约定

- 文件名：kebab-case（如 `arrow-down-to-line.ts`）
- 导出名：PascalCase + `Outlined` 后缀（如 `ArrowDownToLineOutlined`）

## 完整图标清单（42 个）

| 文件名 | 导出名 |
|--------|--------|
| `arrow-down-to-line` | `ArrowDownloadToLineOutlined` |
| `bolt` | `BoltOutlined` |
| `calendar` | `CalendarOutlined` |
| `check` | `CheckOutlined` |
| `chevron-down` | `ChevronDownOutlined` |
| `chevron-left` | `ChevronLeftOutlined` |
| `chevron-right` | `ChevronRightOutlined` |
| `chevron-up` | `ChevronUpOutlined` |
| `circle-arrow-down` | `CircleArrowDownOutlined` |
| `circle-ellipsis` | `CircleEllipsisDownOutlined` |
| `circle-x` | `CircleXOutlined` |
| `clock` | `ClockOutlined` |
| `clock-arrow-down` | `ClockArrowDownOutlined` |
| `cloud-download` | `CloudDownloadOutlined` |
| `download` | `DownloadOutlined` |
| `ellipsis` | `EllipsisOutlined` |
| `ellipsis-vertical` | `EllipsisVerticalOutlined` |
| `file` | `FileOutlined` |
| `file-box` | `FileBoxOutlined` |
| `file-image` | `FileImageOutlined` |
| `file-lock` | `FileLockOutlined` |
| `file-play` | `FilePlayOutlined` |
| `file-symlink` | `FileSymlinkOutlined` |
| `file-video-camera` | `FileVideoCameraOutlined` |
| `file-volume` | `FileVolumeOutlined` |
| `folder` | `FolderOutlined` |
| `folder-closed` | `FolderClosedOutlined` |
| `git-fork` | `GitForkOutlined` |
| `grid-3x3` | `Grid3x3Outlined` |
| `loader` | `LoaderOutlined` |
| `loader-circle` | `LoaderCircleOutlined` |
| `menu` | `MenuOutlined` |
| `pause` | `PauseOutlined` |
| `play` | `PlayOutlined` |
| `refresh-ccw` | `RefreshCcwOutlined` |
| `rss` | `RSSOutlined` |
| `search` | `SearchOutlined` |
| `square-arrow-down` | `SquareArrowDownOutlined` |
| `trash` | `TrashOutlined` |
| `trash-2` | `Trash2Outlined` |
| `undo-2` | `Undo2Outlined` |
| `x` | `XOutlined` |

## 添加新图标

1. 从 [Lucide](https://lucide.dev/) 获取 SVG
2. 确保 SVG 属性：`width="1em" height="1em" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"`
3. 创建 `packages/icons/src/<kebab-name>.ts`：

```ts
import { createIcon } from "./utils";

const svg = `<svg ...>...</svg>`;

export const XxxOutlined = createIcon(svg);
```

4. 在 `packages/icons/src/index.ts` 添加 `export * from "./<kebab-name>";`
5. 运行 `node packages/icons/scripts/update-exports.js` 更新 `package.json` 的 exports map

## 关键文件

- `packages/icons/src/utils.ts` — createIcon 工厂函数
- `packages/icons/src/index.ts` — barrel export
- `packages/icons/scripts/update-exports.js` — 自动更新 package.json exports
