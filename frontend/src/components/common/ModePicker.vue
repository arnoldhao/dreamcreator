<template>
  <div class="mode-picker">
    <button class="mode-btn" :class="{ primary: defaultChoice === 'custom' }" @click="$emit('pick', 'custom')">
      <Icon name="settings" class="w-5 h-5 icon"/>
      <span class="label">{{ $t('download.custom_mode') }}</span>
    </button>
    <button class="mode-btn" :class="{ primary: defaultChoice === 'quick' }" @click="$emit('pick', 'quick')">
      <Icon name="flashlight" class="w-5 h-5 icon"/>
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
.mode-btn { height: 28px; padding: 0 12px; border-radius: 999px; background: var(--macos-background); border: 1px solid var(--macos-separator); color: var(--macos-text-primary); display:inline-flex; align-items:center; gap:8px; box-shadow: 0 1px 2px rgba(0,0,0,0.06); transition: transform .16s ease, box-shadow .16s ease, background .16s ease, border-color .16s ease; }
.mode-btn:hover { background: var(--macos-gray-hover); transform: translateY(-1px); box-shadow: 0 6px 14px rgba(0,0,0,0.08); }
.mode-btn.primary { background: color-mix(in oklab, var(--macos-blue) 12%, #fff); border-color: var(--macos-blue); }
.mode-btn .icon { display:block; }
.mode-btn .label { font-weight: 600; }

/* Dark theme tuning: reduce brightness for better readability */
:global([data-theme="dark"]) .mode-picker .mode-btn.primary {
  background: color-mix(in oklab, var(--macos-blue) 18%, #1c1c1e);
  border-color: color-mix(in oklab, var(--macos-blue) 65%, #1c1c1e);
  color: var(--macos-text-primary);
}
</style>
