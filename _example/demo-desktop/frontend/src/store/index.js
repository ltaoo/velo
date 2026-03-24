/**
 * @file Store 入口 - 路由管理
 */
import { routes, routesWithPathname } from "./routes.js";
import { storage } from "./storage.js";
export { client } from "./http_client.js";
export { views } from "./views.js";

Timeless.NavigatorCore.prefix = "";

// @ts-ignore
export const router = new Timeless.NavigatorCore();
// export const user = new Timeless.UserCore(storage.get("user") || {}, {
//   get: () => Promise.resolve({ data: null }),
//   post: () => Promise.resolve({ data: null }),
// });
export const user = {};
// export const storage = storage;

export const view = new Timeless.RouteViewCore({
  name: "root",
  pathname: "/",
  title: "ROOT",
  visible: true,
  parent: null,
  views: [],
});
view.isRoot = true;

export const history = new Timeless.HistoryCore({
  view,
  router,
  routes,
  views: {
    root: view,
  },
});

export const app = new Timeless.ApplicationModel({
  // @ts-ignore
  user,
  storage,
  async beforeReady() {
    const { pathname, query } = router;
    const route = routesWithPathname[pathname];
    console.log("[Store] beforeReady", pathname, route, routesWithPathname);
    if (!route) {
      // @ts-ignore
      history.push("root.notfound", { replace: true });
      return Timeless.Result.Err("not found");
    }
    // if (!route.options?.require?.includes("login")) {
    //   if (!history.isLayout(route.name)) {
    //     console.log("[Store] beforeReady push to fallback route", route.name);
    //     history.push(route.name, query, { ignore: true });
    //     return Timeless.Result.Ok(null);
    //   }
    //   return Timeless.Result.Err("can't goto layout");
    // }
    // if (!user.isLogin) {
    //   app.tip?.({ text: ["请先登录"] });
    //   history.push("root.login", { redirect: route.pathname });
    //   return Timeless.Result.Err("need login");
    // }
    if (!history.isLayout(route.name)) {
      history.push(route.name, query, { ignore: true });
      return Timeless.Result.Ok(null);
    }
    console.log(
      "[Store] beforeReady push to default page",
      "root.home_layout.index",
    );
    history.push("root.home_layout.index", {}, { ignore: true });
    return Timeless.Result.Ok(null);
  },
});

history.onRouteChange(({ reason, view, href, ignore }) => {
  const { title } = view || {};
  if (title) {
    app.setTitle(title);
  }
  if (ignore) return;
  if (reason === "push") {
    router.pushState(href);
  }
  if (reason === "replace") {
    router.replaceState(href);
  }
});

history.onClickLink(({ href, target }) => {
  // @ts-ignore
  const { pathname, query } = Timeless.NavigatorCore.parse(href);
  const route = routesWithPathname[pathname];
  if (!route) {
    app.tip?.({ text: ["没有匹配的页面"] });
    return;
  }
  if (target === "_blank") {
    window.open(href);
    return;
  }
  history.push(route.name, query);
});

// @ts-ignore
TimelessWeb.provide_app(app);
// @ts-ignore
TimelessWeb.provide_history(history);
