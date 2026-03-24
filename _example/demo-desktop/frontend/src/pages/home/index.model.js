export function HomePageViewModel(props) {
  const version = ref("");
  invoke("/api/app", { method: "GET" }).then((r) => {
    if (r && r.data) {
      version.value = r.data.version;
    }
  });
  return { version };
}
