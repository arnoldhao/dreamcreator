<template>
  <teleport to="body">
    <div
      v-if="show"
      class="macos-modal"
      role="dialog"
      aria-modal="true"
      :aria-label="t('subtitle.chat.title') || 'LLM Conversation'"
    >
      <div class="modal-card card-frosted card-translucent" tabindex="-1">
        <div class="modal-header">
          <ModalTrafficLights @close="close" />
          <div class="title-area">
            <div class="title-text">
              {{ t('subtitle.chat.title') || 'LLM Translation Conversation' }}
            </div>
            <div class="subtitle-text">
              <span class="mono">{{ languageLabel || language }}</span>
              <span class="sep">•</span>
              <span class="mono">{{ headerProvider }}</span>
              <span v-if="headerModel" class="mono"> / {{ headerModel }}</span>
              <span class="sep">•</span>
              <span class="status-pill" :class="statusClass">{{ statusText }}</span>
            </div>
          </div>
        </div>
        <div class="modal-body">
          <div v-if="loadError" class="error-banner">
            <Icon name="alert-triangle" class="w-4 h-4 mr-1" />
            <span class="mono one-line">{{ loadError }}</span>
          </div>
          <div v-if="loading" class="loading-line">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="label">{{ t('download.processing') || 'Loading…' }}</span>
          </div>
          <div ref="scrollEl" class="chat-scroll">
            <div v-if="!loading && !messages.length && !loadError" class="empty-tip">
              {{ t('subtitle.chat.empty') || 'No LLM conversation yet for this translation.' }}
            </div>
            <ChatMessageList
              :messages="messages"
              :app-label="appLabel"
              :provider-label="providerLabel"
              :pending-request-id="pendingRequestId"
              :pending-wait-text="pendingWaitText"
              :pending-running-label="pendingRunningLabel"
              :pending-waiting-label="pendingWaitingLabel"
            />
          </div>
        </div>
        <div class="modal-footer">
          <div class="left-actions">
            <span v-if="messages.length" class="mono meta-pill">
              {{ t('subtitle.chat.messages') || 'Messages' }}: {{ messages.length }}
            </span>
            <span v-if="tokenSummary" class="mono meta-pill">
              {{ t('subtitle.chat.tokens') || 'Tokens' }}: {{ tokenSummary }}
            </span>
            <span v-if="tokenSpeedText" class="mono meta-pill">
              {{ t('subtitle.chat.token_speed') || 'Token speed' }}: {{ tokenSpeedText }}
            </span>
            <span v-if="elapsedText" class="mono meta-pill">
              {{ t('subtitle.chat.elapsed') || 'Elapsed' }}: {{ elapsedText }}
            </span>
          </div>
          <div class="right-actions">
            <button class="btn-chip btn-sm" @click="close">
              {{ t('common.close') || 'Close' }}
            </button>
          </div>
        </div>
      </div>
    </div>
  </teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDtStore } from '@/stores/downloadTasks'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import Icon from '@/components/base/Icon.vue'
import { parseChatBlocks } from '@/utils/chatMarkdown'
import ChatMessageList from '@/components/chat/ChatMessageList.vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  projectId: { type: String, required: true },
  language: { type: String, required: true },
  languageLabel: { type: String, default: '' },
  provider: { type: String, default: '' },
  model: { type: String, default: '' },
  promptTokens: { type: Number, default: 0 },
  completionTokens: { type: Number, default: 0 },
  totalTokens: { type: Number, default: 0 },
  startTime: { type: Number, default: 0 },
  endTime: { type: Number, default: 0 },
  isRunning: { type: Boolean, default: false },
  requestCount: { type: Number, default: 0 },
})

const emit = defineEmits(['update:show'])
const { t } = useI18n()
const dt = useDtStore()

const loading = ref(false)
const loadError = ref('')
const status = ref('running')
const conversationId = ref('')
const messages = ref([])
const scrollEl = ref(null)
// elapsed time state (per modal)
const nowSec = ref(Math.floor(Date.now() / 1000))
let elapsedTimer = null

// pending LLM response state
const pendingRequestId = ref('')
const pendingStartTs = ref(0)
const pendingNowSec = ref(Math.floor(Date.now() / 1000))
let pendingTimer = null

const conversationMeta = computed(() => {
  const first = messages.value[0]
  return first?.meta || {}
})

