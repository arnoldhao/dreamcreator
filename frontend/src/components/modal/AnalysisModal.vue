<template>
  <teleport to="body">
    <div v-if="show" class="macos-modal" role="dialog" aria-modal="true" :aria-label="t('download.analysis.title')">
      <div class="modal-card card-frosted card-translucent" tabindex="-1">
      <div class="modal-header">
        <ModalTrafficLights @close="close" />
        <div class="title-area">
          <div class="title-text">{{ t('download.analysis.title') }}</div>
        </div>
      </div>
      <div class="modal-body">
        <div class="groups">
          <!-- 失败原因：固定高度 + 滚动 + 复制（小图标按钮） -->
          <div class="error-panel" v-if="failureReasonText">
            <div class="error-header">
              <div class="left">
                <Icon name="alert-triangle" class="w-4 h-4 mr-1" />
                <span class="title">{{ t('download.analysis.section_error') }}</span>
              </div>
              <div class="right">
                <button class="btn-chip-icon" :data-tooltip="t('common.copy')" @click="copyError">
                  <Icon name="file-copy" class="w-4 h-4"/>
                </button>
              </div>
            </div>
            <div class="error-body">
              <pre class="error-text">{{ failureReasonText }}</pre>
            </div>
          </div>

          <!-- URL 检查（地铁节点 + 结果行） -->
          <div class="step category" :class="cls(urlState)">
            <div class="dot"></div>
            <div class="content">
              <div class="left">
                <div class="label-row">
                  <div class="label">{{ t('download.analysis.section_url') }}</div>
                  <div class="track">
                    <div class="node" :class="cls(steps.host.state)" :title="t('download.analysis.extract_host')">
                      <div class="mini-dot"></div>
                      <div class="node-text">{{ t('download.analysis.node_domain') }}</div>
                    </div>
                    <div class="node" :class="cls(steps.connectivity.state)" :title="t('download.analysis.connectivity_check', { host: steps.host.meta || '-' })">
                      <div class="mini-dot"></div>
                      <div class="node-text">{{ t('download.analysis.node_connectivity') }}</div>
                    </div>
                  </div>
                </div>
                <div class="note" v-if="urlCurrentText" :class="urlCurrentState">{{ urlCurrentText }}</div>
              </div>
              <div class="right" v-if="urlHasFailure">
                <button class="btn-chip btn-sm" @click="goNetworkSettings"><Icon name="settings" class="w-4 h-4 mr-2"/>{{ t('download.analysis.network_short') }}</button>
              </div>
            </div>
          </div>

          <!-- yt-dlp 检查（地铁节点 + 结果行） -->
          <div class="step category" :class="cls(ytdlpState)">
            <div class="dot"></div>
            <div class="content">
              <div class="left">
                <div class="label-row">
                  <div class="label">{{ t('download.analysis.section_ytdlp') }}</div>
                  <div class="track">
                    <div class="node" :class="cls(steps.ytdlpPresence.state)" title="yt-dlp">
                      <div class="mini-dot"></div>
                      <div class="node-text">{{ t('download.analysis.node_available') }}</div>
                    </div>
                    <div class="node" :class="cls(steps.ytdlpVersion.state)" :title="t('settings.dependency.version')">
                      <div class="mini-dot"></div>
                      <div class="node-text">{{ t('download.analysis.node_version') }}</div>
                    </div>
                  </div>
                </div>
                <div class="note" v-if="ytdlpCurrentText" :class="ytdlpCurrentState">{{ ytdlpCurrentText }}</div>
              </div>
              <div class="right" v-if="ytdlpHasFailure">
                <button class="btn-chip btn-sm" @click="goDependencySettings"><Icon name="wrench" class="w-4 h-4 mr-2"/>{{ t('download.analysis.dependency_short') }}</button>
              </div>
            </div>
          </div>
        </div>
      </div>
        <div class="modal-footer">
          <div class="left-actions">
            <button class="btn-chip btn-sm" @click="retryTask"><Icon name="refresh" class="w-4 h-4 mr-1"/>{{ t('download.retry') }}</button>
          </div>
          <div class="right-actions">
            <button class="btn-chip btn-sm" @click="close">{{ t('common.close') }}</button>
            <button class="btn-chip btn-primary btn-sm" @click="start" :disabled="started">{{ started ? (t('download.processing') || 'Processing...') : (t('download.analyze') || 'Analyze') }}</button>
          </div>
        </div>
      </div>
    </div>
  </teleport>
  
