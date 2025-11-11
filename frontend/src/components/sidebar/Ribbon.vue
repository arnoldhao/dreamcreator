<template>
  <aside class="macos-sidebar" :class="{ 'ribbon-frosted': uiFrosted }" :style="asideStyle">
    <!-- Main source list -->
    <div class="pl-2 pr-2 pt-1 source-group">
      <div
        v-for="(m, i) in navStore.menuOptions"
        :key="i"
        class="source-chip"
        :class="{ 'ribbon-active active': props.value === m.key }"
        @click="emit('update:value', m.key)"
      >
        <span class="icon-cell"><Icon :name="m.icon" class="source-row-icon" /></span>
        <span class="label-cell"><span class="source-row-label truncate">{{ $t(m.label) }}</span></span>
      </div>
    </div>

    <div class="mt-auto pl-2 pr-2 pb-2 source-group">
      <!-- Bottom items; settings shows popover -->
      <div v-for="(m, i) in navStore.bottomMenuOptions" :key="i" class="relative">
        <div
          class="source-chip"
          :class="{ 'ribbon-active active': props.value === m.key }"
          @click.stop="onBottomItemClick(m)"
        >
          <span class="icon-cell"><Icon :name="m.icon" class="source-row-icon" /></span>
          <span class="label-cell"><span class="source-row-label truncate">{{ $t(m.label) }}</span></span>
        </div>

        <!-- Settings popover menu -->
        <div v-if="m.key === navStore.navOptions.SETTINGS && showSettingsMenu"
             class="macos-popover card-frosted card-translucent" @click.stop>
          <div class="popover-item" @mouseenter="onPopoverEnter($event)" @mouseleave="onPopoverLeave($event)" @click="openSettings('general')">
            <Icon name="settings" class="w-4 h-4 mr-2" />
            <span class="popover-label">
              <span class="popover-label-track">
                <span class="popover-label-inner">{{ $t('settings.general.name') }}</span>
                <span class="popover-label-inner" aria-hidden="true">{{ $t('settings.general.name') }}</span>
              </span>
            </span>
          </div>
          <div class="popover-item" @mouseenter="onPopoverEnter($event)" @mouseleave="onPopoverLeave($event)" @click="openSettings('dependency')">
            <Icon name="package" class="w-4 h-4 mr-2" />
            <span class="popover-label">
              <span class="popover-label-track">
                <span class="popover-label-inner">{{ $t('settings.dependency.title') }}</span>
                <span class="popover-label-inner" aria-hidden="true">{{ $t('settings.dependency.title') }}</span>
              </span>
            </span>
          </div>
          <div class="popover-item" @mouseenter="onPopoverEnter($event)" @mouseleave="onPopoverLeave($event)" @click="openProviders()">
            <Icon name="database" class="w-4 h-4 mr-2" />
            <span class="popover-label">
              <span class="popover-label-track">
                <span class="popover-label-inner">{{ $t('settings.model_provider') }}</span>
                <span class="popover-label-inner" aria-hidden="true">{{ $t('settings.model_provider') }}</span>
              </span>
            </span>
          </div>
          
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import useNavStore from 'stores/nav.js'
import useLayoutStore from '@/stores/layout.js'
import useSettingsStore from '@/stores/settings.js'
import usePreferencesStore from '@/stores/preferences.js'

const navStore = useNavStore()
const layout = useLayoutStore()
const settings = useSettingsStore()
const prefStore = usePreferencesStore()

const props = defineProps({
  value: {
    type: String,
    default: 'download',
  },
  width: {
    type: Number,
    default: 260,
  },
})

const emit = defineEmits(['update:value'])

const widthPx = computed(() => `${props.width}px`)
const uiFrosted = computed(() => (prefStore?.general?.uiStyle || 'frosted') === 'frosted')
const isDarkMode = computed(() => !!prefStore?.isDark)
// 根据明暗主题调整毛玻璃底色（仅 ribbon 区域）
const ribbonFrostedBg = computed(() => isDarkMode.value ? 'rgba(0,0,0,0.28)' : 'rgba(255,255,255,0.28)')
const asideStyle = computed(() => ({
  width: widthPx.value,
  minWidth: widthPx.value,
  ...(uiFrosted.value
    ? {
        '--ribbon-bg': ribbonFrostedBg.value,
        WebkitBackdropFilter: 'var(--macos-surface-blur)',
        backdropFilter: 'var(--macos-surface-blur)'
      }
    : {})
}))

