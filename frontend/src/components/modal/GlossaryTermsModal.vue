<template>
  <div v-if="show" class="macos-modal">
    <div class="modal-card card-frosted card-translucent" @keydown.esc.stop.prevent="emit('close')" tabindex="-1">
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title">{{ mode === 'create' ? t('glossary.modal_create_title') : t('glossary.modal_edit_title') }}</div>
      </div>
      <div class="modal-body">
        <div class="row-inline">
          <div class="col">
            <div class="field">
              <div class="field-label">{{ t('common.name') }}</div>
              <input
                class="input-macos"
                v-model="localSet.name"
                :placeholder="t('glossary.name_placeholder')"
                @blur="onMetaBlur"
                @keyup.enter.prevent="onMetaBlur"
              />
            </div>
          </div>
          <div class="col">
            <div class="field">
              <div class="field-label">{{ t('common.description') }}</div>
              <input
                class="input-macos"
                v-model="localSet.description"
                :placeholder="t('glossary.desc_placeholder')"
                @blur="onMetaBlur"
                @keyup.enter.prevent="onMetaBlur"
              />
            </div>
          </div>
        </div>
        <div class="divider"></div>
        <!-- When set not saved, show only a beautified hint -->
        <template v-if="!isReady">
          <div class="empty-state">
            <div class="empty-icon"><Icon name="info" class="w-6 h-6" /></div>
            <div class="empty-title">{{ t('glossary.save_set_first_hint') }}</div>
          </div>
        </template>

        <!-- When set is saved, show term editor and list -->
        <template v-else>
          <!-- Add/Edit toolbar (explicit create/save), two equal columns -->
          <div class="edit-grid2">
            <div class="field term">
              <div class="field-label">{{ t('glossary.term') }}</div>
              <input class="input-macos" v-model="editTerm.source" :placeholder="t('glossary.term_placeholder')" :disabled="!isReady" @input="resetTracking" />
            </div>
            <div class="field controls">
              <div class="field-label">&nbsp;</div>
              <div class="controls-row">
                <div class="left">
                  <div class="left-inline-center">
                    <label class="about-toggle">
                      <input type="checkbox" class="about-toggle-input" v-model="editTerm.case_sensitive" :disabled="!isReady" />
                      <span class="about-toggle-slider"></span>
                    </label>
                    <span class="toggle-label">{{ t('glossary.case_sensitive') }}</span>
                  </div>
                </div>
                <div class="mid">
                  <div class="seg-chip chip-switch chip-frosted chip-sm chip-translucent seg-toggle">
                    <button type="button" class="seg-item" :class="{ active: editMode==='dnt' }" @click="editMode='dnt'">{{ t('glossary.dnt') }}</button>
                    <button type="button" class="seg-item" :class="{ active: editMode==='specify' }" @click="editMode='specify'">{{ t('glossary.specify') }}</button>
                  </div>
                </div>
                <div class="right">
                  <button class="btn-chip-icon" :data-tooltip="t('common.add')" @click="newDraft" :disabled="!isReady">
                    <Icon name="plus" class="w-4 h-4" />
                  </button>
                  <button class="btn-chip-icon" :data-tooltip="t('common.save')" @click="saveDraft" :disabled="!isReady || !canSaveTerm || !isDirtyTerm">
                    <Icon name="save" class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>
          <div v-if="editMode==='specify'" class="edit-second2">
            <div class="field">
              <div class="field-label">{{ t('glossary.lang') }}</div>
              <select class="select-macos select-tight" v-model="topLang" :disabled="!isReady">
                <option value="">{{ t('subtitle.add_language.select_target_language') }}</option>
                <option v-for="opt in langOptions" :key="opt" :value="opt">{{ getLangName(opt) }}</option>
              </select>
            </div>
            <div class="field">
              <div class="field-label">{{ t('glossary.translation_placeholder') }}</div>
              <input class="input-macos" v-model="topTrans" :placeholder="t('glossary.translation_placeholder')" :disabled="!isReady" />
            </div>
          </div>
          
          <div class="gl-table" v-if="entries && entries.length">
            <div class="gl-head">
              <div class="h cell term">{{ t('glossary.term') }} <span class="muted">{{ t('glossary.terms_count', { count: entries.length }) }}</span></div>
              <div class="h cell case">{{ t('glossary.case_sensitive') }}</div>
              <div class="h cell trans">{{ t('glossary.translation_placeholder') }}</div>
              <div class="h cell act">&nbsp;</div>
            </div>
            <div class="terms-scroll">
              <div class="gl-body">
                <div class="row" v-for="e in entries" :key="e.id" @click="selectForEdit(e)">
                <div class="cell term mono" :title="e.source"><span class="one-line">{{ e.source }}</span></div>
                  <div class="cell case"><span class="muted">{{ e.case_sensitive ? t('common.yes') : t('common.no') }}</span></div>
                  <div class="cell trans"><span class="one-line" :title="getSingleTranslation(e)">{{ getSingleTranslation(e) }}</span></div>
                  <div class="cell act">
                    <div class="act-row">
                      <button class="btn-chip-icon btn-danger" :data-tooltip="t('common.delete')" @click.stop="deleteTerm(e)"><Icon name="trash" class="w-4 h-4" /></button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="empty-state">
              <div class="empty-icon"><Icon name="list" class="w-6 h-6" /></div>
              <div class="empty-title">{{ t('glossary.no_terms') }}</div>
              <div class="empty-hint">{{ t('glossary.no_terms_hint') }}</div>
          </div>
        </template>
      </div>
      <div class="modal-footer">
        <button class="btn-chip" @click="emit('close')">{{ t('common.close') }}</button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, reactive, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/base/Icon.vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import { subtitleService } from '@/services/subtitleService.js'

