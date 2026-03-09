import {fileURLToPath, URL} from 'node:url'

import {ConfigEnv, defineConfig, loadEnv,} from 'vite'
import vue from '@vitejs/plugin-vue'
import vueDevTools from 'vite-plugin-vue-devtools'
import{ DevToolsOxlint } from'vite-plugin-devtools-oxlint'

export default defineConfig( (configEnv:  ConfigEnv) => {
    const {mode} = configEnv
    const env = loadEnv(mode, process.cwd(), '')
    // api 地址
    const api = env.VITE_APP_API_URL
    return {
        plugins: [
            vue(),
            vueDevTools(),
            DevToolsOxlint(),
        ],
        resolve: {
            alias: {
                '@': fileURLToPath(new URL('./src', import.meta.url)),
                '@vant/weapp': fileURLToPath(new URL('./node_modules/@vant/weapp/dist', import.meta.url))
            },
        },
        server: {
            proxy: {
                '/api': {
                    target: api,
                    changeOrigin: true,
                    rewrite: (path: string) => path.replace(/^\/api/, '/api')
                }
            }
        }
    }
})
