<template>
  <dialog :open="showModal" :class="['modal', { 'modal-open': showModal }]">
    <div class="modal-box w-11/12 max-w-3xl bg-base-100 overflow-visible">
      <h3 class="font-bold text-lg flex items-center">
        <v-icon name="ri-download-cloud-line" class="w-5 h-5 mr-2 text-primary"></v-icon>
        {{ mode === 'quick' ? $t('download.quick_task') : $t('download.new_task') }}
      </h3>

      <!-- 模式切换按钮 -->
      <div class="tabs tabs-boxed bg-base-200 p-1 mt-2">
        <button class="tab flex-1" :class="{ 'tab-active': mode === 'custom' }" @click="mode = 'custom'">
          {{ $t('download.custom_mode') }}
        </button>
        <button class="tab flex-1" :class="{ 'tab-active': mode === 'quick' }" @click="mode = 'quick'">
          {{ $t('download.quick_mode') }}
        </button>
      </div>

      <!-- 依赖管理 -->
      <div v-if="!dependenciesReady" class="py-8">
        <div class="flex flex-col items-center gap-6">
          <!-- 警告图标和文字 -->
          <div class="flex flex-col items-center gap-2">
            <v-icon name="ri-error-warning-line" class="w-12 h-12 text-warning"></v-icon>
            <p class="text-lg font-medium text-base-content">{{ $t('download.dependencies_not_ready') }}</p>
            <p class="text-sm text-base-content/60">{{ $t('download.dependencies_not_ready_desc') }}</p>
          </div>
          <!-- 操作按钮 -->
          <div class="flex items-center gap-2">
            <button class="btn btn-primary  btn-sm" @click="gotoDependency()">
              <v-icon name="ri-tools-line" class="w-4 h-4 mr-1"></v-icon>
              {{ $t('download.manage_dependencies') }}
            </button>
            <button class="btn  btn-sm" @click="checkDependencies()">
              <v-icon name="ri-refresh-line" class="w-4 h-4 mr-1"></v-icon>
              {{ $t('common.refresh') }}
            </button>
          </div>
          <!-- 取消按钮 -->
          <button class="btn btn-ghost btn-sm" @click="closeModal">{{ $t('common.cancel') }}</button>
        </div>
      </div>

      <div v-else>
        <!-- 自定义下载模式 -->
        <div v-if="mode === 'custom'" class="py-4">
          <form class="flex flex-col gap-4">
            <!-- URL area -->
            <div class="form-control w-full">
              <div class="flex gap-2">
                <input type="text" v-model="url" :placeholder="$t('download.video_url_placeholder')"
                  class="input input-bordered flex-1 input-sm" />
                <button @click="handleParse" type="button" class="btn btn-primary btn-sm" :disabled="isLoading">
                  <div class="flex items-center justify-center">
                    <v-icon v-if="isLoading" name="ri-loader-2-line" class="animate-spin h-4 w-4 mr-1"></v-icon>
                    <span>{{ isLoading ? $t('download.parsing') : $t('download.parse') }}</span>
                  </div>
                </button>
              </div>
            </div>

            <!-- Parse result area -->
            <template v-if="videoData?.title">
              <div class="divider"></div>

              <!-- Video title and preview information -->
              <div class="flex items-center gap-4">
                <div class="w-32 h-20 bg-base-200 rounded-lg overflow-hidden flex-shrink-0">
                  <template v-if="videoData?.thumbnail">
                    <ProxiedImage :src="videoData.thumbnail" :alt="$t('download.thumbnail')"
                      class="w-full h-full object-cover rounded-md" error-icon="ri-video-line" />
                  </template>
                  <div v-else
                    class="w-full h-full flex flex-col items-center justify-center text-base-content/30 bg-base-200">
                    <v-icon name="ri-video-line" class="w-8 h-8 mb-1"></v-icon>
                    <span class="text-xs">{{ $t('download.thumbnail') }}</span>
                  </div>
                </div>
                <div class="flex-1">
                  <div class="text-base font-medium">{{ videoData.title }}</div>
                  <div class="text-sm text-base-content/70 mt-1">{{ videoData.duration }} · {{ videoData.author }}</div>
                </div>
              </div>

              <!-- download options -->
              <div class="space-y-4 mt-2">
                <!-- video, subtitle, and transcode selection -->
                <div class="grid grid-cols-4 gap-4">
                  <!-- video quality selection -->
                  <div class="form-control w-full col-span-2">
                    <label class="label">
                      <span class="label-text">{{ $t('download.video_quality') }}</span>
                    </label>
                    <select v-model="selectedQuality" class="select select-bordered w-full select-sm">
                      <option disabled value="">{{ $t('download.select_video_quality') }}</option>
                      <template v-for="quality in formatQualities(videoData?.formats || [])"
                        :key="quality.id || quality.format_id">
                        <option v-if="quality.isHeader" disabled
                          class="!bg-base-200 !text-base-content/70 !font-semibold !py-2 !my-1 !border-t !border-base-300">
                          ▾ {{ quality.label }}
                        </option>
                        <option v-else :value="quality" class="label-text-alt text-base-content/70">
                          {{ quality.label }}
                        </option>
                      </template>
                    </select>
                  </div>

                  <!-- subtitle language selection (multi-select) -->
                  <div class="form-control w-full">
                    <label class="label">
                      <span class="label-text">{{ $t('download.subtitle_language') }}</span>
                    </label>
                    <div class="dropdown w-full">
                      <div tabindex="0" role="button"
                        class="select select-bordered w-full flex items-center justify-between select-sm"
                        @click="showSubtitleDropdown = !showSubtitleDropdown">
                        <span v-if="selectedSubtitles.length === 0" class="text-base-content/50">{{
                          $t('download.no_subtitles')
                          }}</span>
                        <span v-else>{{ $t('download.selected') }} {{ selectedSubtitles.length }} {{
                          $t('download.subtitles')
                          }}</span>
                        <v-icon name="ri-arrow-down-s-line" class="w-4 h-4"></v-icon>
                      </div>
                      <div v-if="showSubtitleDropdown" tabindex="0"
                        class="dropdown-content z-[1] menu p-2 shadow bg-base-100 rounded-box w-full max-h-60 overflow-y-auto">
                        <div v-for="subtitle in videoData.subtitles" :key="subtitle.value.lang"
                          class="p-2 hover:bg-base-200 rounded cursor-pointer flex items-center"
                          @click.stop="toggleSubtitle(subtitle.value)">
                          <input type="checkbox" :checked="isSubtitleSelected(subtitle.value)"
                            class="checkbox checkbox-sm mr-2" />
                          <span>{{ subtitle.label }}</span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <div class="form-control w-full">
                    <label class="label">
                      <span class="label-text">{{ $t('download.transcoding') }}</span>
                    </label>
                    <select v-model="selectedTranscodeFormat" class="select select-bordered w-full select-sm">
                      <option :value=0>{{ $t('download.no_transcoding') }}</option>
                      <template v-if="availableTranscodeFormats?.video?.length">
                        <optgroup :label="$t('download.video_formats')">
                          <option v-for="format in availableTranscodeFormats.video" :key="format.id" :value="format.id">
                            {{ format.name }}
                          </option>
                        </optgroup>
                      </template>
                      <template v-if="availableTranscodeFormats?.audio?.length">
                        <optgroup :label="$t('download.audio_formats')">
                          <option v-for="format in availableTranscodeFormats.audio" :key="format.id" :value="format.id">
                            {{ format.name }}
                          </option>
                        </optgroup>
                      </template>
                    </select>
                  </div>
                </div>
              </div>

              <!-- Pipeline Options -->
              <div class="mt-4">
                <label class="label">
                  <span class="label-text">{{ $t('download.pipeline') }}</span>
                </label>
                <div class="bg-base-200 p-2 rounded-lg">
                  <div class="flex items-center">
                    <template v-for="(step, index) in pipelineSteps" :key="index">
                      <!-- Step arrow -->
                      <template v-if="index > 0">
                        <div class="flex-none mx-2">
                          <v-icon name="ri-arrow-right-line" class="w-5 h-5 text-base-content/50"></v-icon>
                        </div>
                      </template>
                      <!-- Step content -->
                      <div class="flex-1 flex items-center">
                        <div
                          :class="[step.bg, 'w-6 h-6 rounded-full flex items-center justify-center text-white relative group']">
                          <v-icon :name="step.icon"
                            class="w-4 h-4 absolute opacity-0 group-hover:opacity-100 transition-opacity"></v-icon>
                          <span class="group-hover:opacity-0 transition-opacity">{{ step.number }}</span>
                        </div>
                        <div class="ml-2">
                          <div class="font-medium text-sm">{{ step.title }}</div>
                          <div class="text-xs text-base-content/70">{{ step.desc }}</div>
                        </div>
                      </div>
                    </template>
                  </div>
                </div>
              </div>
            </template>
          </form>
        </div>

        <div v-if="mode === 'quick'" class="py-4">
          <form class="flex flex-col gap-4" @submit.prevent>
            <!-- URL area -->
            <div class="form-control w-full">
              <div class="flex gap-2">
                <input type="text" v-model="url" :placeholder="$t('download.video_url_placeholder')"
                  class="input input-bordered flex-1 input-sm" />
                <div class="flex space-x-2">
                  <select v-model="video" class="select select-bordered select-sm w-full max-w-xs">
                    <option v-for="item in videoItems" :key="item.key" :value="item.key">
                      {{ item.label }}
                    </option>
                  </select>

                  <select v-model="bestCaption" class="select select-bordered select-sm w-full max-w-xs">
                    <option v-for="item in captionItems" :key="item.key" :value="item.key">
                      {{ item.label }}
                    </option>
                  </select>

                  <select v-model="quickSelectedTranscodeFormat" class="select select-bordered select-sm w-full max-w-xs">
                    <option :value="0">{{ $t('download.no_transcoding') }}</option>
                    <template v-if="quickTranscodeOptions.video.length > 0">
                      <optgroup :label="$t('download.video_formats')">
                        <option v-for="format in quickTranscodeOptions.video" :key="format.id" :value="format.id">
                          {{ format.name }}
                        </option>
                      </optgroup>
                    </template>
                    <template v-if="quickTranscodeOptions.audio.length > 0">
                      <optgroup :label="$t('download.audio_formats')">
                        <option v-for="format in quickTranscodeOptions.audio" :key="format.id" :value="format.id">
                          {{ format.name }}
                        </option>
                      </optgroup>
                    </template>
                  </select>
                </div>
              </div>
            </div>
          </form>
        </div>

        <!-- 开始/取消按钮 -->
        <!-- 自定义模式 -->
        <div v-if="mode === 'custom'" class="modal-action">
          <button class="btn btn-sm" @click="closeModal">{{ $t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" @click="startCustomDownload" :disabled="!canDownload">{{
            $t('common.start')
            }}</button>
        </div>
        <!-- 快速下载模式 -->
        <div v-if="mode === 'quick'" class="modal-action">
          <button class="btn btn-sm" @click="closeModal">{{ $t('common.cancel') }}</button>
          <button class="btn btn-primary btn-sm" @click="startQuickDownload" :disabled="!quickModeDownEnabled">{{
            $t('common.start')
          }}</button>
        </div>

      </div>
    </div>
  </dialog>
