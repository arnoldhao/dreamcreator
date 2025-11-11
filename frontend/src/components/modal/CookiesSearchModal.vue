<template>
  <div v-if="show" class="macos-modal">
    <div class="modal-card big card-frosted card-translucent">
      <div class="modal-header">
        <ModalTrafficLights @close="$emit('close')" />
        <div class="title">
          <div class="meta-group">
            <div class="item">
              <span class="num">{{ browsers.length }}</span>
              <span class="label">{{ $t('cookies.browsers') }}</span>
            </div>
            <div class="divider-v"></div>
            <div class="item">
              <span class="label">{{ $t('cookies.total_cookies', { count: totalCookiesCount }) }}</span>
            </div>
          </div>
        </div>
        <div class="header-meta">
          <button class="icon-chip-ghost" :data-tooltip="$t('common.refresh')" @click="fetchCookies" :disabled="isLoading">
            <Icon name="refresh" class="w-4 h-4" :class="{ spinning: isLoading }" />
          </button>
        </div>
      </div>
      <div class="modal-body">
        <!-- controls: search + browser segmented on one line -->
        <div class="controls">
          <div class="search-wrap">
            <Icon name="search" class="search-icon" />
            <input v-model="searchQuery" type="text" class="input-macos h-[26px] w-full pl-7 pr-7 text-sm"
                   :placeholder="$t('cookies.search_placeholder')" />
            <button v-if="searchQuery" class="icon-chip-ghost search-clear" :data-tooltip="$t('common.reset')" @click="searchQuery = ''">
              <Icon name="close" class="w-4 h-4" />
            </button>
          </div>
          <div class="segmented browser-seg" v-if="browsers && browsers.length">
            <button class="seg-item" :class="{ active: !selectedBrowser }" @click="selectedBrowser = ''">{{ $t('cookies.all_browsers') }}</button>
            <button v-for="b in browsers" :key="b" class="seg-item" :class="{ active: selectedBrowser === b }" @click="selectedBrowser = (selectedBrowser === b ? '' : b)">{{ b }}</button>
          </div>
        </div>

        <!-- loading -->
        <div v-if="isLoading" class="state state-row text-secondary">
          <div class="spinner sm"></div>
          <div>{{ $t('cookies.loading_cookies') }}</div>
        </div>

        <!-- empty -->
        <div v-else-if="!filteredBrowsers.length" class="state text-secondary">
          <Icon name="globe" class="w-7 h-7 text-[var(--macos-text-tertiary)]" />
          <div class="mt-1 font-medium">{{ searchQuery ? $t('cookies.no_matching_cookies') : $t('cookies.no_cookies_found') }}</div>
          <div>{{ searchQuery ? $t('cookies.try_different_search') : $t('cookies.try_sync') }}</div>
        </div>

        <!-- list -->
        <div v-else class="list">
          <div v-for="browser in filteredBrowsers" :key="browser" class="b-section" :class="brandClass(browser)">
            <div class="b-header compact">
              <div class="left compact">
                <div class="title name-with-icon" :title="browser">
                  <Icon :name="getBrowserSemanticIcon(browser)" class="mini-icon" />
                  <span class="one-line">{{ browser }}</span>
                </div>
                <div class="meta-group small">
                  <div class="item"><Icon name="database" class="w-3.5 h-3.5" />{{ getBrowserCookieCount(browser) }}</div>
                  <div class="divider-v"></div>
                  <div class="item"><Icon name="globe" class="w-3.5 h-3.5" />{{ getDomainCount(browser) }}</div>
                  <div v-if="getLastSyncTime(browser)" class="divider-v"></div>
                  <div v-if="getLastSyncTime(browser)" class="item"><Icon name="clock" class="w-3.5 h-3.5" />{{ formatSyncTime(getLastSyncTime(browser)) }}</div>
                  <div class="divider-v"></div>
                  <div class="item" v-if="!isBrowserSyncing(browser)">
                    <span :class="statusTextClass(browser)">{{ getStatusText(browser) }}</span>
                  </div>
                  <div v-if="getSyncStatusText(browser)" class="divider-v"></div>
                  <div v-if="getSyncStatusText(browser)" class="item cursor-pointer" @click.stop="showStatus(browser)">
                    <Icon :name="getSyncStatusClass(browser) === 'text-green-600' ? 'status-success' : 'status-warning'" class="w-3.5 h-3.5" />
                    <span class="one-line" :class="getSyncStatusClass(browser)">{{ getSyncStatusText(browser) }}</span>
                  </div>
                </div>
              </div>
              <div class="ops">
                <template v-if="!syncingBrowsers.has(browser)">
                  <div class="segmented ops-actions">
                <button class="seg-item" :class="[syncBtnClass('yt-dlp')]" :data-tooltip="$t('cookies.sync_with', { type: 'yt-dlp' })" data-tip-pos="top"
                            :disabled="syncingBrowsers.has(browser)" @click.stop="syncCookies('yt-dlp', [browser])">
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
                  <button class="icon-chip-ghost" :data-tooltip="isBrowserExpanded(browser) ? $t('common.collapse') : $t('common.expand')" @click.stop="toggleBrowser(browser)">
                    <Icon :name="isBrowserExpanded(browser) ? 'chevron-up' : 'chevron-down'" class="w-4 h-4" />
                  </button>
              </div>
            </div>

            <!-- table collapsed by default; fixed layout + proportional widths -->
            <div class="table-wrap" v-show="isBrowserExpanded(browser)">
              <table class="ck-table w-full text-sm">
                <thead>
                  <tr>
                    <th class="th col-domain">{{ $t('cookies.domain') }}</th>
                    <th class="th col-name">{{ $t('cookies.name') }}</th>
                    <th class="th col-value">{{ $t('cookies.value') }}</th>
                    <th class="th w-10"></th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="(cookie, idx) in getFilteredCookies(browser).slice(0, visibleCounts[browser] || 0)" :key="idx" class="row">
                    <td class="td mono rel" @click="toggleCell(browser, idx, 'Domain')">
                      <div class="truncate" :title="cookie.Domain">{{ cookie.Domain }}</div>
                      <div v-if="isCellActive(browser, idx, 'Domain')" class="cell-pop">
                        <div class="pop-text mono">{{ cookie.Domain }}</div>
                        <div class="pop-ops"><button class="icon-chip-ghost" :data-tooltip="$t('common.copy')" @click.stop="copy(cookie.Domain)"><Icon name="file-copy" class="w-4 h-4" /></button></div>
                      </div>
                    </td>
                    <td class="td mono rel" @click="toggleCell(browser, idx, 'Name')">
                      <div class="truncate" :title="cookie.Name">{{ cookie.Name }}</div>
                      <div v-if="isCellActive(browser, idx, 'Name')" class="cell-pop">
                        <div class="pop-text mono">{{ cookie.Name }}</div>
                        <div class="pop-ops"><button class="icon-chip-ghost" :data-tooltip="$t('common.copy')" @click.stop="copy(cookie.Name)"><Icon name="file-copy" class="w-4 h-4" /></button></div>
                      </div>
                    </td>
                    <td class="td mono rel text-[var(--macos-text-secondary)]" @click="toggleCell(browser, idx, 'Value')">
                      <div class="truncate" :title="cookie.Value">{{ cookie.Value }}</div>
                      <div v-if="isCellActive(browser, idx, 'Value')" class="cell-pop">
                        <div class="pop-text mono break-all">{{ cookie.Value }}</div>
                        <div class="pop-ops"><button class="icon-chip-ghost" :data-tooltip="$t('common.copy')" @click.stop="copy(cookie.Value)"><Icon name="file-copy" class="w-4 h-4" /></button></div>
                      </div>
                    </td>
                    <td class="td ops">
                      <button class="icon-chip-ghost" :data-tooltip="$t('common.copy')" @click="copy(cookie.Value)">
                        <Icon name="file-copy" class="w-4 h-4" />
                      </button>
                    </td>
                  </tr>
                </tbody>
              </table>
              <div class="load-more" v-if="(visibleCounts[browser] || 0) < getFilteredCookies(browser).length">
            <button class="btn-chip-ghost" @click.stop="loadMore(browser)">{{ $t('common.load_more') }} ({{ getFilteredCookies(browser).length - (visibleCounts[browser] || 0) }})</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, onUnmounted } from 'vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import { useI18n } from 'vue-i18n'
