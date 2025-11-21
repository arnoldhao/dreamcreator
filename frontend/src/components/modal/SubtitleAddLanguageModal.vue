<template>

  <div v-if="show" class="macos-modal" @click.self="emit('close')">
    <div class="modal-card card-frosted card-translucent" @keydown.esc.stop.prevent="emit('close')" tabindex="-1">
      <!-- Header: macOS traffic lights + segmented mode switch -->
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title-area">
          <!-- 单个 Chip 内的分段切换 -->
          <div class="seg-chip chip-sm chip-frosted chip-translucent" role="tablist" aria-label="Mode">
            <button role="tab" class="seg-item" :class="{ active: activeTab==='zhconvert' }" :aria-selected="activeTab==='zhconvert'" @click="activeTab='zhconvert'">{{ $t('subtitle.add_language.zhconvert') }}</button>
            <button role="tab" class="seg-item" :class="{ active: activeTab==='llm' }" :aria-selected="activeTab==='llm'" @click="activeTab='llm'">{{ $t('subtitle.add_language.llm') }}</button>
          </div>
        </div>
      </div>

      <!-- Modal内容 -->
      <div class="modal-body">
        <!-- ZHConvert 标签页 -->
        <div v-if="activeTab === 'zhconvert'" class="tab-content">
          <!-- 中文转换：左右下拉框（无中间图标） -->
            <div class="dual-line no-arrow">
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.source_language') }}</div>
                <select v-model="selectedSourceLanguage" class="select-macos select-dual select-tight">
                  <option value="">{{ $t('subtitle.add_language.select_source_language') }}</option>
                  <option v-for="lang in availableSourceLanguages" :key="lang.code" :value="lang.code">{{ lang.name }}</option>
                </select>
              </div>
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.converter_type') }}</div>
                <div v-if="loading" class="loading-state" style="justify-content:center">
                  <div class="loading-spinner"></div>
                  <span>{{ $t('subtitle.add_language.loading_converters') }}</span>
                </div>
                <template v-else>
                  <select v-model="selectedConverter" class="select-macos select-dual select-tight">
                    <option value="">{{ $t('subtitle.add_language.select_converter') }}</option>
                    <option v-for="converter in converters" :key="converter" :value="converter">{{ getConverterDisplayName(converter) }}</option>
                  </select>
                </template>
              </div>
            </div>
            <div v-if="selectedConverter" class="converter-description-box" style="margin-top:10px">
              <div class="description-icon">
                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                </svg>
              </div>
              <span class="description-text">{{ getConverterDescription(selectedConverter) }}</span>
            </div>
            <div v-else-if="!loading && converters.length === 0" class="empty-state">
              <svg class="w-8 h-8 text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              <p class="text-sm text-gray-500">{{ $t('subtitle.add_language.no_converters_available') }}</p>
            </div>
            </div>
          
        <!-- LLM 标签页 -->
        <div v-else-if="activeTab === 'llm'" class="tab-content">
          <!-- AI 翻译：源/目标语言（无中间图标） -->
            <div class="dual-line no-arrow">
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.source_language') }}</div>
                <select v-model="selectedSourceLanguage" class="select-macos select-dual select-tight">
                  <option value="">{{ $t('subtitle.add_language.select_source_language') }}</option>
                  <option v-for="lang in availableSourceLanguages" :key="lang.code" :value="lang.code">{{ lang.name }}</option>
                </select>
              </div>
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.target_language') }}</div>
                <select v-model="targetLanguage" class="select-macos select-dual select-tight">
                  <option value="">{{ $t('subtitle.add_language.select_target_language') }}</option>
                  <option v-for="opt in targetLanguageOptions" :key="opt" :value="opt">{{ getLangName(opt) }}</option>
                </select>
              </div>
            </div>
          <!-- Provider 与 Model：左右并列（无箭头） -->
            <div class="dual-line no-arrow provider-row">
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.provider') }}</div>
                <select v-model="providerID" class="select-macos select-dual select-tight" @change="onProviderChange">
                  <option value="">{{ $t('subtitle.add_language.select_provider') }}</option>
                  <option v-for="p in providers" :key="p.id" :value="p.id">{{ p.name }}</option>
                </select>
              </div>
              <div class="dual-col">
                <div class="field-label model-label">
                  <span>{{ $t('subtitle.add_language.model') }}</span>
                  <button
                    class="btn-chip-icon btn-xxs model-refresh-btn"
                    :data-tooltip="$t('common.refresh')"
                    @click="refreshModelList"
                    :disabled="!providerID || modelsRefreshing"
                  >
                    <div v-if="modelsRefreshing" class="loading-spinner tiny"></div>
                    <Icon v-else name="refresh" class="w-3 h-3" />
                  </button>
                </div>
                <select v-model="model" class="select-macos select-dual select-tight">
                  <option value="">{{ $t('subtitle.add_language.select_model') }}</option>
                  <option v-for="m in modelOptions" :key="m" :value="m">{{ m }}</option>
                </select>
              </div>
            </div>
            <!-- Profile（可选）：独立一行，选择 Profile 后自动带出 model；右侧同列放置“仅重试失败片段” -->
            <div class="dual-line no-arrow" style="margin-top:8px">
              <div class="dual-col">
                <div class="field-label">{{ $t('subtitle.add_language.profile') }} <span class="muted">({{ $t('common.optional') }})</span></div>
                <select v-model="globalProfileID" class="select-macos select-dual select-tight" @change="onGlobalProfileChange()">
                  <option value="">{{ $t('subtitle.add_language.select_profile') }}</option>
                  <option v-for="pf in globalProfiles" :key="pf.id" :value="pf.id">{{ pf.name || ('Profile ' + pf.id) }} (T={{ pf.temperature ?? 0 }}, TopP={{ pf.top_p ?? 1 }}, {{ pf.json_mode ? 'JSON' : 'Text' }})</option>
                </select>
              </div>
              <div class="dual-col">
                <div class="field-label">&nbsp;</div>
                <!-- 显示条件：
                     1) 片段层存在失败记录（严格：status ∈ {error,fallback}），或
                     2) 语言层 sync_status 为 failed/partial_failed（例如任务中途失败未写入片段标记）
                -->
                <div class="left-inline-center" v-if="targetLanguage && (failedCountStrictForTarget > 0 || isLangFailedOrPartial)">
                  <label class="about-toggle">
                    <input type="checkbox" class="about-toggle-input" v-model="retryFailedOnly" />
                    <span class="about-toggle-slider"></span>
                  </label>
                  <span class="toggle-label">{{ $t('subtitle.add_language.retry_failed_only') }} ({{ failedCountStrictForTarget }})</span>
                </div>
              </div>
            </div>
          <!-- 分割线对齐 Glossary -->
          <div class="divider"></div>
          <!-- 全局术语（多选）：宽度与“源语言”框一致，直接摆放，无额外容器盒子 -->
          <div class="dual-line no-arrow sets-row">
            <div class="dual-col">
              <div class="field-label">{{ $t('subtitle.add_language.global_glossary_sets') }}</div>
              <div class="tag-input">
                <template v-if="selectedSets.length">
                  <span v-for="s in selectedSets" :key="s.id" class="chip-frosted chip-sm" :title="s.name">
                    <span class="chip-label">{{ shortSetName(s.name) }}</span>
                    <button class="chip-action" :data-tooltip="$t('common.delete')" @click.stop="removeSet(s.id)"><Icon name="close" class="w-3 h-3"/></button>
                  </span>
                </template>
                <span v-else class="placeholder">{{ $t('subtitle.add_language.no_set_selected') }}</span>
              </div>
            </div>
            <div class="dual-col">
              <div class="field-label">&nbsp;</div>
              <div class="right-inline-center">
                <div class="set-attach-row">
                  <select class="select-macos select-tight" v-model="setToAdd">
                    <option value="">{{ $t('subtitle.add_language.select_set') }}</option>
                    <option v-for="opt in availableSetOptions" :key="opt.id" :value="opt.id">{{ opt.name }}</option>
                  </select>
                  <button class="btn-chip-icon" :data-tooltip="$t('subtitle.add_language.use_set')" @click="useSelectedSet" :disabled="!setToAdd">
                    <Icon name="plus" class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
          </div>
          <!-- 分割线对齐 Glossary -->
          <div class="divider"></div>
          <!-- 术语高级选项：严格模式 + 自定义术语开关 -->
          <div class="term-advanced-row">
            <div class="field-label">{{ $t('subtitle.add_language.advanced_glossary_options') }}</div>
            <div class="advanced-toggles">
              <label class="about-toggle">
                <input type="checkbox" class="about-toggle-input" v-model="strictGlossary" />
                <span class="about-toggle-slider"></span>
              </label>
              <span class="toggle-label">{{ $t('subtitle.add_language.strict_glossary') }}</span>
              <label class="about-toggle">
                <input type="checkbox" class="about-toggle-input" v-model="customGlossaryEnabled" />
                <span class="about-toggle-slider"></span>
              </label>
              <span class="toggle-label">{{ $t('subtitle.add_language.custom_glossary') }}</span>
            </div>
          </div>
          <!-- 当前字幕术语表（任务内 + 可持久化） -->
          <!-- 分割线 + 直接摆放（对齐 GlossaryTermsModal） -->
          <template v-if="customGlossaryEnabled">
          <div class="divider"></div>
            <!-- 编辑工具条：术语 + DNT/指定 + 新建/保存（对齐 GlossaryTermsModal 样式与交互） -->
            <div class="edit-grid2">
              <div class="field term">
                <div class="field-label">{{ t('glossary.term') }}</div>
                <input class="input-macos" v-model="editTerm.source" :placeholder="t('glossary.term_placeholder')" @input="resetTracking" />
              </div>
              <div class="field controls">
                <div class="controls-row">
                  <div class="left">
                    <div class="left-inline-center">
                      <label class="about-toggle">
                        <input type="checkbox" class="about-toggle-input" v-model="editTerm.case_sensitive" />
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
                    <button class="btn-chip-icon" :data-tooltip="t('common.add')" @click="newDraft"><Icon name="plus" class="w-4 h-4"/></button>
                    <button class="btn-chip-icon" :data-tooltip="t('common.save')" :disabled="!canSaveTerm || !isDirtyTerm" @click="saveDraft"><Icon name="save" class="w-4 h-4"/></button>
                  </div>
                </div>
              </div>
            </div>
            <div v-if="editMode==='specify'" class="edit-second2">
              <div class="field">
                <div class="field-label">{{ t('glossary.lang') }}</div>
                <select class="select-macos select-tight" v-model="topLang">
                  <option value="">{{ t('subtitle.add_language.select_target_language') }}</option>
                  <option v-for="opt in glLangOptions" :key="opt" :value="opt">{{ getLangName(opt) }}</option>
                </select>
              </div>
              <div class="field">
                <div class="field-label">{{ t('glossary.translation_placeholder') }}</div>
                <input class="input-macos" v-model="topTrans" :placeholder="t('glossary.translation_placeholder')" />
              </div>
            </div>

            <!-- 列表 -->
            <div class="gl-table" v-if="taskGlossary.length">
              <div class="gl-head">
                <div class="h cell term">{{ t('glossary.term') }} <span class="muted">{{ t('glossary.terms_count', { count: taskGlossary.length }) }}</span></div>
                <div class="h cell case">{{ t('glossary.case_sensitive') }}</div>
                <div class="h cell trans">{{ t('glossary.translation_placeholder') }}</div>
                <div class="h cell act">&nbsp;</div>
              </div>
              <div class="terms-scroll">
                <div class="gl-body">
                  <div class="row" v-for="(g, idx) in taskGlossary" :key="g.id || idx" @click="selectForEdit(g, idx)">
                    <div class="cell term mono" :title="g.source"><span class="one-line">{{ g.source }}</span></div>
                    <div class="cell case"><span class="muted">{{ g.case_sensitive ? t('common.yes') : t('common.no') }}</span></div>
                    <div class="cell trans"><span class="one-line" :title="getSingleTranslation(g)">{{ g.do_not_translate ? t('glossary.dnt') : getSingleTranslation(g) }}</span></div>
                    <div class="cell act">
                      <div class="act-row">
                        <button class="btn-chip-icon btn-danger" :data-tooltip="t('common.delete')" @click.stop="deleteTerm(g, idx)"><Icon name="trash" class="w-4 h-4"/></button>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
            <div v-else class="empty-state">
              <div class="empty-icon"><Icon name="info" class="w-6 h-6" /></div>
              <div class="empty-title">{{ t('glossary.no_terms') }}</div>
              <div class="empty-hint">{{ t('glossary.no_terms_hint') }}</div>
            </div>
          </template>
        </div>
      </div>
      <div class="modal-footer">
        <button @click="emit('close')" class="btn-chip">{{ $t('common.cancel') }}</button>
        <button @click="handleConvert" class="btn-chip btn-primary" :disabled="!canConvert">
          <div v-if="converting" class="loading-spinner"></div>
          {{ converting ? $t('subtitle.add_language.converting') : $t('subtitle.add_language.start_convert') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, reactive, computed, onMounted, defineProps, defineEmits, watch, toRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import useInspectorStore from '@/stores/inspector.js'
import { useTargetLanguagesStore } from '@/stores/targetLanguages.js'
import { listEnabledProviders, listGlobalProfiles as apiListGlobalProfiles, refreshModels as apiRefreshModels, getProvider as apiGetProvider } from '@/services/llmProviderService.js'

const props = defineProps({
  show: Boolean,
  availableLanguages: {
    type: Array,
    default: () => []
  },
  subtitleService: {
    type: Object,
    required: true
  },
  prefill: {
    type: Object,
    default: () => ({})
  }
})

const emit = defineEmits(['close', 'convert-started'])

const { t } = useI18n()
const inspector = useInspectorStore()
const targetLangStore = useTargetLanguagesStore()

// 响应式数据
const activeTab = ref('zhconvert')
const selectedSourceLanguage = ref('')
const selectedConverter = ref('')
const converters = ref([])
const loading = ref(false)
const converting = ref(false)

// LLM
const providers = ref([])
const providerID = ref('')
const models = ref([])
const modelsRefreshing = ref(false)
// Normalize models to string array for UI rendering
const modelOptions = computed(() => {
  const src = Array.isArray(models.value) ? models.value : []
  const out = []
  for (const it of src) {
    if (typeof it === 'string') out.push(it)
    else if (it && typeof it === 'object') {
      const s = it.name || it.model || it.id || it.value || ''
      if (s) out.push(String(s))
    }
  }
  // de-dup
  return Array.from(new Set(out))
})
  const model = ref('')
  const globalProfiles = ref([])
  const globalProfileID = ref('')
  const customGlossaryEnabled = ref(false)
  const targetLanguage = ref('')
  const targetLanguages = computed(() => targetLangStore.list || []) // [{code,name}]
  const targetLanguageOptions = computed(() => (targetLanguages.value || []).map(x => x.code))
  const retryFailedOnly = ref(false)
  const failedCountForTarget = computed(() => {
    // Align with backend filter (方案 A):
    // include segments with no target language, or no process, or status in {error, fallback}
    try {
      const proj = props.subtitleService?.currentProject
      const lang = targetLanguage.value
      if (!proj || !lang) return 0
      const segs = Array.isArray(proj?.segments) ? proj.segments : []
      let cnt = 0
      for (const s of segs) {
        const lc = s?.languages?.[lang]
        if (!lc) { cnt++; continue }
        if (!lc.process) { cnt++; continue }
        const st = String(lc.process.status || '').toLowerCase()
        if (st === 'fallback' || st === 'error') cnt++
      }
      return cnt
    } catch { return 0 }
  })

  // 严格失败计数：仅统计“已有目标语言且标记 error/fallback”的片段
  const failedCountStrictForTarget = computed(() => {
    try {
      const proj = props.subtitleService?.currentProject
      const lang = targetLanguage.value
      if (!proj || !lang) return 0
      const segs = Array.isArray(proj?.segments) ? proj.segments : []
      let cnt = 0
      for (const s of segs) {
        const lc = s?.languages?.[lang]
        if (!lc) continue
        const st = String(lc?.process?.status || '').toLowerCase()
        if (st === 'fallback' || st === 'error') cnt++
      }
      return cnt
    } catch { return 0 }
  })

  // 语言层失败/部分失败（用于任务级失败但未落片段标记的情况）
  const isLangFailedOrPartial = computed(() => {
    try {
      const proj = props.subtitleService?.currentProject
      const lang = targetLanguage.value
      if (!proj || !lang) return false
      const meta = proj?.language_metadata?.[lang]
      const s = String(meta?.sync_status || '').toLowerCase()
      return s === 'failed' || s === 'partial_failed'
    } catch { return false }
  })
// glossary sets + task-only
const glossarySets = ref([])
const selectedSetIDs = ref([])
const setToAdd = ref('')
const taskGlossary = ref([])
// glossary mode: false => hint (default), true => strict (do not expose placeholders in prompt glossary)
const strictGlossary = ref(false)
// inline editor (align with GlossaryTermsModal)
const editTerm = reactive({ id: '', source: '', case_sensitive: false, do_not_translate: true, translations: {} })
const editMode = ref('dnt')
const topLang = ref('')
const topTrans = ref('')
const newEntryId = ref('')
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
function pickSingleTranslation(tr) {
  if (!tr) return { lang: '', val: '' }
  if (tr['all']) return { lang: 'all', val: tr['all'] }
  const keys = Object.keys(tr)
  if (keys.length === 0) return { lang: '', val: '' }
  keys.sort()
  const k = keys[0]
  return { lang: k, val: tr[k] || '' }
}
async function saveDraft() {
  if (!canSaveTerm.value) return
  // build entry from editor
  const entry = {
    id: editTerm.id || `tglo_${Date.now()}`,
    source: (editTerm.source || '').trim(),
    do_not_translate: (editMode.value === 'dnt'),
    case_sensitive: !!editTerm.case_sensitive,
    translations: {}
  }
  if (editMode.value === 'specify' && topLang.value && topTrans.value && topTrans.value.trim()) {
    entry.translations[topLang.value] = topTrans.value.trim()
  }
  // upsert into list
  const idx = taskGlossary.value.findIndex(x => (x.id && x.id === entry.id))
  if (idx >= 0) taskGlossary.value[idx] = entry; else taskGlossary.value.unshift(entry)
  // update editor ids and baseline
  editTerm.id = entry.id
  editTerm.translations = entry.translations
  editTerm.do_not_translate = entry.do_not_translate
  newEntryId.value = entry.id
  syncBaselineFromCurrent()
  // persist immediately
  await persistTaskGlossary()
  $message?.success?.(t('common.saved'))
}
function selectForEdit(e, _idx) {
  editTerm.id = e.id || ''
  editTerm.source = e.source || ''
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
  newEntryId.value = editTerm.id
  syncBaselineFromCurrent()
}
async function deleteTerm(e, idx) {
  // confirm dialog consistent with GlossaryTermsModal
  if (window.$dialog?.confirm) {
    const confirmed = await new Promise((resolve) => {
      window.$dialog.confirm(
        t('common.delete_confirm_detail', { title: e?.source || '' }),
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
    if (!window.confirm(t('common.delete_confirm_detail', { title: e?.source || '' }))) return
  }
  if (typeof idx === 'number') taskGlossary.value.splice(idx, 1)
  else taskGlossary.value = taskGlossary.value.filter(x => (x.id && x.id !== e.id))
  await persistTaskGlossary()
  if (editTerm.id === e.id) newDraft()
  $message?.success?.(t('common.deleted'))
}

// 计算属性
const availableSourceLanguages = computed(() => {
  return props.availableLanguages.map(lang => ({
    code: lang,
    name: getLanguageDisplayName(lang)
  }))
})

function shortSetName(name) {
  const n = String(name || '')
  const limit = 12
  if (n.length <= limit) return n
  return n.slice(0, limit) + '…'
}

const selectedSets = computed(() => {
  const map = new Map((glossarySets.value || []).map(s => [String(s.id), s]))
  return (selectedSetIDs.value || []).map(id => map.get(String(id))).filter(Boolean)
})
const availableSetOptions = computed(() => {
  const used = new Set((selectedSetIDs.value || []).map(id => String(id)))
  return (glossarySets.value || []).filter(s => !used.has(String(s.id)))
})
function removeSet(id) {
  selectedSetIDs.value = (selectedSetIDs.value || []).filter(x => String(x) !== String(id))
}
function useSelectedSet() {
  if (!setToAdd.value) return
  const exists = (selectedSetIDs.value || []).some(x => String(x) === String(setToAdd.value))
  if (!exists) selectedSetIDs.value.push(setToAdd.value)
  setToAdd.value = ''
}

const canConvert = computed(() => {
  if (activeTab.value === 'zhconvert') {
    return selectedSourceLanguage.value && selectedConverter.value && !converting.value
  }
  // llm
  if (globalProfileID.value) {
    return selectedSourceLanguage.value && targetLanguage.value && providerID.value && model.value && globalProfileID.value && !converting.value
  }
  return selectedSourceLanguage.value && targetLanguage.value && providerID.value && model.value && !converting.value
})

// 方法
const loadConverters = async () => {
  try {
    loading.value = true
    const supportedConverters = await props.subtitleService.loadSupportedConverters()
    converters.value = supportedConverters || []
  } catch (error) {
    $message.error(t('subtitle.add_language.load_converters_failed'))
    converters.value = []
  } finally {
    loading.value = false
  }
}

  const handleConvert = async () => {
    if (!canConvert.value) return
    try {
      converting.value = true
      if (activeTab.value === 'zhconvert') {
        await props.subtitleService.convertSubtitle(selectedSourceLanguage.value, selectedConverter.value)
        emit('convert-started', { sourceLanguage: selectedSourceLanguage.value, converter: selectedConverter.value })
      } else {
        const src = String(selectedSourceLanguage.value || '').trim().toLowerCase()
        const tgt = String(targetLanguage.value || '').trim().toLowerCase()
        if (src && tgt && src === tgt) {
          $message.warning(t('subtitle.add_language.same_language_warning') || 'Source and target cannot be the same')
          return
        }
        const extras = customGlossaryEnabled.value ? taskGlossary.value : []
        if (globalProfileID.value) {
          if (retryFailedOnly.value) {
            await props.subtitleService.retryFailedTranslationsWithGlobalProfile(selectedSourceLanguage.value, targetLanguage.value, providerID.value, model.value, globalProfileID.value, selectedSetIDs.value, extras, strictGlossary.value)
          } else {
            await props.subtitleService.translateSubtitleLLMWithGlobalProfileAndGlossary(selectedSourceLanguage.value, targetLanguage.value, providerID.value, model.value, globalProfileID.value, selectedSetIDs.value, extras, strictGlossary.value)
          }
          emit('convert-started', { sourceLanguage: selectedSourceLanguage.value, targetLanguage: targetLanguage.value, profileID: globalProfileID.value })
        } else {
          if (retryFailedOnly.value) {
            await props.subtitleService.retryFailedTranslations(selectedSourceLanguage.value, targetLanguage.value, providerID.value, model.value, selectedSetIDs.value, extras, strictGlossary.value)
          } else {
            await props.subtitleService.translateSubtitleLLMWithGlossary(selectedSourceLanguage.value, targetLanguage.value, providerID.value, model.value, selectedSetIDs.value, extras, strictGlossary.value)
          }
          emit('convert-started', { sourceLanguage: selectedSourceLanguage.value, targetLanguage: targetLanguage.value, providerID: providerID.value, model: model.value })
        }
      }
      $message.success(t('subtitle.add_language.conversion_started'))
      // Proactively refresh AI tasks + projects so Inspector/Subtitle update immediately
      try {
        const { useSubtitleTasksStore } = await import('@/stores/subtitleTasks')
        const { useSubtitleStore } = await import('@/stores/subtitle')
        const st = useSubtitleTasksStore()
        const ss = useSubtitleStore()
        await st.loadAll()
        await ss.fetchProjects({ force: true, showLoading: false })
      } catch {}
      // 打开 Inspector 并切换到任务面板
      try { inspector.open('SubtitleTasksPanel', t('subtitle.tasks_title') || 'Subtitle Translation Tasks') } catch {}
      emit('close')
    } catch (error) {
    console.error('Conversion failed:', error)
    $message.error(error.message || t('subtitle.add_language.conversion_failed'))
  } finally { converting.value = false }
}

const getLanguageDisplayName = (langCode) => {
  return targetLangStore.getName(langCode)
}

const getConverterDisplayName = (converter) => {
  const converterNames = {
    'Simplified': '简体中文',
    'Traditional': '繁体中文',
    'China': '中国大陆简体',
    'Hongkong': '香港繁体',
    'Taiwan': '台湾繁体',
    'Pinyin': '拼音',
    'Bopomofo': '注音符号',
    'Mars': '火星文',
    'WikiSimplified': '维基简体',
    'WikiTraditional': '维基繁体'
  }
  return converterNames[converter] || converter
}

const getConverterDescription = (converter) => {
  const descriptions = {
    'Simplified': '转换为简体中文',
    'Traditional': '转换为繁体中文',
    'China': '转换为中国大陆简体中文',
    'Hongkong': '转换为香港繁体中文',
    'Taiwan': '转换为台湾繁体中文',
    'Pinyin': '转换为拼音',
    'Bopomofo': '转换为注音符号',
    'Mars': '转换为火星文',
    'WikiSimplified': '转换为维基百科简体中文',
    'WikiTraditional': '转换为维基百科繁体中文'
  }
  return descriptions[converter] || ''
}

// 生命周期
onMounted(() => {
  if (props.show) {
    loadConverters()
    initLLM()
  }
})

let prefillApplied = false

function applyPrefillIfAny() {
  try {
    const pf = props.prefill || {}
    if (!props.show || !pf || prefillApplied) return
    if (pf.tab && (pf.tab === 'zhconvert' || pf.tab === 'llm')) activeTab.value = pf.tab
    if (pf.sourceLang) selectedSourceLanguage.value = pf.sourceLang
    if (activeTab.value === 'zhconvert') {
      if (pf.converter) selectedConverter.value = pf.converter
      prefillApplied = true
      return
    }
    // LLM tab
    if (pf.targetLang) targetLanguage.value = pf.targetLang
    // Wait for providers, then set provider/model
    const setProviderAndModel = async () => {
      try {
        if (!providers.value || providers.value.length === 0) { return false }
        if (pf.providerId && providers.value.some(p => p.id === pf.providerId)) {
          providerID.value = pf.providerId
        } else if (pf.providerName) {
          const found = providers.value.find(p => String(p.name || '').toLowerCase() === String(pf.providerName || '').toLowerCase())
          if (found) { providerID.value = found.id }
        }
        // update models via provider change
        onProviderChange()
        await ensureModelsForProvider({ model: pf.model })
        if (pf.model && modelOptions.value && modelOptions.value.length && modelOptions.value.includes(pf.model)) {
          model.value = pf.model
        }
        if (typeof pf.retryFailedOnly === 'boolean') retryFailedOnly.value = pf.retryFailedOnly
        prefillApplied = true
        return true
      } catch { return false }
    }
    // Try immediately, otherwise wait for providers list to load
    setProviderAndModel().then((ok) => {
      if (ok) return
      const stop = watch(() => providers.value && providers.value.length, (len) => {
        if (len) { setProviderAndModel(); stop && stop() }
      }, { immediate: false })
    })
  } catch { /* ignore */ }
}

// 监听 show 属性变化
watch(() => props.show, (newValue) => {
  if (newValue) {
    // 重置表单
    selectedSourceLanguage.value = ''
    selectedConverter.value = ''
    activeTab.value = 'zhconvert'
    providerID.value = ''
    model.value = ''
    targetLanguage.value = ''
    retryFailedOnly.value = false
    prefillApplied = false

    // 加载转换器
    loadConverters()
    initLLM()
    // 尝试应用预填
    setTimeout(() => applyPrefillIfAny(), 0)
  }
})

async function initLLM() {
  // load target languages (used for both target select and glossary-lang options)
  try {
    await targetLangStore.ensureLoaded()
  } catch {}
  try { providers.value = await listEnabledProviders() } catch (e) { providers.value = [] }
  try { globalProfiles.value = await apiListGlobalProfiles() } catch (e) { globalProfiles.value = [] }
  // Only clear dependent fields when current provider is not set or no longer exists
  try {
    const hasProvider = !!(providerID.value && (providers.value || []).some(p => p && p.id === providerID.value))
    if (!hasProvider) {
      models.value = []
      model.value = ''
      globalProfileID.value = ''
    } else {
      /* keep existing selections */
    }
  } catch {}
  // glossary sets
  try {
    const sets = await props.subtitleService.listGlossarySets()
    glossarySets.value = Array.isArray(sets) ? sets : []
    // preselect default if exists
    const def = glossarySets.value.find(x => String(x?.name || '').toLowerCase() === 'default')
    selectedSetIDs.value = def ? [def.id] : []
  } catch { glossarySets.value = []; selectedSetIDs.value = [] }
  // load per-project task terms (assign id if missing for editing)
  try {
    const meta = props.subtitleService?.currentProject?.metadata || {}
    const saved = Array.isArray(meta.task_terms) ? meta.task_terms : []
    const arr = Array.isArray(saved) ? JSON.parse(JSON.stringify(saved)) : []
    taskGlossary.value = arr.map((e) => ({ id: e.id || `tglo_${Date.now()}_${Math.random().toString(16).slice(2)}`, ...e }))
  } catch { taskGlossary.value = [] }
}

// 当 providers 加载后，若有未应用的 prefill，则再次尝试
watch(() => providers.value && providers.value.length, () => applyPrefillIfAny())

/* debug watchers removed */

function onProviderChange() {
  const p = providers.value.find(x => x.id === providerID.value)
  const arr = Array.isArray(p?.models) ? p.models : (Array.isArray(p?.Models) ? p.Models : [])
  models.value = arr
  model.value = ''
  globalProfileID.value = ''
}

async function ensureModelsForProvider(pref = {}) {
  try {
    const pid = providerID.value
    if (!pid) { return false }
    const prefModel = pref?.model
    const forceRefresh = !!pref?.forceRefresh
    let p0 = providers.value.find(x => x.id === pid)
    let arr = Array.isArray(p0?.models) ? p0.models : (Array.isArray(p0?.Models) ? p0.Models : [])
    if (arr && arr.length) { // existing cache
      models.value = arr
      if (!forceRefresh && (!prefModel || arr.includes(prefModel))) { return true }
    }
    if (forceRefresh || !arr || !arr.length || (prefModel && !arr.includes(prefModel))) {
      try { await apiRefreshModels(pid) } catch (e) { /* ignore */ }
    }
    let p1 = null
    try { p1 = await apiGetProvider(pid) } catch (e) { /* ignore */ }
    if (p1 && p1.id) {
      const idx = providers.value.findIndex(x => x.id === pid)
      if (idx >= 0) providers.value[idx] = p1
      arr = Array.isArray(p1?.models) ? p1.models : (Array.isArray(p1?.Models) ? p1.Models : [])
      models.value = arr || []
      return !!(arr && arr.length && (!prefModel || arr.includes(prefModel)))
    }
    return false
  } catch { return false }
}

async function refreshModelList() {
  if (!providerID.value || modelsRefreshing.value) return
  modelsRefreshing.value = true
  try {
    await ensureModelsForProvider({ model: model.value, forceRefresh: true })
    $message?.success?.(t('common.refreshed') || 'Refreshed')
  } catch (e) {
    console.error('Refresh models failed:', e)
    $message?.error?.(e?.message || t('common.refresh_failed') || 'Refresh failed')
  } finally {
    modelsRefreshing.value = false
  }
}

function onGlobalProfileChange() { /* noop */ }
// profiles are managed via inspector panel (not from this modal)

function toggleCustomGlossary() { customGlossaryEnabled.value = !customGlossaryEnabled.value }

async function persistTaskGlossary() {
  const proj = props.subtitleService?.currentProject
  if (!proj) { $message?.error?.('No project'); return }
  try {
    const meta = JSON.parse(JSON.stringify(proj.metadata || {}))
    meta.task_terms = toRaw(taskGlossary.value)
    await props.subtitleService.saveProjectMetadata(meta)
  } catch (e) {
    console.error('Persist task glossary failed:', e)
    $message?.error?.(t('common.save_failed'))
  }
}

// helpers for glossary visuals
const glLangOptions = computed(() => ['all', ...(targetLanguageOptions.value || [])])
const langNameMap = computed(() => targetLangStore.nameMap || {})
function getLangName(code) {
  if (code === 'all') return t('glossary.lang_all')
  return targetLangStore.getName(code)
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
 </script>

<style scoped>
/* use global .macos-modal */

.modal-card { border-radius: 12px; box-shadow: var(--macos-shadow-2); max-width: 700px; width: 100%; max-height: 85vh; overflow: hidden; animation: slideInUp 0.3s ease-out; }
/* Always-on active frosted look */
.modal-card.card-frosted.card-translucent { background: color-mix(in oklab, var(--macos-surface) 76%, transparent); border: 1px solid rgba(255,255,255,0.28); box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0,0,0,0.24); }
.modal-header { display:flex; align-items:center; justify-content: space-between; height: 36px; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.title-area { flex:1; min-width: 0; display:flex; align-items:center; justify-content:flex-end; }
/* 单个 Chip 内的分段切换（融合两项） */
.title-area .seg-chip .seg-item { font-size: var(--fs-sub); }
.modal-body { padding: 12px 16px; display:flex; flex-direction: column; gap: 12px; max-height: calc(85vh - 36px - 48px); min-height: 0; overflow-y: auto; overflow-x: hidden; }
.modal-footer { border-top: 1px solid rgba(255,255,255,0.16); padding: 8px 12px; display:flex; justify-content: center; gap: 12px; }
/* section heading */
.section-title { font-weight: 600; margin-bottom: 10px; }

/* segmented（使用全局基础样式），仅作细节尺寸微调 */
.segmented .seg-item { min-width: 80px; height: 26px; padding: 0 10px; font-size: var(--fs-sub); }
.segmented .seg-item.disabled { opacity: .6; cursor: not-allowed; }

/* Compact selects */
/* Remove fixed widths to let controls fill grid columns */
/* Boxed sections for consistent macOS-like grouping */
/* 已移除术语区域的盒状容器，保持与 Glossary 一致的扁平布局 */
/* .form-section 仍用于非术语区域（如转换/LLM 块）的分组需要时可保留 */
.form-section { border: 1px solid var(--macos-separator); border-radius: 10px; padding: 12px; background: var(--macos-background); }
.form-section + .form-section { margin-top: 10px; }

/* Combined dual controls (left/right half) */
.dual-line { display:grid; grid-template-columns: 1fr 1fr; align-items: end; column-gap: 12px; }
.dual-line.no-arrow { grid-template-columns: 1fr 1fr; }
.dual-col { display:flex; flex-direction: column; gap: 8px; }
.field { display:flex; flex-direction: column; gap: 8px; }
/* spacing above provider/model row */
.provider-row { margin-top: 12px; }
.model-label { display:flex; align-items:center; justify-content: space-between; gap: 8px; }
.model-refresh-btn { margin-left: auto; border-radius: 999px; width: 22px; height: 22px; padding: 0; display:inline-flex; align-items:center; justify-content:center; }
.model-refresh-btn .loading-spinner { width: 14px; height: 14px; }
.right-inline-center { display:flex; align-items:center; justify-content:flex-end; gap: 10px; }
.set-attach-row { margin-left: auto; display:flex; align-items:center; gap: 8px; flex-wrap: nowrap; }
.set-attach-row .select-macos { width: 100%; min-width: 120px; max-width: 180px; white-space: nowrap; text-overflow: ellipsis; overflow: hidden; }
.toggle-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); line-height: 18px; display: inline-flex; align-items: center; vertical-align: middle; white-space: nowrap; }
.left-inline-center { display:inline-flex; align-items:center; justify-content:flex-start; gap: 10px; min-height: 18px; }
/* Adopt the same toggle visuals as Settings/About */
.about-toggle { position: relative; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; vertical-align: middle; }
.about-toggle-input { position: absolute; width: 0; height: 0; opacity: 0; }
.about-toggle-slider { width: 32px; height: 18px; border-radius: 999px; background: var(--macos-divider-weak); box-shadow: inset 0 0 0 1px var(--macos-divider-weak); transition: background 180ms ease, box-shadow 180ms ease; display: inline-block; position: relative; }
.about-toggle-slider::after { content: ""; position: absolute; width: 14px; height: 14px; border-radius: 50%; background: var(--macos-background); top: 2px; left: 2px; box-shadow: var(--macos-shadow-1); transition: transform 180ms ease; }
.about-toggle-input:checked + .about-toggle-slider { background: color-mix(in srgb, var(--macos-blue) 55%, transparent); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--macos-blue) 70%, transparent); }
.about-toggle-input:checked + .about-toggle-slider::after { transform: translateX(14px); background: var(--macos-background); box-shadow: 0 0 0 1px color-mix(in srgb, var(--macos-blue) 65%, transparent), 0 2px 4px rgba(0,0,0,0.12); }
.about-toggle-input:focus-visible + .about-toggle-slider { outline: 2px solid rgba(var(--macos-blue-rgb), 0.7); outline-offset: 2px; }
/* no intermediate column in no-arrow mode; mid placeholder not used */
.field-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); padding-left: 8px; }
.select-dual { width: 100%; }
.select-tight { height: 34px; line-height: 22px; padding: 6px 12px; }
/* unify fixed widths for selects/inputs */
/* removed fixed select width */
/* Glossary 风格的分割线与复选文本样式 */
.divider { height: 1px; background: var(--macos-separator); margin: 8px 0 10px; opacity: 0.6; }
.check.small { font-size: var(--fs-sub); color: var(--macos-text-primary); display:flex; align-items:center; gap: 6px; }