</template>

<script setup>
import { ref, reactive, onMounted, onUnmounted, watch, computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDtStore } from '@/handlers/downtasks'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import Icon from '@/components/base/Icon.vue'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'
import useNavStore from '@/stores/nav.js'
import useSettingsStore from '@/stores/settings.js'

const props = defineProps({
  show: { type: Boolean, default: false },
  taskId: { type: String, required: true },
  taskError: { type: String, default: '' },
})
const emit = defineEmits(['update:show'])
const { t } = useI18n()
const dt = useDtStore()
const navStore = useNavStore()
const settingsStore = useSettingsStore()

const steps = reactive({
  host: { state: 'pending', meta: '' },
  connectivity: { state: 'pending', status: 0, error: '' },
  ytdlpPresence: { state: 'pending', meta: '' },
  ytdlpVersion: { state: 'pending', meta: '' },
})
const started = ref(false)
const errorText = ref('')

const urlHasFailure = computed(() => steps.host.state === 'fail' || steps.connectivity.state === 'fail')
const ytdlpHasFailure = computed(() => steps.ytdlpPresence.state === 'fail' || steps.ytdlpVersion.state === 'fail')

const urlAnyRunning = computed(() => steps.host.state === 'running' || steps.connectivity.state === 'running')
const urlAllOk = computed(() => steps.host.state === 'ok' && steps.connectivity.state === 'ok')
const ytdlpAnyRunning = computed(() => steps.ytdlpPresence.state === 'running' || steps.ytdlpVersion.state === 'running')
const ytdlpAllOk = computed(() => steps.ytdlpPresence.state === 'ok' && steps.ytdlpVersion.state === 'ok')

const urlState = computed(() => urlHasFailure.value ? 'fail' : (urlAnyRunning.value ? 'running' : (urlAllOk.value ? 'ok' : 'pending')))
const ytdlpState = computed(() => ytdlpHasFailure.value ? 'fail' : (ytdlpAnyRunning.value ? 'running' : (ytdlpAllOk.value ? 'ok' : 'pending')))

// 二行结果：只显示“当前检查项”的文字
const urlCurrentState = computed(() => {
  if (steps.connectivity.state !== 'pending') return steps.connectivity.state
  if (steps.host.state !== 'pending') return steps.host.state
  return ''
})
const urlCurrentText = computed(() => {
  // 优先展示连通性，否则展示域名提取；都未开始则不展示
  if (steps.connectivity.state !== 'pending') {
    if (steps.connectivity.state === 'running') return t('download.analysis.connectivity_unknown')
    if (steps.connectivity.state === 'ok') return t('download.analysis.connectivity_ok', { status: steps.connectivity.status || 200 })
    if (steps.connectivity.state === 'fail') return t('download.analysis.connectivity_fail', { error: steps.connectivity.error || 'N/A' })
  }
  if (steps.host.state !== 'pending') {
    if (steps.host.state === 'running') return t('download.processing') || 'Processing...'
    if (steps.host.state === 'ok') return steps.host.meta || '-'
    if (steps.host.state === 'fail') return steps.host.error || (t('common.invalid_url') || 'Invalid URL')
  }
  return ''
})

