import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Icons from 'unplugin-icons/vite'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import Components from 'unplugin-vue-components/vite'
// 移除这行：import tailwindcss from 'tailwindcss'

const rootPath = new URL('.', import.meta.url).pathname

export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            imports: [],
        }),
        Components({
            resolvers: [NaiveUiResolver()],
        }),
        Icons(),
    ],
    resolve: {
        alias: {
            '@': rootPath + 'src',
            stores: rootPath + 'src/stores',
            wailsjs: rootPath + 'wailsjs',
        },
    },
    define: {
        __VUE_PROD_HYDRATION_MISMATCH_DETAILS__: 'false',
    },
})
