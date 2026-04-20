import { existsSync, readFileSync } from "node:fs";
import { resolve } from "node:path";
import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";

function readRepoEnvValue(name: string): string | undefined {
  const envPath = resolve(process.cwd(), "../../..", ".env");
  if (!existsSync(envPath)) {
    return undefined;
  }

  const lines = readFileSync(envPath, "utf8").split(/\r?\n/);
  for (const line of lines) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }
    const [key, ...rest] = trimmed.split("=");
    if (key !== name) {
      continue;
    }
    return rest.join("=").trim();
  }
  return undefined;
}

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const appPort = env.APP_PORT || process.env.APP_PORT || readRepoEnvValue("APP_PORT") || "8080";
  const proxyTarget = env.VITE_API_PROXY_TARGET || process.env.VITE_API_PROXY_TARGET || `http://127.0.0.1:${appPort}`;

  return {
    plugins: [react()],
    resolve: {
      alias: [
        { find: '@metaldocs/editor-ui', replacement: resolve(__dirname, '../../../packages/editor-ui/src/index.ts') },
        { find: '@metaldocs/mddm-layout-tokens', replacement: resolve(__dirname, '../../../shared/mddm-layout-tokens/index.ts') },
        { find: '@metaldocs/mddm-pagination-types', replacement: resolve(__dirname, '../../../shared/mddm-pagination-types/index.ts') },
        { find: '@metaldocs/shared-tokens', replacement: resolve(__dirname, '../../../packages/shared-tokens/src/index.ts') },
        { find: /^@eigenpal\/docx-js-editor\/styles\.css$/, replacement: resolve(__dirname, 'node_modules/@eigenpal/docx-js-editor/dist/styles.css') },
        { find: /^@eigenpal\/docx-js-editor\/react$/, replacement: resolve(__dirname, 'node_modules/@eigenpal/docx-js-editor/dist/react.mjs') },
        { find: /^@eigenpal\/docx-js-editor\/ui$/, replacement: resolve(__dirname, 'node_modules/@eigenpal/docx-js-editor/dist/ui.mjs') },
        { find: /^@eigenpal\/docx-js-editor\/core$/, replacement: resolve(__dirname, 'node_modules/@eigenpal/docx-js-editor/dist/core-reexport.mjs') },
        { find: /^@eigenpal\/docx-js-editor$/, replacement: resolve(__dirname, 'node_modules/@eigenpal/docx-js-editor/dist/index.mjs') },
        { find: 'jszip', replacement: resolve(__dirname, 'node_modules/jszip') },
        { find: 'fast-xml-parser', replacement: resolve(__dirname, 'node_modules/fast-xml-parser') },
      ],
    },
    server: {
      host: "0.0.0.0",
      port: 4173,
      proxy: {
        "/api/v1": {
          target: proxyTarget,
          changeOrigin: false,
          secure: false,
        },
        "/api/v2": {
          target: proxyTarget,
          changeOrigin: false,
          secure: false,
        },
        "/__gotenberg": {
          target: env.GOTENBERG_URL || process.env.GOTENBERG_URL || "http://localhost:3000",
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/__gotenberg/, ""),
        },
      },
    },
  };
});
