<script setup>
import AppContent from './AppContent.vue'
import { onMounted, ref, watch } from 'vue'
import usePreferencesStore from './stores/preferences.js'
import { useI18n } from 'vue-i18n'
import { WindowSetDarkTheme, WindowSetLightTheme } from 'wailsjs/runtime/runtime.js'
import { applyMacosTheme, applyAccent } from '@/utils/theme.js'

const prefStore = usePreferencesStore()
const i18n = useI18n()
const initializing = ref(true)
onMounted(async () => {
  try {
    initializing.value = true
    if (prefStore.autoCheckUpdate) {
      prefStore.checkForUpdate()
    }

  } finally {
    initializing.value = false
  }
})

// watch theme and dynamically switch
watch(
  () => prefStore.isDark,
  (isDark) => {
    // Set Wails window theme
    isDark ? WindowSetDarkTheme() : WindowSetLightTheme()
    
    // Apply custom theme variables and accent
    applyMacosTheme(isDark)
    applyAccent(prefStore.general.theme, isDark)
  },
  { immediate: true } // Apply immediately on component mount
)

// watch accent theme changes and apply immediately
watch(
  () => prefStore.general.theme,
  (accent) => {
    applyAccent(accent, prefStore.isDark)
  },
  { immediate: false }
)

// watch language and dynamically switch
watch(
  () => prefStore.general.language,
  (lang) => (i18n.locale.value = prefStore.currentLanguage),
)
</script>

<template>
  <div class="app-container">
    <app-content :loading="initializing" />
  </div>
</template>

<style lang="scss"></style>
