import { defineConfig } from "vitest/config";
import path from "path";

const dashboardSrc = path.resolve(__dirname, "./src");
const dashboardModules = path.resolve(__dirname, "./node_modules");
const projectRoot = path.resolve(__dirname, "../..");

export default defineConfig({
  test: {
    environment: "jsdom",
    globals: true,
    setupFiles: ["./services/dashboard/src/test-setup.ts"],
    root: projectRoot,
    include: [
      "tests/unit/dashboard/**/*_test.{ts,tsx}",
      "tests/unit/dashboard/**/*.{test,spec}.{ts,tsx}",
      "services/dashboard/src/**/*_test.{ts,tsx}",
      "services/dashboard/src/**/*.{test,spec}.{ts,tsx}",
    ],
  },
  resolve: {
    alias: {
      "@": dashboardSrc,
      "react": path.join(dashboardModules, "react"),
      "react-dom": path.join(dashboardModules, "react-dom"),
      "react/jsx-dev-runtime": path.join(dashboardModules, "react/jsx-dev-runtime"),
      "react/jsx-runtime": path.join(dashboardModules, "react/jsx-runtime"),
      "@testing-library/react": path.join(dashboardModules, "@testing-library/react"),
      "@testing-library/jest-dom": path.join(dashboardModules, "@testing-library/jest-dom"),
    },
  },
  esbuild: {
    jsx: "automatic",
  },
  server: {
    fs: {
      allow: [projectRoot, __dirname],
    },
  },
});
