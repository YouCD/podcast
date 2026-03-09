// src/vue-audio-player.d.ts
declare module "@liripeng/vue-audio-player" {
  import { DefineComponent } from "vue";

  const component: DefineComponent<
    Record<string, unknown>,
    Record<string, unknown>,
    unknown
  >;
  export default component;
}
declare module "@puhaha/vite-plugin-upload-oss" {
  import { Plugin } from "vite";

  interface QiniuPluginOptions {
    accessKey: string;
    secretKey: string;
    bucket: string;
    domain: string;
    zone?: string;
    overwrite?: boolean;
    ignore?: string[];
    prefix?: string;
    // 可以根据实际需要添加更多配置项
  }

  export function qiniuPlugin(options: QiniuPluginOptions): Plugin;
}
declare module "*.vue" {
  import { DefineComponent } from "vue";
  const component: DefineComponent<{}, {}, any>;
  export default component;
}
