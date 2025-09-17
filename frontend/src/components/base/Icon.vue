<template>
  <component :is="asSpan ? 'span' : 'i'" v-bind="a11yAttrs" :class="wrapperClass">
    <span v-if="svg" v-html="svg"></span>
    <span v-else v-html="fallbackSvg"></span>
  </component>
</template>

<script setup>
import { computed } from 'vue'
import { ICON_MAP } from '@/icons/registry.js'

const props = defineProps({
  // 语义名，如 'search' | 'refresh' | 'status-success'
  name: { type: String, required: true },
  // 可选尺寸 token（也可以直接用 w-4 h-4 类）
  size: { type: String, default: '' }, // xs | sm | md | lg
  // 可选：明确传入 title/aria-label 用于可访问性
  title: { type: String, default: '' },
  ariaLabel: { type: String, default: '' },
  asSpan: { type: Boolean, default: true },
  class: { type: [String, Array, Object], default: '' },
})

// 载入所有 open-symbols SVG（以 raw 字符串形式）
const modules = import.meta.glob('/src/assets/open-symbols/*.svg', { eager: true, query: '?raw', import: 'default' })
const osSvgs = Object.fromEntries(
  Object.entries(modules).map(([path, raw]) => {
    const base = path.split('/').pop().replace(/\.svg$/i, '')
    return [base, String(raw)]
  })
)

const osName = computed(() => ICON_MAP[props.name])
const raw = computed(() => (osName.value ? osSvgs[osName.value] : null))

// 统一规范化：去掉固定宽高、使用 currentColor
const svg = computed(() => {
  if (!raw.value) return ''
  let s = raw.value
  s = s.replace(/\s(width|height)="[^"]*"/g, '')
  // 保留 fill="none"，其余强制为 currentColor
  s = s.replace(/fill="(?!none)[^"]*"/g, 'fill="currentColor"')
  s = s.replace(/stroke="(?!none)[^"]*"/g, 'stroke="currentColor"')
  // 确保根节点提供默认的 currentColor（影响未声明 fill/stroke 的路径）
  s = s.replace(/<svg\b([^>]*)>/, (m, attrs) => {
    const hasFill = /\sfill=/.test(attrs)
    const hasStroke = /\sstroke=/.test(attrs)
    let injected = attrs
    if (!hasFill) injected += ' fill="currentColor"'
    if (!hasStroke) injected += ' stroke="currentColor"'
    return `<svg${injected}>`
  })
  return s
})

// 无回退到第三方库；若缺失则显示占位 SVG 并在控制台提示
if (import.meta.env.DEV) {
  if (!ICON_MAP[/* @vite-ignore */props.name]) {
    console.warn('[Icon] Unknown semantic icon name:', props.name)
  }
}

// Fallback：一个简洁的方形占位符（以避免构建失败）
const fallbackSvg = `
  <svg viewBox="0 0 24 24" width="1em" height="1em" fill="currentColor" aria-hidden="true">
    <rect x="4" y="4" width="16" height="16" rx="3" ry="3" fill="currentColor" opacity="0.15" />
    <path d="M8 12h8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
    <path d="M12 8v8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
  </svg>
`

const a11yAttrs = computed(() => {
  const attrs = {}
  if (props.title || props.ariaLabel) {
    attrs.role = 'img'
    if (props.title) attrs.title = props.title
    if (props.ariaLabel) attrs['aria-label'] = props.ariaLabel
  } else {
    attrs['aria-hidden'] = 'true'
  }
  return attrs
})

const wrapperClass = computed(() => [
  'sr-icon',
  props.size ? `sr-icon-${props.size}` : null,
  props.class || null,
])
</script>

<style scoped>
.sr-icon { display: inline-flex; line-height: 1; vertical-align: middle; align-items: center; justify-content: center; }
.sr-icon > span :deep(svg) { display: block; width: 100%; height: 100%; }
/* 尺寸 token（也可直接用 Tailwind 的 w/h 控制容器尺寸） */
.sr-icon-xs { font-size: var(--fs-sub); }
.sr-icon-sm { font-size: var(--fs-title); }
.sr-icon-md { font-size: 16px; }
.sr-icon-lg { font-size: 20px; }
</style>