</template>

<script setup>
import { ref, computed, watch, onMounted } from 'vue'
import { GetContent, Download, QuickDownload, GetFormats } from 'wailsjs/go/api/DowntasksAPI'
import { DependenciesReady } from 'wailsjs/go/api/DependenciesAPI'
import useNavStore from '@/stores/nav.js'
import useSettingsStore from '@/stores/settings'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'

// i18n
const { t } = useI18n()

const settingsStore = useSettingsStore()
const navStore = useNavStore()

const props = defineProps({
  show: Boolean,
  initialMode: {
    type: String,
    default: 'custom',
    validator: (value) => ['custom', 'quick'].includes(value)
  }
})

// 模式切换
const mode = ref(props.initialMode)

const emit = defineEmits(['update:show', 'download-started'])

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const dependenciesReady = ref(false)

const checkDependencies = async () => {
  // reset dependenciesReady
  dependenciesReady.value = false
  try {
    const response = await DependenciesReady()
    if (response.success) {
      dependenciesReady.value = true
    } else {
      $message.warning(response.msg)
    }
  } catch (error) {
    $message.error(error.message)
  }
}

const gotoDependency = () => {
  navStore.setNav('settings')
  settingsStore.setPage('dependency')
}

// Watch modal visibility
watch(() => showModal.value, async(newValue) => {
  if (newValue) {
    // Modal opened
    await checkDependencies()
    // get formats
    if (dependenciesReady.value) {
      await getFormats()
    }
  } else {
    // Modal closed, reset form
    resetForm()
  }
})

