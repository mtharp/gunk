import { fileURLToPath, URL } from "node:url";

import { defineConfig, loadEnv } from "vite";
import { createHtmlPlugin } from "vite-plugin-html";
import vue from "@vitejs/plugin-vue";
import Components from "unplugin-vue-components/vite";
import Icons from "unplugin-icons/vite";
import IconsResolver from "unplugin-icons/resolver";
import { VueUseComponentsResolver } from "unplugin-vue-components/resolvers";
import { BootstrapVueNextResolver } from "bootstrap-vue-next";

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  return {
    build: {
      reportCompressedSize: false,
      sourcemap: true,
    },
    server: {
      hmr: {
        path: "/websocket",
      },
    },
    plugins: [
      vue(),
      Icons({ compiler: "vue3" }),
      Components({
        dts: true,
        resolvers: [
          BootstrapVueNextResolver(),
          IconsResolver(),
          VueUseComponentsResolver(),
        ],
      }),
      createHtmlPlugin({
        minify: false,
        entry: "/src/main.ts",
        inject: {
          data: { title: env.VITE_APP_TITLE },
        },
      }),
    ],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
  };
});
