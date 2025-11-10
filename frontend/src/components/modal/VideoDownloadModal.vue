<template>
  <div
    v-if="showModal"
    class="macos-modal"
    role="dialog"
    aria-modal="true"
    :aria-label="mode === 'quick' ? $t('download.quick_task') : $t('download.new_task')"
  >
    <div class="modal-card card-frosted card-translucent" @keydown.esc.stop.prevent="closeModal" tabindex="-1" ref="dialogEl">
      <!-- Modal Header (macOS sheet style) -->
      <div class="modal-header">
        <ModalTrafficLights @close="closeModal" />
        <div class="title-area">
          <div class="title-text" v-if="!validUrlForTitle">{{ $t('download.new_task') }}</div>
          <!-- Shrunk chips after input -->
          <div class="title-chips" v-if="(validUrlForTitle || selectedBrowserLabel)">
            <div v-if="validUrlForTitle" class="chip-frosted chip-sm chip-translucent url-chip" :title="url">
              <Icon name="link" class="w-3 h-3"/>
              <span class="text">{{ truncatedUrl }}</span>
              <button class="chip-action" @click="resetToInput" :aria-label="$t('common.edit')">
              <Icon name="edit" class="w-3 h-3"/>
              </button>
            </div>
            <div v-if="selectedBrowserLabel" class="chip-frosted chip-sm chip-translucent browser-chip" :title="selectedBrowserLabel">
              <Icon name="compass" class="w-3 h-3"/>
              <span class="text">{{ selectedBrowserLabel }}</span>
              <button class="chip-action" @click="reselectBrowser" :aria-label="$t('common.edit')">
                <Icon name="refresh" class="w-3 h-3"/>
              </button>
            </div>
            <div v-if="modeChipVisible" class="chip-frosted chip-sm chip-translucent mode-chip" :title="modeLabel">
              <Icon :name="mode === 'custom' ? 'settings' : 'flashlight'" class="w-3 h-3"/>
              <span class="text">{{ modeLabel }}</span>
              <button class="chip-action" @click="viewStep = 'mode'" :aria-label="$t('common.edit')">
                <Icon name="arrow-left-right" class="w-3 h-3"/>
              </button>
            </div>
          </div>
        </div>
        <!-- Right close removed; use traffic light only -->
      </div>

      <!-- Modal Content -->
      <div class="modal-body">

        <!-- Dependencies Check -->
        <div v-if="!dependenciesReady" class="dependencies-warning">
          <div class="warning-content">
            <div class="warning-icon">
              <Icon name="status-warning" class="w-8 h-8 text-orange-500" />
            </div>
            <div class="warning-text">
              <h4>{{ $t('download.dependencies_not_ready') }}</h4>
              <p>{{ $t('download.dependencies_not_ready_desc') }}</p>
            </div>
          </div>
          <div class="warning-actions">
            <button class="btn-chip" @click="gotoDependency()">
              <Icon name="wrench" class="w-4 h-4 mr-2"></Icon>
              {{ $t('download.manage_dependencies') }}
            </button>
            <button class="btn-chip" @click="checkDependencies()">
              <Icon name="refresh" class="w-4 h-4 mr-2"></Icon>
              {{ $t('common.refresh') }}
            </button>
          </div>
        </div>

        <!-- Main Content -->
        <div v-else class="main-content">
          <!-- 内联错误显示（解析失败等），固定高度 + 滚动，支持复制与清除 -->
          <div v-if="hasParseError" class="error-panel">
            <div class="error-header">
              <div class="left">
                <Icon name="alert-triangle" class="w-4 h-4 mr-1" />
                <span class="title">{{ $t('download.parse_failed') }}</span>
              </div>
              <div class="right">
                <button class="btn-chip" @click="copyParseError">
                  <Icon name="file-copy" class="w-4 h-4 mr-2"/>
                  {{ $t('common.copy') }}
                </button>
                <button class="btn-chip" @click="clearParseError">
                  <Icon name="close" class="w-4 h-4 mr-2"/>
                  {{ $t('common.close') }}
                </button>
              </div>
            </div>
            <div class="error-body">
              <pre class="error-text">{{ parseErrorText }}</pre>
            </div>
          </div>
          <!-- Only URL input when initial -->
          <div v-if="viewStep === 'input' || viewStep === 'detecting'">
            <div class="url-hero-wrap hero">
              <UrlCookiesBar
                ref="urlBar"
                v-model:url="url"
                :variant="'hero'"
                :browser-options="browserOptions"
                :is-loading-browsers="isLoadingProviders"
                :trailing-visible="true"
                :show-select="false"
                :show-parse="true"
                :parse-text="$t('common.parse')"
                :parse-disabled="!isValidHttpUrl((url||'').trim())"
                :parsing="viewStep === 'detecting'"
                :placeholder="$t('download.video_url_placeholder')"
                @update:url="handleUrlUpdate"
                @parse="onParseClick"
              />
            </div>
          </div>

          <!-- Only browsers list when cookies stage -->
          <div v-else-if="viewStep === 'cookies'" >
            <BrowserChips v-model="browser" :options="chipOptions" :default-option="defaultBrowserOption" @picked="onPickBrowser" />
          </div>

          <!-- Only two mode buttons (native-like) when mode stage -->
          <div v-else-if="viewStep === 'mode'">
            <ModePicker :default-choice="'custom'" @pick="onPickMode" />
          </div>

          <!-- Parsing overlay -->
          <div v-if="showParsingOverlay" class="parsing-overlay" aria-busy="true">
            <div class="spinner"><Icon name="spinner" class="animate-spin w-6 h-6"/></div>
            <div class="tip">{{ $t('download.parsing') }} · {{ parsingClock }}</div>
            <div v-if="parsingOvertime" class="tip overtime">{{ $t('download.maybe_slow_or_issue') || '已超过1分钟，可能存在网络或依赖问题。' }}</div>
            <div v-if="parsingOvertime" class="actions">
              <button class="btn-chip btn-primary" @click="retryParse"><Icon name="refresh" class="w-4 h-4 mr-2" />{{ $t('download.retry') }}</button>
              <button class="btn-chip" @click="cancelParse"><Icon name="close" class="w-4 h-4 mr-2" />{{ $t('common.cancel') }}</button>
            </div>
          </div>
          <!-- Spacer so absolute overlay has enough intrinsic room to fit without forcing scrollbars -->
          <div v-if="showParsingOverlay" class="parsing-spacer"></div>

          <!-- Custom options after parsing -->
          <div v-if="viewStep === 'customOptions'">
            <div v-if="videoData?.title" class="video-info-card">
              <div class="video-preview">
                <div class="thumbnail-container">
                  <ProxiedImage v-if="videoData?.thumbnail" :src="videoData.thumbnail" :alt="$t('download.thumbnail')" class="thumbnail-image" error-icon="video" />
                  <div v-else class="thumbnail-placeholder">
                    <Icon name="video" class="w-8 h-8 text-tertiary"></Icon>
                  </div>
                </div>
                <div class="video-meta">
                  <h4 class="video-title">{{ videoData.title }}</h4>
                  <p class="video-details">{{ videoData.duration }} · {{ videoData.author }}</p>
                </div>
              </div>
            </div>
            <DownloadOptions
              strategy="custom"
              :qualities="formatQualities(videoData?.formats || [])"
              :subtitles="videoData?.subtitles || []"
              :available-transcode-formats="availableTranscodeFormats"
              :model-selected-quality="selectedQuality || null"
              :model-selected-subtitle-lang="selectedSubtitleLang || ''"
              :model-selected-subtitles="selectedSubtitles"
              :model-selected-transcode="selectedTranscodeFormat || 0"
              @update:selectedQuality="setSelectedQuality"
              @update:selectedSubtitleLang="setSelectedSubtitleLang"
              @update:selectedSubtitles="setSelectedSubtitles"
              @update:selectedTranscodeFormat="setSelectedTranscodeFormat"
            />
            <div class="options-actions">
              <button class="btn-chip" @click="closeModal">
                <Icon name="close" class="w-4 h-4 mr-2" />
                {{ $t('common.cancel') }}
              </button>
              <button class="btn-chip btn-primary" @click="start" :disabled="!canStart || startingQuick">
                <Icon v-if="!startingQuick" name="download" class="w-4 h-4 mr-2" />
                <Icon v-else name="spinner" class="w-4 h-4 mr-2 animate-spin" />
                {{ $t('common.start') }}
              </button>
            </div>
          </div>

          <!-- Quick options -->
          <div v-if="viewStep === 'quickOptions'">
            <DownloadOptions
              strategy="quick"
              :available-transcode-formats="availableTranscodeFormats"
              :model-video-preset="video || 'best'"
              :model-best-caption="bestCaption ?? false"
              :model-quick-transcode="quickSelectedTranscodeFormat || 0"
              @update:videoPreset="setVideoPreset"
              @update:bestCaption="setBestCaption"
              @update:quickSelectedTranscodeFormat="setQuickSelectedTranscodeFormat"
            />
            <div class="options-actions">
              <button class="btn-chip" @click="closeModal">
                <Icon name="close" class="w-4 h-4 mr-2" />
                {{ $t('common.cancel') }}
              </button>
              <button class="btn-chip btn-primary" @click="start" :disabled="!canStart">
                <Icon name="download" class="w-4 h-4 mr-2" />
                {{ $t('common.start') }}
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Footer removed: actions shown inline in options stage -->
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick } from 'vue'
import { GetContent, Download, QuickDownload, GetFormats } from 'wailsjs/go/api/DowntasksAPI'
import { DependenciesReady } from 'wailsjs/go/api/DependenciesAPI'
import { GetBrowserByDomain } from 'wailsjs/go/api/CookiesAPI'
import useNavStore from '@/stores/nav.js'
import useSettingsStore from '@/stores/settings'
import { useI18n } from 'vue-i18n'
import ProxiedImage from '@/components/common/ProxiedImage.vue'
import { useLoggerStore } from '@/stores/logger'
import UrlCookiesBar from '@/components/common/UrlCookiesBar.vue'
import DownloadOptions from '@/components/common/DownloadOptions.vue'
import BrowserChips from '@/components/common/BrowserChips.vue'
import ModePicker from '@/components/common/ModePicker.vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import { formatDuration as fmtDuration, formatFileSize as fmtSize } from '@/utils/format.js'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

