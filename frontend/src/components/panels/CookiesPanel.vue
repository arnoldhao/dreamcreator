<template>
  <div class="cookies-panel">
    <!-- top: faux search opens full modal -->
    <div class="toolbar p-3 pt-2 flex items-center gap-2">
      <button class="search-link flex-1" @click="showModal = true">
        <Icon name="search" class="w-4 h-4" />
        <span class="placeholder">{{ $t('cookies.search_placeholder') }}</span>
        <!-- <Icon name="open" class="w-4 h-4 ml-auto text-[var(--macos-text-tertiary)]" /> -->
      </button>
      <button class="icon-chip-ghost" :data-tooltip="$t('common.refresh')" @click="fetchCookies" :disabled="isLoading">
        <Icon name="refresh" class="w-4 h-4" :class="{ spinning: isLoading }" />
      </button>
    </div>
    
    <!-- Windows elevated/admin hint -->
    <div v-if="winAdminHint" class="mx-3 mb-2 p-2 rounded border text-xs"
         style="background: var(--macos-background); border-color: var(--macos-warning-border); color: var(--macos-warning-text);">
      <Icon name="status-warning" class="w-3.5 h-3.5 inline mr-1 align-[-2px]" />
      {{ winAdminHint }}
    </div>

    <!-- loading -->
    <div v-if="isLoading" class="p-6 text-center text-sm text-[var(--macos-text-secondary)]">
      <div class="w-6 h-6 border-2 border-[var(--macos-blue)] border-t-transparent rounded-full animate-spin mx-auto mb-2"></div>
      {{ $t('cookies.loading_cookies') }}
    </div>

    <!-- empty -->
    <div v-else-if="!visibleBrowserCollections.length" class="p-6 text-center text-sm text-[var(--macos-text-secondary)]">
      <div class="w-10 h-10 rounded-full bg-[var(--macos-background)] border border-[var(--macos-separator)] mx-auto mb-2 flex items-center justify-center">
        <Icon name="globe" class="w-5 h-5 text-[var(--macos-text-tertiary)]" />
      </div>
      <div class="font-medium">{{ $t('cookies.no_cookies_found') }}</div>
      <div>{{ $t('cookies.try_sync') }}</div>
    </div>

    <!-- list -->
    <div v-else class="p-3 pt-2 flex flex-col gap-2">
      <div v-for="collection in visibleBrowserCollections" :key="collection?.id || collection?.browser" class="ck-box" :class="brandClass(collection?.browser)">
        <!-- Single-column: icon inline with name; status at right; rows centered where needed -->
        <div class="ck-row single">
          <div class="main">
            <div class="title-row">
              <div class="title name-with-icon" :title="collection?.browser">
                <Icon :name="getBrowserSemanticIcon(collection?.browser)" class="mini-icon" />
                <span class="one-line">{{ collection?.browser || collection?.name }}</span>
              </div>
              <!-- 将胶囊直接作为 flex 子项，使用 margin-left:auto 推到右侧，并在胶囊自身限制宽度 -->
              <div v-if="getSyncStatusText(collection)"
                   class="chip-frosted chip-sm chip-translucent status-pill trunc push-right"
                   :class="statusPillClass(collection)" @click.stop="showStatus(collection)">
                <span class="chip-dot"></span>
                <span class="chip-label one-line">{{ getSyncStatusText(collection) }}</span>
              </div>
            </div>
            <div class="info-row">
              <div class="meta-group small nowrap flex-1 min-w-0">
                <div class="item"><Icon name="database" class="w-3.5 h-3.5" />{{ getBrowserCookieCount(collection) }}</div>
                <div class="divider-v"></div>
                <div class="item"><Icon name="globe" class="w-3.5 h-3.5" />{{ getDomainCount(collection) }}</div>
                <div v-if="getLastSyncTime(collection)" class="divider-v"></div>
                <div v-if="getLastSyncTime(collection)" class="item flex-1 min-w-0">
                  <Icon name="clock" class="w-3.5 h-3.5" />
                  <span class="truncate" :title="formatSyncTime(getLastSyncTime(collection))">{{ formatSyncTime(getLastSyncTime(collection)) }}</span>
                </div>
                <div class="divider-v"></div>
                <div class="item flex-none">
                  <span :class="statusTextClass(collection)" :title="getStatusText(collection)">{{ getStatusText(collection) }}</span>
                </div>
              </div>
            </div>
            <div class="actions">
              <template v-if="collection?.browser && !syncingBrowsers.has(collection.browser)">
                <div class="segmented ops-actions">
                  <button class="seg-item yt" :disabled="syncingBrowsers.has(collection.browser)" @click.stop="syncCookies('yt-dlp', [collection.browser])">
                    <Icon name="terminal" class="w-4 h-4 icon" />
                    <span class="label">{{ $t('cookies.sync_with', { type: 'yt-dlp' }) }}</span>
                  </button>
                </div>
              </template>
              <template v-else>
                <div class="syncing-indicator">
                  <Icon name="spinner" class="w-4 h-4 spin-smooth" />
                  <span class="syncing-text">{{ $t('cookies.status.syncing') }}</span>
                </div>
              </template>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="p-3 pt-1 flex justify-center">
      <div class="meta-group">
        <div class="item">
          <span class="num">{{ visibleBrowserCollections.length }}</span>
          <span class="label">{{ $t('cookies.browsers') }}</span>
        </div>
        <div class="divider-v"></div>
        <div class="item">
          <span class="label">{{ $t('cookies.total_cookies', { count: (visibleBrowserCollections || []).reduce((sum, col) => sum + getBrowserCookieCount(col), 0) }) }}</span>
        </div>
      </div>
    </div>

    <div class="manual-section px-3 pb-3">
      <div class="manual-header">
        <span>{{ $t('cookies.manual_section_title') }}</span>
        <button class="btn-chip-ghost" @click="openManualModal('create', null)">
          <Icon name="plus" class="w-3.5 h-3.5" />
          <span>{{ $t('cookies.manual_add_button') }}</span>
        </button>
      </div>

      <div v-if="manualCollections.length" class="manual-list flex flex-col gap-2">
        <div v-for="manual in manualCollections" :key="manual?.id || manual?.name" class="ck-box manual-entry">
          <div class="ck-row single">
            <div class="main">
              <div class="title-row">
                <div class="title name-with-icon" :title="manual?.name || $t('cookies.manual_default_name')">
                  <Icon name="layers" class="mini-icon" />
                  <span class="one-line">{{ manual?.name || $t('cookies.manual_default_name') }}</span>
                </div>
              </div>

              <div class="info-row">
                <div class="meta-group small nowrap flex-1 min-w-0">
                  <div class="item"><Icon name="database" class="w-3.5 h-3.5" />{{ getBrowserCookieCount(manual) }}</div>
                  <div class="divider-v"></div>
                  <div class="item"><Icon name="globe" class="w-3.5 h-3.5" />{{ getDomainCount(manual) }}</div>
                  <div v-if="getCollectionUpdated(manual)" class="divider-v"></div>
                  <div v-if="getCollectionUpdated(manual)" class="item flex-1 min-w-0">
                    <Icon name="clock" class="w-3.5 h-3.5" />
                    <span class="truncate" :title="formatSyncTime(getCollectionUpdated(manual))">{{ formatSyncTime(getCollectionUpdated(manual)) }}</span>
                  </div>
                </div>
              </div>

              <div class="actions">
                <div class="segmented ops-actions">
                  <button class="seg-item" @click.stop="openManualModal('edit', manual)">
                    <Icon name="edit" class="w-4 h-4 icon" />
                    <span class="label">{{ $t('cookies.manual_edit_coming') }}</span>
                  </button>
                  <button class="seg-item" @click.stop="exportManualCollection(manual)">
                    <Icon name="file-copy" class="w-4 h-4 icon" />
                    <span class="label">{{ $t('common.copy') }}</span>
                  </button>
                  <button class="seg-item danger" @click.stop="deleteManualCollection(manual)">
                    <Icon name="trash" class="w-4 h-4 icon" />
                    <span class="label">{{ $t('common.delete') }}</span>
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="manual-empty">
        <div class="empty-icon"><Icon name="layers" class="w-6 h-6" /></div>
        <div class="title">{{ $t('cookies.manual_empty_title') }}</div>
      </div>
    </div>

    <div class="p-3 pt-1 flex justify-center">
      <div class="meta-group">
        <div class="item">
          <span class="num">{{ manualCollections.length }}</span>
          <span class="label">{{ $t('cookies.manual_collection_sets') }}</span>
        </div>
        <div class="divider-v"></div>
        <div class="item">
          <span class="label">{{ $t('cookies.total_cookies', { count: (manualCollections || []).reduce((sum, col) => sum + getBrowserCookieCount(col), 0) }) }}</span>
        </div>
      </div>
    </div>
  </div>
    <CookiesSearchModal :show="showModal" @close="showModal = false" />
    <ManualCookiesModal
      :show="showManualModal"
      :mode="manualModalMode"
      :collection="manualEditing"
      @close="closeManualModal"
      @submit="handleManualSubmit"
    />
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { ListAllCookies, SyncCookies, CreateManualCollection, UpdateManualCollection, DeleteCookieCollection, ExportCookieCollection } from 'wailsjs/go/api/CookiesAPI'
import { useDtStore } from '@/stores/downloadTasks'
import CookiesSearchModal from '@/components/modal/CookiesSearchModal.vue'
import ManualCookiesModal from '@/components/modal/ManualCookiesModal.vue'
import { isWindows } from '@/utils/platform.js'

