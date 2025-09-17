<template>
  <div class="sep-config">
    <!-- Export group to match DownloadTaskPanel style -->
    <div class="macos-group">
      <div class="macos-box card-frosted card-translucent">
        <!-- Format selector -->
        <div class="macos-row">
          <div class="k">{{ $t('subtitle.export.format') }}</div>
          <div class="v">
            <select v-model="selectedFormat" @change="onFormatChange" class="select-macos select-macos-xs select-fixed">
              <option value="srt">SRT</option>
              <option value="vtt">WebVTT</option>
              <option value="ass">ASS/SSA</option>
              <option value="itt">ITT</option>
              <option value="fcpxml">FCPXML</option>
            </select>
          </div>
        </div>

        <template v-if="currentConfig">
          <!-- FCPXML config -->
          <template v-if="selectedFormat === 'fcpxml'">
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.resolution') }}</div>
              <div class="v">
                <select v-model="resolutionPreset" @change="onResolutionPresetChange" class="select-macos select-macos-xs select-fixed">
                  <option value="1920x1080">1080p</option>
                  <option value="3840x2160">4K</option>
                  <option value="1280x720">720p</option>
                  <option value="2560x1440">1440p</option>
                  <option value="custom">{{ $t('subtitle.export.customize') }}</option>
                </select>
              </div>
            </div>
            <div class="macos-row" v-if="resolutionPreset === 'custom'">
              <div class="k"></div>
              <div class="v">
                <div class="dual-input inline">
                  <div class="input-group">
                    <label class="input-label">{{ $t('subtitle.export.width') }}</label>
                    <input type="number" v-model.number="currentConfig.width" class="input-macos input-compact" placeholder="1920" min="1" max="7680" />
                  </div>
                  <div class="input-separator">×</div>
                  <div class="input-group">
                    <label class="input-label">{{ $t('subtitle.export.height') }}</label>
                    <input type="number" v-model.number="currentConfig.height" class="input-macos input-compact" placeholder="1080" min="1" max="4320" />
                  </div>
                </div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.frame_rate') }}</div>
              <div class="v">
                <select v-model.number="currentConfig.frame_rate" class="select-macos select-macos-xs select-fixed">
                  <option :value="23.976">23.976 fps</option>
                  <option :value="24">24 fps</option>
                  <option :value="25">25 fps</option>
                  <option :value="29.97">29.97 fps</option>
                  <option :value="30">30 fps</option>
                  <option :value="60">60 fps</option>
                </select>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.color_space') }}</div>
              <div class="v">
                <select v-model="currentConfig.color_space" class="select-macos select-macos-xs select-fixed">
                  <option value="1-1-1 (Rec. 709)">Rec. 709</option>
                  <option value="Rec. 2020">Rec. 2020</option>
                  <option value="P3-D65">P3-D65</option>
                </select>
              </div>
            </div>
          </template>

          <!-- ITT config -->
          <template v-else-if="selectedFormat === 'itt'">
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.frame_rate') }}</div>
              <div class="v">
                <select v-model.number="currentConfig.frame_rate" class="select-macos select-macos-xs select-fixed">
                  <option :value="23.976">23.976 fps</option>
                  <option :value="24">24 fps</option>
                  <option :value="25">25 fps</option>
                  <option :value="29.97">29.97 fps</option>
                  <option :value="30">30 fps</option>
                  <option :value="60">60 fps</option>
                </select>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.language') }}</div>
              <div class="v">
                <input v-model="currentConfig.language" class="input-macos input-fixed input-grow-on-focus" placeholder="zh-CN" />
              </div>
            </div>
          </template>

          <!-- SRT config -->
          <template v-else-if="selectedFormat === 'srt'">
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.encoding') }}</div>
              <div class="v">
                <select v-model="currentConfig.encoding" class="select-macos select-macos-xs select-fixed">
                  <option value="utf-8">UTF-8</option>
                  <option value="gbk">GBK</option>
                  <option value="big5">Big5</option>
                </select>
              </div>
            </div>
          </template>

          <!-- ASS config -->
          <template v-else-if="selectedFormat === 'ass'">
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.width') }}</div>
              <div class="v">
                <input type="number" v-model.number="currentConfig.play_res_x" class="input-macos input-fixed" placeholder="1920" min="1" max="7680" />
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.height') }}</div>
              <div class="v">
                <input type="number" v-model.number="currentConfig.play_res_y" class="input-macos input-fixed" placeholder="1080" min="1" max="4320" />
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.file_title') }}</div>
              <div class="v">
                <input v-model="currentConfig.title" class="input-macos input-fixed input-grow-on-focus" :placeholder="$t('subtitle.export.file_title')" />
              </div>
            </div>
          </template>

          <!-- VTT config -->
          <template v-else-if="selectedFormat === 'vtt'">
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.kind') }}</div>
              <div class="v">
                <select v-model="currentConfig.kind" class="select-macos select-macos-xs select-fixed">
                  <option value="subtitles">{{ $t('subtitle.export.subtitles') }}</option>
                  <option value="captions">{{ $t('subtitle.export.captions') }}</option>
                  <option value="descriptions">{{ $t('subtitle.export.descriptions') }}</option>
                </select>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('subtitle.export.language') }}</div>
              <div class="v">
                <input v-model="currentConfig.language" class="input-macos input-fixed input-grow-on-focus" placeholder="zh-CN" />
              </div>
            </div>
          </template>
        </template>

      </div>
    </div>
    
    <!-- Footer actions (outside group) -->
    <div class="footer-actions">
      <button class="btn-glass" @click="saveConfig">
        <Icon name="shield-check" class="w-4 h-4 mr-2" />
        {{ $t('subtitle.common.save') }}
      </button>
      <button class="btn-glass btn-primary" @click="exportSubtitles">
        <Icon name="download-file" class="w-4 h-4 mr-2" />
        {{ $t('subtitle.common.export') }}
      </button>
    </div>
  </div>
