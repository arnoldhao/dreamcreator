import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import AutoImport from 'unplugin-auto-import/vite'
import Icons from 'unplugin-icons/vite'
import { FileSystemIconLoader } from 'unplugin-icons/loaders'
import { NaiveUiResolver } from 'unplugin-vue-components/resolvers'
import Components from 'unplugin-vue-components/vite'
// 移除这行：import tailwindcss from 'tailwindcss'

const rootPath = new URL('.', import.meta.url).pathname

export default defineConfig({
    plugins: [
        vue(),
        AutoImport({
            // Provide common auto-imports to silence warnings and help DX
            imports: ['vue', 'vue-i18n', 'pinia'],
        }),
        Components({
            resolvers: [NaiveUiResolver()],
        }),
        Icons({
            // Optional: custom collection for open-symbols (static usage: <i-os:gearshape />)
            customCollections: {
                os: FileSystemIconLoader('src/assets/open-symbols', svg =>
                    svg
                        .replace(/fill="[^"]*"/g, 'fill="currentColor"')
                        .replace(/stroke="[^"]*"/g, 'stroke="currentColor"')
                ),
            },
        }),
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
