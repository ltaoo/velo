export function UIExamplePageView() {
  return View({ class: "flex" }, [
    View({ class: "w-[320px]" }, [
      For({
        each: ref([
          {
            label: "Controls",
          },
          {
            label: "List",
          },
          {
            label: "Table",
          },
        ]),
        render(menu) {
          return View({ class: "flex items-center px-4 py-2" }, [
            Txt(menu.label),
          ]);
        },
      }),
    ]),
    View({ class: "flex-1 w-0 p-4" }, [
      View({ class: "sections space-y-8" }, [
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Button")]),
          View({ class: "section__body space-x-4" }, [
            Button({}, [Txt("Regular Button")]),
            Button({ type: "primary" }, [Txt("Primary Button")]),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Dropdown Menu")]),
          View({ class: "section__body space-x-4" }, [
            DropdownMenu(
              {
                store: new Timeless.ui.DropdownMenuCore({
                  items: [
                    new Timeless.ui.MenuItemCore({
                      label: "Apple",
                      onClick() {
                        console.log("click apple menu");
                      },
                    }),
                    new Timeless.ui.MenuItemCore({
                      label: "Banana",
                      onClick() {
                        console.log("click banana menu");
                      },
                    }),
                  ],
                }),
              },
              [Button({}, [Txt("Click it")])],
            ),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Select")]),
          View({ class: "section__body space-x-4" }, [
            Select({
              store: new Timeless.ui.SelectCore({
                defaultValue: "apple",
                options: [
                  {
                    value: "apple",
                    label: "苹果",
                  },
                  {
                    value: "banana",
                    label: "香蕉",
                  },
                ],
              }),
            }),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Input")]),
          View({ class: "section__body space-x-4" }, [
            Input({
              store: new Timeless.ui.InputCore({
                defaultValue: "",
              }),
            }),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Checkbox")]),
          View({ class: "section__body space-x-4" }, [
            Checkbox({
              store: new Timeless.ui.CheckboxCore({}),
            }),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Switch")]),
          View({ class: "section__body space-x-4" }, [
            Switch({
              value: "",
              //       store: new Timeless.ui.SwitchCore({
              //         defaultValue: "",
              //       }),
            }),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Slider")]),
          View({ class: "section__body space-x-4" }, [
            Slider({
              value: "",
              //       store: new Timeless.ui.SwitchCore({
              //         defaultValue: "",
              //       }),
            }),
          ]),
        ]),
        View({ class: "section" }, [
          View({ class: "section__title text-2xl" }, [Txt("Tabs")]),
          View({ class: "section__body space-x-4" }, [
            Tabs({
              value: "tab1",
              items: [
                {
                  label: "Tab 1",
                  value: "tab1",
                  content: "Tab 1 Content",
                },
                {
                  label: "Tab 2",
                  value: "tab2",
                  content: "Tab 2 Content",
                },
              ],
            }),
          ]),
        ]),
      ]),
    ]),
  ]);
}