// Form data
const url = ref('')

// custom mode data
const isLoading = ref(false)
const videoData = ref(null)
const selectedQuality = ref(null)
const selectedSubtitles = ref([])
const translateTo = ref('')
const subtitleStyle = ref('default')
const showSubtitleDropdown = ref(false)
const availableTranscodeFormats = ref([])
const selectedTranscodeFormat = ref(0) // New state for transcoding
const quickSelectedTranscodeFormat = ref(0) // New state for transcoding

// quick mode data
const video = ref('best')
const bestCaption = ref(false)
const videoItems = computed(() => [
  { key: 'best', label: t('download.best_quality') },
  { key: 'bestaudio', label: t('download.best_audio') },
])
const captionItems = computed(() => [
  { key: false, label: t('download.no_caption') },
  { key: true, label: t('download.best_caption') },
])
const quickTranscodeOptions = computed(() => {
  const options = {
    video: [],
    audio: []
  };

  if (!availableTranscodeFormats.value || (!availableTranscodeFormats.value.video && !availableTranscodeFormats.value.audio)) {
    return options;
  }

  if (video.value === 'best') {
    if (availableTranscodeFormats.value.video) {
      options.video = [...availableTranscodeFormats.value.video];
    }
    if (availableTranscodeFormats.value.audio) {
      options.audio = [...availableTranscodeFormats.value.audio];
    }
  } else if (video.value === 'bestaudio') {
    if (availableTranscodeFormats.value.audio) {
      options.audio = [...availableTranscodeFormats.value.audio];
    }
  }
  return options;
});

