import { defineConfig } from 'vite'
import { fileURLToPath } from 'node:url'
import { resolve } from 'node:path'

const here = fileURLToPath(new URL('.', import.meta.url))

// Two-project dev architecture:
//   - Shell Vite dev:      :7070  (base /app/) — this config
//   - Renderer Vite dev:   :5173  (base /app/renderer/default/)
// The shell dev server proxies /app/renderer/default/* to the renderer dev
// server so a single origin serves everything at http://localhost:7070/app/.
//
// Prod: each project builds to its own dist/ and Go embeds both via go:embed.
export default defineConfig({
  base: process.env.LIVEBOARD_BASE ?? '/app/',
  resolve: {
    alias: {
      '@shared': fileURLToPath(new URL('../shared/src', import.meta.url)),
    },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    sourcemap: true,
    rollupOptions: {
      input: {
        shell: resolve(here, 'index.html'),
        'renderer-stub': resolve(here, 'renderer-stub/index.html'),
      },
    },
  },
  server: {
    port: 7070,
    strictPort: true,
    proxy: {
      '/app/renderer/default': {
        target: 'http://localhost:5173',
        changeOrigin: true,
        ws: true,
      },
    },
    // When fronted by the Go server (make dev-adapter-test), the browser
    // connects via :7070. Point HMR back through that origin so the WS
    // upgrade is proxied by Go instead of hitting Vite's internal port.
    hmr: {
      clientPort: 7070,
    },
  },
})
