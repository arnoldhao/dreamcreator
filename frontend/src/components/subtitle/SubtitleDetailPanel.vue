<template>
  <div class="subtitle-detail-panel">
    <!-- Failure reason: place ABOVE progress card; no extra macOS group/box wrapper -->
    <div class="error-panel" v-if="hasError">
      <div class="error-header">
        <div class="left">
          <Icon name="alert-triangle" class="w-4 h-4 mr-1" />
          <span class="title">{{ $t('download.analysis.section_error') }}</span>
        </div>
        <div class="right">
          <button class="btn-chip-icon" :data-tooltip="$t('common.copy')" @click="copyFailureReason">
            <Icon name="file-copy" class="w-4 h-4"/>
          </button>
        </div>
      </div>
      <div class="error-body">
        <pre class="error-text mono">{{ errorPlain }}</pre>
      </div>
    </div>

    <div class="macos-group">
        <div class="macos-group-title">{{ title || $t('download.translate_desc') }}</div>
        <div class="macos-box card-frosted card-translucent">
          <!-- Header: language + status + optional actions -->
          <div class="tp-header">
            <div class="left">
              <!-- 状态 + Token 速度合并为一个 Chip，右对齐，点击打开对话 -->
              <span
                class="chip-frosted chip-translucent chip-md clickable status-speed-chip"
                :class="syncStatus ? syncStatusChipClass : statusChipClass"
                role="button"
                :title="(syncStatus ? syncStatusText(syncStatus) : statusText(status)) + (tokenSpeed ? ' · ' + tokenSpeed : '')"
                @click="onStatusClick"
              >
                <span class="chip-dot"></span>
                <span class="chip-label one-line">
                  {{ syncStatus ? syncStatusText(syncStatus) : statusText(status) }}
                  <span v-if="tokenSpeed" class="sep"> · </span>
                  <span v-if="tokenSpeed" class="mono">{{ tokenSpeed }}</span>
                </span>
              </span>
            </div>
          </div>

          <!-- Failure reason moved above (no wrapper) -->

          <!-- Rows -->
          <div class="macos-row">
            <span class="k">{{ $t('subtitle.detail.progress') }}</span>
            <span class="v">
              <div class="tp-progress inline">
                <div class="bar"><span class="fill" :style="{ width: (progressPct || 0) + '%' }"></span></div>
                <div class="pct mono">{{ Number(progressPct || 0).toFixed(0) }}%</div>
              </div>
            </span>
          </div>
          <div class="macos-row"><span class="k">{{ $t('subtitle.list.cues') || 'Cues' }}</span><span class="v mono one-line">{{ Number(projectTotal || 0) }}</span></div>
          <div class="macos-row"><span class="k">{{ $t('subtitle.detail.processed') }}</span><span class="v mono one-line">{{ processed }}/{{ total }}</span></div>
          <div class="macos-row"><span class="k">{{ failedLabel }}</span><span class="v mono one-line">{{ failedCount }}</span></div>
          <div class="macos-row"><span class="k">{{ $t('subtitle.detail.start') }}</span><span class="v mono one-line">{{ startTimeText || '-' }}</span></div>
          <div class="macos-row"><span class="k">{{ $t('subtitle.detail.elapsed') }}</span><span class="v mono one-line">{{ elapsedText || '-' }}</span></div>
          <!-- Token usage -->
          <div class="macos-row">
            <span class="k">{{ $t('subtitle.detail.tokens') || 'Tokens' }}</span>
            <span class="v mono one-line">{{ tokenTotal }} (P {{ tokenPrompt }}, C {{ tokenCompletion }})</span>
          </div>
          <div class="macos-row">
            <span class="k">{{ $t('subtitle.detail.requests') || 'Requests' }}</span>
            <span class="v mono one-line">{{ requestCountNum }}</span>
          </div>
          

          <div class="macos-row">
            <span class="k">{{ $t('subtitle.add_language.provider') }}</span>
            <span class="v mono one-line">{{ provider || '-' }}</span>
          </div>
          <div class="macos-row">
            <span class="k">{{ $t('subtitle.add_language.model') }}</span>
            <span class="v mono one-line">{{ model || '-' }}</span>
          </div>

          <!-- Error / retry controls -->
          <div v-if="showRetryControls" class="macos-row row-full">
            <div v-if="!isNarrow" class="actions-line">
              <div class="left-selects">
                <select :value="providerSelected" @change="onProviderChange($event)" class="input-macos select-macos tp-select" :title="$t('subtitle.add_language.provider')">
                  <option value="">{{ $t('subtitle.add_language.select_provider') }}</option>
                  <option v-for="p in providers" :key="p.id || p" :value="p.id || p">{{ p.name || p }}</option>
                </select>
                <select :value="modelSelected" @change="onModelChange($event)" class="input-macos select-macos tp-select" :title="$t('subtitle.add_language.model')">
                  <option value="">{{ $t('subtitle.add_language.select_model') }}</option>
                  <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
                </select>
              </div>
              <div class="right-ops">
                <label class="about-toggle with-label align-left">
                  <input type="checkbox" class="about-toggle-input" :checked="retryFailedOnly" @change="$emit('update:retryFailedOnly', $event.target.checked)" />
                  <span class="about-toggle-slider"></span>
                  <span class="toggle-text">{{ $t('subtitle.add_language.retry_failed_only') }}</span>
                </label>
                <button class="btn-chip-icon" @click="$emit('retry')" :title="$t('download.retry')">
                  <Icon name="refresh" class="w-4 h-4" />
                </button>
              </div>
            </div>

            <!-- Narrow stacked controls -->
            <div v-else class="stack-actions center">
              <div class="stack-line center">
                <select :value="providerSelected" @change="onProviderChange($event)" class="input-macos select-macos tp-select" :title="$t('subtitle.add_language.provider')">
                  <option value="">{{ $t('subtitle.add_language.select_provider') }}</option>
                  <option v-for="p in providers" :key="p.id || p" :value="p.id || p">{{ p.name || p }}</option>
                </select>
              </div>
              <div class="stack-line center">
                <select :value="modelSelected" @change="onModelChange($event)" class="input-macos select-macos tp-select" :title="$t('subtitle.add_language.model')">
                  <option value="">{{ $t('subtitle.add_language.select_model') }}</option>
                  <option v-for="m in models" :key="m" :value="m">{{ m }}</option>
                </select>
              </div>
              <div class="stack-line center">
                <label class="about-toggle with-label">
                  <input type="checkbox" class="about-toggle-input" :checked="retryFailedOnly" @change="$emit('update:retryFailedOnly', $event.target.checked)" />
                  <span class="about-toggle-slider"></span>
                  <span class="toggle-text">{{ $t('subtitle.add_language.retry_failed_only') }}</span>
                </label>
                <button class="btn-chip-icon" @click="$emit('retry')">
                  <Icon name="refresh" class="w-4 h-4" />
                </button>
              </div>
            </div>
          </div>

          <!-- Message / File / Format -->
          <div class="macos-row" v-if="message">
            <span class="k">{{ $t('subtitle.detail.message') }}</span>
            <span class="v mono one-line">{{ message }}</span>
          </div>
          <!-- File/Format moved to summary; remove duplication here -->
        </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import Icon from '@/components/base/Icon.vue'
