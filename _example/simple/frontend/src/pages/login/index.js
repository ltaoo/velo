export function LoginPageView(props) {
  return View({}, [
    View({}, [
      View({ class: "fields" }, [
        View({ class: "field" }, [
          View({ class: "label" }, [Txt("Username")]),
          View({ class: "input" }, [
            Input({ type: "text", placeholder: "Enter your username" }),
          ]),
        ]),
        View({ class: "field" }, [
          View({ class: "label" }, [Txt("Password")]),
          View({ class: "input" }, [
            Input({ type: "password", placeholder: "Enter your password" }),
          ]),
        ]),
      ]),
    ]),
    View({}, [
      Button(
        {
          type: "primary",
          onClick() {
            props.history.push("root.home_layout.home");
          },
        },
        [Txt("Login")],
      ),
    ]),
  ]);
}
