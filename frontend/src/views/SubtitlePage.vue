<template>
  <div class="macos-page subtitle-page min-w-0" :class="{ narrow: isNarrow, 'inspector-open': inspector.visible }" @click="onPageClick">
    <!-- Editor page (only when project opened), otherwise hub -->
    <template v-if="currentProject">
      <SubtitleEditorShell
        ref="editorRef"
        :current-project="currentProject"
        :is-detail-view="isDetailView"
        :is-narrow="isNarrow"
        :project-style="projectStyle"
        :style-guide-pills="styleGuidePills"
        :current-language-label="currentLanguageLabel"
        :current-lang-task="currentLangTask"
        :lang-meta="langMeta"
        :original-language-code="originalLanguageCode"
        :progress-pct="progressPct"
        :processed="processed"
        :total="total"
        :failed-count="failedCount"
        :start-time-display="startTimeDisplay"
        :elapsed-text="elapsedText"
        :provider-name="providerName"
        :llm-model-name="llmModelName"
        :progress-message="progressMessage"
        :error-text="errorText"
        :display-prompt-tokens="displayPromptTokens"
        :display-completion-tokens="displayCompletionTokens"
        :display-total-tokens="displayTotalTokens"
        :display-request-count="displayRequestCount"
        :token-speed-text="tokenSpeedText"
        :current-language-segments="currentLanguageSegments"
        :current-language="currentLanguage"
        :available-languages="availableLanguages"
        :subtitle-counts="subtitleCounts"
        @open-llm-chat="openLLMChat"
        @copy-project-name="copyProjectName"
        @copy-project-file-name="copyProjectFileName"
        @add-language="addLanguage"
        @update:currentLanguage="setCurrentLanguage"
        @update:projectData="updateCurrentProject"
      />
    </template>
    <template v-else>
      <SubtitleHubView
        :projects="subtitleProjects"
        :refreshing="refreshing"
        :inspector-visible="inspector.visible"
        :fab-right="fabRight"
        @open-file="openFile"
        @refresh="onRefresh"
        @open-project="loadRecentFile"
        @delete-project="removeWithConfirm"
      />
    </template>

    <!-- LLM 对话详情（ChatGPT 风格，只读，无输入框） -->
    <SubtitleLLMChatModal
      v-if="currentProject"
      v-model:show="showLLMChat"
      :project-id="currentProject?.id || ''"
      :language="currentLanguage"
      :language-label="currentLanguageLabel"
      :provider="providerName"
      :model="llmModelName"
      :prompt-tokens="displayPromptTokens"
      :completion-tokens="displayCompletionTokens"
      :total-tokens="displayTotalTokens"
      :start-time="Number(currentLangTask?.start_time || 0)"
      :end-time="Number(currentLangTask?.end_time || 0)"
      :is-running="isTranslating"
      :request-count="displayRequestCount"
    />

    <!-- Import Modal -->
    <SubtitleImportModal
      :show="showImportModal"
      :file-path="selectedFilePath"
      @close="showImportModal = false"
      @reselect="handleReselectFile"
      @import="handleImportWithOptions"
    />

    <SubtitleAddLanguageModal
      :show="showAddLanguageModal"
      :available-languages="Object.keys(availableLanguages)"
      :subtitle-service="subtitleService"
      :prefill="addLangPrefill"
      @close="showAddLanguageModal = false"
      @convert-started="handleConvertStarted"
    />
    <SubtitleMetricsModal :show="showMetrics" :standard-name="metricsStandardName" :standard-desc="metricsStandardDesc" @close="showMetrics = false" />

    <!-- Bottom cues info pill: only on editor (when a project is open) -->
    <div
      v-if="currentProject && showBottomCues"
      class="cues-floating-pill"
      :style="{ left: leftInset + 'px', right: rightInset + 'px' }"
      aria-hidden="true"
    >
      <span class="label">{{ t('subtitle.list.total_cues', { count: currentLanguageSegments.length }) }}</span>
    </div>

    <!-- floating controls / toolbar -->
    <SubtitleEditorToolbar
      v-if="currentProject"
      :is-narrow="isNarrow"
      :metrics-standard-name="metricsStandardName"
      :language-options="languageOptions"
      :current-language="currentLanguage"
      :can-delete-current-language="canDeleteCurrentLanguage"
      :is-translating="isTranslating"
      :refreshing="refreshing"
      :is-detail-view="isDetailView"
      :editor-disabled="editorDisabled"
      :toolbar-style="toolbarStyle"
      @show-metrics="showMetrics = true"
      @update:currentLanguage="setCurrentLanguage"
      @add-language="addLanguage"
      @refresh="onRefresh"
      @open-retry="openRetryModal"
      @delete-language="confirmDeleteLanguage"
      @set-view="activeView = $event"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import eventBus from '@/utils/eventBus.js'
