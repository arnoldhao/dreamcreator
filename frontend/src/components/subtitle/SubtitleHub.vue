<template>
  <div class="list-wrap comfortable">
    <template v-for="(it, idx) in items" :key="it.key || (it.project?.id + ':' + idx)">
      <div v-if="it.type === 'header'" class="list-header">{{ it.label }}</div>
      <div v-else class="list-row" @click.stop="onOpenProject(it.project)">
        <div class="col-icon">
          <div class="tile-icon" :class="extClass(it.project)">
            <Icon :name="extIcon(it.project)" class="w-4 h-4" />
          </div>
        </div>
        <div class="col-title" :title="it.project.project_name">
          <template v-if="editingId === it.project.id">
            <input
              class="rename-input"
              v-model="editingName"
              @keydown.enter.stop.prevent="onConfirmRename(it.project)"
              @keydown.esc.stop.prevent="onCancelRename"
              @click.stop
            />
            <button
              class="btn-chip-icon"
              :data-tooltip="$t('common.confirm')"
              data-tip-pos="top"
              @click.stop="onConfirmRename(it.project)"
            >
              <Icon name="status-success" class="w-3.5 h-3.5" />
            </button>
            <button
              class="btn-chip-icon"
              :data-tooltip="$t('common.cancel')"
              data-tip-pos="top"
              @click.stop="onCancelRename"
            >
              <Icon name="close" class="w-3.5 h-3.5" />
            </button>
          </template>
          <template v-else>
            <span class="name one-line">{{ it.project.project_name || '-' }}</span>
            <button
              v-if="!inspectorVisible"
              class="btn-chip-icon rename-btn"
              :data-tooltip="$t('common.edit')"
              data-tip-pos="top"
              @click.stop="onBeginRename(it.project)"
            >
              <Icon name="edit" class="w-3.5 h-3.5" />
            </button>
          </template>
        </div>
        <div v-if="!inspectorVisible" class="col-pills">
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
        <div v-if="!inspectorVisible" class="col-time">
          <div class="t-rel">{{ formatRelative(it.project.updated_at) }}</div>
          <div class="t-abs">{{ formatDate(it.project.updated_at) }}</div>
        </div>
        <div class="col-actions">
          <button
            class="btn-chip-icon"
            :data-tooltip="$t('common.delete')"
            data-tip-pos="top"
            @click.stop="onDeleteProject(it.project)"
          >
            <Icon name="trash" class="w-4 h-4" />
          </button>
        </div>
      </div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { UpdateProjectName } from 'bindings/dreamcreator/backend/api/subtitlesapi'
import { useSubtitleStore } from '@/stores/subtitle'
import { useSubtitleTasksStore } from '@/stores/subtitleTasks'

const props = defineProps({
  items: { type: Array, default: () => [] },
  inspectorVisible: { type: Boolean, default: false },
})

const emit = defineEmits(['open-project', 'delete-project'])

const { t } = useI18n()
const subtitleStore = useSubtitleStore()
const tasksStore = useSubtitleTasksStore()

const editingId = ref(null)
const editingName = ref('')

function onOpenProject(project) {
  emit('open-project', project)
}

async function onDeleteProject(project) {
  emit('delete-project', project)
}

function onBeginRename(project) {
  editingId.value = project?.id
  editingName.value = project?.project_name || ''
}
function onCancelRename() {
  editingId.value = null
  editingName.value = ''
}

async function onConfirmRename(project) {
  const name = (editingName.value || '').trim()
  if (!name) {
    $message?.warning?.(t('subtitle.common.project_name') + ' ' + (t('common.not_set') || ''))
    return
  }
  try {
    const r = await UpdateProjectName(project.id, name)
    if (!r?.success) throw new Error(r?.msg)
    try { project.project_name = name } catch {}
    onCancelRename()
    await subtitleStore.fetchProjects()
    try { await tasksStore.loadAll() } catch {}
  } catch (e) {
    $message?.error?.(e.message || String(e))
  }
}

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
</script>
