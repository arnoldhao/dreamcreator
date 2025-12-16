import { reactive, computed } from 'vue'

// settings 页面选项常量
export const settingsOptions = {
  GENERAL: 'general',
  APPEARANCE: 'appearance',
  STORAGE: 'storage',
  DEPENDENCIES: 'dependencies',
  COOKIES: 'cookies',
  PROVIDERS: 'providers',
  LLM_ASSETS: 'llm_assets',
  ACKNOWLEDGEMENTS: 'acknowledgements',
  ABOUT: 'about'
}

// 创建一个单例 state
const state = reactive({
  currentPage: settingsOptions.GENERAL
})

// 方法
const actions = {
  setPage(page) {
    if (Object.values(settingsOptions).includes(page)) {
      state.currentPage = page
    }
  }
}

// 导出 settings 管理器
export default function useSettingsStore() {
  // 确保返回相同的 state 实例
  return {
    settingsOptions,
    get currentPage() {
      return state.currentPage || settingsOptions.GENERAL
    },
    setPage: actions.setPage
  }
}