const props = defineProps({ show: Boolean, set: Object, mode: { type: String, default: 'edit' } })
const emit = defineEmits(['close', 'saved'])
const { t } = useI18n()

const entries = ref([])
const langOptions = ref(['all'])
const editTerm = reactive({ id: '', source: '', case_sensitive: false, do_not_translate: true, translations: {} })
const editMode = ref('dnt')
const topLang = ref('')
const topTrans = ref('')
const newEntryId = ref('')
const localSet = reactive({ id: '', name: '', description: '' })

// Track original state of the edit area to disable Save when unchanged
const originalTerm = reactive({ id: '', source: '', case_sensitive: false, mode: 'dnt', lang: '', trans: '' })
const isDirtyTerm = computed(() => {
  const curSource = (editTerm.source || '').trim()
  const curCase = !!editTerm.case_sensitive
  const curMode = editMode.value
  const curLang = (curMode === 'specify') ? (topLang.value || '') : ''
  const curTrans = (curMode === 'specify') ? ((topTrans.value || '').trim()) : ''
  return (
    originalTerm.source !== curSource ||
    originalTerm.case_sensitive !== curCase ||
    originalTerm.mode !== curMode ||
    (curMode === 'specify' && (originalTerm.lang !== curLang || originalTerm.trans !== curTrans))
  )
})
function syncBaselineFromCurrent() {
  originalTerm.id = editTerm.id || ''
  originalTerm.source = (editTerm.source || '').trim()
  originalTerm.case_sensitive = !!editTerm.case_sensitive
  originalTerm.mode = editMode.value
  originalTerm.lang = (editMode.value === 'specify') ? (topLang.value || '') : ''
  originalTerm.trans = (editMode.value === 'specify') ? ((topTrans.value || '').trim()) : ''
}

watch(() => props.show, (v) => {
  if (v) {
    reload()
  } else {
    // 关闭 Modal 后清空术语编辑内容
    newDraft()
  }
})

async function reload() {
  // initialize set fields
  if (props.set?.id) {
    localSet.id = props.set.id
    localSet.name = props.set.name || ''
    localSet.description = props.set.description || ''
  } else {
    localSet.id = ''
    localSet.name = props.set?.name || ''
    localSet.description = props.set?.description || ''
  }
  // load entries if set ready
  if (!localSet.id) { entries.value = []; return }
  try {
    const list = await subtitleService.listGlossaryBySet(localSet.id)
    entries.value = Array.isArray(list) ? list : []
  } catch { entries.value = [] }
  // load available target languages (prepend 'all')
  try {
    const l = await subtitleService.listTargetLanguages()
    const codes = Array.isArray(l) ? l.map(x => x.code) : []
    langOptions.value = ['all', ...codes]
    // build name map
    langNameMap.value = {}
    for (const it of (Array.isArray(l) ? l : [])) {
      langNameMap.value[it.code] = it.name || it.code
    }
  } catch {
    // fallback: minimal defaults when backend not ready
    langOptions.value = ['all']
    langNameMap.value = {}
  }
}

