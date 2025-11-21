<template>
  <div v-if="show" class="macos-modal">
    <div class="modal-card card-frosted card-translucent" tabindex="-1">
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title">{{ mode === 'create' ? $t('profiles.add') : $t('profiles.edit') }}</div>
      </div>
      <div class="modal-body">
        <div class="field">
          <div class="field-label">{{ $t('profiles.name') }}</div>
          <input v-model="form.name" class="input-macos input-full" :placeholder="$t('profiles.name_placeholder')" />
        </div>
        <div class="grid2">
          <div class="field">
            <div class="field-label">{{ $t('profiles.temperature') }}</div>
            <input type="number" step="0.1" min="0" max="2" v-model.number="form.temperature" class="input-macos input-full" />
          </div>
          <div class="field">
            <div class="field-label">{{ $t('profiles.top_p') }}</div>
            <input type="number" step="0.05" min="0" max="1" v-model.number="form.top_p" class="input-macos input-full" />
          </div>
        </div>
        <div class="grid2">
          <div class="field">
            <div class="field-label">{{ $t('profiles.max_tokens') }}</div>
            <input type="number" min="0" v-model.number="form.max_tokens" class="input-macos input-full" />
          </div>
          <div class="field">
            <div class="field-label">&nbsp;</div>
            <div class="left-inline-center">
              <label class="about-toggle">
                <input type="checkbox" class="about-toggle-input" v-model="form.json_mode" />
                <span class="about-toggle-slider"></span>
              </label>
              <span class="toggle-label">{{ $t('profiles.json_mode') }}</span>
            </div>
          </div>
        </div>
        <div class="field">
          <div class="field-label">{{ $t('profiles.sys_prompt_tpl') }}</div>
          <textarea v-model="form.sys_prompt_tpl" rows="6" class="input-macos input-full" :placeholder="$t('profiles.sys_prompt_placeholder')"></textarea>
        </div>
      </div>
      <div class="modal-footer">
        <button class="btn-chip" @click="emit('close')">{{ $t('profiles.cancel') }}</button>
        <button class="btn-chip btn-primary" @click="save">{{ $t('profiles.save') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch } from 'vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import { createGlobalProfile, updateGlobalProfile } from '@/services/llmProviderService.js'

const props = defineProps({ show: Boolean, mode: { type: String, default: 'create' }, profile: { type: Object, default: () => ({}) } })
const emit = defineEmits(['close','saved'])

const form = reactive({ id:'', name:'', temperature:0.2, top_p:1, json_mode:true, max_tokens:2048, sys_prompt_tpl:'' })

watch(() => props.show, (v) => { if (v) applyFromProps() })
watch(() => props.profile, () => { if (props.show) applyFromProps() })

function applyFromProps(){
  form.id = props.profile?.id || ''
  form.name = props.profile?.name || ''
  form.temperature = Number(props.profile?.temperature ?? 0.2)
  form.top_p = Number(props.profile?.top_p ?? 1)
  form.json_mode = !!props.profile?.json_mode
  form.max_tokens = Number(props.profile?.max_tokens ?? 2048)
  form.sys_prompt_tpl = props.profile?.sys_prompt_tpl || ''
}

async function save(){
  const payload = { name: form.name||'', temperature: form.temperature, top_p: form.top_p, json_mode: form.json_mode, max_tokens: form.max_tokens, sys_prompt_tpl: form.sys_prompt_tpl }
  try {
    if (form.id) { await updateGlobalProfile(form.id, payload) } else { await createGlobalProfile(payload) }
    emit('saved')
  } catch (e) {
    window.$message?.error?.(e?.message || 'Save failed')
  }
}
</script>

<style scoped>
/* Align modal visuals with SubtitleAddLanguageModal */
.modal-card { border-radius: 12px; box-shadow: var(--macos-shadow-2); max-width: 700px; width: 100%; max-height: 85vh; overflow: hidden; animation: slideInUp 0.3s ease-out; }
.modal-card.card-frosted.card-translucent { background: color-mix(in oklab, var(--macos-surface) 76%, transparent); border: 1px solid rgba(255,255,255,0.28); box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0,0,0,0.24); }
.modal-header { display:flex; align-items:center; justify-content: space-between; height: 36px; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.title { flex:1; min-width: 0; display:flex; align-items:center; justify-content:flex-end; }
.modal-body { padding: 12px 16px; display:flex; flex-direction: column; gap: 12px; max-height: calc(85vh - 36px - 48px); min-height: 0; overflow-y: auto; overflow-x: hidden; }
.modal-footer { border-top: 1px solid rgba(255,255,255,0.16); padding: 8px 12px; display:flex; justify-content: center; gap: 12px; }

/* Fields and layout parity */
.grid2 { display:grid; grid-template-columns: 1fr 1fr; align-items: end; column-gap: 12px; }
.field { display:flex; flex-direction: column; gap: 8px; }
.field-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); padding-left: 8px; }
.left-inline-center { display:inline-flex; align-items:center; justify-content:flex-start; gap: 10px; min-height: 18px; }
.toggle-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); line-height: 18px; display: inline-flex; align-items: center; vertical-align: middle; white-space: nowrap; }

/* Toggle switch styling (shared look) */
.about-toggle { position: relative; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; vertical-align: middle; }
.about-toggle-input { position: absolute; width: 0; height: 0; opacity: 0; }
.about-toggle-slider { width: 32px; height: 18px; border-radius: 999px; background: var(--macos-divider-weak); box-shadow: inset 0 0 0 1px var(--macos-divider-weak); transition: background 180ms ease, box-shadow 180ms ease; display: inline-block; position: relative; }
.about-toggle-slider::after { content: ""; position: absolute; width: 14px; height: 14px; border-radius: 50%; background: var(--macos-background); top: 2px; left: 2px; box-shadow: var(--macos-shadow-1); transition: transform 180ms ease; }
.about-toggle-input:checked + .about-toggle-slider { background: color-mix(in srgb, var(--macos-blue) 55%, transparent); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--macos-blue) 70%, transparent); }
.about-toggle-input:checked + .about-toggle-slider::after { transform: translateX(14px); background: var(--macos-background); box-shadow: 0 0 0 1px color-mix(in srgb, var(--macos-blue) 65%, transparent), 0 2px 4px rgba(0,0,0,0.12); }
.about-toggle-input:focus-visible + .about-toggle-slider { outline: 2px solid rgba(var(--macos-blue-rgb), 0.7); outline-offset: 2px; }
</style>
