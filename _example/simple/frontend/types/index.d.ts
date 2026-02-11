// export * from "./global";
// export * from "./highlight";
// export * from "./editor";
// export * from "./timeless.umd";
// import * as ProsemirrorMod from './prosemirror';

declare global {
  interface Window {}

  // 如果需要在 Node.js 环境中使用
  //   namespace NodeJS {
  //     interface Global {
  //       MyLib: typeof MyLib;
  //     }
  //   }
}
