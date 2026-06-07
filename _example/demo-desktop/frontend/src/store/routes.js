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
      vault_picker: {
        title: "选择 Vault",
        pathname: "/vault-picker",
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
routesWithPathname["/desktop"] = routesWithPathname["/home/index"];
