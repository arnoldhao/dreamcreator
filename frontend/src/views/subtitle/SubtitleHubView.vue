<template>
  <div class="subtitle-hub-view">
    <div v-if="!processedItems.length" class="hub-empty dc-empty-shell">
      <div v-if="isFormatAll" class="macos-card card-frosted card-translucent dc-empty-card" @click.stop>
        <div class="dc-icon-wrap">
          <div class="dc-icon-bg">
            <Icon name="file-text" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
          </div>
        </div>
        <div class="dc-empty-title">{{ $t('subtitle.history.no_historical_records') }}</div>
        <div class="dc-empty-subtitle">{{ $t('subtitle.history.no_imported_sub_found') }}</div>
        <div class="dc-empty-actions">
          <button class="btn-chip-ghost btn-primary btn-sm" @click.stop="$emit('open-file')">
            <Icon name="plus" class="w-4 h-4 mr-1" />
            {{ $t('subtitle.common.open_file') }}
          </button>
          <button class="btn-chip-ghost btn-sm" @click.stop="$emit('refresh')">
            <Icon name="refresh" class="w-4 h-4 mr-1" />
            {{ $t('common.refresh') }}
          </button>
        </div>
      </div>
      <div v-else class="macos-card card-frosted card-translucent dc-empty-card" @click.stop>
        <div class="dc-icon-wrap">
          <div class="dc-icon-bg">
            <Icon name="filter-x" class="w-8 h-8 text-[var(--macos-text-secondary)]" />
          </div>
        </div>
        <div class="dc-empty-title">{{ $t('download.no_filter_results') }}</div>
        <div class="dc-empty-subtitle">{{ currentFormatLabel }}</div>
        <div class="dc-empty-actions">
          <button class="btn-chip-ghost btn-primary btn-sm" @click.stop="resetFilters">
            <Icon name="refresh" class="w-4 h-4 mr-1" />
            {{ $t('common.reset') }}
          </button>
          <button class="btn-chip-ghost btn-sm" @click.stop="$emit('refresh')">
            <Icon name="refresh" class="w-4 h-4 mr-1" />
            {{ $t('common.refresh') }}
          </button>
        </div>
      </div>
    </div>

    <div v-else class="macos-card card-frosted card-translucent hub" @click.stop>
      <div class="hub-body">
        <SubtitleHub
          :items="visibleItems"
          :inspector-visible="inspectorVisible"
          @open-project="$emit('open-project', $event)"
          @delete-project="$emit('delete-project', $event)"
        />
        <div ref="listEnd" class="io-sentinel"></div>
      </div>
    </div>

    <div class="floating-filter chip-frosted chip-translucent chip-panel" @click.stop :style="{ right: fabRight + 'px' }">
      <button class="icon-chip-ghost" :data-tooltip="$t('download.refresh')" data-tip-pos="top" @click="$emit('refresh')">
        <Icon name="refresh" class="w-4 h-4" :class="{ spinning: refreshing }" />
      </button>
      <div class="divider-v"></div>
      <div class="filter-toggle" @click="floatingFilterExpanded = !floatingFilterExpanded">
        <Icon name="filter" class="w-4 h-4" />
        <span class="chip-frosted chip-sm chip-translucent count-pill"><span class="chip-label">{{ totalCount }}/{{ totalAll }}</span></span>
      </div>
      <select v-if="floatingFilterExpanded" v-model="formatFilter" class="input-macos select-macos select-macos-xs filter-select">
        <option value="all">All</option>
        <option v-for="opt in formatOptions" :key="opt" :value="opt">{{ opt.toUpperCase() }}</option>
      </select>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import SubtitleHub from '@/components/subtitle/SubtitleHub.vue'
import eventBus from '@/utils/eventBus.js'

const props = defineProps({
  projects: { type: Array, default: () => [] },
  refreshing: { type: Boolean, default: false },
  inspectorVisible: { type: Boolean, default: false },
  fabRight: { type: Number, default: 12 },
})

