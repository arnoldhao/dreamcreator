<template>
  <div v-if="show" class="macos-modal">
    <div class="modal-card card-frosted card-translucent" tabindex="-1">
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title">{{ isEdit ? t('subtitle.target_languages.edit_title') : t('subtitle.target_languages.create_title') }}</div>
      </div>
      <div class="modal-body">
        <div class="row-inline">
          <div class="col">
            <div class="field">
              <div class="field-label">{{ t('subtitle.target_languages.code') }}</div>
              <input class="input-macos" v-model="local.code" :placeholder="t('subtitle.target_languages.code_placeholder')" />
            </div>
          </div>
          <div class="col">
            <div class="field">
              <div class="field-label">{{ t('subtitle.target_languages.name') }}</div>
              <input class="input-macos" v-model="local.name" :placeholder="t('subtitle.target_languages.name_placeholder')" />
            </div>
          </div>
        </div>
      </div>
      <div class="modal-footer">
        <button @click="emit('close')" class="btn-chip">{{ t('common.cancel') }}</button>
        <button @click="save" class="btn-chip btn-primary" :disabled="!canSave">{{ t('common.save') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'

const props = defineProps({ show: Boolean, lang: Object })
const emit = defineEmits(['close', 'saved'])
const { t } = useI18n()

const local = ref({ code: '', name: '' })
const isEdit = computed(() => !!props.lang?.code)
const canSave = computed(() => !!(local.value.code && local.value.code.trim()))

watch(() => props.show, (v) => {
  if (v) {
    if (props.lang?.code) local.value = { code: props.lang.code, name: props.lang.name || '' }
    else local.value = { code: '', name: '' }
  }
})

async function save() {
  const code = (local.value.code || '').trim()
  const name = (local.value.name || '').trim()
  if (!code) return
  try {
    const saved = { code, name }
    emit('saved', { originalCode: props.lang?.code || '', saved })
  } catch (e) {
    console.error('Save language failed:', e)
    $message?.error?.(t('common.save_failed'))
  }
}
</script>

<style scoped>
.modal-card { border-radius: 12px; box-shadow: var(--macos-shadow-2); max-width: 640px; width: 100%; overflow: hidden; }
.modal-header { display:flex; align-items:center; justify-content: flex-start; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.modal-header .title { margin-left: auto; text-align: right; font-weight: 600; }
.modal-body { padding: 12px 16px; }
.row-inline { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.field { display:flex; flex-direction: column; gap: 6px; }
.field-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); padding-left: 8px; }
.modal-footer { border-top: 1px solid rgba(255,255,255,0.16); padding: 8px 12px; display:flex; justify-content: center; gap: 12px; }
</style>