import { useI18n } from 'vue-i18n'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

const props = defineProps({
  title: { type: String, default: '' },
  language: { type: String, default: '' },
  status: { type: String, default: '' },
  progressPct: { type: Number, default: 0 },
  processed: { type: [Number, String], default: 0 },
  total: { type: [Number, String], default: 0 },
  failedCount: { type: [Number, String], default: 0 },
  startTimeText: { type: String, default: '' },
  elapsedText: { type: String, default: '' },
  provider: { type: String, default: '' },
  model: { type: String, default: '' },
  message: { type: String, default: '' },
  filePath: { type: String, default: '' },
  format: { type: String, default: '' },
  errorText: { type: String, default: '' },
  providers: { type: Array, default: () => [] },
  models: { type: Array, default: () => [] },
  providerSelected: { type: String, default: '' },
  modelSelected: { type: String, default: '' },
  retryFailedOnly: { type: Boolean, default: true },
  isNarrow: { type: Boolean, default: false },
  showRetryControls: { type: Boolean, default: false },
  // Token usage (optional)
  promptTokens: { type: Number, default: 0 },
  completionTokens: { type: Number, default: 0 },
  totalTokens: { type: Number, default: 0 },
  requestCount: { type: Number, default: 0 },
  // Token speed display text (e.g. "32.1 tok/s" or "120 tok/req")
  tokenSpeed: { type: String, default: '' },
  // Language-level sync status (e.g., done/failed/partial_failed/translating)
  syncStatus: { type: String, default: '' },
  // Project-level total cues (unfiltered)
  projectTotal: { type: Number, default: 0 },
})

