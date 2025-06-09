<template>
  <div class="export-config-macos">
    <!-- 顶部标题栏 -->
    <div class="config-header">
      <h3>{{ $t('subtitle.export.title') }}</h3>
    </div>

    <!-- 主要内容区域 -->
    <div class="config-content">
      <!-- 格式选择 - 紧凑设计 -->
      <div class="format-section">
        <div class="format-selector">
          <label class="format-label">{{ $t('subtitle.export.format') }}</label>
          <select v-model="selectedFormat" @change="onFormatChange" class="select-macos">
            <option value="srt">SRT</option>
            <!-- <option value="vtt">WebVTT</option>
            <option value="ass">ASS/SSA</option> -->
            <option value="fcpxml">FCPXML</option>
          </select>
        </div>
      </div>

      <!-- 配置选项 - 卡片式布局 -->
      <div class="card-macos config-card" v-if="currentConfig">
        <div class="card-header">
          <span class="config-title">{{ formatLabels[selectedFormat] }} {{ $t('subtitle.export.parameters') }}</span>
        </div>

        <div class="card-content">
          <!-- FCPXML 配置 -->
          <template v-if="selectedFormat === 'fcpxml'">
            <!-- 分辨率 -->
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.resolution') }}</label>
              <select v-model="resolutionPreset" @change="onResolutionPresetChange" class="select-macos">
                <option value="1920x1080">1080p</option>
                <option value="3840x2160">4K</option>
                <option value="1280x720">720p</option>
                <option value="2560x1440">1440p</option>
                <option value="custom">{{ $t('subtitle.export.customize') }}</option>
              </select>
            </div>

            <!-- 自定义分辨率 -->
            <div v-if="resolutionPreset === 'custom'" class="config-item">
              <div class="dual-input">
                <div class="input-group">
                  <label class="input-label">{{ $t('subtitle.export.width') }}</label>
                  <input type="number" v-model.number="currentConfig.width" class="input-macos input-compact"
                    placeholder="1920" min="1" max="7680" />
                </div>
                <div class="input-separator">×</div>
                <div class="input-group">
                  <label class="input-label">{{ $t('subtitle.export.height') }}</label>
                  <input type="number" v-model.number="currentConfig.height" class="input-macos input-compact"
                    placeholder="1080" min="1" max="4320" />
                </div>
              </div>
            </div>

            <!-- 帧率 -->
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.frame_rate') }}</label>
              <select v-model.number="currentConfig.frame_rate" class="select-macos">
                <option :value="23.976">23.976 fps</option>
                <option :value="24">24 fps</option>
                <option :value="25">25 fps</option>
                <option :value="29.97">29.97 fps</option>
                <option :value="30">30 fps</option>
                <option :value="60">60 fps</option>
              </select>
            </div>

            <!-- 色彩空间 -->
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.color_space') }}</label>
              <select v-model="currentConfig.color_space" class="select-macos">
                <option value="1-1-1 (Rec. 709)">Rec. 709</option>
                <option value="Rec. 2020">Rec. 2020</option>
                <option value="P3-D65">P3-D65</option>
              </select>
            </div>
          </template>

          <!-- SRT 配置 -->
          <template v-else-if="selectedFormat === 'srt'">
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.encoding') }}</label>
              <select v-model="currentConfig.encoding" class="select-macos">
                <option value="utf-8">UTF-8</option>
                <option value="gbk">GBK</option>
                <option value="big5">Big5</option>
              </select>
            </div>
          </template>

          <!-- ASS 配置 -->
          <template v-else-if="selectedFormat === 'ass'">
            <div class="config-item">
              <div class="dual-input">
                <div class="input-group">
                  <label class="input-label">{{ $t('subtitle.export.resolution_abbreviation') }} X</label>
                  <input type="number" v-model.number="currentConfig.play_res_x" class="input-macos input-compact"
                    placeholder="1920" min="1" max="7680" />
                </div>
                <div class="input-separator">×</div>
                <div class="input-group">
                  <label class="input-label">{{ $t('subtitle.export.resolution_abbreviation') }} Y</label>
                  <input type="number" v-model.number="currentConfig.play_res_y" class="input-macos input-compact"
                    placeholder="1080" min="1" max="4320" />
                </div>
              </div>
            </div>
            <!-- ASS 格式配置中的标题输入框 -->
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.file_title') }}</label>
              <input v-model="currentConfig.title" class="input-macos input-full"/>
            </div>

          </template>

          <!-- VTT 配置 -->
          <template v-else-if="selectedFormat === 'vtt'">
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.kind') }}</label>
              <select v-model="currentConfig.kind" class="select-macos">
                <option value="subtitles">{{ $t('subtitle.export.subtitles') }}</option>
                <option value="captions">{{ $t('subtitle.export.captions') }}</option>
                <option value="descriptions">{{ $t('subtitle.export.descriptions') }}</option>
              </select>
            </div>
            <!-- VTT 格式配置中的语言输入框 -->
            <div class="config-item">
              <label class="item-label">{{ $t('subtitle.export.language') }}</label>
              <input v-model="currentConfig.language" class="input-macos input-full" placeholder="zh-CN" />
            </div>
          </template>
        </div>
      </div>
    </div>

    <!-- 底部操作栏 -->
    <div class="config-actions">
      <button @click="saveConfig" class="btn-macos-secondary">
        {{ $t('subtitle.common.save') }}
      </button>
      <button @click="exportSubtitles" class="btn-macos-primary">
        {{ $t('subtitle.common.export') }}
      </button>
    </div>
  </div>
