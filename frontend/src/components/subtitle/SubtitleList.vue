<template>
  <div class="subtitle-list">
    <!-- 语言标签页 -->
    <div class="language-tabs-container">
      <div class="language-tabs">
        <div class="tabs-left">
          <button v-for="(language, index) in availableLanguages" :key="language.language_name" class="language-tab"
            :class="{ 'active': currentLanguage === language.language_name }"
            @click="selectLanguage(language.language_name)">
            <div class="tab-content">
              <div class="language-indicator" :style="{ backgroundColor: getLanguageColor(index) }"></div>
              <span class="language-name">{{ language.language_name }}</span>
              <span class="subtitle-count">({{ getSubtitleCount(language.language_name) }})</span>
              <button
                v-if="availableLanguages.length > 1"
                @click.stop="removeLanguage(language.language_code)"
                class="tab-close btn-chip-icon"
                :data-tooltip="$t('common.delete')"
                data-tip-pos="top"
                :aria-label="$t('common.delete')"
              >
                <Icon name="close" class="w-3.5 h-3.5" />
              </button>
            </div>
          </button>
        </div>

        <!-- 指标说明按钮 - 使用图标 -->
        <div class="tabs-right">
          <!-- 添加语言按钮：复用全局 btn-chip-icon + Icon + data-tooltip -->
          <button
            class="btn-chip-icon"
            :data-tooltip="$t('subtitle.add_language.title')"
            data-tip-pos="top"
            :aria-label="$t('subtitle.add_language.title')"
            @click="$emit('add-language')">
            <Icon name="plus" class="w-4 h-4" />
          </button>

          <!-- 指标面板切换按钮：复用全局 btn-chip-icon -->
          <button
            class="btn-chip-icon"
            :class="{ active: isMetricsPanelExpanded }"
            :data-tooltip="$t('subtitle.list.metrics_explanation')"
            data-tip-pos="top"
            :aria-label="$t('subtitle.list.metrics_explanation')"
            @click="toggleMetricsPanel">
            <Icon name="info" class="w-4 h-4" />
          </button>
        </div>
      </div>

      <!-- 指标说明面板 -->
      <div v-show="isMetricsPanelExpanded" class="metrics-panel">
        <!-- 当前标准信息卡片 -->
        <!-- 超简洁单行版本 -->
        <div class="standard-info-compact" v-if="currentLanguageStandard">
          <div class="standard-indicator">
            <svg class="indicator-icon" fill="currentColor" viewBox="0 0 24 24">
              <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
          </div>
          <span class="current-standard">{{ $t('subtitle.list.current_standard') }}: <strong>{{ standardDisplayName
          }}</strong></span>
          <span class="standard-desc">{{ standardDescription }}</span>
        </div>

        <!-- 指标说明区域 -->
        <div class="metrics-section">
          <h4 class="section-title">
            <svg class="w-4 h-4 text-primary/60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
            </svg>
            {{ $t('subtitle.list.metrics_explanation') }}
          </h4>

          <div class="metrics-grid">
            <!-- CPS 指标卡片 -->
            <div class="metric-card cps">
              <div class="metric-header">
                <div class="metric-icon-wrapper cps">
                  <svg class="metric-icon" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.94-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z" />
                  </svg>
                </div>
                <div class="metric-title">
                  <h5>{{ $t('subtitle.list.cps_fullname') }}</h5>
                  <span class="metric-label">{{ $t('subtitle.list.cps') }}</span>
                </div>
              </div>
              <div class="metric-description">
                <p>{{ $t('subtitle.list.cps_desc') }}</p>
                <div class="threshold-list" v-if="currentLanguageStandard">
                  <div class="threshold-item normal">
                    <span class="threshold-label">{{ $t('subtitle.list.level_normal') }}</span>
                    <span class="threshold-value">≤ {{ cpsThresholds.normal }} CPS</span>
                  </div>
                  <div class="threshold-item warning">
                    <span class="threshold-label">{{ $t('subtitle.list.level_warning') }}</span>
                    <span class="threshold-value">{{ cpsThresholds.normal + 1 }}-{{ cpsThresholds.warning }} CPS</span>
                  </div>
                  <div class="threshold-item danger">
                    <span class="threshold-label">{{ $t('subtitle.list.level_danger') }}</span>
                    <span class="threshold-value">> {{ cpsThresholds.warning }} CPS</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- WPM 指标卡片 -->
            <div class="metric-card wpm">
              <div class="metric-header">
                <div class="metric-icon-wrapper wpm">
                  <svg class="metric-icon" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M3 17.25V21h3.75L17.81 9.94l-3.75-3.75L3 17.25zM20.71 7.04c.39-.39.39-1.02 0-1.41l-2.34-2.34c-.39-.39-1.02-.39-1.41 0l-1.83 1.83 3.75 3.75 1.83-1.83z" />
                  </svg>
                </div>
                <div class="metric-title">
                  <h5>{{ $t('subtitle.list.wpm_fullname') }}</h5>
                  <span class="metric-label">{{ $t('subtitle.list.wpm') }}</span>
                </div>
              </div>
              <div class="metric-description">
                <p>{{ $t('subtitle.list.wpm_desc') }}</p>
                <div class="threshold-list" v-if="currentLanguageStandard">
                  <div class="threshold-item normal">
                    <span class="threshold-label">{{ $t('subtitle.list.level_normal') }}</span>
                    <span class="threshold-value">≤ {{ wpmThresholds.normal }} WPM</span>
                  </div>
                  <div class="threshold-item warning">
                    <span class="threshold-label">{{ $t('subtitle.list.level_warning') }}</span>
                    <span class="threshold-value">{{ wpmThresholds.normal + 1 }}-{{ wpmThresholds.warning }} WPM</span>
                  </div>
                  <div class="threshold-item danger">
                    <span class="threshold-label">{{ $t('subtitle.list.level_danger') }}</span>
                    <span class="threshold-value">> {{ wpmThresholds.warning }} WPM</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- CPL 指标卡片 -->
            <div class="metric-card cpl">
              <div class="metric-header">
                <div class="metric-icon-wrapper cpl">
                  <svg class="metric-icon" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      d="M14 2H6c-1.1 0-1.99.9-1.99 2L4 20c0 1.1.89 2 2 2h8c1.1 0 2-.9 2-2V8l-6-6zm2 16H8v-2h8v2zm0-4H8v-2h8v2zm-3-5V3.5L18.5 9H13z" />
                  </svg>
                </div>
                <div class="metric-title">
                  <h5>{{ $t('subtitle.list.cpl_fullname') }}</h5>
                  <span class="metric-label">{{ $t('subtitle.list.cpl') }}</span>
                </div>
              </div>
              <div class="metric-description">
                <p>{{ $t('subtitle.list.cpl_desc') }}</p>
                <div class="threshold-list" v-if="currentLanguageStandard">
                  <div class="threshold-item normal">
                    <span class="threshold-label">{{ $t('subtitle.list.level_normal') }}</span>
                    <span class="threshold-value">≤ {{ cplThresholds.normal }} CPL</span>
                  </div>
                  <div class="threshold-item warning">
                    <span class="threshold-label">{{ $t('subtitle.list.level_warning') }}</span>
                    <span class="threshold-value">{{ cplThresholds.normal + 1 }}-{{ cplThresholds.warning }} CPL</span>
                  </div>
                  <div class="threshold-item danger">
                    <span class="threshold-label">{{ $t('subtitle.list.level_danger') }}</span>
                    <span class="threshold-value">> {{ cplThresholds.warning }} CPL</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content">
      <div class="subtitle-items">
        <div v-for="(subtitle, index) in subtitles" :key="subtitle.id || index" class="subtitle-item" :class="{ 'is-editing': isEditingText(subtitle.id) }">
          <div class="item-header">
            <div class="item-controls">
              <span class="item-number">{{ index + 1 }}</span>
              <div class="time-info">
                <div class="time-range">
                  <div class="time-display">
                    {{ formatDate(subtitle.start_time) }} → {{ formatDate(subtitle.end_time) }}
                  </div>
                  <div class="metrics-display">
                    <!-- unified metric group: dots collapsed; expand as capsule on hover (non-narrow) -->
                    <div class="metric-group" :class="{ 'has-guidelines': !!getSubtitleGuideline(subtitle) }">
                      <div class="dots">
                        <!-- duration dot (always white) -->
                        <span class="dot level-neutral has-tooltip" :data-tooltip="$t('subtitle.list.duration') + ': ' + formatSimpleDuration(subtitle.start_time, subtitle.end_time)"></span>
                        <!-- guideline dots -->
                        <template v-if="getSubtitleGuideline(subtitle)">
                          <span class="dot" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).cps.level)"></span>
                          <span class="dot" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).wpm.level)"></span>
                          <span class="dot" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).cpl.level)"></span>
                        </template>
                      </div>
                      <div class="details" v-if="getSubtitleGuideline(subtitle)">
                        <div class="capsule">
                          <span class="item duration">
                            <span class="v">{{ formatSimpleDuration(subtitle.start_time, subtitle.end_time) }}</span>
                          </span>
                          <span class="sep">•</span>
                          <span class="item" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).cps.level)"><span class="k">CPS</span><span class="v">{{ getSubtitleGuideline(subtitle).cps.current }}</span></span>
                          <span class="sep">•</span>
                          <span class="item" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).wpm.level)"><span class="k">WPM</span><span class="v">{{ getSubtitleGuideline(subtitle).wpm.current }}</span></span>
                          <span class="sep">•</span>
                          <span class="item" :class="getGuidelineLevelClass(getSubtitleGuideline(subtitle).cpl.level)"><span class="k">CPL</span><span class="v">{{ getSubtitleGuideline(subtitle).cpl.current }}</span></span>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div class="item-content">
            <div v-if="!isEditingText(subtitle.id)" @click="startEditText(subtitle, index)" class="text-content">
              <span v-if="getSubtitleText(subtitle)" class="subtitle-text">
                {{ getSubtitleText(subtitle) }}
              </span>
            </div>
            <div v-else class="text-editing">
              <textarea v-model="editingTextValue" class="text-input"
                @keydown.enter.ctrl="saveSubtitleSegment(subtitle, index)" @keydown.esc="cancelEditText"
                @blur="saveSubtitleSegment(subtitle, index)" :ref="`textInput-${subtitle.id}`"></textarea>
            </div>
          </div>
        </div>

        <div v-if="!subtitles || subtitles.length === 0" class="empty-state">
          <p>{{ $t('subtitle.list.no_caption_rn') }}</p>
        </div>
      </div>
    </div>

  </div>
