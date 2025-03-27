import { defineStore } from 'pinia'
import { lang } from '@/langs/index.js'
import { cloneDeep, get, isEmpty, pick, set, split } from 'lodash'
import {
    CheckForUpdate,
    GetPreferences,
    RestorePreferences,
    SetPreferences,
    SetProxyConfig,
    SetDownloadConfig,
    SetLoggerConfig,
} from 'wailsjs/go/preferences/Service.js'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { i18nGlobal } from '@/utils/i18n.js'
import { h, nextTick, ref } from 'vue'
import { compareVersion } from '@/utils/version.js'

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
            theme: 'auto',
            language: 'auto',
            checkUpdate: true,
            skipVersion: '',
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
        dependencies: {
            ytdlp: {
                // additional properties, only enabled in frontend
                installing: false,
                installProgress: '',
                installed: false,
                updating: false,
                updateProgress: '',
                updated: false,
                // properties from backend
                path: '',
                execPath: '',
                version: '',
                latestVersion: '',
                needUpdate: false,
            },
            ffmpeg: {
                installed: false,
                path: '',
                execPath: '',
                version: '',
                latestVersion: '',
                needUpdate: false,
            }
        }
    }),
    getters: {
        getSeparator() {
            return ':'
        },

        themeOption() {
            return [
                {
                    value: 'light',
                    label: 'preferences.general.theme_light',
                },
                {
                    value: 'dark',
                    label: 'preferences.general.theme_dark',
                },
                {
                    value: 'auto',
                    label: 'preferences.general.theme_auto',
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
            const theme = get(this.general, 'theme', 'auto')
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
                this._applyPreferences(data)
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
                i18nGlobal.locale.value = this.currentLanguage
            }
        },

        /**
         * save preferences to local
         * @returns {Promise<boolean>}
         */
        async savePreferences() {
            const pf = pick(this, ['behavior', 'general', 'proxy', 'download', 'logger']) 
            const { success, msg } = await SetPreferences(pf)
            // proxy 
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
        async setLoggerConfig(config) {
            try {
                // 更新 store 中的配置
                this.logger = {
                    ...this.logger,
                    ...config
                }

                // 保存到后端
                const { success, msg } = await SetLoggerConfig(config)
                if (!success) {
                    $message.error(msg || '保存日志配置失败')
                    return false
                }
                return true
            } catch (error) {
                console.error('保存日志配置失败:', error)
                $message.error('保存日志配置失败')
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
                return true
            }
            return false
        },

        setAsWelcomed(acceptTrack) {
            this.behavior.welcomed = true
            this.savePreferences()
        },

        async checkForUpdate(manual = false) {
            let msgRef = null
            if (manual) {
                msgRef = $message.loading(i18nGlobal.t('menu.check_update'), { duration: 0 })
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
                        const notiRef = $notification.show({
                            title: i18nGlobal.t('dialogue.upgrade.title'),
                            content: i18nGlobal.t('dialogue.upgrade.new_version_tip', { ver: latest }),
                            action: (destroy) => {
                                // 使用 DaisyUI 组件代替 Naive UI 组件
                                return h('div', { class: 'flex flex-row gap-2 mt-2' }, [
                                    h('button', {
                                        class: 'btn btn-sm btn-outline',
                                        onClick: () => {
                                            // skip this update
                                            this.general.skipVersion = latest
                                            this.savePreferences()
                                            destroy()
                                        }
                                    }, i18nGlobal.t('dialogue.upgrade.skip')),
                                    h('button', {
                                        class: 'btn btn-sm btn-outline',
                                        onClick: destroy
                                    }, i18nGlobal.t('dialogue.upgrade.later')),
                                    h('button', {
                                        class: 'btn btn-sm btn-primary',
                                        onClick: () => BrowserOpenURL(pageUrl)
                                    }, i18nGlobal.t('dialogue.upgrade.download_now'))
                                ])
                            }
                        })
                        return
                    }
                }

                if (manual) {
                    $message.info(i18nGlobal.t('dialogue.upgrade.no_update'))
                }
            } finally {
                nextTick().then(() => {
                    if (msgRef != null) {
                        msgRef.close()
                        msgRef = null
                    }
                })
            }
        },
    },
})

export default usePreferencesStore
