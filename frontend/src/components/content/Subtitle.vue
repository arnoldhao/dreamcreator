<template>
  <div class="macos-page min-w-0" :class="{ narrow: isNarrow }" @click="onPageClick">
    <!-- Editor page (only when project opened) -->
    <template v-if="currentProject">
      
      <div class="macos-card card-frosted card-translucent sr-card w-full">
        <div class="sr-card-body p-3">
          <SubtitleList
            :subtitles="currentLanguageSegments"
            :current-language="currentLanguage"
            :available-languages="availableLanguages"
            :subtitle-counts="subtitleCounts"
            @add-language="addLanguage"
            @update:currentLanguage="setCurrentLanguage"
            @update:projectData="updateCurrentProject"
          />
        </div>
      </div>
    </template>

    <!-- Hub: History-first with import entry -->
    <template v-else>
      <!-- When empty: show a single empty-state card (no outer frame) -->
      <div v-if="!processedItems.length" class="hub-empty">
        <!-- Empty for all: no history yet -->
        <div v-if="isFormatAll" class="macos-card card-frosted card-translucent empty-card" @click.stop>
          <div class="icon-wrap">
            <div class="icon-bg">
              <Icon name="file-text" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
            </div>
          </div>
          <div class="title">{{ $t('subtitle.history.no_historical_records') }}</div>
          <div class="subtitle">{{ $t('subtitle.history.no_imported_sub_found') }}</div>
          <div class="actions">
            <button class="btn-glass btn-primary btn-sm" @click.stop="openFile">
              <Icon name="plus" class="w-4 h-4 mr-1" />
              {{ $t('subtitle.common.open_file') }}
            </button>
            <button class="btn-glass btn-sm" @click.stop="onRefresh">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>
        <!-- Empty for filtered: no results under current filter -->
        <div v-else class="macos-card card-frosted card-translucent empty-card" @click.stop>
          <div class="icon-wrap">
            <div class="icon-bg">
              <Icon name="filter-x" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
            </div>
          </div>
          <div class="title">{{ $t('download.no_filter_results') }}</div>
          <div class="subtitle">{{ currentFormatLabel }}</div>
          <div class="actions">
            <button class="btn-glass btn-primary btn-sm" @click.stop="resetFilters">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.reset') }}
            </button>
            <button class="btn-glass btn-sm" @click.stop="onRefresh">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>
      </div>

      <!-- list view with grouping and incremental loading (wrapped in card) -->
      <div v-else class="macos-card card-frosted card-translucent hub">
        <div class="hub-body">
          <div class="list-wrap comfortable">
            <template v-for="(it, idx) in visibleItems" :key="it.key || (it.project?.id + ':' + idx)">
              <div v-if="it.type === 'header'" class="list-header">{{ it.label }}</div>
              <div v-else class="list-row" @click.stop="loadRecentFile(it.project)">
                <div class="col-icon">
                  <div class="tile-icon" :class="extClass(it.project)">
                    <Icon :name="extIcon(it.project)" class="w-4 h-4" />
                  </div>
                </div>
                <div class="col-title" :title="it.project.project_name">
                  <template v-if="editingProjectId === it.project.id">
                    <input
                      class="rename-input"
                      v-model="editingProjectName"
                      @keydown.enter.stop.prevent="confirmRename(it.project)"
                      @keydown.esc.stop.prevent="cancelRename"
                      @click.stop
                    />
                    <button class="icon-glass" :data-tooltip="$t('common.confirm')" data-tip-pos="top" @click.stop="confirmRename(it.project)"><Icon name="status-success" class="w-3.5 h-3.5" /></button>
                    <button class="icon-glass" :data-tooltip="$t('common.cancel')" data-tip-pos="top" @click.stop="cancelRename"><Icon name="close" class="w-3.5 h-3.5" /></button>
                  </template>
                  <template v-else>
                    <span class="name one-line">{{ it.project.project_name || '-' }}</span>
                    <button class="icon-glass rename-btn" :data-tooltip="$t('common.edit')" data-tip-pos="top" @click.stop="beginRename(it.project)"><Icon name="edit" class="w-3.5 h-3.5" /></button>
                  </template>
                </div>
                <div class="col-pills">
                  <div class="meta-group small">
                    <div class="item"><Icon name="database" class="w-3.5 h-3.5" />{{ it.project.segments?.length || 0 }}</div>
                    <div class="divider-v"></div>
                    <div class="item"><Icon name="languages" class="w-3.5 h-3.5" />{{ langCount(it.project) }}</div>
                    <div class="divider-v"></div>
                    <div v-if="it.project.metadata?.source_info?.file_ext" class="item mono">
                      <span class="ext-tag" :class="extClass(it.project)">{{ (it.project.metadata.source_info.file_ext || '').toUpperCase() }}</span>
                    </div>
                  </div>
                </div>
                <div class="col-time">
                  <div class="t-rel">{{ formatRelative(it.project.updated_at) }}</div>
                  <div class="t-abs">{{ formatDate(it.project.updated_at) }}</div>
                </div>
                <div class="col-actions">
                  <button class="icon-glass" @click.stop="removeWithConfirm(it.project)"><Icon name="trash" class="w-4 h-4" /></button>
                </div>
              </div>
            </template>
            <div ref="listEnd" class="io-sentinel"></div>
          </div>
        </div>
      </div>
    </template>

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

    <!-- floating controls (bottom-right) -->
    <!-- Hub: refresh + format filter -->
    <div v-if="!currentProject" class="floating-filter" @click.stop :style="{ right: fabRight + 'px' }">
      <button class="icon-glass" :data-tooltip="$t('download.refresh')" data-tip-pos="top" @click="onRefresh">
        <Icon name="refresh" class="w-4 h-4" :class="{ spinning: refreshing }" />
      </button>
      <div class="divider-v"></div>
      <div class="filter-toggle" @click="floatingFilterExpanded = !floatingFilterExpanded">
        <Icon name="filter" class="w-4 h-4" />
        <span class="chip-frosted chip-sm chip-translucent count-pill"><span class="chip-label">{{ totalCount }}/{{ totalAll }}</span></span>
      </div>
      <select v-if="floatingFilterExpanded" v-model="formatFilter" class="input-macos select-macos select-macos-xs filter-select">
        <option value="all">All</option>
        <option v-for="opt in formatOptions" :key="opt" :value="opt">{{ opt.toUpperCase() }}</option>
      </select>
    </div>
    <!-- Editor: add language + language picker -->
    <div v-else class="floating-filter" @click.stop :style="{ right: fabRight + 'px' }">
      <button class="sr-icon-btn icon-ghost expand-left" :aria-label="$t('subtitle.add_language.title')" @click="showAddLanguageModal = true">
        <Icon name="plus" class="w-4 h-4" />
        <span class="label">{{ $t('subtitle.add_language.title') }}</span>
      </button>
      <div class="divider-v"></div>
      <div class="filter-toggle" @click="floatingFilterExpanded = !floatingFilterExpanded">
        <Icon name="languages" class="w-4 h-4" />
        <span class="lang-label">{{ currentLanguage || defaultLanguage }}</span>
        <span class="chip-frosted chip-sm chip-translucent count-pill"><span class="chip-label">{{ languageCount }}</span></span>
      </div>
      <select v-if="floatingFilterExpanded" v-model="currentLanguage" class="input-macos select-macos select-macos-xs filter-select">
        <option v-for="code in languageCodes" :key="code" :value="code">{{ code }}</option>
      </select>
    </div>
    <!-- End sentinel for bottom detection (editor only) -->
    <div v-if="currentProject" ref="endSentinel" style="height: 1px; width: 100%;"></div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import eventBus from '@/utils/eventBus.js'
