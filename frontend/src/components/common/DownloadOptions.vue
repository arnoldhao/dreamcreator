<template>
  <div class="download-options">
    <div class="options-grid">
      <!-- Strategy: custom -->
      <template v-if="strategy === 'custom'">
        <!-- Video Quality -->
        <div class="option-group">
          <label for="video-quality-select" class="option-label">{{ $t('download.video_quality') }}</label>
          <select id="video-quality-select" class="select-macos" :value="modelSelectedQuality ? stringify(modelSelectedQuality) : ''"
                  @change="onQualityChange($event)">
            <option disabled value="">{{ $t('download.select_video_quality') }}</option>
            <template v-for="quality in qualities" :key="quality.id || quality.format_id || quality.label">
              <option v-if="quality.isHeader" disabled class="option-header">▾ {{ quality.label }}</option>
              <option v-else :value="stringify(quality)">{{ quality.label }}</option>
            </template>
          </select>
        </div>

        <!-- Subtitle Selection (single-select, consistent appearance) -->
        <div class="option-group">
          <label for="subtitle-lang-select" class="option-label">{{ $t('download.subtitle_language') }}</label>
          <select id="subtitle-lang-select" class="select-macos" :value="modelSelectedSubtitleLang"
                  @change="onSubtitleLangChange($event)">
            <option value="">{{ $t('download.no_subtitles') }}</option>
            <option v-if="(subtitles?.length||0) > 1" value="ALL">{{ $t('download.all_subtitles') }}</option>
            <template v-if="subtitles?.length">
              <option v-for="subtitle in subtitles" :key="subtitle.value.lang" :value="subtitle.value.lang">
                {{ subtitle.label }}
              </option>
            </template>
          </select>
        </div>

        <!-- Transcoding -->
        <div class="option-group">
          <label for="transcode-format-select" class="option-label">{{ $t('download.transcoding') }}</label>
          <select id="transcode-format-select" class="select-macos" :value="modelSelectedTranscode"
                  @change="$emit('update:selectedTranscodeFormat', Number($event.target.value))">
            <option :value="0">{{ $t('download.no_transcoding') }}</option>
            <template v-if="customTranscodeOptions.video.length">
              <optgroup :label="$t('download.video_formats')">
                <option v-for="format in customTranscodeOptions.video" :key="format.id" :value="format.id">
                  {{ format.name }}
                </option>
              </optgroup>
            </template>
            <template v-if="customTranscodeOptions.audio.length">
              <optgroup :label="$t('download.audio_formats')">
                <option v-for="format in customTranscodeOptions.audio" :key="format.id" :value="format.id">
                  {{ format.name }}
                </option>
              </optgroup>
            </template>
          </select>
        </div>
      </template>

      <!-- Strategy: quick -->
      <template v-else>
        <div class="option-group">
          <label for="quick-video-quality-select" class="option-label">{{ $t('download.video_quality') }}</label>
          <select id="quick-video-quality-select" class="select-macos" :value="modelVideoPreset"
                  @change="$emit('update:videoPreset', $event.target.value)">
            <option v-for="item in videoItems" :key="item.key" :value="item.key">{{ item.label }}</option>
          </select>
        </div>

        <div class="option-group">
          <label for="quick-subtitle-lang-select" class="option-label">{{ $t('download.subtitle_language') }}</label>
          <select id="quick-subtitle-lang-select" class="select-macos" :value="modelBestCaption"
                  @change="$emit('update:bestCaption', ($event.target.value === 'true'))">
            <option :value="false">{{ $t('download.no_caption') }}</option>
            <option :value="true">{{ $t('download.best_caption') }}</option>
          </select>
        </div>

        <div class="option-group">
          <label for="quick-transcode-format-select" class="option-label">{{ $t('download.transcoding') }}</label>
          <select id="quick-transcode-format-select" class="select-macos" :value="modelQuickTranscode"
                  @change="$emit('update:quickSelectedTranscodeFormat', Number($event.target.value))">
            <option :value="0">{{ $t('download.no_transcoding') }}</option>
            <template v-if="quickTranscodeOptions.video.length > 0">
              <optgroup :label="$t('download.video_formats')">
                <option v-for="format in quickTranscodeOptions.video" :key="format.id" :value="format.id">{{ format.name }}</option>
              </optgroup>
            </template>
            <template v-if="quickTranscodeOptions.audio.length > 0">
              <optgroup :label="$t('download.audio_formats')">
                <option v-for="format in quickTranscodeOptions.audio" :key="format.id" :value="format.id">{{ format.name }}</option>
              </optgroup>
            </template>
          </select>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { computed, defineProps, watch } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

