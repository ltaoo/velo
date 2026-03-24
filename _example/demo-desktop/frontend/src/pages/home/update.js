export function UpdatePageView(props) {
  const status = ref("idle"); // idle | checking | checked | downloading | downloaded | applying
  const message = ref("");
  const updateInfo = ref(null);
  const progress = ref(0);

  // 监听 Go 端推送的下载进度
  window.onGoMessage((payload) => {
    if (payload && payload.type === "download_progress") {
      progress.value = Math.round(payload.percentage || 0);
      const speed = payload.speed || 0;
      const speedText =
        speed > 1048576
          ? (speed / 1048576).toFixed(1) + " MB/s"
          : (speed / 1024).toFixed(0) + " KB/s";
      message.value =
        "正在下载更新... " + progress.value + "% (" + speedText + ")";
    }
  });

  async function callApi(path) {
    return await invoke(path, { method: "GET" });
  }

  async function checkUpdate() {
    status.value = "checking";
    message.value = "正在检查更新...";
    try {
      const resp = await callApi("/api/update/check");
      if (resp.code !== 0) {
        alert(resp.msg);
        return;
      }
      const data = resp.data;
      console.log("[PAGE]update - before update", data);
      if (data.hasUpdate) {
        status.value = "checked";
        updateInfo.value = data;
        message.value = "发现新版本: " + data.version;
      } else {
        status.value = "idle";
        message.value = "当前已是最新版本 (" + data.currentVersion + ")";
      }
    } catch (e) {
      status.value = "idle";
      message.value = "检查失败: " + e;
    }
  }

  async function downloadUpdate() {
    status.value = "downloading";
    progress.value = 0;
    message.value = "正在下载更新... 0%";
    try {
      const resp = await callApi("/api/update/download");
      if (resp.code !== 0) {
        alert(resp.msg);
        return;
      }
      const data = resp.data;
      if (data.success) {
        progress.value = 100;
        status.value = "downloaded";
        message.value = "下载完成，可以应用更新";
      } else {
        status.value = "checked";
        message.value = "下载失败: " + (data.error || "未知错误");
      }
    } catch (e) {
      status.value = "checked";
      message.value = "下载失败: " + e;
    }
  }

  async function applyUpdate() {
    status.value = "applying";
    message.value = "正在应用更新并重启...";
    try {
      const resp = await callApi("/api/update/restart");
      if (resp.code !== 0) {
        alert(resp.msg);
        return;
      }
      const data = resp.data;
      if (!data.success) {
        status.value = "downloaded";
        message.value = "应用失败: " + (data.error || "未知错误");
      }
    } catch (e) {
      status.value = "downloaded";
      message.value = "应用失败: " + e;
    }
  }

  return View({ class: "page p-6 w-full h-full" }, [
    View({ class: "text-2xl mb-6" }, [Txt("检查更新")]),
    View({ class: "space-y-4" }, [
      View({ class: "flex gap-3" }, [
        Button(
          {
            class: computed({ status }, (s) =>
              s.status === "checking" ? "opacity-50 pointer-events-none" : "",
            ),
            onClick: checkUpdate,
          },
          [
            Txt(
              computed({ status }, (s) =>
                s.status === "checking" ? "检查中..." : "检查更新",
              ),
            ),
          ],
        ),
        Show({ when: computed({ status }, (s) => s.status === "checked") }, [
          Button({ onClick: downloadUpdate }, [
            Txt("下载更新"),
          ]),
        ]),
        Show(
          { when: computed({ status }, (s) => s.status === "downloading") },
          [
            Button({ class: "opacity-50 pointer-events-none" }, [
              Txt("下载中..."),
            ]),
          ],
        ),
        Show({ when: computed({ status }, (s) => s.status === "downloaded") }, [
          Button({ onClick: applyUpdate }, [
            Txt("应用更新并重启"),
          ]),
        ]),
        Show({ when: computed({ status }, (s) => s.status === "applying") }, [
          Button({ class: "opacity-50 pointer-events-none" }, [
            Txt("应用中..."),
          ]),
        ]),
      ]),
      // 进度条
      Show({ when: computed({ status }, (s) => s.status === "downloading") }, [
        View(
          {
            class: "w-full h-4 bg-[var(--BG-2)] rounded-full overflow-hidden",
          },
          [
            View({
              class:
                "h-full bg-[var(--GREEN)] rounded-full transition-all duration-300",
              style: computed({ progress }, (s) => "width:" + s.progress + "%"),
            }),
          ],
        ),
      ]),
      View({ class: "text-sm text-[var(--FG-1)]" }, [
        Txt(computed({ message }, (s) => s.message)),
      ]),
      Show({ when: computed({ updateInfo }, (s) => !!s.updateInfo) }, [
        View(
          {
            class:
              "p-4 bg-[var(--BG-1)] rounded-lg border border-[var(--border)]",
          },
          [
            View({ class: "mb-2 text-[var(--GREEN)] font-bold" }, [
              Txt(
                computed({ updateInfo }, (s) =>
                  s.updateInfo ? "新版本: " + s.updateInfo.version : "",
                ),
              ),
            ]),
            View({ class: "text-sm text-[var(--FG-1)] mb-2" }, [
              Txt(
                computed({ updateInfo }, (s) =>
                  s.updateInfo
                    ? "当前版本: " + s.updateInfo.currentVersion
                    : "",
                ),
              ),
            ]),
            View(
              {
                class:
                  "text-sm p-3 bg-[var(--BG-2)] rounded whitespace-pre-wrap max-h-[200px] overflow-auto",
              },
              [
                Txt(
                  computed({ updateInfo }, (s) =>
                    s.updateInfo && s.updateInfo.releaseNotes
                      ? s.updateInfo.releaseNotes
                      : "暂无更新说明",
                  ),
                ),
              ],
            ),
          ],
        ),
      ]),
    ]),
  ]);
}
