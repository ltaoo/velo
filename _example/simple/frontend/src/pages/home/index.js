import { HomePageViewModel } from "./index.model.js";

export function HomePageView(props) {
  const vm$ = HomePageViewModel(props);
  // const selectedDate = new Date();

  const showFullCalendar = ref(false);
  const ui = {
    presence$: new Timeless.ui.PresenceCore({}),
    popover$: new Timeless.ui.PopoverCore({}),
    presence_calendar$: new Timeless.ui.PresenceCore({}),
  };

  return Flex({ class: "page w-full h-full" }, [
    View({ class: "p-4 pr-0 w-[240px]" }, []),
    View({ class: "p-4 flex-1 w-0" }, []),
    View({ class: "p-4 pl-0 w-[260px]" }, []),
    // Popper({ store: ui.popover$ }, [
    //   View({ class: "bg-[#1E1E1E] p-4 rounded-xl shadow-2xl" }, [
    //     Txt("Popover Content"),
    //   ]),
    // ]),
    Portal({}, [
      Presence(
        {
          store: ui.presence_calendar$,
        },
        [
          View(
            {
              class:
                "fixed inset-0 bg-black/80 z-50 flex items-center justify-center backdrop-blur-sm",
              onClick: (e) => {
                // Close on background click (when clicking the backdrop)
                // We need to check if the click target is the backdrop itself
                // Note: In this custom View/baseui implementation, 'e' is the native event.
                if (e.target === e.currentTarget) {
                  ui.presence_calendar$.hide();
                }
              },
            },
            [
              View(
                {
                  class:
                    "bg-[#1E1E1E] w-[90%] h-[90%] rounded-xl shadow-2xl overflow-hidden relative border border-[#333] flex flex-col",
                },
                [
                  // Close button
                  Button(
                    {
                      class:
                        "absolute top-4 right-4 z-10 text-gray-400 hover:text-white p-2 cursor-pointer bg-black/20 rounded-full w-8 h-8 flex items-center justify-center",
                      onClick: () => {
                        ui.presence_calendar$.hide();
                      },
                    },
                    [Txt("✕")],
                  ),
                  // Content
                  View({ class: "flex-1 h-full w-full" }, [
                    View({}, [Txt("Hello!")]),
                  ]),
                ],
              ),
            ],
          ),
        ],
      ),
    ]),
  ]);
}