// Declare custom events to avoid extraneous listener warnings on fragment roots
defineEmits(['open-modal'])

const { t } = useI18n()
const dtStore = useDtStore()

const isLoading = ref(false)
const browserCollections = ref([])
const manualCollections = ref([])
const syncingBrowsers = ref(new Set())
const showModal = ref(false)
const showManualModal = ref(false)
const manualModalMode = ref('create')
const manualEditing = ref(null)
const winAdminHint = ref('')

const visibleBrowserCollections = computed(() => (browserCollections.value || []).filter(Boolean))

const totalCookiesCount = computed(() => {
  const browserTotal = (visibleBrowserCollections.value || []).reduce((sum, col) => sum + getBrowserCookieCount(col), 0)
  const manualTotal = (manualCollections.value || []).reduce((sum, col) => sum + getBrowserCookieCount(col), 0)
  return browserTotal + manualTotal
})

// no expand in inspector

function getBrowserCookieCount(collection) {
  const domains = collection?.domain_cookies
  if (!domains) return 0
  return Object.values(domains).reduce((acc, d) => acc + (Array.isArray(d.cookies) ? d.cookies.length : 0), 0)
}

function getDomainCount(collection) {
  const domains = collection?.domain_cookies
  if (!domains) return 0
  return Object.keys(domains).length
}

