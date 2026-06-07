import { mountMemosHome } from "./memos.js";

export function HomePageView() {
  let memoApp = null;

  return View(
    {
      class: "page memos-page w-full h-full",
      onMounted(el) {
        memoApp = mountMemosHome(el);
      },
      onUnmounted() {
        if (memoApp) memoApp.destroy();
        memoApp = null;
      },
    },
    [],
  );
}
