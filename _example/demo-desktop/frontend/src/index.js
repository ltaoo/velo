import { app, history, client, views } from "./store/index.js";
import { storage } from "./store/storage.js";
import { RouterSubViews } from "./components/sub-views.js";

const render = ($root) => {
  const { innerWidth, innerHeight, location } = window;
  history.$router.prepare(location);
  app
    .start({
      width: innerWidth,
      height: innerHeight,
    })
    .then(() => {
      const v$ = RouterSubViews({
        class: classnames("root-view w-full h-full"),
        view: history.$view,
        app,
        views,
        history,
        storage,
        client,
      });
      $root.appendChild(v$.render());
      v$.onMounted();
    });
};

document.addEventListener("DOMContentLoaded", function () {
  const $root = document.querySelector("#root");
  if (!$root) {
    console.error("[Render] Root element not found");
    return;
  }
  render($root);
});