import { ListAllCookies, SyncCookies } from 'wailsjs/go/api/CookiesAPI'
import { useDtStore } from '@/handlers/downtasks'
// 统一的语义图标：浏览器用概念图标（不混入品牌）

const props = defineProps({ show: Boolean })
const emit = defineEmits(['close'])

const { t } = useI18n()
const dtStore = useDtStore()

const isLoading = ref(false)
const browsers = ref([])
const cookiesByBrowser = ref({})
const searchQuery = ref('')
const debouncedQuery = ref('')
const selectedBrowser = ref('')
const syncingBrowsers = ref(new Set())
const expandedBrowsers = ref([])
const visibleCounts = ref({}) // per browser visible row count
const PAGE_SIZE = 300
const activeCellKey = ref('')

// debounce search to keep UI responsive on large datasets
let debounceTimer
watch(searchQuery, (v) => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(() => { debouncedQuery.value = v }, 180)
})

// Pre-filter map per browser when searching. When no query, we compute lazily on expand.
const filteredCookiesMap = computed(() => {
  const map = {}
  const q = (debouncedQuery.value || '').toLowerCase().trim()
  if (!q) return map
  for (const b of browsers.value || []) {
    const data = cookiesByBrowser.value[b]
    if (!data) { map[b] = []; continue }
    let all = []
    if (data.domain_cookies) {
      Object.values(data.domain_cookies).forEach(dom => { if (Array.isArray(dom.cookies)) all = all.concat(dom.cookies) })
    }
    map[b] = all.filter(c =>
      (c.Domain && String(c.Domain).toLowerCase().includes(q)) ||
      (c.Name && String(c.Name).toLowerCase().includes(q)) ||
      (c.Value && String(c.Value).toLowerCase().includes(q))
    )
  }
  return map
})