const emit = defineEmits(['update:provider', 'update:model', 'update:retryFailedOnly', 'retry', 'open-llm-talk'])
const { t } = useI18n()
// Token helpers computed from props to avoid direct template/JS variable lookup issues
const tokenTotal = computed(() => Number(props.totalTokens || 0))
const tokenPrompt = computed(() => Number(props.promptTokens || 0))
const tokenCompletion = computed(() => Number(props.completionTokens || 0))
const requestCountNum = computed(() => Number(props.requestCount || 0))
const hasTokens = computed(() => tokenTotal.value > 0 || tokenPrompt.value > 0 || tokenCompletion.value > 0)
// error helpers
const errorPlain = computed(() => String((/** @type any */(props.errorText)) || '').trim())
const hasError = computed(() => errorPlain.value.length > 0)

function statusText(s) {
  const key = String(s || '').toLowerCase()
  const map = {
    processing: t('download.processing') || 'Processing',
    translating: t('download.processing') || 'Processing',
    completed: 'Completed',
    done: 'Done',
    failed: t('download.failed') || 'Failed',
    cancelled: t('download.cancelled') || 'Cancelled',
    pending: t('download.pending') || 'Pending',
  }
  return map[key] || s || '-'
}

function syncStatusText(s) {
  const key = String(s || '').toLowerCase()
  const map = {
    translating: t('download.processing') || 'Processing',
    processing: t('download.processing') || 'Processing',
    done: t('subtitle.task_ended') || 'Task ended',
    failed: t('download.failed') || 'Failed',
    partial_failed: 'Partially Failed',
    pending: t('download.pending') || 'Pending',
    cancelled: t('download.cancelled') || 'Cancelled',
  }
  return map[key] || s || '-'
}

const statusChipClass = computed(() => {
  const s = String((/** @type any */(status))).toLowerCase()
  if (s === 'failed') return 'badge-error'
  if (s === 'completed' || s === 'done') return 'badge-success'
  if (s === 'cancelled') return 'badge-info'
  if (s === 'processing' || s === 'translating' || s === 'pending') return 'badge-primary'
  return 'badge-ghost'
})

const syncStatusChipClass = computed(() => {
  const s = String((/** @type any */(props.syncStatus))).toLowerCase()
  if (s === 'failed') return 'badge-error'
  if (s === 'partial_failed') return 'badge-warning'
  if (s === 'done' || s === 'completed') return 'badge-success'
  if (s === 'cancelled') return 'badge-info'
  if (s === 'processing' || s === 'translating' || s === 'pending') return 'badge-primary'
  return 'badge-ghost'
})

const failedLabel = computed(() => t('subtitle.add_language.failed_segments') || 'Failed Segments')

function onProviderChange(e) {
  const v = e?.target?.value ?? ''
  emit('update:provider', v)
}
function onModelChange(e) {
  const v = e?.target?.value ?? ''
  emit('update:model', v)
}

async function copyFileName() {
  try { if (filePath) await copyToClipboard(filePath, t) } catch {}
}

function onStatusClick() {
  try { emit('open-llm-talk') } catch {}
}

