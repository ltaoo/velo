/**
 * @file 本地存储服务
 */
const DEFAULT_CACHE_VALUES = {
  user: {
    id: "",
    username: "anonymous",
    email: "",
    token: "",
    avatar: "",
    expires_at: 0,
  },
  theme: "system",
};

const key = "global";
const e = globalThis.localStorage.getItem(key);
export const storage = new Timeless.StorageCore({
  key,
  defaultValues: DEFAULT_CACHE_VALUES,
  values: (() => {
    const prev = JSON.parse(e || "{}");
    return {
      ...prev,
    };
  })(),
  client: globalThis.localStorage,
});