</template>

<script>
import { subtitleService } from '@/services/subtitleService.js'
import { useI18n } from 'vue-i18n'

export default {
  name: 'SubtitleExportConfig',
  props: {
    projectData: {
      type: Object,
      required: true
    },
    currentLanguage: {
      type: String,
      required: true
    }
  },
  setup() {
    const { t } = useI18n()
    return { t }
  },
  data() {
    return {
      selectedFormat: 'srt',
      formatLabels: {
        srt: 'SRT',
        vtt: 'WebVTT',
        ass: 'ASS/SSA',
        fcpxml: 'FCPXML'
      },
      currentConfig: null,
      resolutionPreset: '1920x1080',
      showAdvanced: false,
      exportConfigs: {
        srt: {
          encoding: 'utf-8'
        },
        vtt: {
          kind: 'subtitles',
          language: 'zh-CN'
        },
        ass: {
          play_res_x: 1920,
          play_res_y: 1080,
          title: ''
        },
        fcpxml: {
          frame_rate: 25,
          width: 1920,
          height: 1080,
          color_space: '1-1-1 (Rec. 709)',
          version: '1.9',
          library_name: '',
          event_name: '',
          project_name: '',
          default_lane: 1,
          title_effect: '',
          start_timecode: ''
        }
      },
      savedExportConfigs: null
    }
  },
  mounted() {
    this.initializeConfigs()
    this.onFormatChange()
  },
  methods: {
    initializeConfigs() {
      if (this.projectData.metadata?.export_configs) {
        const configs = this.projectData.metadata.export_configs

        if (configs.fcpxml) {
          this.exportConfigs.fcpxml = { ...this.exportConfigs.fcpxml, ...configs.fcpxml }
        }
        if (configs.srt) {
          this.exportConfigs.srt = { ...this.exportConfigs.srt, ...configs.srt }
        }
        if (configs.ass) {
          this.exportConfigs.ass = { ...this.exportConfigs.ass, ...configs.ass }
        }
        if (configs.vtt) {
          this.exportConfigs.vtt = { ...this.exportConfigs.vtt, ...configs.vtt }
        }
      }

      if (!this.exportConfigs.fcpxml.project_name) {
        this.exportConfigs.fcpxml.project_name = this.projectData.metadata?.name || this.projectData.project_name
      }

      if (!this.exportConfigs.fcpxml.library_name) {
        this.exportConfigs.fcpxml.library_name = this.exportConfigs.fcpxml.project_name
      }
      if (!this.exportConfigs.fcpxml.event_name) {
        this.exportConfigs.fcpxml.event_name = this.exportConfigs.fcpxml.project_name
      }

      this.updateResolutionPreset()
      this.savedExportConfigs = JSON.parse(JSON.stringify(this.exportConfigs))
    },

    updateResolutionPreset() {
      const { width, height } = this.exportConfigs.fcpxml
      const resolution = `${width}x${height}`

      const presets = ['1920x1080', '3840x2160', '1280x720', '2560x1440']
      if (presets.includes(resolution)) {
        this.resolutionPreset = resolution
      } else {
        this.resolutionPreset = 'custom'
      }
    },

    onResolutionPresetChange() {
      if (this.resolutionPreset !== 'custom') {
        const [width, height] = this.resolutionPreset.split('x').map(Number)
        this.currentConfig.width = width
        this.currentConfig.height = height
      }
    },

    onFormatChange() {
      this.currentConfig = this.exportConfigs[this.selectedFormat]
      if (this.selectedFormat === 'fcpxml') {
        this.updateResolutionPreset()
      }
    },

    hasConfigChanges() {
      if (!this.savedExportConfigs) return false
      return JSON.stringify(this.exportConfigs) !== JSON.stringify(this.savedExportConfigs)
    },

    validateFcpxmlConfig() {
      const config = this.exportConfigs.fcpxml
      const errors = []

      if (!config.width || config.width <= 0) {
        errors.push(this.t('subtitle.export.width_warning'))
      }
      if (!config.height || config.height <= 0) {
        errors.push(this.t('subtitle.export.height_warning'))
      }
      if (!config.frame_rate || config.frame_rate <= 0) {
        errors.push(this.t('subtitle.export.frame_rate_warning'))
      }
      if (!config.project_name?.trim()) {
        errors.push(this.t('subtitle.export.file_title_warning'))
      }

      return errors
    },

    async saveConfig() {
      try {
        if (this.selectedFormat === 'fcpxml') {
          const errors = this.validateFcpxmlConfig()
          if (errors.length > 0) {
            $message.error(errors.join(','))
            return
          }
        }

        const updatedMetadata = {
          ...this.projectData.metadata,
          export_configs: this.exportConfigs
        }

        const result = await subtitleService.saveProjectMetadata(updatedMetadata)
        if (result.success) {
          const projectData = JSON.parse(result.data)
          this.$emit('update:projectData', projectData)
          this.savedExportConfigs = JSON.parse(JSON.stringify(this.exportConfigs))
        } else {
          throw new Error(result.msg)
        }
      } catch (error) {
        $message.error(error.message)
      }
    },

    async exportSubtitles() {
      try {
        if (this.hasConfigChanges()) {
          const confirmed = await $dialog.confirm(this.t('subtitle.export.config_change_warning'))
          if (!confirmed) {
            return
          }

          try {
            await this.saveConfig()
          } catch (error) {
            return
          }
        }

        const result = await subtitleService.exportSubtitles(this.projectData.id, this.currentLanguage, this.selectedFormat)
        if (result.success) {
          $message.info(this.t('subtitle.export.export_success'))
        }
      } catch (error) {
        $message.error(error.message)
      }
    }
  }
}
</script>

