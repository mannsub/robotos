import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  plugins: [svelte()],
  server: {
    proxy: {
      '/ws': { target: 'ws://localhost:8080', ws: true }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