</template>

<script>
import { subtitleService } from '@/services/subtitleService.js'
import { useI18n } from 'vue-i18n'

export default {
  name: 'SubtitleList',
  props: {
    subtitles: {
      type: Array,
      default: () => []
    },
    currentLanguage: {
      type: String,
      default: 'English'
    },
    availableLanguages: {
      type: Object,
      default: () => []
    },
    subtitleCounts: {
      type: Object,
      default: () => ({})
    }
  },
  emits: [
    'update:currentLanguage', 'add-language', 'remove-language',
    'update:projectData'
  ],
  setup() {
    const { t } = useI18n()
    return { t }
  },
  data() {
    return {
      editingTextId: null,
      editingTextValue: '',
      editingSubtitleIndex: -1,
      isMetricsPanelExpanded: false
    }
  },
  computed: {
    currentLanguageStandard() {
      if (!this.subtitles?.length || !this.currentLanguage) return null

      const firstSubtitle = this.subtitles[0]
      return firstSubtitle?.guideline_standard?.[this.currentLanguage] || 'netflix'
    },

    cpsThresholds() {
      if (!this.currentLanguageStandard) return { normal: 20, warning: 25 }

      const thresholds = {
        'netflix': { normal: 20, warning: 25 },
        'bbc': { normal: 15, warning: 20 },
        'ade': { normal: 18, warning: 23 }
      }
      return thresholds[this.currentLanguageStandard] || thresholds['netflix']
    },

    wpmThresholds() {
      if (!this.currentLanguageStandard) return { normal: 160, warning: 200 }

      const thresholds = {
        'netflix': { normal: 160, warning: 200 },
        'bbc': { normal: 180, warning: 220 },
        'ade': { normal: 170, warning: 210 }
      }
      return thresholds[this.currentLanguageStandard] || thresholds['netflix']
    },

    cplThresholds() {
      if (!this.currentLanguageStandard) return { normal: 42, warning: 50 }

      const thresholds = {
        'netflix': { normal: 42, warning: 50 },
        'bbc': { normal: 37, warning: 45 },
        'ade': { normal: 40, warning: 48 }
      }
      return thresholds[this.currentLanguageStandard] || thresholds['netflix']
    },

    standardDisplayName() {
      if (!this.currentLanguageStandard) return 'Netflix'

      const standardNames = {
        'netflix': 'Netflix',
        'bbc': 'BBC',
        'ade': 'ADE'
      }
      return standardNames[this.currentLanguageStandard] || 'Netflix'
    },

    standardDescription() {
      if (!this.currentLanguageStandard) return ''

      const descriptions = {
        'netflix': this.t('subtitle.list.netflix_standard_desc'),
        'bbc': this.t('subtitle.list.bbc_standard_desc'),
        'ade': this.t('subtitle.list.ade_standard_desc'),
      }

      return descriptions[this.currentLanguageStandard] || ''
    }
  },
  methods: {
    selectLanguage(languageCode) {
      this.$emit('update:currentLanguage', languageCode)
    },

    removeLanguage(languageCode) {
      this.$emit('remove-language', languageCode)
    },

    getLanguageColor(index) {
      const colors = [
        '#3B82F6', '#EF4444', '#10B981', '#F59E0B',
        '#8B5CF6', '#EC4899', '#06B6D4', '#84CC16'
      ]
      return colors[index % colors.length]
    },

    getSubtitleCount(languageCode) {
      return this.subtitleCounts[languageCode] || 0
    },

    isEditingText(subtitleId) {
      return this.editingTextId === subtitleId
    },

    startEditText(subtitle, index) {
      this.editingTextId = subtitle.id
      this.editingTextValue = this.getSubtitleText(subtitle)
      this.editingSubtitleIndex = index
      this.$nextTick(() => {
        const textInput = this.$refs[`textInput-${subtitle.id}`]?.[0]
        if (textInput) {
          textInput.focus()
          textInput.select()
        }
      })
    },

    async saveSubtitleSegment(subtitle, index) {
      if (this.editingTextValue.trim() !== this.getSubtitleText(subtitle).trim()) {
        try {
          const updatedSegment = {
            ...subtitle,
            languages: {
              ...subtitle.languages,
              [this.currentLanguage]: {
                ...subtitle.languages[this.currentLanguage],
                text: this.editingTextValue.trim()
              }
            }
          }

          const result = await subtitleService.saveSubtitleSegment(subtitle.id, updatedSegment)
          if (result.success) {
            const projectData = JSON.parse(result.data)
            this.$emit('update:projectData', projectData)
          } else {
            throw new Error(result.msg)
          }
        } catch (error) {
          $message.error(error.message)
        }
      }
      this.cancelEditText()
    },

    cancelEditText() {
      this.editingTextId = null
      this.editingTextValue = ''
      this.editingSubtitleIndex = -1
    },

    formatDate(timecode) {
      if (!timecode || !timecode.time) return 'N/A'
      const totalMs = Math.floor(timecode.time / 1000000)
      const hours = Math.floor(totalMs / 3600000)
      const minutes = Math.floor((totalMs % 3600000) / 60000)
      const seconds = Math.floor((totalMs % 60000) / 1000)
      const milliseconds = totalMs % 1000
      return `${hours.toString().padStart(2, '0')}:${minutes.toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}.${milliseconds.toString().padStart(3, '0')}`
    },

    formatSimpleDuration(startTimecode, endTimecode) {
      if (!startTimecode || !endTimecode || !startTimecode.time || !endTimecode.time) return 'N/A'
      const durationNs = endTimecode.time - startTimecode.time
      const durationSeconds = durationNs / 1000000000

      if (durationSeconds % 1 === 0) {
        return `${durationSeconds}s`
      }
      return `${parseFloat(durationSeconds.toFixed(3))}s`
    },


    getSubtitleText(subtitle) {
      if (!subtitle.languages || !this.currentLanguage) return ''
      return subtitle.languages[this.currentLanguage]?.text || ''
    },

    getSubtitleGuideline(subtitle) {
      if (!subtitle.languages || !this.currentLanguage) return null
      return subtitle.languages[this.currentLanguage]?.subtitle_guideline || null
    },

    getGuidelineLevelClass(level) {
      switch (level) {
        case 0: return 'level-normal'
        case 1: return 'level-warning'
        case 2: return 'level-danger'
        default: return 'level-normal'
      }
    },

    toggleMetricsPanel() {
      this.isMetricsPanelExpanded = !this.isMetricsPanelExpanded
    }
  }
}
</script>

