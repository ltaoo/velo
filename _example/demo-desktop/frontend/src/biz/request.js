// import { ListCore } from "@/domains/list";
// import { request_factory } from "@/domains/request/utils";
// import { Result } from "@/domains/result";

export const request = Timeless.rutil.request_factory({
  hostnames: {},
  process(r) {
    if (r.error) {
      return Timeless.Result.Err(r.error);
    }
    const { code, msg, data } = r.data;
    if (code !== 0) {
      return Timeless.Result.Err(msg, code, data);
    }
    // console.log("[common]", JSON.stringify(data));
    return Timeless.Result.Ok(data);
  },
});

Timeless.ListCore.commonProcessor = (originalResponse) => {
  if (originalResponse === null) {
    return {
      dataSource: [],
      page: 1,
      pageSize: 20,
      total: 0,
      noMore: false,
      empty: false,
      error: null,
    };
  }
  try {
    const data = originalResponse.data || originalResponse;
    // console.log("[BIZ]commonProcessor", data.list);
    const {
      list,
      page,
      page_size,
      total,
      noMore,
      no_more,
      has_more,
      next_marker,
    } = data;
    const result = {
      dataSource: list,
      page,
      pageSize: page_size,
      total,
      empty: false,
      noMore: false,
      error: null,
      next_marker,
    };
    if (total <= page_size * page) {
      result.noMore = true;
    }
    if (no_more !== undefined) {
      result.noMore = no_more;
    }
    if (has_more !== undefined) {
      result.noMore = !has_more;
    }
    if (noMore !== undefined) {
      result.noMore = noMore;
    }
    if (next_marker === null) {
      result.noMore = true;
    }
    if (list.length === 0 && page === 1) {
      result.empty = true;
    }
    if (list.length === 0) {
      result.noMore = true;
    }
    // console.log("[STORE]ListCore.commonProcessor", data, result);
    return result;
  } catch (error) {
    return {
      dataSource: [],
      page: 1,
      pageSize: 20,
      total: 0,
      noMore: false,
      empty: false,
      error: new Timeless.BizError([`${error.message}`]),
      // next_marker: "",
    };
  }
};
