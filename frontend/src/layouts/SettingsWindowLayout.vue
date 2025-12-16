<script setup>
import { computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import SettingsWindowPage from '@/views/settings/SettingsWindowPage.vue'

const prefStore = usePreferencesStore()
const isDarkMode = computed(() => !!prefStore?.isDark)
</script>

<template>
  <div class="dc-shadcn relative min-h-screen settings-window" :class="{ dark: isDarkMode }">
    <div class="h-screen w-screen overflow-hidden flex">
      <div class="flex-1 min-h-0 min-w-0">
        <SettingsWindowPage />
      </div>
    </div>
  </div>
</template>