import useInspectorStore from '@/stores/inspector.js'
import useLayoutStore from '@/stores/layout.js'
import { SelectFile } from 'wailsjs/go/systems/Service'
import { OpenFileWithOptions, GetSubtitle, DeleteSubtitle, DeleteAllSubtitle, UpdateProjectName } from 'wailsjs/go/api/SubtitlesAPI'
import { useSubtitleStore } from '@/stores/subtitle'
import { subtitleService } from '@/services/subtitleService.js'

import SubtitleList from '@/components/subtitle/SubtitleList.vue'
import SubtitleImportModal from '@/components/subtitle/SubtitleImportModal.vue'
import SubtitleAddLanguageModal from '@/components/subtitle/SubtitleAddLanguageModal.vue'
import SubtitleMetricsModal from '@/components/modal/SubtitleMetricsModal.vue'

const { t } = useI18n()
const inspector = useInspectorStore()
const layout = useLayoutStore()
const subtitleStore = useSubtitleStore()

// store-backed computed
const currentProject = computed(() => subtitleStore.currentProject)
const subtitleProjects = computed(() => subtitleStore.projects)
const isLoading = computed(() => subtitleStore.isLoading)
const refreshing = ref(false)
const query = ref('')
const formatFilter = ref('all')
const floatingFilterExpanded = ref(false)
const displayCount = ref(80)
const listEnd = ref(null)
let io = null
let ioBottom = null