watch(video, () => {
  quickSelectedTranscodeFormat.value = 0 // Reset transcode selection to "No Transcoding"
})

// Whether can start download
const canDownload = computed(() => {
  return videoData.value && selectedQuality.value
})

// Format video quality options
const formatQualities = (formats) => {
  if (!formats?.length) return []

  // Group by format type
  const formatGroups = new Map()
  const audioFormats = []

  formats.forEach(format => {
    const hasVideo = format.vcodec && format.vcodec !== 'none'
    const hasAudio = format.acodec && format.acodec !== 'none'
    const hasSize = format.filesize || format.filesize_approx

    // Skip formats without size information
    if (!hasSize) return
    const formatInfo = {
      ...format,
      hasVideo,
      hasAudio,
      hasSize,
      resolution: hasVideo || hasSize ? `${format.height}P${format.fps ? ` ${Math.round(Number(format.fps))}FPS` : ''}` : null,
      size: format.filesize ?? format.filesize_approx,
      formatType: format.ext || format.container || 'unknown'
    }

    // Audio and video formats are stored separately
    if (!hasVideo && hasAudio) {
      audioFormats.push(formatInfo)
    } else if (hasVideo) {
      if (!formatGroups.has(formatInfo.formatType)) {
        formatGroups.set(formatInfo.formatType, [])
      }
      formatGroups.get(formatInfo.formatType).push(formatInfo)
    } else if (!hasVideo && !hasAudio && hasSize) {
      if (!formatGroups.has(formatInfo.formatType)) {
        formatGroups.set(formatInfo.formatType, [])
      }
      formatGroups.get(formatInfo.formatType).push(formatInfo)
    }
  })

  const result = []

  // Process video formats
  formatGroups.forEach((formats, formatType) => {
    // Add format type header
    result.push({
      id: `group-${formatType}`,
      label: `${formatType.toUpperCase()}`,
      isHeader: true,
      disabled: true
    })

    // Group by resolution and select best quality
    const resolutionGroups = new Map()
    formats.forEach(format => {
      const key = format.resolution
      if (!resolutionGroups.has(key)) {
        resolutionGroups.set(key, [])
      }
      resolutionGroups.get(key).push(format)
    })

    // Select best quality from each resolution group
    const qualityOptions = []
    resolutionGroups.forEach((items, resolution) => {
      const bestFormat = items.reduce((best, current) =>
        current.size > best.size ? current : best, items[0])

      qualityOptions.push({
        ...bestFormat,
        label: bestFormat.hasAudio
          ? `${resolution} ${formatFileSize(bestFormat.size)}`
          : `${resolution} - ${formatFileSize(bestFormat.size)}`,
        type: bestFormat.hasAudio ? 'combined' : 'video_only'
      })
    })

    // Sort by resolution and add to result
    qualityOptions
      .sort((a, b) => {
        if (b.height !== a.height) return b.height - a.height
        return (b.fps || 0) - (a.fps || 0)
      })
      .forEach(option => result.push(option))
  })

  // Process audio formats
  if (audioFormats.length > 0) {
    // Add audio group header
    result.push({
      id: 'group-audio',
      label: 'Audio Only',
      isHeader: true,
      disabled: true
    })

    // Sort by bitrate
    audioFormats
      .sort((a, b) => (b.abr || 0) - (a.abr || 0))
      .forEach(format => {
        result.push({
          ...format,
          label: `${format.abr}kbps - ${formatFileSize(format.size)}`,
          type: 'audio_only'
        })
      })
  }

  return result
}

