<template>
  <div class="macos-page min-w-0" @click="onPageClick">

    <div class="dl-grid">
      <div v-if="filteredTasks.length === 0" class="dl-empty">
        <!-- Empty for all: no tasks yet -->
        <div v-if="isFilterAll" class="macos-card card-frosted card-translucent empty-card" @click.stop>
          <div class="icon-wrap">
            <div class="icon-bg">
              <Icon name="download-cloud" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
            </div>
          </div>
          <div class="title">{{ $t('download.no_download_tasks') }}</div>
          <div class="subtitle">{{ $t('download.start_first_download_task') }}</div>
          <div class="actions">
            <button class="btn-glass btn-primary btn-sm" @click.stop="showDownloadModal = true">
              <Icon name="plus" class="w-4 h-4 mr-1" />
              {{ $t('download.new_task') }}
            </button>
            <button class="btn-glass btn-sm" @click.stop="onRefreshClick">
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
          <div class="subtitle">{{ currentFilterLabel }}</div>
          <div class="actions">
            <button class="btn-glass btn-primary btn-sm" @click.stop="resetFilters">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.reset') }}
            </button>
            <button class="btn-glass btn-sm" @click.stop="onRefreshClick">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>
      </div>
      <TransitionGroup v-else name="dl-tiles" tag="div" class="tiles">
        <DownloadTaskCard v-for="task in filteredTasks" :key="task.id" :task="task" :active="activeTaskId === task.id"
          @card-click="onTaskClick(task)"
          @open-directory="openDirectory(task.outputDir)"
          @delete="deleteTaskWithConfirm(task.id)" />
      </TransitionGroup>
    </div>

    <!-- 占位拉伸：确保页面下方空白属于本组件，从而触发根点击关闭 inspector -->
    <div class="dl-spacer"></div>

    <!-- modals -->
    <VideoDownloadModal :show="showDownloadModal" @update:show="showDownloadModal = $event" @download-started="onDownloadStarted" />
    <CookiesManagerModal v-if="showCookies" @close="showCookies = false" />

    <!-- floating filter at bottom-right -->
    <div class="floating-filter" @click.stop :style="{ right: (inspector.visible ? (layout.inspectorWidth + 12) : 12) + 'px' }">
      <button class="icon-glass" :data-tooltip="$t('download.refresh')" data-tip-pos="top" @click="onRefreshClick">
        <Icon name="refresh" class="w-4 h-4" :class="{ spinning: refreshing }" />
      </button>
      <div class="divider-v"></div>
      <div class="filter-toggle" @click="floatingFilterExpanded = !floatingFilterExpanded">
        <Icon name="filter" class="w-4 h-4" />
        <span class="chip-frosted chip-sm chip-translucent count-pill">
          <span class="chip-label">{{ filteredTasks.length }}/{{ tasks.length }}</span>
        </span>
      </div>
      <select v-if="floatingFilterExpanded" v-model="filter" class="input-macos select-macos select-macos-xs filter-select">
        <option v-for="opt in filterOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</option>
      </select>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import { ListTasks, DeleteTask } from 'wailsjs/go/api/DowntasksAPI'
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useDtStore } from '@/handlers/downtasks'
import eventBus from '@/utils/eventBus.js'
import useInspectorStore from '@/stores/inspector.js'
import useLayoutStore from '@/stores/layout.js'
import DownloadTaskCard from '@/components/download/DownloadTaskCard.vue'
import VideoDownloadModal from '@/components/modal/VideoDownloadModal.vue'
import CookiesManagerModal from '@/components/modal/CookiesManagerModal.vue'
import { formatDuration as fmtDuration, formatFileSize as fmtSize } from '@/utils/format.js'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

const { t } = useI18n()

const tasks = ref([])
const query = ref('')
const filter = ref('all')
const activeTaskId = ref(null)
const dtStore = useDtStore()
const inspector = useInspectorStore()
const layout = useLayoutStore()
const showDownloadModal = ref(false)
const showCookies = ref(false)
const floatingFilterExpanded = ref(false)
const refreshing = ref(false)

const filterOptions = computed(() => [
  { value: 'all', label: t('download.all') },
  { value: 'downloading', label: t('download.downloading') },
  { value: 'translating', label: t('download.translating') },
  { value: 'embedding', label: t('download.embedding') },
  { value: 'completed', label: t('download.completed') },
  { value: 'failed', label: t('download.failed') },
])

