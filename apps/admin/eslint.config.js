import eslint from "@eslint/js";
import svelte from "eslint-plugin-svelte";
import globals from "globals";
import ts from "typescript-eslint";
import svelteConfig from "./svelte.config.js";

export default ts.config(
  eslint.configs.recommended,
  ...ts.configs.recommended,
  ...svelte.configs["flat/recommended"],
  {
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
      },
    },
  },
  {
    files: ["**/*.svelte"],
    languageOptions: {
      parserOptions: {
        parser: ts.parser,
        svelteConfig,
      },
    },
  },
  {
    // no paths.base configured, so resolve() adds nothing
    rules: { "svelte/no-navigation-without-resolve": "off" },
  },
  {
    ignores: [".svelte-kit/", "build/", "node_modules/", "vite.config.ts.timestamp-*"],
  },
);
