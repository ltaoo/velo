/**
 * 首页布局
 */
import { defaultRouteName } from "@/store/index.js";

export default function HomeLayoutView(props) {
  return Flex({ class: "layout_home w-full h-full" }, [
    View(
      {
        class:
          "sidebar-wrapper w-[72px] h-full flex flex-col items-center py-6 border-r border-zinc-200 bg-white dark:bg-zinc-950 dark:border-zinc-800",
      },
      [
        // Logo
        View(
          {
            class:
              "relative w-10 h-10 rounded-xl bg-black text-white flex items-center justify-center font-bold text-xl mb-8 shadow-sm cursor-pointer hover:opacity-90 transition-opacity dark:bg-white dark:text-black",
            onClick() {
              props.history.push(defaultRouteName);
            },
          },
          ["T"],
        ),
      ],
    ),
    View({ class: "relative overflow-y-auto flex-1 w-0 h-full" }, [
      View({}, "Hello"),
    ]),
  ]);
}
