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

window.addEventListener("click", (event) => {
  if (event.defaultPrevented || event.button !== 0) return;
  const link = closestAnchor(event.target);
  if (!link) return;

  const externalURL = externalBrowserURL(link.getAttribute("href") || link.href || "");
  if (!externalURL) return;

  event.preventDefault();
  event.stopPropagation();
  confirmOpenExternalLink(externalURL);
}, true);

history.onClickLink(({ href, target }) => {
  const externalURL = externalBrowserURL(href);
  if (externalURL) {
    confirmOpenExternalLink(externalURL);
    return;
  }

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

function closestAnchor(target) {
  let node = target;
  if (node && node.nodeType === 3) node = node.parentElement;
  if (!node || typeof node.closest !== "function") return null;
  return node.closest("a[href]");
}

function externalBrowserURL(href) {
  const value = String(href || "").trim();
  if (!/^https?:\/\//i.test(value)) return "";

  try {
    const url = new URL(value);
    if ((url.protocol === "http:" || url.protocol === "https:") && url.host) {
      return url.href;
    }
  } catch (_) {}
  return "";
}

function confirmOpenExternalLink(url) {
  openExternalLinkInDefaultBrowser(url);
}

function openExternalLinkInDefaultBrowser(url) {
  if (typeof invoke !== "function") {
    window.open(url, "_blank", "noopener");
    return;
  }

  invoke("/api/external/open?url=" + encodeURIComponent(url), { method: "GET" }).then(
    (resp) => {
      if (!resp || resp.code !== 0) {
        app.tip?.({ text: [(resp && resp.msg) || "打开链接失败"] });
      }
    },
    (err) => {
      app.tip?.({ text: ["打开链接失败: " + err] });
    },
  );
}

// @ts-ignore
TimelessWeb.provide_app(app);
// @ts-ignore
TimelessWeb.provide_history(history);
