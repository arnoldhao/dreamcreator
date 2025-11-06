<template>
  <!-- host fills its parent absolutely to avoid height chain issues -->
  <div class="sr-root">

    <!-- left list (independent of background and divider) -->
    <aside class="sr-left" :style="{ width: leftWidth, minWidth: leftWidth }">
      <div v-for="g in groups" :key="g.key" class="sr-item" :class="{ active: current === g.key }"
        @click="current = g.key">
        <Icon :name="g.icon" class="sr-item-icon" />
        <span class="sr-item-label truncate">{{ g.label }}</span>
      </div>
    </aside>

    <!-- right content area (background fills, card only wraps content) -->
    <section class="sr-right">
      <div class="sr-section-head">
        <div class="sr-section-title">{{ title }}</div>
      </div>
      <div class="sr-section-divider"></div>
      <!-- About section renders without card/frame -->
      <template v-if="current === 'about'">
        <div class="about-wrap">
          <img :src="iconUrl" alt="app icon" class="about-icon" />
          <div class="about-title">{{ Project.Name }}（{{ Project.DisplayNameZh }}） <span class="ver">v{{ appVersion }}</span></div>
          <div class="about-links">
            <a href="#" @click.prevent="openWebsite" class="link-text" :title="$t('dialogue.about.official_website')">{{ $t('dialogue.about.official_website') }}</a>
          </div>
          <div class="about-social">
            <a v-if="Project.Email" :href="`mailto:${Project.Email}`" class="contact-email link-text">
              <Icon name="mail" class="w-4 h-4 mr-1" />
              <span>{{ Project.Email }}</span>
            </a>
            <button class="icon-btn" @click="openGithub" :data-tooltip="'GitHub'" aria-label="GitHub">
              <Icon name="github" class="w-5 h-5" />
            </button>
            <button class="icon-btn" @click="openTwitter" :data-tooltip="'Twitter'" aria-label="Twitter">
              <Icon name="twitter" class="w-5 h-5" />
            </button>
          </div>
          <div class="about-options">
            <div class="about-option">
              <div class="about-option-title">{{ $t('settings.about.auto_check_update') }}</div>
              <label class="about-toggle">
                <input type="checkbox" class="about-toggle-input" v-model="prefStore.general.checkUpdate" @change="onAboutToggleChange" />
                <span class="about-toggle-slider"></span>
              </label>
            </div>
            <div class="about-option">
              <div class="about-option-title">{{ $t('settings.about.telemetry_opt_in') }}</div>
              <label class="about-toggle">
                <input type="checkbox" class="about-toggle-input" v-model="prefStore.telemetry.enabled" @change="onTelemetryToggleChange" />
                <span class="about-toggle-slider"></span>
              </label>
            </div>
          </div>
          <div class="about-actions">
            <button class="btn-glass" @click="openChangelog">
              <Icon name="list" class="w-4 h-4 mr-1" />
              {{ $t('settings.about.changelog') }}
            </button>
            <button class="btn-glass btn-primary" @click="checkUpdates">
              <Icon name="refresh" class="w-4 h-4 mr-1" />
              {{ $t('menu.check_update') }}
            </button>
          </div>
        </div>
      </template>

      <!-- other sections use card container -->
      <template v-else>
      <div class="sr-card card-frosted card-translucent">
        <div class="sr-card-body">
          <!-- Appearance -->
          <template v-if="current === 'appearance'">
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.appearance') }}</div>
              <div class="v">
                <div class="segmented">
                  <button v-for="option in prefStore.appearanceOption" :key="option.value" class="seg-item"
                    :class="{ active: prefStore.general.appearance === option.value }"
                    @click="prefStore.general.appearance = option.value; prefStore.savePreferences()" :title="option.label">
                    <Icon v-if="option.value === 'light'" name="sun" class="h-4 w-4" />
                    <Icon v-else-if="option.value === 'dark'" name="moon" class="h-4 w-4" />
                    <Icon v-else name="sun-moon" class="h-4 w-4" />
                  </button>
                </div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.theme') }}</div>
              <div class="v">
                <div class="accent-grid">
                  <button
                    v-for="c in accentOptions"
                    :key="c.value"
                    class="accent-btn"
                    :class="{ active: prefStore.general.theme === c.value }"
                    :style="{ '--accent-color': c.color }"
                    :title="c.label"
                    @click="onPickAccent(c.value)"
                  >
                    <span class="dot"></span>
                  </button>
                  <label class="accent-custom" :title="t('common.custom') || 'Custom'">
                    <input type="color" class="accent-color" v-model="customAccent" @input="onPickCustom" />
                  </label>
                </div>
              </div>
            </div>
            <!-- UI Style: Classic vs Frosted -->
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.ui_style') }}</div>
              <div class="v">
                <div class="segmented">
                  <button class="seg-item" :class="{ active: prefStore.general.uiStyle === 'classic' }"
                          @click="prefStore.general.uiStyle = 'classic'; applyUIStyle('classic'); prefStore.savePreferences()" :title="$t('settings.general.ui_style_classic')">
                    {{ $t('settings.general.ui_style_classic') }}
                  </button>
                  <button class="seg-item" :class="{ active: prefStore.general.uiStyle !== 'classic' }"
                          @click="prefStore.general.uiStyle = 'frosted'; applyUIStyle('frosted'); prefStore.savePreferences()" :title="$t('settings.general.ui_style_frosted')">
                    {{ $t('settings.general.ui_style_frosted') }}
                  </button>
                </div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.language') }}</div>
              <div class="v">
                <select class="input-macos select-macos select-macos-xs" v-model="prefStore.general.language"
                  @change="onLanguageChange">
                  <option v-for="option in prefStore.langOption" :key="option.value" :value="option.value">{{
                    option.label }}</option>
                </select>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.proxy') }}</div>
              <div class="v">
                <select class="input-macos select-macos select-macos-xs" v-model="prefStore.proxy.type"
                  @change="onProxyChange">
                  <option value="none">{{ $t('settings.general.proxy_none') }}</option>
                  <option value="system">{{ $t('settings.general.proxy_system') }}</option>
                  <option value="manual">{{ $t('settings.general.proxy_manual') }}</option>
                </select>
              </div>
            </div>
            <div class="macos-row" v-if="prefStore.proxy.type === 'manual'">
              <div class="k">{{ $t('settings.general.proxy_address') }}</div>
              <div class="v">
                <input type="text" class="input-macos select-macos text-left" v-model="proxyAddress"
                  placeholder="http://127.0.0.1:7890" @change="onProxyChange" />
              </div>
            </div>
          </template>

          <!-- Download -->
          <template v-else-if="current === 'download'">
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.download_directory') }}</div>
              <div class="v">
                <span class="result-field mr-2 text-secondary" :class="{ 'text-tertiary': !prefStore.download?.dir }" :title="prefStore.download?.dir">{{ prefStore.download?.dir }}</span>
                <button class="icon-glass" @click="onSelectDownloadDir" :data-tooltip="t('download.open_folder')" :aria-label="t('download.open_folder')"><Icon name="folder" class="w-4 h-4" /></button>
              </div>
            </div>
          </template>

          <!-- LLM Providers & Profiles -->
          <!-- removed: managed in Providers page -->

          <!-- Logs -->
          <template v-else-if="current === 'logs'">
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_level') }}</div>
              <div class="v">
                <select class="input-macos select-macos select-macos-xs text-left" v-model="prefStore.logger.level"
                  @change="prefStore.setLoggerConfig()">
                  <option value="debug">Debug</option>
                  <option value="info">Info</option>
                  <option value="warn">Warn</option>
                  <option value="error">Error</option>
                </select>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_max_size') }}</div>
              <div class="v control-short">
                <div class="measure-row"><input type="number" class="input-macos measure-input text-center" min="1"
                    max="100" v-model="prefStore.logger.max_size" @change="prefStore.setLoggerConfig()" /><span
                    class="measure-unit text-sm text-secondary">MB</span></div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_max_age') }}</div>
              <div class="v control-short">
                <div class="measure-row"><input type="number" class="input-macos measure-input text-center" min="1"
                    max="365" v-model="prefStore.logger.max_age" @change="prefStore.setLoggerConfig()" /><span
                    class="measure-unit text-sm text-secondary">{{ $t('settings.general.days') }}</span></div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_output') }}</div>
              <div class="v">
                <div class="toggle-row toggle-row-sm">
                  <label class="macos-check"><input type="checkbox" class="checkbox-macos"
                      v-model="prefStore.logger.enable_console" @change="prefStore.setLoggerConfig()" /><span>{{
                        $t('settings.general.log_console') }}</span></label>
                  <label class="macos-check"><input type="checkbox" class="checkbox-macos"
                      v-model="prefStore.logger.enable_file" @change="prefStore.setLoggerConfig()" /><span>{{
                        $t('settings.general.log_file') }}</span></label>
                </div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_max_backups') }}</div>
              <div class="v control-short">
                <div class="measure-row"><input type="number" class="input-macos measure-input text-center" min="0"
                    max="100" v-model="prefStore.logger.max_backups" @change="prefStore.setLoggerConfig()" /><span
                    class="measure-unit text-sm text-secondary">{{ $t('settings.general.files') }}</span></div>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.log_compress') }}</div>
              <div class="v">
                <label class="macos-check"><input type="checkbox" class="checkbox-macos"
                    v-model="prefStore.logger.compress" @change="prefStore.setLoggerConfig()" /><span>{{
                      $t('settings.general.log_compress') }}</span></label>
              </div>
            </div>
            <template v-if="prefStore.logger.enable_file">
              <div class="macos-row">
                <div class="k">{{ $t('settings.general.log_dir') }}</div>
                <div class="v">
                  <span class="result-field mr-2 text-secondary"
                    :class="{ 'text-tertiary': !prefStore.logger.directory }"
                    :title="prefStore.logger.directory">{{ prefStore.logger.directory }}</span>
                <button class="icon-glass" @click="openDirectory(prefStore.logger.directory)" :data-tooltip="t('download.open_folder')" :aria-label="t('download.open_folder')"><Icon
                      name="folder" class="w-4 h-4" /></button>
                <button class="icon-glass" @click="onSelectLoggerDir" :data-tooltip="t('common.change') || 'Change'" :aria-label="t('common.change') || 'Change'"><Icon name="settings" class="w-4 h-4" /></button>
                </div>
              </div>
            </template>
          </template>

          <!-- Paths -->
          <template v-else-if="current === 'paths'">
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.config_path') }}</div>
              <div class="v">
                <span class="result-field mr-2 text-secondary">{{ prefPath }}</span>
                <button class="icon-glass" @click="openDirectory(prefPath)" :data-tooltip="t('download.open_folder')" :aria-label="t('download.open_folder')"><Icon
                    name="folder" class="w-4 h-4" /></button>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">{{ $t('settings.general.data_path') }}</div>
              <div class="v">
                <span class="result-field mr-2 text-secondary">{{ taskDbPath }}</span>
                <button class="icon-glass" @click="openDirectory(taskDbPath)" :data-tooltip="t('download.open_folder')" :aria-label="t('download.open_folder')"><Icon
                    name="folder" class="w-4 h-4" /></button>
              </div>
            </div>
          </template>

          <!-- Listened -->
          <template v-else-if="current === 'listened'">
            <div class="macos-row">
              <div class="k">WebSocket</div>
              <div class="v">
                <span class="result-field mr-2 text-secondary" :title="wsListendAddress">{{ wsListendAddress }}</span>
                <button class="icon-glass" @click="copyText(wsListendAddress)" :data-tooltip="$t('common.copy')" :aria-label="$t('common.copy')">
                  <Icon name="file-copy" class="w-4 h-4" />
                </button>
              </div>
            </div>
            <div class="macos-row">
              <div class="k">MCP Server</div>
              <div class="v">
                <span class="result-field mr-2 text-secondary" :title="mcpListendAddress">{{ mcpListendAddress }}</span>
                <button class="icon-glass" @click="copyText(mcpListendAddress)" :data-tooltip="$t('common.copy')" :aria-label="$t('common.copy')">
                  <Icon name="file-copy" class="w-4 h-4" />
                </button>
              </div>
            </div>
          </template>

          

          <!-- Acknowledgments (single item, no inner title/divider) -->
          <template v-else-if="current === 'ack'">
            <div class="macos-row">
              <div class="k">ZHConvert</div>
              <div class="v">
                <span class="text-secondary mr-2">{{ $t('dialogue.about.zhconvert_desc') }}</span>
                <button class="btn-glass" @click="openUrl(Project.ZHConvert)" :data-tooltip="$t('dialogue.about.website')" :aria-label="$t('dialogue.about.website')">
                  <Icon name="globe" class="w-4 h-4 mr-1"/> {{ $t('dialogue.about.website') }}
                </button>
              </div>
            </div>
          </template>
        </div>
      </div>
      </template>
    </section>
  </div>