</template>

<script>
import { subtitleService } from '@/services/subtitleService.js'

  export default {
    name: 'SubtitleExportConfig',
    props: {
      projectData: { type: Object, required: true },
      currentLanguage: { type: String, required: true },
    },
  // i18n: use global $t in template and methods
    data() {
      return {
      selectedFormat: 'srt',
      formatLabels: { srt: 'SRT', vtt: 'WebVTT', ass: 'ASS/SSA', itt: 'ITT', fcpxml: 'FCPXML' },
      currentConfig: null,
      resolutionPreset: '1920x1080',
      showAdvanced: false,
      exportConfigs: {
        srt: { encoding: 'utf-8' },
        vtt: { kind: 'subtitles', language: 'zh-CN' },
        ass: { play_res_x: 1920, play_res_y: 1080, title: '' },
        // Note: frame_rate starts at 0 so we can intelligently
        // default from source FPS or FCPXML FPS during init
        itt: { frame_rate: 0, language: 'zh-CN' },
        fcpxml: {
          frame_rate: 25, width: 1920, height: 1080, color_space: '1-1-1 (Rec. 709)',
          version: '1.9', library_name: '', event_name: '', project_name: '', default_lane: 1,
          title_effect: '', start_timecode: ''
        }
      },
      savedExportConfigs: null,
    }
  },
  mounted() {
    this.initializeConfigs();
    this.onFormatChange();
  },
  methods: {
    initializeConfigs() {
      if (this.projectData.metadata?.export_configs) {
        const configs = this.projectData.metadata.export_configs
        if (configs.fcpxml) this.exportConfigs.fcpxml = { ...this.exportConfigs.fcpxml, ...configs.fcpxml }
        if (configs.srt) this.exportConfigs.srt = { ...this.exportConfigs.srt, ...configs.srt }
        if (configs.ass) this.exportConfigs.ass = { ...this.exportConfigs.ass, ...configs.ass }
        if (configs.vtt) this.exportConfigs.vtt = { ...this.exportConfigs.vtt, ...configs.vtt }
        if (configs.itt) this.exportConfigs.itt = { ...this.exportConfigs.itt, ...configs.itt }
      }
      // Robust ITT fps fallback — prefer server ITT fps, else source nominal fps, else FCPXML fps, else source effective fps, else 25
      const serverIttFps = Number(this.exportConfigs?.itt?.frame_rate || 0)
      const srcNominal = Number(this.projectData?.metadata?.source_itt?.frame_rate || 0)
      const fcpxFps = Number(this.exportConfigs?.fcpxml?.frame_rate || 0)
      const srcEffective = Number(this.projectData?.metadata?.source_info?.original_fps || 0)
      if (!(serverIttFps > 0)) {
        this.exportConfigs.itt.frame_rate = (srcNominal > 0 ? srcNominal : (fcpxFps > 0 ? fcpxFps : (srcEffective > 0 ? srcEffective : 25)))
      } else if (serverIttFps === 25 && srcNominal > 0 && srcNominal !== 25) {
        // 修正旧项目被错误保存为25fps的情况（优先原始名义帧率）
        this.exportConfigs.itt.frame_rate = srcNominal
      }
      // Ensure ITT language fallback
      if (!this.exportConfigs.itt.language) {
        const meta = this.projectData?.language_metadata || {}
        const langs = Object.keys(meta)
        this.exportConfigs.itt.language = langs.length ? langs[0] : (this.currentLanguage || 'en-US')
      }
      if (!this.exportConfigs.fcpxml.project_name) {
        this.exportConfigs.fcpxml.project_name = this.projectData.metadata?.name || this.projectData.project_name
      }
      if (!this.exportConfigs.fcpxml.library_name) this.exportConfigs.fcpxml.library_name = this.exportConfigs.fcpxml.project_name
      if (!this.exportConfigs.fcpxml.event_name) this.exportConfigs.fcpxml.event_name = this.exportConfigs.fcpxml.project_name
      this.updateResolutionPreset()
      this.savedExportConfigs = JSON.parse(JSON.stringify(this.exportConfigs))
    },
    updateResolutionPreset() {
      const { width, height } = this.exportConfigs.fcpxml
      const resolution = `${width}x${height}`
      const presets = ['1920x1080', '3840x2160', '1280x720', '2560x1440']
      this.resolutionPreset = presets.includes(resolution) ? resolution : 'custom'
    },
    onResolutionPresetChange() {
      if (this.resolutionPreset !== 'custom') {
        const [w, h] = this.resolutionPreset.split('x').map(Number)
        this.currentConfig.width = w; this.currentConfig.height = h
      }
    },
    onFormatChange() {
      this.currentConfig = this.exportConfigs[this.selectedFormat]
      if (this.selectedFormat === 'fcpxml') this.updateResolutionPreset()
    },
    hasConfigChanges() { if (!this.savedExportConfigs) return false; return JSON.stringify(this.exportConfigs) !== JSON.stringify(this.savedExportConfigs) },
    validateFcpxmlConfig() {
      const c = this.exportConfigs.fcpxml, errs = []
      if (!c.width || c.width <= 0) errs.push(this.$t('subtitle.export.width_warning'))
      if (!c.height || c.height <= 0) errs.push(this.$t('subtitle.export.height_warning'))
      if (!c.frame_rate || c.frame_rate <= 0) errs.push(this.$t('subtitle.export.frame_rate_warning'))
      if (!c.project_name?.trim()) errs.push(this.$t('subtitle.export.file_title_warning'))
      return errs
    },
    async saveConfig() {
      try {
        if (this.selectedFormat === 'fcpxml') {
          const errors = this.validateFcpxmlConfig(); if (errors.length) { $message.error(errors.join(',')); return }
        }
        const updatedMetadata = { ...this.projectData.metadata, export_configs: this.exportConfigs }
        const result = await subtitleService.saveProjectMetadata(updatedMetadata)
        if (result.success) {
          const projectData = JSON.parse(result.data)
          this.$emit('update:projectData', projectData)
          this.savedExportConfigs = JSON.parse(JSON.stringify(this.exportConfigs))
          try { $message?.success?.(this.$t('subtitle.common.save') + ' ' + this.$t('common.success')) } catch {}
        } else { throw new Error(result.msg) }
      } catch (e) { $notification?.error?.({ title: this.$t('common.error'), content: e?.message || String(e) }) }
    },
    async exportSubtitles() {
      try {
        if (this.selectedFormat === 'fcpxml') {
          const errors = this.validateFcpxmlConfig(); if (errors.length) { $message.error(errors.join(',')); return }
        }
        const response = await subtitleService.exportSubtitles(this.projectData.id, this.currentLanguage, this.selectedFormat, this.exportConfigs[this.selectedFormat])
        if (response.cancelled) { return }
        if (response.success) { $notification?.success?.({ title: this.$t('subtitle.export.title'), content: this.$t('subtitle.export.export_success') }) }
        else { throw new Error(response.msg || this.$t('common.error')) }
      } catch (e) { $notification?.error?.({ title: this.$t('common.error'), content: e?.message || String(e) }) }
    },
  }
}
</script>