function getStatusText(collection) {
  const status = collection?.status
  const map = { synced: t('cookies.status.synced'), never: t('cookies.status.never'), syncing: t('cookies.status.syncing'), error: t('cookies.status.error'), manual: t('cookies.status.manual') }
  return map[status] || t('cookies.status.unknown')
}

function statusTextClass(collection) {
  const status = collection?.status
  const map = {
    synced: 'text-[var(--macos-success-text)]',
    error: 'text-[var(--macos-danger-text)]',
    syncing: 'text-[var(--macos-text-secondary)]',
    never: 'text-[var(--macos-text-secondary)]',
    manual: 'text-[var(--macos-text-primary)]'
  }
  return map[status] || 'text-[var(--macos-text-secondary)]'
}

function getSyncStatusText(collection) {
  const st = collection?.last_sync_status
  if (!st) return ''
  if (st === 'success') return t('cookies.sync_success')
  if (st === 'failed') return t('cookies.sync_error', { msg: collection?.status_description })
  if (st === 'manual') return t('cookies.status.manual')
  return ''
}

function getSyncStatusClass(collection) {
  const st = collection?.last_sync_status
  if (!st) return ''
  return st === 'success' ? 'text-green-600' : (st === 'failed' ? 'text-red-600' : '')
}

function getSyncStatusIcon(collection) {
  const st = collection?.last_sync_status
  if (st === 'success') return 'status-success'
  if (st === 'failed') return 'status-error'
  return 'status-warning'
}

