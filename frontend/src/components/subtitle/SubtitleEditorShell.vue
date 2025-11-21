<template>
  <div class="editor-shell">
    <template v-if="currentProject">
      <!-- Detail view: in-page task detail + project meta -->
      <div v-if="isDetailView" class="macos-card card-frosted card-translucent w-full">
        <div class="trans-panel p-3">
          <SubtitleDetailPanel
            :language="currentLanguageLabel"
            :status="currentLangTask?.status || langMeta?.sync_status"
            :sync-status="langMeta?.sync_status || ''"
            :progress-pct="progressPct"
            :processed="processed"
            :total="total"
            :failed-count="failedCount"
            :start-time-text="startTimeDisplay"
            :elapsed-text="elapsedText"
            :provider="providerName"
            :model="llmModelName"
            :message="progressMessage"
            :file-path="currentProject?.metadata?.source_info?.file_name || ''"
            :format="(currentProject?.metadata?.source_info?.file_ext || '')?.toUpperCase?.() || ''"
            :error-text="errorText"
            :providers="[]"
            :models="[]"
            :provider-selected="''"
            :model-selected="''"
            :retry-failed-only="false"
            :is-narrow="isNarrow"
            :show-retry-controls="false"
            :prompt-tokens="displayPromptTokens"
            :completion-tokens="displayCompletionTokens"
            :total-tokens="displayTotalTokens"
            :request-count="displayRequestCount"
            :token-speed="tokenSpeedText"
            :project-total="Number(currentLangTask?.project_total_segments || currentProject?.segments?.length || 0)"
            @open-llm-talk="$emit('open-llm-chat')"
          />

          <!-- Project meta -->
          <div class="macos-group">
            <div class="macos-group-title">{{ $t('subtitle.export.summary') }}</div>
            <div class="macos-box card-frosted card-translucent">
              <div class="macos-row">
                <span class="k">{{ $t('common.name') }}</span>
                <span class="v">
                  <span class="file-line">
                    <span class="name mono one-line" :title="currentProject?.project_name || ''">{{ currentProject?.project_name || '-' }}</span>
                    <button
                      v-if="currentProject?.project_name"
                      class="btn-chip-icon btn-xxs"
                      :data-tooltip="$t('common.copy')"
                      data-tip-pos="top"
                      @click="$emit('copy-project-name')"
                    >
                      <Icon name="file-copy" class="w-3.5 h-3.5" />
                    </button>
                  </span>
                </span>
              </div>
              <div class="macos-row">
                <span class="k">ID</span>
                <span class="v one-line mono">{{ currentLangTask?.id || langMeta?.active_task_id || '-' }}</span>
              </div>
              <div class="macos-row">
                <span class="k">{{ $t('subtitle.add_language.source_language') }}</span>
                <span class="v one-line">{{ currentLangTask?.source_lang || originalLanguageCode }}</span>
              </div>
              <div class="macos-row">
                <span class="k">{{ $t('subtitle.detail.file') }}</span>
                <span class="v">
                  <span class="file-line">
                    <span class="name mono one-line" :title="currentProject?.metadata?.source_info?.file_name || ''">
                      {{ currentProject?.metadata?.source_info?.file_name || '-' }}
                    </span>
                    <button
                      v-if="currentProject?.metadata?.source_info?.file_name"
                      class="btn-chip-icon btn-xxs"
                      :data-tooltip="$t('common.copy')"
                      data-tip-pos="top"
                      @click="$emit('copy-project-file-name')"
                    >
                      <Icon name="file-copy" class="w-3.5 h-3.5" />
                    </button>
                  </span>
                </span>
              </div>
              <div class="macos-row">
                <span class="k">{{ $t('subtitle.detail.format') }}</span>
                <span class="v one-line mono">
                  {{ (currentProject?.metadata?.source_info?.file_ext || '-')?.toUpperCase?.() || '-' }}
                </span>
              </div>
              <!-- Global style summary derived from project analysis -->
              <div class="macos-row" v-if="projectStyle && projectStyle.genre">
                <span class="k">{{ $t('subtitle.export.genre') }}</span>
                <span class="v mono one-line">{{ projectStyle.genre }}</span>
              </div>
              <div class="macos-row" v-if="projectStyle && projectStyle.tone">
                <span class="k">{{ $t('subtitle.export.tone') }}</span>
                <span class="v one-line">{{ projectStyle.tone }}</span>
              </div>
              <div class="macos-row" v-if="styleGuidePills.length">
                <span class="k">{{ $t('subtitle.export.style_guide') }}</span>
                <span class="v">
                  <div class="style-pills">
                    <span
                      v-for="rule in styleGuidePills"
                      :key="rule"
                      class="chip-frosted chip-sm chip-translucent"
                    >
                      <span class="chip-label">{{ rule }}</span>
                    </span>
                  </div>
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Editor view -->
      <div v-else class="macos-card card-frosted card-translucent sr-card w-full">
        <div class="sr-card-body p-3">
          <SubtitleList
            :subtitles="currentLanguageSegments"
            :current-language="currentLanguage"
            :available-languages="availableLanguages"
            :subtitle-counts="subtitleCounts"
            @add-language="$emit('add-language')"
            @update:currentLanguage="$emit('update:currentLanguage', $event)"
            @update:projectData="$emit('update:projectData', $event)"
          />
        </div>
      </div>

      <!-- End sentinel for bottom detection (editor only) -->
      <div ref="endSentinel" style="height: 1px; width: 100%;"></div>
    </template>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import SubtitleList from '@/components/subtitle/SubtitleList.vue'
