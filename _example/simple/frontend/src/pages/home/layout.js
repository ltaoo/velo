import { RouterSubViews } from "../../components/sub-views.js";

const nav_menus = [
  { id: "root.home_layout.index", icon: "home", label: "首页" },
  { id: "root.home_layout.example", icon: "explore", label: "组件" },
  { id: "root.home_layout.update", icon: "settings", label: "更新" },
];
const icons = {
  home: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"></path><polyline points="9 22 9 12 15 12 15 22"></polyline></svg>',
  calendar:
    '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2" ry="2"></rect><line x1="16" y1="2" x2="16" y2="6"></line><line x1="8" y1="2" x2="8" y2="6"></line><line x1="3" y1="10" x2="21" y2="10"></line></svg>',
  chat: '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z"></path></svg>',
  explore:
    '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>',
  settings:
    '<svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>',
};
const LogoSVG = `<svg width="32" height="32" viewBox="0 0 24 24" fill="var(--GREEN)">
  <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-2 15l-5-5 1.41-1.41L10 14.17l7.59-7.59L19 8l-9 9z"/>
</svg>`;

export function HomeLayoutView(props) {
  /** @type {Timeless.RouteViewCore} */
  const view = props.view;
  const curSubView = ref(view.curView);
  view.onCurViewChange((view) => {
    curSubView.value = view;
  });

  return Flex(
    {
      class: "layout_home w-full h-full",
      dataset: {
        name: props.view.name,
        pathname: props.view.pathname,
      },
    },
    [
      View({ class: "sidebar-wrapper w-[72px]" }, [
        View(
          {
            class:
              "w-[72px] h-full bg-[var(--BG-1)] flex flex-col items-center py-4 border-r border-[var(--border)]",
          },
          [
            View({ class: "mb-6" }, [
              View({ class: "w-10 h-10 flex items-center justify-center" }, [
                DangerouslyInnerHTML(LogoSVG),
              ]),
            ]),
            For({
              class: "flex-1 flex flex-col gap-2 w-full",
              each: nav_menus,
              render(menu) {
                return View(
                  {
                    class: classnames(
                      computed({ curView: curSubView }, (draft) => {
                        const isSelected =
                          draft.curView.name === menu.id ||
                          menu.id.startsWith(draft.curView.name);
                        return [
                          "w-full py-3 flex flex-col items-center gap-1 border-none bg-transparent cursor-pointer transition-all duration-200 rounded-none hover:bg-[var(--BG-3)]",
                          isSelected
                            ? "text-[var(--GREEN)] bg-[var(--BG-3)]"
                            : "text-[var(--FG-1)] hover:text-[var(--FG-0)]",
                        ].join(" ");
                      }),
                    ),
                    dataset: { id: menu.id, label: menu.label },
                    onClick() {
                      console.log("Navigate to:", menu);
                      props.history.push(menu.id);
                    },
                  },
                  [
                    DangerouslyInnerHTML(icons[menu.icon]),
                    View({ class: "text-[10px] font-medium" }, [
                      Txt(menu.label),
                    ]),
                  ],
                );
              },
            }),
            View({ class: "mt-auto" }, [
              View(
                {
                  class:
                    "w-9 h-9 rounded-full bg-[var(--GREEN)] text-white flex items-center justify-center text-sm font-semibold",
                },
                [View({ class: "avatar-letter" }, [Txt("U")])],
              ),
            ]),
          ],
        ),
      ]),
      RouterSubViews({
        class: "absolute inset-0 left-[72px] right-0 h-full",
        view: view,
        app: props.app,
        history: props.history,
        views: props.views,
        storage: props.storage,
        client: props.client,
      }),
    ],
  );
}