/* Tag input for selected sets */
.tag-input { @extend .input-macos; display:flex; align-items:center; flex-wrap: wrap; gap: 6px; min-height: 34px; padding: 6px 8px; width: 100%; max-width: 100%; overflow: visible; }
.tag-input .placeholder { color: var(--macos-text-tertiary); font-size: var(--fs-sub); }
.tag-input .chip-frosted .chip-label { max-width: 120px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.sets-tag-row {
  justify-content: space-between;
}
.sets-tag-left {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  flex: 1 1 auto;
  min-width: 0;
}
.sets-tag-right {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: 0 0 auto;
  margin-left: 8px;
}

.custom-glossary-toggle-row {
  /* deprecated: kept for backward compatibility if referenced elsewhere */
}

.term-advanced-row {
  margin-top: 6px;
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: center;
  gap: 4px;
}
.advanced-toggles {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
}

/* Ensure sets row left grows, right stays tight */
.sets-row { grid-template-columns: minmax(0,1fr) auto; }
.sets-row .toggle-label { max-width: 140px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.tab-navigation {
  display: flex;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
}

.tab-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--macos-text-secondary);
  font-size: var(--fs-title);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.tab-button:hover:not(:disabled) {
  background: var(--macos-gray-hover);
  color: var(--macos-text-primary);
}

.tab-button.active {
  color: var(--macos-blue);
  background: var(--macos-background);
}

.tab-button.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--macos-blue);
}

