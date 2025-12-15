// 主题变量与 macOS 风格统一管理（已移除 DaisyUI 依赖）

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
    
    // Surfaces/Materials/Shadows/Dividers
    '--macos-surface': 'rgba(255, 255, 255, 0.75)',
    '--macos-surface-opaque': '#ffffff',
    '--macos-surface-blur': 'saturate(180%) blur(20px)',
    '--macos-shadow-1': '0 1px 2px rgba(0,0,0,0.08)',
    '--macos-shadow-2': '0 6px 18px rgba(0,0,0,0.06)',
    '--macos-divider-weak': 'rgba(60,60,67,0.15)',
    '--macos-divider-strong': 'rgba(60,60,67,0.29)',
    
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
    // translucent overlays
    '--macos-hover-translucent': 'rgba(0,0,0,0.06)',
    // unified sidebar background
    '--sidebar-bg': '#F2F2F7',
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
    
    // Surfaces/Materials/Shadows/Dividers
    '--macos-surface': 'rgba(22,22,24, 0.6)',
    '--macos-surface-opaque': '#1C1C1E',
    '--macos-surface-blur': 'saturate(180%) blur(20px)',
    '--macos-shadow-1': '0 1px 2px rgba(0,0,0,0.5)',
    '--macos-shadow-2': '0 10px 30px rgba(0,0,0,0.35)',
    '--macos-divider-weak': 'rgba(235,235,245,0.1)',
    '--macos-divider-strong': 'rgba(235,235,245,0.18)',
    
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
    // translucent overlays
    '--macos-hover-translucent': 'rgba(255,255,255,0.08)',
    // unified sidebar background
    '--sidebar-bg': '#1C1C1E',
  }
};

// 传统主题颜色（保持向后兼容）
export const themeColors = {
  primary: '#007AFF', // 使用 macOS 蓝色
  primaryHover: '#0056CC',
  primaryPressed: '#004499',
  primarySuppl: '#409CFF',
};

// Accent palettes for light/dark
const accentPalettes = {
  blue: {
    light: { base: '#007AFF', hover: '#0056CC', active: '#004499' },
    dark: { base: '#0A84FF', hover: '#409CFF', active: '#0A84FF' },
  },
  purple: {
    light: { base: '#8E44AD', hover: '#7A3797', active: '#5E2D73' },
    dark: { base: '#8E44AD', hover: '#A36CBC', active: '#8E44AD' },
  },
  pink: {
    light: { base: '#FF2D55', hover: '#E52248', active: '#C61D3E' },
    dark: { base: '#FF375F', hover: '#FF6B8A', active: '#FF375F' },
  },
  red: {
    light: { base: '#FF3B30', hover: '#E62F25', active: '#BF281F' },
    dark: { base: '#FF453A', hover: '#FF7A71', active: '#FF453A' },
  },
  orange: {
    light: { base: '#FF9500', hover: '#CC7700', active: '#995900' },
    dark: { base: '#FF9F0A', hover: '#FFB84D', active: '#FF9F0A' },
  },
  green: {
    light: { base: '#34C759', hover: '#28A745', active: '#1F7F34' },
    dark: { base: '#30D158', hover: '#63E089', active: '#30D158' },
  },
  teal: {
    light: { base: '#30B0C7', hover: '#2796A9', active: '#1E7482' },
    dark: { base: '#30B0C7', hover: '#64C8D8', active: '#30B0C7' },
  },
  indigo: {
    light: { base: '#5856D6', hover: '#4645AB', active: '#34347F' },
    dark: { base: '#5E5CE6', hover: '#8B89F0', active: '#5E5CE6' },
  },
};

function clamp01(x){ return Math.max(0, Math.min(1, x)) }
function hexToRgb(hex){
  const m = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(hex);
  if (!m) return null;
  return { r: parseInt(m[1], 16), g: parseInt(m[2], 16), b: parseInt(m[3], 16) };
}
function rgbToHex({r,g,b}){
  const toHex = (v)=> v.toString(16).padStart(2, '0')
  return `#${toHex(r)}${toHex(g)}${toHex(b)}`.toUpperCase()
}
function adjustRgb({r,g,b}, amt){
  // amt in [-1,1]; positive lightens, negative darkens
  const f = (v)=> Math.round(clamp01(v/255 + amt) * 255)
  return { r: f(r), g: f(g), b: f(b) }
}
function rgbToHsl({r,g,b}){
  r/=255; g/=255; b/=255;
  const max=Math.max(r,g,b), min=Math.min(r,g,b);
  let h, s, l=(max+min)/2;
  if(max===min){ h=s=0 } else {
    const d = max-min
    s = l>0.5 ? d/(2-max-min) : d/(max+min)
    switch(max){
      case r: h=(g-b)/d + (g<b?6:0); break;
      case g: h=(b-r)/d + 2; break;
      case b: h=(r-g)/d + 4; break;
    }
    h/=6
  }
  return { h: Math.round(h*360), s: Math.round(s*100), l: Math.round(l*100) }
}
function hexToHsl(hex){ const rgb = hexToRgb(hex); return rgb ? rgbToHsl(rgb) : null }
function luminance({r,g,b}){
  const a=[r,g,b].map(v=>{v/=255; return v<=0.03928?v/12.92:Math.pow((v+0.055)/1.055,2.4)})
  return 0.2126*a[0]+0.7152*a[1]+0.0722*a[2]
}