defineEmits(['open-file', 'refresh', 'open-project', 'delete-project'])

const { t } = useI18n()

const query = ref('')
const formatFilter = ref('all')
const floatingFilterExpanded = ref(false)
const displayCount = ref(80)
const listEnd = ref(null)
let io = null

const filteredProjects = computed(() => {
  const q = (query.value || '').toLowerCase()
  const f = (formatFilter.value || 'all').toLowerCase()
  return (props.projects || []).filter(p => {
    const nameHit = !q || (p.project_name || '').toLowerCase().includes(q)
    const ext = (p.metadata?.source_info?.file_ext || '').toLowerCase()
    const formatHit = f === 'all' || ext === f
    return nameHit && formatHit
  })
})

const sortedProjects = computed(() => {
  const arr = [...filteredProjects.value]
  arr.sort((a, b) => (b.updated_at || 0) - (a.updated_at || 0))
  return arr
})

const processedItems = computed(() => {
  const items = []
  let lastGroup = null
  for (const p of sortedProjects.value) {
    const g = groupLabel(p.updated_at)
    if (g !== lastGroup) {
      items.push({ type: 'header', key: 'g:' + g + ':' + (items.length), label: g })
      lastGroup = g
    }
    items.push({ type: 'item', project: p })
  }
  return items
})

const visibleItems = computed(() => processedItems.value.slice(0, displayCount.value))
const totalCount = computed(() => sortedProjects.value.length)
const totalAll = computed(() => (props.projects || []).length)

const formatOptions = computed(() => {
  const s = new Set()
  for (const p of props.projects || []) {
    const e = (p.metadata?.source_info?.file_ext || '').toLowerCase()
    if (e) s.add(e)
  }
  return Array.from(s).sort()
})

const isFormatAll = computed(() => (formatFilter.value || 'all') === 'all')
const currentFormatLabel = computed(() => {
  if (isFormatAll.value) return ''
  const f = (formatFilter.value || '').toUpperCase()
  return f
})

function groupLabel(ts) {
  if (!ts) return t('subtitle.group.earlier') || 'Earlier'
  const d = new Date(ts * 1000)
  const now = new Date()
  const startOfToday = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime()
  const dayMs = 24 * 60 * 60 * 1000
  if (d.getTime() >= startOfToday) return t('subtitle.group.today') || 'Today'
  if ((startOfToday - d.getTime()) < 6 * dayMs) return t('subtitle.group.this_week') || 'This Week'
  return t('subtitle.group.earlier') || 'Earlier'
}

function resetFilters() {
  formatFilter.value = 'all'
  query.value = ''
}

function handleSearch(q) {
  query.value = q || ''
}

function setupListObserver() {
  try {
    cleanupListObserver()
    io = new IntersectionObserver((entries) => {
      if (entries.some(e => e.isIntersecting)) {
        displayCount.value = Math.min(displayCount.value + 80, processedItems.value.length + 80)
      }
    })
    setTimeout(() => { if (listEnd.value) io.observe(listEnd.value) }, 0)
  } catch {}
}

function cleanupListObserver() {
  try { io && io.disconnect() } catch {}
  io = null
}

watch(() => props.projects, () => {
  displayCount.value = 80
  setupListObserver()
})

watch([formatFilter, query], () => {
  displayCount.value = 80
})

onMounted(() => {
  setupListObserver()
  eventBus.on('subtitle:search', handleSearch)
})

onUnmounted(() => {
  cleanupListObserver()
  eventBus.off('subtitle:search', handleSearch)
})
</script>

<style scoped>
.subtitle-hub-view {
  position: relative;
  min-height: 72vh;
  padding: 12px 16px 24px;
}
.hub {
  padding: 10px;
  border: 1px solid rgba(255, 255, 255, 0.18);
}
</style>
