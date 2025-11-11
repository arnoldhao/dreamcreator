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

        <!-- Settings popover menu (reused component) -->
        <PopoverMenu
          v-if="m.key === navStore.navOptions.SETTINGS && showSettingsMenu"
          :items="settingsMenuItems"
          :style="{ position: 'absolute', bottom: '36px', left: '8px', right: '8px' }"
          @select="onSettingsSelect"
        />
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
import PopoverMenu from '@/components/common/PopoverMenu.vue'

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

// Settings popover menu items (reused)
const settingsMenuItems = computed(() => ([
  { key: 'general', icon: 'settings', labelKey: 'settings.general.name' },
  { key: 'dependency', icon: 'package', labelKey: 'settings.dependency.title' },
  { key: 'providers', icon: 'database', labelKey: 'settings.model_provider' },
]))

function onSettingsSelect(it) {
  if (!it) return
  if (it.key === 'providers') return openProviders()
  if (it.key === 'dependency') return openSettings('dependency')
  return openSettings('general')
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

/* Popover 样式已全局化：.macos-popover/.popover-item 等在 styles/macos-components.scss */
</style>