import useInspectorStore from '@/stores/inspector.js'
import useLayoutStore from '@/stores/layout.js'
import { useSubtitleTasksStore } from '@/stores/subtitleTasks'
import { useTargetLanguagesStore } from '@/stores/targetLanguages.js'
import { SelectFile } from 'bindings/dreamcreator/backend/services/systems/service'
import { OpenFileWithOptions, GetSubtitle, DeleteSubtitle, DeleteAllSubtitle, UpdateProjectName } from 'bindings/dreamcreator/backend/api/subtitlesapi'
import { useSubtitleStore } from '@/stores/subtitle'
import usePreferencesStore from '@/stores/preferences.js'
import { subtitleService } from '@/services/subtitleService.js'
import { listEnabledProviders, refreshModels } from '@/services/llmProviderService.js'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

import SubtitleEditorShell from '@/components/subtitle/SubtitleEditorShell.vue'
import SubtitleEditorToolbar from '@/components/subtitle/SubtitleEditorToolbar.vue'
import SubtitleImportModal from '@/components/modal/SubtitleImportModal.vue'
import SubtitleAddLanguageModal from '@/components/modal/SubtitleAddLanguageModal.vue'
import SubtitleMetricsModal from '@/components/modal/SubtitleMetricsModal.vue'
import SubtitleLLMChatModal from '@/components/modal/SubtitleLLMChatModal.vue'
import SubtitleHubView from '@/views/subtitle/SubtitleHubView.vue'

const { t } = useI18n()
const inspector = useInspectorStore()
const layout = useLayoutStore()
const subtitleStore = useSubtitleStore()
const tasksStore = useSubtitleTasksStore()
const prefStore = usePreferencesStore()
const targetLangStore = useTargetLanguagesStore()

// store-backed computed
const currentProject = computed(() => subtitleStore.currentProject)
const subtitleProjects = computed(() => subtitleStore.projects)
const isLoading = computed(() => subtitleStore.isLoading)
const refreshing = ref(false)
const editorRef = ref(null)
let ioBottom = null

// narrow 定义：当检查器（Inspector）打开时为窄态
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1200)
const isNarrow = computed(() => !!inspector.visible)

// local state
const currentLanguage = ref('English')
const selectedFilePath = ref('')
const showImportModal = ref(false)
const showAddLanguageModal = ref(false)
const showMetrics = ref(false)
const showLLMChat = ref(false)
const showBottomCues = ref(false)
// tasks panel removed (integrated into Inspector)
const rightInset = computed(() => (inspector.visible ? (layout.inspectorWidth + 12) : 12))
const leftInset = computed(() => (layout.ribbonVisible ? (layout.ribbonWidth + 12) : 12))
// For editor page, align floating panel to inner content (card body has 12px padding)
// Keep floating controls aligned to the inner content edge on both home and edit pages
const fabRight = computed(() => rightInset.value + 6)
// Handlers for child panel v-model style updates
function updateProvider(v) { try { providerID.value = v || '' } catch {} }
function updateModel(v) { try { model.value = v || '' } catch {} }
function updateRetryFailedOnly(v) { try { retryFailedOnly.value = !!v } catch {} }
function onRetryFromPanel() { onRetryClick() }
// Prefill for Add Language modal
const addLangPrefill = ref({})
function openRetryModal() {
  try {
    const meta = langMeta.value || {}
    const status = meta?.status || {}
    const tasks = Array.isArray(status?.conversion_tasks) ? status.conversion_tasks : []
    const last = tasks.length ? tasks[tasks.length - 1] : null
    // Detect mode
    const hasLLMMarks = !!(providerName.value || llmModelName.value || (currentLangTask.value && (currentLangTask.value.provider || currentLangTask.value.model)))
    const isZhconvert = String(last?.type || meta?.translator || '').toLowerCase().includes('zhconvert') || !hasLLMMarks
    if (isZhconvert) {
      addLangPrefill.value = {
        tab: 'zhconvert',
        sourceLang: last?.source_lang || originalLanguageCode.value || '',
        converter: last?.converter_name || '',
      }
    } else {
      addLangPrefill.value = {
        tab: 'llm',
        sourceLang: last?.source_lang || originalLanguageCode.value || '',
        targetLang: currentLanguage.value || '',
        providerId: currentLangTask.value?.provider_id || '',
        providerName: currentLangTask.value?.provider || providerName.value || '',
        model: currentLangTask.value?.model || llmModelName.value || '',
        retryFailedOnly: true,
      }
    }
    showAddLanguageModal.value = true
  } catch { showAddLanguageModal.value = true }
}

function openLLMChat() {
  try {
    if (!currentProject.value?.id || !currentLanguage.value) return
    showLLMChat.value = true
  } catch {
    showLLMChat.value = true
  }
}

async function copyProjectName() {
  try {
    const name = currentProject.value?.project_name || ''
    if (name) await copyToClipboard(name, t)
  } catch {}
}
async function copyProjectFileName() {
  try {
    const name = currentProject.value?.metadata?.source_info?.file_name || ''
    if (name) await copyToClipboard(name, t)
  } catch {}
}

