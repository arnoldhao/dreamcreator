<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { debounce } from 'lodash'
import { useThemeVars } from 'naive-ui'
import Ribbon from './components/sidebar/Ribbon.vue'
import SubtitleCommand from "@/components/sidebar/SubtitleCommand.vue";
import SubtitleContent from './components/content/SubtitleContent.vue';
import useSuperTabStore from './stores/supertab.js'
import usePreferencesStore from './stores/preferences.js'
import ToolbarControlWidget from '@/components/common/ToolbarControlWidget.vue'
import { EventsOn, WindowIsFullscreen, WindowIsMaximised, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'
import { isMacOS, isWindows } from '@/utils/platform.js'
import { extraTheme } from "@/utils/extra_theme.js";
import ResizeableWrapper from "@/components/common/ResizeableWrapper.vue";
import ContentValueTab from "@/components/content/ContentValueTab.vue";
import AiConfigurationSidebar from "@/components/sidebar/AIConfiguration.vue";
import LLMConfiguration from "@/components/content/LLMConfiguration.vue";
import HistoryPane from "@/components/content/HistoryPane.vue";
import DownloadVideoPage from "@/components/content/DownloadVideoPage.vue";
const themeVars = useThemeVars()

const props = defineProps({
  loading: Boolean,
})

const data = reactive({
  navMenuWidth: 50,
  toolbarHeight: 38,
})

const tabStore = useSuperTabStore()
const prefStore = usePreferencesStore()
const exThemeVars = computed(() => {
  return extraTheme(prefStore.isDark)
})

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
const wrapperStyle = computed(() => {
  if (isWindows()) {
    return {}
  }
  return hideRadius.value
    ? {}
    : {
      border: `0.1px solid ${themeVars.value.borderColor}`,
      borderRadius: '10px',
    }
})
const spinStyle = computed(() => {
  if (isWindows()) {
    return {
      backgroundColor: themeVars.value.bodyColor,
    }
  }
  return hideRadius.value
    ? {
      backgroundColor: themeVars.value.bodyColor,
    }
    : {
      backgroundColor: themeVars.value.bodyColor,
      borderRadius: '10px',
    }
})

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
  <n-spin :show="props.loading" :style="spinStyle" :theme-overrides="{ opacitySpinning: 0 }">
    <div id="app-content-wrapper" :style="wrapperStyle" class="flex-box-v">
      <!-- title bar -->
      <div id="app-toolbar" :style="{ height: data.toolbarHeight + 'px' }" class="flex-box-h"
        style="--wails-draggable: drag" @dblclick="WindowToggleMaximise">
        <!-- title -->
        <div id="app-toolbar-title" :style="{
          width: logoWrapperWidth,
          minWidth: logoWrapperWidth,
          paddingLeft: `${logoPaddingLeft}px`,
        }">
          <n-space :size="3" :wrap="false" :wrap-item="false" align="center">
            <div style="min-width: 68px; white-space: nowrap; font-weight: 800; margin-left: 8px; text-transform: capitalize">{{ tabStore.nav }}</div>
          </n-space>
        </div>
        <!-- browser tabs -->
        <div v-show="tabStore.nav === 'subtitle'" class="app-toolbar-tab flex-item-expand">
          <content-value-tab />
        </div>
        <div class="flex-item-expand" style="min-width: 15px"></div>
        <!-- simulate window control buttons -->
        <toolbar-control-widget v-if="!isMacOS()" :maximised="maximised" :size="data.toolbarHeight"
          style="align-self: flex-start" />
      </div>

      <!-- content -->
      <div id="app-content" :style="prefStore.generalFont" class="flex-box-h flex-item-expand"
        style="--wails-draggable: none">
        <ribbon v-model:value="tabStore.nav" :width="data.navMenuWidth" />
        <!-- subtitle page -->
        <div v-show="tabStore.nav === 'subtitle'" class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper v-model:size="prefStore.behavior.asideWidth" :min-size="300" :offset="data.navMenuWidth"
            class="flex-item" @update:size="handleResize">
            <subtitle-command class="app-side flex-item-expand" />
          </resizeable-wrapper>
          <subtitle-content v-for="t in tabStore.tabs" v-show="tabStore.currentTabId === t.id" :key="t.id"
            :name="t.id" :title="t.title" class="flex-item-expand" />
        </div>

        <!-- ai page -->
        <div v-show="tabStore.nav === 'ai'" class="content-area flex-box-h flex-item-expand">
          <resizeable-wrapper v-model:size="prefStore.behavior.asideWidth" :min-size="300" :offset="data.navMenuWidth"
            class="flex-item" @update:size="handleResize">
            <ai-configuration-sidebar class="app-side flex-item-expand" />
          </resizeable-wrapper>
          <LLMConfiguration class="flex-item-expand" />
        </div>

        <!-- history page -->
        <div v-show="tabStore.nav === 'history'" class="content-area flex-box-h flex-item-expand">
          <history-pane class="flex-item-expand" />
        </div>

        <!-- download video page -->
        <div v-show="tabStore.nav === 'download'" class="content-area flex-box-h flex-item-expand">
          <download-video-page class="flex-item-expand" />
        </div>
      </div>
    </div>
  </n-spin>
</template>

<style lang="scss" scoped>
#app-content-wrapper {
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  box-sizing: border-box;
  background-color: v-bind('themeVars.bodyColor');
  color: v-bind('themeVars.textColorBase');

  #app-toolbar {
    background-color: v-bind('exThemeVars.uniFrameColor');

    &-title {
      padding-left: 10px;
      padding-right: 10px;
      box-sizing: border-box;
      align-self: center;
      align-items: baseline;
    }
  }

  .app-toolbar-tab {
    align-self: flex-end;
    margin-bottom: -1px;
    margin-left: 3px;
    overflow: auto;
  }

  #app-content {
    height: calc(100% - 60px);

    .content-area {
      border-top: v-bind('exThemeVars.splitColor') solid 0.1px;
      border-left: v-bind('exThemeVars.splitColor') solid 0.1px;
      border-top-left-radius: 8px; // 左上角圆角
      display: flex;
      overflow: hidden; // 确保子元素圆角裁剪 
    }
  }

  .app-side {
    //overflow: hidden;
    height: 100%;
    background-color: v-bind('exThemeVars.sidebarColor');
    border-right: 1px solid v-bind('exThemeVars.splitColor');
  }
}
</style>
