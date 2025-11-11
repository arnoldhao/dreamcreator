<template>
  <div class="dlp-root">
    <!-- progress moved below thumbnail -->

    <!-- Title -->
    <div v-if="task" class="title-bar">
      <div class="title-text one-line" :title="task?.title">{{ task?.title }}</div>
    </div>

    <!-- Cover preview (thumbnail) -->
    <div v-if="task?.thumbnail" class="thumb-box mt-2">
      <ProxiedImage :src="task.thumbnail" :alt="$t('download.thumbnail')"
        class="w-full h-full object-cover" error-icon="image" />
      <!-- Restore overlay progress on thumbnail header -->
      <div class="thumb-progress" v-if="isVideoDownloading">
        <div class="bar" :style="{ width: percentText }"></div>
      </div>
      <!-- Play overlay when a video file exists -->
      <button
        v-if="canPlayVideo"
        class="thumb-play"
        @click="onPlayVideo"
        :title="$t('download.output')"
        aria-label="Play video"
      >
        <Icon name="play" class="w-6 h-6 text-white" />
      </button>
      <div class="thumb-status">
        <button
          class="chip-frosted chip-lg"
          :class="statusBadgeClass(task?.stage)"
          @click.stop="onStageBadgeClick"
          :title="statusText(task?.stage)"
          aria-label="Status"
        >
          <span class="chip-dot"></span>
          <span class="chip-label">
            <span class="label-swap" :class="{ hoverable: isFailed }">
              <span class="label-a">{{ statusText(task?.stage) }}</span>
              <span class="label-b" v-if="isFailed">{{ t('download.analysis.title') }}</span>
            </span>
          </span>
        </button>
      </div>
    </div>
    <AnalysisModal v-if="analysisVisible" v-model:show="analysisVisible" :task-id="task?.id || ''" :task-error="task?.error || ''" />

    <!-- Download Process group -->
    <div class="macos-group">
      <div class="macos-group-title">{{ $t('download.download_process') }}</div>
      <div class="macos-box card-frosted card-translucent">
        <div class="macos-row"><span class="k">{{ $t('download.stage_video') }}</span><span class="v">
          <span class="chip-frosted chip-md chip-translucent" :class="stageBadge(stages.video)"><span class="chip-dot"></span><span class="chip-label">{{ stageText(stages.video) }}</span></span>
        </span></div>
        <div class="macos-row"><span class="k">{{ $t('download.stage_merge') }}</span><span class="v">
          <span class="chip-frosted chip-md chip-translucent" :class="stageBadge(stages.merge)"><span class="chip-dot"></span><span class="chip-label">{{ stageText(stages.merge) }}</span></span>
        </span></div>
        <div class="macos-row"><span class="k">{{ $t('download.stage_finalize') }}</span><span class="v">
          <span class="chip-frosted chip-md chip-translucent" :class="stageBadge(stages.finalize)"><span class="chip-dot"></span><span class="chip-label">{{ stageText(stages.finalize) }}</span></span>
        </span></div>
        <div class="macos-row"><span class="k">{{ $t('download.speed') }}</span><span class="v one-line">{{ task?.downloadProcess?.speed || '-' }}</span></div>
        <div class="macos-row"><span class="k">{{ $t('download.estimated_time') }}</span>
          <span class="v one-line" v-if="!isEtaCompleted">{{ formatEstimated(task?.downloadProcess?.estimatedTime) }}</span>
          <span class="v" v-else>
            <span class="chip-frosted chip-md chip-translucent" :class="stageBadge('done')">
              <span class="chip-dot"></span>
              <span class="chip-label">{{ t('download.completed') }}</span>
            </span>
          </span>
        </div>
      </div>
    </div>

    <!-- Subtitle Process group -->
    <div class="macos-group">
      <div class="macos-group-title">{{ $t('download.subtitle_process') }}</div>
      <div class="macos-box card-frosted card-translucent">
        <!-- Put Subtitle Languages as the first row for better visibility -->
        <div class="macos-row"><span class="k">{{ $t('download.sub_langs') }}</span><span class="v one-line">
          <template v-if="subtitleLangs.length">
            <span class="one-line" :title="subtitleLangs.join(', ')">{{ subtitleLangs.join(', ') }}</span><span> · {{ subtitleLangs.length }}</span>
          </template>
          <template v-else>{{ $t('download.no_subtitles') || '-' }}</template>
        </span></div>
        <div class="macos-row"><span class="k">{{ $t('download.status_label') }}</span><span class="v">
          <span class="chip-frosted chip-md chip-translucent" :class="stageBadge(task?.subtitleProcess?.status)"><span class="chip-dot"></span><span class="chip-label">{{ stageText(task?.subtitleProcess?.status) }}</span></span>
        </span></div>
        <div class="macos-row"><span class="k">{{ $t('download.format') }}</span><span class="v one-line">{{ task?.subtitleProcess?.format || '-' }}</span></div>
        <div class="macos-row">
          <span class="k">{{ $t('subtitle.common.edit') || '编辑' }}</span>
          <span class="v">
            <button class="btn-chip-ghost" @click="onOpenSubtitleClick" :disabled="!canEditSubtitle" :title="!canEditSubtitle ? (t('download.processing') || 'Processing') : ''">
              <Icon name="edit" class="w-4 h-4 mr-1" />
              {{ $t('subtitle.common.edit') }}
            </button>
          </span>
        </div>
      </div>
    </div>

    <!-- Subtitle picker modal -->
    <SubtitlePickerModal :show="showPicker"
                         :items="pickerItems"
                         @update:show="(v) => showPicker = v"
                         @confirm="onPickerConfirm" />

    <!-- Group 1: Task Info (boxed) -->
    <div class="macos-group"><div class="macos-group-title">{{ $t('download.task_info') }}</div></div>
    <div class="macos-box card-frosted card-translucent">
      <div class="macos-row"><span class="k">{{ $t('download.source') }}</span><span class="v one-line">{{ taskSource }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.uploader') }}</span><span class="v one-line">{{ task?.uploader || '-' }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.extractor') }}</span><span class="v one-line">{{ task?.extractor || '-' }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.resolution') }}</span><span class="v one-line">{{ task?.resolution || '-' }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.format') }}</span><span class="v one-line">{{ task?.format || '-' }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.duration') }}</span><span class="v one-line">{{ formatDuration(task?.duration) }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('download.file_size') }}</span><span class="v one-line">{{ formatFileSize(task?.fileSize) }}</span></div>
    </div>

    <!-- Group 2: Download Info (URL etc.) -->
    <div class="macos-group">
      <div class="macos-group-title">{{ $t('download.download_info') }}</div>
      <div class="macos-box card-frosted card-translucent">
        <div class="macos-row"><span class="k">{{ $t('download.created_at') }}</span><span class="v one-line">{{ formatDate(task?.createdAt) }}</span></div>
        <div class="macos-row"><span class="k">{{ $t('download.updated_at') }}</span><span class="v one-line">{{ formatDate(task?.updatedAt) }}</span></div>
        <div class="macos-row url-row">
          <span class="k">URL</span>
          <span class="v v-flex">
            <span class="text-clip" :title="task?.url">{{ task?.url }}</span>
            <button class="icon-chip-ghost" :data-tooltip="$t('download.copy_url')" data-tip-pos="top" @click="copy(task?.url)"><Icon name="file-copy" class="w-4 h-4"/></button>
          </span>
        </div>
      </div>
    </div>

    <!-- (Download Settings group removed; fields moved into Task Info / Subtitle Process) -->

    

    <!-- Group 4: Output Files (name line + value line) -->
    <div class="macos-group">
      <div class="macos-group-title">{{ $t('download.output_info') || $t('download.output_files') }}</div>
      <div class="macos-box card-frosted card-translucent">
        <div class="row2">
          <div class="k2">{{ $t('download.output_dir') }}</div>
          <div class="v2">
            <div class="text-clip" :title="task?.outputDir">{{ task?.outputDir }}</div>
            <button v-if="task?.outputDir" class="icon-chip-ghost" :data-tooltip="$t('download.open_folder')" data-tip-pos="top" @click="openDirectory(task.outputDir)"><Icon name="folder" class="w-4 h-4"/></button>
          </div>
        </div>
        <div class="files" v-if="task?.allFiles?.length">
          <div v-for="(f,idx) in task.allFiles" :key="idx" class="row2 file-row2">
            <div class="k2">{{ $t('download.output_files') }} #{{ idx + 1 }}</div>
            <div class="v2">
            <div class="text-clip" :title="f">{{ getFileName(f) }}</div>
            <button class="icon-chip-ghost" :data-tooltip="$t('download.copy_name')" data-tip-pos="top" @click="copy(f)"><Icon name="file-copy" class="w-4 h-4"/></button>
            </div>
          </div>
        </div>
        <div v-else class="row2"><div class="k2">{{ $t('download.output_files') }}</div><div class="v2 text-secondary text-xs">{{ $t('download.no_output_files') }}</div></div>
      </div>
    </div>

    <!-- footer actions removed (moved refresh to inspector header) -->
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import useNavStore from '@/stores/nav.js'
import { useSubtitleStore } from '@/stores/subtitle'
import { useDtStore } from '@/handlers/downtasks'
import { ListTasks } from 'wailsjs/go/api/DowntasksAPI'
import { OpenDirectory, OpenPath } from 'wailsjs/go/systems/Service'
import useInspectorStore from '@/stores/inspector.js'
import useLayoutStore from '@/stores/layout.js'
import eventBus from '@/utils/eventBus.js'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import SubtitlePickerModal from '@/components/modal/SubtitlePickerModal.vue'
import { ImportTaskSubtitle } from 'wailsjs/go/api/DowntasksAPI'
import { formatDuration as fmtDuration, formatFileSize as fmtSize, formatDate as fmtDate, formatEstimated as fmtETA } from '@/utils/format.js'
defineEmits(['open-modal'])

const props = defineProps({ taskId: { type: String, required: true }, onClose: { type: Function, default: null } })
const { t } = useI18n()
const dtStore = useDtStore()
const inspector = useInspectorStore()
const layout = useLayoutStore()
const navStore = useNavStore()
const subtitleStore = useSubtitleStore()
const task = ref(null)
// lightweight stage state for current task
const stages = ref({ video: 'idle', subtitle: 'idle', merge: 'idle', finalize: 'idle' })
const isVideoDownloading = computed(() => (task.value?.stage === 'downloading'))
const canPlayVideo = computed(() => {
  const v = task.value
  if (!v) return false
  const files = Array.isArray(v.videoFiles) ? v.videoFiles : []
  return v.stage === 'completed' && files.length > 0
})

const isFailed = computed(() => (task.value?.stage === 'failed'))

// ETA 是否完成（展示胶囊样式）
const isEtaCompleted = computed(() => {
  try {
    const eta = task.value?.downloadProcess?.estimatedTime
    if (eta == null) return false
    return String(eta).toLowerCase() === 'completed'
  } catch { return false }
})

function resolveAbsolutePath(p) {
  if (!p) return ''
  // crude absolute check for posix/windows
  const isAbs = p.startsWith('/') || /^[A-Za-z]:\\/.test(p) || p.startsWith('\\\\')
  if (isAbs) return p
  const dir = task.value?.outputDir || ''
  if (!dir) return p
  const sep = (dir.includes('\\') && !dir.includes('/')) ? '\\' : '/'
  return dir.replace(/[\\/]$/, '') + sep + p
}

async function onPlayVideo() {
  try {
    const files = Array.isArray(task.value?.videoFiles) ? task.value.videoFiles : []
    if (!files.length) return
    const full = resolveAbsolutePath(files[0])
    const r = await OpenPath(full)
    if (!r?.success) {
      const msg = r?.msg || t('common.error')
      $notification?.error?.(msg, { title: t('common.error') })
    }
  } catch (e) {
    const msg = e?.message || String(e)
    $notification?.error?.(msg, { title: t('common.error') })
  }
}

import AnalysisModal from '@/components/modal/AnalysisModal.vue'
import { ref as vref } from 'vue'
const analysisVisible = vref(false)

async function onStageBadgeClick() {
  try {
    if (task.value?.stage === 'failed') {
      analysisVisible.value = true
      return
    }
  } catch {}
}

function onOpenSubtitleClick() {
  const status = task.value?.subtitleProcess?.status
  const hasPid = !!task.value?.subtitleProcess?.projectId
  if (!hasPid && status === 'error') {
    $dialog?.error?.({ title: t('subtitle.title'), content: t('download.failed_desc') || t('download.failed') })
    return
  }
  if (!hasPid && status !== 'done') {
    $dialog?.info?.({ title: t('subtitle.title'), content: t('download.processing') })
    return
  }
  // 始终走后台导入/验证：如果已存在project则直接返回；若被删除则重新导入
  const importByFile = async (id, filename) => {
    try {
      const fn = window?.go?.api?.DowntasksAPI?.ImportTaskSubtitleByFile
      if (typeof fn === 'function') return await fn(id, filename)
      // fallback: legacy single-file import
      return await ImportTaskSubtitle(id)
    } catch (e) {
      return Promise.reject(e)
    }
  }

  const files = (() => {
    const sp = task.value?.subtitleProcess || {}
    const list = Array.isArray(sp.files) ? sp.files : []
    if (list.length) return list
    return Array.isArray(task.value?.subtitleFiles) ? task.value.subtitleFiles : []
  })()

  const id = task.value?.id
  if (!id) return

  const handleResult = (res) => {
    if (res?.success) {
      try {
        const data = JSON.parse(res.data || '{}')
        const newId = data.projectId
        if (newId) { openSubtitleProject(newId); return }
      } catch {}
      $dialog?.error?.({ title: t('subtitle.title'), content: t('subtitle.common.not_set') || 'Not available' })
      return
    }
    $dialog?.error?.({ title: t('subtitle.title'), content: res?.msg || (t('subtitle.common.not_set') || 'Not available') })
  }

  try {
    if (files.length <= 1) {
      ImportTaskSubtitle(id).then(handleResult).catch(err => $dialog?.error?.({ title: t('subtitle.title'), content: err?.message || String(err) }))
      return
    }
    // open modal picker (by language list)
    pickerTaskId.value = id
    const langs = subtitleLangs.value || []
    pickerItems.value = langs.map(code => ({ label: code, value: code }))
    showPicker.value = true
  } catch (e) {
    $dialog?.error?.({ title: t('subtitle.title'), content: e?.message || String(e) })
  }
}

const refresh = async () => {
  try { const r = await ListTasks(); if (r?.success) {
    const arr = JSON.parse(r.data||'[]'); task.value = arr.find(x => x.id === props.taskId) || task.value
  }} catch {}
}

const onProgress = (data) => {
  if (data?.id !== props.taskId) return
  // merge top-level
  task.value = { ...(task.value||{}), ...data }
  // also mirror into persisted downloadProcess for live refresh
  try {
    task.value.downloadProcess = task.value.downloadProcess || {}
    if (Object.prototype.hasOwnProperty.call(data, 'speed')) task.value.downloadProcess.speed = data.speed
    if (Object.prototype.hasOwnProperty.call(data, 'estimatedTime')) task.value.downloadProcess.estimatedTime = data.estimatedTime
  } catch {}
}
const onSignal = (_data) => refresh()
const onStage = (data) => {
  if (!data || data.id !== props.taskId) return
  const kind = String(data.kind||'').toLowerCase()
  const action = String(data.action||'').toLowerCase()
  if (!['video','subtitle','merge','finalize'].includes(kind)) return
  const val = (action === 'complete') ? 'done' : (action === 'error' ? 'error' : 'working')
  stages.value[kind] = val
  // reflect into persisted fields for current task
  if (!task.value) return
  task.value.downloadProcess = task.value.downloadProcess || {}
  task.value.subtitleProcess = task.value.subtitleProcess || {}
  if (kind === 'subtitle') task.value.subtitleProcess.status = val
  else task.value.downloadProcess[kind] = val
}
import { watch } from 'vue'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'
const subtitleLangs = computed(() => {
  try {
    const sp = task.value?.subtitleProcess || {}
    // Strictly derive from downloaded files if present
    const files = Array.isArray(sp.files) ? sp.files : []
    if (files.length) {
      const langs = []
      const seen = new Set()
      for (const f of files) {
        const b = String(f||'').split(/[\\/]/).pop()
        const dot = b.lastIndexOf('.')
        if (dot <= 0) continue
        const noext = b.slice(0, dot)
        const dot2 = noext.lastIndexOf('.')
        if (dot2 <= 0) continue
        const code = noext.slice(dot2+1)
        const k = code.toLowerCase()
        if (!seen.has(k)) { seen.add(k); langs.push(code) }
      }
      return langs
    }
    // else fallback to persisted languages, sanitize
    let arr = Array.isArray(sp.languages) ? [...sp.languages] : []
    arr = arr.map(s => String(s||'').trim()).filter(Boolean)
    arr = arr.filter(s => !/^\[?download\]?/i.test(s) && !/downloading\s+subtitles/i.test(s) && !/:/.test(s))
    const seen = new Set(); const out = []
    for (const s of arr) { const k = s.toLowerCase(); if (!seen.has(k)) { seen.add(k); out.push(s) } }
    return out
  } catch { return [] }
})

// 仅当实际已下载的字幕文件存在，且字幕流程状态为 done 时，允许点击“编辑”
const canEditSubtitle = computed(() => {
  try {
    const sp = task.value?.subtitleProcess || {}
    const files = Array.isArray(sp.files) ? sp.files : (Array.isArray(task.value?.subtitleFiles) ? task.value.subtitleFiles : [])
    const st = String(sp.status || '').toLowerCase()
    return files.length > 0 && st === 'done'
  } catch { return false }
})
watch(() => props.taskId, () => { task.value = null; refresh() })
// reset local stages when switching task
watch(() => props.taskId, () => {
  stages.value = { video: 'idle', subtitle: 'idle', merge: 'idle', finalize: 'idle' }
})

// seed stages from persisted process if available after refresh
watch(() => task.value, (t) => {
  try {
    if (!t) return
    const dp = t.downloadProcess || {}
    const sp = t.subtitleProcess || {}
    stages.value = {
      video: dp.video || 'idle',
      subtitle: sp.status || 'idle',
      merge: dp.merge || 'idle',
      finalize: dp.finalize || 'idle'
    }
  } catch {}
})

onMounted(async () => { await refresh(); dtStore.registerProgressCallback(onProgress); dtStore.registerSignalCallback(onSignal); dtStore.registerStageCallback(onStage) })
onUnmounted(() => { dtStore.unregisterProgressCallback(onProgress); dtStore.unregisterSignalCallback(onSignal); dtStore.unregisterStageCallback(onStage) })

// picker state
const showPicker = ref(false)
const pickerItems = ref([])
const pickerTaskId = ref('')
function onPickerConfirm(langCode) {
  const id = pickerTaskId.value
  if (!id || !langCode) return
  const importByFile = async (tid, fn) => {
    try {
      const api = window?.go?.api?.DowntasksAPI
      if (api && typeof api.ImportTaskSubtitleByFile === 'function') {
        return await api.ImportTaskSubtitleByFile(tid, fn)
      }
      return await ImportTaskSubtitle(tid)
    } catch (e) { return Promise.reject(e) }
  }
  const files = (() => {
    const sp = task.value?.subtitleProcess || {}
    const list = Array.isArray(sp.files) ? sp.files : []
    if (list.length) return list
    return Array.isArray(task.value?.subtitleFiles) ? task.value.subtitleFiles : []
  })()
  const baseOf = (f) => (f || '').split(/[\\/]/).pop()
  const langFromBase = (b) => {
    const dot = b.lastIndexOf('.')
    if (dot <= 0) return ''
    const noext = b.slice(0, dot)
    const dot2 = noext.lastIndexOf('.')
    if (dot2 <= 0) return ''
    return noext.slice(dot2 + 1)
  }
  const wanted = String(langCode).toLowerCase()
  let chosenBase = ''
  for (const f of files) {
    const b = baseOf(f)
    const lc = langFromBase(b).toLowerCase()
    if (lc && (lc === wanted || lc === wanted.replace('_','-') || lc === wanted.replace('-','_'))) { chosenBase = b; break }
  }
  if (!chosenBase && files.length) { chosenBase = baseOf(files[0]) }
  if (!chosenBase) return
  importByFile(id, chosenBase)
    .then((res) => {
      if (res?.success) {
        try { const data = JSON.parse(res.data || '{}'); const newId = data.projectId; if (newId) { openSubtitleProject(newId); return } } catch {}
      }
      $dialog?.error?.({ title: t('subtitle.title'), content: res?.msg || (t('subtitle.common.not_set') || 'Not available') })
    })
    .catch((err) => $dialog?.error?.({ title: t('subtitle.title'), content: err?.message || String(err) }))
}

const percentText = computed(() => {
  const p = task.value?.percentage
  let n
  if (typeof p === 'number') n = p
  else if (p == null || p === '') n = 0
  else {
    const num = Number(p)
    n = Number.isFinite(num) ? num : 0
  }
  const s = (typeof n === 'number' && Number.isFinite(n) && n.toFixed) ? n.toFixed(1) : String(n)
  return s + '%'
})
const formatDate = (ts) => fmtDate(ts)
const formatEstimated = (eta) => fmtETA(eta)
const taskSource = computed(() => {
  const src = task.value?.source
  if (src && String(src).trim()) return src
  const url = task.value?.url
  try { return url ? new URL(url).hostname : ('' ) } catch { return '' }
})

const copy = async (text) => { await copyToClipboard(text, t) }
const formatDuration = (sec) => fmtDuration(sec, t)
const formatFileSize = (bytes) => fmtSize(bytes, t)
const getFileName = (p='') => p.split(/[\\/]/).pop()

function openSubtitleProject(id) {
  try {
    if (!id) return
    navStore.currentNav = navStore.navOptions.SUBTITLE
    try { subtitleStore.setPendingOpenProjectId(id) } catch {}
    eventBus.emit('subtitle:open-project', id)
  } catch {}
}

// open output directory
const openDirectory = async (path) => {
  try {
    if (!path) return
    OpenDirectory(path)
  } catch (e) {
    $dialog?.error?.({ title: t('common.error'), content: e?.message || String(e) })
  }
}

const statusText = (stage) => ({
  'initializing': t('download.initializing'),
  'downloading': t('download.downloading'),
  'paused': t('download.paused'),
  'translating': t('download.translating'),
  'embedding': t('download.embedding'),
  'completed': t('download.completed'),
  'failed': t('download.failed'),
  'cancelled': t('download.cancelled'),
})[stage] || t('download.unknown_status')

const statusBadgeClass = (stage) => ({
  'initializing': 'badge-info',
  'downloading': 'badge-primary',
  'paused': 'badge-warning',
  'translating': 'badge-primary',
  'embedding': 'badge-primary',
  'completed': 'badge-success',
  'failed': 'badge-error',
  'cancelled': 'badge-error',
})[stage] || 'badge-ghost'

function stageBadge(status) {
  switch (status) {
    case 'working': return 'badge-info'
    case 'done': return 'badge-success'
    case 'error': return 'badge-error'
    default: return 'badge-ghost'
  }
}
function stageText(status) {
  switch (status) {
    case 'working': return (t('download.processing') || 'Processing')
    case 'done': return (t('download.completed') || 'Completed')
    case 'error': return (t('download.failed') || 'Failed')
    default: return '-'
  }
}

function closePanel() {
  try { props.onClose && props.onClose() } catch {}
  inspector.close()
}

// listen toolbar refresh
onMounted(() => eventBus.on('download_task:refresh', refresh))
onUnmounted(() => eventBus.off('download_task:refresh', refresh))

</script>

<style scoped>
.dlp-root { padding: 10px; font-size: var(--fs-base); color: var(--macos-text-primary); }
.progress { width:100%; height:2px; background: var(--macos-divider-weak); border-radius: 999px; overflow:hidden; margin-top: 6px; }
.progress.big { height:3px; }
.progress .bar { height:100%; background: var(--macos-blue); }
.thumb-box { width: 100%; height: 140px; border: 1px solid var(--macos-separator); border-radius: 10px; overflow: hidden; background: var(--macos-background-secondary); box-shadow: var(--macos-shadow-1); }
.thumb-box { position: relative; }
.thumb-box::after { content: ''; position: absolute; inset: 0; background: linear-gradient(to top, rgba(0,0,0,0.22), rgba(0,0,0,0.06) 40%, rgba(0,0,0,0) 70%); pointer-events: none; }
.thumb-status { position:absolute; right:8px; bottom:8px; z-index:1; }
.thumb-status .chip-frosted .chip-label { display:inline-flex; align-items:center; height:100%; line-height:1; }
.thumb-status .label-swap { display:inline-block; }
.thumb-status .label-b { display:none; }
.thumb-status .chip-frosted.badge-error:hover .label-swap.hoverable .label-a { display:none; }
.thumb-status .chip-frosted.badge-error:hover .label-swap.hoverable .label-b { display:inline; }
.thumb-progress { position:absolute; left:0; right:0; top:0; height:3px; background: rgba(0,0,0,0.18); z-index:2; border-top-left-radius:10px; border-top-right-radius:10px; overflow:hidden; }
.thumb-progress .bar { height:100%; background: var(--macos-blue); }
.thumb-play { position:absolute; inset: 0; margin: auto; z-index:3; 
  /* 可调参数：
     - --play-d: 毛玻璃圆直径
     - --play-shift: 光学居中所需的水平偏移（右正）
     - --play-scale: 播放三角的缩放，保证与圆的边距更均衡
   */
  --play-d: 48px; 
  --play-shift: 2.33px; 
  --play-scale: 0.96; 
  width: var(--play-d); height: var(--play-d); border-radius: 999px; 
  border: 1px solid rgba(255,255,255,0.28); background: rgba(30,30,30,0.45); 
  -webkit-backdrop-filter: saturate(150%) blur(10px); backdrop-filter: saturate(150%) blur(10px); 
  display:flex; align-items:center; justify-content:center; color: #fff; 
  box-shadow: 0 1px 0 rgba(255,255,255,0.18) inset, 0 10px 24px rgba(0,0,0,0.35), 0 2px 6px rgba(0,0,0,0.2); 
  transition: transform .15s ease, background-color .2s ease, box-shadow .2s ease, border-color .2s ease, opacity .2s ease; 
  opacity: .96; transform: scale(1); transform-origin: 50% 50%; 
}
.thumb-play:hover { background: rgba(30,30,30,0.55); border-color: rgba(255,255,255,0.38); box-shadow: 0 1px 0 rgba(255,255,255,0.22) inset, 0 12px 28px rgba(0,0,0,0.38), 0 3px 8px rgba(0,0,0,0.22); transform: scale(1.04); }
.thumb-play:active { transform: scale(0.98); }
@supports not ((-webkit-backdrop-filter: blur(10px)) or (backdrop-filter: blur(10px))) {
  .thumb-play { background: rgba(255,255,255,0.14); color: #111; border-color: rgba(0,0,0,0.15); }
}
/* 光学居中：用视框质心推算出的偏移（约 2.33px），并略微缩放以均衡三顶点与圆边的间距 */
.thumb-play :deep(svg) { transform: translateX(var(--play-shift)) scale(var(--play-scale)); transform-origin: 50% 50%; }

/* Frosted status chip (bottom-right), matches play control aesthetic */
/* row chips and thumb chips now use global .chip-frosted */
.macos-box .v { position: relative; }
.macos-box .v-flex { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 8px; min-width: 0; }
.macos-box .v-flex .text-clip { min-width: 0; }
.url-row .icon-chip-ghost, .v2 .icon-chip-ghost { width: 28px; height: 28px; padding: 0; }
.one-line { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.macos-group {} /* spacing handled globally */
.macos-group-title {} /* typographic style handled globally */
.group-body { display:flex; flex-direction: column; gap: 6px; }
.kv { display:grid; grid-template-columns: 90px 1fr auto; align-items:center; gap:6px; }
.kv .k { color: var(--macos-text-secondary); font-size: var(--fs-sub); }
.kv .v { font-size: var(--fs-sub); color: var(--macos-text-primary); }
.kv2 { display: flex; flex-direction: column; gap: 4px; }
.k2 { color: var(--macos-text-secondary); font-size: var(--fs-sub); }
.v2 { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 8px; min-width: 0; }
.row2 { display:flex; flex-direction: column; gap:4px; padding: 8px 10px; }
.row2 + .row2 { position: relative; }
.row2 + .row2::before { content: ''; position: absolute; top: 0; left: 10px; right: 10px; height: 1px; background: var(--macos-divider-weak); }
.file-row2 { }
.text-clip { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.files { display:flex; flex-direction:column; gap:6px; max-height: none; overflow: visible; }
.file-row { display:flex; align-items:center; justify-content: space-between; gap:8px; font-size: var(--fs-sub); }
.truncate { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dlp-root { overflow-x: hidden; padding-bottom: 56px; }
.macos-box > .files { position: relative; }
.macos-box > .files::before { content: ''; position: absolute; top: 0; left: 10px; right: 10px; height: 1px; background: var(--macos-divider-weak); }

/* title + status row */
.title-bar { display:flex; align-items:center; justify-content: space-between; gap:8px; padding: 6px 2px 2px; }
.title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); min-width: 0; }

/* (picker styles removed; using prompt for now) */
.macos-box .chip-frosted.chip-translucent { background: transparent; border-color: var(--macos-separator); color: var(--macos-text-secondary); box-shadow: none; }
.macos-box .chip-frosted.chip-translucent:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); border-color: var(--macos-blue); color: #fff; }
</style>
