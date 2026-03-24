/**
 * @file 路由配置
 */
const configure = {
  root: {
    title: "ROOT",
    pathname: "/",
    children: {
      home_layout: {
        title: "首页",
        pathname: "/home",
        children: {
          index: {
            title: "工作台",
            pathname: "/home/index",
          },
          example: {
            title: "组件示例",
            pathname: "/home/ui",
          },
          update: {
            title: "检查更新",
            pathname: "/home/update",
          },
        },
        options: {
          require: [],
        },
      },
      login: {
        title: "登录",
        pathname: "/login",
        options: {
          require: [],
        },
      },
      notfound: {
        title: "404",
        pathname: "/notfound",
      },
    },
  },
};
const result = Timeless.build(configure);
export const routes = result.routes;
export const routesWithPathname = result.routesWithPathname;
