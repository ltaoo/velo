/**
 * @file Store 入口 - 路由管理
 */
import LoginPage from "@/pages/login/index.js";
import NotFoundPageView from "@/pages/notfound/index.js";
import HomeLayoutView from "@/pages/home/index.js";

const route_configure = {
  home: {
    title: "首页",
    pathname: "/home",
    component: HomeLayoutView,
    default: true,
  },
  login: {
    title: "登录",
    pathname: "/login",
    component: LoginPage,
  },
  notfound: {
    title: "404",
    pathname: "/notfound",
    component: NotFoundPageView,
    notfound: true,
  },
};

const r = Timeless.buildRoutes(route_configure);
const routes = r.routes;
export const views = r.views;
export const defaultRouteName = r.defaultRouteName;
export const notfoundRouteName = r.notfoundRouteName;

// LocalStorage
const DEFAULT_CACHE_VALUES = {
  user: {
    id: "",
    username: "anonymous",
    email: "",
    token: "",
    avatar: "",
  },
  theme: "system",
};
const key = "timeless";
const e = globalThis.localStorage.getItem(key);
export const storage = new Timeless.StorageCore({
  key,
  defaultValues: DEFAULT_CACHE_VALUES,
  values: (() => {
    const prev = JSON.parse(e || "{}");
    return {
      ...prev,
    };
  })(),
  client: globalThis.localStorage,
});
// HttpClient
export const client = new Timeless.HttpClientCore({
  headers: {
    "Content-Type": "application/json",
  },
});
Timeless.web.provide_http_client(client);
export const user = /** @type {any} */ ({});
// History
Timeless.NavigatorCore.prefix = "/timeless";
export const router = new Timeless.NavigatorCore();
export const rootview = new Timeless.RouteViewCore({
  name: "root",
  pathname: "/",
  title: "ROOT",
  visible: true,
  parent: null,
  views: [],
});
rootview.isRoot = true;
export const history = new Timeless.HistoryCore({
  view: rootview,
  router,
  routes,
  views: {
    root: rootview,
  },
});
Timeless.web.provide_history(history);

export const app = new Timeless.ApplicationModel({
  user,
  storage,
  async beforeReady() {
    const { pathname, query } = router;
    const route = r.routesWithPathname[pathname];
    console.log("[Store] beforeReady", pathname, route);
    // if (route.options?.require?.includes("login")) {
    //   if (!user.isLogin) {
    //     app.tip?.({ text: ["请先登录"] });
    //     history.push("root.login", { redirect: route.pathname });
    //     return Timeless.Result.Err("need login");
    //   }
    // }
    if (!route || history.isRoot(route.name)) {
      history.push(defaultRouteName, {}, { ignore: true });
      return Timeless.Result.Ok(null);
    }
    history.push(route.name, query, { ignore: true });
    return Timeless.Result.Ok(null);
  },
});
Timeless.web.provide_app(app);

history.onRouteChange(({ reason, view, href, ignore }) => {
  const { title } = view || {};
  if (title) {
    app.setTitle(title);
  }
  if (ignore) {
    return;
  }
  if (reason === "push") {
    router.pushState(href);
  }
  if (reason === "replace") {
    router.replaceState(href);
  }
});
history.onClickLink(({ href, target }) => {
  const { pathname, query } = Timeless.NavigatorCore.parse(href);
  const route = r.routesWithPathname[pathname];
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
