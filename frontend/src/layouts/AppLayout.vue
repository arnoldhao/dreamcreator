<script setup>
import AppContent from '@/AppContent.vue'
import { computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'

const props = defineProps({
  loading: { type: Boolean, default: false },
})

const prefStore = usePreferencesStore()
const isDarkMode = computed(() => !!prefStore?.isDark)
</script>

<template>
  <div class="dc-shadcn relative min-h-screen main-window" :class="{ dark: isDarkMode }">
    <div class="h-screen w-screen overflow-hidden">
      <AppContent :loading="props.loading" />
    </div>
  </div>
</template>