.tab-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.tab-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.coming-soon-badge {
  font-size: var(--fs-micro);
  padding: 2px 6px;
  background: var(--macos-orange);
  color: white;
  border-radius: 10px;
  margin-left: 4px;
}

.modal-content {
  padding: 20px;
  max-height: 400px;
  overflow-y: auto;
}

.tab-content {
    min-height: 200px;
  display: block;
  visibility: visible;
}

.form-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 12px 0;
}

.select-macos {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--macos-separator);
  border-radius: 6px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: var(--fs-base);
  transition: border-color 0.2s ease;
}

/* Ensure text inputs match select width within grid columns */
.input-macos {
  width: 100%;
}

.select-macos:focus {
  outline: none;
  border-color: var(--macos-blue);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--macos-blue) 15%, transparent);
}

.loading-state {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px;
  text-align: center;
  color: var(--macos-text-secondary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 20px;
  text-align: center;
}
.empty-state .empty-title { font-size: var(--fs-sub); font-weight: 600; color: var(--macos-text-primary); margin-top: 6px; }
.empty-state .empty-hint { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-top: 4px; }

.converter-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.converter-option {
  display: flex;
  align-items: center;
  padding: 12px;
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.converter-option:hover {
  background: var(--macos-gray-hover);
  border-color: var(--macos-blue);
}

.converter-option.active {
  background: color-mix(in oklab, var(--macos-blue) 10%, var(--macos-background));
  border-color: var(--macos-blue);
}

.converter-radio {
  margin-right: 12px;
}

.converter-content {
  flex: 1;
}

.converter-name {
  font-weight: 500;
  color: var(--macos-text-primary);
  margin-bottom: 4px;
}

.converter-description {
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
}


.converter-description-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 12px;
  background: color-mix(in oklab, var(--macos-blue) 10%, var(--macos-background));
  border: 1px solid color-mix(in oklab, var(--macos-blue) 20%, transparent);
  border-radius: 6px;
  font-size: var(--fs-base);
  color: color-mix(in oklab, var(--macos-blue) 80%, var(--macos-text-secondary));
}

.description-icon {
  display: flex;
  align-items: center;
  color: var(--macos-blue);
}

.description-text {
  flex: 1;
}

/* 删除了固定蓝色回退，统一使用变量以支持自定义主题色 */

.coming-soon-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.coming-soon-icon {
  margin-bottom: 16px;
}

.coming-soon-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 8px 0;
}