const refreshTasks = async () => {
  try {
    const r = await ListTasks()
    if (r?.success) {
      tasks.value = JSON.parse(r.data || '[]')
    }
  } catch (e) { console.warn('ListTasks failed', e) }
}

async function onRefreshClick() {
  if (refreshing.value) return
  try {
    refreshing.value = true
    await refreshTasks()
  } finally {
    setTimeout(() => { refreshing.value = false }, 200) // allow animation to complete
  }
}

const filteredTasks = computed(() => {
  const q = query.value.toLowerCase()
  return (tasks.value || []).filter(tk => {
    const m1 = filter.value === 'all' || tk.stage === filter.value
    const m2 = !q || (tk.title || '').toLowerCase().includes(q) || (tk.url || '').toLowerCase().includes(q)
    return m1 && m2
  })
})

const isFilterAll = computed(() => filter.value === 'all')
const currentFilterLabel = computed(() => {
  const opt = (filterOptions.value || []).find(o => o.value === filter.value)
  return opt ? opt.label : ''
})

const selectedTask = computed(() => tasks.value.find(t => t.id === activeTaskId.value))

const copy = async (text) => { await copyToClipboard(text, t) }

const formatDuration = (sec) => fmtDuration(sec, t)
const formatFileSize = (bytes) => fmtSize(bytes, t)

const statusText = (stage) => {
  const m = {
    'initializing': t('download.initializing'),
    'downloading': t('download.downloading'),
    'paused': t('download.paused'),
    'translating': t('download.translating'),
    'embedding': t('download.embedding'),
    'completed': t('download.completed'),
    'failed': t('download.failed'),
    'cancelled': t('download.cancelled'),
  }
  return m[stage] || t('download.unknown_status')
}

const statusBadgeClass = (stage) => {
  const colorMap = {
    'initializing': 'badge-info',
    'downloading': 'badge-primary',
    'paused': 'badge-warning',
    'translating': 'badge-primary',
    'embedding': 'badge-primary',
    'completed': 'badge-success',
    'failed': 'badge-error',
    'cancelled': 'badge-error'
  }
  return colorMap[stage] || 'badge-ghost'
}

const deleteTask = async (taskId) => {
  try {
    const r = await DeleteTask(taskId)
    if (!r?.success) { throw new Error(r?.msg || 'Delete failed') }
    $message?.success?.(t('common.delete_success'))
    if (activeTaskId.value === taskId) activeTaskId.value = null
    refreshTasks()
  } catch (e) {
    $dialog?.error?.({ title: t('common.error'), content: e.message })
  }
}