</template>

<script setup>
import { computed, ref, watch, onMounted } from 'vue'
import useLayoutStore from '@/stores/layout.js'
import usePreferencesStore from '@/stores/preferences.js'
import useSettingsStore from '@/stores/settings.js'
import { applyAccent, applyUIStyle } from '@/utils/theme.js'
import { useI18n } from 'vue-i18n'
import { OpenDirectoryDialog, OpenDirectory } from 'wailsjs/go/systems/Service'
import { GetPreferencesPath, GetTaskDbPath } from 'wailsjs/go/api/PathsAPI'
import { copyText as copyToClipboard } from '@/utils/clipboard.js'

const layout = useLayoutStore()
const prefStore = usePreferencesStore()
const settingsStore = useSettingsStore()
const { t, locale } = useI18n()

const leftWidth = computed(() => layout.ribbonWidth + 'px')

const groups = computed(() => {
  // explicitly depend on locale for reactivity
  const _ = locale.value
  return [
    { key: 'appearance', label: t('settings.general.name'), icon: 'settings' },
    { key: 'download', label: t('settings.general.download'), icon: 'download-cloud' },
    { key: 'logs', label: t('settings.general.log'), icon: 'file-text' },
    { key: 'paths', label: t('settings.general.saved_path'), icon: 'folder' },
    { key: 'listened', label: t('settings.general.listend'), icon: 'link' },
    { key: 'ack', label: t('settings.acknowledgments'), icon: 'heart' },
    { key: 'about', label: t('settings.about.title'), icon: 'info' },
  ]
})

