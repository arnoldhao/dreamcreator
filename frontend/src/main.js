import { createPinia } from 'pinia'
import { createApp, nextTick } from 'vue'
import App from './App.vue'
import './index.css'
import './styles/style.scss'
import './styles/macos-tokens.scss'
import './styles/macos-components.scss'
import { i18n } from '@/utils/i18n.js'
import { initThemeSystem, applyUIStyle } from '@/utils/theme.js'
import { setupDiscreteApi } from '@/utils/discrete.js'
import usePreferencesStore from 'stores/preferences.js'
import useDependenciesStore from 'stores/dependencies.js'
import { loadEnvironment } from '@/utils/platform.js'
import Icon from '@/components/base/Icon.vue'
import WebSocketService from '@/services/websocket'
import { useDtStore } from '@/handlers/downtasks'

// 已迁移为语义化 + 本地 open-symbols，无需 oh-vue-icons 注册

async function setupApp() {
    // 初始化主题系统
    initThemeSystem()
    const app = createApp(App)
    app.use(i18n)
    app.use(createPinia())

    // 全局注册：统一使用语义化 <Icon name="..." />；同时别名 v-icon 指向 Icon 兼容历史调用
    app.component("v-icon", Icon);
    app.component("Icon", Icon);

    // 1. 启动永不断开的WebSocket连接
    WebSocketService.startAutoReconnect();

    // 等待初始连接建立（最多等待5秒）
    let connectionAttempts = 0;
    while (!WebSocketService.isConnected() && connectionAttempts < 50) {
        await new Promise(resolve => setTimeout(resolve, 100));
        connectionAttempts++;
    }

    if (WebSocketService.isConnected()) {
        console.log('WebSocket connected successfully');
    } else {
        console.warn('WebSocket initial connection failed, but auto-reconnect is active');
    }
    // 开启自动重连
    WebSocketService.startAutoReconnect()

    // 2.初始化全局WS通信状态管理
    const dtStore = useDtStore()
    dtStore.init()

    // 3.初始化依赖store的WebSocket监听
    const dependenciesStore = useDependenciesStore()
    dependenciesStore.setupWebSocketListeners();

    await loadEnvironment()
    const prefStore = usePreferencesStore()
    await prefStore.loadPreferences()
    try { applyUIStyle(prefStore?.general?.uiStyle || 'frosted') } catch {}
    await setupDiscreteApi()
    app.config.errorHandler = (err, instance, info) => {
        nextTick().then(() => {
            try {
                const content = err.toString()
                $notification.error(content, {
                    title: i18n.global.t('common.error'),
                    meta: i18n.global.t('message.console_tip'),
                })
                console.error(err)
            } catch (e) { }
        })
    }

    app.mount('#app')
}

setupApp()