function getSyncFromOptions(collection) {
  return collection?.sync_from || ['yt-dlp']
}

function getCollectionUpdated(collection) {
  return collection?.updated_at || collection?.last_sync_time || collection?.created_at || null
}

// no per-cookie rendering in inspector

async function fetchCookies() {
  isLoading.value = true
  try {
    const res = await ListAllCookies()
    if (!res?.success) throw new Error(res?.msg || 'Fetch failed')
    const data = JSON.parse(res.data || '{}') || {}
    const browsers = Array.isArray(data.browser_collections) ? data.browser_collections : []
    const manuals = Array.isArray(data.manual_collections) ? data.manual_collections : []
    browserCollections.value = browsers
    manualCollections.value = manuals
    if (isWindows() && browsers.length === 0) {
      winAdminHint.value = 'No browser cookies detected. On Windows, avoid running as Administrator; run as the same normal user who uses the browser.'
    } else {
      winAdminHint.value = ''
    }
  } catch (e) {
    $message?.error?.('Fetch cookies error: ' + (e?.message || String(e)))
  } finally {
    isLoading.value = false
  }
}

async function syncCookies(syncFrom, list) {
  if (!syncFrom || !list || !list.length) return
  list.forEach(b => syncingBrowsers.value.add(b))
  try {
    const handle = (data) => {
      // feedback
      if (data?.status === 'started') $message?.info?.(t('cookies.sync_started'))
      else if (data?.status === 'success') $message?.success?.(t('cookies.sync_success'))
      else if (data?.status === 'failed') $message?.error?.(t('cookies.sync_error', { msg: data.error }))
      if (data?.done) {
        list.forEach(b => syncingBrowsers.value.delete(b))
        fetchCookies()
        dtStore.removeCookieSyncCallback(handle)
      }
    }
    dtStore.registerCookieSyncCallback(handle)
    const res = await SyncCookies(syncFrom, list)
    if (!res?.success) {
      list.forEach(b => syncingBrowsers.value.delete(b))
      dtStore.removeCookieSyncCallback(handle)
      throw new Error(res?.msg || 'Sync start failed')
    }
  } catch (e) {
    list.forEach(b => syncingBrowsers.value.delete(b))
    $message?.error?.(t('cookies.sync_start_error', { error: e?.message || String(e) }))
  }
}

function getBrowserSemanticIcon(name) {
  // 品牌图标不并入通用 UI 图标集合；这里统一用概念图标
  return 'globe'
}

function getLastSyncTime(collection) {
  return collection?.last_sync_time || null
}

function formatSyncTime(syncTime) {
  if (!syncTime) return ''
  if (typeof syncTime === 'string' && syncTime.startsWith('0001-01-01')) return ''
  const date = new Date(syncTime)
  const now = new Date()
  const diffMs = now - date
  const mins = Math.floor(diffMs / (1000 * 60))
  const hours = Math.floor(diffMs / (1000 * 60 * 60))
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  if (mins < 1) return t('cookies.just_now')
  if (mins < 60) return t('cookies.minutes_ago', { count: mins })
  if (hours < 24) return t('cookies.hours_ago', { count: hours })
  if (days < 7) return t('cookies.days_ago', { count: days })
  return date.toLocaleDateString()
}

onMounted(() => { fetchCookies() })
onUnmounted(() => {})



function brandClass(name) {
  const n = String(name || '').toLowerCase()
  if (n.includes('chrome') || n.includes('chromium') || n.includes('brave') || n.includes('vivaldi')) return 'brand-chrome'
  if (n.includes('firefox')) return 'brand-firefox'
  if (n.includes('safari')) return 'brand-safari'
  if (n.includes('edge')) return 'brand-edge'
  if (n.includes('opera')) return 'brand-opera'
  return ''
}