const current = ref('appearance')
onMounted(() => {
  if (settingsStore?.currentPage === 'about') current.value = 'about'
})
const title = computed(() => groups.value.find(g => g.key === current.value)?.label || '')

const onLanguageChange = async () => { await prefStore.savePreferences(); locale.value = prefStore.currentLanguage }

const proxyAddress = ref('')
const isValidProxyAddress = (address) => /^(http|https|socks5):\/\/[a-zA-Z0-9.-]+:[0-9]+$/.test(address)
watch(() => prefStore.proxy, (p) => { proxyAddress.value = p?.proxy_address || '' }, { immediate: true, deep: true })
const onProxyChange = async () => {
  if (prefStore.proxy.type === 'manual') {
    if (!proxyAddress.value.trim() || !isValidProxyAddress(proxyAddress.value)) { return $message?.error?.(t('settings.general.proxy_address_err')) }
    prefStore.proxy.proxy_address = proxyAddress.value
  }
  await prefStore.setProxyConfig()
}

// Accent theme handling
const accentOptions = [
  { value: 'blue', label: 'Blue', color: '#007AFF' },
  { value: 'purple', label: 'Purple', color: '#8E44AD' },
  { value: 'pink', label: 'Pink', color: '#FF2D55' },
  { value: 'red', label: 'Red', color: '#FF3B30' },
  { value: 'orange', label: 'Orange', color: '#FF9500' },
  { value: 'green', label: 'Green', color: '#34C759' },
  { value: 'teal', label: 'Teal', color: '#30B0C7' },
  { value: 'indigo', label: 'Indigo', color: '#5856D6' },
]
const onPickAccent = async (name) => {
  if (prefStore.general.theme === name) return
  prefStore.general.theme = name
  applyAccent(name, prefStore.isDark)
  await prefStore.savePreferences()
}