const ytdlpCurrentState = computed(() => {
  if (steps.ytdlpVersion.state !== 'pending') return steps.ytdlpVersion.state
  if (steps.ytdlpPresence.state !== 'pending') return steps.ytdlpPresence.state
  return ''
})
const ytdlpCurrentText = computed(() => {
  // 优先展示版本结果，否则展示存在性；都未开始则不展示
  if (steps.ytdlpVersion.state !== 'pending') {
    if (steps.ytdlpVersion.state === 'running') return t('download.processing') || 'Processing...'
    if (steps.ytdlpVersion.state === 'ok') return t('download.analysis.ytdlp_ok', { version: steps.ytdlpPresence.meta || '-' })
    if (steps.ytdlpVersion.state === 'fail') return t('download.analysis.ytdlp_outdated', { version: steps.ytdlpPresence.meta || '-', latest: steps.ytdlpVersion.meta || '-' })
  }
  if (steps.ytdlpPresence.state !== 'pending') {
    if (steps.ytdlpPresence.state === 'running') return t('download.processing') || 'Processing...'
    if (steps.ytdlpPresence.state === 'ok') return t('download.analysis.ytdlp_ok', { version: steps.ytdlpPresence.meta || '-' })
    if (steps.ytdlpPresence.state === 'fail') return t('download.analysis.ytdlp_missing')
  }
  return ''
})

// 失败原因：优先父级传入的 taskError，其次本地加载/WS 的 errorText
const failureReasonText = computed(() => {
  const pe = (props.taskError || '').trim()
  if (pe) return pe
  return errorText.value || ''
})

function cls(state) {
  return {
    running: state === 'running',
    ok: state === 'ok',
    fail: state === 'fail'
  }
}

function close() { emit('update:show', false) }

function resetSteps() {
  steps.host = Object.assign(steps.host, { state: 'pending', meta: '' })
  steps.connectivity = Object.assign(steps.connectivity, { state: 'pending', status: 0, error: '' })
  steps.ytdlpPresence = Object.assign(steps.ytdlpPresence, { state: 'pending', meta: '' })
  steps.ytdlpVersion = Object.assign(steps.ytdlpVersion, { state: 'pending', meta: '' })
}

function onAnalysisEvent(data) {
  if (!data || data.id !== props.taskId) return
  const step = String(data.step || '')
  const action = String(data.action || '')
  if (step === 'extract_host') {
    if (action === 'start') steps.host.state = 'running'
    else if (action === 'ok') { steps.host.state = 'ok'; steps.host.meta = data.message || '' }
    else if (action === 'fail') { steps.host.state = 'fail'; steps.host.error = data.error || '' }
  } else if (step === 'connectivity') {
    if (action === 'start') steps.connectivity.state = 'running'
    else if (action === 'ok') { steps.connectivity.state = 'ok'; steps.connectivity.status = data.status || 200 }
    else if (action === 'fail') { steps.connectivity.state = 'fail'; steps.connectivity.error = data.error || '' }
  } else if (step === 'ytdlp_presence') {
    if (action === 'start') steps.ytdlpPresence.state = 'running'
    else if (action === 'ok') { steps.ytdlpPresence.state = 'ok'; steps.ytdlpPresence.meta = data.message || '' }
    else if (action === 'fail') { steps.ytdlpPresence.state = 'fail' }
  } else if (step === 'ytdlp_version') {
    if (action === 'start') steps.ytdlpVersion.state = 'running'
    else if (action === 'ok') { steps.ytdlpVersion.state = 'ok'; steps.ytdlpVersion.meta = data.message || '' }
    else if (action === 'fail') { steps.ytdlpVersion.state = 'fail'; steps.ytdlpVersion.meta = data.message || '' }
  } else if (step === 'complete') {
    started.value = false
  }
}

async function start() {
  try {
    resetSteps()
    started.value = true
    const api = window?.go?.api?.DowntasksAPI
    if (api && typeof api.StartAnalysis === 'function') {
      await api.StartAnalysis(props.taskId)
    }
  } catch {}
}

// removed copyReport (no longer used)

async function copyError() { if (failureReasonText.value) await copyToClipboard(failureReasonText.value, t) }

async function retryTask() {
  try {
    const api = window?.go?.api?.DowntasksAPI
    if (api && typeof api.RetryTask === 'function') {
      const r = await api.RetryTask(props.taskId)
      if (r?.success) {
        $message?.success?.(t('download.retried'))
        close()
      } else {
        throw new Error(r?.msg || 'retry failed')
      }
    }
  } catch (e) {
    $message?.error?.(t('download.retry_failed'))
  }
}

