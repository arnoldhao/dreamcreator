<template>
  <div class="macos-card card-frosted card-translucent dl-card bg-progress" :class="{ active }"
       @click.stop="$emit('card-click')" :style="{ '--progress': progressValue }">
    <div class="thumb">
      <ProxiedImage v-if="task.thumbnail" :src="task.thumbnail" :alt="$t('download.thumbnail')"
        class="w-full h-full object-cover rounded" error-icon="video" />
      <div v-else class="thumb-fallback">
        <Icon name="video" class="w-4 h-4"></Icon>
      </div>
    </div>
    <div class="main">
      <div class="title-row">
        <div class="title" :title="task.title">{{ task.title }}</div>
      </div>
      <div class="meta">{{ formatDuration(task.duration) }} · {{ formatFileSize(task.fileSize) }} · {{ task.extractor }}</div>
      <div class="bottom-row">
        <div class="uploader" v-if="task.uploader" :title="task.uploader">{{ task.uploader }}</div>
        <div class="chip-frosted chip-md chip-translucent bottom-stats" :class="combinedStatClass" @click.stop="onBottomPillClick" :title="pillTitle">
          <span class="chip-dot"></span>
          <span class="chip-label">{{ combinedStatText }}</span>
        </div>
      </div>
    </div>
    <div class="ops">
      <button v-if="task.outputDir" class="icon-glass" :data-tooltip="$t('download.open_folder')" @click.stop="$emit('open-directory')">
        <Icon name="folder" class="w-4 h-4" />
      </button>
      <button class="icon-glass" :data-tooltip="$t('download.delete')" @click.stop="$emit('delete')">
        <Icon name="trash" class="w-4 h-4" />
      </button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import { formatDuration as fmtDuration, formatFileSize as fmtSize } from '@/utils/format.js'

const props = defineProps({
  task: { type: Object, required: true },
  active: { type: Boolean, default: false },
})
const { t } = useI18n()
const pillTitle = computed(() => props.task?.stage === 'failed' ? (t('download.failed_desc') || '') : '')

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


const progressWidth = computed(() => {
  const p = props.task?.percentage
  let nRaw
  if (typeof p === 'number') nRaw = p
  else if (p == null) nRaw = 0
  else {
    // 支持字符串如 "45.6" 或 "45.6%"
    const s = String(p).trim().replace('%', '')
    const num = parseFloat(s)
    nRaw = Number.isFinite(num) ? num : 0
  }
  const n = Math.min(100, Math.max(0, nRaw))
  return n.toFixed(1) + '%'
})

const progressValue = computed(() => {
  const s = progressWidth.value
  return parseFloat(String(s).replace('%','')) || 0
})

const progressShort = computed(() => {
  const n = progressValue.value
  return `${Math.round(n)}%`
})

const speedText = computed(() => {
  const s = props.task?.speed
  if (s == null || s === '') return ''
  return String(s)
})

const showOverlay = computed(() => {
  const stage = props.task?.stage
  if (stage !== 'downloading') return false
  return (progressValue.value >= 0 && progressValue.value <= 100) || !!speedText.value
})

// Combined indicator text (prefer active sub/merge/finalize phases over raw percent)
const combinedStatText = computed(() => {
  const task = props.task || {}
  const stage = task.stage
  const dp = task.downloadProcess || {}
  const sp = task.subtitleProcess || {}

  // If subtitles are in progress or finished, surface them even while overall stage remains 'downloading'
  const total = (() => {
    const a = Array.isArray(sp.languages) ? sp.languages.length : 0
    const b = Array.isArray(task.subLangs) ? task.subLangs.length : 0
    return Math.max(a, b)
  })()
  if (sp.status === 'working') {
    const current = Array.isArray(sp.files) ? sp.files.length : 0
    return `${t('download.stage_subtitles')}: ${t('download.processing')}${total ? ` · ${current}/${total}` : ''}`
  }
  if (sp.status === 'done' && stage !== 'completed') {
    return `${t('download.stage_subtitles')}: ${t('download.complete')}${total ? ` · ${total}` : ''}`
  }
  if (sp.status === 'error') return `${t('download.stage_subtitles')}: ${t('download.failed')}`

  // Otherwise, if downloading video, show percent + speed
  if (stage === 'downloading') {
    const s = progressShort.value
    return speedText.value ? `${s} · ${speedText.value}` : s
  }

  // Or other working phases
  if (dp.merge === 'working') return (t('download.merging') || 'Merging')
  if (dp.finalize === 'working') return (t('download.finalizing') || 'Finalizing')

  // Fallback to overall status text
  return statusText(stage)
})

