<template>
<div class="macos-card card-frosted card-translucent chip-card dl-card bg-progress" :class="{ active }"
       @click.stop="$emit('card-click')" :style="{ '--progress': progressValue }">
    <div class="thumb">
      <ProxiedImage v-if="task.thumbnail" :src="task.thumbnail" :alt="$t('download.thumbnail')"
        class="rounded" error-icon="video" />
      <div v-else class="thumb-fallback">
        <Icon name="video" class="w-4 h-4"></Icon>
      </div>
      <!-- overlay actions on hover: show open-folder + delete when completed; otherwise only centered delete -->
      <div class="thumb-overlay" @click.stop>
        <button v-if="isCompleted && task.outputDir" class="icon-chip-ghost" :aria-label="$t('download.open_folder')" @click.stop="$emit('open-directory')">
          <Icon name="folder" class="w-4 h-4" />
        </button>
        <button class="icon-chip-ghost" :aria-label="$t('download.delete')" @click.stop="$emit('delete')">
          <Icon name="trash" class="w-4 h-4" />
        </button>
      </div>
    </div>
    <div class="main">
      <div class="title-row">
        <div class="title" :title="titleText">{{ titleText }}</div>
      </div>
      <div class="meta">{{ formatDuration(task.duration) }} · {{ formatFileSize(task.fileSize) }} · {{ extractorText }}</div>
      <div class="bottom-row">
        <div class="uploader" :title="uploaderText">{{ uploaderText }}</div>
        <div class="chip-frosted chip-md chip-translucent bottom-stats" :class="combinedStatClass" @click.stop="onBottomPillClick" :title="pillTitle">
          <span class="chip-dot"></span>
          <span class="chip-label">
            <span class="label-swap" :class="{ hoverable: isFailed }">
              <span class="label-a">{{ combinedStatText }}</span>
              <span class="label-b" v-if="isFailed">{{ t('download.analysis.title') }}</span>
            </span>
          </span>
        </div>
      </div>
    </div>
    
    <!-- mount analysis modal locally; fixed overlay ensures global appearance -->
    <AnalysisModal 
      v-if="analysisVisible" 
      v-model:show="analysisVisible" 
      :task-id="task?.id || ''" 
      :task-error="task?.error || ''" 
    />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import { formatDuration as fmtDuration, formatFileSize as fmtSize } from '@/utils/format.js'
import AnalysisModal from '@/components/modal/AnalysisModal.vue'

const props = defineProps({
  task: { type: Object, required: true },
  active: { type: Boolean, default: false },
})
const { t } = useI18n()
const titleText = computed(() => {
  const s = props.task?.title
  return (s && String(s).trim()) ? s : (t('download.unknown_video') || 'Unknown Video')
})
const extractorText = computed(() => {
  const s = props.task?.extractor
  return (s && String(s).trim()) ? s : (t('download.unknown_source') || 'Unknown Source')
})
const uploaderText = computed(() => {
  const s = props.task?.uploader
  return (s && String(s).trim()) ? s : (t('download.unknown_uploader') || t('common.unknown') || 'Unknown')
})
const isCompleted = computed(() => props.task?.stage === 'completed')
const isFailed = computed(() => props.task?.stage === 'failed')
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

// removed unused showOverlay (no overlay UI)

// no sweep on select; keep selection subtle

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

const analysisVisible = ref(false)

async function onBottomPillClick() {
  try {
    const stage = props.task?.stage
    if (stage === 'failed') {
      analysisVisible.value = true
      return
    }
  } catch {}
}

const formatDuration = (sec) => fmtDuration(sec, t)
const formatFileSize = (bytes) => fmtSize(bytes, t)
</script>