// Format subtitle options
const formatSubtitles = (subtitles) => {
  const result = []

  // Process regular subtitles
  if (subtitles && Object.keys(subtitles).length > 0) {
    Object.entries(subtitles).forEach(([lang, subs]) => {
      if (subs?.length > 0) {
        result.push({
          value: {
            lang,
            url: subs[0].url,
            isAuto: false
          },
          label: `${lang}`
        })
      }
    })
  }

  return result.sort((a, b) => a.label.localeCompare(b.label))
}

// Format file size
const formatFileSize = (size) => {
  if (!size) return t('download.unknown_size')

  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let fileSize = size

  while (fileSize >= 1024 && i < units.length - 1) {
    fileSize /= 1024
    i++
  }

  return `${fileSize.toFixed(2)} ${units[i]}`
}

// Calculate pipeline steps
const pipelineSteps = computed(() => {
  const steps = []

  // Step 1: Download (required)
  steps.push({
    number: 1,
    title: t('download.download_video'),
    desc: t('download.download_video_desc'),
    bg: 'bg-primary',
    icon: 'ri-download-cloud-line'
  })

  let currentStep = 2

  if (selectedSubtitles.value.length > 0) {
    // If selected subtitles and need translation
    if (translateTo.value) {
      steps.push({
        number: currentStep++,
        title: t('download.translate'),
        desc: t('download.translate_desc'),
        bg: 'bg-primary/80',
        icon: 'ri-translate-2'
      })
    }

    // If selected subtitles and need to embed
    if (subtitleStyle.value !== 'default') {
      steps.push({
        number: currentStep++,
        title: t('download.process_video'),
        desc: t('download.process_video_desc'),
        bg: 'bg-primary/60',
        icon: 'ri-movie-line'
      })
    }
  }

  // Last step: Complete (required)
  steps.push({
    number: currentStep,
    title: t('download.complete'),
    desc: t('download.complete_desc'),
    bg: 'bg-success',
    icon: 'ri-check-line'
  })

  return steps
})

const isSubtitleSelected = (subtitle) => {
  return selectedSubtitles.value.includes(subtitle)
}

const toggleSubtitle = (subtitle) => {
  if (isSubtitleSelected(subtitle)) {
    selectedSubtitles.value = selectedSubtitles.value.filter(s => s !== subtitle)
  } else {
    selectedSubtitles.value.push(subtitle)
  }
}

// Parse video URL
const getFormats = async () => {
  try {
    const response = await GetFormats()
    if (response.success) {
      availableTranscodeFormats.value = JSON.parse(response.data)
    } else {
      $message.warning(response.msg)
    }
  } catch (error) {
    $message.error(error.message)
  }
}