<style scoped>
.sep-config { font-size: var(--fs-base); color: var(--macos-text-primary); }

/* use global .macos-group/.macos-box/.macos-row from macos-components */

/* inside-box chip behavior same as panel */
.macos-box .chip-frosted.chip-translucent { background: transparent; border-color: var(--macos-separator); color: var(--macos-text-secondary); box-shadow: none; }
.macos-box .chip-frosted.chip-translucent:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); border-color: var(--macos-blue); color: #fff; }

/* layout helpers */
.dual-input { display:flex; align-items:center; gap:8px; }
.dual-input.inline { display:inline-flex; }
.input-group { display:flex; align-items:center; gap:6px; }
.input-label { font-size: var(--fs-sub); color: var(--macos-text-secondary); }
.input-separator { color: var(--macos-text-tertiary); }
/* bottom action buttons */
.footer-actions { display:flex; align-items:center; justify-content:center; gap:8px; margin-top: 10px; }

/* Fixed select width for visual consistency across inspector */
.select-fixed { width: 130px; }

/* Match input to select size and expand on focus for better readability */
.input-fixed { width: 130px; height: 26px; padding: 4px 8px; font-size: var(--fs-sub); transition: height .12s ease; }
.input-grow-on-focus:focus { height: 52px; }

/* compact pills for summary (kept for potential reuse) */
.meta-group.small { display:inline-flex; align-items:center; gap:8px; padding: 0 6px; height: 22px; border: 1px solid var(--macos-separator); border-radius: 999px; background: var(--macos-background); color: var(--macos-text-secondary); font-size: var(--fs-sub); }
.meta-group.small .item { display:inline-flex; align-items:center; gap: 4px; font-size: 11.5px; }
.meta-group.small .divider-v { width: 1px; height: 12px; background: var(--macos-divider-weak); }
.meta-group.small .w-3.5, .meta-group.small .h-3.5 { display:block; }

</style>
