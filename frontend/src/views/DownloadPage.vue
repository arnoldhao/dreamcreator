<template>
  <div class="macos-page min-w-0" @click.capture="onBackgroundClick">

    <div class="dl-grid">
      <div v-if="filteredTasks.length === 0" class="dl-empty dc-empty-shell">
        <!-- Empty for all: no tasks yet -->
        <div v-if="isFilterAll" class="macos-card card-frosted card-translucent dc-empty-card" @click.stop>
          <div class="dc-icon-wrap">
            <div class="dc-icon-bg">
              <Icon name="download-cloud" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
            </div>
          </div>
          <div class="dc-empty-title">{{ $t('download.no_download_tasks') }}</div>
          <div class="dc-empty-subtitle">{{ $t('download.start_first_download_task') }}</div>
          <div class="dc-empty-actions">
            <button class="btn-chip-ghost btn-primary btn-sm" @click.stop="showDownloadModal = true">
              <Icon name="plus" class="w-4 h-4 mr-1" />
              {{ $t('download.new_task') }}
            </button>
            <button class="btn-chip-ghost btn-sm" @click.stop="onRefreshClick">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>
        <!-- Empty for filtered: no results under current filter -->
        <div v-else class="macos-card card-frosted card-translucent dc-empty-card" @click.stop>
          <div class="dc-icon-wrap">
            <div class="dc-icon-bg">
              <Icon name="filter-x" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
            </div>
          </div>
          <div class="dc-empty-title">{{ $t('download.no_filter_results') }}</div>
          <div class="dc-empty-subtitle">{{ currentFilterLabel }}</div>
          <div class="dc-empty-actions">
            <button class="btn-chip-ghost btn-primary btn-sm" @click.stop="resetFilters">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('common.reset') }}
            </button>
            <button class="btn-chip-ghost btn-sm" @click.stop="onRefreshClick">
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

    <!-- floating filter at bottom-right -->
    <div class="floating-filter chip-frosted chip-translucent chip-panel" @click.stop :style="{ right: floatingRight + 'px' }">
      <button class="icon-chip-ghost" :data-tooltip="$t('download.refresh')" data-tip-pos="top" @click="onRefreshClick">
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
import { ref, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import { ListTasks, DeleteTask } from 'bindings/dreamcreator/backend/api/downtasksapi'
import { OpenDirectory } from 'bindings/dreamcreator/backend/services/systems/service'
import { useDtStore } from '@/stores/downloadTasks'
import eventBus from '@/utils/eventBus.js'
import useInspectorStore from '@/stores/inspector.js'
import useNavStore from '@/stores/nav.js'
import useLayoutStore from '@/stores/layout.js'
import DownloadTaskCard from '@/components/download/DownloadTaskCard.vue'
import VideoDownloadModal from '@/components/modal/VideoDownloadModal.vue'
import { formatDuration as fmtDuration, formatFileSize as fmtSize } from '@/utils/format.js'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

const { t } = useI18n()

const tasks = ref([])
const query = ref('')
const filter = ref('all')
const activeTaskId = ref(null)
const dtStore = useDtStore()
const inspector = useInspectorStore()
const navStore = useNavStore()
const layout = useLayoutStore()
const showDownloadModal = ref(false)
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
      const arr = JSON.parse(r.data || '[]')
      // 合并最近一次的进度（来自 dtStore.taskProgressMap），确保切回页面时不会出现短暂的进度缺失
      try {
        const map = dtStore?.taskProgressMap || {}
        tasks.value = (arr || []).map(item => {
          const id = item?.id
          const overlay = id ? map[id] : null
          if (!overlay) return item
          const merged = { ...item, ...overlay }
          // 回填速度/ETA 到持久字段，避免 UI 在切回时出现空白
          merged.downloadProcess = merged.downloadProcess || {}
          if (Object.prototype.hasOwnProperty.call(overlay, 'speed')) merged.downloadProcess.speed = overlay.speed
          if (Object.prototype.hasOwnProperty.call(overlay, 'estimatedTime')) merged.downloadProcess.estimatedTime = overlay.estimatedTime
          return merged
        })
      } catch {
        tasks.value = arr
      }
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

const floatingRight = computed(() => {
  const base = inspector.visible ? (layout.inspectorWidth + 12) : 12
  return base + 6
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
  eventBus.on('download:toggle-detail', onToggleDetail)
  // If inspector is closed (e.g., returning to this page), ensure selection is cleared
  if (!inspector.visible) activeTaskId.value = null
  // 页面可见性/焦点恢复时主动刷新一次，兜底获取后端最新数据
  try {
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('focus', onWindowFocus)
  } catch {}
})
onUnmounted(() => {
  dtStore.unregisterProgressCallback(onProgress)
  dtStore.unregisterSignalCallback(onSignal)
  dtStore.unregisterStageCallback(onStage)
  eventBus.off('download:search', () => {})
  eventBus.off('download:refresh', refreshTasks)
  eventBus.off('download:new-task', () => {})
  eventBus.off('download:toggle-detail', onToggleDetail)
  try {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('focus', onWindowFocus)
  } catch {}
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

// 页面恢复刷新：当页面重新可见或窗口获得焦点时拉取一次任务列表
function onVisibilityChange() {
  try { if (document.visibilityState === 'visible') refreshTasks() } catch {}
}
function onWindowFocus() { refreshTasks() }

const isDetailActive = computed(() => inspector.visible && inspector.panel === 'DownloadTaskPanel')

// Capture clicks at page root to ensure blank areas reliably close inspector
function onBackgroundClick(ev) {
  try {
    const path = typeof ev?.composedPath === 'function' ? ev.composedPath() : []
    const hasClass = (el, cls) => !!(el && el.classList && el.classList.contains && el.classList.contains(cls))
    const inCard = path.some(el => hasClass(el, 'dl-card'))
    const inFloating = path.some(el => hasClass(el, 'floating-filter'))
    const inModal = path.some(el => hasClass(el, 'macos-modal'))
    if (inCard || inFloating || inModal) return
    inspector.close()
    activeTaskId.value = null
    floatingFilterExpanded.value = false
  } catch {}
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

// Keep selection in sync with inspector visibility and navigation
watch(() => inspector.visible, (v) => {
  if (!v) activeTaskId.value = null
})
watch(() => navStore.currentNav, (v) => {
  if (v === navStore.navOptions.DOWNLOAD && !inspector.visible) activeTaskId.value = null
})
</script>

<style scoped>
.macos-page { min-height: 100%; display:flex; flex-direction: column; }
.dl-grid { display: block; min-height: 0; width: 100%; padding: 0 12px; }
.dl-spacer { flex: 1 1 auto; }


.tiles { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px; transition: grid-template-columns .18s ease, gap .18s ease; }
.dl-tiles-move { transition: transform .2s ease; will-change: transform; }
.dl-tiles-enter-active, .dl-tiles-leave-active { transition: all .2s ease; }
.dl-tiles-enter-from, .dl-tiles-leave-to { opacity: 0; transform: scale(0.98); }
.dl-actions { display:none; }
.dl-search { display:none; }

/* remove legacy task row styles (migrated to DownloadTaskCard) */

/* use global .segmented/.seg-item */
</style>
