import { defineStore } from 'pinia'
import { lang } from '@/langs/index.js'
import { cloneDeep, get, isEmpty, pick, set, split } from 'lodash'
import {
    CheckForUpdate,
    GetAppVersion,
    GetPreferences,
    RestorePreferences,
    SetPreferences,
    SetProxyConfig,
    SetDownloadConfig,
    SetLoggerConfig,
} from 'bindings/dreamcreator/backend/services/preferences/service'
import { Info as GetSystemInfo } from 'bindings/dreamcreator/backend/services/systems/service'
import { Browser } from '@wailsio/runtime'
import { i18nGlobal } from '@/utils/i18n.js'
import { h, nextTick, ref } from 'vue'
import { compareVersion } from '@/utils/version.js'
import { startTelemetry, stopTelemetry, sendTelemetry } from '@/utils/telemetry.js'

// 使用原生方法检测系统主题
const systemDarkMode = ref(window.matchMedia('(prefers-color-scheme: dark)').matches)

// 监听系统主题变化
if (typeof window !== 'undefined') {
  const darkModeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  const updateSystemTheme = (e) => {
    systemDarkMode.value = e.matches
  }
  darkModeMediaQuery.addEventListener('change', updateSystemTheme)
}

const usePreferencesStore = defineStore('preferences', {
    /**
     * @typedef {Object} FontItem
     * @property {string} name
     * @property {string} path
     */
    /**
     * @typedef {Object} Preferences
     * @property {Object} general
     * @property {Object} editor
     * @property {FontItem[]} fontList
     */
    /**
     *
     * @returns {Preferences}
     */
    state: () => ({
        behavior: {
            welcomed: false,
            asideWidth: 300,
            windowWidth: 0,
            windowHeight: 0,
            windowMaximised: false,
        },
        general: {
            appearance: 'auto', // light/dark/auto
            theme: 'blue', // accent color
            language: 'auto',
            checkUpdate: true,
            skipVersion: '',
        },
        telemetry: {
            enabled: true,
            appId: '',
            clientId: '',
            endpoint: '',
            version: '',
        },
        proxy: {
            type: 'none',
            proxy_address: '',
            no_proxy: [],
            username: '',
            password: '',
        },
        download: {
            dir: '',
        },
        buildInDecoder: [],
        decoder: [],
        lastPref: {},
        logger: {}, 
        telemetryRuntime: {
            active: false,
            bootTracked: false,
            bootKey: '',
            meta: {
                version: '',
                os: '',
                arch: '',
            },
        },
    }),
    getters: {
        getSeparator() {
            return ':'
        },

        appearanceOption() {
            return [
                {
                    value: 'light',
                    label: 'settings.general.appearance_light',
                },
                {
                    value: 'dark',
                    label: 'settings.general.appearance_dark',
                },
                {
                    value: 'auto',
                    label: 'settings.general.appearance_auto',
                },
            ]
        },

        /**
         * all available language
         * @returns {{label: string, value: string}[]}
         */
        langOption() {
            const options = Object.entries(lang).map(([key, value]) => ({
                value: key,
                label: value['name'],
            }))
            options.splice(0, 0, {
                value: 'auto',
                label: 'Auto',
            })
            return options
        },

        /**
         * get current language setting
         * @return {string}
         */
        currentLanguage() {
            let lang = get(this.general, 'language', 'auto')
            if (lang === 'auto') {
                const systemLang = navigator.language || navigator.userLanguage
                lang = split(systemLang, '-')[0]
            }
            return lang || 'en'
        },

        isDark() {
            const theme = get(this.general, 'appearance', 'auto')
            if (theme === 'auto') {
                return systemDarkMode.value
            }
            return theme === 'dark'
        },

        autoCheckUpdate() {
            return get(this.general, 'checkUpdate', false)
        },

    },
    actions: {
        _applyPreferences(data) {
            for (const key in data) {
                set(this, key, data[key])
            }
        },

        /**
         * load preferences from local
         * @returns {Promise<void>}
         */
        async loadPreferences() {
            const { success, data } = await GetPreferences()
            if (success) {
                this.lastPref = cloneDeep(data)
                // migrate: old general.theme (auto/light/dark) -> general.appearance
                const migrated = cloneDeep(data)
                try {
                    const g = migrated.general || {}
                    if (g && typeof g.theme === 'string' && ['auto','light','dark'].includes(g.theme)) {
                        // move to appearance
                        g.appearance = g.appearance || g.theme
                        // set a default accent theme if none present
                        g.theme = g.theme && ['auto','light','dark'].includes(g.theme) ? 'blue' : (g.theme || 'blue')
                    } else {
                        // ensure defaults exist
                        if (!g.appearance) g.appearance = 'auto'
                        if (!g.theme) g.theme = 'blue'
                    }
                    // Only keep supported general fields.
                    migrated.general = pick(g, ['appearance', 'theme', 'language', 'checkUpdate', 'skipVersion'])
                } catch (e) {}
                this._applyPreferences(migrated)
                const proxy = get(data, 'proxy')
                if (proxy === undefined) {
                    set(data, 'proxy', {
                        type: 'none',
                        proxy_address: '',
                        no_proxy: [],
                        username: '',
                        password: '',
                    })
                }
                const telemetry = get(migrated, 'telemetry')
                if (!telemetry) {
                    set(migrated, 'telemetry', { enabled: true, clientId: '', appId: '', endpoint: '', version: '' })
                } else {
                    if (!('endpoint' in telemetry) || telemetry.endpoint === undefined || telemetry.endpoint === null) {
                        telemetry.endpoint = ''
                    }
                    if (!('version' in telemetry) || telemetry.version === undefined || telemetry.version === null) {
                        telemetry.version = ''
                    }
                }
                i18nGlobal.locale.value = this.currentLanguage
                await this.refreshTelemetryRuntime({ emitToggleEvents: false })
            }
        },

        /**
         * save preferences to local
         * @returns {Promise<boolean>}
         */
        async savePreferences() {
            const pf = pick(this, ['behavior', 'general', 'proxy', 'download', 'logger', 'telemetry']) 
            if (pf?.general) pf.general = pick(pf.general, ['appearance', 'theme', 'language', 'checkUpdate', 'skipVersion'])
            if (pf.telemetry) {
                pf.telemetry = { ...pf.telemetry }
                delete pf.telemetry.appId
                delete pf.telemetry.endpoint
                delete pf.telemetry.version
            }
            const { success, msg } = await SetPreferences(pf)
            if (success) {
                await this.refreshTelemetryRuntime()
            }
            return success === true
        },
        /**
         * reset to last-loaded preferences
         * @returns {Promise<void>}
         */
        async resetToLastPreferences() {
            if (!isEmpty(this.lastPref)) {
                this._applyPreferences(this.lastPref)
            }
        },

        async setProxyConfig() {
            try {
                const config = {
                    type: this.proxy.type,
                    proxy_address: this.proxy.proxy_address || ""
                };
                
                // 验证手动代理设置
                if (config.type === 'manual') {
                    if (!config.proxy_address) {
                        $message.error(i18nGlobal.t('settings.general.proxy_required'))
                        return false
                    }
                }

                const { success, msg } = await SetProxyConfig(config)
                if (!success) {
                    $message.error(i18nGlobal.t('settings.general.proxy_set_failed') + (msg ? `: ${msg}` : ''))
                    return false
                }
                
                if (config.type === 'none') {
                    $message.success(i18nGlobal.t('settings.general.proxy_disable_success'))
                } else {
                    $message.success(i18nGlobal.t('settings.general.proxy_set_success'))
                }
                return true
            } catch (error) {
                console.error('Failed to set proxy:', error)
                $message.error(i18nGlobal.t('settings.general.proxy_set_failed'))
                return false
            }
        },

        async SetDownloadConfig() {
            try {
                const config = {
                    dir: this.download.dir || ""
                };
                
                // 验证下载目录设置
                if (!config.dir) {
                    $message.error(i18nGlobal.t('settings.general.download_dir_required'))
                    return false
                }

                const { success, msg } = await SetDownloadConfig(config)
                if (!success) {
                    $message.error(i18nGlobal.t('settings.general.download_dir_set_failed') + (msg ? `: ${msg}` : ''))
                    return false
                }
                
                $message.success(i18nGlobal.t('settings.general.download_dir_set_success'))
                return true
            } catch (error) {
                console.error('Failed to set download directory:', error)
                $message.error(i18nGlobal.t('settings.general.download_dir_set_failed'))
                return false
            }
        },

        /**
         * 更新日志配置
         * @param {Object} config 日志配置
         * @returns {Promise<boolean>}
         */
        async setLoggerConfig() {
            try {
                // 保存到后端
                const { success, msg } = await SetLoggerConfig(this.logger)
                if (!success) {
                    $message.error(msg || 'Save logger config failed')
                    return false
                }
                return true
            } catch (error) {
                $message.error('Save logger config failed')
                return false
            }
        },

        /**
         * restore preferences to default
         * @returns {Promise<boolean>}
         */
        async restorePreferences() {
            const { success, data } = await RestorePreferences()
            if (success === true) {
                const { pref } = data
                this._applyPreferences(pref)
                await this.refreshTelemetryRuntime()
                return true
            }
            return false
        },

        setAsWelcomed(acceptTrack) {
            this.behavior.welcomed = true
            this.savePreferences()
        },

        async refreshTelemetryRuntime(options = {}) {
            const { emitToggleEvents = true } = options
            const appId = (get(this.telemetry, 'appId', '') || '').trim()
            const clientId = (get(this.telemetry, 'clientId', '') || '').trim()
            const endpoint = (get(this.telemetry, 'endpoint', '') || '').trim()
            const enabled = !!get(this.telemetry, 'enabled', false)
            const shouldActive = Boolean(enabled && appId && clientId && appId !== 'undefined' && appId !== 'null')
            const wasActive = this.telemetryRuntime.active

            if (shouldActive) {
                const meta = await this.ensureTelemetryMeta()
                const startOptions = endpoint ? { endpoint } : {}
                const resolvedVersion = (meta.version || 'dev').toLowerCase()
                if (resolvedVersion === 'dev') {
                    startOptions.testMode = true
                }
                const key = `${appId}:${clientId}`
                const instance = startTelemetry(appId, clientId, startOptions)
                this.telemetryRuntime.active = !!instance

                if (this.telemetryRuntime.active && emitToggleEvents && !wasActive) {
                    const { payload, version } = await this.buildTelemetryPayload({}, meta)
                    await sendTelemetry('telemetry_opt_in', payload, { version })
                }

                const alreadyTracked = this.telemetryRuntime.bootTracked && this.telemetryRuntime.bootKey === key
                if (this.telemetryRuntime.active && !alreadyTracked) {
                    const { payload, version } = await this.buildTelemetryPayload({}, meta)
                    await sendTelemetry('app_start', payload, { version })
                    this.telemetryRuntime.bootTracked = true
                    this.telemetryRuntime.bootKey = key
                }
            } else {
                if (emitToggleEvents && wasActive) {
                    const meta = await this.ensureTelemetryMeta()
                    const { payload, version } = await this.buildTelemetryPayload({}, meta)
                    await sendTelemetry('telemetry_opt_out', payload, { version })
                }
                stopTelemetry()
                this.telemetryRuntime.active = false
            }
        },

        async ensureTelemetryMeta() {
            const meta = this.telemetryRuntime.meta || {}
            if (!meta.version) {
                const telemetryVersion = (get(this.telemetry, 'version', '') || '').trim()
                if (telemetryVersion) {
                    meta.version = telemetryVersion
                }
            }
            if (!meta.version) {
                try {
                    const res = await GetAppVersion()
                    if (res?.success && res.data?.version) {
                        meta.version = String(res.data.version)
                    }
                } catch (e) {}
            }
            if (!meta.os || !meta.arch) {
                try {
                    const res = await GetSystemInfo()
                    if (res?.success && res.data) {
                        meta.os = String(res.data.os || '').toLowerCase()
                        meta.arch = String(res.data.arch || '').toLowerCase()
                    }
                } catch (e) {}
            }
            this.telemetryRuntime.meta = meta
            return meta
        },

        async buildTelemetryPayload(extra = {}, metaOverride = null) {
            const meta = metaOverride || (await this.ensureTelemetryMeta()) || {}
            const resolvedVersion = meta.version || 'dev'
            const payload = {
                appVersion: resolvedVersion,
                os: meta.os || '',
                arch: meta.arch || '',
                ...extra,
            }

            const normalizedPayload = {}
            Object.entries(payload).forEach(([key, rawValue]) => {
                if (rawValue === undefined || rawValue === null) {
                    return
                }

                normalizedPayload[key] = typeof rawValue === 'string' ? rawValue : String(rawValue)
            })

            return { payload: normalizedPayload, version: resolvedVersion }
        },

        async checkForUpdate(manual = false) {
            let msgRef = null
            if (manual) {
                // show a lightweight persistent notification while checking
                msgRef = $notification.info({ title: i18nGlobal.t('menu.check_update'), content: i18nGlobal.t('common.loading'), duration: 0, closable: true })
            }
            try {
                const { success, data = {} } = await CheckForUpdate()
                if (success) {
                    const { version = 'v1.0.0', latest, page_url: pageUrl } = data
                    if (
                        (manual || latest > this.general.skipVersion) &&
                        compareVersion(latest, version) > 0 &&
                        !isEmpty(pageUrl)
                    ) {
                        // show sticky update card with actions
                        const notiRef = $notification.show({
                            title: i18nGlobal.t('dialogue.upgrade.title'),
                            content: i18nGlobal.t('dialogue.upgrade.new_version_tip', { ver: latest }),
                            duration: 0,
                            closable: true,
                            action: (destroy) => {
                                // 使用自定义 macOS 风格组件渲染操作按钮
                                return h('div', { class: 'flex flex-row gap-2 mt-2' }, [
                                    h('button', {
                                        class: 'btn-chip-ghost btn-sm',
                                        onClick: () => {
                                            // skip this update
                                            this.general.skipVersion = latest
                                            this.savePreferences()
                                            destroy()
                                        }
                                    }, i18nGlobal.t('dialogue.upgrade.skip')),
                                    h('button', {
                                        class: 'btn-chip-ghost btn-sm',
                                        onClick: destroy
                                    }, i18nGlobal.t('dialogue.upgrade.later')),
                                    h('button', {
                                        class: 'btn-chip-ghost btn-primary btn-sm',
                                        onClick: () => {
                                            try {
                                                Browser.OpenURL(pageUrl)
                                            } catch {
                                                try { window.open(pageUrl, '_blank') } catch {}
                                            }
                                        }
                                    }, i18nGlobal.t('dialogue.upgrade.download_now'))
                                ])
                            }
                        })
                        return
                    }
                }
                if (manual) { $notification.info({ title: i18nGlobal.t('menu.check_update'), content: i18nGlobal.t('dialogue.upgrade.no_update'), duration: 3200 }) }
            } finally {
                nextTick().then(() => {
                    if (msgRef != null) {
                        msgRef.close && msgRef.close()
                        msgRef = null
                    }
                })
            }
        },
    },
})

export default usePreferencesStore