function goNetworkSettings() {
  try {
    navStore.setNav(navStore.navOptions.SETTINGS)
    settingsStore.setPage(settingsStore.settingsOptions.GENERAL)
    close()
  } catch {}
}

function goDependencySettings() {
  try {
    navStore.setNav(navStore.navOptions.SETTINGS)
    settingsStore.setPage(settingsStore.settingsOptions.DEPENDENCY)
    close()
  } catch {}
}

onMounted(() => {
  dt.registerAnalysisCallback(onAnalysisEvent)
  // 同步监听进度中的错误（DtProgress.error），保证任务错误能即时显示
  dt.registerProgressCallback(onProgressEvent)
  try { document?.body?.classList?.add('modal-open') } catch {}
})
onUnmounted(() => { 
  dt.unregisterAnalysisCallback(onAnalysisEvent)
  dt.unregisterProgressCallback(onProgressEvent)
  try { document?.body?.classList?.remove('modal-open') } catch {}
})

function onProgressEvent(p) {
  try {
    if (!p || p.id !== props.taskId) return
    // 仅当后端提供非空错误时才更新，避免把已展示的错误清空
    if (typeof p.error === 'string' && p.error.trim() !== '') {
      errorText.value = p.error
    }
  } catch {}
}

async function loadTaskError() {
  try {
    const api = window?.go?.api?.DowntasksAPI
    if (api && typeof api.ListTasks === 'function') {
      const r = await api.ListTasks()
      if (r?.success) {
        const arr = JSON.parse(r.data || '[]')
        const tsk = Array.isArray(arr) ? arr.find(x => x && x.id === props.taskId) : null
        const e = tsk && typeof tsk.error === 'string' ? tsk.error.trim() : ''
        if (e) errorText.value = e
      }
    }
  } catch { /* 保留已有的 errorText */ }
}

watch(() => props.show, (v) => { 
  if (v) { 
    started.value = false; 
    resetSteps(); 
    // 优先使用外部传入的错误（如卡片的 task.error）
    errorText.value = props.taskError || ''
    loadTaskError() 
    try { document?.body?.classList?.add('modal-open') } catch {}
  } else {
    try { document?.body?.classList?.remove('modal-open') } catch {}
  }
})

watch(() => props.taskError, (v) => { if (props.show && v) errorText.value = v })

</script>

<style scoped>
.modal-card { width: min(560px, 96vw); max-height: 82vh; display:flex; flex-direction: column; overflow: hidden; border-radius: 12px; }
.modal-header { padding: 10px 12px; display:flex; align-items:center; justify-content: space-between; border-bottom: 1px solid rgba(255,255,255,0.16); }
.title-area { padding-left: 6px; }
.title-text { font-weight: 600; font-size: 13px; color: var(--macos-text-primary); }
.modal-body { padding: 12px; }
.modal-footer { padding: 10px 12px; display:flex; align-items:center; justify-content: space-between; border-top: 1px solid rgba(255,255,255,0.16); }
.right-actions, .left-actions { display:flex; align-items:center; gap: 8px; }
.modal-card.card-frosted.card-translucent { 
  /* Always-on active look for analysis modal */
  background: color-mix(in oklab, var(--macos-surface) 76%, transparent);
  border-color: rgba(255,255,255,0.28);
  box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0,0,0,0.24);
}
.groups { display:flex; flex-direction: column; gap: 14px; }
.steps { display:flex; flex-direction: column; gap: 10px; }

