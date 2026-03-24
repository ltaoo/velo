/**
 * @file HTTP 客户端
 */
export const client = new Timeless.HttpClientCore({
  headers: {
    "Content-Type": "application/json",
  },
});

function provide_box() {
  client.fetch = async (options) => {
    const { id, method, url, data, headers } = options;
    try {
      console.log("[store]client.fetch", url, data);
      // @ts-ignore
      const r = await invoke(url, {
        method,
        headers: (() => {
          if (!headers) {
            return null;
          }
          return Object.keys(headers)
            .map((k) => {
              const v = headers[k];
              return {
                [k]: [v],
              };
            })
            .reduce((a, b) => {
              return { ...a, ...b };
            }, {});
        })(),
        args: data,
      });
      if (!r) {
        throw new Error("Missing the response");
      }
      console.log("[store]client.fetch result", r);
      return Promise.resolve({ data: r ?? {} });
    } catch (err) {
      throw err;
    }
  };
  client.cancel = (id) => {
    return Timeless.Result.Ok(null);
  };
}
function provide_http() {
  TimelessWeb.provide_http_client(client);
}

provide_http();
