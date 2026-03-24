import { NotFoundPageView } from "../pages/notfound/index.js";

export function RouterSubViews(props) {
  const subViews = ref(props.view.subViews);
  const curSubView = ref(props.view.curView);

  props.view.onCurViewChange((view) => {
    curSubView.value = view;
  });
  props.view.onSubViewAppended((v) => {
    subViews.value.push(v);
  });

  const nodes = [];

  return For({
    class: props.class,
    each: subViews,
    onMounted() {
      // console.log("router sub views mounted", nodes);
      if (props.onMounted) {
        props.onMounted();
      }
      for (const node of nodes) {
        if (typeof node.onMounted === "function") {
          node.onMounted();
        }
      }
    },
    render(subView) {
      const PageView = props.views[subView.name];
      if (!PageView) {
        return NotFoundPageView({
          history: props.history,
        });
      }
      const displayed = computed({ curSubView: curSubView }, (draft) => {
        return [
          "page__wrap absolute inset-0",
          (() => {
            if (!draft.curSubView || !draft.curSubView.name) {
              return "hidden";
            }
            return draft.curSubView.name === subView.name
              ? "display"
              : "hidden";
          })(),
        ].join(" ");
      });
      const p$ = PageView({
        view: subView,
        app: props.app,
        history: props.history,
        storage: props.storage,
        client: props.client,
        views: props.views,
      });
      nodes.push(p$);
      return View(
        {
          class: classnames(displayed),
          style: {},
          dataset: {
            name: subView.name,
            pathname: subView.pathname,
          },
          // onMounted() {
          //   console.log("page view mounted");
          //   if (props.onMounted) {
          //     props.onMounted();
          //   }
          //   if (typeof p$.onMounted === "function") {
          //     p$.onMounted();
          //   }
          // },
        },
        [p$],
      );
    },
  });
}
