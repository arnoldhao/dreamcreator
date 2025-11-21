<template>
  <div
    class="bottom-toolbar chip-frosted chip-translucent"
    :style="toolbarStyle"
    @click.stop
  >
    <!-- Left group: language select (+ add language on normal mode) -->
    <div class="tb-left">
      <!-- Metrics standard chip -->
      <button
        class="chip-frosted chip-sm chip-translucent"
        :data-tooltip="$t('subtitle.list.metrics_explanation')"
        data-tip-pos="top"
        :aria-label="$t('subtitle.list.metrics_explanation')"
        @click="$emit('show-metrics')"
      >
        <span class="chip-label">{{ metricsStandardName || $t('subtitle.list.metrics_explanation') }}</span>
      </button>
      <div class="lang-select">
        <Icon name="languages" class="w-4 h-4" />
        <select
          class="input-macos select-macos select-macos-xs"
          :value="currentLanguage"
          @change="$emit('update:currentLanguage', $event.target.value)"
        >
          <option
            v-for="opt in languageOptions"
            :key="opt.code"
            :value="opt.code"
          >
            {{ opt.name }}
          </option>
        </select>
      </div>
      <button
        v-if="!isNarrow"
        class="btn-chip-icon"
        :data-tooltip="$t('subtitle.add_language.title')"
        data-tip-pos="top"
        @click="$emit('add-language')"
        aria-label="add language"
      >
        <Icon name="plus" class="w-4 h-4" />
      </button>
    </div>

    <!-- Middle group: view switch (normal mode only) -->
    <div class="tb-middle" v-if="!isNarrow">
      <div class="seg-chip chip-sm chip-frosted chip-translucent">
        <button
          type="button"
          class="seg-item"
          :class="{ active: !isDetailView, disabled: editorDisabled }"
          @click="!editorDisabled && $emit('set-view', 'editor')"
          :aria-disabled="editorDisabled ? 'true' : 'false'"
          :data-tooltip="$t('settings.editor.name') || 'Editor'"
          data-tip-pos="top"
        >
          {{ $t('settings.editor.name') || 'Editor' }}
        </button>
        <button
          type="button"
          class="seg-item"
          :class="{ active: isDetailView }"
          @click="$emit('set-view', 'detail')"
          :data-tooltip="$t('download.detail') || 'Detail'"
          data-tip-pos="top"
        >
          {{ $t('download.detail') || 'Detail' }}
        </button>
      </div>
    </div>

    <!-- Right group: refresh + retry + delete -->
    <div class="tb-right">
      <button
        class="btn-chip-icon"
        :data-tooltip="$t('download.refresh')"
        data-tip-pos="top"
        @click="$emit('refresh')"
        aria-label="refresh"
      >
        <Icon name="refresh" class="w-4 h-4" :class="{ spinning: refreshing }" />
      </button>
      <template v-if="canDeleteCurrentLanguage">
        <button
          class="btn-chip-icon"
          :data-tooltip="$t('download.retry')"
          data-tip-pos="top"
          @click="$emit('open-retry')"
          aria-label="retry translate"
          v-if="!isTranslating"
        >
          <Icon name="retry" class="w-4 h-4" />
        </button>
        <button
          class="btn-chip-icon btn-danger"
          :data-tooltip="$t('common.delete')"
          data-tip-pos="top"
          @click="$emit('delete-language')"
          aria-label="delete current translation"
        >
          <Icon name="trash" class="w-4 h-4" />
        </button>
      </template>
    </div>
  </div>
</template>

<script setup>
const props = defineProps({
  isNarrow: { type: Boolean, default: false },
  metricsStandardName: { type: String, default: '' },
  languageOptions: { type: Array, default: () => [] },
  currentLanguage: { type: String, default: '' },
  canDeleteCurrentLanguage: { type: Boolean, default: false },
  isTranslating: { type: Boolean, default: false },
  refreshing: { type: Boolean, default: false },
  isDetailView: { type: Boolean, default: false },
  editorDisabled: { type: Boolean, default: false },
  toolbarStyle: { type: Object, default: () => ({}) },
})

defineEmits([
  'update:currentLanguage',
  'show-metrics',
  'add-language',
  'refresh',
  'open-retry',
  'delete-language',
  'set-view',
])
</script>
