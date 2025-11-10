<template>
  <div class="mode-picker">
    <button class="btn-chip" :class="{ 'btn-primary': defaultChoice === 'custom' }" @click="$emit('pick', 'custom')">
      <Icon name="settings" class="w-4 h-4 mr-2"/>
      <span class="label">{{ $t('download.custom_mode') }}</span>
    </button>
    <button class="btn-chip" :class="{ 'btn-primary': defaultChoice === 'quick' }" @click="$emit('pick', 'quick')">
      <Icon name="flashlight" class="w-4 h-4 mr-2"/>
      <span class="label">{{ $t('download.quick_mode') }}</span>
    </button>
  </div>
</template>

<script setup>
import { defineProps } from 'vue'
defineProps({ defaultChoice: { type: String, default: 'custom' } })
// simple picker with macOS-like pill buttons and subtle hover
</script>

<style scoped>
.mode-picker { display:flex; align-items:center; justify-content:center; gap: 12px; padding-top: 12px; }
.label { font-weight: 600; }
</style>
