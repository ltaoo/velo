/**
 * 首页布局
 */
import { defaultRouteName } from "@/store/index.js";

function TodoListViewModel(props) {
  const ui = {
    incBtn$: new Timeless.ui.ButtonCore({
      onClick() {
        count$.as((v) => v + 1);
      },
    }),
    decBtn$: new Timeless.ui.ButtonCore({
      variant: "outline",
      onClick() {
        count$.as((v) => v - 1);
      },
    }),
    resetBtn$: new Timeless.ui.ButtonCore({
      variant: "ghost",
      onClick() {
        count$.as(0);
      },
    }),
    todoInput$: new Timeless.ui.InputCore({
      defaultValue: "",
      placeholder: "请输入待办事项",
    }),
    addTodoBtn$: new Timeless.ui.ButtonCore({
      onClick() {
        const v = (ui.todoInput$.value || "").trim();
        if (!v) {
          return;
        }
        todos$.push({ id: Date.now(), text: v, done: false });
        ui.todoInput$.setValue("");
      },
    }),
  };

  const count$ = ref(0);
  const double$ = computed(count$, (v) => v * 2);
  const todos$ = refarr(
    [
      { id: 1, text: "学习 Timeless", done: false },
      { id: 2, text: "编写示例组件", done: true },
    ],
    { key: "id" },
  );
  const state = {
    count$,
    todos$,
    double$,
  };

  return {
    ui,
    state,
  };
}

export default function HomeLayoutView(props) {
  const vm$ = TodoListViewModel(props);

  return Flex({ class: "layout_home w-full h-full" }, [
    View(
      {
        class:
          "sidebar-wrapper w-[72px] h-full flex flex-col items-center py-6 border-r border-zinc-200 bg-white dark:bg-zinc-950 dark:border-zinc-800",
      },
      [
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
      View(
        {
          class: "max-w-3xl mx-auto p-6 space-y-10 text-[var(--foreground)]",
        },
        [
          View({ class: "space-y-4" }, [
            View({ class: "text-xl font-semibold tracking-tight" }, [
              "Counter 示例",
            ]),
            View({ class: "flex items-center gap-4" }, [
              View(
                { class: "px-3 py-2 rounded bg-zinc-100 dark:bg-zinc-800" },
                ["当前：", vm$.state.count$],
              ),
              View(
                { class: "px-3 py-2 rounded bg-zinc-100 dark:bg-zinc-800" },
                ["双倍：", vm$.state.double$],
              ),
              Button({ store: vm$.ui.incBtn$ }, ["+1"]),
              Button({ store: vm$.ui.decBtn$ }, ["-1"]),
              Button({ store: vm$.ui.resetBtn$ }, ["重置"]),
            ]),
          ]),
          View({ class: "space-y-4" }, [
            View({ class: "text-xl font-semibold tracking-tight" }, [
              "Todo 示例",
            ]),
            View({ class: "flex items-center gap-3" }, [
              Input({
                store: vm$.ui.todoInput$,
                class: "w-full max-w-md",
              }),
              Button({ store: vm$.ui.addTodoBtn$ }, ["添加"]),
            ]),
            For({
              key: "id",
              each: vm$.state.todos$,
              render(item) {
                const index = computed(vm$.state.todos$, (t) => {
                  return t.indexOf(item);
                });
                const cb$ = new Timeless.ui.CheckboxCore({
                  checked: !!item.done,
                  onChange(checked) {
                    // vm$.state.todos$.set(index, { ...item, done: checked });
                    getobj(item).assign({
                      done: checked,
                    });
                  },
                });
                return View(
                  {
                    class:
                      "flex items-center justify-between px-3 py-2 rounded hover:bg-zinc-50 dark:hover:bg-zinc-900",
                  },
                  [
                    View({ class: "flex items-center gap-3" }, [
                      index,
                      Checkbox({ store: cb$ }),
                      View(
                        {
                          class: cn([
                            "text-sm",
                            computed(item, (t) => {
                              return t.done
                                ? "line-through text-zinc-400"
                                : "text-zinc-800 dark:text-zinc-200";
                            }),
                          ]),
                        },
                        [computed(item, (t) => t.text)],
                      ),
                    ]),
                    View({ class: "flex items-center gap-2" }, [
                      Button(
                        {
                          store: new Timeless.ui.ButtonCore({
                            variant: "outline",
                            onClick() {
                              vm$.state.todos$.as((arr) => {
                                if (index.value <= 0) return arr;
                                const next = arr.slice();
                                const [itemMove] = next.splice(index.value, 1);
                                next.splice(index.value - 1, 0, itemMove);
                                return next;
                              });
                            },
                          }),
                        },
                        ["上移"],
                      ),
                      Button(
                        {
                          store: new Timeless.ui.ButtonCore({
                            variant: "outline",
                            onClick() {
                              vm$.state.todos$.as((arr) => {
                                if (index.value >= arr.length - 1) return arr;
                                const next = arr.slice();
                                const [itemMove] = next.splice(index.value, 1);
                                next.splice(index.value + 1, 0, itemMove);
                                return next;
                              });
                            },
                          }),
                        },
                        ["下移"],
                      ),
                      Button(
                        {
                          store: new Timeless.ui.ButtonCore({
                            variant: "secondary",
                            onClick() {
                              vm$.state.todos$.as((arr) => {
                                if (index.value <= 0) return arr;
                                const next = arr.slice();
                                const [itemMove] = next.splice(index.value, 1);
                                next.unshift(itemMove);
                                return next;
                              });
                            },
                          }),
                        },
                        ["置顶"],
                      ),
                      Button(
                        {
                          store: new Timeless.ui.ButtonCore({
                            variant: "ghost",
                            onClick() {
                              vm$.state.todos$.delete(index);
                            },
                          }),
                        },
                        ["删除"],
                      ),
                    ]),
                  ],
                );
              },
            }),
          ]),
        ],
      ),
    ]),
  ]);
}