// derive
const availableLanguages = computed(() => currentProject.value?.language_metadata || {})
const languageCodes = computed(() => Object.keys(availableLanguages.value || {}))
const projectAnalysis = computed(() => {
  try { return currentProject.value?.metadata?.analysis || null } catch { return null }
})
const projectStyle = computed(() => {
  const a = projectAnalysis.value || {}
  const genre = (a.genre || '').trim?.() || ''
  const tone = (a.tone || '').trim?.() || ''
  // Only expose when there is at least one non-empty field
  if (!genre && !tone) return null
  return { genre, tone }
})
const styleGuidePills = computed(() => {
  try {
    const arr = projectAnalysis.value?.style_guide || []
    if (!Array.isArray(arr) || !arr.length) return []
    // Limit to first 8 rules to keep UI compact
    return arr.filter(Boolean).map(x => String(x)).slice(0, 8)
  } catch { return [] }
})
const langMeta = computed(() => {
  try { return (availableLanguages.value || {})[currentLanguage.value] || null } catch { return null }
})
const isOriginalLang = computed(() => !!langMeta.value?.status?.is_original)
const isLangDone = computed(() => String(langMeta.value?.sync_status || '').toLowerCase() === 'done')
const showTranslationPanel = computed(() => !isOriginalLang.value && !isLangDone.value)
// toolbar view switch
const activeView = ref('editor') // 'editor' | 'detail'
const editorDisabled = computed(() => showTranslationPanel.value)
const isDetailView = computed(() => showTranslationPanel.value || activeView.value === 'detail')
const firstLanguageCode = computed(() => Object.keys(availableLanguages.value || {})[0] || '')
const originalLanguageCode = computed(() => {
  try {
    const meta = currentProject.value?.language_metadata || {}
    const codes = Object.keys(meta)
    for (const code of codes) {
      if (meta?.[code]?.status?.is_original) return code
    }
  } catch {}
  return firstLanguageCode.value
})

function findCurrentLangTask() {
  try {
    const pid = currentProject.value?.id
    const lang = currentLanguage.value
    if (!pid || !lang) return null
    const activeId = currentProject.value?.language_metadata?.[lang]?.active_task_id
    if (activeId && tasksStore.tasksMap && tasksStore.tasksMap[activeId]) return tasksStore.tasksMap[activeId]
    // fallback: pick latest task matching project/lang
    const list = tasksStore.tasks || []
    for (const t of list) { if (t?.project_id === pid && t?.target_lang === lang) return t }
  } catch {}
  return null
}
const currentLangTask = computed(findCurrentLangTask)

// Fallback provider/model sourced from language process when task does not include them
const providerName = computed(() => {
  try {
    if (currentLangTask.value?.provider) return currentLangTask.value.provider
    const segs = currentProject.value?.segments || []
    const code = currentLanguage.value
    for (const s of segs) {
      const lc = s?.languages?.[code]
      const prov = lc?.process?.provider
      if (prov) return prov
    }
  } catch {}
  return ''
})
const llmModelName = computed(() => {
  try {
    if (currentLangTask.value?.model) return currentLangTask.value.model
    const segs = currentProject.value?.segments || []
    const code = currentLanguage.value
    for (const s of segs) {
      const lc = s?.languages?.[code]
      const m = lc?.process?.model
      if (m) return m
    }
  } catch {}
  return ''
})

function openInspectorTasks() {
  try {
    inspector.open('SubtitleTasksPanel', t('subtitle.tasks_title') || 'Subtitle Translation Tasks', {
      projectId: currentProject.value?.id || '',
      highlightTaskId: currentLangTask.value?.id || '',
    })
  } catch {}
}

function fmtTime(ts) {
  try { const d = new Date(Number(ts) * 1000); if (isNaN(d.getTime())) return ''; const p=n=>String(n).padStart(2,'0'); return `${d.getFullYear()}-${p(d.getMonth()+1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}` } catch { return '' }
}