<style scoped>
.subtitle-list {
  background: var(--macos-background);
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  height: 100%;
  /* 关键：确保不会超出父容器 */
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
}

.language-tabs-container {
  display: flex;
  align-items: center;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
  padding: 12px 20px;
  min-height: 48px;
  flex-shrink: 0;
  position: relative;
  width: 100%;
  max-width: 100%;
  box-sizing: border-box;
  /* 关键：确保右侧按钮始终在右侧 */
  justify-content: space-between;
}

.language-tabs {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0;
  background: transparent;
  border: none;
  border-radius: 0;
  /* 关键：让左侧区域占据剩余空间但不超出 */
  flex: 1;
  min-width: 0;
  overflow: hidden;
  box-sizing: border-box;
}

.tabs-left {
  display: flex;
  align-items: center;
  gap: 8px;
  max-width: calc(100vw - 48px - 48px - 320px - 48px - 60px); /* 减去右侧按钮区域宽度 */
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  -ms-overflow-style: none;
  scroll-behavior: smooth;
  box-sizing: border-box;
}

/* 隐藏 webkit 滚动条 */
.tabs-left::-webkit-scrollbar {
  display: none;
}

.tabs-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  min-width: 80px;
  justify-content: flex-end;
  padding-left: 12px;
  border-left: 1px solid var(--macos-separator);
  margin-left: 12px;
}

