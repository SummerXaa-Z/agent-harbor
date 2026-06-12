import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rolldownOptions: {
      output: {
        codeSplitting: {
          groups: [
            {
              name: "react-vendor",
              test: /node_modules\/(?:react|react-dom)\//
            },
            {
              name: "icons-vendor",
              test: /node_modules\/lucide-react\//
            }
          ]
        }
      }
    }
  },
  server: {
    host: "127.0.0.1",
    port: 5174
  },
  preview: {
    host: "127.0.0.1",
    port: 4174
  }
});