const deleteTaskWithConfirm = async (taskId) => {
  const task = tasks.value.find(t => t.id === taskId)
  const content = t('common.delete_confirm_detail', { title: task?.title || '' })
  $dialog?.confirm(content, {
    title: t('common.delete_confirm'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => deleteTask(taskId),
  })
}

// open output directory
const openDirectory = async (path) => {
  try {
    if (!path) return
    OpenDirectory(path)
  } catch (e) {
    $dialog?.error?.({ title: t('common.error'), content: e.message || String(e) })
  }
}

onMounted(refreshTasks)

// live updates via websocket
const onProgress = (data) => {
  if (!data || !data.id) return
  const i = tasks.value.findIndex(t => t.id === data.id)
  if (i >= 0) {
    const next = { ...tasks.value[i], ...data }
    // also mirror speed/eta into persisted process for list/inspector coherence
    next.downloadProcess = next.downloadProcess || {}
    if (Object.prototype.hasOwnProperty.call(data, 'speed')) next.downloadProcess.speed = data.speed
    if (Object.prototype.hasOwnProperty.call(data, 'estimatedTime')) next.downloadProcess.estimatedTime = data.estimatedTime
    tasks.value[i] = next
  }
}
const onSignal = (_data) => { refreshTasks() }
const onStage = (data) => {
  if (!data || !data.id) return
  const i = tasks.value.findIndex(t => t.id === data.id)
  if (i >= 0) {
    const prev = tasks.value[i] || {}
    const stages = { video: 'idle', subtitle: 'idle', merge: 'idle', finalize: 'idle', ...(prev.downloadProcess ? { video: prev.downloadProcess.video, merge: prev.downloadProcess.merge, finalize: prev.downloadProcess.finalize } : {}), ...(prev.subtitleProcess ? { subtitle: prev.subtitleProcess.status } : {}) }
    const kind = String(data.kind || '').toLowerCase()
    const action = String(data.action || '').toLowerCase()
    if (['video','subtitle','merge','finalize'].includes(kind)) {
      stages[kind] = (action === 'complete') ? 'done' : (action === 'error' ? 'error' : 'working')
    }
    // also reflect into persisted fields
    const next = { ...prev }
    next.downloadProcess = next.downloadProcess || {}
    next.subtitleProcess = next.subtitleProcess || {}
    if (kind === 'subtitle') next.subtitleProcess.status = stages.subtitle
    else next.downloadProcess[kind] = stages[kind]

    // When video phase completes, clear transient speed to avoid stale "100% · X MB/s" during subtitle stage
    if (kind === 'video' && action === 'complete') {
      next.speed = ''
      if (next.downloadProcess) next.downloadProcess.speed = ''
    }
    tasks.value[i] = next
  }
}
onMounted(() => {
  dtStore.registerProgressCallback(onProgress)
  dtStore.registerSignalCallback(onSignal)
  dtStore.registerStageCallback(onStage)
  eventBus.on('download:search', (q) => { query.value = q || '' })
  eventBus.on('download:refresh', refreshTasks)
  eventBus.on('download:new-task', () => { showDownloadModal.value = true })
  // bridge clicks from Inspector header
  eventBus.on('download:toggle-cookies', onToggleCookies)
  eventBus.on('download:toggle-detail', onToggleDetail)
})
onUnmounted(() => {
  dtStore.unregisterProgressCallback(onProgress)
  dtStore.unregisterSignalCallback(onSignal)
  dtStore.unregisterStageCallback(onStage)
  eventBus.off('download:search', () => {})
  eventBus.off('download:refresh', refreshTasks)
  eventBus.off('download:new-task', () => {})
  eventBus.off('download:toggle-cookies', onToggleCookies)
  eventBus.off('download:toggle-detail', onToggleDetail)
})

function onTaskClick(task) {
  // Toggle behavior: clicking the current task closes the inspector
  if (inspector.visible && activeTaskId.value === task.id) {
    inspector.close()
    activeTaskId.value = null
    return
  }
  activeTaskId.value = task.id
  inspector.open('DownloadTaskPanel', t('download.detail'), { taskId: task.id, onClose: () => { if (activeTaskId.value === task.id) activeTaskId.value = null } })
  // inspector.visible controlled by store
}

function onToggleCookies() {
  if (!inspector.visible) {
    inspector.open('CookiesPanel', t('cookies.title'))
    return
  }
  if (inspector.panel === 'CookiesPanel') {
    inspector.close()
  } else {
    inspector.open('CookiesPanel', t('cookies.title'))
  }
}

function onToggleDetail() {
  if (!inspector.visible) {
    if (activeTaskId.value) {
      inspector.open('DownloadTaskPanel', t('download.detail'), { taskId: activeTaskId.value, onClose: () => { if (activeTaskId.value && inspector.panel !== 'DownloadTaskPanel') activeTaskId.value = activeTaskId.value } })
    } else {
      inspector.open('InspectorHomePanel', t('download.detail'))
    }
    return
  }
  if (inspector.panel === 'DownloadTaskPanel' || (inspector.panel === 'InspectorHomePanel' && inspector.title === t('download.detail'))) {
    inspector.close()
  } else {
    if (activeTaskId.value) {
      inspector.open('DownloadTaskPanel', t('download.detail'), { taskId: activeTaskId.value, onClose: () => { if (activeTaskId.value && inspector.panel !== 'DownloadTaskPanel') activeTaskId.value = activeTaskId.value } })
    } else {
      inspector.open('InspectorHomePanel', t('download.detail'))
    }
  }
}

const isCookiesActive = computed(() => inspector.visible && inspector.panel === 'CookiesPanel')
const isDetailActive = computed(() => inspector.visible && inspector.panel === 'DownloadTaskPanel')

function onPageClick() {
  // Click anywhere on the page closes inspector and collapses filter
  inspector.close()
  activeTaskId.value = null
  floatingFilterExpanded.value = false
}

function onDownloadStarted(payload) {
  try { showDownloadModal.value = false } catch {}
  refreshTasks()
  const id = payload && (payload.id || payload.taskId)
  if (id) activeTaskId.value = id
}

function resetFilters() {
  filter.value = 'all'
  query.value = ''
}
</script>

<style scoped>
.macos-page { min-height: 100%; display:flex; flex-direction: column; }
.dl-grid { display: block; min-height: 0; width: 100%; padding: 0 12px; }
.dl-spacer { flex: 1 1 auto; }


.tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; transition: grid-template-columns .18s ease, gap .18s ease; }
.dl-tiles-move { transition: transform .2s ease; will-change: transform; }
.dl-tiles-enter-active, .dl-tiles-leave-active { transition: all .2s ease; }
.dl-tiles-enter-from, .dl-tiles-leave-to { opacity: 0; transform: scale(0.98); }
.dl-actions { display:flex; align-items:center; gap: 8px; }
.dl-search { width: 200px; height: 26px; }