const customAccent = ref('#007AFF')
watch(() => prefStore.general.theme, (v) => { if (typeof v === 'string' && v.startsWith('#')) customAccent.value = v }, { immediate: true })
const onPickCustom = async (e) => {
  const val = (typeof e === 'string') ? e : e?.target?.value
  if (!val) return
  prefStore.general.theme = val
  applyAccent(val, prefStore.isDark)
  await prefStore.savePreferences()
}

const onSelectDownloadDir = async () => {
  const { success, data, msg } = await OpenDirectoryDialog(prefStore.download?.dir || '')
  if (success && data?.path?.trim()) { prefStore.download.dir = data.path; await prefStore.SetDownloadConfig() } else if (msg) { $message?.error?.(msg) }
}

const onSelectLoggerDir = async () => {
  const seed = prefStore.logger?.directory || ''
  const { success, data, msg } = await OpenDirectoryDialog(seed)
  if (success && data?.path?.trim()) {
    prefStore.logger.directory = data.path
    await prefStore.setLoggerConfig()
  } else if (msg) {
    $message?.error?.(msg)
  }
}

// About actions
import iconUrl from '@/assets/images/icon.png'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { GetAppVersion, CheckForUpdate } from 'wailsjs/go/preferences/Service.js'
import { Project } from '@/consts/global.js'