// legacy add/edit helpers removed (explicit New/Save model only)

const langNameMap = ref({})
function getLangName(code) {
  if (code === 'all') return t('glossary.lang_all')
  return langNameMap.value[code] || code
}

function getSingleTranslation(e) {
  if (e.do_not_translate) return t('glossary.dnt')
  const tr = e.translations || {}
  if (tr['all']) return tr['all']
  const keys = Object.keys(tr)
  if (keys.length === 0) return ''
  keys.sort()
  return tr[keys[0]]
}

function pickSingleTranslation(tr) {
  if (!tr) return { lang: '', val: '' }
  if (tr['all']) return { lang: 'all', val: tr['all'] }
  const keys = Object.keys(tr)
  if (keys.length === 0) return { lang: '', val: '' }
  keys.sort()
  const k = keys[0]
  return { lang: k, val: tr[k] || '' }
}

// removed edit-buffers helpers (no inline translation editing rows)

// Explicit term editing flow (no autosave for term fields)
function resetTracking() { newEntryId.value = editTerm.id || '' }
function newDraft() {
  editTerm.id = ''
  editTerm.source = ''
  editTerm.case_sensitive = false
  editTerm.do_not_translate = true
  editTerm.translations = {}
  editMode.value = 'dnt'
  topLang.value = ''
  topTrans.value = ''
  newEntryId.value = ''
  // establish fresh baseline for a new draft
  syncBaselineFromCurrent()
}

const canSaveTerm = computed(() => {
  const hasSource = !!(editTerm.source && editTerm.source.trim())
  if (!hasSource) return false
  if (editMode.value === 'specify') {
    const hasLang = !!topLang.value
    const hasTrans = !!(topTrans.value && topTrans.value.trim())
    return hasLang && hasTrans
  }
  return true
})
async function saveDraft() {
  if (!canSaveTerm.value) return
  if (!localSet.id) { const ok = await ensureSetSaved(); if (!ok) return }
  const payload = {
    id: editTerm.id || undefined,
    set_id: localSet.id,
    source: editTerm.source.trim(),
    do_not_translate: (editMode.value === 'dnt'),
    case_sensitive: !!editTerm.case_sensitive,
    translations: {}
  }
  if (editMode.value === 'specify' && topLang.value && topTrans.value && topTrans.value.trim()) {
    payload.translations[topLang.value] = topTrans.value.trim()
  }
  try {
    const saved = await subtitleService.upsertGlossaryEntry(payload)
    if (saved) {
      const idx = entries.value.findIndex(x => x.id === saved.id)
      if (idx >= 0) entries.value[idx] = saved; else entries.value.unshift(saved)
      editTerm.id = saved.id
      editTerm.translations = saved.translations || {}
      editTerm.do_not_translate = saved.do_not_translate
      newEntryId.value = saved.id
      $message?.success?.(t('common.saved'))
      // reset baseline after successful save
      syncBaselineFromCurrent()
    }
  } catch { $message?.error?.(t('common.save_failed')) }
}

function selectForEdit(e) {
  editTerm.id = e.id
  editTerm.source = e.source
  editTerm.case_sensitive = !!e.case_sensitive
  editTerm.do_not_translate = !!e.do_not_translate
  editTerm.translations = { ...(e.translations || {}) }
  editMode.value = editTerm.do_not_translate ? 'dnt' : 'specify'
  if (editMode.value === 'specify') {
    const picked = pickSingleTranslation(editTerm.translations)
    topLang.value = picked.lang
    topTrans.value = picked.val
  } else {
    topLang.value = ''
    topTrans.value = ''
  }
  newEntryId.value = e.id
  // set baseline to the loaded entry
  syncBaselineFromCurrent()
}

async function deleteTerm(e) {
  // If custom dialog is available, wrap it into a Promise and only delete on positive click
  if (window.$dialog?.confirm) {
    const confirmed = await new Promise((resolve) => {
      window.$dialog.confirm(
        t('common.delete_confirm_detail', { title: e.source || '' }),
        {
          title: t('common.delete_confirm'),
          positiveText: t('common.delete'),
          negativeText: t('common.cancel'),
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        }
      )
    })
    if (!confirmed) return
  } else {
    if (!window.confirm(t('common.delete_confirm_detail', { title: e.source || '' }))) return
  }
  try {
    await subtitleService.deleteGlossaryEntry(e.id)
    entries.value = entries.value.filter(x => x.id !== e.id)
    if (editTerm.id === e.id) newDraft()
    $message?.success?.(t('common.deleted'))
  } catch { $message?.error?.(t('common.delete_failed')) }
}