/* 严格控制语言标签的宽度 */
.language-tab {
  display: flex;
  align-items: center;
  padding: 6px 12px;
  background: var(--macos-background);
  border: 1px solid var(--macos-border);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: var(--fs-base);
  font-weight: 500;
  color: var(--macos-text-secondary);
  flex-shrink: 0;
  white-space: nowrap;
  height: 32px;
  min-width: 80px;
  max-width: 120px;
  overflow: hidden;
  box-sizing: border-box;
  margin-right: 8px;
}

.language-tab:hover {
  background: var(--macos-background-secondary);
  border-color: var(--macos-gray-hover);
  transform: translateY(-1px);
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
}

.language-tab.active {
  background: var(--macos-blue);
  border-color: var(--macos-blue);
  color: var(--macos-background);
  box-shadow: 0 2px 6px rgba(0, 123, 255, 0.25);
}

.language-tab.active:hover {
  background: var(--macos-blue-hover);
  border-color: var(--macos-blue-hover);
  transform: translateY(-1px);
}

/* 顶部右侧按钮改为复用 .btn-chip-icon（全局样式控制 hover/active/tooltip） */


.tab-content {
  display: flex;
  align-items: center;
  gap: 4px; /* 减小间距 */
  width: 100%;
  overflow: hidden;
  min-width: 0;
  box-sizing: border-box;
}