// i18n
const { t } = useI18n()
const logger = useLoggerStore()

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
// 视图步骤：input -> cookies -> mode -> parsing -> customOptions | quickOptions
const viewStep = ref('input')

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

// Refs for a11y/focus management
const dialogEl = ref(null)
const urlBar = ref(null)
// single URL bar (hero)
const selectedBrowserLabel = ref('')
const defaultBrowserOption = computed(() => {
  const list = availableProviders.value || []
  return list.length > 0 ? list[0].id : ''
})

// Watch modal visibility
watch(() => showModal.value, async (newValue) => {
  if (newValue) {
    // Modal opened
    viewStep.value = 'input'
    await checkDependencies()
    // get formats
    if (dependenciesReady.value) {
      await getFormats()
    }
    // focus URL input for quicker entry
    await nextTick()
    try { urlBar.value?.focus?.() } catch {}
    // move focus to dialog to capture Esc
    try { dialogEl.value?.focus?.() } catch {}
    try { document?.body?.classList?.add('modal-open') } catch {}
  } else {
    // Modal closed, reset form
    resetForm()
    try { document?.body?.classList?.remove('modal-open') } catch {}
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
const availableTranscodeFormats = ref({ video: [], audio: [] })
const selectedTranscodeFormat = ref(0) // New state for transcoding
const quickSelectedTranscodeFormat = ref(0) // New state for transcoding
const browser = ref('')
const availableProviders = ref([])
const isLoadingProviders = ref(false)

// 解析错误（在当前 Modal 内联显示，避免覆盖全屏的对话框问题）
const parseErrorText = ref('')
const hasParseError = computed(() => !!(parseErrorText.value && String(parseErrorText.value).trim()))
const clearParseError = () => { parseErrorText.value = '' }
const copyParseError = async () => { if (hasParseError.value) await copyToClipboard(parseErrorText.value, t) }

// 解析阶段：计时与超时提示
const PARSE_TIMEOUT_MS = 60000
const parsingElapsed = ref(0)
const parsingTimer = ref(null)
const parsingRunId = ref(0) // 用于忽略过期请求
const parsingOvertime = computed(() => parsingElapsed.value >= PARSE_TIMEOUT_MS)
const parsingClock = computed(() => {
  const s = Math.floor(parsingElapsed.value / 1000)
  const mm = String(Math.floor(s / 60)).padStart(2, '0')
  const ss = String(s % 60).padStart(2, '0')
  return `${mm}:${ss}`
})
function startParsingTimer() {
  stopParsingTimer()
  parsingElapsed.value = 0
  parsingTimer.value = setInterval(() => { parsingElapsed.value += 1000 }, 1000)
}
function stopParsingTimer() {
  if (parsingTimer.value) { clearInterval(parsingTimer.value); parsingTimer.value = null }
}
function cancelParse() {
  // 通过递增 runId 来让在途请求结果失效
  parsingRunId.value++
  stopParsingTimer()
  viewStep.value = 'mode'
}
async function retryParse() {
  cancelParse()
  await handleParse()
}

// quick mode data
const video = ref('best')
const bestCaption = ref(false)
const startingQuick = ref(false)

const noneProviderLabel = computed(() => t('download.provider_none'))
const manualProviderBadge = computed(() => t('download.provider_badge_manual'))

const normaliseProviders = (raw) => {
  try {
    const parsed = typeof raw === 'string' ? JSON.parse(raw) : raw
    if (!Array.isArray(parsed)) return []

    const unique = new Map()
    parsed.forEach((item) => {
      if (typeof item === 'string') {
        const id = item.trim()
        if (!id) return
        unique.set(id, {
          id,
          label: id,
          source: 'yt-dlp',
          kind: 'browser'
        })
        return
      }

      if (!item || typeof item !== 'object') return
      const id = String(item.id || item.browser || '').trim()
      if (!id) return
      const rawSource = String(item.source || '').trim().toLowerCase()
      const rawKind = String(item.kind || '').trim().toLowerCase()
      const source = rawSource || (rawKind === 'manual' ? 'manual' : 'yt-dlp')
      const label = String(item.label || item.name || item.browser || id).trim() || id
      unique.set(id, {
        id,
        label,
        source,
        kind: rawKind || (source === 'manual' ? 'manual' : 'browser')
      })
    })

    return Array.from(unique.values())
  } catch (err) {
    console.error('Failed to parse cookie providers:', err)
    return []
  }
}

const providerOptions = computed(() => {
  const base = [{ value: '', label: noneProviderLabel.value, source: '', kind: 'none', badge: '' }]
  const providers = Array.isArray(availableProviders.value) ? availableProviders.value : []
  providers.forEach((provider) => {
    if (!provider || !provider.id) return
    const badge = provider.source === 'manual' ? manualProviderBadge.value : ''
    base.push({
      value: provider.id,
      label: provider.label || provider.id,
      source: provider.source || '',
      kind: provider.kind || (provider.source === 'manual' ? 'manual' : 'browser'),
      badge
    })
  })
  return base
})

const browserOptions = computed(() => providerOptions.value.map(({ value, label }) => ({ value, label })))
const chipOptions = computed(() => providerOptions.value)

// 获取可用 Cookie 提供者的方法（浏览器或手动集合）
const fetchCookieProviders = async (url) => {
  const targetUrl = (url || '').trim()
  if (!targetUrl) {
    availableProviders.value = []
    browser.value = ''
    selectedBrowserLabel.value = ''
    return
  }

  try {
    isLoadingProviders.value = true
    const response = await GetBrowserByDomain(targetUrl)
    if (response.success) {
      availableProviders.value = normaliseProviders(response.data)
      browser.value = ''
      selectedBrowserLabel.value = ''
    } else {
      availableProviders.value = []
      browser.value = ''
      selectedBrowserLabel.value = ''
    }
  } catch (error) {
    console.error('Failed to fetch cookie providers:', error)
    availableProviders.value = []
    browser.value = ''
    selectedBrowserLabel.value = ''
  } finally {
    isLoadingProviders.value = false
  }
}

watch(video, () => {
  quickSelectedTranscodeFormat.value = 0 // Reset transcode selection to "No Transcoding"
})

// Setter helpers to avoid template ref auto-unwrapping issues
const setSelectedQuality = (v) => { selectedQuality.value = v }
const setSelectedSubtitleLang = (v) => { selectedSubtitleLang.value = v }
const setSelectedSubtitles = (arr) => { selectedSubtitles.value = arr }
const setSelectedTranscodeFormat = (n) => { selectedTranscodeFormat.value = Number(n) || 0 }
const setQuickSelectedTranscodeFormat = (n) => { quickSelectedTranscodeFormat.value = Number(n) || 0 }
const setVideoPreset = (v) => { video.value = v }
const setBestCaption = (v) => { bestCaption.value = v }

// URL 输入处理（仅更新，不触发检测）
const handleUrlUpdate = (newUrl) => {
  clearTimeout(urlInputTimer.value)
  urlInputTimer.value = setTimeout(() => {
    if ((newUrl || '').trim()) {
      viewStep.value = 'input'
    } else {
      availableProviders.value = []
      browser.value = ''
      viewStep.value = 'input'
    }
    parseErrorText.value = ''
  }, 300)
}

// Basic http(s) URL validation
function isValidHttpUrl(u) {
  try {
    const parsed = new URL(u)
    return parsed.protocol === 'http:' || parsed.protocol === 'https:'
  } catch (_) { return false }
}

// 添加防抖timer
const urlInputTimer = ref(null)

// 移除立即监听，统一走防抖处理

// Whether can start download
const canDownload = computed(() => videoData.value && selectedQuality.value)
const quickModeDownEnabled = computed(() => url.value !== '' && video.value !== '' && bestCaption.value !== null)
const canStart = computed(() => {
  if (viewStep.value === 'customOptions') return !!canDownload.value
  if (viewStep.value === 'quickOptions') return !!quickModeDownEnabled.value
  return false
})

// 派生 UI 状态
const urlVariant = computed(() => (viewStep.value === 'input' ? 'hero' : 'compact'))
const showChips = computed(() => ['cookies', 'mode'].includes(viewStep.value))
const showPicker = computed(() => viewStep.value === 'mode')
const showParsingOverlay = computed(() => ['parsing','detecting'].includes(viewStep.value))
// Only show mode chip after user picked a mode (i.e., in options views)
const modeChipVisible = computed(() => ['customOptions','quickOptions'].includes(viewStep.value))
const modeLabel = computed(() => mode.value === 'custom' ? t('download.custom_mode') : t('download.quick_mode'))
const truncatedUrl = computed(() => {
  const u = (url.value || '').trim()
  if (!u) return ''
  return u.length > 42 ? u.slice(0, 19) + '…' + u.slice(-18) : u
})
// 标记是否是由 Parse/Paste 主动触发的检测流程
const detectInitiated = ref(false)
const validUrlForTitle = computed(() => isValidHttpUrl((url.value || '').trim()))

async function onParseClick() {
  const u = (url.value || '').trim()
  if (!isValidHttpUrl(u)) { try { $message?.error?.('URL 不合法') } catch {} ; return }
  clearParseError()
  detectInitiated.value = true
  viewStep.value = 'detecting'
  await fetchCookieProviders(u)
}

async function onPasteClick() {
  const prev = url.value || ''
  try {
    const text = await navigator.clipboard.readText()
    if (text && text.trim()) {
      url.value = text.trim()
      await onParseClick()
      return
    }
  } catch (e) {
    // ignore; fallback to user-mediated paste
  }
  try { urlBar.value?.focus?.() } catch {}
  // Fallback: wait for user OS paste confirmation, then auto-parse
  let elapsed = 0
  const step = 120
  const timer = setInterval(async () => {
    elapsed += step
    if ((url.value || '') !== prev && isValidHttpUrl((url.value || '').trim())) {
      clearInterval(timer)
      await onParseClick()
    } else if (elapsed >= 1800) {
      clearInterval(timer)
    }
  }, step)
}

// 可用的 Cookie 提供者就绪 → 切换到 cookies/模式 阶段
watch([url, availableProviders, isLoadingProviders], () => {
  if (!detectInitiated.value) return
  if ((url.value || '').trim() && !isLoadingProviders.value) {
    if ((availableProviders.value || []).length > 0) {
      viewStep.value = browser.value ? 'mode' : 'cookies'
      detectInitiated.value = false
    } else if ((availableProviders.value || []).length === 0) {
      viewStep.value = 'cookies'
      detectInitiated.value = false
    }
  }
})

// 选择 Cookie 提供者 → 进入模式选择
watch(browser, (b) => {
  if ((b || '').length) {
    const provider = (availableProviders.value || []).find(item => item && item.id === b)
    selectedBrowserLabel.value = provider?.label || b
    viewStep.value = 'mode'
  }
})

// Ensure default quality selected when entering custom options
watch([() => viewStep.value, () => videoData.value], () => {
  if (viewStep.value === 'customOptions') {
    const list = formatQualities(videoData.value?.formats || [])
    if (!selectedQuality.value) {
      selectedQuality.value = list.find(q => !q.isHeader) || null
    }
  }
})

// Format video quality options
const formatQualities = (formats) => {
  if (!formats?.length) return []

  // Group by format type
  const formatGroups = new Map()
  const audioFormats = []

  formats.forEach(format => {
    const hasVideo = format?.vcodec && format.vcodec !== 'none'
    const hasAudio = format?.acodec && format.acodec !== 'none'
    const hasSize = format?.filesize || format?.filesize_approx

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
const formatFileSize = (size) => fmtSize(size, t)

// 移至 DownloadOptions 组件内处理字幕选择变化

// Parse video URL
const normalizeTranscodeFormats = (obj) => {
  try {
    const o = obj && typeof obj === 'object' ? obj : {}
    return {
      video: Array.isArray(o.video) ? o.video : [],
      audio: Array.isArray(o.audio) ? o.audio : []
    }
  } catch { return { video: [], audio: [] } }
}

const getFormats = async () => {
  try {
    const response = await GetFormats()
    if (response.success) {
      const parsed = JSON.parse(response.data)
      availableTranscodeFormats.value = normalizeTranscodeFormats(parsed)
    } else {
      $message.warning(response.msg)
    }
  } catch (error) {
    $message.error(error.message)
  }
}

const handleParse = async () => {
  if (!url.value) return
  mode.value = 'custom'
  viewStep.value = 'parsing'
  clearParseError()
  startParsingTimer()
  const runId = ++parsingRunId.value

  // get video info
  try {
    isLoading.value = true
    const response = await GetContent(url.value, browser.value) // enable cookies
    // 若期间用户已取消/重试，忽略过期结果
    if (runId !== parsingRunId.value) return
    if (response.success) {
      const data = JSON.parse(response.data)
      // thumbnail with robust fallbacks for YouTube/shortlink cases
      let thumb = ''
      const rawThumb = (data && data.thumbnail) ? String(data.thumbnail) : ''
      if (rawThumb) { thumb = rawThumb }
      // fallback: pick first item from thumbnails array
      if (!thumb && Array.isArray(data?.thumbnails)) {
        const first = data.thumbnails.find(x => x && (x.url || x.URL))
        if (first) thumb = String(first.url || first.URL || '')
      }
      // fallback: construct from id for YouTube
      if (!thumb && /youtube/i.test(String(data?.extractor || '')) && data?.id) {
        thumb = `https://i.ytimg.com/vi/${data.id}/hqdefault.jpg`
      }
      if (thumb && thumb.startsWith('http:')) thumb = thumb.replace('http:', 'https:')

      videoData.value = {
        title: data.title,
        author: data.uploader || data.channel || data.extractor,
        duration: formatDuration(data.duration),
        thumbnail: thumb,
        formats: data.formats,
        subtitles: formatSubtitles(data.subtitles)
      }
      try { logger.info('Parse content: thumbnail chosen', { raw: rawThumb, final: thumb, extractor: data?.extractor, id: data?.id }) } catch {}

      // Default select the first non-header format
      const qualities = formatQualities(videoData.value.formats || [])
      selectedQuality.value = qualities.find(q => !q.isHeader) || null

      // Default do not select subtitles
      selectedSubtitles.value = []
      translateTo.value = ''
      subtitleStyle.value = 'default'
      selectedTranscodeFormat.value = 0 // Reset transcode selection
      viewStep.value = 'customOptions'
    } else {
      // 在当前 Modal 内联展示错误，避免对话框内容过长造成无法关闭的情况
      parseErrorText.value = String(response.msg || t('download.parse_failed'))
      viewStep.value = 'mode'
    }
  } catch (error) {
    // 同样在当前 Modal 内联展示异常信息
    if (runId === parsingRunId.value) {
      parseErrorText.value = String(error?.message || t('download.parse_failed'))
      viewStep.value = 'mode'
    }
  } finally {
    isLoading.value = false
    stopParsingTimer()
  }
}

// 格式化时长
const formatDuration = (seconds) => fmtDuration(seconds, t)

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
      // Default to "best" so backend and inspector have a consistent, non-empty format
      subFormat: "best",
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

// duplicate removed; defined earlier

// Start download
const startQuickDownload = async () => {
  if (!quickModeDownEnabled.value) return

  try {
    startingQuick.value = true
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
  } finally {
    startingQuick.value = false
  }
}

// Unified start entry
const start = async () => {
  if (viewStep.value === 'customOptions') {
    await startCustomDownload()
  } else if (viewStep.value === 'quickOptions') {
    await startQuickDownload()
  }
}

// 编辑与重选交互
const resetToInput = () => {
  url.value = ''
  browser.value = ''
  selectedBrowserLabel.value = ''
  availableProviders.value = []
  videoData.value = null
  selectedQuality.value = null
  selectedSubtitles.value = []
  selectedSubtitleLang.value = ''
  selectedTranscodeFormat.value = 0
  availableTranscodeFormats.value = { video: [], audio: [] }
  viewStep.value = 'input'
  parseErrorText.value = ''
}

const reselectBrowser = () => {
  browser.value = ''
  selectedBrowserLabel.value = ''
  viewStep.value = 'cookies'
  parseErrorText.value = ''
}

const onPickBrowser = (opt) => {
  if (opt && typeof opt === 'object') {
    selectedBrowserLabel.value = opt.label || opt.value || noneProviderLabel.value
  } else if (typeof opt === 'string') {
    selectedBrowserLabel.value = opt || noneProviderLabel.value
  } else {
    selectedBrowserLabel.value = noneProviderLabel.value
  }
  // show mode selection next; default highlight is custom (not auto-select)
  viewStep.value = 'mode'
}

// Handle mode pick interactions
const onPickMode = async (picked) => {
  if (picked === 'custom') {
    await handleParse()
  } else if (picked === 'quick') {
    mode.value = 'quick'
    viewStep.value = 'quickOptions'
  }
}

// Close modal
const closeModal = () => {
  showModal.value = false
}

// Reset form
const resetForm = () => {
  // core shared state
  url.value = ''
  browser.value = ''
  selectedBrowserLabel.value = ''
  availableProviders.value = []
  viewStep.value = 'input'
  parseErrorText.value = ''
  stopParsingTimer()
  parsingRunId.value++

  // custom-mode related
  videoData.value = null
  selectedQuality.value = null
  selectedSubtitles.value = []
  selectedSubtitleLang.value = ''
  translateTo.value = ''
  subtitleStyle.value = 'default'
  selectedTranscodeFormat.value = 0
  availableTranscodeFormats.value = { video: [], audio: [] }

  // quick-mode related
  video.value = 'best'
  bestCaption.value = false
  quickSelectedTranscodeFormat.value = 0

  // reset mode to initial (used only when actually chosen)
  mode.value = props.initialMode
}

</script>

<style lang="scss" scoped>
/* macOS Tahoe 风格：半透明磨砂 + 柔和阴影，与 AnalysisModal 保持一致 */
.macos-modal { position: fixed; inset: 0; background: rgba(0,0,0,0.2); backdrop-filter: blur(8px); display:flex; align-items:center; justify-content:center; z-index: 2000; padding: 16px; }
.modal-card { width: min(640px, 96vw); max-height: 82vh; display:flex; flex-direction: column; overflow: hidden; border-radius: 12px; border: 1px solid rgba(60,60,67,0.20); box-shadow: var(--macos-shadow-2, 0 12px 30px rgba(0,0,0,0.24)); }
.modal-card.card-frosted.card-translucent { background: color-mix(in oklab, var(--macos-surface) 88%, transparent); border-color: rgba(255,255,255,0.28); box-shadow: var(--macos-shadow-2, 0 12px 30px rgba(0,0,0,0.24)), 0 12px 30px rgba(0,0,0,0.24); }
.modal-header { height: 36px; display:flex; align-items:center; justify-content: space-between; padding: 10px 12px; border-bottom: 1px solid rgba(255,255,255,0.16); }
.modal-header .title-area { display:flex; align-items:center; gap: 10px; min-width: 0; }
.modal-header .title-text { font-size: var(--fs-base); font-weight: 600; color: var(--macos-text-primary); }
.traffic-lights { display:flex; align-items:center; gap:6px; margin-right: 6px; -webkit-app-region: no-drag; --wails-draggable: no-drag; }
.traffic-lights .light { width: 10px; height:10px; border-radius: 50%; display:inline-block; box-shadow: inset 0 0 0 1px rgba(0,0,0,0.12); }
.traffic-lights .red { background:#ff5f56; }
.traffic-lights .yellow { background:#d9d9d9; }
.traffic-lights .green { background:#d9d9d9; }
.traffic-lights .clickable { cursor: pointer; }
.traffic-lights .disabled { opacity: .6; cursor: default; }
.title-chips { display:flex; align-items:center; gap:6px; min-width:0; }
.title-chips .chip-frosted .text { max-width: 260px; overflow:hidden; text-overflow: ellipsis; white-space: nowrap; }
.title-chips .chip-frosted .chip-action { display:inline-flex; align-items:center; justify-content:center; width:16px; height:16px; border-radius: 4px; border: none; background: transparent; color: rgba(255,255,255,0.8); }
.title-chips .chip-frosted .chip-action:hover { background: var(--macos-blue); color: #fff; }
.modal-body { position: relative; flex: 1; overflow: visible; padding: 12px; }

/* URL hero shrink wrap */
.url-hero-wrap { transition: width .18s ease; }
.url-hero-wrap.hero { width: 100%; }
.url-hero-wrap.compact { width: 70%; max-width: 520px; }

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
    font-size: var(--fs-base);
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

/* 错误信息面板（参考 AnalysisModal 的样式，带滚动与复制按钮） */
.error-panel { border:1px solid rgba(255, 69, 58, 0.55); border-radius: 12px; background: color-mix(in oklab, var(--macos-surface) 88%, transparent); box-shadow: 0 6px 18px rgba(0,0,0,0.08); overflow: hidden; }
.error-header { display:flex; align-items:center; justify-content: space-between; padding: 10px 12px; background: linear-gradient(180deg, rgba(255, 69, 58, 0.08), transparent); border-bottom: 1px solid rgba(255,255,255,0.12); }
.error-header .left { display:flex; align-items:center; color: var(--macos-danger-text, #ff6b6b); font-weight: 600; font-size: 12px; }
.error-header .title { color: var(--macos-text-primary); margin-left: 2px; }
.error-header .right { display:flex; align-items:center; gap: 8px; }
.error-body { position: relative; max-height: 32vh; overflow: auto; }
.error-text { margin: 0; padding: 10px 12px; white-space: pre-wrap; word-break: break-word; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace; font-size: 12px; color: var(--macos-text-secondary); line-height: 1.4; }

/* Parsing overlay */
.parsing-overlay { position: absolute; inset: 0; background: color-mix(in oklab, var(--macos-surface) 86%, transparent); backdrop-filter: saturate(180%) blur(10px); -webkit-backdrop-filter: saturate(180%) blur(10px); display:flex; flex-direction:column; align-items:center; justify-content:center; gap:8px; z-index: 10; border-radius: 12px; border: 1px solid rgba(60,60,67,0.14); }
.parsing-overlay .tip { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.parsing-overlay .tip.overtime { color: var(--macos-danger-text, #ff6b6b); }
.parsing-overlay .actions { display:flex; align-items:center; gap:8px; margin-top: 6px; }
.parsing-spacer { height: 140px; }

/* Flip-in animation for mode picker */
.flip-in-enter-active, .flip-in-leave-active { transition: transform .22s ease, opacity .22s ease; transform-style: preserve-3d; }
.flip-in-enter-from { transform: rotateX(-8deg); opacity: 0; }
.flip-in-enter-to { transform: rotateX(0deg); opacity: 1; }
.flip-in-leave-from { transform: rotateX(0deg); opacity: 1; }
.flip-in-leave-to { transform: rotateX(8deg); opacity: 0; }

/* Chips appear animation */
.chips-in-enter-active, .chips-in-leave-active { transition: all .18s ease; }
.chips-in-enter-from, .chips-in-leave-to { opacity: 0; transform: translateY(-4px); }

// Input Section
.input-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-label {
  font-size: var(--fs-base);
  font-weight: 500;
  color: var(--macos-text-primary);
}

/* URL 输入行（macOS 搜索框 + 右侧附属控件） */
.url-row { display:flex; align-items:center; gap: 8px; }
.search-field { position: relative; flex: 1 1 auto; min-width: 0; }
.search-field .input-macos { width: 100%; padding-left: 28px; height: 28px; }
.search-field .icon { position: absolute; left: 8px; top: 50%; transform: translateY(-50%); width: 14px; height: 14px; color: var(--macos-text-tertiary); }
.trailing-controls { display:inline-flex; align-items:center; gap: 8px; }
.trailing-controls .select-macos { width: 140px; }

// 加载状态
.loading-indicator-below {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 8px;
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
}

.loading-text {
  font-size: var(--fs-sub);
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
  font-size: var(--fs-title);
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0;
  line-height: 1.3;
}

.video-details {
  font-size: var(--fs-sub);
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
  font-size: var(--fs-sub);
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

// cancel/start button
.options-actions { display:flex; align-items:center; justify-content:center; gap: 16px; padding-top: 22px; width: 100%; }

// Footer
.modal-footer { display:flex; justify-content: flex-end; gap: 12px; padding: 8px 10px; border-top: 1px solid var(--macos-divider-weak); background: var(--macos-background-secondary); }

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
  .modal-card { width: 95%; max-height: 95vh; }
  
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