const filteredBrowsers = computed(() => {
  let list = browsers.value || []
  if (selectedBrowser.value) list = list.filter(b => b === selectedBrowser.value)
  const q = (debouncedQuery.value || '').trim()
  if (q) {
    list = list.filter(b => (filteredCookiesMap.value[b] || []).length > 0 || b.toLowerCase().includes(q.toLowerCase()))
  }
  return list
})

const totalCookiesCount = computed(() => (browsers.value || []).reduce((sum, b) => sum + getBrowserCookieCount(b), 0))

function getAllCookiesFlat(browser) {
  const data = cookiesByBrowser.value[browser]
  if (!data) return []
  let all = []
  if (data.domain_cookies) {
    Object.values(data.domain_cookies).forEach(dom => { if (Array.isArray(dom.cookies)) all = all.concat(dom.cookies) })
  }
  return all
}

function getFilteredCookies(browser) {
  const q = (debouncedQuery.value || '').trim()
  if (!q) return getAllCookiesFlat(browser)
  return filteredCookiesMap.value[browser] || []
}

async function fetchCookies() {
  isLoading.value = true
  try {
    const res = await ListAllCookies()
    if (!res?.success) throw new Error(res?.msg || 'Fetch failed')
    const data = JSON.parse(res.data || '{}') || {}
    const browserCols = Array.isArray(data.browser_collections) ? data.browser_collections : []
    const map = {}
    const list = []
    browserCols.forEach(col => {
      if (col?.browser) {
        map[col.browser] = col
        list.push(col.browser)
      }
    })
    cookiesByBrowser.value = map
    browsers.value = list
  } catch (e) {
    $message?.error?.(t('cookies.fetch_error', { msg: e?.message || String(e) }))
  } finally {
    isLoading.value = false
  }
}

