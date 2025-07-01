<template>
  <div v-if="showModal" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
    <div class="modal-container">
      <!-- Modal Header -->
      <div class="modal-header">
        <div class="header-content">
          <div class="header-icon">
            <v-icon name="ri-download-cloud-line" class="w-5 h-5 text-primary"></v-icon>
          </div>
          <h3 class="modal-title">
            {{ mode === 'quick' ? $t('download.quick_task') : $t('download.new_task') }}
          </h3>
        </div>
        <button @click="closeModal" class="close-button">
          <v-icon name="ri-close-line" class="w-4 h-4"></v-icon>
        </button>
      </div>

      <!-- Modal Content -->
      <div class="modal-content">
        <!-- Mode Switcher -->
        <div class="mode-switcher">
          <div class="switcher-track">
            <button 
              class="switcher-option" 
              :class="{ 'active': mode === 'custom' }" 
              @click="mode = 'custom'"
            >
              {{ $t('download.custom_mode') }}
            </button>
            <button 
              class="switcher-option" 
              :class="{ 'active': mode === 'quick' }" 
              @click="mode = 'quick'"
            >
              {{ $t('download.quick_mode') }}
            </button>
          </div>
        </div>

        <!-- Dependencies Check -->
        <div v-if="!dependenciesReady" class="dependencies-warning">
          <div class="warning-content">
            <div class="warning-icon">
              <v-icon name="ri-error-warning-line" class="w-8 h-8 text-orange-500"></v-icon>
            </div>
            <div class="warning-text">
              <h4>{{ $t('download.dependencies_not_ready') }}</h4>
              <p>{{ $t('download.dependencies_not_ready_desc') }}</p>
            </div>
          </div>
          <div class="warning-actions">
            <button class="btn-macos-primary btn-macos-sm" @click="gotoDependency()">
              <v-icon name="ri-tools-line" class="w-4 h-4 mr-2"></v-icon>
              {{ $t('download.manage_dependencies') }}
            </button>
            <button class="btn-macos-secondary btn-macos-sm" @click="checkDependencies()">
              <v-icon name="ri-refresh-line" class="w-4 h-4 mr-2"></v-icon>
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>

        <!-- Main Content -->
        <div v-else class="main-content">
          <!-- Custom Mode -->
          <div v-if="mode === 'custom'" class="custom-mode">
            <!-- URL Input Section -->
            <div class="input-section">
              <label class="section-label">{{ $t('download.video_url') }}</label>
              
              <!-- URL输入框和browser选择、parse按钮在同一行 -->
              <div class="url-input-row">
                <input 
                  type="text" 
                  v-model="url" 
                  @input="onUrlInput"
                  :placeholder="$t('download.video_url_placeholder')"
                  class="input-macos url-input"
                />
                
                <!-- Browser选择和Parse按钮（仅在有URL且获取到browser信息后显示） -->
                <transition name="slide-in-right">
                  <div v-if="url.trim() && (availableBrowsers.length > 0 || !isLoadingBrowsers)" 
                       :class="['browser-controls', mode === 'custom' ? 'browser-controls-custom' : 'browser-controls-quick']">
                    <select 
                      v-model="browser" 
                      class="select-macos browser-select" 
                      :disabled="isLoadingBrowsers"
                    >
                      <option v-for="option in browserOptions" :key="option.value" :value="option.value">
                        {{ option.label }}
                      </option>
                    </select>
                    
                    <!-- Parse按钮只在custom模式显示 -->
                    <button 
                      v-if="mode === 'custom'"
                      @click="handleParse" 
                      type="button" 
                      class="btn-macos-primary" 
                      :disabled="isLoading || !url.trim()"
                    >
                      <v-icon 
                        v-if="isLoading" 
                        name="ri-loader-2-line" 
                        class="animate-spin w-4 h-4 mr-2"
                      ></v-icon>
                      <span>{{ isLoading ? $t('download.parsing') : $t('download.parse') }}</span>
                    </button>
                  </div>
                </transition>
              </div>
              
              <!-- 加载状态提示（显示在下方） -->
              <div v-if="isLoadingBrowsers" class="loading-indicator-below">
                <v-icon name="ri-loader-2-line" class="animate-spin w-4 h-4 text-blue-500"></v-icon>
                <span class="loading-text">{{ $t('download.checking_cookies') }}</span>
              </div>
            </div>

            <!-- 分割线 -->
            <div v-if="videoData?.title" class="section-divider"></div>

            <!-- Video Info Card -->
            <div v-if="videoData?.title" class="video-info-card">
              <div class="video-preview">
                <div class="thumbnail-container">
                  <ProxiedImage 
                    v-if="videoData?.thumbnail"
                    :src="videoData.thumbnail" 
                    :alt="$t('download.thumbnail')"
                    class="thumbnail-image" 
                    error-icon="ri-video-line" 
                  />
                  <div v-else class="thumbnail-placeholder">
                    <v-icon name="ri-video-line" class="w-8 h-8 text-base-content/30"></v-icon>
                  </div>
                </div>
                <div class="video-meta">
                  <h4 class="video-title">{{ videoData.title }}</h4>
                  <p class="video-details">{{ videoData.duration }} · {{ videoData.author }}</p>
                </div>
              </div>

              <!-- Download Options -->
              <div class="download-options">
                <div class="options-grid">
                  <!-- Video Quality -->
                  <div class="option-group">
                    <label for="video-quality-select" class="option-label">{{ $t('download.video_quality') }}</label>
                    <select id="video-quality-select" v-model="selectedQuality" class="select-macos">
                      <option disabled value="">{{ $t('download.select_video_quality') }}</option>
                      <template v-for="quality in formatQualities(videoData?.formats || [])" :key="quality.id || quality.format_id">
                        <option v-if="quality.isHeader" disabled class="option-header">
                          ▾ {{ quality.label }}
                        </option>
                        <option v-else :value="quality">
                          {{ quality.label }}
                        </option>
                      </template>
                    </select>
                  </div>

                  <!-- Subtitle Selection -->
                  <div class="option-group">
                    <label for="subtitle-lang-select" class="option-label">{{ $t('download.subtitle_language') }}</label>
                    <select id="subtitle-lang-select" v-model="selectedSubtitleLang" class="select-macos" @change="handleSubtitleChange">
                      <option value="">{{ $t('download.no_subtitles') }}</option>
                      <template v-if="videoData?.subtitles?.length">
                        <option v-for="subtitle in videoData.subtitles" :key="subtitle.value.lang" :value="subtitle.value.lang">
                          {{ subtitle.label }}
                        </option>
                      </template>
                    </select>
                  </div>

                  <!-- Transcoding -->
                  <div class="option-group">
                    <label for="transcode-format-select" class="option-label">{{ $t('download.transcoding') }}</label>
                    <select id="transcode-format-select" v-model="selectedTranscodeFormat" class="select-macos">
                      <option :value="0">{{ $t('download.no_transcoding') }}</option>
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
            </div>
          </div>

          <!-- Quick Mode -->
          <div v-if="mode === 'quick'" class="quick-mode">
            <!-- URL Input -->
            <div class="input-section">
              <label class="section-label">{{ $t('download.video_url') }}</label>
              
              <!-- URL输入框和browser选择在同一行 -->
              <div class="url-input-row">
                <input 
                  type="text" 
                  v-model="url" 
                  @input="onUrlInput"
                  :placeholder="$t('download.video_url_placeholder')"
                  class="input-macos url-input"
                />
                
                <!-- Browser选择（仅在有URL且获取到browser信息后显示） -->
                <transition name="slide-in-right">
                  <div v-if="url.trim() && (availableBrowsers.length > 0 || !isLoadingBrowsers)" 
                       :class="['browser-controls', mode === 'custom' ? 'browser-controls-custom' : 'browser-controls-quick']">
                    <select 
                      v-model="browser" 
                      class="select-macos browser-select" 
                      :disabled="isLoadingBrowsers"
                    >
                      <option v-for="option in browserOptions" :key="option.value" :value="option.value">
                        {{ option.label }}
                      </option>
                    </select>
                  </div>
                </transition>
              </div>
              
              <!-- 加载状态提示（显示在下方） -->
              <div v-if="isLoadingBrowsers" class="loading-indicator-below">
                <v-icon name="ri-loader-2-line" class="animate-spin w-4 h-4 text-blue-500"></v-icon>
                <span class="loading-text">{{ $t('download.checking_cookies') }}</span>
              </div>
            </div>

            <!-- Quick Options（仅在获取到browser信息后显示） -->
            <transition name="fade-in">
              <div v-if="url.trim() && (availableBrowsers.length > 0 || !isLoadingBrowsers)" class="quick-options">
                <div class="options-grid">
                  <div class="option-group">
                    <label for="quick-video-quality-select" class="option-label">{{ $t('download.video_quality') }}</label>
                    <select id="quick-video-quality-select" v-model="video" class="select-macos">
                      <option v-for="item in videoItems" :key="item.key" :value="item.key">
                        {{ item.label }}
                      </option>
                    </select>
                  </div>

                  <div class="option-group">
                    <label for="quick-subtitle-lang-select" class="option-label">{{ $t('download.subtitle_language') }}</label>
                    <select id="quick-subtitle-lang-select" v-model="bestCaption" class="select-macos">
                      <option v-for="item in captionItems" :key="item.key" :value="item.key">
                        {{ item.label }}
                      </option>
                    </select>
                  </div>

                  <div class="option-group">
                    <label for="quick-transcode-format-select" class="option-label">{{ $t('download.transcoding') }}</label>
                    <select id="quick-transcode-format-select" v-model="quickSelectedTranscodeFormat" class="select-macos">
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
            </transition>
          </div>
        </div>
      </div>

      <!-- Modal Footer -->
      <div class="modal-footer">
        <button class="btn-macos-secondary" @click="closeModal">
          {{ $t('common.cancel') }}
        </button>
        <button 
          v-if="mode === 'custom' && canDownload"
          class="btn-macos-primary" 
          @click="startCustomDownload" 
          :disabled="!canDownload"
        >
          {{ $t('common.start') }}
        </button>
        <button 
          v-if="mode === 'quick' && quickModeDownEnabled"
          class="btn-macos-primary" 
          @click="startQuickDownload" 
          :disabled="!quickModeDownEnabled"
        >
          {{ $t('common.start') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { GetContent, Download, QuickDownload, GetFormats } from 'wailsjs/go/api/DowntasksAPI'
import { DependenciesReady } from 'wailsjs/go/api/DependenciesAPI'
import { GetBrowserByDomain } from 'wailsjs/go/api/CookiesAPI'
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
watch(() => showModal.value, async (newValue) => {
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
const selectedSubtitleLang = ref('')
const translateTo = ref('')
const subtitleStyle = ref('default')
const showSubtitleDropdown = ref(false)
const availableTranscodeFormats = ref([])
const selectedTranscodeFormat = ref(0) // New state for transcoding
const quickSelectedTranscodeFormat = ref(0) // New state for transcoding
const browser = ref('')
const availableBrowsers = ref([])
const isLoadingBrowsers = ref(false)

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

// 浏览器选项计算属性
const browserOptions = computed(() => {
  const options = [{ value: '', label: '无' }]
  if (availableBrowsers.value && availableBrowsers.value.length > 0) {
    availableBrowsers.value.forEach(browser => {
      options.push({ value: browser, label: browser })
    })
  }
  return options
})

// 获取可用浏览器的方法
const fetchAvailableBrowsers = async (url) => {
  if (!url) {
    availableBrowsers.value = []
    browser.value = ''
    return
  }

  try {
    isLoadingBrowsers.value = true
    const response = await GetBrowserByDomain(url)
    if (response.success) {
      const browsers = JSON.parse(response.data)
      availableBrowsers.value = browsers || []
      // 重置browser选择
      browser.value = ''
    } else {
      availableBrowsers.value = []
      browser.value = ''
    }
  } catch (error) {
    console.error('Failed to fetch browsers:', error)
    availableBrowsers.value = []
    browser.value = ''
  } finally {
    isLoadingBrowsers.value = false
  }
}

watch(video, () => {
  quickSelectedTranscodeFormat.value = 0 // Reset transcode selection to "No Transcoding"
})

// 添加URL输入处理方法
const onUrlInput = (event) => {
  const newUrl = event.target.value
  // 防抖处理，避免频繁调用API
  clearTimeout(urlInputTimer.value)
  urlInputTimer.value = setTimeout(() => {
    if (newUrl.trim() && newUrl !== url.value) {
      fetchAvailableBrowsers(newUrl)
    } else if (!newUrl.trim()) {
      // 清空URL时重置browser相关状态
      availableBrowsers.value = []
      browser.value = ''
    }
  }, 500)
}

// 添加防抖timer
const urlInputTimer = ref(null)

// 监听URL变化
watch(url, (newUrl) => {
  fetchAvailableBrowsers(newUrl)
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

// 处理字幕选择变化
const handleSubtitleChange = (event) => {
  const value = event.target.value
  if (value === '') {
    selectedSubtitles.value = []
  } else {
    const subtitle = videoData.value?.subtitles?.find(sub => sub.value.lang === value)
    selectedSubtitles.value = subtitle ? [subtitle.value] : []
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
    const response = await GetContent(url.value, browser.value) // enable cookies
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
      browser: browser.value,
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
      browser: browser.value,
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
  browser.value = ''
  availableBrowsers.value = []
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

<style lang="scss" scoped>
// 模态框容器
.modal-container {
  position: relative;
  width: 90%;
  max-width: 640px;
  max-height: 90vh;
  background: var(--macos-background);
  border-radius: 12px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  animation: slideInUp 0.3s ease;
}

// Header样式
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--macos-blue);
  border-radius: 8px;
  color: white;
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0;
}

.close-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: var(--macos-gray);
  border-radius: 6px;
  color: var(--macos-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
  
  &:hover {
    background: var(--macos-gray-hover);
    color: var(--macos-text-primary);
  }
}

// Content样式
.modal-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

// Mode Switcher
.mode-switcher {
  margin-bottom: 24px;
}

.switcher-track {
  display: flex;
  background: var(--macos-gray);
  border-radius: 8px;
  padding: 2px;
  position: relative;
}

.switcher-option {
  flex: 1;
  padding: 8px 16px;
  border: none;
  background: transparent;
  color: var(--macos-text-secondary);
  font-size: 13px;
  font-weight: 500;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  position: relative;
  z-index: 1;
  
  &.active {
    background: var(--macos-background);
    color: var(--macos-text-primary);
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }
}

// Dependencies Warning
.dependencies-warning {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  padding: 40px 20px;
  text-align: center;
}

.warning-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
}

.warning-text {
  h4 {
    font-size: 16px;
    font-weight: 600;
    color: var(--macos-text-primary);
    margin: 0 0 4px 0;
  }
  
  p {
    font-size: 13px;
    color: var(--macos-text-secondary);
    margin: 0;
  }
}

.warning-actions {
  display: flex;
  gap: 12px;
}

// Main Content
.main-content {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

// Input Section
.input-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--macos-text-primary);
}

// URL输入行样式
.url-input-row {
  display: flex;
  gap: 8px;
  align-items: center;
}

.url-input {
  flex: 1;
  min-width: 0;
}

// Browser控件容器
.browser-controls {
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.browser-controls-custom {
  gap: 8px;
  
  .browser-select {
    width: 120px;
    flex-shrink: 0;
  }
}

.browser-controls-quick {
  .browser-select {
    width: 120px;
    flex-shrink: 0;
  }
}

// 加载状态
.loading-indicator-below {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--macos-text-secondary);
}

.loading-text {
  font-size: 12px;
  color: var(--macos-text-secondary);
}

// 动画
.slide-in-right-enter-active,
.slide-in-right-leave-active {
  transition: all 0.3s ease;
}

.slide-in-right-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.slide-in-right-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

.fade-in-enter-active,
.fade-in-leave-active {
  transition: all 0.3s ease;
}

.fade-in-enter-from {
  opacity: 0;
  transform: translateY(-10px);
}

.fade-in-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}

// Video Info Card
.video-info-card {
  background: var(--macos-background-secondary);
  border: 1px solid var(--macos-separator);
  border-radius: 10px;
  overflow: hidden;
}

.video-preview {
  display: flex;
  gap: 16px;
  padding: 16px;
  border-bottom: 1px solid var(--macos-separator);
}

.thumbnail-container {
  width: 120px;
  height: 68px;
  border-radius: 8px;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--macos-gray);
}

.thumbnail-image {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumbnail-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--macos-gray);
}

.video-meta {
  flex: 1;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
}

.video-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0;
  line-height: 1.3;
}

