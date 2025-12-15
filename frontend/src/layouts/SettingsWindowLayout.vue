<script setup>
import { computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import SettingsWindowPage from '@/views/settings/SettingsWindowPage.vue'

const prefStore = usePreferencesStore()
const isDarkMode = computed(() => !!prefStore?.isDark)
</script>

<template>
  <div class="dc-shadcn relative min-h-screen settings-window" :class="{ dark: isDarkMode }">
    <div class="h-screen w-screen overflow-hidden">
      <SettingsWindowPage />
    </div>
  </div>
</template>