// Use status color mapping for pill styling (prefer subtitle phase when active)
const combinedStatClass = computed(() => {
  const task = props.task || {}
  const stage = task.stage
  const sp = task.subtitleProcess || {}
  if (sp.status === 'working') return 'badge-info'
  if (sp.status === 'done' && stage !== 'completed') return 'badge-success'
  if (sp.status === 'error') return 'badge-error'
  return statusBadgeClass(stage)
})

function onBottomPillClick() {
  try {
    const stage = props.task?.stage
    if (stage === 'failed') {
      const msg = props.task?.error || t('download.failed_desc') || t('download.failed')
      $dialog?.error?.({ title: t('download.failed'), content: msg })
    }
  } catch {}
}

const formatDuration = (sec) => fmtDuration(sec, t)
const formatFileSize = (bytes) => fmtSize(bytes, t)
</script>

<style scoped>
.dl-card { position: relative; display:grid; grid-template-columns: 64px 1fr auto; gap: 10px; align-items:center; padding: 8px; border-radius: 8px; transition: border-color .2s ease, box-shadow .2s ease, transform .15s ease; }
.dl-card.card-frosted { background: var(--macos-surface); border-color: rgba(255,255,255,0.22); -webkit-backdrop-filter: var(--macos-surface-blur); backdrop-filter: var(--macos-surface-blur); box-shadow: var(--macos-shadow-2); }
.dl-card.bg-progress::before { content: ''; position: absolute; inset: 0 auto 0 0; width: calc(var(--progress, 0) * 1%); background: color-mix(in oklab, var(--macos-blue) 14%, transparent); border-radius: 8px; pointer-events: none; transition: width .2s ease; }
.dl-card:hover, .dl-card.active { border-color: rgba(255,255,255,0.32); box-shadow: 0 10px 30px rgba(0,0,0,0.20); transform: translateY(-0.5px); }
.thumb { width: 64px; height: 40px; border-radius: 6px; overflow:hidden; background: var(--macos-background-secondary); display:flex; align-items:center; justify-content:center; }
.thumb-fallback { width:100%; height:100%; display:flex; align-items:center; justify-content:center; color: var(--macos-text-tertiary); }
.main { min-width: 0; }
.title-row { display:flex; align-items:center; gap: 8px; min-width:0; }
.title-row { display:flex; align-items:center; gap: 8px; min-width:0; justify-content: space-between; }
.title-row .title { flex: 1 1 auto; min-width: 0; }
.title-row .badge { flex-shrink: 0; }
.title { font-size: var(--fs-base); font-weight: 500; color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding-right: 0; }
.meta { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bottom-row { display:flex; align-items:center; justify-content: space-between; margin-top: 2px; gap: 8px; min-width: 0; }
.uploader { flex: 1 1 auto; min-width: 0; font-size: var(--fs-sub); color: var(--macos-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bottom-stats { flex: 0 0 auto; white-space: nowrap; font-size: var(--fs-caption); }
/* Softer status chip on cards: transparent by default, subtle hover fill */
.bottom-stats.chip-frosted.chip-translucent { background: transparent; border-color: rgba(255,255,255,0.18); color: var(--macos-text-secondary); box-shadow: none; }
.bottom-stats.chip-frosted.chip-translucent:hover { background: color-mix(in oklab, var(--macos-blue) 22%, transparent); border-color: var(--macos-blue); color: #fff; }

/* Make the two ops buttons less prominent by default */
.ops .icon-glass { background: transparent; border-color: rgba(255,255,255,0.16); box-shadow: none; color: var(--macos-text-secondary); }
.ops .icon-glass:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); border-color: var(--macos-blue); color: #fff; }

/* Classic mode: keep the same de-emphasis (avoid solid white buttons) */
:global([data-ui='classic']) .dl-card .ops .icon-glass { background: transparent !important; border-color: var(--macos-separator) !important; box-shadow: none !important; color: var(--macos-text-secondary) !important; }
:global([data-ui='classic']) .dl-card .ops .icon-glass:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent) !important; border-color: var(--macos-blue) !important; color: var(--macos-text-primary) !important; }
:global([data-ui='classic']) .dl-card .bottom-stats.chip-frosted.chip-translucent { background: transparent !important; border-color: var(--macos-separator) !important; color: var(--macos-text-secondary) !important; box-shadow: none !important; }
.progress { width: 100%; height: 2px; background: var(--macos-divider-weak); border-radius: 999px; overflow: hidden; margin-top: 6px; }
.progress .bar { height: 100%; background: var(--macos-blue); transition: width .18s ease; }
.ops { display:flex; align-items:center; gap: 6px; white-space: nowrap; }
</style>