.video-details {
  font-size: 12px;
  color: var(--macos-text-secondary);
  margin: 0;
}

// Download Options
.download-options {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.options-grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  gap: 16px;
}

.option-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.option-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--macos-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.select-macos option.option-header {
  background: var(--macos-background-secondary);
  color: var(--macos-text-secondary);
  font-weight: 600;
}

// Quick Mode
.quick-mode {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.quick-options {
  margin-top: 20px;
  
  .options-grid {
    grid-template-columns: 1fr 1fr 1fr;
  }
}

// Footer
.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: var(--macos-background-secondary);
  border-top: 1px solid var(--macos-separator);
}

// 分割线样式
.section-divider {
  height: 1px;
  background: linear-gradient(90deg, 
    transparent 0%, 
    var(--macos-separator) 20%, 
    var(--macos-separator) 80%, 
    transparent 100%
  );
  margin: 20px 0;
  opacity: 0.6;
}

// 响应式设计
@media (max-width: 768px) {
  .modal-container {
    width: 95%;
    max-height: 95vh;
  }
  
  .options-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }
  
  .url-input-row {
    flex-direction: column;
    align-items: stretch;
  }
  
  .browser-controls {
    width: 100%;
    
    .browser-select {
      flex: 1;
      width: auto;
    }
  }
}
</style>