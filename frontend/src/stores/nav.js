import { reactive } from 'vue'

// 导航选项
const navOptions = {
  DOWNLOAD: 'download',
  SETTINGS: 'settings'
}

// 创建响应式状态
const state = reactive({
  // 当前激活的导航项
  currentNav: navOptions.DOWNLOAD,
  
  // 主菜单选项
  menuOptions: [
    {
      label: 'ribbon.download',
      key: navOptions.DOWNLOAD,
      icon: 'ri-download-cloud-line',
    },
  ],
  
  // 底部菜单选项
  bottomMenuOptions: [
    {
      label: 'bottom.theme',
      key: 'theme',
      icon: 'ri-sun-line', // 默认图标，会根据主题动态变化
    },
    {
      label: 'bottom.settings',
      key: navOptions.SETTINGS,
      icon: 'ri-settings-3-line',
    },
  ]
})

// 导航相关方法
const actions = {
  // 设置当前导航
  setNav(nav) {
    if (Object.values(navOptions).includes(nav)) {
      state.currentNav = nav
    }
  },
  
  // 更新主题图标
  updateThemeIcon(isDark) {
    const themeOption = state.bottomMenuOptions.find(option => option.key === 'theme')
    if (themeOption) {
      themeOption.icon = isDark ? 'ri-moon-line' : 'ri-sun-line'
    }
  }
}

// 导出导航管理器
export default function useNavStore() {
  return {
    // 导航选项常量
    navOptions,
    
    // 菜单选项
    menuOptions: state.menuOptions,
    bottomMenuOptions: state.bottomMenuOptions,
    
    // 当前导航，可读写
    get currentNav() {
      return state.currentNav
    },
    set currentNav(value) {
      actions.setNav(value)
    },
    
    // 操作方法
    setNav: actions.setNav,
    updateThemeIcon: actions.updateThemeIcon
  }
}