const props = defineProps({
  strategy: { type: String, default: 'custom' },
  // custom
  qualities: { type: Array, default: () => [] },
  subtitles: { type: Array, default: () => [] },
  availableTranscodeFormats: { type: Object, default: () => ({ video: [], audio: [] }) },
  modelSelectedQuality: null,
  modelSelectedSubtitleLang: { type: String, default: '' }, // kept for backward-compat, unused when multi-select is enabled
  modelSelectedSubtitles: { type: Array, default: () => [] },
  modelSelectedTranscode: { type: Number, default: 0 },
  // quick
  modelVideoPreset: { type: String, default: 'best' },
  modelBestCaption: { type: [Boolean, String], default: false },
  modelQuickTranscode: { type: Number, default: 0 }
})

// Setup emits (declare early for watchers)
const emit = defineEmits([
  'update:selectedQuality',
  'update:selectedSubtitleLang',
  'update:selectedSubtitles',
  'update:selectedTranscodeFormat',
  'update:videoPreset',
  'update:bestCaption',
  'update:quickSelectedTranscodeFormat'
])

const videoItems = computed(() => [
  { key: 'best', label: t('download.best_quality') },
  { key: 'bestaudio', label: t('download.best_audio') },
])

// Custom strategy: filter transcode options by selected quality
const customTranscodeOptions = computed(() => {
  const opts = { video: [], audio: [] }
  const src = props.availableTranscodeFormats || { video: [], audio: [] }
  const qType = props.modelSelectedQuality?.type
  if (qType === 'audio_only') {
    opts.audio = [...(src.audio || [])]
  } else {
    opts.video = [...(src.video || [])]
    opts.audio = [...(src.audio || [])]
  }
  return opts
})

const quickTranscodeOptions = computed(() => {
  const options = { video: [], audio: [] }
  if (!props.availableTranscodeFormats) return options
  if (props.modelVideoPreset === 'best') {
    options.video = [...(props.availableTranscodeFormats.video || [])]
    options.audio = [...(props.availableTranscodeFormats.audio || [])]
  } else if (props.modelVideoPreset === 'bestaudio') {
    options.audio = [...(props.availableTranscodeFormats.audio || [])]
  }

  return options
})

const stringify = (obj) => JSON.stringify(obj)
const onQualityChange = (e) => {
  try {
    const v = JSON.parse(e.target.value)
    // Emit using Vue model modifier (kebab) keys
    // Consumers should bind v-model:selectedQuality
    // We follow emit-by-prop-name pattern here via defineEmits implicit by v-model usage
    // But since we used prop alias modelSelectedQuality, parent binds manually via :modelSelectedQuality and @update:selectedQuality
    // To keep simple, emit the canonical name
    // eslint-disable-next-line no-undef
    emit('update:selectedQuality', v)
  } catch {}
}
const onSubtitleLangChange = (e) => {
  const value = e.target.value
  emit('update:selectedSubtitleLang', value)
  if (!value) {
    emit('update:selectedSubtitles', [])
  } else if (value === 'ALL') {
    // Sentinel: request yt-dlp to download all subtitles
    emit('update:selectedSubtitles', [{ lang: 'all' }])
  } else {
    const sub = props.subtitles.find(s => s?.value?.lang === value)
    emit('update:selectedSubtitles', sub ? [sub.value] : [])
  }
}

// Ensure custom transcode selection remains valid when quality/preset changes
watch(
  () => [props.strategy, props.modelSelectedQuality, props.modelSelectedTranscode, props.availableTranscodeFormats],
  () => {
    if (props.strategy !== 'custom') return
    const selected = props.modelSelectedTranscode || 0
    if (selected === 0) return
    const allowed = new Set([
      ...customTranscodeOptions.value.video.map(f => f.id),
      ...customTranscodeOptions.value.audio.map(f => f.id),
    ])
    if (!allowed.has(selected)) {
      emit('update:selectedTranscodeFormat', 0)
    }
  },
  { immediate: true }
)
</script>

<style scoped>
.download-options { padding: 16px 0; display: flex; flex-direction: column; gap: 20px; }
.options-grid { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; }
.option-group { display: flex; flex-direction: column; gap: 6px; }
.option-label { font-weight: 500; text-transform: uppercase; letter-spacing: 0.5px; }
.select-macos option.option-header { background: var(--macos-background-secondary); color: var(--macos-text-secondary); font-weight: 600; }
@media (max-width: 768px) { .options-grid { grid-template-columns: 1fr; gap: 12px; } }


</style>
