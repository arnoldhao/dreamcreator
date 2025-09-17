<template>
  <div class="macos-modal" @click.self="$emit('close')">
    <div class="modal-card">
      <div class="modal-header">
        <ModalTrafficLights @close="$emit('close')" />
        <div class="title">{{ $t('cookies.title') }}</div>
      </div>
      <div class="modal-body">
        <!-- TODO: 重构 Cookie 管理器，现放占位 -->
        <div class="text-secondary text-sm">{{ $t('common.coming_soon') }}</div>
      </div>
      <div class="modal-footer">
        <button class="btn-glass" @click="$emit('close')">{{ $t('common.close') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
// placeholder for new macOS-style Cookies manager
</script>

<style scoped>
.macos-modal { position: fixed; inset: 0; background: rgba(0,0,0,0.2); display:flex; align-items:center; justify-content:center; z-index: 2000; }
.modal-card { width: 680px; max-width: calc(100% - 32px); background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 12px; box-shadow: var(--macos-shadow-3); overflow:hidden; }
.modal-header { height: 36px; display:flex; align-items:center; justify-content: space-between; padding: 0 10px; border-bottom: 1px solid var(--macos-divider-weak); }
.modal-header .title { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); }
.modal-body { padding: 12px; }
.modal-footer { display:flex; align-items:center; justify-content:flex-end; gap:8px; padding: 8px 10px; border-top: 1px solid var(--macos-divider-weak); }
</style>