function getBrowserCookieCount(browser) {
  const data = cookiesByBrowser.value[browser]
  if (!data || !data.domain_cookies) return 0
  return Object.values(data.domain_cookies).reduce((acc, d) => acc + (Array.isArray(d.cookies) ? d.cookies.length : 0), 0)
}
function getDomainCount(browser) {
  const data = cookiesByBrowser.value[browser]
  if (!data || !data.domain_cookies) return 0
  return Object.keys(data.domain_cookies).length
}
function getStatusText(browser) {
  const s = cookiesByBrowser.value[browser]?.status
  const map = { synced: t('cookies.status.synced'), never: t('cookies.status.never'), syncing: t('cookies.status.syncing'), error: t('cookies.status.error'), manual: t('cookies.status.manual') }
  return map[s] || t('cookies.status.unknown')
}
function isBrowserSyncing(browser) {
  return cookiesByBrowser.value[browser]?.status === 'syncing'
}
function statusBadge(browser) {
  const s = cookiesByBrowser.value[browser]?.status
  return ({ synced: 'badge-success', never: 'badge-ghost', syncing: 'badge-info', error: 'badge-error' }[s]) || 'badge-ghost'
}
function statusTextClass(browser) {
  const s = cookiesByBrowser.value[browser]?.status
  const map = {
    synced: 'text-[var(--macos-success-text)]',
    error: 'text-[var(--macos-danger-text)]',
    syncing: 'text-[var(--macos-text-secondary)]',
    never: 'text-[var(--macos-text-secondary)]',
    manual: 'text-[var(--macos-text-primary)]'
  }
  return map[s] || 'text-[var(--macos-text-secondary)]'
}
function getSyncFromOptions(browser) { return cookiesByBrowser.value[browser]?.sync_from || ['yt-dlp'] }
function syncBtnClass(syncType) { return 'yt' }
function statusPillClass(browser) {
  const c = getSyncStatusClass(browser)
  if (c === 'text-green-600') return 'pill-success'
  if (c === 'text-red-600') return 'pill-error'
  return 'pill-neutral'
}