<style scoped>
.dl-card.chip-card { /* make chip-card visibly lighter than default */ --chip-card-surface: 66%; }
.dl-card { position: relative; display:grid; grid-template-columns: minmax(96px, 1fr) minmax(var(--dl-main-min, 220px), var(--dl-main-max, 380px)); gap: 10px; align-items:center; padding: 8px; border-radius: 8px; transition: border-color .2s ease, box-shadow .2s ease; }
.dl-card.card-frosted { -webkit-backdrop-filter: var(--macos-surface-blur); backdrop-filter: var(--macos-surface-blur); box-shadow: var(--macos-shadow-2); }
.dl-card.bg-progress::before { content: ''; position: absolute; inset: 0 auto 0 0; width: calc(var(--progress, 0) * 1%); background: color-mix(in oklab, var(--macos-blue) 14%, transparent); border-radius: 8px; pointer-events: none; transition: width .2s ease; z-index: 0; }
/* Hover: subtle lift only */
.dl-card:hover { /* no hover lift to reduce GPU usage */ }
/* Active: accent-tinted focus ring + inner stroke for clear selection */
.dl-card.active { border-color: rgba(255,255,255,0.34); box-shadow: 0 10px 26px rgba(0,0,0,0.20); }
.dl-card::after { content: none; }
.dl-card:hover::after { content: none; }
.dl-card.active::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 8px;
  pointer-events: none;
  z-index: 1;
  /* subtle inner accent stroke only */
  box-shadow: inset 0 0 0 1px color-mix(in oklab, var(--macos-blue) 28%, white 8%);
}
/* no sweep on selection */
/*
  Fix card/thumb deformation by enforcing a fixed 16:9 area for thumbnails.
  - The thumb container keeps a 16:9 ratio based on its width.
  - ProxiedImage fills the container and uses object-fit: cover (default) to avoid distortion.
*/
.thumb {
  position: relative;
  width: 100%;
  aspect-ratio: 16 / 9;
  border-radius: 6px;
  overflow: hidden;
  background: var(--macos-background-secondary);
}
.thumb-fallback {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  display:flex;
  align-items:center;
  justify-content:center;
  color: var(--macos-text-tertiary);
}
.thumb :deep(.proxied-image-wrapper) {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}
/* Ensure the img fully covers the fixed-ratio box */
.thumb :deep(.proxied-image-wrapper .image-content) {
  width: 100% !important;
  height: 100% !important;
  object-fit: cover !important;
  display: block;
  border-radius: 6px;
}
.thumb-overlay { position: absolute; inset: 0; display:flex; align-items:center; justify-content:center; gap: 6px; background: rgba(0,0,0,0.22); border-radius: 6px; opacity: 0; pointer-events: none; transition: opacity .15s ease; }
.dl-card:hover .thumb-overlay { opacity: 1; pointer-events: auto; }
.thumb-overlay .icon-chip-ghost {
  border-color: rgba(255,255,255,0.28);
  color: #fff;
  box-shadow: none;
  /* Avoid GPU re-composite flicker when overlay fades in */
  -webkit-backdrop-filter: none !important;
  backdrop-filter: none !important;
  background: rgba(0,0,0,0.32);
  transform: none !important;
  transition: background-color .18s ease, border-color .18s ease, color .12s ease !important;
}
.thumb-overlay .icon-chip-ghost:hover {
  background: color-mix(in oklab, var(--macos-blue) 22%, transparent);
  border-color: var(--macos-blue);
  color: #fff;
  transform: none !important;
}
.main { min-width: 0; }
.title-row { display:flex; align-items:center; gap: 8px; min-width:0; justify-content: space-between; }
.title-row .title { flex: 1 1 auto; min-width: 0; }
.title-row .badge { flex-shrink: 0; }
.title { font-size: var(--fs-base); font-weight: 500; color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding-right: 0; }
.meta { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin-top: 2px; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bottom-row { display:flex; align-items:center; justify-content: space-between; margin-top: 2px; gap: 8px; min-width: 0; }
.uploader { flex: 1 1 auto; min-width: 0; font-size: var(--fs-sub); color: var(--macos-text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.bottom-stats { flex: 0 0 auto; white-space: nowrap; font-size: var(--fs-caption); }
.bottom-stats .chip-label { position: relative; overflow: hidden; display: inline-flex; align-items: center; height: 100%; line-height: 1; }
.bottom-stats .label-swap { display: inline-block; }
.bottom-stats .label-b { display: none; }
.bottom-stats.badge-error:hover .label-swap.hoverable .label-a { display: none; }
.bottom-stats.badge-error:hover .label-swap.hoverable .label-b { display: inline; }
/* Softer status chip on cards: transparent by default, subtle hover fill */
.bottom-stats.chip-frosted.chip-translucent { background: transparent; border-color: rgba(255,255,255,0.18); color: var(--macos-text-secondary); box-shadow: none; }
.bottom-stats.chip-frosted.chip-translucent:hover { background: color-mix(in oklab, var(--macos-blue) 22%, transparent); border-color: var(--macos-blue); color: #fff; }

/* Failed state: make it semi-transparent danger tint for visibility */
.bottom-stats.chip-frosted.chip-translucent.badge-error {
  background: var(--macos-danger-bg);
  border-color: var(--macos-danger-text);
  color: var(--macos-danger-text);
}
.bottom-stats.chip-frosted.chip-translucent.badge-error:hover {
  background: color-mix(in oklab, var(--macos-danger-text) 18%, transparent);
  border-color: var(--macos-danger-text);
  color: #fff;
}

/* Classic UI: still emphasize failed state with warning tint */
:global([data-ui='classic']) .dl-card .bottom-stats.chip-frosted.chip-translucent.badge-error {
  background: var(--macos-danger-bg) !important;
  border-color: var(--macos-danger-text) !important;
  color: var(--macos-danger-text) !important;
}
:global([data-ui='classic']) .dl-card .bottom-stats.chip-frosted.chip-translucent.badge-error:hover {
  background: color-mix(in oklab, var(--macos-danger-text) 16%, transparent) !important;
  border-color: var(--macos-danger-text) !important;
  color: var(--macos-text-primary) !important;
}

</style>