function applyShadcnAccentVars(root, baseHex) {
  const baseRgb = hexToRgb(baseHex)
  const baseHsl = hexToHsl(baseHex)
  if (!baseRgb || !baseHsl) return

  // shadcn tokens expect "H S% L%" (no `hsl()` wrapper)
  root.style.setProperty('--dc-shadcn-primary', `${baseHsl.h} ${baseHsl.s}% ${baseHsl.l}%`)
  root.style.setProperty('--dc-shadcn-ring', `${baseHsl.h} ${baseHsl.s}% ${baseHsl.l}%`)

  // Choose a readable foreground for the primary color.
  // Use shadcn defaults: near-black or near-white.
  const isLightForeground = luminance(baseRgb) < 0.5
  root.style.setProperty('--dc-shadcn-primary-foreground', isLightForeground ? '0 0% 98%' : '240 5.9% 10%')

  // Keep sidebar in sync with the same accent.
  root.style.setProperty('--dc-shadcn-sidebar-primary', `${baseHsl.h} ${baseHsl.s}% ${baseHsl.l}%`)
  root.style.setProperty('--dc-shadcn-sidebar-ring', `${baseHsl.h} ${baseHsl.s}% ${baseHsl.l}%`)
  root.style.setProperty('--dc-shadcn-sidebar-primary-foreground', isLightForeground ? '0 0% 98%' : '240 5.9% 10%')
}

// Apply only accent variables without changing scheme
export function applyAccent(name = 'blue', isDark = null) {
  const root = document.documentElement;
  let palette = accentPalettes[name];
  const dark = isDark == null ? detectSystemTheme() : !!isDark;
  if (!palette && typeof name === 'string' && name.startsWith('#')) {
    const baseRgb = hexToRgb(name)
    if (baseRgb) {
      // compute hover/active by simple lighten/darken tuned for scheme
      const hoverRgb = adjustRgb(baseRgb, dark ? 0.14 : -0.14)
      const activeRgb = adjustRgb(baseRgb, dark ? 0.0 : -0.28)
      const colors = { base: name.toUpperCase(), hover: rgbToHex(hoverRgb), active: rgbToHex(activeRgb) }
      root.style.setProperty('--macos-blue', colors.base);
      root.style.setProperty('--macos-blue-hover', colors.hover);
      root.style.setProperty('--macos-blue-active', colors.active);
      applyShadcnAccentVars(root, colors.base)
      themeColors.primary = colors.base;
      themeColors.primaryHover = colors.hover;
      themeColors.primarySuppl = colors.hover;
      return
    }
  }
  if (!palette) palette = accentPalettes.blue
  const colors = dark ? palette.dark : palette.light;
  root.style.setProperty('--macos-blue', colors.base);
  root.style.setProperty('--macos-blue-hover', colors.hover);
  root.style.setProperty('--macos-blue-active', colors.active);
  applyShadcnAccentVars(root, colors.base)
  // Backward compatible primary vars
  themeColors.primary = colors.base;
  themeColors.primaryHover = colors.hover;
  themeColors.primarySuppl = colors.hover;
}

// 统一的主题应用函数
export function applyMacosTheme(isDark = false) {
  const root = document.documentElement;
  const theme = isDark ? macosTheme.dark : macosTheme.light;
  
  // 应用所有 macOS 变量
  Object.entries(theme).forEach(([key, value]) => {
    root.style.setProperty(key, value);
  });
  // 设置标记属性以便样式根据明暗主题分支（不依赖 DaisyUI）
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
}

// 辅助函数 - 获取主题变量
export function getThemeVariable(name, defaultValue = '') {
  return getComputedStyle(document.documentElement).getPropertyValue(name) || defaultValue;
}

// 手动切换主题
export function toggleTheme() {
  const bg = getComputedStyle(document.documentElement).getPropertyValue('--macos-background').trim().toUpperCase();
  const isDark = bg === '#000000';
  applyMacosTheme(!isDark);
  return !isDark;
}