.language-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.language-name {
  flex: 1;
  text-align: left;
  font-weight: 500;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
  max-width: 70px;
}

.subtitle-count {
  font-size: var(--fs-caption);
  opacity: 0.75;
  flex-shrink: 0;
  font-weight: 400;
  /* 确保数字不会被截断 */
  min-width: fit-content;
}

.language-tab.active .subtitle-count {
  opacity: 0.9;
}

.tab-close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  border: none; /* 覆盖 btn-chip-icon 的边框，使其更贴合标签 */
  background: transparent; /* 贴合标签背景 */
  border-radius: 50%;
  color: inherit;
  opacity: 0; /* 悬停标签时才出现 */
  transition: all 0.18s ease;
  flex-shrink: 0;
  margin-left: 2px;
  box-shadow: none;
}

.language-tab:hover .tab-close {
  opacity: 0.7;
}

.tab-close:hover { background: rgba(255, 255, 255, 0.20); opacity: 1; transform: scale(1.05); }

.language-tab.active .tab-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

.content {
  flex: 1;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
}

.subtitle-items {
  flex: 1;
  overflow-y: auto;
}

.subtitle-item {
  padding: 16px 20px;
  border-bottom: 1px solid var(--macos-separator);
  transition: background-color 0.2s;
}

.subtitle-item:hover {
  background: var(--macos-background-secondary);
}

.item-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.item-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.item-number {
  font-size: var(--fs-sub);
  color: var(--macos-text-tertiary);
  font-weight: 500;
  min-width: 24px;
}

.time-info {
  flex: 1;
  margin: 0 16px;
}

.time-range {
  font-size: var(--fs-base);
  color: var(--macos-text-primary);
  font-family: var(--font-mono);
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
}

.time-display {
  display: flex;
  align-items: center;
  gap: 8px;
}

.metrics-display {
  display: flex;
  align-items: center;
  gap: 8px;
}

.duration-badge {
  display: inline-flex;
  align-items: center;
  padding: 4px 8px;
  background: var(--macos-background-secondary);
  color: var(--macos-text-secondary);
  border-radius: 4px;
  font-size: var(--fs-sub);
  font-weight: 500;
  border: 1px solid var(--macos-border);
  cursor: help;
}

.subtitle-metrics {
  display: flex;
  gap: 6px;
  align-items: center;
}

.metric-item {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 6px;
  border-radius: 8px;
  font-size: var(--fs-micro);
  font-weight: 600;
  border: 1px solid;
  min-width: fit-content;
}

