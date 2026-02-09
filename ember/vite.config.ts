import { defineConfig, loadEnv } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath, URL } from "node:url";

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), "");
  const backendOrigin = env.VITE_DEV_BACKEND_ORIGIN?.trim();

  return {
    plugins: [react(), tailwindcss()],
    server: {
      port: 5173,
      proxy: backendOrigin
        ? {
            "/api": backendOrigin,
            "/uploads": backendOrigin,
            "/ws": {
              target: backendOrigin,
              ws: true,
            },
          }
        : undefined,
    },
    resolve: {
      alias: {
        "@": "/src",
        "@pulse/drift": fileURLToPath(new URL("../drift/ts", import.meta.url)),
      },
    },
  };
});
