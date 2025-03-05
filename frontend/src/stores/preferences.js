import { defineStore } from 'pinia'
import { lang } from '@/langs/index.js'
import { cloneDeep, findIndex, get, isEmpty, join, map, merge, pick, set, some, split } from 'lodash'
import {
    CheckForUpdate,
    GetFontList,
    GetPreferences,
    RestorePreferences,
    SetPreferences,
    SetProxy,
} from 'wailsjs/go/preferences/Service.js'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { i18nGlobal } from '@/utils/i18n.js'
import { enUS, NButton, NSpace, useOsTheme, zhCN } from 'naive-ui'
import { h, nextTick } from 'vue'
import { compareVersion } from '@/utils/version.js'
import {TextAlignType} from "@/consts/text_align_type.js";

const osTheme = useOsTheme()
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
            font: '',
            fontFamily: [],
            fontSize: 14,
            scanSize: 3000,
            keyIconStyle: 0,
            useSysProxy: false,
            useSysProxyHttp: false,
            checkUpdate: true,
            skipVersion: '',
            allowTrack: true,
        },
        editor: {
            font: '',
            fontFamily: [],
            fontSize: 14,
            showLineNum: true,
            showFolding: true,
            dropText: true,
            links: true,
            entryTextAlign: TextAlignType.Center,
        },
        cli: {
            fontFamily: [],
            fontSize: 14,
            cursorStyle: 'block',
        },
        proxy: {
            enabled:  false,
            protocal: '',
            addr: '',
            port: '',
        },
        download: {
            dir: '',
        },
        buildInDecoder: [],
        decoder: [],
        lastPref: {},
        fontList: [],
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
         * all systems font list
         * @returns {{path: string, label: string, value: string}[]}
         */
        fontOption() {
            return map(this.fontList, (font) => ({
                value: font.name,
                label: font.name,
                path: font.path,
            }))
        },

        protocalOption() {
            return [
                {
                    value: 'http',
                    label: 'preferences.proxy.protocal_http',
                },
                {
                    value: 'https',
                    label: 'preferences.proxy.protocal_https',
                },
                {
                    value: 'socks5',
                    label: 'preferences.proxy.protocal_socks5',
                },
            ]
        },

        /**
         * current font selection
         * @returns {{fontSize: string, fontFamily?: string}}
         */
        generalFont() {
            const fontStyle = {
                fontSize: this.general.fontSize + 'px',
            }
            if (!isEmpty(this.general.fontFamily)) {
                fontStyle['fontFamily'] = join(
                    map(this.general.fontFamily, (f) => `"${f}"`),
                    ',',
                )
            }
            // compatible with old preferences
            // if (isEmpty(fontStyle['fontFamily'])) {
            //     if (!isEmpty(this.general.font) && this.general.font !== 'none') {
            //         const font = find(this.fontList, { name: this.general.font })
            //         if (font != null) {
            //             fontStyle['fontFamily'] = `${font.name}`
            //         }
            //     }
            // }
            return fontStyle
        },

        editorFont() {
            const fontStyle = {
                fontSize: (this.editor.fontSize || 14) + 'px',
            }
            if (!isEmpty(this.editor.fontFamily)) {
                fontStyle['fontFamily'] = join(
                    map(this.editor.fontFamily, (f) => `"${f}"`),
                    ',',
                )
            }
            // compatible with old preferences
            // if (isEmpty(fontStyle['fontFamily'])) {
            //     if (!isEmpty(this.editor.font) && this.editor.font !== 'none') {
            //         const font = find(this.fontList, { name: this.editor.font })
            //         if (font != null) {
            //             fontStyle['fontFamily'] = `${font.name}`
            //         }
            //     }
            // }
            if (isEmpty(fontStyle['fontFamily'])) {
                fontStyle['fontFamily'] = ['monaco']
            }
            return fontStyle
        },

        cliCursorStyleOption() {
            return [
                {
                    value: 'block',
                    label: 'preferences.cli.cursor_style_block',
                },
                {
                    value: 'underline',
                    label: 'preferences.cli.cursor_style_underline',
                },
                {
                    value: 'bar',
                    label: 'preferences.cli.cursor_style_bar',
                },
            ]
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
            const th = get(this.general, 'theme', 'auto')
            if (th !== 'auto') {
                return th === 'dark'
            } else {
                return osTheme.value === 'dark'
            }
        },

        themeLocale() {
            const lang = this.currentLanguage
            switch (lang) {
                case 'zh':
                    return zhCN
                default:
                    return enUS
            }
        },

        autoCheckUpdate() {
            return get(this.general, 'checkUpdate', false)
        },

        showLineNum() {
            return get(this.editor, 'showLineNum', true)
        },

        showFolding() {
            return get(this.editor, 'showFolding', true)
        },

        dropText() {
            return get(this.editor, 'dropText', true)
        },

        editorLinks() {
            return get(this.editor, 'links', true)
        },

        keyIconType() {
            return get(this.general, 'keyIconStyle', typesIconStyle.SHORT)
        },

        entryTextAlign() {
            return get(this.editor, 'entryTextAlign', TextAlignType.Center)
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
                // default value
                const showLineNum = get(data, 'editor.showLineNum')
                if (showLineNum === undefined) {
                    set(data, 'editor.showLineNum', true)
                }
                const showFolding = get(data, 'editor.showFolding')
                if (showFolding === undefined) {
                    set(data, 'editor.showFolding', true)
                }
                const dropText = get(data, 'editor.dropText')
                if (dropText === undefined) {
                    set(data, 'editor.dropText', true)
                }
                const links = get(data, 'editor.links')
                if (links === undefined) {
                    set(data, 'editor.links', true)
                }
                const proxy = get(data, 'proxy')
                if (proxy === undefined) {
                    set(data, 'proxy', {
                        Enabled: false,
                        Protocal: 'http',
                        Addr: '',
                        Port: '',
                    })
                }
                i18nGlobal.locale.value = this.currentLanguage
            }
        },

        /**
         * load systems font list
         * @returns {Promise<string[]>}
         */
        async loadFontList() {
            const { success, data } = await GetFontList()
            if (success) {
                const { fonts = [] } = data
                this.fontList = fonts
            } else {
                this.fontList = []
            }
            return this.fontList
        },

        /**
         * save preferences to local
         * @returns {Promise<boolean>}
         */
        async savePreferences() {
            const pf = pick(this, ['behavior', 'general', 'editor', 'cli', 'decoder', 'proxy', 'download'])
            const { success, msg } = await SetPreferences(pf)
            // proxy 
            return success === true
        },

        async savePreferencesAndSetProxy() {
            const pf = pick(this, ['behavior', 'general', 'editor', 'cli', 'decoder', 'proxy', 'download'])
            const { success, msg } = await SetPreferences(pf)
            // proxy 
            this.setProxy()
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

        async setProxy() {
            try {
                if (this.proxy.enabled) {
                    // 
                    if (isEmpty(this.proxy.addr)) {
                        $message.error(i18nGlobal.t('preferences.proxy.addr_required'))
                        return false
                    }
                    if (this.proxy.port === 0) {
                        $message.error(i18nGlobal.t('preferences.proxy.port_required'))
                        return false
                    }
                    if (isEmpty(this.proxy.protocal)) {
                        $message.error(i18nGlobal.t('preferences.proxy.protocal_required'))
                        return false
                    }

                    // 
                    const url = `${this.proxy.protocal}://${this.proxy.addr}:${this.proxy.port}`
                    const { success, msg } = await SetProxy(url)
                    if (!success) {
                        $message.error(i18nGlobal.t('preferences.proxy.set_failed') + (msg ? `: ${msg}` : ''))
                        return false
                    }
                    $message.success(i18nGlobal.t('preferences.proxy.set_success'))
                    return true
                } else {
                    // 
                    const { success, msg } = await SetProxy('')
                    if (!success) {
                        $message.error(i18nGlobal.t('preferences.proxy.disable_failed') + (msg ? `: ${msg}` : ''))
                        return false
                    }
                    $message.success(i18nGlobal.t('preferences.proxy.disable_success'))
                    return true
                }
            } catch (error) {
                console.error('Failed to set proxy:', error)
                $message.error(i18nGlobal.t('preferences.proxy.error'))
                return false
            }
        },

        updateDownloadDir(newDir) {
            this.download.dir = newDir
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
            this.general.allowTrack = acceptTrack
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
