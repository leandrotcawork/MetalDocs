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
    server: {
      host: "0.0.0.0",
      port: 4173,
      proxy: {
        "/api/v1": {
          target: proxyTarget,
          changeOrigin: false,
          secure: false,
        },
      },
    },
  };
});