.metric-label { text-transform: uppercase; letter-spacing: 0.5px; }

.metric-value {
  font-weight: 700;
  font-size: var(--fs-micro);
}

.metric-item.level-normal {
  background: var(--macos-success-bg);
  border-color: var(--macos-success-text);
  color: var(--macos-success-text);
}

.metric-item.level-warning {
  background: var(--macos-warning-bg);
  border-color: var(--macos-warning-text);
  color: var(--macos-warning-text);
}

.metric-item.level-danger {
  background: var(--macos-danger-bg);
  border-color: var(--macos-danger-text);
  color: var(--macos-danger-text);
}

.item-content {
  margin-left: 32px;
}

.text-content {
  min-height: 2.5rem;
  display: flex;
  align-items: center;
  border: 1px solid transparent;
  padding: 8px 12px;
  border-radius: 4px;
  background: var(--macos-background-secondary);
  transition: all 0.2s;
  cursor: pointer;
}

.text-content:hover {
  border-color: var(--macos-border);
}

.subtitle-text {
  color: var(--macos-text-primary);
  line-height: 1.4;
}

.placeholder-text {
  color: var(--macos-text-tertiary);
  font-style: italic;
}

.text-input {
  width: 100%;
  font-family: inherit;
  font-size: inherit;
  line-height: 1.4;
  border: 1px solid var(--macos-blue);
  border-radius: 4px;
  padding: 8px 12px;
  font-size: var(--fs-title);
  resize: vertical;
  min-height: 40px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
}

.text-input:focus {
  outline: none;
  box-shadow: 0 0 0 2px var(--macos-blue-hover);
}

.empty-state {
  text-align: center;
  padding: 40px 20px;
  color: var(--macos-text-tertiary);
}

.empty-state p {
  margin: 0;
  font-size: var(--fs-title);
}

.content::-webkit-scrollbar,
.subtitle-items::-webkit-scrollbar {
  width: 6px;
}

.content::-webkit-scrollbar-track,
.subtitle-items::-webkit-scrollbar-track {
  background: var(--macos-background-secondary);
  border-radius: 3px;
}

.content::-webkit-scrollbar-thumb,
.subtitle-items::-webkit-scrollbar-thumb {
  background: var(--macos-scrollbar-thumb);
  border-radius: 3px;
}

.content::-webkit-scrollbar-thumb:hover,
.subtitle-items::-webkit-scrollbar-thumb:hover {
  background: var(--macos-scrollbar-thumb-hover);
}

/* 添加滚动提示样式 */
.language-tabs-container::before,
.language-tabs-container::after {
  content: '';
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 20px;
  height: 20px;
  background: var(--macos-background-secondary);
  pointer-events: none;
  z-index: 1;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.language-tabs-container::before {
  left: 20px;
  background: linear-gradient(to right, var(--macos-background-secondary), transparent);
}

.language-tabs-container::after {
  right: 60px; /* 为右侧按钮留出空间 */
  background: linear-gradient(to left, var(--macos-background-secondary), transparent);
}

/* 当可以滚动时显示渐变提示 */
.language-tabs-container.can-scroll-left::before {
  opacity: 1;
}

.language-tabs-container.can-scroll-right::after {
  opacity: 1;
}

/* 添加滚动按钮（可选） */
.scroll-button {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 24px;
  height: 24px;
  border: none;
  background: var(--macos-background);
  border-radius: 50%;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  opacity: 0;
  transition: all 0.3s ease;
  z-index: 2;
}

.scroll-button:hover {
  background: var(--macos-background-secondary);
  transform: translateY(-50%) scale(1.1);
}

.scroll-button.left {
  left: 24px;
}

.scroll-button.right {
  right: 64px;
}

.language-tabs-container:hover .scroll-button {
  opacity: 1;
}

/* 响应式调整 */
@media (max-width: 1200px) {
  .tabs-left {
    max-width: calc(100vw - 48px - 24px - 280px - 48px - 40px - 70px);
  }
  
  .tabs-right {
    width: 70px;
    min-width: 70px;
    gap: 6px;
    padding-left: 8px;
    margin-left: 8px;
  }
  
  .language-tab {
    max-width: 100px;
    min-width: 70px;
  }
  
  .language-name {
    max-width: 50px;
  }
}

@media (max-width: 900px) {
  .tabs-left {
    max-width: calc(100vw - 48px - 16px - 240px - 48px - 30px - 60px);
  }
  
  .tabs-right {
    width: 60px;
    min-width: 60px;
  }
  
  .language-tab {
    max-width: 80px;
    min-width: 60px;
  }
  
  .language-name {
    max-width: 40px;
  }
}

@media (max-width: 768px) {
  .language-tabs-container {
    padding: 8px 16px;
  }
  
  .language-tab {
    padding: 4px 8px;
    font-size: var(--fs-sub);
    max-width: 60px;
    min-width: 50px;
    height: 28px;
  }
  
  .language-name {
    max-width: 30px;
  }

  .tabs-right {
    gap: 6px;
    min-width: 70px;
    padding-left: 8px;
    margin-left: 8px;
  }
  
  .tabs-left {
    mask-image: linear-gradient(to right, 
      transparent 0px, 
      black 10px, 
      black calc(100% - 10px), 
      transparent 100%);
    -webkit-mask-image: linear-gradient(to right, 
      transparent 0px, 
      black 10px, 
      black calc(100% - 10px), 
      transparent 100%);
  }

  .subtitle-item {
    padding: 12px 16px;
  }

  .time-range {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
  }

  .metrics-display {
    align-self: flex-end;
  }
}



/* 指标说明面板样式 */
.metrics-panel {
  position: absolute;
  top: 100%;
  left: 0;
  right: 0;
  background: var(--macos-background);
  border: 1px solid var(--macos-separator);
  border-top: none;
  border-radius: 0 0 10px 10px;
  box-shadow: 0 8px 25px rgba(0, 0, 0, 0.15);
  z-index: 10;
  max-height: 500px;
  overflow-y: auto;
  animation: slideInUp 0.2s ease-out;
  padding: 20px;
}

@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(-10px) scale(0.98);
  }

  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

