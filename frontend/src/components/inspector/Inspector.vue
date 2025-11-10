<template>
  <aside class="macos-inspector h-full flex flex-col">
    <div class="flex-1 overflow-auto p-0">
      <component v-if="ResolvedComponent" :is="ResolvedComponent" v-bind="inspector.props" @open-modal="openModal" />
      <div v-else class="text-xs p-3" :style="{ color: 'var(--macos-text-secondary)' }">Empty inspector</div>
    </div>
  </aside>
</template>

<script setup>
import { computed } from 'vue'
import useInspectorStore from '@/stores/inspector.js'
import SubtitleExportPanel from '@/components/panels/SubtitleExportPanel.vue'
import CookiesPanel from '@/components/panels/CookiesPanel.vue'
import InspectorHomePanel from '@/components/panels/InspectorHomePanel.vue'
import DownloadTaskPanel from '@/components/panels/DownloadTaskPanel.vue'

const inspector = useInspectorStore()

const map = {
  SubtitleExportPanel,
  CookiesPanel,
  InspectorHomePanel,
  DownloadTaskPanel,
}

const ResolvedComponent = computed(() => {
  const key = inspector.panel
  if (!key) return null
  return map[key] || null
})

function openModal() {
  // optional hook to open legacy modal if needed
}

</script>

<style scoped>
.macos-inspector {
  background: var(--macos-background-secondary);
  border-left: 1px solid var(--macos-divider-weak);
}
/* Avoid clipped pseudo-tooltips inside scrolling inspector; we will use explicit titles instead */
.macos-inspector :deep([data-tooltip])::after { display: none !important; content: none !important; }
</style>
