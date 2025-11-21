<template>
  <div class="stp-root">
    <div class="macos-group stp-header">
      <div class="grow"></div>
    </div>
    <div v-if="!visibleTasks.length" class="empty">{{ t('subtitle.tasks_empty') }}</div>

    <div v-else class="macos-box card-frosted card-translucent">
      <div
        v-for="task in visibleTasks"
        :key="task.id"
        class="macos-row st-row"
        :class="{ highlight: highlightTaskId && task.id === highlightTaskId }"
        @click="openDetail(task)"
      >
        <div class="row-grid">
          <!-- row 1 left: status + filename -->
          <div class="r1l">
            <span class="title" :title="task.project_name || '-'">{{ task.project_name || '-' }}</span>
            <span
              class="chip-frosted chip-lg status-chip"
              :class="statusBadge(task.status)"
              role="button"
              tabindex="0"
              @click.stop="openChat(task)"
              @keydown.enter.stop.prevent="openChat(task)"
              @keydown.space.stop.prevent="openChat(task)"
            >
              <span class="chip-label">{{ statusText(task.status) }}</span>
              <span class="chip-dot"></span>
            </span>
          </div>
          <!-- second row removed per design -->
          <!-- right column (spans 2 rows): delete icon centered -->
          <div class="rxy">
            <button
              class="btn-chip-icon btn-danger"
              :data-tooltip="t('common.delete')"
              :disabled="!canDelete(task)"
              @click.stop="confirmDelete(task)"
            >
              <Icon name="trash" class="w-4 h-4" />
            </button>
          </div>
        </div>
      </div>

      <!-- footer in container: refresh + total -->
      <div class="macos-row st-row st-footer" @click.stop>
        <div class="footer-wrap">
          <button class="btn-chip" :data-tooltip="t('common.refresh')" @click="refresh">
            <Icon name="refresh" class="w-4 h-4" />
          </button>
        </div>
      </div>
    </div>

    <!-- Removed modal: clicking task opens Subtitle page at target language -->

    <!-- bottom actions: refresh and stats -->
    <div class="p-2 flex justify-center">
      <div class="stats-pills">
        <span class="stats-pill">{{ totalCount }} {{ t('download.tasks') }}</span>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSubtitleTasksStore } from '@/stores/subtitleTasks'
import useNavStore from '@/stores/nav.js'
import { useSubtitleStore } from '@/stores/subtitle'
import eventBus from '@/utils/eventBus.js'

const props = defineProps({
  projectId: { type: String, default: '' },
  highlightTaskId: { type: String, default: '' },
})

const { t } = useI18n()
const st = useSubtitleTasksStore()
const allTasks = computed(() => st.tasks)
const visibleTasks = computed(() => {
  const list = allTasks.value || []
  if (!props.projectId) return list
  return list.filter(t => String(t?.project_id || '') === props.projectId)
})

const navStore = useNavStore()
const subtitleStore = useSubtitleStore()

function statusText(s) {
  const m = { processing: t('download.processing') || 'Processing', completed: t('download.completed') || 'Completed', failed: t('download.failed') || 'Failed', cancelled: t('download.cancelled') || 'Cancelled', pending: t('download.pending') || 'Pending' }
  return m[s] || String(s)
}
function statusBadge(s) {
  return {
    'badge-ok': s === 'completed',
    'badge-error': s === 'failed',
    'badge-running': s === 'processing',
    'badge-pending': s === 'pending' || s === 'cancelled',
  }
}
function openDetail(task) {
  try {
    const pid = task?.project_id
    const lang = task?.target_lang
    if (!pid) return
    // switch to Subtitle page and open project
    navStore.currentNav = navStore.navOptions.SUBTITLE
    try { subtitleStore.setPendingOpenProjectId(pid) } catch {}
    eventBus.emit('subtitle:open-project', pid)
    // switch language (Subtitle page watches store.currentLanguage)
    if (lang) try { subtitleStore.currentLanguage = lang } catch {}
  } catch (e) { console.error('open task to subtitle failed:', e) }
}
function openChat(task) {
  try {
    openDetail(task)
    eventBus.emit('subtitle:open-chat', {
      projectId: task?.project_id,
      targetLang: task?.target_lang,
      taskId: task?.id,
    })
  } catch (e) { console.error('open chat modal failed:', e) }
}
async function refresh() { await st.loadAll() }
function canDelete(t) { return ['failed','completed','cancelled','pending'].includes(String(t?.status || '')) }
async function confirmDelete(task) {
  if (!canDelete(task)) return
  const name = task?.project_name || ''
  const content = t('common.delete_confirm_detail', { title: name })
  const doDelete = async () => {
    const ok = await st.deleteTask(task?.id)
    if (!ok) $message?.error?.(t('common.delete_failed') || 'Delete failed')
  }
  // Align confirm usage with GlossaryPanel: use callbacks, not Promise
  const confirmed = window?.$dialog?.confirm
    ? await new Promise((resolve) => {
        window.$dialog.confirm(content, {
          title: t('common.delete_confirm'),
          positiveText: t('common.delete'),
          negativeText: t('common.cancel'),
          onPositiveClick: () => resolve(true),
          onNegativeClick: () => resolve(false),
        })
      })
    : window.confirm(content)
  if (confirmed) await doDelete()
}

