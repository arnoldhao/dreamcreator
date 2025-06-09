import { createPinia } from 'pinia'
import { createApp, nextTick } from 'vue'
import App from './App.vue'
import './index.css'
import './styles/style.scss'
import './styles/macos-components.scss'
import { i18n } from '@/utils/i18n.js'
import { initThemeSystem } from '@/utils/theme.js'
import { setupDiscreteApi } from '@/utils/discrete.js'
import usePreferencesStore from 'stores/preferences.js'
import { loadEnvironment } from '@/utils/platform.js'
import { OhVueIcon, addIcons } from 'oh-vue-icons'
import { useDtStore } from '@/handlers/downtasks'

// 导入所有图标包
import * as AiIcons from 'oh-vue-icons/icons/ai'
import * as BiIcons from 'oh-vue-icons/icons/bi'
import * as CoIcons from 'oh-vue-icons/icons/co'
import * as CiIcons from 'oh-vue-icons/icons/ci'
import * as FaIcons from 'oh-vue-icons/icons/fa'
import * as FcIcons from 'oh-vue-icons/icons/fc'
import * as FiIcons from 'oh-vue-icons/icons/fi'
import * as GiIcons from 'oh-vue-icons/icons/gi'
import * as HiIcons from 'oh-vue-icons/icons/hi'
import * as IoIcons from 'oh-vue-icons/icons/io'
import * as LaIcons from 'oh-vue-icons/icons/la'
import * as MdIcons from 'oh-vue-icons/icons/md'
import * as OiIcons from 'oh-vue-icons/icons/oi'
import * as PiIcons from 'oh-vue-icons/icons/pi'
import * as PrIcons from 'oh-vue-icons/icons/pr'
import * as PxIcons from 'oh-vue-icons/icons/px'
import * as RiIcons from 'oh-vue-icons/icons/ri'
import * as SiIcons from 'oh-vue-icons/icons/si'
import * as ViIcons from 'oh-vue-icons/icons/vi'
import * as WiIcons from 'oh-vue-icons/icons/wi'

// 合并所有图标
const allIcons = Object.values({
  ...AiIcons,
  ...BiIcons,
  ...CoIcons,
  ...CiIcons,
  ...FaIcons,
  ...FcIcons,
  ...FiIcons,
  ...GiIcons,
  ...HiIcons,
  ...IoIcons,
  ...LaIcons,
  ...MdIcons,
  ...OiIcons,
  ...PiIcons,
  ...PrIcons,
  ...PxIcons,
  ...RiIcons,
  ...SiIcons,
  ...ViIcons,
  ...WiIcons
})

// 注册所有图标
addIcons(...allIcons)

async function setupApp() {
    // 初始化主题系统
    initThemeSystem()
    const app = createApp(App)
    app.use(i18n)
    app.use(createPinia())

    // Register OhVueIcon component globally
    app.component("v-icon", OhVueIcon);

    // 初始化全局WS通信状态管理
    const dtStore = useDtStore()
    dtStore.init()

    await loadEnvironment()
    const prefStore = usePreferencesStore()
    await prefStore.loadPreferences()
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