/* 分类行（大类标题 + 地铁节点 + 结果） */
.category .content { grid-template-columns: 1fr auto; }
.label-row { display:flex; align-items:center; gap: 10px; }
.track { display:flex; align-items:center; gap: 12px; opacity: 0.9; }
.node { display:flex; align-items:center; gap: 6px; padding: 2px 6px; border-radius: 999px; background: color-mix(in oklab, var(--macos-surface) 74%, transparent); border: 1px solid rgba(255,255,255,0.16); }
.node .mini-dot { width:6px; height:6px; border-radius:999px; background: rgba(255,255,255,0.7); box-shadow: 0 0 6px rgba(255,255,255,0.45); }
.node .node-text { font-size: 11px; color: var(--macos-text-secondary); line-height: 1; }
.node.ok .mini-dot { background:#fff; box-shadow: 0 0 8px rgba(48,209,88,0.7); }
.node.fail .mini-dot { background:#fff; box-shadow: 0 0 8px rgba(255,69,58,0.7); }
.node.running .mini-dot { background:#fff; box-shadow: 0 0 8px rgba(255,255,255,0.7); }

.note .sep { margin: 0 6px; opacity: 0.6; }
.note.fail, .note .fail { color: rgba(255, 99, 92, 0.95); }

/* Error panel: distinct look from step rows */
.error-panel { border:1px solid rgba(255, 69, 58, 0.55); border-radius: 12px; background: color-mix(in oklab, var(--macos-surface) 88%, transparent); box-shadow: 0 6px 18px rgba(0,0,0,0.08); overflow: hidden; }
.error-header { display:flex; align-items:center; justify-content: space-between; padding: 10px 12px; background: linear-gradient(180deg, rgba(255, 69, 58, 0.08), transparent); border-bottom: 1px solid rgba(255,255,255,0.12); }
.error-header .left { display:flex; align-items:center; color: var(--macos-danger-text, #ff6b6b); font-weight: 600; font-size: 12px; }
.error-header .title { color: var(--macos-text-primary); margin-left: 2px; }
.error-header .right { display:flex; align-items:center; gap: 8px; }
.error-body { position: relative; max-height: calc(2 * 1.4em + 20px); overflow: auto; }
.error-text { line-height: 1.4; }
.error-text { margin: 0; padding: 10px 12px; white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; font-size: 12px; color: var(--macos-text-secondary); }
.step { display:flex; align-items:flex-start; gap:10px; padding: 10px 12px; border-radius: 10px; border:1px solid rgba(255,255,255,0.20); background: color-mix(in oklab, var(--macos-surface) 82%, transparent); position: relative; overflow: hidden; }
.step .dot { width:8px; height:8px; border-radius:999px; background: rgba(255,255,255,0.70); box-shadow: 0 0 8px rgba(255,255,255,0.55); margin-top: 5px; }
.step .content { flex:1 1 auto; min-width:0; display:grid; grid-template-columns: 1fr auto; align-items:center; gap: 8px; }
.step .content .left { min-width:0; }
.step .content .right { display:flex; align-items:center; justify-content:flex-end; gap:6px; }
.step .label { font-weight: 600; font-size: 12px; color: var(--macos-text-primary); }
.step .note { font-size: 12px; color: var(--macos-text-secondary); margin-top: 4px; white-space: pre-wrap; word-break: break-word; }
.step .content .left .label, .step .content .left .note { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.step.ok { border-color: rgba(48, 209, 88, 0.5); }
.step.ok .dot { background: #fff; box-shadow: 0 0 10px rgba(48,209,88,0.7); }
.step.fail { border-color: rgba(255, 69, 58, 0.5); }
.step.fail .dot { background: #fff; box-shadow: 0 0 10px rgba(255,69,58,0.7); }

/* Tahoe 风格进行中高光：柔和白色扫光，而非蓝色渐变 */
.step.running::after { content:''; position:absolute; inset: 0; background: linear-gradient(110deg, transparent 0%, rgba(255,255,255,0.18) 45%, rgba(255,255,255,0.28) 50%, rgba(255,255,255,0.18) 55%, transparent 100%); filter: blur(0.5px); animation: sweep 1.25s linear infinite; }
@keyframes sweep { from { transform: translateX(-120%); } to { transform: translateX(120%); } }

/* 禁用遮罩点击：不在根容器使用 @click.self 关闭，确保只能点击右上角或底部关闭 */
</style>