const appVersion = ref('')
onMounted(async () => {
  try { const r = await GetAppVersion(); if (r?.data?.version) appVersion.value = r.data.version } catch {}
})

const openUrl = (url) => { try { BrowserOpenURL(url) } catch { window.open(url, '_blank') } }
const openWebsite = () => openUrl(Project.OfficialWebsite)
const openGithub = () => openUrl(Project.Github)
const openTwitter = () => openUrl(Project.Twitter)

const openChangelog = () => openUrl(Project.Github + '/releases')
const checkUpdates = async () => { try { await prefStore.checkForUpdate(true); } catch (e) {} }
const onAboutToggleChange = async () => { await prefStore.savePreferences() }
const onTelemetryToggleChange = async () => { await prefStore.savePreferences() }

const prefPath = ref('')
const taskDbPath = ref('')
onMounted(async () => {
  try { const r = await GetPreferencesPath(); if (r.success) prefPath.value = r.data } catch { }
  try { const r = await GetTaskDbPath(); if (r.success) taskDbPath.value = r.data } catch { }
})
const openDirectory = (p) => OpenDirectory(p)

const wsListendAddress = computed(() => { const info = prefStore.listendInfo?.ws; return info ? `${info.protocol}://${info.ip}:${info.port}/${info.path}` : '' })
const mcpListendAddress = computed(() => { const info = prefStore.listendInfo?.mcp; return info ? `${info.protocol}://${info.ip}:${info.port}/${info.path}` : '' })

const copyText = async (text) => { await copyToClipboard(text, t) }

// Debug helpers removed
</script>

<style scoped>
/* token-based text utilities for this view */
.text-secondary { color: var(--macos-text-secondary); }
.text-tertiary { color: var(--macos-text-tertiary); }
/* Fill parent absolutely; parent (#app-content .settings-host) is positioned relative */
.sr-root {
  position: absolute;
  inset: 0;
  display: grid;
  grid-template-columns: 160px 1fr;
}

.sr-left {
  position: relative;
  z-index: 1;
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 6px;
  justify-content: flex-start;
}

.sr-right {
  position: relative;
  z-index: 1;
  background: var(--macos-background);
  padding: 12px;
  overflow: auto;
}

/* divider/background moved to settings-host (::before/::after) */

/* left menu row mimics Ribbon */
.sr-item {
  height: 28px;
  display: flex;
  align-items: center;
  border-radius: 8px;
  padding: 0 10px 0 12px;
  color: var(--macos-text-secondary);
  cursor: pointer;
  transition: background 120ms ease, color 120ms ease;
}

.sr-item:hover {
  background: var(--macos-gray-hover);
}

.sr-item.active {
  background: var(--macos-gray-hover);
  color: var(--macos-blue);
  font-weight: 400;
}

.sr-item.active:hover {
  background: var(--macos-gray-hover);
}

.sr-item-icon {
  width: 18px;
  height: 18px;
  margin-right: 8px;
  color: var(--macos-text-secondary);
}

.sr-item.active .sr-item-icon {
  color: var(--macos-blue);
}

.sr-item-label {
  font-size: var(--fs-base);
  line-height: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* right content card: only wraps content, background fills column */
.sr-right {
  font-size: var(--fs-base);
}

.sr-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 2px 2px 6px 2px;
}

.sr-section-title {
  font-size: var(--fs-base);
  color: var(--macos-text-secondary);
  letter-spacing: .3px;
  font-weight: 600;
}

