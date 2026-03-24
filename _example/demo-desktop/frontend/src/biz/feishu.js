// 对应 feishu.go 中的常量配置
const AppID = "cli_a9f2fe4087f85bc2";
const AppSecret = "cy53HgngvMqtdfB3BXrJEc22d6WcE7hm";
const BaseToken = "S3lpbn20JaPs9XsiQBNcKPxNnXe"; // 多维表格的 base_token
const TableID = "tblnGTqA5vEAX2S3";            // 数据表的 table_id

/**
 * 飞书 API 客户端
 * 用于处理认证和请求发送
 */
class FeishuClient {
  constructor(appId, appSecret) {
    this.appId = appId;
    this.appSecret = appSecret;
    this.token = null;
    this.expireTime = 0;
  }

  /**
   * 获取 tenant_access_token
   * 对应 Go SDK 的内部 token 管理
   */
  async getAccessToken() {
    // 检查缓存是否有效
    if (this.token && Date.now() < this.expireTime) {
      return this.token;
    }

    const url = "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal";
    try {
      const resp = await fetch(url, {
        method: "POST",
        headers: {
          "Content-Type": "application/json; charset=utf-8",
        },
        body: JSON.stringify({
          app_id: this.appId,
          app_secret: this.appSecret,
        }),
      });

      const data = await resp.json();
      if (data.code !== 0) {
        throw new Error(`获取 AccessToken 失败: code=${data.code}, msg=${data.msg}`);
      }

      this.token = data.tenant_access_token;
      // expire 是秒，提前 60 秒过期以确保安全
      this.expireTime = Date.now() + (data.expire - 60) * 1000;
      return this.token;
    } catch (err) {
      console.error("获取 AccessToken 异常:", err);
      throw err;
    }
  }

  /**
   * 发送 API 请求
   */
  async request(method, path, body = null, params = null) {
    const token = await this.getAccessToken();
    let url = `https://open.feishu.cn/open-apis${path}`;
    
    if (params) {
      const queryString = new URLSearchParams(params).toString();
      url += `?${queryString}`;
    }

    const options = {
      method,
      headers: {
        Authorization: `Bearer ${token}`,
        "Content-Type": "application/json; charset=utf-8",
      },
    };

    if (body) {
      options.body = JSON.stringify(body);
    }

    try {
      const resp = await fetch(url, options);
      const data = await resp.json();
      return data;
    } catch (err) {
      console.error(`API 请求失败 [${method} ${path}]:`, err);
      throw err;
    }
  }
}

/**
 * 检查多维表格元数据
 * 对应 feishu.go 中的 checkBaseInfo
 */
export async function checkBaseInfo(client, baseToken) {
  console.log("\n=== 检查多维表格元数据 ===");
  
  // GET /bitable/v1/apps/:app_token
  const path = `/bitable/v1/apps/${baseToken}`;
  const resp = await client.request("GET", path);

  if (resp.code !== 0) {
    console.error(`获取多维表格信息失败: code=${resp.code}, msg=${resp.msg}`);
    if (resp.code === 91403 || resp.code === 403) {
      console.log("提示: 请确保已将应用机器人添加为多维表格的协作者（点击多维表格右上角'...' -> '添加协作者' -> 搜索应用名称）。");
    }
    return;
  }

  console.log(`成功连接到多维表格: ${resp.data.app.name} (版本: ${resp.data.app.revision})`);
}

/**
 * 测试基础权限
 * 对应 feishu.go 中的 testBasicPermissions
 */
export async function testBasicPermissions(client, baseToken, tableID) {
  console.log("=== 测试基础权限 ===");

  // 1. 先尝试列出记录（读权限）
  // GET /bitable/v1/apps/:app_token/tables/:table_id/records
  const listPath = `/bitable/v1/apps/${baseToken}/tables/${tableID}/records`;
  const listResp = await client.request("GET", listPath, null, { page_size: 10 });

  if (listResp.code !== 0) {
    console.error(`读权限测试失败: code=${listResp.code}, msg=${listResp.msg}`);
  } else {
    console.log(`读权限正常，获取到 ${listResp.data.items ? listResp.data.items.length : 0} 条记录`);
  }

  // 2. 测试创建权限
  // POST /bitable/v1/apps/:app_token/tables/:table_id/records
  const createPath = `/bitable/v1/apps/${baseToken}/tables/${tableID}/records`;
  const createResp = await client.request("POST", createPath, {
    fields: {
      "标题": "权限测试记录"
    }
  });

  if (createResp.code !== 0) {
    console.error(`写权限测试失败: code=${createResp.code}, msg=${createResp.msg}`);
    console.log("详细错误信息:", createResp);
  } else {
    console.log("写权限正常");
  }
}

/**
 * 导入赞赏者数据
 * 对应 feishu.go 中的 importSponsors
 * 
 * @param {FeishuClient} client 
 * @param {string} baseToken 
 * @param {string} tableID 
 * @param {Object} configData JSON 数据对象 (对应 sponsors.json 的内容)
 */
export async function importSponsors(client, baseToken, tableID, configData) {
  // 注意：在 JS 版本中，我们假设 configData 已经是解析好的 JSON 对象
  // 如果需要读取文件，请在调用此函数前处理文件读取
  
  let count = 0;
  
  if (!configData || !configData.sections) {
    console.error("配置数据格式错误: 缺少 sections");
    return;
  }

  for (const section of configData.sections) {
    if (!section.list) continue;
    
    for (const item of section.list) {
      // 构建字段
      const fields = {
        "赞赏者名称": item.text,
        "赞赏者头像链接": item.image
      };

      // 创建记录
      const path = `/bitable/v1/apps/${baseToken}/tables/${tableID}/records`;
      const resp = await client.request("POST", path, { fields });

      if (resp.code !== 0) {
        console.error(`创建记录失败 [${item.text}]: ${resp.msg}`);
        continue;
      }

      console.log(`成功创建记录: ${item.text}`);
      count++;
    }
  }
  
  console.log(`共导入 ${count} 条记录`);
}

// 导出默认配置和客户端创建函数
export const config = {
  AppID,
  AppSecret,
  BaseToken,
  TableID
};

export function createClient() {
  return new FeishuClient(AppID, AppSecret);
}

// 主函数示例 (对应 feishu.go 的 main)
export async function main(sponsorsData) {
  const client = createClient();

  // 0. 检查多维表格是否可访问
  // await checkBaseInfo(client, BaseToken);

  // 1. 测试基础权限
  // await testBasicPermissions(client, BaseToken, TableID);

  // 2. 导入赞赏者数据
  if (sponsorsData) {
    await importSponsors(client, BaseToken, TableID, sponsorsData);
  }
}