function fmtTimeShort(ts) {
  try { const d = new Date(Number(ts) * 1000); if (isNaN(d.getTime())) return ''; const p=n=>String(n).padStart(2,'0'); return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}` } catch { return '' }
}

function statusText(s) {
  const key = String(s)
  const map = {
    processing: t('download.processing') || 'Processing',
    translating: t('download.processing') || 'Processing',
    completed: t('subtitle.task_ended') || 'Task ended',
    failed: t('download.failed') || 'Failed',
    cancelled: t('download.cancelled') || 'Cancelled',
    pending: t('download.pending') || 'Pending',
  }
  return map[key] || key || '-'
}

const statusBadgeClass = computed(() => {
  const s = String(currentLangTask.value?.status || langMeta.value?.sync_status || '').toLowerCase()
  return {
    'badge-running': s === 'processing' || s === 'translating' || s === 'pending',
    'badge-ok': s === 'completed' || s === 'done',
    'badge-error': s === 'failed',
    'badge-pending': s === 'cancelled',
  }
})

const statusRaw = computed(() => String(currentLangTask.value?.status || langMeta.value?.sync_status || '').toLowerCase())
const isTranslating = computed(() => ['processing','translating','pending'].includes(statusRaw.value))
const syncStatusRaw = computed(() => String(langMeta.value?.sync_status || '').toLowerCase())
const isErrorish = computed(() => statusRaw.value === 'failed' || syncStatusRaw.value === 'failed' || syncStatusRaw.value === 'partial_failed')

// Map status to shared chip badge classes (Tahoe-like coloring)
const statusChipClass = computed(() => {
  const s = statusRaw.value
  if (s === 'failed') return 'badge-error'
  if (s === 'completed' || s === 'done') return 'badge-success'
  if (s === 'cancelled') return 'badge-info'
  if (s === 'processing' || s === 'translating' || s === 'pending') return 'badge-primary'
  return 'badge-ghost'
})

const startTimeDisplay = computed(() => {
  try {
    const ts = currentLangTask.value?.start_time
    if (!ts) return '-'
    // 在窄模式下也展示完整日期
    return fmtTime(ts)
  } catch { return '-' }
})

const progressPct = computed(() => Number(currentLangTask.value?.progress || 0))
const processed = computed(() => Number(currentLangTask.value?.processed_segments || 0))
const total = computed(() => Number(currentLangTask.value?.total_segments || 0))
const failedCount = computed(() => Number(currentLangTask.value?.failed_segments || 0))
const errorText = computed(() => String(currentLangTask.value?.error_message || ''))

// Fallback to project metadata for tokens/requests so values persist across reloads even before WS
const lastTaskFromMeta = computed(() => {
  try {
    const meta = langMeta.value || {}
    const tasks = Array.isArray(meta?.status?.conversion_tasks) ? meta.status.conversion_tasks : []
    if (!tasks.length) return null
    const activeId = meta?.active_task_id
    if (activeId) {
      const found = tasks.find(t => String(t?.id || '') === String(activeId))
      if (found) return found
    }
    return tasks[tasks.length - 1]
  } catch { return null }
})
const displayPromptTokens = computed(() => {
  const a = Number(currentLangTask.value?.prompt_tokens || 0)
  return a || Number(lastTaskFromMeta.value?.prompt_tokens || 0)
})
const displayCompletionTokens = computed(() => {
  const a = Number(currentLangTask.value?.completion_tokens || 0)
  return a || Number(lastTaskFromMeta.value?.completion_tokens || 0)
})
const displayTotalTokens = computed(() => {
  const a = Number(currentLangTask.value?.total_tokens || 0)
  return a || Number(lastTaskFromMeta.value?.total_tokens || 0)
})
const displayRequestCount = computed(() => {
  const a = Number(currentLangTask.value?.request_count || 0)
  return a || Number(lastTaskFromMeta.value?.request_count || 0)
})
// 驱动 UI 每秒重算的时钟（翻译进行时）
const nowSec = ref(Math.floor(Date.now() / 1000))
let tickTimer = null
watch(() => isTranslating.value, (running) => {
  try { if (tickTimer) { clearInterval(tickTimer); tickTimer = null } } catch {}
  if (running) {
    tickTimer = setInterval(() => { nowSec.value = Math.floor(Date.now() / 1000) }, 1000)
  }
})
onUnmounted(() => { try { if (tickTimer) clearInterval(tickTimer) } catch {} })
const elapsedText = computed(() => {
  try {
    const st = Number(currentLangTask.value?.start_time || 0)
    const et = Number(currentLangTask.value?.end_time || 0)
    const now = nowSec.value
    const end = et || now
    if (!st) return ''
    const ms = (end - st) * 1000
    const h = Math.floor(ms / 3600000)
    const m = Math.floor((ms % 3600000) / 60000)
    const s = Math.floor((ms % 60000) / 1000)
    const parts = []
    if (h) parts.push(`${h}h`)
    if (m || h) parts.push(`${m}m`)
    parts.push(`${s}s`)
    return parts.join(' ')
  } catch { return '' }
})
// Token 速率：任务级平均 token/s（total_tokens / elapsed_seconds）
const tokenSpeedText = computed(() => {
  try {
    const totalTok = Number(displayTotalTokens.value || 0)
    if (!totalTok) return ''
    const st = Number(currentLangTask.value?.start_time || lastTaskFromMeta.value?.start_time || 0)
    if (!st) return ''
    const et = Number(currentLangTask.value?.end_time || lastTaskFromMeta.value?.end_time || 0)
    const now = nowSec.value
    const end = et || now
    const durSec = Math.max(1, end - st)
    const avgSpeed = totalTok / durSec
    return `${avgSpeed.toFixed(1)} tok/s`
  } catch {
    return ''
  }
})
// 语言展示名：优先使用“目标语言”全局配置中的 name，其次回退到项目 metadata，再次回退到 code 本身
const targetLangNameMap = computed(() => {
  const map = { ...(targetLangStore.nameMap || {}) }
  try {
    const meta = currentProject.value?.language_metadata || {}
    for (const code of Object.keys(meta)) {
      if (!code || map[code]) continue
      const m = meta[code] || {}
      const name = m.language_name || m.name || code
      map[code] = name || code
    }
  } catch {}
  return map
})
function getLanguageDisplayName(code) {
  const c = code || ''
  return targetLangNameMap.value[c] || c
}
const languageOptions = computed(() => (languageCodes.value || []).map(code => ({ code, name: getLanguageDisplayName(code) })))
const currentLanguageLabel = computed(() => getLanguageDisplayName(currentLanguage.value))

// Translation progress message (simple English, avoids i18n for dynamic batch text)
const progressMessage = computed(() => {
  if (!isTranslating.value) return ''
  const stage = String(currentLangTask.value?.stage || '')
  const detail = String(currentLangTask.value?.stage_detail || '')
  switch (stage) {
    case 'analysis_started':
      return 'Analyzing project…'
    case 'analysis_done':
      return 'Analysis complete'
    case 'batch_sending':
      return `Sending ${detail}` // e.g., "batch 1/12, size 20 items"
    case 'batch_received':
      return `Received ${detail}`
    case 'batch_parsed':
      return `Parsed ${detail}`
    case 'batch_applied':
      return `Applied ${detail}`
    default:
      return 'Processing…'
  }
})
const currentLanguageSegments = computed(() => {
  const segs = currentProject.value?.segments || []
  return segs.filter(s => s.languages && s.languages[currentLanguage.value])
})
const subtitleCounts = computed(() => {
  const counts = {}
  const meta = currentProject.value?.language_metadata || {}
  Object.keys(meta).forEach(code => { counts[code] = getLanguageSegmentCount(code) })
  return counts
})

// 当前语言是否允许删除（不可删除原始语言：元数据中 translator 为空视为原始）
const canDeleteCurrentLanguage = computed(() => {
  try {
    const code = currentLanguage.value
    if (!code) return false
    const meta = availableLanguages.value?.[code]
    const isOriginal = !!meta?.status?.is_original
    return !!currentProject.value && !!code && !isOriginal
  } catch { return false }
})

// Retry button visibility now follows delete logic + task status handled in template

async function confirmDeleteLanguage() {
  const code = currentLanguage.value
  if (!code) return
  const confirmed = window?.$dialog?.confirm
    ? await new Promise((resolve) => {
        window.$dialog.confirm(t('subtitle.list.delete_lang_confirm', { code }) || `Delete ${code}?`, {
          title: t('common.confirm') || 'Confirm',
          positiveText: t('common.delete') || 'Delete',
          negativeText: t('common.cancel') || 'Cancel',
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        })
      })
    : window.confirm(t('subtitle.list.delete_lang_confirm', { code }) || `Delete ${code}?`)
  if (!confirmed) return
  try {
    await subtitleService.deleteLanguage(code)
    $message?.success?.(t('common.deleted') || 'Deleted')
    try { await tasksStore.loadAll() } catch {}
    // 选择一个可用语言作为当前语言
    const codes = Object.keys(availableLanguages.value || {})
    const next = codes[0] || ''
    if (next && next !== code) { currentLanguage.value = next; try { subtitleStore.currentLanguage = next } catch {} }
  } catch (e) {
    console.error('Delete language failed:', e)
    $message?.error?.(e?.message || t('common.delete_failed') || 'Delete failed')
  }
}

// 保证在页面切换后字幕服务依然可用（autoSaveManager 不会为空）
watch(currentProject, (project) => {
  if (!project) return
  try {
    if (!subtitleService.autoSaveManager) {
      subtitleService.initialize(project)
    } else {
      subtitleService.handleProjectUpdate(project)
    }
  } catch (err) {
    console.warn('Failed to sync project with subtitle service:', err)
  }
}, { immediate: true })
// Toolbar background to match Ribbon/system theme
const uiFrosted = computed(() => true)
const isDarkMode = computed(() => !!prefStore?.isDark)
const toolbarBg = computed(() => (isDarkMode.value ? 'rgba(0,0,0,0.28)' : 'rgba(255,255,255,0.28)'))
const toolbarStyle = computed(() => {
  const base = { left: leftInset.value + 'px', right: rightInset.value + 'px', '--toolbar-bg': toolbarBg.value }
  if (uiFrosted.value) {
    base.WebkitBackdropFilter = 'var(--macos-surface-blur)'
    base.backdropFilter = 'var(--macos-surface-blur)'
  }
  return base
})

// Ensure currentLanguage always has a valid value
watch(languageCodes, (codes) => {
  const cur = currentLanguage.value
  if (!cur || !codes.includes(cur)) {
    const d = codes[0] || 'English'
    currentLanguage.value = d
    try { subtitleStore.currentLanguage = d } catch {}
  }
})
// Sync store.currentLanguage whenever UI currentLanguage changes (covers v-model path)
watch(currentLanguage, (val) => {
  try { subtitleStore.currentLanguage = val } catch {}
})
// Sync default view: when translating, force detail view and disable editor
watch(showTranslationPanel, (v) => {
  if (v) activeView.value = 'detail'
}, { immediate: true })

// helpers
function getLanguageSegmentCount(code) {
  const segs = currentProject.value?.segments || []
  return segs.filter(s => s.languages && s.languages[code]).length
}

// Metrics standard for current language (for modal)
const metricsStandardKey = computed(() => {
  const subs = currentLanguageSegments.value || []
  const first = subs[0]
  if (!first) return null
  try { return first?.guideline_standard?.[currentLanguage.value] || null } catch { return null }
})
const metricsStandardName = computed(() => {
  const m = metricsStandardKey.value
  const map = { netflix: 'Netflix', bbc: 'BBC', ade: 'ADE' }
  return m ? (map[m] || String(m).toUpperCase()) : 'Netflix'
})
const metricsStandardDesc = computed(() => {
  const m = metricsStandardKey.value
  if (!m) return ''
  const d = {
    netflix: t('subtitle.list.netflix_standard_desc'),
    bbc: t('subtitle.list.bbc_standard_desc'),
    ade: t('subtitle.list.ade_standard_desc'),
  }
  return d[m] || ''
})

function normalizeLang(s) {
  try { return String(s || '').trim().toLowerCase().replace(/_/g, '-').replace(/\s+/g, '') } catch { return String(s || '') }
}
function findBestLangMatch(pref, langs, project) {
  if (!pref) return ''
  const normPref = normalizeLang(pref)
  // 1) exact
  if (langs.includes(pref)) return pref
  // 2) case/sep-insensitive
  const byNorm = langs.find(l => normalizeLang(l) === normPref)
  if (byNorm) return byNorm
  // 3) try metadata hints if present
  try {
    const meta = project?.language_metadata || {}
    for (const code of langs) {
      const m = meta[code] || {}
      const alt = normalizeLang(m?.code || m?.lang || m?.name)
      if (alt && alt === normPref) return code
    }
  } catch {}
  return ''
}

function setProjectData(projectData) {
  subtitleStore.setCurrentProject(projectData)
  const langs = Object.keys(projectData.language_metadata || {})
  // Prefer language requested by inspector/store if provided
  let target = ''
  try { target = findBestLangMatch(subtitleStore.currentLanguage, langs, projectData) } catch {}
  // Fallback: 优先原始语言，其次第一个可用语言
  if (!target) {
    try {
      const meta = projectData.language_metadata || {}
      const original = Object.keys(meta).find(code => !!meta?.[code]?.status?.is_original)
      if (original) target = original
    } catch {}
  }
  if (!target && langs.length) target = langs[0]
  // 默认回到 Editor 视图；若当前语言处于翻译中，watch(showTranslationPanel) 会将其切换回 Detail
  try { activeView.value = 'editor' } catch {}
  if (target) { currentLanguage.value = target; try { subtitleStore.currentLanguage = target } catch {} }
  subtitleService.initialize(projectData)
}

async function loadSubtitleProject(projectId) {
  try {
    subtitleStore.isLoading = true
    const result = await GetSubtitle(projectId)
    if (!result?.success) throw new Error(result?.msg)
    const data = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
    setProjectData(data)
  } catch (e) { $message?.error?.(e.message || String(e)) }
  finally { subtitleStore.isLoading = false }
}

async function handleImportWithOptions({ filePath, options }) {
  try {
    showImportModal.value = false
    subtitleStore.isLoading = true
    const r = await OpenFileWithOptions(filePath, options)
    if (!r?.success) throw new Error(r?.msg)
    const data = typeof r.data === 'string' ? JSON.parse(r.data) : r.data
    setProjectData(data)
    await subtitleStore.fetchProjects()
  } catch (e) { $message?.error?.(e.message || String(e)) }
  finally { subtitleStore.isLoading = false }
}

async function openFile() {
  subtitleStore.isLoading = true
  try {
    const r = await SelectFile(t('subtitle.common.select_sub_file'), ['srt','itt','vtt','ass','ssa'])
    if (r?.success && r.data?.path) {
      selectedFilePath.value = r.data.path
      showImportModal.value = true
    }
  } finally {
    // Always clear loading, even on cancel or error
    subtitleStore.isLoading = false
  }
}

async function handleReselectFile() {
  try {
    const r = await SelectFile(t('subtitle.common.select_sub_file'), ['srt','itt','vtt','ass','ssa'])
    if (r?.success && r.data?.path) {
      selectedFilePath.value = r.data.path
    }
  } catch {}
}

function setCurrentLanguage(code) { currentLanguage.value = code; try { subtitleStore.currentLanguage = code } catch {} }
async function loadRecentFile(item) { await loadSubtitleProject(item.id) }
async function removeFromHistory(id) {
  try {
    const r = await DeleteSubtitle(id)
    if (!r?.success) throw new Error(r?.msg)
    await subtitleStore.fetchProjects()
    // Refresh AI tasks in Inspector to reflect deletion
    try { await tasksStore.loadAll() } catch {}
  } catch (e) { $message?.error?.(e.message || String(e)) }
}
function removeWithConfirm(project) {
  const content = t('common.delete_confirm_detail', { title: project?.project_name || '' })
  $dialog?.confirm?.(content, {
    title: t('common.delete_confirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => removeFromHistory(project?.id),
  })
}
async function clearAllHistory() {
  try {
    const r = await DeleteAllSubtitle()
    if (!r?.success) throw new Error(r?.msg)
    await subtitleStore.fetchProjects()
    // Refresh AI tasks in Inspector after clearing projects
    try { await tasksStore.loadAll() } catch {}
  } catch (e) { $message?.error?.(e.message || String(e)) }
}

function updateCurrentProject(data) { subtitleStore.setCurrentProject(data) }
function addLanguage() {
  // open as a fresh add flow; clear any previous retry prefill
  addLangPrefill.value = {}
  showAddLanguageModal.value = true
}
function handleConvertStarted(info) {
  try {
    const lang = info?.targetLanguage || info?.target_language || ''
    if (lang) { currentLanguage.value = lang; try { subtitleStore.currentLanguage = lang } catch {} }
  } catch {}
}

async function onRefresh() { if (refreshing.value) return; try { refreshing.value = true; await subtitleStore.refreshProjects() } finally { setTimeout(() => { refreshing.value = false }, 200) } }

function onClearAll() { /* removed from UI per new design; leave helper if needed */ }
// removed: toggleTasksPanel/onClickTask, tasks floating panel migrated into Inspector

// page click closes inspector for consistency
function onPageClick() { inspector.close() }

// stable handlers for event bus (so we can off() on unmount)
function handleOpenFile() { openFile() }
function handleAddLanguage() { addLanguage() }
function handleMetrics() { showMetrics.value = true }
function handleBackHome() { backHome() }
async function handleOpenChat(payload = {}) {
  try {
    const pid = payload?.projectId || payload?.project_id || ''
    const lang = payload?.targetLang || payload?.target_lang || payload?.language || ''
    if (pid && (!currentProject.value || currentProject.value.id !== pid)) {
      await loadSubtitleProject(pid)
    }
    if (lang) { currentLanguage.value = lang; try { subtitleStore.currentLanguage = lang } catch {} }
    if (pid || currentProject.value) openLLMChat()
  } catch {
    openLLMChat()
  }
}
const onResize = () => { try { viewportWidth.value = window.innerWidth } catch {} }

// 页面恢复刷新：当页面重新可见或窗口获得焦点时，主动同步任务与项目
async function refreshSubtitleContext() {
  try { await tasksStore.loadAll() } catch {}
  try { await subtitleStore.fetchProjects({ showLoading: false, force: true }) } catch {}
  try {
    const pid = subtitleStore.currentProject?.id
    if (pid) { await loadSubtitleProject(pid) }
  } catch {}
}
function onVisibilityChange() { try { if (document.visibilityState === 'visible') refreshSubtitleContext() } catch {} }
function onWindowFocus() { refreshSubtitleContext() }

// lifecycle
let unsubscribeConversion = null
onMounted(async () => {
  try { await targetLangStore.ensureLoaded() } catch {}
  try { await subtitleStore.fetchProjects() } catch {}
  try { tasksStore.init?.() } catch {}
  // if a pending project id is queued (from cross-page open), open it and clear
  try {
    const pid = subtitleStore.pendingOpenProjectId
    if (pid) {
      await loadSubtitleProject(pid)
      subtitleStore.setPendingOpenProjectId(null)
    }
  } catch {}
  unsubscribeConversion = subtitleService.onConversionEvent((evt) => {
    // Refresh project list when conversions finish; floating tasks UI removed
    if (evt?.isTerminal) subtitleStore.fetchProjects({ showLoading: false }).catch(() => {})
  })
  // register event listeners
  eventBus.on('subtitle:open-file', handleOpenFile)
  eventBus.on('subtitle:open-project', loadSubtitleProject)
  eventBus.on('subtitle:open-chat', handleOpenChat)
  eventBus.on('subtitle:back-home', handleBackHome)
  eventBus.on('subtitle:add-language', handleAddLanguage)
  eventBus.on('subtitle:metrics', handleMetrics)
  // resize observer for narrow mode
  window.addEventListener('resize', onResize)
  // 页面恢复可见/焦点时刷新任务与项目，保证返回页面后能拿到最新进度
  try {
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('focus', onWindowFocus)
  } catch {}
  // bottom sentinel to toggle cues pill when reaching the end
  try {
    ioBottom = new IntersectionObserver((entries) => {
      showBottomCues.value = entries.some(e => e.isIntersecting)
    })
    setTimeout(() => {
      const shell = editorRef.value
      const end = shell && shell.endSentinel
      if (end) ioBottom.observe(end)
    }, 0)
    // re-observe when currentProject toggles
    watch(currentProject, () => {
      showBottomCues.value = false
      try { ioBottom.disconnect() } catch {}
      setTimeout(() => {
        const shell = editorRef.value
        const end = shell && shell.endSentinel
        if (currentProject.value && end) ioBottom.observe(end)
      }, 0)
    })
  } catch {}
})

// Back to hub: clear current project and close inspector
function backHome() {
  try {
    subtitleStore.setCurrentProject(null)
    inspector.close()
  } catch {}
}

// ----- Retry controls (provider/model selects) when error -----
const providers = ref([])
const providerID = ref('')
const models = ref([])
const model = ref('')
const retryFailedOnly = ref(true)
let prefilledOnce = false

async function loadProviders() {
  try {
    const list = await listEnabledProviders()
    providers.value = Array.isArray(list) ? list : []
    // Try prefill from current task once providers are available
    tryPrefillFromTask()
    // Ensure models list is in sync after potential prefill
    syncModels()
  } catch { providers.value = [] }
}

function syncModels() {
  try {
    const p = providers.value.find(x => x && x.id === providerID.value)
    const arr = Array.isArray(p?.models) ? p.models : (Array.isArray(p?.Models) ? p.Models : [])
    models.value = arr
    if (!models.value.includes(model.value)) {
      model.value = models.value[0] || ''
    }
  } catch { models.value = []; model.value = '' }
}

watch(providerID, () => { syncModels() })
watch(isErrorish, (v) => { if (v) loadProviders() })

function tryPrefillFromTask() {
  if (prefilledOnce) return
  try {
    // Map provider by name -> id
    const pid = (currentLangTask?.value?.provider_id || '').trim?.() || ''
    const pname = (providerName?.value || '').trim()
    if (pid && providers.value.some(p => p?.id === pid)) {
      providerID.value = pid
    } else if (pname && providers.value.length) {
      const found = providers.value.find(rec => String(rec?.name || '').toLowerCase() === pname.toLowerCase())
      if (found && found.id) providerID.value = found.id
    }
    // Sync models before picking model name
    syncModels()
    const mname = (llmModelName?.value || '').trim()
    if (mname && models.value.includes(mname)) {
      model.value = mname
    }
    // If still not set, keep current fallback selection
    prefilledOnce = true
  } catch { /* ignore */ }
}

// Attempt prefill when task info becomes available
watch([() => providerName.value, () => llmModelName.value, () => providers.value.length], () => {
  if (isErrorish.value) tryPrefillFromTask()
})

// Reset prefill flag when task changes (e.g., switch language or new task arrives)
watch(currentLangTask, () => { prefilledOnce = false; if (isErrorish.value) loadProviders() })

async function onRetryClick() {
  try {
    const src = currentLangTask.value?.source_lang || originalLanguageCode.value
    const tgt = currentLanguage.value
    const same = String(src || '').trim().toLowerCase() === String(tgt || '').trim().toLowerCase()
    if (same) { $message?.warning?.(t('subtitle.add_language.same_language_warning') || 'Source and target cannot be the same'); return }
    if (!providerID.value || !model.value) { $message?.warning?.(t('subtitle.add_language.select_model') || 'Select model'); return }
    if (retryFailedOnly.value) {
      await subtitleService.retryFailedTranslations(src, tgt, providerID.value, model.value, [], [])
    } else {
      await subtitleService.translateSubtitleLLMWithGlossary(src, tgt, providerID.value, model.value, [], [])
    }
    $message?.success?.(t('subtitle.add_language.conversion_started') || 'Started')
    // Proactively refresh tasks + projects so UI reflects the new translation immediately
    try { await tasksStore.loadAll() } catch {}
    try { await subtitleStore.fetchProjects({ showLoading: false, force: true }) } catch {}
  } catch (e) { $message?.error?.(e?.message || String(e)) }
}

/* sort toggle removed; default order by updated_at desc */

// global cleanup on unmount
onUnmounted(() => {
  try { unsubscribeConversion && unsubscribeConversion() } catch {}
  // 清理字幕服务，移除注册的回调，避免重复消息
  try { subtitleService.destroy() } catch {}
  eventBus.off('subtitle:open-file', handleOpenFile)
  eventBus.off('subtitle:open-project', loadSubtitleProject)
  eventBus.off('subtitle:open-chat', handleOpenChat)
  eventBus.off('subtitle:back-home', handleBackHome)
  eventBus.off('subtitle:add-language', handleAddLanguage)
  eventBus.off('subtitle:metrics', handleMetrics)
  // 解绑页面恢复事件
  try {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('focus', onWindowFocus)
  } catch {}
  try { ioBottom && ioBottom.disconnect() } catch {}
  window.removeEventListener('resize', onResize)
})

</script>