// Settings popover state
const showSettingsMenu = ref(false)

function onBottomItemClick(m) {
  if (m.key === navStore.navOptions.SETTINGS) {
    // toggle popover instead of direct navigation
    showSettingsMenu.value = !showSettingsMenu.value
  } else {
    emit('update:value', m.key)
  }
}

function openSettings(key) {
  settings.setPage(key)
  emit('update:value', navStore.navOptions.SETTINGS)
  showSettingsMenu.value = false
}

function openProviders() {
  emit('update:value', navStore.navOptions.PROVIDERS)
  showSettingsMenu.value = false
}

function closeSettingsMenu() {
  showSettingsMenu.value = false
}

// simple outside click handler for safety
function handleDocClick() { showSettingsMenu.value = false }
onMounted(() => document.addEventListener('click', handleDocClick))
onBeforeUnmount(() => document.removeEventListener('click', handleDocClick))

// Popover label marquee-on-hover for overflowed text (one-direction seamless)
function onPopoverEnter(ev) {
  try {
    const item = ev.currentTarget
    const label = item && item.querySelector && item.querySelector('.popover-label')
    const inner = label && label.querySelector && label.querySelector('.popover-label-inner')
    if (!label || !inner) return
    // Ensure layout updated
    const cw = label.clientWidth
    const sw = inner.scrollWidth
    if (sw <= cw + 1) return // no overflow, no marquee
    const gap = 32 // separation between original and clone
    const distance = sw + gap // seamless loop distance equals content width + gap
    const duration = Math.min(12, Math.max(2, distance / 40)) // ~40px/s, clamp 2s..12s
    label.style.setProperty('--marquee-gap', gap + 'px')
    label.style.setProperty('--marquee-distance', distance + 'px')
    label.style.setProperty('--marquee-duration', duration + 's')
    label.classList.add('marquee')
  } catch {}
}
function onPopoverLeave(ev) {
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

<style lang="scss" scoped>
.macos-sidebar { height: 100%; }

/* Frosted variant for ribbon area only when UI style is frosted */
.macos-sidebar.ribbon-frosted {
  /* 具体底色使用组件传入的变量，强制覆盖所有主题色染色 */
  background: var(--ribbon-bg, rgba(0,0,0,0.28)) !important;
  background-image: none !important;
  -webkit-backdrop-filter: var(--macos-surface-blur);
  backdrop-filter: var(--macos-surface-blur);
  isolation: isolate; /* 防止外层混色影响 */
}

.ribbon-header {
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding: 0 6px 0 6px;
}

/* Light + frosted tweaks for items now live in global styles */

/* macOS-style popover */
.macos-popover {
  position: absolute;
  bottom: 36px; /* just above the bottom row */
  left: 8px;
  right: 8px;
  border-radius: 10px;
  box-shadow: var(--macos-shadow-2);
  padding: 6px;
  z-index: 1000;
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
.popover-item .popover-label { flex: 1 1 auto; min-width: 0; overflow: hidden; white-space: nowrap; }
.popover-item .popover-label-track { display: inline-flex; align-items: center; will-change: transform; }
.popover-item .popover-label-inner + .popover-label-inner { margin-left: var(--marquee-gap, 32px); }
/* Enable marquee when parent label has .marquee class; animate only on hover */
.popover-item:hover .popover-label.marquee .popover-label-track {
  animation: ribbon-marquee var(--marquee-duration, 4s) linear 0.25s infinite;
}
@keyframes ribbon-marquee {
  from { transform: translateX(0); }
  to { transform: translateX(calc(-1 * var(--marquee-distance, 0px))); }
}
/* fade edges to avoid abrupt cut while animating */
.popover-item .popover-label.marquee {
  -webkit-mask-image: linear-gradient(to right, transparent 0, #000 8px, #000 calc(100% - 8px), transparent 100%);
  mask-image: linear-gradient(to right, transparent 0, #000 8px, #000 calc(100% - 8px), transparent 100%);
}
.popover-item :deep(.sr-icon) {
  /* Fix icon column width so text starts at the same x for all rows */
  width: 16px; height: 16px; margin-right: 8px; flex: 0 0 16px; display: inline-flex; align-items: center; justify-content: center;
}
.popover-item + .popover-item { margin-top: 4px; }
.popover-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
</style>