onMounted(() => {
  st.init()
  // 页面切回/窗口获得焦点时主动刷新一次任务列表
  try {
    document.addEventListener('visibilitychange', onVisibilityChange)
    window.addEventListener('focus', onWindowFocus)
  } catch {}
})
onUnmounted(() => {
  try {
    document.removeEventListener('visibilitychange', onVisibilityChange)
    window.removeEventListener('focus', onWindowFocus)
  } catch {}
})

function onVisibilityChange() {
  try { if (document.visibilityState === 'visible') st.loadAll() } catch {}
}
function onWindowFocus() { st.loadAll() }

// stats
const totalCount = computed(() => visibleTasks.value.length)
</script>

<style scoped>
.stp-root { padding: 8px; font-size: var(--fs-base); }
.stp-header { display:flex; align-items:center; gap:8px; }
.stp-header + .macos-box { margin-top: 8px; }
.empty { font-size: 12px; color: var(--macos-text-secondary); padding: 8px; }
.st-row { cursor: pointer; }
.st-row:hover { background: color-mix(in oklab, var(--macos-blue) 8%, transparent); }
.row-grid { display: grid; grid-template-columns: 1fr auto; grid-template-rows: auto; align-items: center; gap: 4px 10px; width: 100%; }
.st-row :deep(.row-grid) { grid-column: 1 / -1; }
.r1l { grid-column: 1 / 2; grid-row: 1 / 2; display: flex; align-items: center; gap: 8px; min-width: 0; }
.rxy { grid-column: 2 / 3; grid-row: 1 / 2; display: flex; align-items: center; justify-content: center; }
.title { font-size: 13px; color: var(--macos-text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
/* second row pill removed */
.footer-wrap { width: 100%; display:flex; align-items:center; justify-content: center; gap: 12px; padding: 6px 0; }
.st-footer .footer-wrap { grid-column: 1 / -1; }
.meta-group { display:flex; align-items:center; gap: 8px; font-size: 12px; color: var(--macos-text-secondary); }
.meta-group .num { font-weight: 600; color: var(--macos-text-primary); }
.badge-ok { border-color: rgba(48, 209, 88, 0.5); }
.badge-error { border-color: rgba(255, 69, 58, 0.5); }
.badge-running { border-color: rgba(255,255,255,0.18); }
.badge-pending { border-color: rgba(255,255,255,0.16); }

.meta-group { display:flex; align-items:center; gap: 10px; font-size: 12px; color: var(--macos-text-secondary); }
.meta-group .item { display: inline-flex; align-items: baseline; gap: 4px; }
.meta-group .item .num { font-weight: 600; color: var(--macos-text-primary); line-height: 1; }
.meta-group .item .label { line-height: 1; }
.divider-v { width:1px; height: 16px; background: var(--macos-divider-weak); opacity: 0.8; }
.refresh-wrap { display:flex; align-items:center; justify-content:center; }
.highlight { outline: 2px solid color-mix(in oklab, var(--macos-blue) 50%, transparent); border-radius: 10px; }
.status-chip { cursor: pointer; }

/* Inspector bottom stats: align style with subtitle editor bottom pill */
.stats-pills { display:flex; align-items:center; gap: 8px; }
.stats-pill { display:inline-flex; align-items:center; height: 22px; padding: 0 10px; border-radius: 999px;
  border: 1px solid rgba(255,255,255,0.22);
  background: color-mix(in oklab, var(--macos-surface) 78%, transparent);
  color: var(--macos-text-secondary); font-size: var(--fs-sub);
  -webkit-backdrop-filter: var(--macos-surface-blur); backdrop-filter: var(--macos-surface-blur);
  box-shadow: var(--macos-shadow-1);
}
</style>