// responsive width to enable compact/narrow rendering
const viewportWidth = ref(typeof window !== 'undefined' ? window.innerWidth : 1200)
const isNarrow = computed(() => {
  // When inspector is closed, always treat as non-narrow for better UX
  if (!inspector.visible) return false
  // When inspector is open, fall back to width-based calculation
  const left = layout.ribbonVisible ? layout.ribbonWidth : 0
  const right = layout.inspectorWidth
  const avail = viewportWidth.value - left - right - 24 /* paddings */
  return avail <= 760
})

// local state
const currentLanguage = ref('English')
const selectedFilePath = ref('')
const showImportModal = ref(false)
const showAddLanguageModal = ref(false)
const showMetrics = ref(false)
const endSentinel = ref(null)
const showBottomCues = ref(false)
const rightInset = computed(() => (inspector.visible ? (layout.inspectorWidth + 12) : 12))
const leftInset = computed(() => (layout.ribbonVisible ? (layout.ribbonWidth + 12) : 12))
// For editor page, align floating panel to inner content (card body has 12px padding)
// Keep floating controls aligned to the inner content edge on both home and edit pages
const fabRight = computed(() => rightInset.value + 6)
// inline rename state for history list
const editingProjectId = ref(null)
const editingProjectName = ref('')

// derive
const availableLanguages = computed(() => currentProject.value?.language_metadata || {})
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

const filteredProjects = computed(() => {
  const q = (query.value || '').toLowerCase()
  const f = (formatFilter.value || 'all').toLowerCase()
  return (subtitleProjects.value || []).filter(p => {
    const nameHit = !q || (p.project_name || '').toLowerCase().includes(q)
    const ext = (p.metadata?.source_info?.file_ext || '').toLowerCase()
    const formatHit = f === 'all' || ext === f
    return nameHit && formatHit
  })
})

const sortedProjects = computed(() => {
  const arr = [...filteredProjects.value]
  arr.sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
  return arr
})

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

function groupLabel(ts) {
  if (!ts) return t('subtitle.group.earlier') || 'Earlier'
  const d = new Date(ts * 1000)
  const now = new Date()
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const dayMs = 24 * 60 * 60 * 1000
  if (d.getTime() >= startOfToday) return t('subtitle.group.today') || 'Today'
  if ((startOfToday - d.getTime()) < 6 * dayMs) return t('subtitle.group.this_week') || 'This Week'
  return t('subtitle.group.earlier') || 'Earlier'
}

const processedItems = computed(() => {
  const items = []
  let lastGroup = null
  for (const p of sortedProjects.value) {
    const g = groupLabel(p.updated_at)
    if (g !== lastGroup) {
      items.push({ type: 'header', key: 'g:' + g + ':' + (items.length), label: g })
      lastGroup = g
    }
    items.push({ type: 'item', project: p })
  }
  return items
})

const visibleItems = computed(() => processedItems.value.slice(0, displayCount.value))
const totalCount = computed(() => sortedProjects.value.length)
const totalAll = computed(() => (subtitleProjects.value || []).length)
const languageCodes = computed(() => Object.keys(availableLanguages.value || {}))
const languageCount = computed(() => languageCodes.value.length)
const defaultLanguage = computed(() => languageCodes.value[0] || 'English')

