export default function LoginPage(props) {
  const ui = {
    input_username: new Timeless.ui.InputCore({
      defaultValue: "",
    }),
    input_pwd: new Timeless.ui.InputCore({
      defaultValue: "",
    }),
  };

  const handleLogin = () => {
    const username = ui.input_username.value;
    const password = ui.input_pwd.value;

    if (username === "admin" && password === "123456") {
      // Login successful
      console.log("Login successful");
      // Redirect to home page
      // Assuming the route name is "root.home_layout.index" based on views.js
      props.history.replace("root.home_layout.index.general");
      return;
    }
    // props.app.tip({
    //   msg: ["Invalid username or password"],
    // });
    alert("Invalid username or password");
  };

  return View(
    {
      class: cn([
        "flex min-h-screen flex-col items-center justify-center bg-gray-100 py-12 sm:px-6 lg:px-8",
        "dark:bg-zinc-900", // Dark mode background
      ]),
    },
    [
      View({ class: cn(["sm:mx-auto sm:w-full sm:max-w-md"]) }, [
        // Logo (Text for now)
        View(
          {
            class: cn([
              "mx-auto text-center text-3xl font-bold tracking-tight text-gray-900",
              "dark:text-white", // Dark mode text
            ]),
          },
          ["Timeless"],
        ),
        View(
          {
            class: cn([
              "mt-2 text-center text-sm text-gray-600",
              "dark:text-zinc-400", // Dark mode secondary text
            ]),
          },
          ["Sign in to your account"],
        ),
      ]),

      View({ class: cn(["mt-8 sm:mx-auto sm:w-full sm:max-w-md"]) }, [
        View(
          {
            class: cn([
              "bg-white py-8 px-4 shadow sm:rounded-lg sm:px-10 space-y-6",
              "dark:bg-zinc-800", // Dark mode card background
            ]),
          },
          [
            // Username Input
            View({ class: cn(["space-y-1"]) }, [
              Label(
                {
                  class: cn([
                    "block text-sm font-medium text-gray-700",
                    "dark:text-zinc-300", // Dark mode label text
                  ]),
                },
                ["Username"],
              ),
              View({ class: "mt-1" }, [
                Input({
                  store: ui.input_username,
                  placeholder: "Enter your username",
                  class: cn([
                    "block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm",
                    "dark:bg-zinc-900 dark:border-zinc-700 dark:text-white dark:placeholder-zinc-500 dark:focus:border-indigo-500 dark:focus:ring-indigo-500", // Dark mode input styles
                  ]),
                }),
              ]),
            ]),

            // Password Input
            View({ class: "space-y-1" }, [
              Label(
                {
                  class: cn([
                    "block text-sm font-medium text-gray-700",
                    "dark:text-zinc-300", // Dark mode label text
                  ]),
                },
                ["Password"],
              ),
              View({ class: "mt-1" }, [
                Input({
                  store: ui.input_pwd,
                  type: "password",
                  placeholder: "Enter your password",
                  class: cn([
                    "block w-full appearance-none rounded-md border border-gray-300 px-3 py-2 placeholder-gray-400 shadow-sm focus:border-indigo-500 focus:outline-none focus:ring-indigo-500 sm:text-sm",
                    "dark:bg-zinc-900 dark:border-zinc-700 dark:text-white dark:placeholder-zinc-500 dark:focus:border-indigo-500 dark:focus:ring-indigo-500", // Dark mode input styles
                  ]),
                }),
              ]),
            ]),

            // Login Button
            View({}, [
              Button(
                {
                  store: new Timeless.ui.ButtonCore({
                    class: cn([
                      "flex w-full justify-center rounded-md border border-transparent bg-indigo-600 py-2 px-4 text-sm font-medium text-white shadow-sm hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2",
                      "dark:bg-indigo-500 dark:hover:bg-indigo-600 dark:focus:ring-indigo-400", // Dark mode button styles
                    ]),
                    onClick: handleLogin,
                  }),
                },
                ["Sign in"],
              ),
            ]),

            // Hint
            View(
              {
                class: cn([
                  "mt-6 text-center text-xs text-gray-500",
                  "dark:text-zinc-500", // Dark mode hint text
                ]),
              },
              [
                "Hint: Use username ",
                View(
                  {
                    type: "span",
                    class: cn([
                      "font-mono font-medium text-gray-700",
                      "dark:text-zinc-300", // Dark mode code text
                    ]),
                  },
                  ["admin"],
                ),
                " and password ",
                View(
                  {
                    type: "span",
                    class: cn([
                      "font-mono font-medium text-gray-700",
                      "dark:text-zinc-300", // Dark mode code text
                    ]),
                  },
                  ["123456"],
                ),
              ],
            ),
          ],
        ),
      ]),
    ],
  );
}