import SubtitleDetailPanel from '@/components/subtitle/SubtitleDetailPanel.vue'

const {
  currentProject,
  isDetailView,
  isNarrow,
  projectStyle,
  styleGuidePills,
  currentLanguageLabel,
  currentLangTask,
  langMeta,
  originalLanguageCode,
  progressPct,
  processed,
  total,
  failedCount,
  startTimeDisplay,
  elapsedText,
  providerName,
  llmModelName,
  progressMessage,
  errorText,
  displayPromptTokens,
  displayCompletionTokens,
  displayTotalTokens,
  displayRequestCount,
  tokenSpeedText,
  currentLanguageSegments,
  currentLanguage,
  availableLanguages,
  subtitleCounts,
} = defineProps({
  currentProject: { type: Object, default: null },
  isDetailView: { type: Boolean, default: false },
  isNarrow: { type: Boolean, default: false },
  projectStyle: { type: Object, default: null },
  styleGuidePills: { type: Array, default: () => [] },
  currentLanguageLabel: { type: String, default: '' },
  currentLangTask: { type: Object, default: null },
  langMeta: { type: Object, default: null },
  originalLanguageCode: { type: String, default: '' },
  progressPct: { type: Number, default: 0 },
  processed: { type: Number, default: 0 },
  total: { type: Number, default: 0 },
  failedCount: { type: Number, default: 0 },
  startTimeDisplay: { type: String, default: '' },
  elapsedText: { type: String, default: '' },
  providerName: { type: String, default: '' },
  llmModelName: { type: String, default: '' },
  progressMessage: { type: String, default: '' },
  errorText: { type: String, default: '' },
  displayPromptTokens: { type: Number, default: 0 },
  displayCompletionTokens: { type: Number, default: 0 },
  displayTotalTokens: { type: Number, default: 0 },
  displayRequestCount: { type: Number, default: 0 },
  tokenSpeedText: { type: String, default: '' },
  currentLanguageSegments: { type: Array, default: () => [] },
  currentLanguage: { type: String, default: '' },
  availableLanguages: { type: Object, default: () => ({}) },
  subtitleCounts: { type: Object, default: () => ({}) },
})

defineEmits([
  'open-llm-chat',
  'copy-project-name',
  'copy-project-file-name',
  'add-language',
  'update:currentLanguage',
  'update:projectData',
])

const endSentinel = ref(null)

defineExpose({ endSentinel })
</script>

