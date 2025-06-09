<script setup>
import AppContent from './AppContent.vue'
import { onMounted, ref, watch } from 'vue'
import usePreferencesStore from './stores/preferences.js'
import { useI18n } from 'vue-i18n'
import { WindowSetDarkTheme, WindowSetLightTheme } from 'wailsjs/runtime/runtime.js'
import { applyMacosTheme } from '@/utils/theme.js'
import { WSON } from '@/handlers/websockets.js'

const { connect } = WSON()
const prefStore = usePreferencesStore()
const i18n = useI18n()
const initializing = ref(true)
onMounted(async () => {
  try {
    initializing.value = true
    if (prefStore.autoCheckUpdate) {
      prefStore.checkForUpdate()
    }

    // websocket connect
    connect()
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
    
    // Set DaisyUI theme
    document.documentElement.setAttribute('data-theme', isDark ? 'dark' : 'light')
    
    // Apply custom theme variables
    applyMacosTheme(isDark)
  },
  { immediate: true } // Apply immediately on component mount
)

// watch language and dynamically switch
watch(
  () => prefStore.general.language,
  (lang) => (i18n.locale.value = prefStore.currentLanguage),
)
</script>

<template>
  <div class="app-container" :data-theme="prefStore.isDark ? 'dark' : 'light'">
    <app-content :loading="initializing" />
  </div>
</template>

<style lang="scss"></style>