.coming-soon-description {
  font-size: var(--fs-title);
  color: var(--macos-text-secondary);
  margin: 0;
}

/* remove legacy footer visuals */

.loading-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid transparent;
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }

  to {
    opacity: 1;
  }
}

@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* Task glossary editor/list — follow GlossaryTermsModal classes */
.edit-grid2 { display:grid; grid-template-columns: 1fr 1fr; gap: 10px; align-items: end; margin: 8px 0; }
.edit-grid2 .field.term { grid-column: 1 / span 1; }
.edit-grid2 .field.controls { grid-column: 2 / span 1; }
.controls-row { display:grid; grid-template-columns: 1fr auto 1fr; align-items:center; column-gap: 8px; }
.controls-row .left { justify-self: start; }
.controls-row .mid { justify-self: center; }
.controls-row .right { justify-self: end; display:flex; align-items:center; gap: 8px; }
.controls-row .left, .controls-row .mid, .controls-row .right { display:flex; align-items:center; }
.controls-row .mid .seg-chip { margin: 0; display:inline-flex; align-items:center; }
.controls-row .right .btn-chip-icon { display:inline-flex; align-items:center; justify-content:center; }
.seg-toggle { display: inline-flex; white-space: nowrap; }
.seg-toggle .seg-item { height: 28px; padding: 0 10px; white-space: nowrap; }
.controls-row .mid { min-width: max-content; }
.check.small { white-space: nowrap; }
.edit-second2 { display:grid; grid-template-columns: 1fr 1fr; gap: 10px; align-items: end; margin: 8px 0 8px; }

.gl-table { border: 1px solid var(--macos-separator); border-radius: 8px; margin-top: 12px; margin-bottom: 16px; display:block; overflow: hidden; }
.gl-head { display:grid; grid-template-columns: minmax(0,1fr) 140px 300px 60px; align-items:center; gap: 0; background: var(--macos-background-secondary); padding: 6px 8px; font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.gl-head .cell { text-align: left; }
.gl-head .cell.act { text-align: center; padding-right: 0; }
.gl-head .cell { padding: 0 6px; }
.gl-body { display:block; }
.row { display:grid; grid-template-columns: minmax(0,1fr) 140px 300px 60px; align-items:center; padding: 6px 8px; border-top: 1px solid var(--macos-separator); cursor: pointer; }
.row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
.terms-scroll { display:block; overflow: visible; }
.cell.case { text-align: center; }
.cell.term { overflow: hidden; }
.cell.term .one-line { display: inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }
.cell.trans { overflow: hidden; }
.cell.trans .one-line { display: inline-block; max-width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; vertical-align: middle; }
.gl-body .row { font-size: var(--fs-sub); }
.row .cell { padding: 0 6px; }
.row .mono { font-family: var(--font-mono); }
</style>