// download-like empty logic helpers for hub
const isFormatAll = computed(() => (formatFilter.value || 'all') === 'all')
const currentFormatLabel = computed(() => {
  if (isFormatAll.value) return ''
  const f = (formatFilter.value || '').toUpperCase()
  return f
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
const formatOptions = computed(() => {
  const s = new Set()
  for (const p of subtitleProjects.value || []) {
    const e = (p.metadata?.source_info?.file_ext || '').toLowerCase()
    if (e) s.add(e)
  }
  return Array.from(s).sort()
})

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

function setProjectData(projectData) {
  subtitleStore.setCurrentProject(projectData)
  const langs = Object.keys(projectData.language_metadata || {})
  if (langs.length) { currentLanguage.value = langs[0]; try { subtitleStore.currentLanguage = langs[0] } catch {} }
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
  } catch (e) { $message?.error?.(e.message || String(e)) }
}

function updateCurrentProject(data) { subtitleStore.setCurrentProject(data) }
function addLanguage() { showAddLanguageModal.value = true }
function handleConvertStarted(_info) {}

function langCount(p) {
  const meta = p?.language_metadata
  if (meta && typeof meta === 'object') return Object.keys(meta).length
  return 1
}

function extOf(p) {
  return (p?.metadata?.source_info?.file_ext || '').toLowerCase()
}
function extClass(p) {
  const e = extOf(p)
  return e ? ('ext-' + e) : ''
}
function extIcon(p) {
  const e = extOf(p)
  if (e === 'srt') return 'captions'
  if (e === 'vtt') return 'file-text'
  if (e === 'ass') return 'file-code'
  if (e === 'itt') return 'languages'
  return 'file-text'
}

async function onRefresh() { if (refreshing.value) return; try { refreshing.value = true; await subtitleStore.refreshProjects() } finally { setTimeout(() => { refreshing.value = false }, 200) } }

function resetFilters() {
  formatFilter.value = 'all'
  query.value = ''
}

function onClearAll() { /* removed from UI per new design; leave helper if needed */ }

// page click closes inspector for consistency
function onPageClick() { inspector.close() }

// stable handlers for event bus (so we can off() on unmount)
function handleOpenFile() { openFile() }
function handleAddLanguage() { showAddLanguageModal.value = true }
function handleMetrics() { showMetrics.value = true }
function handleSearch(q) { query.value = q || '' }
function handleBackHome() { backHome() }
const onResize = () => { try { viewportWidth.value = window.innerWidth } catch {} }

// lifecycle
let unsubscribeConversion = null
onMounted(async () => {
  try { await subtitleStore.fetchProjects() } catch {}
  // if a pending project id is queued (from cross-page open), open it and clear
  try {
    const pid = subtitleStore.pendingOpenProjectId
    if (pid) {
      await loadSubtitleProject(pid)
      subtitleStore.setPendingOpenProjectId(null)
    }
  } catch {}
  unsubscribeConversion = subtitleService.onConversionEvent((evt) => {
    if (evt?.isTerminal) subtitleStore.fetchProjects({ showLoading: false }).catch(() => {})
  })
  // register event listeners
  eventBus.on('subtitle:open-file', handleOpenFile)
  eventBus.on('subtitle:open-project', loadSubtitleProject)
  eventBus.on('subtitle:back-home', handleBackHome)
  eventBus.on('subtitle:add-language', handleAddLanguage)
  eventBus.on('subtitle:metrics', handleMetrics)
  eventBus.on('subtitle:search', handleSearch)
  // incremental load observer
  try {
    io = new IntersectionObserver((entries) => {
      if (entries.some(e => e.isIntersecting)) {
        displayCount.value = Math.min(displayCount.value + 80, processedItems.value.length + 80)
      }
    })
    setTimeout(() => { if (listEnd.value) io.observe(listEnd.value) }, 0)
  } catch {}
  // resize observer for narrow mode
  window.addEventListener('resize', onResize)
  // bottom sentinel to toggle cues pill when reaching the end
  try {
    ioBottom = new IntersectionObserver((entries) => {
      showBottomCues.value = entries.some(e => e.isIntersecting)
    })
    setTimeout(() => { if (endSentinel.value) ioBottom.observe(endSentinel.value) }, 0)
    // re-observe when currentProject toggles
    watch(currentProject, () => {
      showBottomCues.value = false
      try { ioBottom.disconnect() } catch {}
      setTimeout(() => { if (currentProject.value && endSentinel.value) ioBottom.observe(endSentinel.value) }, 0)
    })
  } catch {}
})


function formatDate(timestamp) {
  if (!timestamp) return 'N/A'
  const date = new Date((typeof timestamp === 'number' ? timestamp : Number(timestamp)) * 1000)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  if (diff < 24 * 60 * 60 * 1000) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  } else if (diff < 7 * 24 * 60 * 60 * 1000) {
    return date.toLocaleString([], { weekday: 'short', hour: '2-digit', minute: '2-digit' })
  }
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}

function formatRelative(timestamp) {
  if (!timestamp) return 'N/A'
  const d = new Date((typeof timestamp === 'number' ? timestamp : Number(timestamp)) * 1000)
  const now = new Date()
  const diffMs = now.getTime() - d.getTime()
  const mins = Math.floor(diffMs / (1000 * 60))
  const hours = Math.floor(diffMs / (1000 * 60 * 60))
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  if (mins < 1) return t('cookies.just_now') || 'Just now'
  if (mins < 60) return t('cookies.minutes_ago', { count: mins }) || `${mins}m`
  if (hours < 24) return t('cookies.hours_ago', { count: hours }) || `${hours}h`
  if (days < 7) return t('cookies.days_ago', { count: days }) || `${days}d`
  return d.toLocaleDateString()
}

