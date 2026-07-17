import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import preprocess from 'svelte-preprocess'

// https://vitejs.dev/config/
// TypeScript 支持：Vite 内置 esbuild 处理 .ts 文件，
// Svelte 3 组件内的 <script lang="ts"> 由 svelte-preprocess 预处理后再交给 Svelte 编译器。
export default defineConfig({
  plugins: [svelte({
    preprocess: preprocess()
  })],
  esbuild: {
    target: 'esnext'
  }
})
