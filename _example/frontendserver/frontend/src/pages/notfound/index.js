import { defaultRouteName } from "@/store/index.js";

export default function NotFoundPageView(props) {
  return View(
    {
      class: cn([
        "flex flex-col items-center justify-center min-h-screen",
        "bg-[var(--background)] text-[var(--foreground)]",
      ]),
    },
    [
      View({ class: "flex flex-col items-center space-y-4 text-center" }, [
        View(
          {
            class: "text-9xl font-bold opacity-10 select-none",
          },
          ["404"],
        ),
        View({ class: "text-2xl font-medium" }, ["页面未找到"]),
        View({ class: "opacity-60" }, ["抱歉，您访问的页面不存在。"]),
        View({ class: "mt-8" }, [
          Button(
            {
              store: new Timeless.ui.ButtonCore({
                class: cn([
                  "px-6 py-3 rounded-lg font-medium transition-opacity",
                  "bg-[var(--foreground)] text-[var(--background)] hover:opacity-90",
                ]),
                onClick() {
                  props.history.push(defaultRouteName);
                },
              }),
            },
            ["返回首页"],
          ),
        ]),
      ]),
    ],
  );
}