const handleParse = async () => {
  if (!url.value) return

  // get video info
  try {
    isLoading.value = true
    const response = await GetContent(url.value)
    if (response.success) {
      const data = JSON.parse(response.data)
      videoData.value = {
        title: data.title,
        author: data.uploader || data.channel || data.extractor,
        duration: formatDuration(data.duration),
        thumbnail: data.thumbnail ? (data.thumbnail.startsWith('http:') ? data.thumbnail.replace('http:', 'https:') : data.thumbnail) : '',
        formats: data.formats,
        subtitles: formatSubtitles(data.subtitles)
      }

      // Default select the first non-header format
      const qualities = formatQualities(videoData.value.formats || [])
      selectedQuality.value = qualities.find(q => !q.isHeader) || null

      // Default do not select subtitles
      selectedSubtitles.value = []
      translateTo.value = ''
      subtitleStyle.value = 'default'
      selectedTranscodeFormat.value = 0 // Reset transcode selection
    } else {
      $dialog.error({
        title: t('download.parse_failed'),
        content: response.msg,
      })
    }
  } catch (error) {
    $dialog.error({
      title: t('download.parse_failed'),
      content: error.message,
    })
  } finally {
    isLoading.value = false
  }
}

// 格式化时长
const formatDuration = (seconds) => {
  if (!seconds) return t('download.unknown_duration')

  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)

  if (hours > 0) {
    return `${hours}:${minutes.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  } else {
    return `${minutes}:${secs.toString().padStart(2, '0')}`
  }
}

// Start download
const startCustomDownload = async () => {
  if (!canDownload.value) return

  try {
    // Prepare download parameters
    const downloadParams = {
      url: url.value,
      formatId: selectedQuality.value.format_id,
      downloadSubs: selectedSubtitles.value.length > 0,
      subLangs: selectedSubtitles.value.map(sub => sub.lang),
      subFormat: "",
      translateTo: translateTo.value,
      subtitleStyle: subtitleStyle.value,
      recodeFormatNumber: selectedTranscodeFormat.value
    }

    const response = await Download(downloadParams)

    if (response.success) {
      const data = JSON.parse(response.data)
      // Trigger download start event
      emit('download-started', {
        id: data.id,
        title: videoData.value.title,
        url: url.value,
        quality: selectedQuality.value.label,
        thumbnail: videoData.value.thumbnail,
        status: data.status,
        progress: 0,
        currentStage: 'download',
        createdAt: new Date().toISOString()
      })

      // Close modal
      showModal.value = false
    } else {
      $dialog.error({
        title: t('download.download_failed'),
        content: response.msg,
      })
    }
  } catch (error) {
    $dialog.error({
      title: t('download.download_failed'),
      content: error.message,
    })
  }
}

const quickModeDownEnabled = computed(() => {
  return url.value !== '' && video.value !== '' && bestCaption.value !== null
})

// Start download
const startQuickDownload = async () => {
  if (!quickModeDownEnabled.value) return

  try {
    // Prepare download parameters
    const downloadParams = {
      url: url.value,
      video: video.value,
      bestCaption: bestCaption.value,
      recodeFormatNumber: quickSelectedTranscodeFormat.value
    }

    const response = await QuickDownload(downloadParams)

    // reset form
    quickSelectedTranscodeFormat.value = 0
    if (response.success) {
      // 触发下载开始事件
      emit('download-started', {
        url: url.value,
        video: video.value,
        bestCaption: bestCaption.value,
        createdAt: new Date().toISOString()
      })
      // Close modal
      closeModal()
    } else {
      $dialog.error({
        title: t('download.download_failed'),
        content: response.msg,
      })
    }
  } catch (error) {
    $dialog.error({
      title: t('download.download_failed'),
      content: error.message || t('common.unknown_error'),
    })
  }
}

// Close modal
const closeModal = () => {
  showModal.value = false
}

// Reset form
const resetForm = () => {
  url.value = ''
  if (mode.value === 'quick') {
    video.value = 'best'
    bestCaption.value = false
    quickSelectedTranscodeFormat.value = 0
  } else {
    videoData.value = null
    selectedQuality.value = null
    selectedSubtitles.value = []
    translateTo.value = ''
    subtitleStyle.value = 'default'
    selectedTranscodeFormat.value = 0 
  }
}

</script>

<style scoped>
.modal-box {
  max-height: 85vh;
  overflow-y: auto;
}

.animate-fade-in {
  animation: fadeIn 0.3s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-10px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}
</style>