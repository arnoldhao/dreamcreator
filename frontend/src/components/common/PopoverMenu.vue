<template>
  <div class="macos-popover card-frosted card-translucent" @click.stop>
    <template v-for="(it, idx) in items" :key="it.key || idx">
      <div v-if="it && it.type === 'divider'" class="popover-divider"></div>
      <div v-else class="popover-item"
           @mouseenter="onEnter($event)"
           @mouseleave="onLeave($event)"
           @click.stop="onSelect(it)">
        <template v-if="it.icon">
          <Icon :name="it.icon" class="w-4 h-4 mr-2" />
        </template>
        <span class="popover-label">
          <span class="popover-label-track">
            <span class="popover-label-inner">{{ resolveLabel(it) }}</span>
            <span class="popover-label-inner" aria-hidden="true">{{ resolveLabel(it) }}</span>
          </span>
        </span>
      </div>
    </template>
  </div>
</template>

<script setup>
import { useI18n } from 'vue-i18n'

const props = defineProps({
  items: { type: Array, default: () => [] },
})

const emit = defineEmits(['select'])
const { t } = useI18n()

function resolveLabel(it) {
  if (!it) return ''
  if (it.labelKey) return t(it.labelKey)
  return it.label || ''
}

function onSelect(it) {
  emit('select', it)
}

function onEnter(ev) {
  try {
    const item = ev.currentTarget
    const label = item && item.querySelector && item.querySelector('.popover-label')
    const inner = label && label.querySelector && label.querySelector('.popover-label-inner')
    if (!label || !inner) return
    const cw = label.clientWidth
    const sw = inner.scrollWidth
    if (sw <= cw + 1) return
    const gap = 32
    const distance = sw + gap
    const duration = Math.min(12, Math.max(2, distance / 40))
    label.style.setProperty('--marquee-gap', gap + 'px')
    label.style.setProperty('--marquee-distance', distance + 'px')
    label.style.setProperty('--marquee-duration', duration + 's')
    label.classList.add('marquee')
  } catch {}
}
function onLeave(ev) {
  try {
    const item = ev.currentTarget
    const label = item && item.querySelector && item.querySelector('.popover-label')
    if (!label) return
    label.classList.remove('marquee')
    label.style.removeProperty('--marquee-distance')
    label.style.removeProperty('--marquee-duration')
  } catch {}
}
</script>

<style scoped>
/* 样式已迁移到全局 styles/macos-components.scss（macos-popover、popover-item 等） */
</style>