function statusPillClass(collection) {
  const c = getSyncStatusClass(collection)
  if (c === 'text-green-600') return 'badge-success'
  if (c === 'text-red-600') return 'badge-error'
  return 'badge-info'
}

function showStatus(collection) {
  const text = getSyncStatusText(collection)
  if (!text) return
  try { $dialog?.info?.({ title: t('cookies.title'), content: text }) } catch { alert(text) }
}

function openManualModal(mode = 'create', collection = null) {
  manualModalMode.value = mode
  manualEditing.value = collection ? { ...collection } : null
  showManualModal.value = true
}

function closeManualModal() {
  manualEditing.value = null
  showManualModal.value = false
}

async function handleManualSubmit(payload) {
  const isEdit = manualModalMode.value === 'edit' && manualEditing.value?.id
  const normalizedName = normalizeCollectionName(payload?.name)
  const finalPayload = {
    ...payload,
    name: normalizedName
  }
  try {
    if (isEdit) {
      const res = await UpdateManualCollection(manualEditing.value.id, finalPayload)
      if (!res?.success) throw new Error(res?.msg || 'update failed')
      const updated = safeParseJSON(res?.data)
      if (updated) {
        manualEditing.value = updated
      }
      $message?.success?.(t('common.saved'))
    } else {
      const res = await CreateManualCollection(finalPayload)
      if (!res?.success) throw new Error(res?.msg || 'create failed')
      const created = safeParseJSON(res?.data)
      if (created) {
        manualEditing.value = created
        manualModalMode.value = 'edit'
      }
      $message?.success?.(t('common.saved'))
    }
    fetchCookies()
  } catch (e) {
    $message?.error?.(t('common.save_failed') + ': ' + (e?.message || String(e)))
  }
}

function safeParseJSON(data) {
  if (!data) return null
  try {
    return typeof data === 'string' ? JSON.parse(data) : data
  } catch (error) {
    console.warn('Failed to parse manual collection payload', error)
    return null
  }
}

function normalizeCollectionName(rawName) {
  return (rawName || '').trim()
}

function deleteManualCollection(collection) {
  if (!collection?.id) return

  const executeDelete = async () => {
    try {
      const res = await DeleteCookieCollection(collection.id)
      if (!res?.success) throw new Error(res?.msg || 'delete failed')
      if (window.$message?.success) window.$message.success(t('common.deleted'))
      else $message?.success?.(t('common.deleted'))
      fetchCookies()
    } catch (e) {
      const msg = t('common.delete_failed') + ': ' + (e?.message || String(e))
      if (window.$message?.error) window.$message.error(msg)
      else $message?.error?.(msg)
    }
  }

  if (window?.$dialog?.confirm) {
    window.$dialog.confirm(t('cookies.manual_delete_confirm'), {
      title: t('common.confirm'),
      positiveText: t('common.delete'),
      negativeText: t('common.cancel'),
      onPositiveClick: () => { executeDelete() }
    })
    return
  }

  if (window.confirm(t('cookies.manual_delete_confirm'))) {
    executeDelete()
  }
}

async function exportManualCollection(collection) {
  if (!collection?.id) return
  try {
    const res = await ExportCookieCollection(collection.id)
    if (!res?.success) throw new Error(res?.msg || 'export failed')
    const text = res.data || ''
    if (!text) {
      $message?.warning?.(t('cookies.manual_export_empty'))
      return
    }
    if (navigator?.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      $message?.success?.(t('cookies.manual_export_copied'))
    } else {
      const temp = document.createElement('textarea')
      temp.value = text
      temp.style.position = 'fixed'
      temp.style.opacity = '0'
      document.body.appendChild(temp)
      temp.select()
      document.execCommand('copy')
      document.body.removeChild(temp)
      $message?.success?.(t('cookies.manual_export_copied'))
    }
  } catch (e) {
    $message?.error?.(t('cookies.manual_export_failed') + ': ' + (e?.message || String(e)))
  }
}
</script>