function showTaskResultDetail() {
  try {
    const lines = []
    lines.push(`Result: ${syncStatusText((/** @type any */(props.syncStatus)))}`)
    lines.push(`Task: ${statusText((/** @type any */(status)))}`)
    lines.push(`Progress: ${Number((/** @type any */(processed))).toFixed?.(0) || processed}/${Number((/** @type any */(total))).toFixed?.(0) || total}`)
    if (requestCountNum.value) lines.push(`Requests: ${requestCountNum.value}, Tokens: ${tokenTotal.value} (P ${tokenPrompt.value}, C ${tokenCompletion.value})`)
    const err = String((/** @type any */(props.errorText)) || '').trim()
    if (err) { lines.push(''); lines.push('Error:'); lines.push(err) }
    const content = lines.join('\n')
    if (window?.$dialog?.alert) { window.$dialog.alert(content, { title: 'Task Result' }) }
    else { window.alert(content) }
  } catch (e) { try { window.alert('No detail available') } catch {}
  }
}

async function copyFailureReason() {
  try {
    const txt = String((/** @type any */(props.errorText)) || '').trim()
    if (txt) await copyToClipboard(txt, t)
  } catch {}
}
</script>

<style scoped>
.subtitle-detail-panel :deep(.tp-header) {
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  padding: 10px 8px 12px;
  border-bottom: 1px solid var(--macos-divider-weak);
}
.subtitle-detail-panel :deep(.tp-progress.inline) {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 8px;
  width: 100%;
}
.subtitle-detail-panel :deep(.tp-progress .bar) {
  width: 100%;
  min-width: 0;
  height: 6px;
  background: color-mix(in oklab, var(--macos-blue) 12%, transparent);
  border-radius: 999px;
  overflow: hidden;
}
.subtitle-detail-panel :deep(.tp-progress .fill) {
  display: block;
  height: 100%;
  background: rgb(var(--macos-blue-rgb));
}
.subtitle-detail-panel :deep(.pct) {
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
  white-space: nowrap;
}
.subtitle-detail-panel :deep(.lang-pill) { gap: 6px; }
.subtitle-detail-panel :deep(.tp-header .left) {
  display:flex;
  align-items:center;
  justify-content:flex-end;
  width:100%;
  gap: 12px;
}
.subtitle-detail-panel :deep(.clickable) { cursor: pointer; }
.subtitle-detail-panel :deep(.status-speed-chip) {
  margin-left: auto;
}

/* Failure reason panel (aligned with AnalysisModal */
.subtitle-detail-panel :deep(.error-panel) {
  border:1px solid rgba(255, 69, 58, 0.55);
  border-radius: 12px;
  background: color-mix(in oklab, var(--macos-surface) 88%, transparent);
  box-shadow: 0 6px 18px rgba(0,0,0,0.08);
  overflow: hidden;
  margin: 6px 0 12px;
}
.subtitle-detail-panel :deep(.error-header) {
  display:flex; align-items:center; justify-content: space-between;
  padding: 10px 12px;
  background: linear-gradient(180deg, rgba(255, 69, 58, 0.08), transparent);
  border-bottom: 1px solid rgba(255,255,255,0.12);
}
.subtitle-detail-panel :deep(.error-header .left) { display:flex; align-items:center; gap:6px; color: var(--macos-danger-text, #ff6b6b); font-weight: 600; font-size: 12px; }
.subtitle-detail-panel :deep(.error-header .title) { color: var(--macos-text-primary); margin-left: 2px; }
.subtitle-detail-panel :deep(.error-header .right) { display:flex; align-items:center; gap: 8px; }
.subtitle-detail-panel :deep(.error-body) { max-height: 160px; overflow: auto; }
.subtitle-detail-panel :deep(.error-text) {
  margin: 0; padding: 10px 12px; white-space: pre-wrap; word-break: break-word;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace;
  font-size: 12px; color: var(--macos-text-secondary);
}
.subtitle-detail-panel :deep(.actions-line) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.subtitle-detail-panel :deep(.actions-line .left-selects) {
  display: flex;
  align-items: center;
  gap: 8px;
}
.subtitle-detail-panel :deep(.stack-actions .stack-line) { margin: 4px 0; }
.subtitle-detail-panel :deep(.one-line) { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.subtitle-detail-panel :deep(.file-line) { display: inline-flex; align-items: center; gap: 8px; min-width: 0; max-width: 100%; }
.subtitle-detail-panel :deep(.file-line .name) { display: inline-block; flex: 1 1 auto; min-width: 0; max-width: 100%; }
</style>
