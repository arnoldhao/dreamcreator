<template>
  <div v-if="show" class="macos-modal" @click.self="close">
    <div class="modal-card" role="dialog" aria-modal="true">
      <div class="modal-header sheet">
        <ModalTrafficLights @close="close" />
        <div class="title-area">
          <div class="title-text">{{ $t('subtitle.common.edit') }} {{ $t('subtitle.title') }}</div>
        </div>
      </div>
      <div class="modal-body">
        <div class="list">
          <button v-for="(it, idx) in items" :key="idx" class="picker-item" @click="quickConfirm(it.value)">
            <span class="name" :title="it.label || it.value">{{ it.label || it.value }}</span>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch } from 'vue'
import Icon from '@/components/base/Icon.vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  items: { type: Array, default: () => [] }, // [{ label, value, meta }]
})
const emit = defineEmits(['update:show', 'confirm'])
const selected = ref('')

watch(() => props.show, (v) => { if (v) { selected.value = props.items?.[0]?.value || '' } })

function close() { emit('update:show', false) }
function confirm() { if (!selected.value) return; emit('confirm', selected.value); close() }
function quickConfirm(v) { emit('confirm', v); close() }
</script>

<style scoped>
.macos-modal { position: fixed; inset: 0; background: rgba(0,0,0,0.2); display:flex; align-items:center; justify-content:center; z-index: 2000; }
.modal-card { width: 420px; max-width: calc(100% - 32px); background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 12px; box-shadow: 0 20px 60px rgba(0,0,0,0.30); overflow:hidden; }
.modal-header.sheet { height: 36px; display:flex; align-items:center; justify-content: flex-start; padding: 0 10px; border-bottom: 1px solid var(--macos-divider-weak); }
/* no traffic lights for sheet-like modal */
.title-area { display:flex; align-items:center; gap: 10px; min-width: 0; flex:1; justify-content: flex-end; }
.title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); }
.modal-body { padding: 12px; }
.list { display:flex; flex-wrap: wrap; align-items:center; justify-content:center; gap: 8px; max-height: 360px; overflow:auto; }
.picker-item { display:flex; align-items:center; justify-content:center; gap:8px; padding:8px 10px; border-radius: 8px; border: 1px solid var(--macos-separator); background: var(--macos-background-secondary); cursor: pointer; flex: 0 0 calc(33% - 8px); min-width: 100px; }
.picker-item:hover { background: color-mix(in oklab, var(--macos-background) 85%, transparent); }
.name { font-size: var(--fs-sub); color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; text-align:center; }
</style>
