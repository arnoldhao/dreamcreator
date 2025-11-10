<script setup>
import { computed, onMounted, onUnmounted, reactive, ref, watch } from 'vue'
import eventBus from '@/utils/eventBus.js'
import Ribbon from '@/components/sidebar/Ribbon.vue'
import useNavStore from '@/stores/nav.js'
import usePreferencesStore from '@/stores/preferences.js'
import ToolbarControlWidget from '@/components/common/ToolbarControlWidget.vue'
import ToolbarTrafficLights from '@/components/common/ToolbarTrafficLights.vue'
// Sidebar toggle uses semantic icons now
import useLayoutStore from '@/stores/layout.js'
import Inspector from '@/components/inspector/Inspector.vue'
import useInspectorStore from '@/stores/inspector.js'
import { EventsOn, WindowIsFullscreen, WindowIsMaximised, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'
import { isMacOS, isWindows } from '@/utils/platform.js'
import VideoDownloadPage from "@/components/content/VideoDownloadPage.vue";
import Subtitle from '@/components/content/Subtitle.vue';
import { useSubtitleStore } from '@/stores/subtitle.js'
import { subtitleService } from '@/services/subtitleService.js'
import { useI18n } from 'vue-i18n'
import useSettingsStore from '@/stores/settings.js'
import Dependency from '@/components/content/Dependency.vue'
import Settings from '@/components/content/Settings.vue'
import Providers from '@/components/content/Providers.vue'

const props = defineProps({
  loading: Boolean,
})

const data = reactive({})

const navStore = useNavStore()
const prefStore = usePreferencesStore()
const layout = useLayoutStore()
const inspector = useInspectorStore()
const subtitleStore = useSubtitleStore()
const { t } = useI18n()
const settingsStore = useSettingsStore()
// download toolbar search
const downloadNewSearch = ref('')
const subtitleSearch = ref('')
const searchFocused = ref(false)
function emitDownloadNewSearch() { eventBus.emit('download:search', downloadNewSearch.value) }
function emitDownloadNewRefresh() { eventBus.emit('download:refresh') }
function onDownloadSearchFocus() {
  searchFocused.value = true
  if (inspector.visible) inspector.close()
}
function emitSubtitleSearch() { eventBus.emit('subtitle:search', subtitleSearch.value) }
function onSubtitleSearchFocus() {
  searchFocused.value = true
  if (inspector.visible) inspector.close()
}

// Current metrics standard info for subtitle edit header button
const metricsStandardKey = computed(() => {
  try {
    const proj = subtitleStore.currentProject
    const lang = subtitleStore.currentLanguage
    if (!proj || !lang) return null
    const segs = Array.isArray(proj.segments) ? proj.segments : []
    const first = segs.find(s => s?.languages && s.languages[lang]) || segs[0]
    const gs = first?.guideline_standard || {}
    return gs[lang] || null
  } catch { return null }
})
const metricsStandardName = computed(() => {
  const m = metricsStandardKey.value
  const map = { netflix: 'Netflix', bbc: 'BBC', ade: 'ADE' }
  return m ? (map[m] || String(m).toUpperCase()) : ''
})
const metricsStandardDesc = computed(() => {
  const m = metricsStandardKey.value
  if (!m) return ''
  const d = {
    netflix: t('subtitle.list.netflix_standard_desc'),
    bbc: t('subtitle.list.bbc_standard_desc'),
    ade: t('subtitle.list.ade_standard_desc'),
  }
  return d[m] || ''
})

// Toolbar CTA buttons (primary emphasis) for key modal actions
const primaryModalActions = new Set(['download:new-task', 'subtitle:open-file'])

// 动态内容标题：在设置页显示当前子页标题
const contentTitle = computed(() => {
  if (navStore.currentNav === navStore.navOptions.SETTINGS) {
    if (settingsStore.currentPage === 'dependency') return t('settings.dependency.title')
    if (settingsStore.currentPage === 'about') return t('settings.about.title')
    return t('settings.general.name')
  }
  if (navStore.currentNav === navStore.navOptions.PROVIDERS) {
    return t('settings.model_provider')
  }
  return t('ribbon.' + navStore.currentNav)
})

// Inline project name editing for Subtitle Edit page
const editingProjectName = ref(false)
const tempProjectName = ref('')
function beginEditProjectName() {
  tempProjectName.value = subtitleStore.currentProject?.project_name || ''
  editingProjectName.value = true
}
function cancelEditProjectName() {
  editingProjectName.value = false
  tempProjectName.value = ''
}
async function saveEditProjectName() {
  const name = (tempProjectName.value || '').trim()
  if (!name) { $message?.warning?.(t('subtitle.common.project_name') + ' ' + (t('common.not_set') || '')); return }
  try {
    const result = await subtitleService.saveProjectName(name)
    if (result?.success && result.data) {
      let projectData = result.data
      if (typeof projectData === 'string') {
        try {
          projectData = JSON.parse(projectData)
        } catch (err) {
          console.warn('Failed to parse project data after renaming:', err)
          projectData = null
        }
      }
      if (projectData) {
        subtitleStore.setCurrentProject(projectData)
        try { subtitleStore.updateProject(projectData) } catch (err) { console.warn('Failed to update project list after renaming:', err) }
      }
    }
    editingProjectName.value = false
  } catch (e) { $message?.error?.(e?.message || String(e)) }
}

const logoWrapperWidth = computed(() => {
  const left = layout.ribbonVisible ? layout.ribbonWidth : 0
  const right = inspector.visible ? layout.inspectorWidth : 0
  return `${left + right}px`
})

const leftCapWidth = computed(() => {
  // When ribbon collapsed, reserve space for traffic lights + ribbon toggle (approx 40px)
  const safe = logoPaddingLeft.value + 40
  const w = layout.ribbonVisible ? layout.ribbonWidth : safe
  return `${w}px`
})

// UI style: frosted/classic comes from preferences
const uiFrosted = computed(() => (prefStore?.general?.uiStyle || 'frosted') === 'frosted')
const isDarkMode = computed(() => !!prefStore?.isDark)

// ribbon/left-cap 的毛玻璃底色，按明暗主题区分
const ribbonFrostedBg = computed(() => isDarkMode.value ? 'rgba(0,0,0,0.28)' : 'rgba(255,255,255,0.28)')

// Left cap dynamic style: in frosted mode we paint background via a unified toolbar stripe overlay to avoid flicker
const leftCapStyle = computed(() => {
  const base = { width: leftCapWidth.value }
  if (!layout.ribbonVisible) {
    // Let the parent toolbar background paint this area for a perfect match
    return {
      ...base,
      borderRight: 'none',
      background: 'transparent',
      backdropFilter: 'none',
      WebkitBackdropFilter: 'none'
    }
  }
  // ribbon visible
  if (uiFrosted.value) {
    // Background is painted by toolbar-left-stripe overlay; keep left cap transparent but with divider
    return {
      ...base,
      borderRight: '1px solid var(--macos-divider-weak)',
      background: 'transparent',
      backdropFilter: 'none',
      WebkitBackdropFilter: 'none',
      isolation: 'isolate',
      mixBlendMode: 'normal'
    }
  }
  // classic
  return {
    ...base,
    borderRight: '1px solid var(--macos-divider-weak)',
    background: 'var(--sidebar-bg)'
  }
})

// Opaque layer to repaint middle+right backgrounds when opening a transparent hole under ribbon
const opaqueLayerStyle = computed(() => {
  if (!uiFrosted.value) return { display: 'none' }
  // Offset equals ribbon width when visible, otherwise 0
  const left = layout.ribbonVisible ? layout.ribbonWidth : 0
  return { left: left + 'px' }
})

// Background for the whole window: make only the left ribbon stripe transparent in frosted mode
const windowBgVars = computed(() => {
  if (!uiFrosted.value || !layout.ribbonVisible) return {}
  const w = Math.max(0, Number(layout.ribbonWidth) || 0)
  const bg = `linear-gradient(to right, transparent 0, transparent ${w}px, var(--macos-background) ${w}px, var(--macos-background) 100%)`
  return { '--window-bg': bg }
})

// Toolbar painting: remove toolbar's own frosted background under left cap and repaint middle+right
const toolbarBgOverride = computed(() => {
  // classic: 使用默认样式
  if (!uiFrosted.value) return {}
  // frosted + ribbon 可见：清空 toolbar 自身底色，由我们分区绘制（左透右不透）
  if (layout.ribbonVisible) {
    return { background: 'transparent', backdropFilter: 'none', WebkitBackdropFilter: 'none' }
  }
  // frosted + ribbon 关闭：标题栏整体不透明，颜色与内容保持一致
  return { background: 'var(--macos-surface-opaque)', backdropFilter: 'none', WebkitBackdropFilter: 'none' }
})

const toolbarLayerStyle = computed(() => {
  if (!uiFrosted.value || !layout.ribbonVisible) return { display: 'none' }
  return { left: (layout.ribbonWidth || 0) + 'px' }
})

// Unified left frosted stripe for toolbar to match ribbon area; animates as one block
const toolbarLeftStripeStyle = computed(() => {
  if (!uiFrosted.value) return { display: 'none' }
  const bg = layout.ribbonVisible ? ribbonFrostedBg.value : 'var(--macos-surface-opaque)'
  const blur = layout.ribbonVisible ? 'var(--macos-surface-blur)' : 'none'
  return {
    width: (layout.ribbonVisible ? (layout.ribbonWidth || 0) + 'px' : '0px'),
    background: bg,
    backdropFilter: blur,
    WebkitBackdropFilter: blur,
  }
})

const logoPaddingLeft = ref(10)
// Top fade under toolbar when content scrolls
const pageScrollEl = ref(null)
const hasTopScroll = ref(false)
let scrollEl = null
const onScroll = () => { try { hasTopScroll.value = (scrollEl?.scrollTop || 0) > 0 } catch {} }
const maximised = ref(false)
const hideRadius = ref(false)
// 使用系统原生的交通灯外观，不再自绘占位

const onToggleFullscreen = (fullscreen) => {
  hideRadius.value = fullscreen
  if (fullscreen) {
    logoPaddingLeft.value = 10
  } else {
    // Reserve space for traffic lights on both macOS (native) and Windows (custom)
    logoPaddingLeft.value = (isMacOS() || isWindows()) ? 70 : 10
  }
}

const onToggleMaximize = (isMaximised) => {
  if (isMaximised) {
    maximised.value = true
    if (!isMacOS()) {
      hideRadius.value = true
    }
  } else {
    maximised.value = false
    if (!isMacOS()) {
      hideRadius.value = false
    }
  }
}

EventsOn('window_changed', (info) => {
  const { fullscreen, maximised } = info
  onToggleFullscreen(fullscreen === true)
  onToggleMaximize(maximised)
})

onMounted(async () => {
  const fullscreen = await WindowIsFullscreen()
  onToggleFullscreen(fullscreen === true)
  const maximised = await WindowIsMaximised()
  onToggleMaximize(maximised)

  // initialize inspector actions for current page
  inspector.setActions(panelActions.value)

  // observe scroll to toggle top fade under toolbar
  scrollEl = pageScrollEl.value
  if (scrollEl) {
    scrollEl.addEventListener('scroll', onScroll, { passive: true })
    // init
    onScroll()
  }
})

onUnmounted(() => { try { scrollEl && scrollEl.removeEventListener('scroll', onScroll) } catch {} })

// Update actions when page changes
watch(() => navStore.currentNav, () => {
  inspector.setActions(panelActions.value)
  // If current panel no longer available, fallback to first action
  const acts = panelActions.value
  if (!acts || !acts.some(a => a.key === inspector.panel)) {
    inspector.close()
  }
})

// Also react to subtitle project availability (affects modal actions like Add Language)
watch(() => subtitleStore.currentProject, () => {
  inspector.setActions(panelActions.value)
})

// Make detail icon reactive when inspector opens/closes or switches panel
watch(() => [inspector.visible, inspector.panel], () => {
  inspector.setActions(panelActions.value)
})

// page-specific actions definition
const pageActions = computed(() => getPageActions())
const modalActions = computed(() => pageActions.value.modals)
const panelActions = computed(() => pageActions.value.panels)
// Add bottom padding in subtitle edit to avoid floating buttons overlap
const needsBottomPad = computed(() => navStore.currentNav === navStore.navOptions.SUBTITLE)

function getPageActions() {
  // make icons reactive to inspector state for better affordance
  const isDetailOpen = inspector.visible && inspector.panel === 'DownloadTaskPanel'
  if (navStore.currentNav === navStore.navOptions.DOWNLOAD) {
    return {
      modals: [
        { key: 'download:new-task', icon: 'plus', titleKey: 'download.new_task' },
      ],
      panels: [
        { key: 'CookiesPanel', icon: 'leaf', titleKey: 'cookies.title' },
        // Use a steady, semantic icon that represents a right-side detail pane
        { key: 'DownloadTaskPanel', icon: 'panel-right', titleKey: 'download.detail' },
      ],
    }
  }
  if (navStore.currentNav === navStore.navOptions.SETTINGS) {
    // Provide actions for specific settings subpages
    if (settingsStore.currentPage === settingsStore.settingsOptions.DEPENDENCY) {
      return {
        modals: [
          { key: 'dependency:quick-validate', icon: 'search-check', titleKey: 'settings.dependency.quick_validate' },
          { key: 'dependency:validate', icon: 'shield-check', titleKey: 'settings.dependency.validate' },
          { key: 'dependency:check-updates', icon: 'refresh', titleKey: 'settings.dependency.check_updates' },
        ],
        panels: [],
      }
    }
    return { modals: [], panels: [] }
  }
  if (navStore.currentNav === navStore.navOptions.SUBTITLE) {
    const modals = [
      { key: 'subtitle:open-file', icon: 'file-plus', titleKey: 'subtitle.common.open_file' },
    ]

    if (subtitleStore.currentProject) {
      modals.push({ key: 'subtitle:metrics', icon: 'info', titleKey: 'subtitle.list.metrics_explanation' })
      modals.push({ key: 'subtitle:back-home', icon: 'home', titleKey: 'ribbon.subtitle' })
    }

    return {
      modals,
      // show export as panel action when a project is open
      panels: [
        ... (subtitleStore.currentProject ? [{ key: 'SubtitleExportPanel', icon: 'download-file', titleKey: 'subtitle.export.title' }] : []),
      ],
    }
  }
  return { modals: [], panels: [] }
}

function onActionClick(act) {
  // toggle/switch behavior
  // DOWNLOAD uses page-level handlers to respect default/selection logic
  if (navStore.currentNav === navStore.navOptions.DOWNLOAD) {
    if (act.key === 'CookiesPanel') { eventBus.emit('download:toggle-cookies'); return }
    if (act.key === 'DownloadTaskPanel') { eventBus.emit('download:toggle-detail'); return }
  }
  if (inspector.visible && inspector.panel === act.key) {
    inspector.close(); return
  }
  inspector.open(act.key, t(act.titleKey) || act.titleKey)
  inspector.setActions(panelActions.value)
}

function onInspectorAction(key) {
  // clicking icon in inspector header switches/close similarly
  if (key === 'inspector:close') { inspector.close(); return }
  if (key === 'download:refresh') { eventBus.emit('download_task:refresh'); return }
  // page-specific handling for DOWNLOAD to route cookies/detail via page logic
  if (navStore.currentNav === navStore.navOptions.DOWNLOAD) {
    if (key === 'CookiesPanel') { eventBus.emit('download:toggle-cookies'); return }
    if (key === 'DownloadTaskPanel') { eventBus.emit('download:toggle-detail'); return }
  }
  const act = panelActions.value.find(a => a.key === key)
  if (!act) return
  if (inspector.panel === key) {
    inspector.close()
  } else {
    inspector.open(key, t(act.titleKey) || act.titleKey)
  }
}

// Active state calculation for DOWNLOAD icons
function isDownloadNewActive(key) {
  if (!inspector.visible) return false
  if (key === 'CookiesPanel') return inspector.panel === 'CookiesPanel'
  if (key === 'DownloadTaskPanel') {
    // Active when showing Detail panel OR default (home) with detail title
    return inspector.panel === 'DownloadTaskPanel' || (inspector.panel === 'InspectorHomePanel')
      && (inspector.title === (t('download.detail') || ''))
  }
  return inspector.panel === key
}

function onModalClick(act) {
  // Forward to pages via event bus
  if (act.key === 'SubtitleExportPanel') {
    inspector.open('SubtitleExportPanel', t('subtitle.export.title'))
    inspector.setActions(panelActions.value)
    return
  }
  eventBus.emit(act.key)
}

</script>

<template>
  <!-- app content-->
  <div class="relative min-h-screen macos-window" :class="[{ 'loading': props.loading }]" :style="windowBgVars">
    <div id="app-content-wrapper" class="flex flex-col h-screen"
      :class="[ hideRadius ? '' : 'rounded-xl', isWindows() ? '' : 'overflow-hidden' ]">
      <!-- title bar -->
      <div id="app-toolbar" class="macos-toolbar w-full" style="--wails-draggable: drag" :style="toolbarBgOverride"
        @dblclick="WindowToggleMaximise">
        <!-- Unified left frosted stripe to match ribbon; animates with left cap width -->
        <div v-if="uiFrosted" class="toolbar-left-stripe" :style="toolbarLeftStripeStyle"></div>
        <!-- Paint-only layer for toolbar middle+right when frosted: keeps them opaque while left cap stays translucent -->
        <div v-if="uiFrosted && layout.ribbonVisible" class="toolbar-opaque-layer" :style="toolbarLayerStyle"></div>
        <!-- left cap to extend sidebar background and host the ribbon toggle -->
        <div class="macos-toolbar-leftcap"
             :style="leftCapStyle">
          <!-- Windows: render mac-style traffic lights at top-left -->
          <div v-if="isWindows()" class="no-drag" style="position:absolute; left:8px; top:0; bottom:0; display:flex; align-items:center;">
            <ToolbarTrafficLights />
          </div>
          <div class="no-drag h-full flex items-center"
               :style="{ paddingLeft: (logoPaddingLeft + 6) + 'px' }">
            <button class="toolbar-chip" @click.stop="layout.toggleRibbon()"
                    :data-tooltip="layout.ribbonVisible ? t('sidebar.close_sidebar') : t('sidebar.open_sidebar')"
                    :aria-label="layout.ribbonVisible ? t('sidebar.close_sidebar') : t('sidebar.open_sidebar')">
              <Icon :name="layout.ribbonVisible ? 'panel-left-close' : 'panel-left-open'" class="w-4 h-4" />
            </button>
          </div>
        </div>

        <!-- middle area: content title (left) + actions (right) -->
        <div id="app-toolbar-center" class="flex-1 flex items-center justify-between px-4 relative min-w-0">
          <div class="title-strong capitalize flex items-center gap-2 min-w-0">
            <span class="shrink-0">{{ contentTitle }}</span>
            <!-- Show editable project name on subtitle edit page
                 Hide when export inspector is open to prevent layout squeeze -->
            <template
              v-if="navStore.currentNav === navStore.navOptions.SUBTITLE
                     && subtitleStore.currentProject
                     && !(inspector.visible && inspector.panel === 'SubtitleExportPanel')">
              <span class="text-[var(--macos-text-tertiary)]">—</span>
              <div class="project-inline min-w-0">
                <template v-if="!editingProjectName">
                  <span class="pill one-line" :title="subtitleStore.currentProject?.project_name || '-'">{{ subtitleStore.currentProject?.project_name || '-' }}</span>
                  <button class="toolbar-chip" :data-tooltip="$t('common.edit')" data-tip-pos="top" @click="beginEditProjectName"><Icon name="edit" class="w-4 h-4" /></button>
                </template>
                <template v-else>
                  <input v-model="tempProjectName" class="inline-edit pill-input" @keydown.enter.stop.prevent="saveEditProjectName" @keydown.esc.stop.prevent="cancelEditProjectName" />
                  <button class="toolbar-chip" :data-tooltip="$t('common.confirm')" data-tip-pos="top" @click="saveEditProjectName"><Icon name="status-success" class="w-4 h-4" /></button>
                  <button class="toolbar-chip" :data-tooltip="$t('common.cancel')" data-tip-pos="top" @click="cancelEditProjectName"><Icon name="close" class="w-4 h-4" /></button>
                </template>
              </div>
            </template>
          </div>
          <!-- centered search for DOWNLOAD -->
          <div v-if="navStore.currentNav === navStore.navOptions.DOWNLOAD" class="toolbar-center-search">
            <input v-model="downloadNewSearch" type="text" class="input-macos h-[26px] px-2 text-sm"
                   :placeholder="$t('sidebar.search_placeholder')"
                   @focus="onDownloadSearchFocus" @blur="searchFocused = false"
                   @input="emitDownloadNewSearch"
                   :style="{ width: (searchFocused ? 320 : 200) + 'px', transition: 'width 120ms ease' }" />
          </div>
          <!-- centered search for SUBTITLE: only on home (no project) -->
          <div v-else-if="navStore.currentNav === navStore.navOptions.SUBTITLE && !subtitleStore.currentProject" class="toolbar-center-search">
            <input v-model="subtitleSearch" type="text" class="input-macos h-[26px] px-2 text-sm"
                   :placeholder="$t('sidebar.search_placeholder')"
                   @focus="onSubtitleSearchFocus" @blur="searchFocused = false"
                   @input="emitSubtitleSearch"
                   :style="{ width: (searchFocused ? 320 : 200) + 'px', transition: 'width 120ms ease' }" />
          </div>
          <div class="flex items-center gap-2">
            <!-- modal actions (always visible) -->
              <template v-for="act in modalActions" :key="act.key">
                <!-- Replace subtitle metrics button with current standard chip when in subtitle edit -->
                <button v-if="act.key === 'subtitle:metrics' && subtitleStore.currentProject"
                        class="chip-frosted chip-sm chip-translucent"
                        :data-tooltip="$t(act.titleKey)"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <span class="chip-label">{{ metricsStandardName || $t('subtitle.list.metrics_explanation') }}</span>
                </button>
                <button v-else-if="primaryModalActions.has(act.key)"
                        class="chip-frosted chip-sm chip-primary-action"
                        :data-tooltip="$t(act.titleKey)"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <Icon :name="act.icon" class="chip-icon" />
                  <span class="chip-label">{{ $t(act.titleKey) }}</span>
                </button>
                <button v-else class="toolbar-chip"
                        :data-tooltip="$t(act.titleKey)" :aria-label="$t(act.titleKey)"
                        :data-tip-align="act.key === 'dependency:check-updates' ? 'right' : null"
                        @click="onModalClick(act)">
                  <Icon :name="act.icon" class="w-4 h-4" />
                </button>
              </template>
            <!-- panel actions (only when sidebar closed) -->
            <template v-if="!inspector.visible">
              <div class="w-px h-4 bg-[var(--macos-divider-weak)] mx-1" v-if="panelActions.length && modalActions.length"></div>
              <button v-for="act in panelActions" :key="act.key" class="toolbar-chip"
                :data-tooltip="$t(act.titleKey)" :aria-label="$t(act.titleKey)" @click="onActionClick(act)">
                <Icon :name="act.icon" class="w-4 h-4" />
              </button>
            </template>
            <!-- DOWNLOAD removed right-side search & moved refresh to floating panel -->
          </div>
        </div>

        <!-- right cap aligns with inspector and hosts its title + actions -->
        <div class="macos-toolbar-rightcap" :style="{ width: (inspector.visible ? layout.inspectorWidth : 0) + 'px' }">
          <div v-if="inspector.visible" class="no-drag h-full flex items-center justify-between px-3">
            <!-- Download: title left, icons right; no close button -->
            <template v-if="navStore.currentNav === navStore.navOptions.DOWNLOAD">
              <div class="inspector-title text-xs uppercase tracking-wide text-[var(--macos-text-tertiary)]">{{ inspector.title }}</div>
              <div class="flex items-center gap-2">
                <button v-for="act in inspector.actions" :key="act.key" class="toolbar-chip"
                  :aria-label="$t(act.titleKey)" :class="{ active: isDownloadNewActive(act.key) }"
                  @click="onInspectorAction(act.key)">
                  <Icon :name="act.icon" class="w-4 h-4" />
                </button>
              </div>
            </template>
            <!-- Subtitle pages: show actions like Download (no close) -->
            <template v-else-if="navStore.currentNav === navStore.navOptions.SUBTITLE">
              <div class="inspector-title text-xs uppercase tracking-wide text-[var(--macos-text-tertiary)]">{{ inspector.title }}</div>
              <div class="flex items-center gap-2">
                <button v-for="act in inspector.actions" :key="act.key" class="toolbar-chip"
                  :aria-label="$t(act.titleKey)" :class="{ active: inspector.panel === act.key }"
                  @click="onInspectorAction(act.key)">
                  <Icon :name="act.icon" class="w-4 h-4" />
                </button>
              </div>
            </template>
            <!-- Other pages: title left, icons right, with close button -->
            <template v-else>
              <div class="inspector-title text-xs uppercase tracking-wide text-[var(--macos-text-tertiary)]">{{ inspector.title }}</div>
              <div class="flex items-center gap-2">
                <button v-for="act in inspector.actions" :key="act.key" class="toolbar-chip"
                  :aria-label="$t(act.titleKey)" :class="{ active: inspector.panel === act.key }"
                  @click="onInspectorAction(act.key)">
                  <Icon :name="act.icon" class="w-4 h-4" />
                </button>
                <button class="toolbar-chip" :data-tooltip="$t('common.close')" @click="inspector.close()">
                  <Icon name="close" class="w-4 h-4" />
                </button>
              </div>
            </template>
          </div>
        </div>

        <!-- hide default Windows right-side controls; using mac-style traffic lights instead -->
        <div class="no-drag flex items-center gap-2 pr-2" v-if="false">
          <toolbar-control-widget :maximised="maximised" :size="38" />
        </div>
      </div>

      <!-- content -->
      <div id="app-content" class="flex flex-1 min-h-0 overflow-hidden macos-content" style="--wails-draggable: none" :style="uiFrosted ? { '--window-bg': 'transparent', '--content-bg': 'transparent' } : {}">
        <!-- Paint-only layer for middle+right when frosted: keeps them opaque while ribbon stays translucent -->
        <div v-if="uiFrosted" class="content-opaque-layer" :style="opaqueLayerStyle"></div>
        <!-- left ribbon (collapsible) -->
        <div class="collapsible-left" :style="{ width: (layout.ribbonVisible ? layout.ribbonWidth : 0) + 'px' }">
          <ribbon v-model:value="navStore.currentNav" :width="layout.ribbonWidth" />
        </div>

        <!-- content column: includes local header when ribbon is visible -->
        <div class="flex flex-col flex-1 min-h-0 min-w-0">
          <div ref="pageScrollEl" class="page-scroll flex-1 min-h-0 overflow-auto"
               :class="[{ 'with-top-divider': (navStore.currentNav === navStore.navOptions.SETTINGS) || (navStore.currentNav === navStore.navOptions.PROVIDERS) }, { 'scrolled': hasTopScroll }, { 'pad-bottom-for-fab': needsBottomPad }]">
            <!-- download page (macOS style) -->
            <div v-show="navStore.currentNav === navStore.navOptions.DOWNLOAD" class="content-container min-w-0">
              <video-download-page />
            </div>

            <!-- subtitle page -->
            <div v-if="navStore.currentNav === navStore.navOptions.SUBTITLE" class="content-container min-w-0">
              <subtitle />
            </div>

            <!-- providers page (1:1 复刻 Settings 的承载容器) -->
            <div v-if="navStore.currentNav === navStore.navOptions.PROVIDERS" class="content-container min-w-0 flex flex-1 min-h-0 relative">
              <div class="settings-host with-split" :style="{ '--settings-left': layout.ribbonWidth + 'px' }">
                <Providers />
              </div>
            </div>

            <!-- settings page -->
            <div v-show="navStore.currentNav === navStore.navOptions.SETTINGS" class="content-container min-w-0 flex flex-1 min-h-0 relative">
              <div class="settings-host" :class="{ 'with-split': [settingsStore.settingsOptions.GENERAL, settingsStore.settingsOptions.ABOUT].includes(settingsStore.currentPage) }" :style="{ '--settings-left': layout.ribbonWidth + 'px' }">
                <component :is="{
                  general: Settings,
                  dependency: Dependency,
                  about: Settings,
                }[settingsStore.currentPage] || Settings" />
              </div>
            </div>
          </div>
        </div>

        <!-- right inspector (collapsible) -->
        <div class="collapsible-right" :style="{ width: (inspector.visible ? layout.inspectorWidth : 0) + 'px' }">
          <Inspector />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.loading {
  @apply animate-pulse;
}