<style scoped>
.export-config-macos {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--macos-background);
  border-radius: 10px;
  overflow: hidden;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
}

.config-header {
  padding: 12px 16px;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
  backdrop-filter: blur(20px);
}

.config-header h3 {
  margin: 0;
  font-size: 15px;
  font-weight: 600;
  color: var(--macos-text-primary);
  text-align: center;
}

.config-content {
  flex: 1;
  padding: 16px;
  overflow-y: auto;
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent;
}

.config-content::-webkit-scrollbar {
  width: 4px;
}

.config-content::-webkit-scrollbar-track {
  background: transparent;
}

.config-content::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 2px;
}

.format-section {
  margin-bottom: 16px;
}

.format-selector {
  display: flex;
  align-items: center;
  gap: 12px;
}

.format-label {
  font-size: 13px;
  font-weight: 500;
  color: var(--macos-text-secondary);
  min-width: 40px;
  flex-shrink: 0;
}

.config-card {
  margin-top: 0;
}

.card-header {
  padding: 10px 14px;
  background: rgba(0, 0, 0, 0.02);
  border-bottom: 1px solid var(--macos-separator);
}

.config-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--macos-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.card-content {
  padding: 14px;
}

.config-item {
  margin-bottom: 14px;
}

.config-item:last-child {
  margin-bottom: 0;
}

.item-label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--macos-text-primary);
  margin-bottom: 6px;
}

.dual-input {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  width: 100%;
  max-width: 100%;
}

.input-group {
  flex: 1;
  min-width: 0;
  /* 防止flex子项溢出 */
}

.input-compact {
  width: 100%;
  min-width: 60px;
  max-width: 100%;
  box-sizing: border-box;
}

.input-label {
  display: block;
  font-size: 11px;
  font-weight: 500;
  color: var(--macos-text-secondary);
  margin-bottom: 4px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.input-separator {
  font-size: 14px;
  color: var(--macos-text-secondary);
  margin-bottom: 7px;
  font-weight: 500;
  flex-shrink: 0;
  /* 防止分隔符被压缩 */
}

/* 确保config-item容器也有正确的宽度控制 */
.config-item {
  margin-bottom: 14px;
  width: 100%;
  max-width: 100%;
}

/* 响应式优化 */
@media (max-width: 480px) {
  .config-item {
    margin-bottom: 12px;
  }

  .input-full {
    font-size: 14px;
    padding: 8px 10px;
  }

  .item-label {
    font-size: 12px;
    margin-bottom: 5px;
  }
}

/* 响应式设计优化 */
@media (max-width: 480px) {
  .dual-input {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .input-separator {
    display: none;
  }

  .input-group {
    width: 100%;
  }

  .input-label {
    font-size: 12px;
  }
}

/* 更小屏幕的额外优化 */
@media (max-width: 360px) {
  .dual-input {
    gap: 8px;
  }

  .input-compact {
    font-size: 14px;
    padding: 6px 8px;
  }
}

.config-actions {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: var(--macos-background-secondary);
  border-top: 1px solid var(--macos-separator);
}

.config-actions .btn-macos-primary,
.config-actions .btn-macos-secondary {
  flex: 1;
}

/* 响应式优化 */
@media (max-width: 400px) {
  .config-content {
    padding: 12px;
  }

  .format-selector {
    flex-direction: column;
    align-items: stretch;
    gap: 6px;
  }

  .format-label {
    min-width: auto;
  }

  .config-actions {
    flex-direction: column;
    gap: 6px;
  }

  .dual-input {
    flex-direction: column;
    gap: 8px;
  }

  .input-separator {
    display: none;
  }
}
</style>