const headerProvider = computed(() => props.provider || (conversationMeta.value.provider || 'LLM'))
const headerModel = computed(() => props.model || conversationMeta.value.model || '')

const appLabel = computed(() => t('subtitle.chat.app') || 'App')
const providerLabel = computed(() => headerProvider.value || t('subtitle.chat.provider') || 'LLM')
const pendingRunningLabel = computed(() => t('subtitle.chat.status_running') || 'Running')
const pendingWaitingLabel = computed(() => t('subtitle.chat.waiting') || 'Waiting for LLM…')

const tokenSummary = computed(() => {
  const total = Number(props.totalTokens || 0)
  const prompt = Number(props.promptTokens || 0)
  const completion = Number(props.completionTokens || 0)
  if (!total && !prompt && !completion) return ''
  return `${total} (P ${prompt}, C ${completion})`
})

const tokenSpeedText = computed(() => {
  try {
    const total = Number(props.totalTokens || 0)
    if (!total) return ''
    const st = Number(props.startTime || 0)
    if (!st) return ''
    const et = Number(props.endTime || 0)
    const now = nowSec.value
    const end = et || now
    const durSec = Math.max(1, end - st)
    if (props.isRunning) {
      const speed = total / durSec
      return `${speed.toFixed(1)} tok/s`
    }
    const req = Number(props.requestCount || 0)
    if (req > 0) {
      const avg = total / req
      return `${avg.toFixed(0)} tok/req`
    }
    const avgSpeed = total / durSec
    return `${avgSpeed.toFixed(1)} tok/s`
  } catch {
    return ''
  }
})

const elapsedText = computed(() => {
  try {
    const st = Number(props.startTime || 0)
    const et = Number(props.endTime || 0)
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
  } catch {
    return ''
  }
})

const statusText = computed(() => {
  const s = String(status.value || '').toLowerCase()
  if (s === 'finished') return t('subtitle.chat.status_finished') || 'Finished'
  if (s === 'failed') return t('subtitle.chat.status_failed') || 'Failed'
  return t('subtitle.chat.status_running') || 'Running'
})

const statusClass = computed(() => {
  const s = String(status.value || '').toLowerCase()
  return {
    'badge-success': s === 'finished',
    'badge-error': s === 'failed',
    'badge-primary': s === 'running',
  }
})

const pendingWaitText = computed(() => {
  try {
    const start = Number(pendingStartTs.value || 0)
    if (!start) return ''
    const now = pendingNowSec.value
    const diff = Math.max(0, now - start)
    const m = Math.floor(diff / 60)
    const s = diff % 60
    if (m > 0) return `${m}m ${s}s`
    return `${s}s`
  } catch {
    return ''
  }
})

function startPending(msg) {
  try {
    pendingRequestId.value = msg.id
    pendingStartTs.value = Number(msg.ts || Math.floor(Date.now() / 1000))
    if (pendingTimer) {
      clearInterval(pendingTimer)
      pendingTimer = null
    }
    pendingTimer = setInterval(() => {
      pendingNowSec.value = Math.floor(Date.now() / 1000)
    }, 1000)
  } catch {}
}

function stopPending() {
  try {
    pendingRequestId.value = ''
    pendingStartTs.value = 0
    if (pendingTimer) {
      clearInterval(pendingTimer)
      pendingTimer = null
    }
  } catch {}
}

function close() {
  emit('update:show', false)
}

function toKindLabel(kind) {
  const k = String(kind || '').toLowerCase()
  if (k === 'request') return t('subtitle.chat.kind_request') || 'Request'
  if (k === 'response') return t('subtitle.chat.kind_response') || 'Response'
  if (k === 'error') return t('subtitle.chat.kind_error') || 'Error'
  return ''
}

