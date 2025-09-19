<template>
  <aside class="macos-sidebar" :class="{ 'ribbon-frosted': uiFrosted }" :style="asideStyle">
    <!-- Main source list -->
    <div class="pl-2 pr-2 pt-1">
      <div v-for="(m, i) in navStore.menuOptions" :key="i" class="source-row"
        :class="{ active: props.value === m.key }"
        @click="emit('update:value', m.key)">
        <Icon :name="m.icon" class="source-row-icon" />
        <span class="source-row-label truncate">{{ $t(m.label) }}</span>
      </div>
    </div>

    <div class="mt-auto pl-2 pr-2 pb-2">
      <!-- Bottom items; settings shows popover -->
      <div v-for="(m, i) in navStore.bottomMenuOptions" :key="i" class="relative">
        <div class="source-row mt-2" :class="{ active: props.value === m.key }"
          @click.stop="onBottomItemClick(m)">
          <Icon :name="m.icon" class="source-row-icon" />
          <span class="source-row-label truncate">{{ $t(m.label) }}</span>
        </div>

        <!-- Settings popover menu -->
        <div v-if="m.key === navStore.navOptions.SETTINGS && showSettingsMenu"
             class="macos-popover card-frosted card-translucent" @click.stop>
          <div class="popover-item" @click="openSettings('general')">
            <Icon name="settings" class="w-4 h-4 mr-2" />
            <span>{{ $t('settings.general.name') }}</span>
          </div>
          <div class="popover-item" @click="openSettings('dependency')">
            <Icon name="package" class="w-4 h-4 mr-2" />
            <span>{{ $t('settings.dependency.title') }}</span>
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

function closeSettingsMenu() {
  showSettingsMenu.value = false
}

// simple outside click handler for safety
function handleDocClick() { showSettingsMenu.value = false }
onMounted(() => document.addEventListener('click', handleDocClick))
onBeforeUnmount(() => document.removeEventListener('click', handleDocClick))

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

.source-row {
  height: 28px;
  display: flex;
  align-items: center;
  border-radius: 8px;
  padding: 0 10px 0 12px;
  color: var(--macos-text-secondary);
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease;
}
.source-row:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
.source-row.active { background: color-mix(in oklab, var(--macos-blue) 22%, transparent); color: #fff; font-weight: 500; }
.source-row-icon { width: 18px; height: 18px; margin-right: 8px; color: var(--macos-text-secondary); }
.source-row:hover .source-row-icon, .source-row.active .source-row-icon { color: #fff; }
.source-row-label { font-size: var(--fs-base); line-height: 1; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

/* Light + frosted: improve clarity by using primary text color by default */
[data-ui="frosted"][data-theme="light"] .macos-sidebar .source-row { color: var(--macos-text-primary); }
[data-ui="frosted"][data-theme="light"] .macos-sidebar .source-row-icon { color: var(--macos-text-primary); }

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
.popover-item :deep(.sr-icon) {
  /* Fix icon column width so text starts at the same x for all rows */
  width: 16px; height: 16px; margin-right: 8px; flex: 0 0 16px; display: inline-flex; align-items: center; justify-content: center;
}
.popover-item + .popover-item { margin-top: 4px; }
.popover-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
</style>
