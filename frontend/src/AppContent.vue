<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { debounce } from 'lodash'
import Ribbon from './components/sidebar/Ribbon.vue'
import SubtitleCommand from "@/components/sidebar/SubtitleCommand.vue";
import SubtitleContent from './components/content/SubtitleContent.vue';
import useSuperTabStore from './stores/supertab.js'
import usePreferencesStore from './stores/preferences.js'
import ToolbarControlWidget from '@/components/common/ToolbarControlWidget.vue'
import { EventsOn, WindowIsFullscreen, WindowIsMaximised, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'
import { isMacOS, isWindows } from '@/utils/platform.js'
import ResizeableWrapper from "@/components/common/ResizeableWrapper.vue";
import ContentValueTab from "@/components/content/ContentValueTab.vue";
import AiConfigurationSidebar from "@/components/sidebar/AIConfiguration.vue";
import LLMConfiguration from "@/components/content/LLMConfiguration.vue";
import HistoryPane from "@/components/content/HistoryPane.vue";
import DownloadVideoPage from "@/components/content/DownloadVideoPage.vue";
import Settings from "@/components/content/Settings.vue";
import Optimize from "@/components/content/Optimize.vue";

const props = defineProps({
  loading: Boolean,
})

const data = reactive({
  navMenuWidth: 50,
})

const tabStore = useSuperTabStore()
const prefStore = usePreferencesStore()

const saveSidebarWidth = debounce(prefStore.savePreferences, 1000, { trailing: true })
const handleResize = () => {
  saveSidebarWidth()
}

const logoWrapperWidth = computed(() => {
  return `${data.navMenuWidth + prefStore.behavior.asideWidth - 4}px`
})

const logoPaddingLeft = ref(10)
const maximised = ref(false)
const hideRadius = ref(false)

const onToggleFullscreen = (fullscreen) => {
  hideRadius.value = fullscreen
  if (fullscreen) {
    logoPaddingLeft.value = 10
  } else {
    logoPaddingLeft.value = isMacOS() ? 70 : 10
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
  window.addEventListener('keydown', onKeyShortcut)
  initializeTabs()
})

onUnmounted(() => {
  window.removeEventListener('keydown', onKeyShortcut)
})

const onKeyShortcut = (e) => {
  const isCtrlOn = isMacOS() ? e.metaKey : e.ctrlKey
  switch (e.key) {
    case 'w':
      if (isCtrlOn) {
        // close current tab
        const tabStore = useSuperTabStore()
        const currentTab = tabStore.currentTab
        if (currentTab != null) {
          tabStore.closeTab(currentTab.id)
        }
      }
      break
  }
}

function initializeTabs(tabName) {
  if (tabStore.tabs.length === 0) {
    tabStore.openBlankTab()
  }
}
</script>

<template>
  <!-- app content-->
  <div class="relative min-h-screen" :class="[
    prefStore.isDark ? 'bg-neutral' : 'bg-white',
    { 'loading': props.loading }
  ]">
    <div id="app-content-wrapper" class="flex flex-col h-screen" :class="[
      hideRadius ? '' : 'rounded-xl border border-base-300',
      isWindows() ? '' : 'overflow-hidden'
    ]">
      <!-- title bar -->
      <div id="app-toolbar" 
        class="flex items-center flex-shrink-0 h-[38px]" 
        :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'"
        style="--wails-draggable: drag" 
        @dblclick="WindowToggleMaximise">
        <!-- title -->
        <div id="app-toolbar-title" 
          class="flex items-center"
          :style="{
            width: logoWrapperWidth,
            minWidth: logoWrapperWidth,
            paddingLeft: `${logoPaddingLeft}px`,
          }">
          <div class="ml-2 font-extrabold capitalize">{{ $t('ribbon.'+(tabStore.nav)) }}</div>
        </div>
        
        <!-- browser tabs -->
        <div v-show="tabStore.nav === 'subtitle'" class="flex-1"> 
          <content-value-tab />
        </div>
        <div class="flex-1 min-w-[15px]"></div>
        
        <!-- window controls -->
        <toolbar-control-widget 
          v-if="!isMacOS()" 
          :maximised="maximised" 
          :size="38" 
          class="self-start" />
      </div>

      <!-- content -->
      <div id="app-content" 
        class="flex flex-1 min-h-0 overflow-hidden" 
        :style="prefStore.generalFont" 
        style="--wails-draggable: none">
        <ribbon v-model:value="tabStore.nav" :width="data.navMenuWidth" />
        
        <!-- subtitle page -->
        <!-- <div v-show="tabStore.nav === 'subtitle'" class="flex-1 content-container">
          <resizeable-wrapper 
            v-model:size="prefStore.behavior.asideWidth" 
            :min-size="300" 
            :offset="data.navMenuWidth"
            :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'"
            @update:size="handleResize">
            <subtitle-command class="h-full" />
          </resizeable-wrapper>
          <subtitle-content 
            v-for="t in tabStore.tabs" 
            v-show="tabStore.currentTabId === t.id" 
            :key="t.id"
            :name="t.id" 
            :title="t.title" 
            class="flex-1" />
        </div> -->

        <!-- ai page -->
        <!-- <div v-show="tabStore.nav === 'ai'" class="flex-1 content-container">
          <resizeable-wrapper 
            v-model:size="prefStore.behavior.asideWidth" 
            :min-size="300" 
            :offset="data.navMenuWidth"
            class="bg-base-200"
            @update:size="handleResize">
            <ai-configuration-sidebar class="h-full" />
          </resizeable-wrapper>
          <LLMConfiguration class="flex-1" />
        </div> -->

        <!-- download video page -->
        <div v-show="tabStore.nav === 'download'" class="flex-1 content-container">
          <download-video-page class="flex-1" />
        </div>

        <!-- settings page -->
        <div v-show="tabStore.nav === 'optimize'" class="flex-1 content-container">
          <optimize class="flex-1" />
        </div>

        <!-- history page -->
        <div v-show="tabStore.nav === 'history'" class="flex-1 content-container">
          <history-pane class="flex-1" />
        </div>

        <!-- settings page -->
        <div v-show="tabStore.nav === 'settings'" class="flex-1 content-container">
          <settings class="flex-1" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.loading {
  @apply animate-pulse;
}

.content-container {
  @apply rounded-tl-lg border-t border-l border-base-300;
}
</style>