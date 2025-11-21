<template>
  <div class="sep-root">
    <!-- Top spacing header (remove visible label '字幕摘要' for consistency) -->
    <div class="macos-group sep-header"><div class="grow"></div></div>
    <div class="macos-box card-frosted card-translucent" v-if="project">
      <div class="macos-row"><span class="k">{{ $t('subtitle.common.project_name') }}</span><span class="v one-line" :title="project?.project_name">{{ project?.project_name || '-' }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('subtitle.common.lang') }}</span><span class="v">{{ lang }}</span></div>
      <div class="macos-row"><span class="k">{{ $t('subtitle.common.cues') }}</span><span class="v">{{ cueCount }}</span></div>
    </div>

    <!-- Group: Export Configuration -->
    <!-- Panel delegates group+box UI to SubtitleExportConfig to avoid duplicate headings -->
    <SubtitleExportConfig
      v-if="project"
      :project-data="project"
      :current-language="lang"
      @save-config="onSaveConfig"
      @export-subtitles="onExport"/>
  </div>
  
</template>

<script setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import SubtitleExportConfig from '@/components/subtitle/SubtitleExportConfig.vue'
import { useSubtitleStore } from '@/stores/subtitle'

const { t } = useI18n()
const store = useSubtitleStore()
const project = computed(() => store.currentProject)
const lang = computed(() => {
  const meta = store.currentProject?.language_metadata
  if (!meta) return 'English'
  const keys = Object.keys(meta)
  return keys.length ? (store.currentLanguage || keys[0]) : 'English'
})

const cueCount = computed(() => {
  const segs = store.currentProject?.segments || []
  const l = lang.value
  return segs.filter(s => s.languages && s.languages[l]).length
})

const emit = defineEmits(['open-modal'])
const onSaveConfig = () => {}
const onExport = () => {}
</script>

<style scoped>
.sep-root { padding: 10px; font-size: var(--fs-base); color: var(--macos-text-primary); }
.sep-header { display:flex; align-items:center; }
.sep-header + .macos-box { margin-top: 8px; }
.title-bar { display:flex; align-items:center; justify-content: space-between; gap: 8px; margin-top: 2px; margin-bottom: 8px; }
.title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); min-width:0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pill { font-size: var(--fs-caption); padding: 2px 8px; border-radius: 999px; border: 1px solid var(--macos-separator); color: var(--macos-text-secondary); background: var(--macos-background); }
/* use global macos-group/macos-box/macos-row */
.one-line { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
/* Remove inner card shadows from config */
:deep(.card-macos), :deep(.macos-card), :deep(.card) { box-shadow: none; }
</style>
