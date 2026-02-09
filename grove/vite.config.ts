import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";
import tailwindcss from "@tailwindcss/vite";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    port: 5174,
  },
  resolve: {
    alias: {
      "@": "/src",
      "@pulse/drift": fileURLToPath(new URL("../drift/ts", import.meta.url)),
    },
  },
});
