<script setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import eventBus from '@/utils/eventBus.js'
import MainSidebarMenu from "@/components/sidebar/MainSidebarMenu.vue"
import MainSidebarSettingsMenu from "@/components/sidebar/MainSidebarSettingsMenu.vue"
import useNavStore from '@/stores/nav.js'
import ToolbarControlWidget from '@/components/common/ToolbarControlWidget.vue'
import useLayoutStore from '@/stores/layout.js'
import Inspector from '@/components/inspector/Inspector.vue'
import useInspectorStore from '@/stores/inspector.js'
import { Button } from "@/components/ui/button"
import { Events, Window } from '@wailsio/runtime'
import { isMacOS, isWindows } from '@/utils/platform.js'
import VideoDownloadPage from "@/views/DownloadPage.vue";
import Subtitle from '@/views/SubtitlePage.vue';
import { useSubtitleStore } from '@/stores/subtitle.js'
import { subtitleService } from '@/services/subtitleService.js'
import { useI18n } from 'vue-i18n'
import { Sidebar, SidebarContent, SidebarFooter, SidebarHeader, SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
import { PanelLeftClose, PanelLeftOpen } from "lucide-vue-next"

const props = defineProps({
  loading: Boolean,
})

const navStore = useNavStore()
const layout = useLayoutStore()
const inspector = useInspectorStore()
const subtitleStore = useSubtitleStore()
const { t } = useI18n()
// download toolbar search
const downloadNewSearch = ref('')
const subtitleSearch = ref('')
const searchFocused = ref(false)
function emitDownloadNewSearch() { eventBus.emit('download:search', downloadNewSearch.value) }
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

// Toolbar CTA buttons (primary emphasis) for key modal actions
const primaryModalActions = new Set(['download:new-task', 'subtitle:open-file'])

const contentTitle = computed(() => {
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

// Top fade under toolbar when content scrolls
const pageScrollEl = ref(null)
const hasTopScroll = ref(false)
let scrollEl = null
const onScroll = () => { try { hasTopScroll.value = (scrollEl?.scrollTop || 0) > 0 } catch {} }
const maximised = ref(false)
const hideRadius = ref(false)
// Use system traffic lights on macOS (hidden-titlebar window)

const onToggleFullscreen = (fullscreen) => {
  hideRadius.value = fullscreen
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

const contentDragbarStyle = computed(() => {
  const base = 12
  // When the sidebar is collapsed on macOS, traffic lights move to the window top-left.
  // Reserve a safe inset so the sidebar trigger/title don't sit under the controls.
  const trafficInset = (isMacOS() && !layout.ribbonVisible) ? 70 : 0
  return { paddingLeft: `${base + trafficInset}px`, paddingRight: `${base}px` }
})

// Mirror old v2 EventsOn behaviour using the v3 Events API.
Events.On('window_changed', (ev) => {
  const info = ev?.data ?? ev
  const { fullscreen, maximised } = info || {}
  onToggleFullscreen(fullscreen === true)
  onToggleMaximize(!!maximised)
})

onMounted(async () => {
  const fullscreen = await Window.IsFullscreen()
  onToggleFullscreen(fullscreen === true)
  const maximised = await Window.IsMaximised()
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
        // Use a steady, semantic icon that represents a right-side detail pane
        { key: 'DownloadTaskPanel', icon: 'panel-right', titleKey: 'download.detail' },
      ],
    }
  }
  if (navStore.currentNav === navStore.navOptions.SUBTITLE) {
    const modals = []
    if (!subtitleStore.currentProject) {
      // Only show Open File on the all-subtitles (hub) view
      modals.push({ key: 'subtitle:open-file', icon: 'file-plus', titleKey: 'subtitle.common.open_file' })
    } else {
      // When a project is open: hide Open File; show back-home (metrics moved to bottom toolbar)
      modals.push({ key: 'subtitle:back-home', icon: 'home', titleKey: 'subtitle.all_subs' })
    }

    return {
      modals,
      // Glossary and Target Languages visible on subtitle page; Export only when a project is open
      panels: [
        { key: 'GlossaryPanel', icon: 'database', titleKey: 'glossary.title' },
        { key: 'TargetLanguagesPanel', icon: 'languages', titleKey: 'subtitle.target_languages.title' },
        { key: 'SubtitleTasksPanel', icon: 'list', titleKey: 'subtitle.tasks_title' },
        { key: 'ProfilesPanel', icon: 'layers', titleKey: 'profiles.inspector_title' },
        ... (subtitleStore.currentProject ? [
          { key: 'SubtitleExportPanel', icon: 'download-file', titleKey: 'subtitle.export.title' }
        ] : []),
      ],
    }
  }
  return { modals: [], panels: [] }
}

function onActionClick(act) {
  // toggle/switch behavior
  // DOWNLOAD uses page-level handlers to respect default/selection logic
  if (navStore.currentNav === navStore.navOptions.DOWNLOAD) {
    if (act.key === 'DownloadTaskPanel') { eventBus.emit('download:toggle-detail'); return }
  }
  if (inspector.visible && inspector.panel === act.key) {
    inspector.close(); return
  }
  inspector.open(act.key, t(act.titleKey) || act.titleKey)
  inspector.setActions(panelActions.value)
}

// Active state calculation for DOWNLOAD icons
function isDownloadNewActive(key) {
  if (!inspector.visible) return false
  if (key === 'DownloadTaskPanel') {
    // Active when showing Detail panel OR default (home) with detail title
    return inspector.panel === 'DownloadTaskPanel' || (inspector.panel === 'InspectorHomePanel')
      && (inspector.title === (t('download.detail') || ''))
  }
  return inspector.panel === key
}

function isPanelActive(key) {
  if (!inspector.visible) return false
  if (navStore.currentNav === navStore.navOptions.DOWNLOAD) return isDownloadNewActive(key)
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

// Preserve template hook name from v2 runtime
const WindowToggleMaximise = () => {
  try { Window.ToggleMaximise() } catch {}
}

</script>

<template>
  <div class="relative min-h-screen" :class="[{ 'loading': props.loading }]">
    <div
      class="h-screen w-screen"
      :class="[ hideRadius ? '' : (isMacOS() ? '' : 'rounded-xl'), isWindows() ? '' : 'overflow-hidden' ]"
    >
      <SidebarProvider
        :force-mobile="false"
        :open="layout.ribbonVisible"
        :sidebar-width="`${layout.ribbonWidth || 160}px`"
        class="h-full w-full"
        @update:open="(v) => (layout.ribbonVisible = v)"
      >
        <Sidebar variant="floating" collapsible="offcanvas">
          <!-- Sidebar header: macOS traffic-lights safe zone + collapse button -->
          <SidebarHeader class="dc-main-dragbar h-[38px] p-0" />

          <SidebarContent class="p-0 overflow-hidden">
            <MainSidebarMenu v-model:value="navStore.currentNav" />
          </SidebarContent>

          <SidebarFooter class="p-2 pt-0">
            <MainSidebarSettingsMenu />
          </SidebarFooter>
        </Sidebar>

        <SidebarInset class="flex flex-col min-h-0">
          <!-- Content header: title + actions + (Windows) window controls -->
          <div
            class="dc-main-dragbar flex h-[38px] shrink-0 items-center gap-2 relative"
            :style="contentDragbarStyle"
            @dblclick="WindowToggleMaximise"
          >
            <div class="flex items-center gap-2 min-w-0 flex-1">
              <!-- Sidebar trigger is required when sidebar is collapsed -->
              <Button
                type="button"
                variant="ghost"
                size="icon"
                class="h-7 w-7"
                :title="layout.ribbonVisible ? t('sidebar.close_sidebar') : t('sidebar.open_sidebar')"
                :aria-label="layout.ribbonVisible ? t('sidebar.close_sidebar') : t('sidebar.open_sidebar')"
                @click.stop="layout.toggleRibbon()"
              >
                <PanelLeftClose v-if="layout.ribbonVisible" class="h-4 w-4" aria-hidden="true" />
                <PanelLeftOpen v-else class="h-4 w-4" aria-hidden="true" />
              </Button>

              <div class="title-strong capitalize flex items-center gap-2 min-w-0 flex-1">
                <span class="shrink-0">{{ contentTitle }}</span>
                <template
                  v-if="navStore.currentNav === navStore.navOptions.SUBTITLE
                         && subtitleStore.currentProject
                         && !(inspector.visible && inspector.panel === 'SubtitleExportPanel')"
                >
                  <span class="text-[var(--macos-text-tertiary)]">—</span>
                  <div class="project-inline min-w-0">
                    <template v-if="!editingProjectName">
                      <span class="project-name-text" :title="subtitleStore.currentProject?.project_name || '-'">{{ subtitleStore.currentProject?.project_name || '-' }}</span>
                      <button class="btn-chip-icon btn-xxs" :data-tooltip="$t('common.edit')" data-tip-pos="bottom" @click="beginEditProjectName">
                        <Icon name="edit" class="w-3 h-3" />
                      </button>
                    </template>
                    <template v-else>
                      <input v-model="tempProjectName" class="inline-edit pill-input" @keydown.enter.stop.prevent="saveEditProjectName" @keydown.esc.stop.prevent="cancelEditProjectName" />
                      <button class="btn-chip-icon btn-xxs" :data-tooltip="$t('common.confirm')" data-tip-pos="bottom" @click="saveEditProjectName">
                        <Icon name="status-success" class="w-3 h-3" />
                      </button>
                      <button class="btn-chip-icon btn-xxs" :data-tooltip="$t('common.cancel')" data-tip-pos="bottom" @click="cancelEditProjectName">
                        <Icon name="close" class="w-3 h-3" />
                      </button>
                    </template>
                  </div>
                </template>
              </div>
            </div>

            <!-- centered search for DOWNLOAD -->
            <div v-if="navStore.currentNav === navStore.navOptions.DOWNLOAD" class="toolbar-center-search">
              <div class="btn-chip btn-sm search-chip" :style="{ width: (searchFocused ? 320 : 200) + 'px', transition: 'width 120ms ease' }">
                <Icon name="search" class="search-icon" />
                <input v-model="downloadNewSearch" type="text" class="search-input"
                       :placeholder="$t('sidebar.search_placeholder')"
                       @focus="onDownloadSearchFocus" @blur="searchFocused = false"
                       @input="emitDownloadNewSearch" />
              </div>
            </div>
            <!-- centered search for SUBTITLE: only on home (no project) -->
            <div v-else-if="navStore.currentNav === navStore.navOptions.SUBTITLE && !subtitleStore.currentProject" class="toolbar-center-search">
              <div class="btn-chip btn-sm search-chip" :style="{ width: (searchFocused ? 320 : 200) + 'px', transition: 'width 120ms ease' }">
                <Icon name="search" class="search-icon" />
                <input v-model="subtitleSearch" type="text" class="search-input"
                       :placeholder="$t('sidebar.search_placeholder')"
                       @focus="onSubtitleSearchFocus" @blur="searchFocused = false"
                       @input="emitSubtitleSearch" />
              </div>
            </div>

            <div class="flex items-center gap-2 ml-2">
              <!-- modal actions -->
              <template v-for="act in modalActions" :key="act.key">
                <button v-if="act.key === 'subtitle:metrics' && subtitleStore.currentProject"
                        class="chip-frosted chip-sm chip-translucent"
                        :data-tooltip="$t(act.titleKey)" data-tip-pos="bottom"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <span class="chip-label">{{ metricsStandardName || $t('subtitle.list.metrics_explanation') }}</span>
                </button>
                <button v-else-if="act.key === 'subtitle:open-file'"
                        class="chip-frosted chip-sm chip-translucent-primary"
                        :data-tooltip="$t(act.titleKey)" data-tip-pos="bottom"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <Icon :name="act.icon" class="chip-icon" />
                  <span class="chip-label">{{ $t(act.titleKey) }}</span>
                </button>
                <button v-else-if="act.key === 'subtitle:back-home'"
                        class="chip-frosted chip-sm chip-translucent-primary"
                        :data-tooltip="$t(act.titleKey)" data-tip-pos="bottom"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <Icon :name="act.icon" class="chip-icon" />
                  <span class="chip-label">{{ $t(act.titleKey) }}</span>
                </button>
                <button v-else-if="act.key === 'download:new-task'"
                        class="chip-frosted chip-sm chip-translucent-primary"
                        :data-tooltip="$t(act.titleKey)" data-tip-pos="bottom"
                        :aria-label="$t(act.titleKey)"
                        @click="onModalClick(act)">
                  <Icon :name="act.icon" class="chip-icon" />
                  <span class="chip-label">{{ $t(act.titleKey) }}</span>
                </button>
                <button v-else-if="primaryModalActions.has(act.key)"
                        class="chip-frosted chip-sm chip-primary-action"
                        :data-tooltip="$t(act.titleKey)" data-tip-pos="bottom"
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

              <!-- panel actions (inspector toggles) -->
              <div class="w-px h-4 bg-[var(--macos-divider-weak)] mx-1" v-if="panelActions.length && modalActions.length"></div>
              <button v-for="act in panelActions" :key="act.key" class="toolbar-chip"
                      :data-tooltip="$t(act.titleKey)" :aria-label="$t(act.titleKey)"
                      :class="{ active: isPanelActive(act.key) }"
                      @click="onActionClick(act)">
                <Icon :name="act.icon" class="w-4 h-4" />
              </button>
              <button v-if="inspector.visible" class="toolbar-chip" :data-tooltip="$t('common.close')" data-tip-pos="bottom" @click="inspector.close()">
                <Icon name="close" class="w-4 h-4" />
              </button>
            </div>

            <!-- Windows window controls live in content header -->
            <div v-if="isWindows()" class="ml-2 flex items-center">
              <ToolbarControlWidget :maximised="maximised" :size="34" />
            </div>
          </div>

          <!-- Content + Inspector -->
          <div id="app-content" class="flex flex-1 min-h-0 overflow-hidden">
	            <div
	              ref="pageScrollEl"
	              class="page-scroll flex-1 min-h-0 overflow-auto"
	              :class="[
	                { 'scrolled': hasTopScroll },
	                { 'pad-bottom-for-fab': needsBottomPad },
	              ]"
	            >
              <div v-show="navStore.currentNav === navStore.navOptions.DOWNLOAD" class="content-container min-w-0">
                <VideoDownloadPage />
              </div>

              <div v-if="navStore.currentNav === navStore.navOptions.SUBTITLE" class="content-container min-w-0">
                <Subtitle />
              </div>

	            </div>

            <div class="dc-inspector-shell" :style="{ width: (inspector.visible ? layout.inspectorWidth : 0) + 'px' }">
              <Inspector />
            </div>
          </div>
        </SidebarInset>
      </SidebarProvider>
    </div>
  </div>
</template>

<style scoped>
.loading {
  @apply animate-pulse;
}

.content-container {
  border-top-left-radius: 0;
  min-height: 100%;
}

/* Keep the inspector panel width transition smooth */
.dc-inspector-shell {
  flex: 0 0 auto;
  min-width: 0;
  overflow: hidden;
  transition: width 180ms ease;
}

/* add subtle divider between toolbar title and content */
#app-content .with-top-divider { border-top: 1px solid var(--macos-divider-weak); }
#app-content .page-scroll { padding: 0 !important; }
#app-content .content-container { padding: 0 !important; }
#app-content .settings-host { position: absolute; inset: 0; }
#app-content .with-split.settings-host::before { content: ''; position: absolute; inset: 0 auto 0 0; width: var(--settings-left, 160px); background: var(--sidebar-bg); }
#app-content .with-split.settings-host::after { content: ''; position: absolute; top: 0; bottom: 0; left: var(--settings-left, 160px); width: 1px; background: var(--macos-divider-weak); }
/* centered search overlay */
.toolbar-center-search { position: absolute; left: 0; right: 0; text-align: center; pointer-events: none; }
.toolbar-center-search input { pointer-events: auto; }

/* Search field styled to match btn-chip aesthetics */
.search-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 28px; /* match btn-chip geometry */
  /* make it colorless/quiet by default */
  background: transparent !important;
  border-color: var(--macos-separator) !important;
  box-shadow: none !important;
  text-shadow: none !important;
  color: var(--macos-text-secondary);
}
.search-chip:hover { background: var(--macos-hover-translucent) !important; border-color: var(--macos-divider-weak) !important; }
.search-chip .search-icon {
  width: 14px;
  height: 14px;
  color: var(--macos-text-tertiary);
  pointer-events: none; /* decorative */
}
.search-chip .search-input {
  flex: 1 1 auto;
  min-width: 0;
  height: 100%;
  border: none;
  outline: none;
  background: transparent;
  color: var(--macos-text-primary);
  font-size: var(--fs-sub);
}
.search-chip .search-input::placeholder { color: var(--macos-text-tertiary); }
/* Focus ring on the chip container when input focused */
.search-chip:focus-within {
  box-shadow: 0 0 0 2px color-mix(in oklab, var(--macos-blue) 28%, transparent);
  border-color: color-mix(in oklab, var(--macos-blue) 24%, white 12%);
}

/* Inline project name editor in toolbar */
.project-inline { display:flex; align-items:center; gap:6px; flex: 1 1 auto; min-width: 0; }
/* Project name as plain text with ellipsis and flex-resize */
.project-inline .project-name-text { flex: 1 1 auto; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; color: var(--macos-text-primary); font-size: var(--fs-sub); font-weight: 400; }
/* Inline editor adapts to available width; keep compact height for toolbar */
.project-inline .pill-input { flex: 1 1 auto; min-width: 120px; max-width: 100%; height: 22px; padding: 0 8px; border-radius: 999px; border: 1px solid var(--macos-separator); background: var(--macos-background); color: var(--macos-text-primary); font-size: var(--fs-sub); }
.project-inline .pill-input:focus { outline: none; border-color: var(--macos-blue); box-shadow: 0 0 0 2px color-mix(in oklab, var(--macos-blue) 30%, transparent); }

/* chip-primary-action 与图标尺寸样式已迁移到全局 styles/macos-components.scss */

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
</style>
