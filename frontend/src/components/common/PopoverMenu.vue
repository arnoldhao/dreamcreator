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
.macos-popover {
  border-radius: 10px;
  box-shadow: var(--macos-shadow-2);
  padding: 6px;
  z-index: 1200;
  background: transparent; /* real bg from card-frosted */
  /* Ensure long menus can scroll; override card-frosted overflow */
  max-height: var(--menu-max-h, 420px);
  overflow-y: auto;
  overflow-x: hidden;
}
.popover-item {
  height: 28px;
  display: flex;
  align-items: center;
  padding: 0 8px;
  border-radius: 6px;
  font-size: var(--fs-base);
  color: var(--macos-text-primary);
  cursor: pointer;
}
.popover-item + .popover-item { margin-top: 4px; }
.popover-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }

/* marquee label infra */
.popover-item .popover-label { flex: 1 1 auto; min-width: 0; overflow: hidden; white-space: nowrap; }
.popover-item .popover-label-track { display: inline-flex; align-items: center; will-change: transform; }
.popover-item .popover-label:not(.marquee) .popover-label-inner:last-child { display: none; }
.popover-item .popover-label-inner + .popover-label-inner { margin-left: var(--marquee-gap, 32px); }
.popover-item:hover .popover-label.marquee .popover-label-track { animation: ribbon-marquee var(--marquee-duration, 4s) linear 0.25s infinite; }
@keyframes ribbon-marquee { from { transform: translateX(0); } to { transform: translateX(calc(-1 * var(--marquee-distance, 0px))); } }
.popover-item .popover-label.marquee {
  -webkit-mask-image: linear-gradient(to right, transparent 0, #000 8px, #000 calc(100% - 8px), transparent 100%);
  mask-image: linear-gradient(to right, transparent 0, #000 8px, #000 calc(100% - 8px), transparent 100%);
}

.popover-divider { height: 1px; background: var(--macos-divider-weak); margin: 6px -8px; }

/* Icon column sizing for rows with icons */
.popover-item :deep(.sr-icon) { width: 16px; height: 16px; margin-right: 8px; flex: 0 0 16px; display: inline-flex; align-items: center; justify-content: center; }
</style>