// Back to hub: clear current project and close inspector
function backHome() {
  try {
    subtitleStore.setCurrentProject(null)
    inspector.close()
  } catch {}
}

/* sort toggle removed; default order by updated_at desc */

// global cleanup on unmount
onUnmounted(() => {
  try { unsubscribeConversion && unsubscribeConversion() } catch {}
  // 清理字幕服务，移除注册的回调，避免重复消息
  try { subtitleService.destroy() } catch {}
  eventBus.off('subtitle:open-file', handleOpenFile)
  eventBus.off('subtitle:open-project', loadSubtitleProject)
  eventBus.off('subtitle:back-home', handleBackHome)
  eventBus.off('subtitle:add-language', handleAddLanguage)
  eventBus.off('subtitle:metrics', handleMetrics)
  eventBus.off('subtitle:search', handleSearch)
  try { io && io.disconnect() } catch {}
  try { ioBottom && ioBottom.disconnect() } catch {}
  window.removeEventListener('resize', onResize)
})

// ----- inline rename handlers (history list) -----
function beginRename(project) {
  editingProjectId.value = project?.id
  editingProjectName.value = project?.project_name || ''
}
function cancelRename() {
  editingProjectId.value = null
  editingProjectName.value = ''
}
async function confirmRename(project) {
  const name = (editingProjectName.value || '').trim()
  if (!name) { $message?.warning?.(t('subtitle.common.project_name') + ' ' + (t('common.not_set') || '')); return }
  try {
    const r = await UpdateProjectName(project.id, name)
    if (!r?.success) throw new Error(r?.msg)
    try { project.project_name = name } catch {}
    cancelRename()
    await subtitleStore.fetchProjects()
  } catch (e) { $message?.error?.(e.message || String(e)) }
}

</script>

<style scoped>
/* Editor outer container: match hub card visuals */
.sr-card { background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 8px; box-shadow: var(--macos-shadow-1); }
.sr-card-body { padding: 12px; }

/* Hub styles */
.hub { padding: 0; overflow: visible; }
.hub-head { display:flex; align-items:center; justify-content: space-between; padding: 10px 12px; }
.hub-head .left { display:flex; align-items: baseline; gap: 10px; }
.hub-head .title { font-size: var(--fs-title); font-weight: 600; color: var(--macos-text-primary); }
.hub-head .actions { display:flex; align-items:center; gap: 8px; }
.hub-divider { height: 1px; background: var(--macos-divider-weak); }
.hub-body { padding: 12px; }
.hub-empty { padding: 36px 12px; display:flex; flex-direction:column; align-items:center; justify-content:center; gap: 6px; }

