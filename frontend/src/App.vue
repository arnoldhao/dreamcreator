<script setup>
import AppLayout from '@/layouts/AppLayout.vue'
import SettingsWindowLayout from '@/layouts/SettingsWindowLayout.vue'
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import usePreferencesStore from './stores/preferences.js'
import { useI18n } from 'vue-i18n'
import { applyMacosTheme, applyAccent } from '@/utils/theme.js'
import { Events } from '@wailsio/runtime'

const prefStore = usePreferencesStore()
const i18n = useI18n()
const initializing = ref(true)
const windowMode = ref('main')
let offPreferencesChanged = null

try {
  const mode = new URLSearchParams(window.location.search).get('window')
  windowMode.value = mode || 'main'
} catch {}
onMounted(async () => {
  try {
    initializing.value = true
    if (prefStore.autoCheckUpdate) {
      prefStore.checkForUpdate()
    }

  } finally {
    initializing.value = false
  }

  // Cross-window preferences sync: update this window when Settings window saves.
  try {
    offPreferencesChanged = Events.On("app:preferencesChanged", async () => {
      try { await prefStore.loadPreferences() } catch {}
    })
  } catch {}
})
onBeforeUnmount(() => { try { offPreferencesChanged?.() } catch {} })

// watch theme and dynamically switch
watch(
  () => prefStore.isDark,
  (isDark) => {
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
    <SettingsWindowLayout v-if="windowMode === 'settings'" />
    <app-layout v-else :loading="initializing" />
  </div>
</template>

<style lang="scss"></style>
