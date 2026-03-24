# RouteViewCore / buildRoutes — 路由视图

源文件：`packages/kit/src/route_view/index.ts`
工具：`packages/kit/src/route_view/utils.ts`

## RouteViewCore

管理单个路由视图的挂载/显示/隐藏生命周期：

```ts
new RouteViewCore({
  name: "root.home",
  pathname: "/home",
  title: "Home",
  parent?: parentView,
  visible?: boolean,
  layout?: boolean,
  children?: RouteViewCore[],
})

view.name / view.pathname / view.title
view.mounted / view.visible / view.layered
view.curView          // 当前活跃的子视图
view.subViews         // 所有子视图
view.$presence        // PresenceCore（动画控制）

view.show() / view.hide()
view.mount() / view.unmount()
view.appendView(sub) / view.removeView(sub)
view.showView(sub)
view.buildUrl(query)

view.onShow(fn) / view.onHidden(fn) / view.onMounted(fn)
view.onStateChange(fn)
```

## buildRoutes

将路由配置树转换为扁平路由表：

```js
const { routes, views, defaultRouteName, notfoundRouteName } =
  Timeless.buildRoutes({
    root: {
      title: "Root",
      pathname: "/",
      children: {
        login: {
          title: "Login",
          pathname: "/login",
          component: Timeless.lazy("@/pages/login.js"),
        },
        home_layout: {
          title: "Home",
          pathname: "/home",
          layout: true,
          component: Timeless.lazy("@/pages/home/layout.js"),
          children: {
            index: {
              title: "Dashboard",
              pathname: "/home/index",
              component: Timeless.lazy("@/pages/home/index.js"),
              options: { default: true },
            },
          },
        },
        notfound: {
          title: "404",
          pathname: "/notfound",
          options: { notfound: true },
        },
      },
    },
  });
```

## RouteMenusModel

创建与路由联动的侧边栏菜单模型，通过 `view.curView` 自动跟踪当前路由：

```js
const sidemenu$ = Timeless.RouteMenusModel({
  view: props.view,
  menus: [
    { title: "Dashboard", url: "root.home_layout.index" },
    { title: "Settings", url: "root.home_layout.settings" },
  ],
});

// sidemenu$.cur — 当前活跃的 RouteViewCore（或 null）
// sidemenu$.isSubRoute(url) — 当前路由是否为 url 的子路由
// sidemenu$.isActive(url) — 当前路由是否精确匹配 url
computed(sidemenu$.cur, () => sidemenu$.isSubRoute("root.home_layout.index"));
computed(sidemenu$.cur, () => sidemenu$.isActive("root.home_layout.settings"));
```
