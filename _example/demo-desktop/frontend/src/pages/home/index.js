import { HomePageViewModel } from "./index.model.js";
import { mountMemosHome } from "./memos.js";

export function HomePageView() {
  const vm$ = HomePageViewModel(props);

  let memoApp = null;

  return View(
    {
      class: "page memos-page w-full h-full",
      onMounted(el) {
        memoApp = mountMemosHome(el, {
          version: vm$.version,
        });
      },
      onUnmounted() {
        if (memoApp) memoApp.destroy();
        memoApp = null;
      },
    },
    [],
  );
}