/* empty card (align with download page) */
.empty-card { width: 100%; max-width: 560px; padding: 24px; border-radius: 12px; display:flex; flex-direction:column; align-items:center; text-align:center; gap: 12px; }
.empty-card .icon-wrap { margin-bottom: 4px; }
.empty-card .icon-bg { width: 56px; height: 56px; border-radius: 50%; display:flex; align-items:center; justify-content:center; background: linear-gradient(180deg, rgba(0,0,0,0.03), rgba(0,0,0,0.06)); border: 1px solid var(--macos-separator); }
.empty-card .title { font-size: var(--fs-title); font-weight: 600; color: var(--macos-text-primary); }
.empty-card .subtitle { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.empty-card .actions { display:flex; align-items:center; gap: 8px; margin-top: 4px; }

.tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 12px; }
.tile { position: relative; border: 1px solid var(--macos-separator); border-radius: 10px; padding: 12px; background: var(--macos-background); cursor: pointer; transition: background .12s ease, border-color .12s ease, transform .1s ease; }
.tile:hover { background: var(--macos-gray-hover); }
.tile:active { transform: scale(0.995); }
.tile-head { display:flex; align-items:center; justify-content: space-between; }
.tile-icon { width: 26px; height: 26px; display:flex; align-items:center; justify-content:center; background: var(--macos-background-secondary); border: 1px solid var(--macos-separator); border-radius: 6px; }
.tile-title { margin-top: 8px; font-size: var(--fs-base); font-weight: 500; color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tile-meta { margin-top: 6px; font-size: var(--fs-sub); color: var(--macos-text-secondary); display:flex; align-items:center; gap: 6px; }
.tile-tag { position: absolute; top: 10px; right: 10px; font-size: var(--fs-caption); color: var(--macos-text-secondary); background: var(--macos-background-secondary); border: 1px solid var(--macos-separator); border-radius: 999px; padding: 0 6px; height: 18px; display:flex; align-items:center; }

.spinning { animation: macos-spin .6s ease-in-out both; }
@keyframes macos-spin { to { transform: rotate(360deg); } }

/* small icon button */
.sr-icon-btn { width: 22px; height: 22px; display:inline-flex; align-items:center; justify-content:center; line-height:0; border-radius:6px; border: 1px solid var(--macos-separator); color: var(--macos-text-secondary); background: var(--macos-background-secondary); }
.sr-icon-btn:hover { background: var(--macos-gray-hover); color: var(--macos-text-primary); }

/* Hide legacy header (language tabs/metrics toggle) inside SubtitleList on editor view */
:deep(.subtitle-list .language-tabs-container) { display: none !important; }
.editor-head { display:none; }
/* Light tweaks to simplify subtitle editing visuals without changing logic */
:deep(.subtitle-list .item-header) { padding: 6px 0; }
:deep(.subtitle-list .metrics-display) { gap: 8px; }
:deep(.subtitle-list .subtitle-item) { border-radius: 8px; }

/* Align subtitle edit items to home (history list) visual style */
:deep(.subtitle-list .subtitle-items) { position: relative; }
/* flatten row look: no per-row card radius/border, subtle hover only */
:deep(.subtitle-list .subtitle-item) { position: relative; padding: 8px 6px; transition: background .12s ease; border-radius: 0; border-bottom: 0; }
:deep(.subtitle-list .subtitle-item:hover) { background: var(--macos-gray-hover); }
/* separators with side insets (don’t touch borders) */
:deep(.subtitle-list .subtitle-item + .subtitle-item)::before { content: ''; position: absolute; top: 0; left: 6px; right: 6px; height: 1px; background: var(--macos-divider-weak); }
/* container: no extra border to avoid double-framing with outer layout */
:deep(.subtitle-list) { box-shadow: none !important; border: 0; border-radius: 0; }
/* selected/editing row state */
:deep(.subtitle-list .subtitle-item.is-editing) {
  background: color-mix(in oklab, var(--macos-blue) 6%, var(--macos-background));
}
/* focus-within ring for keyboard users */
:deep(.subtitle-list .subtitle-item:focus-within) { outline: 2px solid color-mix(in oklab, var(--macos-blue) 50%, #fff 0%); outline-offset: 2px; }

/* Header row compact alignment */
:deep(.subtitle-list .item-header .item-controls) { display:flex; align-items:center; gap: 8px; }
:deep(.subtitle-list .item-header .item-number) { display:inline-flex; align-items:center; justify-content:center; min-width: 20px; height: 20px; font-size: var(--fs-caption); color: var(--macos-text-secondary); border: 1px solid var(--macos-separator); border-radius: 999px; padding: 0 6px; }
:deep(.subtitle-list .time-info .time-range .time-display) { font-size: var(--fs-sub); color: var(--macos-text-secondary); }

/* Metrics pills similar to home chips */
:deep(.subtitle-list .metrics-display) { display:flex; align-items:center; gap: 6px; }
/* unified metric group */
:deep(.subtitle-list .metric-group) { position: relative; display:inline-flex; align-items:center; gap: 6px; }
:deep(.subtitle-list .metric-group .dots) { display:inline-flex; align-items:center; gap: 6px; }
/* Dot base: all white to reduce visual fatigue */
:deep(.subtitle-list .metric-group .dot) { width: 8px; height: 8px; border-radius: 50%; background: #fff; color: #fff; display:inline-block; }
/* details hidden by default */
:deep(.subtitle-list .metric-group .details) { display:none; align-items:center; }
/* Capsule container shown on hover */
:deep(.subtitle-list .metric-group .details .capsule) { display:inline-flex; align-items:center; gap: 6px; height: 22px; padding: 0 8px; border-radius: 999px; border: 1px solid var(--macos-separator); background: var(--macos-background); font-size: var(--fs-caption); font-weight: 400; }
:deep(.subtitle-list .metric-group .details .capsule .item) { display:inline-flex; align-items:center; gap: 6px; font-weight: 400; }
:deep(.subtitle-list .metric-group .details .capsule .item .k) { color: var(--macos-text-secondary); font-weight: 400; }
:deep(.subtitle-list .metric-group .details .capsule .item .v) { color: currentColor; font-weight: 500; }
:deep(.subtitle-list .metric-group .details .capsule .item.duration .k) { display:none; }
:deep(.subtitle-list .metric-group .details .capsule .sep) { color: var(--macos-text-tertiary); opacity: 0.6; line-height: 1; }
/* hover expand (non-narrow) */
:deep(.subtitle-list .metric-group:hover .dots) { display:none; }
:deep(.subtitle-list .metric-group:hover .details) { display:inline-flex; }
/* color accents for text in details by level; keep dots white except danger */
:deep(.subtitle-list .metric-group .dot.level-neutral) { background: #fff; color: #fff; }
:deep(.subtitle-list .metric-group .details .item.level-normal) { color: var(--macos-success-text); }
:deep(.subtitle-list .metric-group .details .item.level-warning) { color: #ff9f0a; }
:deep(.subtitle-list .metric-group .details .item.level-danger) { color: var(--macos-danger-text); }
/* Only danger dot gets color */
:deep(.subtitle-list .metric-group .dot.level-danger) { color: var(--macos-danger-text); background: var(--macos-danger-text); }

/* Content text area spacing */
:deep(.subtitle-list .item-content) { margin-top: 4px; }
:deep(.subtitle-list .text-content .subtitle-text) { font-size: var(--fs-base); color: var(--macos-text-primary); }
.editor-head .left { display:flex; align-items:center; gap:8px; }
.editor-head .left .k { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.editor-head .right { display:flex; align-items:center; gap:8px; }

/* segmented (local minimal copy) */
/* use global .segmented/.seg-item */
.seg-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
.seg-item.active, .seg-item.active:hover { background: var(--macos-blue); color: #fff; }

/* list view */
.list-wrap { display:block; }
.list-wrap.compact .list-row { height: 34px; }
.list-wrap.comfortable .list-row { height: 44px; }
.list-header { position: sticky; top: 0; z-index: 1; background: var(--macos-background); color: var(--macos-text-tertiary); font-size: var(--fs-caption); text-transform: uppercase; letter-spacing: .02em; padding: 8px 6px; border: 0; }
.list-wrap .list-header { margin-top: 10px; }
.list-wrap .list-header:first-of-type { margin-top: 0; }
.list-row { display:grid; grid-template-columns: 34px 1fr auto 120px 40px; align-items:center; gap: 8px; padding: 0 6px; border-radius: 6px; transition: background .12s ease; }
.list-row:hover { background: var(--macos-gray-hover); }
.col-icon { display:flex; align-items:center; justify-content:center; }
.col-title { min-width:0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; font-size: var(--fs-base); color: var(--macos-text-primary); }
.col-title { display:flex; align-items:center; gap: 6px; }
.col-title .rename-input { height: 22px; padding: 0 6px; border-radius: 6px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-primary); font-size: var(--fs-sub); min-width: 160px; }
.col-title .rename-btn { visibility: hidden; }
.list-row:hover .col-title .rename-btn { visibility: visible; }
.col-pills { display:flex; justify-content:flex-end; }
.col-time { display:flex; flex-direction: column; align-items:flex-end; justify-content:center; gap: 2px; }
.col-time .t-rel { font-size: var(--fs-sub); color: var(--macos-text-secondary); line-height: 1; }
.col-time .t-abs { font-size: var(--fs-caption); color: var(--macos-text-tertiary); line-height: 1; }
.col-meta, .col-date, .col-ext { font-size: var(--fs-sub); color: var(--macos-text-secondary); white-space: nowrap; }
.col-actions { display:flex; align-items:center; justify-content:flex-end; }
.io-sentinel { height: 1px; }

/* pill group (borrowed from CookiesPanel with tweaks) */
.meta-group.small .item { display:inline-flex; align-items:center; gap: 4px; font-size: 11.5px; }
.meta-group.small .divider-v { width: 1px; height: 12px; background: var(--macos-divider-weak); }
.mono { font-family: var(--font-mono); }

/* floating filter */
.floating-filter { position: fixed; bottom: 16px; z-index: 1200; display: inline-flex; align-items: center; gap: 4px; padding: 6px; border-radius: 10px; 
  /* stronger frosted look */
  background: color-mix(in oklab, var(--macos-surface) 80%, transparent);
  border: 1px solid rgba(255,255,255,0.22);
  -webkit-backdrop-filter: var(--macos-surface-blur);
  backdrop-filter: var(--macos-surface-blur);
  box-shadow: var(--macos-shadow-2);
}
.floating-filter .count-pill { font-size: var(--fs-sub); }
.floating-filter .lang-label { font-size: var(--fs-sub); color: var(--macos-text-primary); }
.floating-filter .filter-toggle { display:inline-flex; align-items:center; gap:6px; cursor: pointer; color: var(--macos-text-secondary); height: 28px; padding: 0 6px; border-radius: 6px; line-height: 0; }
.floating-filter .count-pill { line-height: 1; }
.floating-filter .filter-select { height: 28px; }
.floating-filter .divider-v { width: 1px; height: 18px; background: var(--macos-divider-weak); margin: 0; }
.floating-filter .sr-icon-btn { width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center; line-height: 0; background: transparent; border: 0; color: var(--macos-text-secondary); }
.floating-filter .sr-icon-btn.expand-left { width: auto; padding-left: 6px; padding-right: 0; gap: 6px; overflow: hidden; }
.floating-filter .sr-icon-btn.expand-left .label { max-width: 0; opacity: 0; transform: translateX(4px); transition: max-width .18s ease, opacity .18s ease, transform .18s ease, color .12s ease; white-space: nowrap; font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-left: -6px; }
.floating-filter .sr-icon-btn.expand-left .w-4 { transition: transform .18s ease, color .12s ease; }
.floating-filter .sr-icon-btn.expand-left:hover { color: var(--macos-blue); padding-right: 6px; }
.floating-filter .sr-icon-btn.expand-left:hover .label { max-width: 140px; opacity: 1; transform: translateX(0); color: var(--macos-blue); margin-left: 0; }
.floating-filter .sr-icon-btn.expand-left:hover .w-4 { transform: translateX(-2px); }

/* bottom cues floating pill (shows when end sentinel is visible) */
.cues-floating-pill { position: fixed; bottom: 24px; z-index: 1100; pointer-events: none; text-align: center; }
.cues-floating-pill .label { display:inline-flex; align-items:center; height: 22px; padding: 0 10px; border-radius: 999px; border: 1px solid rgba(255,255,255,0.22); 
  background: color-mix(in oklab, var(--macos-surface) 78%, transparent); color: var(--macos-text-secondary); font-size: var(--fs-sub);
  -webkit-backdrop-filter: var(--macos-surface-blur); backdrop-filter: var(--macos-surface-blur); box-shadow: var(--macos-shadow-1);
}
.floating-filter .sr-icon-btn .w-4, .floating-filter .filter-toggle .w-4 { display: block; }
.floating-filter .filter-toggle .count { display: block; }

/* danger button tone for delete */
/* subtitle page delete button follows download style: icon only, no bg */
.col-actions .sr-icon-btn { background: transparent; border: 0; color: var(--macos-text-secondary); }
.col-actions .sr-icon-btn:hover { color: var(--macos-danger-text); }

/* format accents */
.ext-tag.ext-srt { color: var(--macos-blue); }
.ext-tag.ext-ass { color: #8e8ce7; }
.ext-tag.ext-vtt { color: #16a34a; }
.ext-tag.ext-itt { color: #ff9f0a; }
.tile-icon.ext-srt { border-color: var(--ext-srt); color: var(--ext-srt); background: color-mix(in oklab, var(--ext-srt) 10%, var(--macos-background)); }
.tile-icon.ext-ass { border-color: var(--ext-ass); color: var(--ext-ass); background: color-mix(in oklab, var(--ext-ass) 10%, var(--macos-background)); }
.tile-icon.ext-vtt { border-color: var(--ext-vtt); color: var(--ext-vtt); background: color-mix(in oklab, var(--ext-vtt) 10%, var(--macos-background)); }
.tile-icon.ext-itt { border-color: var(--ext-itt); color: var(--ext-itt); background: color-mix(in oklab, var(--ext-itt) 10%, var(--macos-background)); }

/* Disable generic tooltips in non-narrow mode for metric dots, so expand view dominates */
:not(.narrow) :deep(.has-tooltip[data-tooltip])::after { display: none !important; }

/* Narrow mode: compress each subtitle item to 2 lines (header + 1-line text) */
.narrow :deep(.subtitle-list .subtitle-item) { padding: 6px 6px; }
.narrow :deep(.subtitle-list .time-info .time-range .time-display) { font-size: var(--fs-caption); }
.narrow :deep(.subtitle-list .metrics-display) { gap: 4px; }
.narrow :deep(.subtitle-list .metric-group .dots) { display:inline-flex; }
.narrow :deep(.subtitle-list .metric-group .details) { display:none !important; }
.narrow :deep(.subtitle-list .text-content .subtitle-text) { display: -webkit-box; -webkit-line-clamp: 1; -webkit-box-orient: vertical; overflow: hidden; }
</style>