/* Repaint layer for middle+right when frosted UI is active */
.content-opaque-layer {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  left: 0; /* will be overridden by inline style to ribbon width */
  background: var(--macos-background);
  z-index: -1;
  pointer-events: none; /* do not intercept input */
  transition: left 180ms ease;
}

/* Toolbar repaint layer for middle+right in frosted mode */
.toolbar-opaque-layer {
  position: absolute;
  top: 0;
  bottom: 0;
  right: 0;
  left: 0; /* overridden by inline style to leftCapWidth */
  background: var(--macos-surface);
  -webkit-backdrop-filter: var(--macos-surface-blur);
  backdrop-filter: var(--macos-surface-blur);
  z-index: -1;
  pointer-events: none;
  transition: left 180ms ease;
}

/* Left frosted stripe for toolbar to visually unify with ribbon */
.toolbar-left-stripe {
  position: absolute;
  top: 0;
  bottom: 0;
  left: 0;
  width: 0;
  background: transparent;
  -webkit-backdrop-filter: none;
  backdrop-filter: none;
  z-index: -1;
  pointer-events: none;
  transition: width 180ms ease, background 180ms ease, backdrop-filter 180ms ease;
}

/* Anchor the opaque layer to content bounds */
#app-content { position: relative; }

.content-container {
  border-top-left-radius: 0;
  min-height: 100%;
}

