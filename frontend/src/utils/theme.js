// DaisyUI 主题变量
// 这个文件现在用于定义 DaisyUI 的主题变量和辅助函数

// 主题颜色
export const themeColors = {
  primary: '#2196f3',
  primaryHover: '#64b5f6',
  primaryPressed: '#1a237e',
  primarySuppl: '#64b5f6',
}

// 应用主题变量到根元素
export function applyThemeVariables(isDark) {
  const root = document.documentElement;
  
  // 设置基础变量
  root.style.setProperty('--primary', themeColors.primary);
  root.style.setProperty('--primary-hover', themeColors.primaryHover);
  
  // 根据主题设置其他变量
  if (isDark) {
    root.style.setProperty('--body-bg', '#1E1E1E');
    root.style.setProperty('--border-color', '#515151');
    root.style.setProperty('--card-bg', '#212121');
    root.style.setProperty('--dropdown-bg', '#272727');
    root.style.setProperty('--text-color', '#CECED0');
  } else {
    root.style.setProperty('--body-bg', '#FFFFFF');
    root.style.setProperty('--border-color', '#EAEAEA');
    root.style.setProperty('--card-bg', '#FAFAFA');
    root.style.setProperty('--dropdown-bg', '#FFFFFF');
    root.style.setProperty('--text-color', '#333333');
  }
}

// 辅助函数 - 获取主题变量
export function getThemeVariable(name, defaultValue = '') {
  return getComputedStyle(document.documentElement).getPropertyValue(name) || defaultValue;
}