// sync status text/class (last sync result)
function getSyncStatusText(browser) {
  const data = cookiesByBrowser.value[browser]
  if (!data || !data.last_sync_status) return ''
  if (data.last_sync_status === 'success') return t('cookies.sync_success')
  if (data.last_sync_status === 'failed') return t('cookies.sync_error', { msg: data.status_description })
  return ''
}
function getSyncStatusClass(browser) {
  const data = cookiesByBrowser.value[browser]
  if (!data || !data.last_sync_status) return ''
  return data.last_sync_status === 'success' ? 'text-green-600' : (data.last_sync_status === 'failed' ? 'text-red-600' : '')
}
function getBrowserSemanticIcon() { return 'globe' }
function showStatus(browser) {
  const text = getSyncStatusText(browser)
  if (!text) return
  try { $dialog?.info?.({ title: t('cookies.title'), content: text }) } catch { alert(text) }
}
function brandClass(name) {
  const n = String(name || '').toLowerCase()
  if (n.includes('chrome') || n.includes('chromium') || n.includes('brave') || n.includes('vivaldi')) return 'brand-chrome'
  if (n.includes('firefox')) return 'brand-firefox'
  if (n.includes('safari')) return 'brand-safari'
  if (n.includes('edge')) return 'brand-edge'
  if (n.includes('opera')) return 'brand-opera'
  return ''
}
function getLastSyncTime(browser) { return cookiesByBrowser.value[browser]?.last_sync_time || null }
function formatSyncTime(syncTime) {
  if (!syncTime) return ''
  if (typeof syncTime === 'string' && syncTime.startsWith('0001-01-01')) return ''
  const date = new Date(syncTime), now = new Date()
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

async function syncCookies(syncFrom, list) {
  if (!syncFrom || !list || !list.length) return
  list.forEach(b => syncingBrowsers.value.add(b))
  try {
    const handle = (data) => {
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
    $message?.error?.(t('cookies.sync_start_error'))
  }
}

watch(() => props.show, (v) => { if (v) { fetchCookies() } })
onMounted(() => { if (props.show) { fetchCookies() } })
onUnmounted(() => {})

import { copyText as copyToClipboard } from '@/utils/clipboard.js'
async function copy(text) { await copyToClipboard(text, t) }

// expand / collapse
function isBrowserExpanded(browser) { return expandedBrowsers.value.includes(browser) }
function toggleBrowser(browser) {
  const i = expandedBrowsers.value.indexOf(browser)
  if (i >= 0) expandedBrowsers.value.splice(i, 1)
  else {
    expandedBrowsers.value.push(browser)
    // init visible count lazily
    const total = getFilteredCookies(browser).length
    visibleCounts.value[browser] = Math.min(PAGE_SIZE, total)
  }
}

// cell popup
function keyOf(browser, idx, col) { return `${browser}:${idx}:${col}` }
function toggleCell(browser, idx, col) {
  const k = keyOf(browser, idx, col)
  activeCellKey.value = (activeCellKey.value === k) ? '' : k
}
function isCellActive(browser, idx, col) { return activeCellKey.value === keyOf(browser, idx, col) }

function loadMore(browser) {
  const total = getFilteredCookies(browser).length
  const cur = visibleCounts.value[browser] || 0
  visibleCounts.value[browser] = Math.min(total, cur + PAGE_SIZE)
}

// 根据搜索意图自动展开/收起：有查询 -> 展开匹配；清空 -> 收起
watch(debouncedQuery, (q) => {
  const has = !!(q && String(q).trim())
  if (has) {
    const list = filteredBrowsers.value
    expandedBrowsers.value = [...list]
    for (const b of list) {
      const total = getFilteredCookies(b).length
      visibleCounts.value[b] = Math.min(PAGE_SIZE, total)
    }
  } else {
    expandedBrowsers.value = []
  }
})
</script>

<style scoped>
.macos-modal { position: fixed; inset: 0; background: rgba(0,0,0,0.2); display:flex; align-items:center; justify-content:center; z-index: 2000; backdrop-filter: blur(8px); -webkit-backdrop-filter: blur(8px); }
.modal-card { width: 720px; max-width: calc(100% - 32px); border-radius: 12px; overflow:hidden; }
.modal-card.big { width: 860px; }
/* Always-on active frosted look */
.modal-card.card-frosted.card-translucent { background: color-mix(in oklab, var(--macos-surface) 76%, transparent); border: 1px solid rgba(255,255,255,0.28); box-shadow: var(--macos-shadow-2), 0 12px 30px rgba(0,0,0,0.24); }
.modal-header { height: 36px; display:grid; grid-template-columns: auto 1fr auto; align-items:center; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.modal-header .title { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); text-align: center; }
.modal-body { padding: 12px; max-height: 70vh; overflow: auto; }

/* traffic lights */
.traffic { display:flex; align-items:center; gap:6px; width: 60px; -webkit-app-region: no-drag; --wails-draggable: no-drag; }
.header-meta { display:flex; align-items:center; justify-content:flex-end; min-width: 0; }
.meta-group { display:inline-flex; align-items:center; gap:8px; padding: 0 8px; height: 24px; border: 1px solid var(--macos-separator); border-radius: 8px; background: var(--macos-background); }
.meta-group .item { display:inline-flex; align-items:center; gap:6px; font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.meta-group .item .num { color: var(--macos-text-primary); font-weight: 600; }
.meta-group .divider-v { width: 1px; height: 14px; background: var(--macos-divider-weak); }
.tl { width: 14px; height: 14px; border-radius: 50%; display:flex; align-items:center; justify-content:center; }
.tl button, .tl { border: none; background: transparent; padding: 0; cursor: default; }
.tl.tl-close { cursor: pointer; }
.dot { width: 10px; height: 10px; border-radius: 50%; display:block; }
.dot-red { background: #ff5f57; border: 1px solid rgba(0,0,0,0.1); }
.dot-yellow { background: #febc2e; border: 1px solid rgba(0,0,0,0.1); }
.dot-green { background: #28c940; border: 1px solid rgba(0,0,0,0.1); }
/* keep modal yellow/green visually grey to match macOS sheets */
.disabled .dot { background: #d9d9d9; border-color: rgba(0,0,0,0.1); }

.controls { display:flex; align-items:center; gap:8px; margin-bottom: 10px; }
.search-wrap { position: relative; flex: 1 1 auto; min-width: 200px; }
.search-wrap .search-icon { position: absolute; left: 8px; top: 50%; transform: translateY(-50%); width: 16px; height: 16px; color: var(--macos-text-tertiary); pointer-events: none; }
.search-wrap .search-clear { position: absolute; right: 4px; top: 50%; transform: translateY(-50%); }
/* ensure input text does not overlap the left icon */
.search-wrap input { padding-left: 32px !important; padding-right: 28px !important; }
/* segmented browser chips */
.segmented { display:inline-flex; background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 8px; padding: 2px; }
.seg-item { min-width: 34px; height: 28px; padding: 0 8px; border-radius: 6px; color: var(--macos-text-secondary); transition: background .15s ease, color .15s ease; font-size: var(--fs-sub); line-height: 1; }
.seg-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
.seg-item.active, .seg-item.active:hover { background: var(--macos-blue); color: #fff; }

.state { display:flex; flex-direction: column; align-items:center; justify-content:center; gap:6px; padding: 24px; font-size: var(--fs-base); }
.spinner { width: 26px; height: 26px; border: 2px solid var(--macos-blue); border-top-color: transparent; border-radius: 50%; animation: macos-spin .8s linear infinite; }
@keyframes macos-spin { to { transform: rotate(360deg); } }
/* loading row layout + small spinner */
.state-row { flex-direction: row; gap: 8px; }
.spinner.sm { width: 16px; height: 16px; border-width: 2px; }

.list { display:flex; flex-direction: column; gap: 10px; }
.b-section { border: 1px solid var(--macos-separator); border-radius: 10px; overflow: hidden; background: var(--macos-background); }
.b-header { padding: 10px; display:flex; align-items:center; justify-content: space-between; gap:8px; border-bottom: 1px solid var(--macos-divider-weak); }
.b-header .left { display:flex; align-items:center; gap:10px; min-width: 0; }
.b-header.compact { padding: 8px 10px; }
.left.compact { gap: 8px; }
.bicon { width: 24px; height: 24px; border-radius: 6px; display:flex; align-items:center; justify-content:center; background: var(--macos-background-secondary); border: 1px solid var(--macos-separator); color: var(--macos-text-secondary); }
.title { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); margin-right: 4px; }
.name-with-icon { display:inline-flex; align-items:center; gap:6px; }
.mini-icon { width: 14px; height: 14px; color: var(--macos-text-secondary); }
.status-right { position: relative; }
.b-info-row { display:flex; align-items:center; justify-content: center; gap: 8px; width: 100%; margin-top: 6px; }
.b-info-row.centered { justify-content: center; }
.status-pill.trunc .one-line { max-width: 220px; display:inline-block; vertical-align: middle; }
.b-info-row .badge { border: 0; background: transparent; height: auto; padding: 0; font-size: 11.5px; }
.b-info-row .badge-success { color: var(--macos-success-text); }
.b-info-row .badge-error { color: var(--macos-danger-text); }
.b-info-row .badge-warning { color: #ff9f0a; }
.b-info-row .badge-primary { color: var(--macos-blue); }
.b-info-row .badge-info { color: var(--macos-text-secondary); }
.b-info-row .badge-ghost { color: var(--macos-text-secondary); }
.meta-group.small { display:inline-flex; align-items:center; gap:8px; padding: 0 6px; height: 20px; border: 1px solid var(--macos-separator); border-radius: 999px; background: var(--macos-background); color: var(--macos-text-secondary); font-size: var(--fs-sub); line-height: 20px; max-width: 100%; overflow: hidden; }
.meta-group.small .item { display:inline-flex; align-items:center; gap: 4px; font-size: 11.5px; line-height: 20px; min-width: 0; }
.meta-group.small .item > span { display:inline-flex; align-items:center; line-height: 20px; }
/* 限制“上次同步结果”消息长度，长文本省略，避免挤压其它元素 */
.meta-group.small .item .one-line { max-width: 260px; }
.meta-group.small .badge { border: 0; background: transparent; height: auto; padding: 0; font-size: 11.5px; }
.meta-group.small .badge-success { color: var(--macos-success-text); }
.meta-group.small .badge-error { color: var(--macos-danger-text); }
.meta-group.small .badge-warning { color: #ff9f0a; }
.meta-group.small .badge-primary { color: var(--macos-blue); }
.meta-group.small .badge-info { color: var(--macos-text-secondary); }
.meta-group.small .badge-ghost { color: var(--macos-text-secondary); }
.meta-group.small .divider-v { width: 1px; height: 12px; background: var(--macos-divider-weak); }
.sync-text { font-size: var(--fs-sub); margin-left: 6px; }
.ops { display:flex; align-items:center; gap: 6px; }
.ops .ops-actions { overflow: visible; display: inline-flex; align-items: center; }
.ops .ops-actions .seg-item { min-width: 28px; height: 28px; padding: 0 8px; display: inline-flex; align-items: center; justify-content: center; gap: 4px; position: relative; overflow: hidden; line-height: 1; }
.ops .ops-actions .seg-item .icon { transition: transform .18s ease; }
.ops .ops-actions .seg-item .label { max-width: 0; opacity: 0; transform: translateX(14px); transition: max-width .22s ease, opacity .22s ease, transform .22s ease, color .12s ease; white-space: nowrap; color: var(--macos-text-secondary); }
.ops .ops-actions .seg-item:hover .label,
.ops .ops-actions .seg-item:active .label,
.ops .ops-actions .seg-item.working .label { max-width: 220px; opacity: 1; transform: translateX(0); color: var(--macos-blue); }
.ops .ops-actions .seg-item:hover .icon { transform: translateX(-2px); }
.icon-ghost { width: 28px; height: 28px; display:inline-flex; align-items:center; justify-content:center; border-radius: 6px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-secondary); }
.ops .icon-tonal { width: 28px; height: 28px; display:inline-flex; align-items:center; justify-content:center; border-radius: 6px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-secondary); }
.ops .icon-tonal.yt { border-color: var(--macos-blue); color: var(--macos-blue); background: color-mix(in oklab, var(--macos-blue) 10%, transparent); }
/* removed dreamcreator button styling */
.header-meta .icon-tonal { width: 28px; height: 28px; display:inline-flex; align-items:center; justify-content:center; border-radius: 6px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-secondary); margin-left: 8px; }
.syncing-indicator { display:flex; align-items:center; gap: 6px; color: var(--macos-text-secondary); white-space: nowrap; word-break: keep-all; flex-wrap: nowrap; }
.syncing-text { font-size: var(--fs-sub); white-space: nowrap; }
@keyframes sync-rotate { to { transform: rotate(360deg); } }
.spin-smooth { animation: sync-rotate .8s linear infinite; will-change: transform; }
.table-wrap { overflow-x: auto; }
/* fixed layout and proportional widths */
.ck-table { table-layout: fixed; font-size: var(--fs-sub); }
.ck-table .col-domain { width: 22%; }
.ck-table .col-name { width: 28%; }
.ck-table .col-value { width: auto; }
thead { background: var(--macos-background-secondary); }
.th { text-align: left; font-weight: 600; color: var(--macos-text-secondary); padding: 6px 8px; font-size: 11.5px; }
.td { padding: 6px 8px; border-top: 1px solid var(--macos-divider-weak); color: var(--macos-text-primary); vertical-align: top; font-size: var(--fs-sub); }
.mono { font-family: var(--font-mono); }
.mono { letter-spacing: 0.1px; font-variant-ligatures: none; }
.row:hover { background: var(--macos-gray-hover); }
.text-secondary { color: var(--macos-text-secondary); }
.one-line { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.spinning { animation: macos-spin .6s ease-in-out both; }

/* cell popover */
.rel { position: relative; cursor: zoom-in; }
.cell-pop { position: absolute; left: 10px; right: 10px; top: calc(100% + 4px); z-index: 10; background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 8px; box-shadow: var(--macos-shadow-2); padding: 8px; }
.cell-pop .pop-text { max-height: 160px; overflow: auto; }
.cell-pop .pop-ops { display:flex; justify-content:flex-end; margin-top: 6px; }
.load-more { padding: 8px 10px; display:flex; justify-content:center; border-top: 1px dashed var(--macos-divider-weak); }

/* sync status pill */
.status-pill { display:inline-flex; align-items:center; gap:8px; height: 20px; padding: 0 6px; border-radius: 999px; border: 1px solid var(--macos-separator); font-size: var(--fs-sub); background: var(--macos-background); }
.status-pill.pill-success { border-color: var(--macos-success-text); color: var(--macos-success-text); background: var(--macos-background); }
.status-pill.pill-error { border-color: var(--macos-danger-text); color: var(--macos-danger-text); background: var(--macos-background); }
.status-pill.pill-neutral { color: var(--macos-text-secondary); background: var(--macos-background); }

/* 防止小图标影响基线，保证垂直居中 */
.status-pill .w-3.5, .status-pill .h-3.5, .meta-group.small .w-3.5, .meta-group.small .h-3.5 { display:block; }

/* browser segmented same height as search */
.browser-seg { height: 28px; display:inline-flex; align-items:center; }
.browser-seg .seg-item { height: 28px; font-size: var(--fs-sub); }

/* brand accents (subtle) */
.brand-chrome .bicon { border-color: #1a73e8; color: #1a73e8; background: color-mix(in oklab, #1a73e8 10%, var(--macos-background)); }
.brand-firefox .bicon { border-color: #ff9500; color: #ff9500; background: color-mix(in oklab, #ff9500 10%, var(--macos-background)); }
.brand-safari .bicon { border-color: #0fb5ee; color: #0fb5ee; background: color-mix(in oklab, #0fb5ee 10%, var(--macos-background)); }
.brand-edge .bicon { border-color: #0b84ed; color: #0b84ed; background: color-mix(in oklab, #0b84ed 10%, var(--macos-background)); }
.brand-opera .bicon { border-color: #ff1b2d; color: #ff1b2d; background: color-mix(in oklab, #ff1b2d 10%, var(--macos-background)); }
.brand-chrome .mini-icon { color: #1a73e8; }
.brand-firefox .mini-icon { color: #ff9500; }
.brand-safari .mini-icon { color: #0fb5ee; }
.brand-edge .mini-icon { color: #0b84ed; }
.brand-opera .mini-icon { color: #ff1b2d; }
</style>