.sr-section-divider {
  height: 1px;
  background: var(--macos-divider-weak);
  margin: 0 -12px 10px -12px;
}

.sr-card {
  border-radius: 10px;
  padding: 12px;
  max-width: 100%;
  box-sizing: border-box;
}

/* theme-aware subtle surface for card: slightly gray */
/* background/border handled by card-frosted/card-translucent or classic override */

/* option rows */
.sr-row {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 8px 6px;
  min-height: 36px;
}

.sr-label {
  text-align: left;
  color: var(--macos-text-primary);
}

.sr-control {
  justify-self: end;
  display: inline-flex;
  align-items: center;
}

.control-short {
  width: 160px;
  justify-content: flex-end;
}

/* segmented and inputs */
/* use global .segmented/.seg-item */

.select-macos {
  width: min(260px, 100%);
  height: 26px;
  border-radius: 6px;
  border: 1px solid var(--macos-separator);
  background: var(--macos-background);
  color: var(--macos-text-primary);
  padding: 0 28px 0 10px;
  appearance: none;
  -webkit-appearance: none;
  background-image: url('data:image/svg+xml;utf8,<svg xmlns=\"http://www.w3.org/2000/svg\" width=\"12\" height=\"12\" viewBox=\"0 0 24 24\" fill=\"none\" stroke=\"%23a1a1a1\" stroke-width=\"2\" stroke-linecap=\"round\" stroke-linejoin=\"round\"><polyline points=\"6 9 12 15 18 9\"/></svg>');
  background-repeat: no-repeat;
  background-position: right 8px center;
  background-size: 12px 12px;
}

.select-macos-xs {
  width: 160px;
}

.select-macos:focus {
  outline: none;
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--macos-blue) 25%, transparent);
  border-color: var(--macos-blue);
}

.result-field {
  display: inline-block;
  width: 260px;
  max-width: 260px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  text-align: right;
  font-family: var(--font-mono);
  font-size: var(--fs-sub);
}

.toggle-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.macos-check {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--macos-text-primary);
}

.checkbox-macos {
  width: 14px;
  height: 14px;
  accent-color: var(--macos-blue);
}

.nowrap {
  white-space: nowrap;
}

.measure-row {
  width: 160px;
  height: 26px;
  display: grid;
  grid-template-columns: 110px 1fr;
  align-items: center;
  justify-items: end;
  gap: 6px;
  white-space: nowrap;
}

.measure-row .measure-input.input-macos {
  height: 26px;
  line-height: 24px;
  padding: 0 6px;
  width: 110px;
}

.measure-unit {
  width: 40px;
  text-align: left;
}

.toggle-row-sm {
  justify-content: flex-end;
}

/* 原生 macOS 风格图标按钮；避免使用外部 UI 库的类名 */
.sr-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--macos-text-secondary);
}

.sr-icon-btn:hover {
  background: var(--macos-gray-hover);
}

.sr-icon-btn:active {
  background: var(--macos-gray-active);
}

/* remove local mid divider; rely on host (#app-content .with-split) */

/* inner-row dividers inside card (not touching edges) */
.sr-card-body {
  padding-top: 0;
}

.sr-card-body .sr-row {
  border-bottom: 1px solid var(--macos-divider-weak);
  margin: 0 8px;
}

.sr-card-body .sr-row:last-child {
  border-bottom: none;
}

/* Standardized macOS rows in Settings: label left, control right */
.sr-card :deep(.macos-row) {
  display: grid;
  grid-template-columns: 1fr auto;
  align-items: center;
  gap: 12px;
  padding: 8px 6px;
  min-height: 36px;
}
.sr-card :deep(.macos-row) .k { text-align: left; color: var(--macos-text-primary); }
.sr-card :deep(.macos-row) .v { justify-self: end; display: inline-flex; align-items: center; }

/* make grid columns follow Ribbon width */
.sr-root {
  grid-template-columns: var(--left-col, 160px) 1fr;
}

:host,
.sr-root {
  --left-col: 160px;
}