// legacy autosave path removed; explicit New/Save model only

const canSaveSet = computed(() => !!(localSet.name && localSet.name.trim()))
const isReady = computed(() => !!localSet.id)
async function ensureSetSaved() {
  if (!canSaveSet.value) return false
  try {
    const payload = { name: localSet.name.trim(), description: (localSet.description || '').trim() }
    const saved = localSet.id ? await subtitleService.upsertGlossarySet({ id: localSet.id, ...payload }) : await subtitleService.upsertGlossarySet(payload)
    if (saved && saved.id) {
      localSet.id = saved.id
      localSet.name = saved.name || localSet.name
      localSet.description = saved.description || localSet.description
      emit('saved', saved)
      return true
    }
  } catch { $message?.error?.(t('common.save_failed')) }
  return false
}
async function onMetaBlur() { if (localSet.name && localSet.name.trim()) await ensureSetSaved() }
</script>

<style scoped>
.modal-card { border-radius: 12px; box-shadow: var(--macos-shadow-2); max-width: 700px; width: 100%; max-height: 85vh; overflow: hidden; }
.modal-header { display:flex; align-items:center; justify-content: space-between; height: 36px; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.title { flex:1; display:flex; align-items:center; justify-content:flex-end; font-weight: 600; }
.modal-body { padding: 12px 16px; display:flex; flex-direction: column; gap: 8px; max-height: calc(85vh - 36px - 48px); min-height: 0; }
.modal-footer { border-top: 1px solid rgba(255,255,255,0.16); padding: 8px 12px; display:flex; justify-content: center; }
.row-inline { display:flex; gap: 8px; }
.row-inline .col { flex: 1 1 0; min-width: 0; }
.field { display:flex; flex-direction: column; gap: 4px; }
.field-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); padding-left: 8px; }
.form-section { display:flex; flex-direction: column; gap:6px; margin-bottom: 10px; }
.lbl { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.divider { height: 1px; background: var(--macos-separator); margin: 8px 0 10px; opacity: 0.6; }
.section-title { font-weight: 600; margin-bottom: 6px; }
.section-bar { display:flex; align-items:center; justify-content: space-between; }
.header-actions { display:flex; align-items:center; gap: 8px; }
.terms-scroll { flex: 1 1 auto; overflow: auto; min-height: 0; scrollbar-gutter: stable; }
.edit-grid2 { display:grid; grid-template-columns: 1fr 1fr; gap: 8px; align-items: end; margin: 6px 0; }
.edit-grid2 .field.term { grid-column: 1 / span 1; }
.edit-grid2 .field.controls { grid-column: 2 / span 1; }
.controls-row { display:grid; grid-template-columns: 1fr auto 1fr; align-items:center; column-gap: 8px; }
.controls-row .left { justify-self: start; }
.controls-row .mid { justify-self: center; }
.controls-row .right { justify-self: end; display:flex; align-items:center; gap: 8px; }
.seg-toggle { display: inline-flex; white-space: nowrap; }
.seg-toggle .seg-item { white-space: nowrap; }
.controls-row .left, .controls-row .mid, .controls-row .right { display:flex; align-items:center; }
.left-inline-center { display:inline-flex; align-items:center; justify-content:flex-start; gap: 10px; min-height: 18px; }
.toggle-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); line-height: 18px; display:inline-flex; align-items:center; vertical-align: middle; white-space: nowrap; }
/* Adopt the same toggle visuals as Settings/About */
.about-toggle { position: relative; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; vertical-align: middle; }
.about-toggle-input { position: absolute; width: 0; height: 0; opacity: 0; }
.about-toggle-slider { width: 32px; height: 18px; border-radius: 999px; background: var(--macos-divider-weak); box-shadow: inset 0 0 0 1px var(--macos-divider-weak); transition: background 180ms ease, box-shadow 180ms ease; display: inline-block; position: relative; }
.about-toggle-slider::after { content: ""; position: absolute; width: 14px; height: 14px; border-radius: 50%; background: var(--macos-background); top: 2px; left: 2px; box-shadow: var(--macos-shadow-1); transition: transform 180ms ease; }
.about-toggle-input:checked + .about-toggle-slider { background: color-mix(in srgb, var(--macos-blue) 55%, transparent); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--macos-blue) 70%, transparent); }
.about-toggle-input:checked + .about-toggle-slider::after { transform: translateX(14px); background: var(--macos-background); box-shadow: 0 0 0 1px color-mix(in srgb, var(--macos-blue) 65%, transparent), 0 2px 4px rgba(0,0,0,0.12); }
.about-toggle-input:focus-visible + .about-toggle-slider { outline: 2px solid rgba(var(--macos-blue-rgb), 0.7); outline-offset: 2px; }
.edit-grid { display:grid; grid-template-columns: 1fr 160px auto; gap: 8px; align-items: end; margin: 6px 0; grid-auto-flow: row dense; }
.edit-grid .field.term { grid-column: 1 / span 1; }
.edit-grid .field.case { grid-column: 2 / span 1; }
.edit-grid .field.mode { grid-column: 3 / span 1; justify-self: end; align-items: flex-end; }
.edit-grid .field.actions { display:none; }
.seg-toggle .seg-item { height: 28px; padding: 0 10px; }
.seg-toggle { display: inline-flex; white-space: nowrap; }
.seg-toggle .seg-item { white-space: nowrap; }
.select-tight { height: 34px; line-height: 22px; padding: 6px 12px; }
.edit-grid2 .select-tight, .edit-second2 .select-tight { height: 34px; }
.check.small { font-size: var(--fs-sub); color: var(--macos-text-primary); display:flex; align-items:center; gap: 6px; }
.checkbox-macos { width: 14px; height: 14px; accent-color: var(--macos-blue); }
.gl-table { border: 1px solid var(--macos-separator); border-radius: 8px; margin-top: 6px; display:flex; flex-direction: column; max-height: 100%; flex: 1 1 auto; min-height: 0; overflow: hidden; }
.gl-head { display:grid; grid-template-columns: minmax(0,1fr) 140px 300px 60px; align-items:center; gap: 0; background: var(--macos-background-secondary); padding: 6px 8px; font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.gl-head .cell { text-align: left; }
.gl-head .cell.act { text-align: center; padding-right: 0; }
.gl-head .cell { padding: 0 6px; }
.gl-body { display:block; }
.row { display:grid; grid-template-columns: minmax(0,1fr) 140px 300px 60px; align-items:center; padding: 6px 8px; border-top: 1px solid var(--macos-separator); }
.cell.case { text-align: center; }
.cell.term { overflow: hidden; }
.cell.term .one-line { display: inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }
.cell.trans { overflow: hidden; }
.cell.trans .one-line { display: inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }
.row .cell { padding: 0 6px; }
.row .mono { font-family: var(--font-mono); }
.row-detail { padding: 8px; border-top: 1px dashed var(--macos-separator); background: var(--macos-background); }
.tr-table { border: 1px solid var(--macos-separator); border-radius: 6px; overflow: hidden; }
.tr-head { display:grid; grid-template-columns: 240px 1fr 80px; background: var(--macos-background-secondary); padding: 6px 8px; font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.tr-body { display:block; }
.tr { display:grid; grid-template-columns: 240px 1fr 80px; align-items:center; padding: 6px 8px; border-top: 1px solid var(--macos-separator); }
.td.lang .select-macos { width: 100%; }
.td.val .input-macos { width: 100%; }
.cell.act .act-row { display:flex; align-items:center; gap: 6px; justify-content: center; }
.row { cursor: pointer; }
.row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
.gl-entry .mono.src { font-family: var(--font-mono); }
.actions-center { display:flex; align-items:center; justify-content:center; gap:10px; margin-top: 8px; }
.hint { font-size: var(--fs-sub); }
.empty-state { text-align:center; border: 1px dashed var(--macos-separator); border-radius: 8px; padding: 16px; color: var(--macos-text-secondary); }
.empty-icon { width: 40px; height: 40px; border-radius: 999px; background: var(--macos-background-secondary); display:flex; align-items:center; justify-content:center; margin: 0 auto 8px; }
.empty-title { font-weight: 600; color: var(--macos-text-primary); }
.empty-hint { margin-top: 4px; font-size: var(--fs-sub); }
.edit-second { display:none; }
.edit-second2 { display:grid; grid-template-columns: 1fr 1fr; gap: 8px; align-items: end; margin: 4px 0 6px; }
</style>
