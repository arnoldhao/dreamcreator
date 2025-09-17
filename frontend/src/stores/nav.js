import { reactive } from 'vue'

// 导航选项
const navOptions = {
  DOWNLOAD: 'download',
  SUBTITLE: 'subtitle',
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
      icon: 'download',
    },
    {
      label: 'ribbon.subtitle',
      key: navOptions.SUBTITLE,
      icon: 'captions',
    },
  ],
  
  // 底部菜单选项（移除主题切换）
  bottomMenuOptions: [
    {
      label: 'bottom.settings',
      key: navOptions.SETTINGS,
      icon: 'settings',
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
  
  // 预留：后续可添加底部项操作
  
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
    
  }
}
