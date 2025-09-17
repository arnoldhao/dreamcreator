<template>
  <aside class="macos-sidebar" :style="{ width: widthPx, minWidth: widthPx }">
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

const navStore = useNavStore()
const layout = useLayoutStore()
const settings = useSettingsStore()

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