/* 当前标准信息卡片 */
.standard-info-card {
  background: var(--macos-background-secondary);
  border: 1px solid var(--macos-separator);
  border-radius: 10px;
  margin-bottom: 20px;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--macos-background-tertiary);
  border-bottom: 1px solid var(--macos-separator);
}

.standard-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-title {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--macos-text-primary);
}

.standard-content {
  padding: 16px;
}

.standard-badge-large {
  display: inline-flex;
  align-items: center;
  padding: 8px 16px;
  background: linear-gradient(135deg, var(--macos-blue) 0%, rgba(var(--macos-blue-rgb), 0.8) 100%);
  color: white;
  border-radius: 20px;
  font-size: var(--fs-base);
  font-weight: 600;
  margin-bottom: 12px;
  box-shadow: 0 2px 8px rgba(var(--macos-blue-rgb), 0.3);
}

.standard-description {
  margin: 0;
  font-size: var(--fs-base);
  color: var(--macos-text-secondary);
  line-height: 1.5;
}

/* 指标说明区域 - 紧凑版本 */
.metrics-section {
  background: linear-gradient(135deg, rgba(var(--macos-blue-rgb), 0.03) 0%, rgba(var(--macos-blue-rgb), 0.01) 100%);
  border: 1px solid rgba(var(--macos-blue-rgb), 0.08);
  border-radius: 8px;
  padding: 16px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--fs-title);
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 16px 0;
}

.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 12px;
}

/* 指标卡片 - 紧凑版本 */
.metric-card {
  background: var(--macos-background);
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  overflow: hidden;
  transition: all 0.2s ease;
}

.metric-card:hover {
  border-color: var(--macos-blue);
  box-shadow: 0 4px 12px rgba(var(--macos-blue-rgb), 0.1);
  transform: translateY(-1px);
}

.metric-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
}

.metric-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 8px;
  color: white;
  flex-shrink: 0;
}