.dl-card { padding: 8px; }
 .dl-empty { padding: 40px 12px; display:flex; align-items:center; justify-content:center; }
 .empty-card { width: 100%; max-width: 560px; padding: 24px; border-radius: 12px; display:flex; flex-direction:column; align-items:center; text-align:center; gap: 12px; }
 .empty-card .icon-wrap { margin-bottom: 4px; }
 .empty-card .icon-bg { width: 56px; height: 56px; border-radius: 50%; display:flex; align-items:center; justify-content:center; background: linear-gradient(180deg, rgba(0,0,0,0.03), rgba(0,0,0,0.06)); border: 1px solid var(--macos-separator); }
.empty-card .title { font-size: var(--fs-title); font-weight: 600; color: var(--macos-text-primary); }
.empty-card .subtitle { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
 .empty-card .actions { display:flex; align-items:center; gap: 8px; margin-top: 4px; }

.task-row { display:grid; grid-template-columns: 64px 1fr auto; gap: 10px; align-items:center; padding: 8px; border-radius: 8px; transition: background .12s ease; }
.task-row:hover { background: var(--macos-gray-hover); }
.task-row.active { background: var(--macos-gray-hover); }
.thumb { width: 64px; height: 40px; border-radius: 6px; overflow:hidden; background: var(--macos-background-secondary); display:flex; align-items:center; justify-content:center; }
.thumb-fallback { width:100%; height:100%; display:flex; align-items:center; justify-content:center; color: var(--macos-text-tertiary); }
.title-row { display:flex; align-items:center; gap: 8px; min-width:0; }
.title { font-size: var(--fs-base); font-weight: 500; color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.meta { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-top: 2px; }
.progress { width: 100%; height: 2px; background: var(--macos-divider-weak); border-radius: 999px; overflow: hidden; margin-top: 6px; }
.progress .bar { height: 100%; background: var(--macos-blue); }
.ops { display:flex; align-items:center; gap: 6px; white-space: nowrap; }

/* use global .segmented/.seg-item */

/* floating filter */
.floating-filter { position: fixed; bottom: 16px; z-index: 1200; display: inline-flex; align-items: center; gap: 6px; padding: 6px; border-radius: 10px; 
  /* stronger frosted look */
  background: color-mix(in oklab, var(--macos-surface) 80%, transparent);
  border: 1px solid rgba(255,255,255,0.22);
  -webkit-backdrop-filter: var(--macos-surface-blur);
  backdrop-filter: var(--macos-surface-blur);
  box-shadow: var(--macos-shadow-2);
}
/* Align count chip visuals with Subtitle page (use global chip styles) */
.floating-filter .count-pill { font-size: var(--fs-sub); line-height: 1; }
.floating-filter .filter-toggle { display:inline-flex; align-items:center; gap:6px; cursor: pointer; color: var(--macos-text-secondary); height: 28px; padding: 0 6px; border-radius: 6px; line-height: 0; }
.floating-filter .count { line-height: 1; }
.floating-filter .filter-select { height: 28px; }
.floating-filter .divider-v { width: 1px; height: 18px; background: var(--macos-divider-weak); margin: 0 2px; }
/* refresh click animation */
.floating-filter .spinning { animation: macos-spin .6s ease-in-out both; }
@keyframes macos-spin { to { transform: rotate(360deg); } }
/* normalize icon vertical metrics to avoid baseline drift */
.floating-filter .icon-glass, .floating-filter .sr-icon-btn { width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center; line-height: 0; background: transparent; border-color: var(--macos-separator); color: var(--macos-text-secondary); box-shadow: none; }
.floating-filter .icon-glass:hover, .floating-filter .sr-icon-btn:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); border-color: var(--macos-blue); color: #fff; }
.floating-filter .sr-icon-btn .w-4, .floating-filter .filter-toggle .w-4 { display: block; }
.floating-filter .filter-toggle .count-pill { display: block; line-height: 1; }
</style>
