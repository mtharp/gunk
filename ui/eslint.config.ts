import { globalIgnores } from "eslint/config";
import pluginVue from "eslint-plugin-vue";
import skipFormattingConfig from "@vue/eslint-config-prettier/skip-formatting";
import {
  defineConfigWithVueTs,
  vueTsConfigs,
} from "@vue/eslint-config-typescript";

export default defineConfigWithVueTs(
  globalIgnores(["dest/**", ".pnp*", ".yarn/**"]),
  pluginVue.configs["flat/essential"],
  vueTsConfigs.recommended,
  skipFormattingConfig,
);
