import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    include: ["src/**/*.test.{ts,tsx}"],
    environment: "jsdom",
    globals: false,
    // "forks" isolates each test file in a child process. On Windows, forking
    // many processes simultaneously causes STACK_TRACE_ERROR for some files
    // because child_process.fork() fails under high concurrency. Capping at 4
    // concurrent forks eliminates this without sacrificing isolation.
    pool: "forks",
    poolOptions: {
      forks: {
        minForks: 1,
        maxForks: 4,
      },
    },
  },
});