/* About section styles */
.about-wrap { display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 32px 8px; gap: 14px; }
.about-icon { width: 64px; height: 64px; border-radius: 14px; box-shadow: var(--macos-shadow-1); }
.about-title { font-size: 18px; font-weight: 700; color: var(--macos-text-primary); }
.about-title .ver { font-weight: 400; color: var(--macos-text-secondary); margin-left: 6px; }
.about-links { display: flex; align-items: center; gap: 8px; color: var(--macos-text-secondary); }
.about-links .link-btn { color: var(--macos-text-secondary); cursor: pointer; padding: 0 2px; }
.about-links .link-btn:hover { text-decoration: underline; }
.about-links .link-text { color: var(--macos-text-secondary); text-decoration: none; }
.about-links .link-text:hover { text-decoration: underline; }
.about-links .dot { color: var(--macos-text-tertiary); }
.about-social { display: flex; align-items: center; gap: 12px; margin-top: 6px; }
.about-social .icon-btn { @extend .sr-icon-btn; width: 28px; height: 28px; }
.about-actions { display: flex; gap: 10px; margin-top: 10px; }
.about-options { display: flex; flex-direction: column; gap: 6px; width: min(360px, 100%); margin-top: 6px; }
.about-option { display: grid; grid-template-columns: 1fr auto; align-items: center; gap: 12px; padding: 8px 12px; border-radius: 12px; background: var(--macos-surface); backdrop-filter: var(--macos-surface-blur); border: 1px solid var(--macos-divider-weak); width: 100%; box-shadow: var(--macos-shadow-1); min-height: 32px; }
.about-option-title { font-weight: 600; color: var(--macos-text-primary); font-size: 13px; }
.about-toggle { position: relative; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; }
.about-toggle-input { position: absolute; width: 0; height: 0; opacity: 0; }
.about-toggle-slider { width: 32px; height: 18px; border-radius: 999px; background: var(--macos-divider-weak); box-shadow: inset 0 0 0 1px var(--macos-divider-weak); transition: background 180ms ease, box-shadow 180ms ease; display: inline-block; position: relative; }
.about-toggle-slider::after { content: ""; position: absolute; width: 14px; height: 14px; border-radius: 50%; background: var(--macos-background); top: 2px; left: 2px; box-shadow: var(--macos-shadow-1); transition: transform 180ms ease; }
.about-toggle-input:checked + .about-toggle-slider { background: color-mix(in srgb, var(--macos-blue) 55%, transparent); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--macos-blue) 70%, transparent); }
.about-toggle-input:checked + .about-toggle-slider::after { transform: translateX(14px); background: var(--macos-background); box-shadow: 0 0 0 1px color-mix(in srgb, var(--macos-blue) 65%, transparent), 0 2px 4px rgba(0,0,0,0.12); }
.about-toggle-input:focus-visible + .about-toggle-slider { outline: 2px solid rgba(var(--macos-blue-rgb), 0.7); outline-offset: 2px; }
.contact-email { display: inline-flex; align-items: center; gap: 6px; }

/* Accent picker */
.accent-grid { display:flex; flex-wrap:wrap; gap:10px; }
.accent-btn { width: 28px; height: 28px; border-radius: 999px; border: 2px solid var(--macos-separator); background: color-mix(in oklab, var(--accent-color) 16%, transparent); display:inline-flex; align-items:center; justify-content:center; transition: transform .12s ease, box-shadow .12s ease, border-color .12s ease; }
.accent-btn:hover { transform: translateY(-1px); box-shadow: var(--macos-shadow-2); border-color: var(--accent-color); }
.accent-btn .dot { width: 14px; height: 14px; border-radius: 999px; background: var(--accent-color); display:block; }
.accent-btn.active { border-color: var(--accent-color); box-shadow: 0 0 0 3px color-mix(in oklab, var(--accent-color) 20%, transparent); }

.accent-custom { display:inline-flex; align-items:center; justify-content:center; width: 28px; height: 28px; border-radius: 999px; border: 2px dashed var(--macos-separator); overflow: hidden; }
.accent-custom:hover { border-color: var(--macos-blue); }
.accent-color { appearance: none; -webkit-appearance: none; border: none; padding: 0; width: 28px; height: 28px; cursor: pointer; background: transparent; }
.accent-color::-webkit-color-swatch-wrapper { padding: 0; }
.accent-color::-webkit-color-swatch { border: none; border-radius: 999px; }
</style>
