<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import Ribbon from '@/components/sidebar/Ribbon.vue'
import useNavStore from '@/stores/nav.js'
import usePreferencesStore from '@/stores/preferences.js'
import ToolbarControlWidget from '@/components/common/ToolbarControlWidget.vue'
import { EventsOn, WindowIsFullscreen, WindowIsMaximised, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'
import { isMacOS, isWindows } from '@/utils/platform.js'
import Settings from "@/components/content/Settings.vue";
import VideoDownloadPage from "@/components/content/VideoDownloadPage.vue";

const props = defineProps({
  loading: Boolean,
})

const data = reactive({
  navMenuWidth: 50,
})

const navStore = useNavStore()
const prefStore = usePreferencesStore()

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
})

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
      <div id="app-toolbar" class="flex items-center flex-shrink-0 h-[38px]"
        :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'" style="--wails-draggable: drag"
        @dblclick="WindowToggleMaximise">
        <!-- title -->
        <div id="app-toolbar-title" class="flex items-center" :style="{
          width: logoWrapperWidth,
          minWidth: logoWrapperWidth,
          paddingLeft: `${logoPaddingLeft}px`,
        }">
          <div class="ml-2 font-extrabold capitalize">{{ $t('ribbon.' + (navStore.currentNav)) }}</div>
        </div>

        <div class="flex-1 min-w-[15px]"></div>
        <!-- window controls -->
        <toolbar-control-widget v-if="!isMacOS()" :maximised="maximised" :size="38" class="self-start" />
      </div>

      <!-- content -->
      <div id="app-content" class="flex flex-1 min-h-0 overflow-hidden" :style="prefStore.generalFont"
        style="--wails-draggable: none">
        <ribbon v-model:value="navStore.currentNav" :width="data.navMenuWidth" />

        <!-- download video page -->
        <div v-show="navStore.currentNav === navStore.navOptions.DOWNLOAD" class="flex-1 content-container">
          <video-download-page class="flex-1" />
        </div>

        <!-- settings page -->
        <div v-show="navStore.currentNav === navStore.navOptions.SETTINGS" class="flex-1 content-container">
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