.with-left-divider {
  border-left: 1px solid var(--macos-divider-weak);
}

/* add subtle divider between toolbar title and content */
#app-content .with-top-divider { border-top: 1px solid var(--macos-divider-weak); }
#app-content .page-scroll { padding: 0 !important; }
#app-content .content-container { padding: 0 !important; }
#app-content .settings-host { position: absolute; inset: 0; }
#app-content .with-split.settings-host::before { content: ''; position: absolute; inset: 0 auto 0 0; width: var(--settings-left, 160px); background: var(--sidebar-bg); }
#app-content .with-split.settings-host::after { content: ''; position: absolute; top: 0; bottom: 0; left: var(--settings-left, 160px); width: 1px; background: var(--macos-divider-weak); }

/* Prevent inspector title wrapping (keep single line with ellipsis) */
.inspector-title { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
/* centered search overlay */
.toolbar-center-search { position: absolute; left: 0; right: 0; text-align: center; pointer-events: none; }
.toolbar-center-search input { pointer-events: auto; }

/* Inline project name editor in toolbar */
.project-inline { display:inline-flex; align-items:center; gap:6px; max-width: min(70%, 480px); }
.project-inline .pill { display:inline-block; max-width: min(65vw, 420px); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--macos-text-primary); font-size: var(--fs-sub); font-weight: 400; height: 22px; line-height: 22px; padding: 0 8px; border: 1px solid var(--macos-separator); border-radius: 999px; background: var(--macos-background); }
.project-inline .pill-input { height: 22px; padding: 0 8px; border-radius: 999px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-primary); font-size: var(--fs-sub); min-width: 200px; max-width: min(65vw, 420px); }

