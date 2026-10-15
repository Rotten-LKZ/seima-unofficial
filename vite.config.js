import { defineConfig } from 'vite';

export default defineConfig({
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    cssMinify: 'esbuild',
    minify: 'esbuild',
  },
});
