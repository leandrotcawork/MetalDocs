import { defineConfig } from "vitest/config"
import { resolve } from "node:path"

export default defineConfig({
  resolve: {
    alias: {
      '@metaldocs/mddm-layout-tokens': resolve(__dirname, '../../shared/mddm-layout-tokens/index.ts'),
    },
  },
  test: {
    environment: "node"
  }
})