.metric-icon-wrapper.cps {
  background: linear-gradient(135deg, #3b82f6 0%, #1d4ed8 100%);
}

.metric-icon-wrapper.wpm {
  background: linear-gradient(135deg, #10b981 0%, #059669 100%);
}

.metric-icon-wrapper.cpl {
  background: linear-gradient(135deg, #f59e0b 0%, #d97706 100%);
}

.metric-icon {
  width: 16px;
  height: 16px;
}

.metric-title h5 {
  margin: 0 0 3px 0;
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--macos-text-primary);
}

.metric-label { text-transform: uppercase; letter-spacing: 0.5px; }

.metric-description {
  padding: 12px;
}

.metric-description p {
  margin: 0 0 12px 0;
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
  line-height: 1.4;
}

/* 阈值列表 - 紧凑版本 */
.threshold-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.threshold-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 10px;
  background: var(--macos-background-secondary);
  border-radius: 5px;
  border-left: 3px solid;
}

.threshold-item.normal {
  border-left-color: #10b981;
  background: rgba(16, 185, 129, 0.05);
}

.threshold-item.warning {
  border-left-color: #f59e0b;
  background: rgba(245, 158, 11, 0.05);
}

.threshold-item.danger {
  border-left-color: #ef4444;
  background: rgba(239, 68, 68, 0.05);
}

.threshold-label { font-weight: 500; }

.threshold-value {
  font-size: var(--fs-caption);
  font-weight: 600;
  font-family: var(--font-mono);
}

.threshold-item.normal .threshold-value {
  color: #10b981;
}

.threshold-item.warning .threshold-value {
  color: #f59e0b;
}

.threshold-item.danger .threshold-value {
  color: #ef4444;
}

.level-normal {
  color: var(--macos-success-text);
  font-weight: 600;
}

.level-warning {
  color: var(--macos-warning-text);
  font-weight: 600;
}

.level-danger {
  color: var(--macos-danger-text);
  font-weight: 600;
}

/* 滚动条样式 */
.metrics-panel::-webkit-scrollbar {
  width: 6px;
}

.metrics-panel::-webkit-scrollbar-track {
  background: transparent;
}

.metrics-panel::-webkit-scrollbar-thumb {
  background: var(--macos-scrollbar-thumb);
  border-radius: 3px;
}

.metrics-panel::-webkit-scrollbar-thumb:hover {
  background: var(--macos-scrollbar-thumb-hover);
}

/* 响应式设计 - 紧凑版本 */
@media (max-width: 768px) {
  .language-tabs {
    flex-direction: column;
    gap: 8px;
  }

  .tabs-left {
    justify-content: center;
  }

  .tabs-right {
    justify-content: center;
  }

  .metrics-panel {
    padding: 12px;
    max-height: 350px;
  }

  .metrics-grid {
    grid-template-columns: 1fr;
    gap: 10px;
  }

  .metric-header {
    padding: 10px;
  }

  .metric-description {
    padding: 10px;
  }

  .threshold-list {
    gap: 5px;
  }

  .threshold-item {
    padding: 5px 8px;
  }
}

/* 精致单行标准信息样式 - 紧凑版本 */
.standard-info-compact {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: linear-gradient(135deg, rgba(var(--macos-blue-rgb), 0.08) 0%, rgba(var(--macos-blue-rgb), 0.03) 100%);
  border: 1px solid rgba(var(--macos-blue-rgb), 0.12);
  border-left: 4px solid var(--macos-blue);
  border-radius: 6px;
  margin-bottom: 16px;
  font-size: var(--fs-sub);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  transition: all 0.2s ease;
}

.standard-info-compact:hover {
  background: linear-gradient(135deg, rgba(var(--macos-blue-rgb), 0.12) 0%, rgba(var(--macos-blue-rgb), 0.06) 100%);
  border-color: rgba(var(--macos-blue-rgb), 0.2);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
  transform: translateY(-1px);
}

.standard-indicator {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  background: linear-gradient(135deg, var(--macos-blue) 0%, #4A90E2 100%);
  border-radius: 50%;
  color: white;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(var(--macos-blue-rgb), 0.3);
}

.indicator-icon {
  width: 12px;
  height: 12px;
  filter: drop-shadow(0 1px 2px rgba(0, 0, 0, 0.1));
}

.current-standard {
  color: var(--macos-text-primary);
  font-weight: 500;
  flex-shrink: 0;
  letter-spacing: 0.2px;
}

.current-standard strong {
  font-weight: 700;
  color: var(--macos-blue);
  background: linear-gradient(135deg, var(--macos-blue) 0%, #4A90E2 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.standard-desc {
  color: var(--macos-text-secondary);
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-style: italic;
  opacity: 0.85;
  position: relative;
}

.standard-desc::before {
  content: "";
  position: absolute;
  left: -12px;
  top: 50%;
  transform: translateY(-50%);
  width: 1px;
  height: 16px;
  background: rgba(var(--macos-text-secondary-rgb), 0.2);
}

/* 响应式优化 - 紧凑版本 */
@media (max-width: 640px) {
  .standard-info-compact {
    flex-wrap: wrap;
    gap: 10px;
    padding: 10px 14px;
  }

  .standard-desc {
    flex-basis: 100%;
    margin-top: 3px;
    white-space: normal;
    overflow: visible;
    text-overflow: unset;
    padding-left: 6px;
  }

  .standard-desc::before {
    display: none;
  }
}
</style>
