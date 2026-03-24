export function NotFoundPageView(props) {
  return View({}, [
    View({}, [Txt("404 Not Found")]),
    View({}, [
      Button(
        {
          type: "primary",
          onClick() {
            props.history.push("root.home_layout.home");
          },
        },
        [Txt("Go to Home")],
      ),
    ]),
  ]);
}
