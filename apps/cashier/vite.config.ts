import { sveltekit } from "@sveltejs/kit/vite";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [sveltekit()],
  // @stery/kit is linked from the sibling stery-kit checkout (file: dependency), outside this root;
  // dedupe keeps its @sveltejs/kit imports on this app's copy, so error() and redirects stay recognised.
  resolve: { dedupe: ["@sveltejs/kit", "svelte"] },
  server: {
    port: 5174,
    fs: { allow: [".", "../../../stery-kit/web"] },
  },
});
