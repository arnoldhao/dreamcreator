// DaisyUI 主题变量和 macOS 风格统一管理

// macOS 风格主题定义
export const macosTheme = {
  light: {
    // macOS 系统颜色
    '--macos-blue': '#007AFF',
    '--macos-blue-hover': '#0056CC',
    '--macos-blue-active': '#004499',
    '--macos-gray': '#F2F2F7',
    '--macos-gray-hover': '#E5E5EA',
    '--macos-gray-active': '#D1D1D6',
    '--macos-text-primary': '#000000',
    '--macos-text-secondary': '#3C3C43',
    '--macos-text-tertiary': '#3C3C4399',
    '--macos-border': '#C6C6C8',
    '--macos-separator': '#C6C6C8',
    '--macos-background': '#FFFFFF',
    '--macos-background-secondary': '#F2F2F7',
    
    // 状态颜色
    '--macos-success-bg': 'rgba(52, 199, 89, 0.15)',
    '--macos-success-text': '#34c759',
    '--macos-warning-bg': 'rgba(255, 149, 0, 0.15)',
    '--macos-warning-text': '#ff9500',
    '--macos-danger-bg': 'rgba(255, 69, 58, 0.15)',
    '--macos-danger-text': '#ff453a',
    
    // 滚动条
    '--macos-scrollbar-thumb': 'rgba(0, 0, 0, 0.2)',
    '--macos-scrollbar-thumb-hover': 'rgba(0, 0, 0, 0.3)',
  },
  
  dark: {
    // macOS 暗色模式
    '--macos-blue': '#0A84FF',
    '--macos-blue-hover': '#409CFF',
    '--macos-blue-active': '#0A84FF',
    '--macos-gray': '#1C1C1E',
    '--macos-gray-hover': '#2C2C2E',
    '--macos-gray-active': '#3A3A3C',
    '--macos-text-primary': '#FFFFFF',
    '--macos-text-secondary': '#EBEBF5',
    '--macos-text-tertiary': '#EBEBF599',
    '--macos-border': '#38383A',
    '--macos-separator': '#38383A',
    '--macos-background': '#000000',
    '--macos-background-secondary': '#1C1C1E',
    
    // 状态颜色
    '--macos-success-bg': 'rgba(52, 199, 89, 0.15)',
    '--macos-success-text': '#30d158',
    '--macos-warning-bg': 'rgba(255, 149, 0, 0.15)',
    '--macos-warning-text': '#ff9f0a',
    '--macos-danger-bg': 'rgba(255, 69, 58, 0.15)',
    '--macos-danger-text': '#ff453a',
    
    // 滚动条
    '--macos-scrollbar-thumb': 'rgba(255, 255, 255, 0.2)',
    '--macos-scrollbar-thumb-hover': 'rgba(255, 255, 255, 0.3)',
  }
};

// 传统主题颜色（保持向后兼容）
export const themeColors = {
  primary: '#007AFF', // 使用 macOS 蓝色
  primaryHover: '#0056CC',
  primaryPressed: '#004499',
  primarySuppl: '#409CFF',
};

// 统一的主题应用函数
export function applyMacosTheme(isDark = false) {
  const root = document.documentElement;
  const theme = isDark ? macosTheme.dark : macosTheme.light;
  
  // 应用所有 macOS 变量
  Object.entries(theme).forEach(([key, value]) => {
    root.style.setProperty(key, value);
  });
  
  // 设置 DaisyUI 主题属性
  document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light');
  
  // 兼容旧的变量名
  root.style.setProperty('--primary', themeColors.primary);
  root.style.setProperty('--primary-hover', themeColors.primaryHover);
  
  // 映射到旧的变量名（向后兼容）
  root.style.setProperty('--body-bg', theme['--macos-background']);
  root.style.setProperty('--border-color', theme['--macos-border']);
  root.style.setProperty('--card-bg', theme['--macos-background-secondary']);
  root.style.setProperty('--dropdown-bg', theme['--macos-background']);
  root.style.setProperty('--text-color', theme['--macos-text-primary']);
}

// 检测系统主题偏好
export function detectSystemTheme() {
  return window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
}

// 初始化主题系统
export function initThemeSystem() {
  const isDark = detectSystemTheme();
  applyMacosTheme(isDark);
  
  // 监听系统主题变化
  if (window.matchMedia) {
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      applyMacosTheme(e.matches);
    });
  }
}

// 辅助函数 - 获取主题变量
export function getThemeVariable(name, defaultValue = '') {
  return getComputedStyle(document.documentElement).getPropertyValue(name) || defaultValue;
}

// 手动切换主题
export function toggleTheme() {
  const currentTheme = document.documentElement.getAttribute('data-theme');
  const isDark = currentTheme === 'dark';
  applyMacosTheme(!isDark);
  return !isDark;
}