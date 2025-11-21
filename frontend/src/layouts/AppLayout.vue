<script setup>
import AppContent from '@/AppContent.vue'

const props = defineProps({
  loading: { type: Boolean, default: false },
})
</script>

<template>
  <AppContent :loading="props.loading" />
  
</template>