<style scoped>
.cookies-panel { font-size: var(--fs-base); color: var(--macos-text-primary); }
.toolbar { }
.search-link { width: 100%; height: 28px; display:flex; align-items:center; gap:8px; border: 1px solid var(--macos-separator); border-radius: 8px; background: var(--macos-background); padding: 0 8px; color: var(--macos-text-secondary); }
.search-link .placeholder { font-size: var(--fs-sub); }
.ck-box { border: 1px solid var(--macos-separator); border-radius: 10px; background: var(--macos-background); overflow: visible; }
.ck-row { display:grid; grid-template-columns: 1fr; gap: 8px; align-items: center; padding: 10px; }
.ck-row.single { grid-template-columns: 1fr; }
/* inspector 行纯展示，不要交互态 */
.ck-row { cursor: default; }
.ck-row:hover { background: transparent; }
.left .bicon { width: 24px; height: 24px; border-radius: 6px; display:flex; align-items:center; justify-content:center; background: var(--macos-background-secondary); border: 1px solid var(--macos-separator); color: var(--macos-text-secondary); }
.main { min-width: 0; }
.title-row { display:flex; align-items:center; justify-content: flex-start; gap: 8px; }
.title-row .title { flex: 1 1 auto; min-width: 0; }
.title { font-size: var(--fs-base); font-weight: 500; color: var(--macos-text-primary); }
.name-with-icon { display:inline-flex; align-items:center; gap:6px; }
.mini-icon { width: 14px; height: 14px; color: var(--macos-text-secondary); }
/* 不再使用外层 status-right 限宽，直接在胶囊上限制 */
.status-right { position: relative; }
.status-right { min-width: 0; flex: 0 1 auto; max-width: 50%; overflow: hidden; display:flex; align-items:center; justify-content:flex-end; }
.info-row { display:flex; align-items:center; justify-content: center; gap: 6px; width: 100%; margin-top: 12px; }
.meta-group.small { display:inline-flex; align-items:center; gap:8px; padding: 0 6px; height: 20px; border: 1px solid var(--macos-separator); border-radius: 999px; background: var(--macos-background); color: var(--macos-text-secondary); font-size: var(--fs-sub); }
.meta-group.small.nowrap { white-space: nowrap; max-width: 100%; overflow: hidden; }
.meta-group.small .item { display:inline-flex; align-items:center; gap: 4px; font-size: 11.5px; }
.meta-group.small .item.flex-1 { min-width: 0; }
.meta-group.small .divider-v { width: 1px; height: 12px; background: var(--macos-divider-weak); }
.meta-group.small .badge { border: 0; background: transparent; height: auto; padding: 0; font-size: 11.5px; }
.meta-group.small .badge-success { color: var(--macos-success-text); }
.meta-group.small .badge-error { color: var(--macos-danger-text); }
.meta-group.small .badge-warning { color: #ff9f0a; }
.meta-group.small .badge-primary { color: var(--macos-blue); }
.meta-group.small .badge-info { color: var(--macos-text-secondary); }
.meta-group.small .badge-ghost { color: var(--macos-text-secondary); }
.status-pill { display:inline-flex; align-items:center; gap:4px; height: 20px; padding: 0 8px; border-radius: 999px; font-size: 11.5px; max-width: 100%; overflow: hidden; }
.status-pill.trunc { max-width: 50%; }
.push-right { margin-left: auto; }
/* 使用全局 chip-frosted 半透明背景，不再强制白底 */
.status-pill .chip-label { display: inline-flex; align-items: center; line-height: 1; height: 100%; }
/* Status colors now use global chip-frosted badge-* with a dot; keep text default for better contrast */
.status-pill.trunc .one-line { max-width: 100%; display: inline-block; vertical-align: middle; }
.status-pill .w-3.5, .status-pill .h-3.5, .meta-group.small .w-3.5, .meta-group.small .h-3.5 { display:block; }
/* 统一胶囊内文字/图标垂直居中，且光学居中（通过与容器等高的 line-height 实现） */
.status-pill .chip-label { display:inline-flex; align-items:center; height:100%; line-height: 20px; }
.status-pill .chip-label,.status-pill { -webkit-font-smoothing: antialiased; }
/* 让错误态图标在视觉上与对号一致（略大一丢） */
/* 无图标，仅文本显示 */
.info-row .badge { border: 0; background: transparent; height: 20px; line-height: 20px; padding: 0 6px; font-size: 11.5px; display: inline-flex; align-items: center; }
.info-row .badge-success { color: var(--macos-success-text); }
.info-row .badge-error { color: var(--macos-danger-text); }
.info-row .badge-warning { color: #ff9f0a; }
.info-row .badge-primary { color: var(--macos-blue); }
.info-row .badge-info { color: var(--macos-text-secondary); }
.info-row .badge-ghost { color: var(--macos-text-secondary); }
.status-line { display:flex; align-items:center; gap: 6px; margin-top: 4px; }
.actions { display:flex; align-items:center; justify-content:center; margin-top: 12px; }
.ops-actions { display:inline-flex; align-items:center; padding: 2px; border: 1px solid var(--macos-separator); border-radius: 8px; background: var(--macos-background); }
.ops-actions .seg-item { min-width: 28px; height: 22px; padding: 0 6px; border-radius: 6px; display:inline-flex; align-items:center; justify-content:center; color: var(--macos-text-secondary); position: relative; overflow: hidden; }
.ops-actions .seg-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
.ops-actions .seg-item .icon { transition: transform .18s ease; }
.ops-actions .seg-item .label { max-width: 0; opacity: 0; transform: translateX(4px); transition: max-width .18s ease, opacity .18s ease, transform .18s ease, color .12s ease; white-space: nowrap; color: var(--macos-text-secondary); }
.ops-actions .seg-item:hover .label, .ops-actions .seg-item:active .label, .ops-actions .seg-item.working .label { max-width: 120px; opacity: 1; transform: translateX(0); color: var(--macos-blue); }
.ops-actions .seg-item:hover .icon { transform: translateX(-2px); }
.ops-actions .seg-item.yt { border: 1px solid transparent; }
.ops-actions .seg-item.danger { color: #ff3b30; }
.ops-actions .seg-item.danger .label { color: #ff3b30; }
.ops-actions .seg-item.danger:hover .label, .ops-actions .seg-item.danger:active .label { color: #ff3b30; }
/* removed dreamcreator button */
.one-line { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.table-wrap { overflow-x: auto; border-top: 1px solid var(--macos-divider-weak); }
thead { background: var(--macos-background-secondary); }
.th { text-align: left; font-weight: 600; color: var(--macos-text-secondary); padding: 8px 10px; }
.td { padding: 8px 10px; border-top: 1px solid var(--macos-divider-weak); color: var(--macos-text-primary); }
.mono { font-family: var(--font-mono); }
.truncate { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 100%; }
.sr-mini-btn { display:inline-flex; align-items:center; gap:6px; height: 28px; padding: 0 10px; border-radius: 6px; font-size: var(--fs-sub); color: var(--macos-text-primary); background: var(--macos-background); border: 1px solid var(--macos-separator); }
.sr-mini-btn:disabled { opacity: .6; cursor: not-allowed; }
.spinning { animation: macos-spin .6s ease-in-out both; }
@keyframes macos-spin { to { transform: rotate(360deg); } }
/* smoother syncing spinner for inspector */
@keyframes sync-rotate { to { transform: rotate(360deg); } }
.sync-rot { animation: sync-rotate 1s linear infinite; }
.spin-smooth { animation: sync-rotate .8s linear infinite; will-change: transform; }
.syncing-indicator {
  display: inline-flex;
  flex-direction: row;
  align-items: center;
  gap: 6px;
  color: var(--macos-text-secondary);
  white-space: nowrap;
  word-break: keep-all;
  flex-wrap: nowrap;
}
.syncing-text {
  font-size: var(--fs-sub);
  white-space: nowrap;
  word-break: keep-all;
}

/* row interactions */
.row-cookie { cursor: pointer; }
.row-cookie:hover { background: var(--macos-gray-hover); }
.row-detail .td { background: var(--macos-background-secondary); }
.detail-card { display:flex; flex-direction: column; gap: 6px; }
.detail-row { display:grid; grid-template-columns: 80px 1fr; align-items: center; gap: 8px; }
.detail-row .k { color: var(--macos-text-secondary); font-size: var(--fs-sub); }
.detail-row .v { font-size: var(--fs-sub); color: var(--macos-text-primary); display:flex; align-items: center; justify-content: space-between; gap: 8px; }
.value-box { white-space: pre-wrap; word-break: break-all; padding: 6px; border: 1px dashed var(--macos-separator); border-radius: 6px; background: var(--macos-background); }
.ops { display:flex; gap: 6px; }

/* brand accents (subtle) */
.brand-chrome .bicon { border-color: var(--brand-chrome); color: var(--brand-chrome); background: color-mix(in oklab, var(--brand-chrome) 10%, var(--macos-background)); }
.brand-firefox .bicon { border-color: var(--brand-firefox); color: var(--brand-firefox); background: color-mix(in oklab, var(--brand-firefox) 10%, var(--macos-background)); }
.brand-safari .bicon { border-color: var(--brand-safari); color: var(--brand-safari); background: color-mix(in oklab, var(--brand-safari) 10%, var(--macos-background)); }
.brand-edge .bicon { border-color: var(--brand-edge); color: var(--brand-edge); background: color-mix(in oklab, var(--brand-edge) 10%, var(--macos-background)); }
.brand-opera .bicon { border-color: #ff1b2d; color: #ff1b2d; background: color-mix(in oklab, #ff1b2d 10%, var(--macos-background)); }
.brand-chrome .mini-icon { color: var(--brand-chrome); }
.brand-firefox .mini-icon { color: var(--brand-firefox); }
.brand-safari .mini-icon { color: var(--brand-safari); }
.brand-edge .mini-icon { color: var(--brand-edge); }
.brand-chrome .mini-icon { color: #1a73e8; }
.brand-firefox .mini-icon { color: #ff9500; }
.brand-safari .mini-icon { color: #0fb5ee; }
.brand-edge .mini-icon { color: #0b84ed; }
.brand-opera .mini-icon { color: #ff1b2d; }

/* header/meta utilities for panel */
.icon-tonal { width: 28px; height: 28px; display:inline-flex; align-items:center; justify-content:center; border-radius: 6px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-secondary); }
.meta-group .item .num { color: var(--macos-text-primary); font-weight: 600; }

.manual-section { display:flex; flex-direction: column; gap: 8px; }
.manual-header { font-size: 13px; font-weight: 600; color: var(--macos-text-secondary); padding: 0 2px; display:flex; align-items:center; justify-content: space-between; }
.manual-header .btn-chip-ghost { display:inline-flex; align-items:center; gap: 6px; }
.manual-list .ck-box { border-style: solid; }
.manual-entry .actions { margin-top: 10px; }
.manual-entry .ops-actions .seg-item { min-width: 32px; }
.manual-entry .ops-actions .seg-item.danger { color: #ff3b30; }
.manual-entry .ops-actions .seg-item.danger:hover { background: rgba(255,76,58,0.12); color: #ff3b30; }
.manual-empty { border: 1px dashed var(--macos-separator); border-radius: 10px; padding: 28px 18px; text-align: center; color: var(--macos-text-secondary); display:flex; flex-direction: column; gap: 10px; align-items:center; justify-content:center; }
.manual-empty .empty-icon { width: 46px; height: 46px; border-radius: 14px; border: 1px solid var(--macos-separator); display:flex; align-items:center; justify-content:center; background: color-mix(in oklab, var(--macos-background) 92%, var(--macos-separator)); color: var(--macos-text-tertiary); }
.manual-empty .title { font-weight: 600; color: var(--macos-text-primary); }
.manual-empty .desc { font-size: 12px; color: var(--macos-text-secondary); max-width: 320px; }
/* 按钮危险态统一迁移为全局 .btn-chip.btn-danger */
</style>