/* Toolbar primary CTA chips for modal actions */
[data-ui='frosted'] .chip-frosted.chip-primary-action {
  background: color-mix(in oklab, var(--macos-blue) 82%, transparent);
  border-color: color-mix(in oklab, var(--macos-blue) 68%, white 22%);
  color: #fff;
  font-weight: 600;
}
[data-ui='frosted'] .chip-frosted.chip-primary-action:hover {
  background: color-mix(in oklab, var(--macos-blue) 90%, white 8%);
  border-color: color-mix(in oklab, var(--macos-blue) 76%, white 18%);
}
[data-ui='frosted'] .chip-frosted.chip-primary-action .chip-label { font-weight: 600; }
[data-ui='frosted'] .chip-frosted.chip-primary-action .chip-icon {
  width: 14px;
  height: 14px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-right: 4px;
  color: inherit;
}
@supports not ((-webkit-backdrop-filter: blur(10px)) or (backdrop-filter: blur(10px))) {
  [data-ui='frosted'] .chip-frosted.chip-primary-action {
    background: var(--macos-blue);
    border-color: var(--macos-blue);
  }
}
[data-ui='classic'] .chip-frosted.chip-primary-action {
  background: var(--macos-blue) !important;
  border-color: var(--macos-blue) !important;
  color: #fff !important;
}
[data-ui='classic'] .chip-frosted.chip-primary-action .chip-icon {
  width: 12px;
  height: 12px;
  margin-right: 4px;
  color: inherit;
}
[data-ui='classic'] .chip-frosted.chip-primary-action .chip-label { font-weight: 600; }

/* Top fade under the toolbar when scrolled */
#app-content .page-scroll { position: relative; }
#app-content .page-scroll::before {
  content: '';
  position: sticky;
  top: 0;
  display: block;
  height: 10px;
  margin-top: -10px; /* overlay without affecting layout flow */
  background: linear-gradient(to bottom, var(--macos-hover-translucent), rgba(0,0,0,0.0));
  opacity: 0;
  transition: opacity 120ms ease;
  z-index: 5; /* above sticky section headers like Today */
  pointer-events: none;
}
#app-content .page-scroll.scrolled::before { opacity: 1; }
/* Ensure content can scroll past floating controls on subtitle edit page */
#app-content .page-scroll.pad-bottom-for-fab { padding-bottom: 48px !important; }

/* Light + frosted: make left-cap ribbon toggle icon use primary text color for clarity */
[data-ui="frosted"][data-theme="light"] .macos-toolbar-leftcap .toolbar-chip { color: var(--macos-text-primary); }
</style>
