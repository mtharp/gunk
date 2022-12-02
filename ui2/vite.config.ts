import { fileURLToPath, URL } from "node:url";

import { defineConfig, loadEnv } from "vite";
import Compression from "vite-compression-plugin";
import { createHtmlPlugin } from "vite-plugin-html";
import vue from "@vitejs/plugin-vue";

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
        "~bootstrap": "bootstrap",
      },
    },
  };
});