function formatTime(ts) {
  try {
    if (!ts) return ''
    const d = new Date(Number(ts) * 1000)
    if (Number.isNaN(d.getTime())) return ''
    const p = n => String(n).padStart(2, '0')
    return `${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
  } catch {
    return ''
  }
}

function normalizeBlocks(text, role) {
  // 当前只读视图，统一按“chat markdown”规则解析：
  // - 整条纯 JSON -> 单 json 块
  // - 否则：Markdown AST 中切分出 markdown / code / json 多块
  return parseChatBlocks(text)
}

function normalizeMessage(raw) {
  const role = String(raw?.role || '').toLowerCase()
  const kind = String(raw?.kind || '').toLowerCase()
  const ts = Number(raw?.created_at || 0)
  const timeText = formatTime(ts)
  const rawContent = String(raw?.content || '')
  return {
    id: raw?.id || `${Date.now()}-${Math.random()}`,
    role: role === 'provider' ? 'provider' : 'app',
    kind,
    kindLabel: toKindLabel(kind),
    timeText,
    content: rawContent,
    meta: raw?.metadata || {},
    roleClass: role === 'provider' ? 'from-provider' : 'from-app',
    ts,
    blocks: normalizeBlocks(rawContent, role),
  }
}

function applyConversation(conv) {
  try {
    if (!conv) return
    conversationId.value = conv.id || conv.task_id || ''
    status.value = conv.status || 'running'
    const list = Array.isArray(conv.messages) ? conv.messages : []
    const merged = []
    for (const raw of list) {
      const nm = normalizeMessage(raw)
      const append = !!(nm.meta && (nm.meta.append || nm.meta.delta))
      if (append && merged.length) {
        const last = merged[merged.length - 1]
        if (last && last.role === nm.role) {
          last.content = String(last.content || '') + String(nm.content || '')
          if (nm.timeText) last.timeText = nm.timeText
          last.blocks = normalizeBlocks(last.content, last.role)
          continue
        }
      }
      merged.push(nm)
    }
    messages.value = merged
    // 初次加载历史时，不重建 pending 状态；仅在实时交互中展示等待提示
  } catch {}
}

async function loadConversation() {
  if (!props.projectId || !props.language) return
  loading.value = true
  loadError.value = ''
  try {
    const api = window?.go?.api?.SubtitlesAPI
    if (!api || typeof api.GetSubtitleLLMConversation !== 'function') {
      throw new Error('GetSubtitleLLMConversation not available')
    }
    const res = await api.GetSubtitleLLMConversation(props.projectId, props.language)
    if (!res?.success) {
      const rawMsg = String(res?.msg || '')
      const lower = rawMsg.toLowerCase()
      // 特殊情况：该语言从未执行过 llm_translate（可能来自导入或 zhconvert），
      // 后端会返回 "no llm_translate task for language: xx"。
      // 这里将其视为“无会话空态”，而不是错误。
      if (lower.includes('no llm_translate task for language')) {
        status.value = 'finished'
        loadError.value = ''
        messages.value = []
        return
      }
      throw new Error(rawMsg || 'Failed to load conversation')
    }
    const raw = res.data
    const conv = raw ? (typeof raw === 'string' ? JSON.parse(raw) : raw) : null
    applyConversation(conv)
  } catch (e) {
    loadError.value = e?.message || String(e || '')
  } finally {
    loading.value = false
    await nextTick()
    scrollToBottom()
  }
}

function scrollToBottom() {
  try {
    const el = scrollEl.value
    if (!el) return
    el.scrollTop = el.scrollHeight || 0
  } catch {}
}

function onTalkEvent(ev) {
  try {
    if (!ev || !props.projectId || !props.language) return
    if (ev.project_id && ev.project_id !== props.projectId) return
    if (ev.language && ev.language !== props.language) return
    if (ev.conversation_id && conversationId.value && ev.conversation_id !== conversationId.value) return
    if (ev.status) status.value = ev.status
    if (ev.message) {
      const nm = normalizeMessage(ev.message)
      const stage = String(nm.meta?.stage || '').toLowerCase()
      // 当流式 delta 已经实时展示时，后端在流结束后会再推送一次聚合后的完整回复（batch_json / batch_jsonl），
      // 这里将其视为“终态更新”，避免在当前会话中出现重复的 Provider 气泡。
      if (
        nm.role === 'provider' &&
        nm.kind === 'response' &&
        (stage === 'batch_json' || stage === 'batch_jsonl') &&
        messages.value.length
      ) {
        const last = messages.value[messages.value.length - 1]
        const lastStage = String(last?.meta?.stage || '').toLowerCase()
        if (
          last &&
          last.role === 'provider' &&
          (lastStage === 'batch_json_stream' || lastStage === 'batch_jsonl_stream')
        ) {
          // 合并元数据与时间戳，保留现有内容（流式已完整累积）
          last.meta = { ...(last.meta || {}), ...(nm.meta || {}), stage }
          if (nm.timeText) last.timeText = nm.timeText
          last.blocks = normalizeBlocks(last.content, last.role)
          // pending 状态：Provider 完成回复时结束等待
          stopPending()
          nextTick().then(scrollToBottom)
          return
        }
      }
      const append = !!(nm.meta && (nm.meta.append || nm.meta.delta))
      if (append && messages.value.length) {
        const last = messages.value[messages.value.length - 1]
        if (last && last.role === nm.role) {
          last.content = String(last.content || '') + String(nm.content || '')
          if (nm.timeText) last.timeText = nm.timeText
          last.blocks = normalizeBlocks(last.content, last.role)
        } else {
          messages.value.push(nm)
        }
      } else {
        messages.value.push(nm)
      }
      // pending 状态：App request -> start; Provider response/error -> stop
      if (nm.role === 'app' && nm.kind === 'request') {
        startPending(nm)
      } else if (nm.role === 'provider' && (nm.kind === 'response' || nm.kind === 'error')) {
        stopPending()
      }
      nextTick().then(scrollToBottom)
    }
  } catch {}
}

watch(
  () => props.show,
  (v) => {
    if (v) {
      messages.value = []
      conversationId.value = ''
      status.value = 'running'
      loadConversation()
      try { document?.body?.classList?.add('modal-open') } catch {}
    } else {
      try { document?.body?.classList?.remove('modal-open') } catch {}
    }
  }
)

onMounted(() => {
  dt.registerSubtitleChatCallback(onTalkEvent)
})

onUnmounted(() => {
  try { dt.unregisterSubtitleChatCallback(onTalkEvent) } catch {}
  try { if (elapsedTimer) clearInterval(elapsedTimer) } catch {}
})

watch(
  () => props.isRunning,
  (running) => {
    try { if (elapsedTimer) { clearInterval(elapsedTimer); elapsedTimer = null } } catch {}
    if (running) {
      elapsedTimer = setInterval(() => {
        nowSec.value = Math.floor(Date.now() / 1000)
      }, 1000)
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.modal-card {
  width: min(860px, 96vw);
  max-height: 82vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border-radius: 12px;
}
.modal-header {
  padding: 10px 12px;
  display: flex;
  align-items: center;
  justify-content: flex-start;
  gap: 8px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.16);
}
.title-area {
  padding-left: 6px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.title-text {
  font-weight: 600;
  font-size: 13px;
  color: var(--macos-text-primary);
}
.subtitle-text {
  font-size: 11px;
  color: var(--macos-text-secondary);
  display: flex;
  align-items: center;
  gap: 4px;
}
.subtitle-text .sep {
  opacity: 0.6;
}
.status-pill {
  padding: 2px 6px;
  border-radius: 999px;
  font-size: 11px;
}
.modal-body {
  padding: 12px;
}
.modal-footer {
  padding: 10px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-top: 1px solid rgba(255, 255, 255, 0.16);
}
.right-actions,
.left-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}
.modal-card.card-frosted.card-translucent {
  background: color-mix(in oklab, var(--macos-surface) 76%, transparent);
  border-color: rgba(255, 255, 255, 0.28);
  box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0, 0, 0, 0.24);
}
.chat-scroll {
  max-height: 58vh;
  overflow-y: auto;
  padding: 6px 6px 8px 6px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.error-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: 8px;
  margin-bottom: 8px;
  font-size: 11px;
  background: color-mix(in oklab, var(--macos-danger-bg, #ff3b30) 14%, var(--macos-surface) 86%);
  color: var(--macos-text-primary);
}
.loading-line {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  font-size: 11px;
  color: var(--macos-text-secondary);
}
.loading-line .dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.7);
  animation: blink 1.2s infinite ease-in-out;
}
.loading-line .dot:nth-child(2) {
  animation-delay: 0.2s;
}
.loading-line .dot:nth-child(3) {
  animation-delay: 0.4s;
}
.loading-line .label {
  margin-left: 4px;
}
.empty-tip {
  font-size: 12px;
  color: var(--macos-text-secondary);
  text-align: center;
  padding: 16px 4px;
}
.meta-pill {
  font-size: 11px;
  padding: 2px 6px;
  border-radius: 999px;
  background: color-mix(in oklab, var(--macos-surface) 84%, transparent);
  border: 1px solid rgba(255, 255, 255, 0.14);
  color: var(--macos-text-primary);
}

@keyframes blink {
  0%, 80%, 100% { opacity: 0.2; transform: translateY(0); }
  40% { opacity: 1; transform: translateY(-1px); }
}
</style>
