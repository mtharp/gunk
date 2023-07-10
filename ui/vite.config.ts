import { fileURLToPath, URL } from "node:url";

import { defineConfig, loadEnv } from "vite";
import Compression from "vite-compression-plugin";
import { createHtmlPlugin } from "vite-plugin-html";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import { BootstrapVueNextResolver } from "unplugin-vue-components/resolvers";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  return {
    server: {
      hmr: {
        path: "/websocket",
      },
    },
    plugins: [
      vue(),
      Components({
        resolvers: [BootstrapVueNextResolver()],
      }),
      createHtmlPlugin({
        minify: false,
        entry: "/src/main.ts",
        inject: {
          data: { title: env.VITE_APP_TITLE },
        },
      }),
      Compression({
        algorithm: "brotliCompress",
        exclude: ["**/*.png"],
      }),
    ],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
  };
});
