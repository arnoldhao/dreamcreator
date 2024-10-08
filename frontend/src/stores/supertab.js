import { find, findIndex, get, size } from 'lodash'
import { defineStore } from 'pinia'
import { i18nGlobal } from "@/utils/i18n.js";
import { SuperTabItem } from '../objects/supertebItem';
import { Convert, GetCaptions, UpdateCaptions } from 'wailsjs/go/subtitles/Service'
import { CancelTranslation } from 'wailsjs/go/trans/WorkQueue';
import emitter from '@/utils/eventBus'

const useSuperTabStore = defineStore('supertab', {
    state: () => ({
        nav: 'subtitle',
        asideWidth: 300,
        tabList: [],
        activatedIndex: 0,  // current active index
    }),
    getters: {
        tabs() {
            return this.tabList
        },

        currentTab() {
            return get(this.tabs, this.activatedIndex)
        },

        currentTabId() {
            return get(this.tabs, [this.activatedIndex, 'id'])
        },

        currentTabTitle() {
            return get(this.tabs, [this.activatedIndex, 'title'])
        },

        currentNav() {
            return get(this.tabs, [this.activatedIndex, 'nav'])
        },

        latestTab() {
            const ids = Object.keys(this.tabs)
            return ids.length > 0 ? this.tabs[ids[ids.length - 1]] : null
        },
        currentTabCaption() {
            const tab = this.currentTab
            return get(tab, 'captions', '')
        }
    },
    actions: {
        _setActivatedIndex(idx) {
            this.activatedIndex = idx
            this.nav = idx >= 0 ? 'subtitle' : 'ai'
        },

        async convertSubtitle(filePath) {
            let id, title, captions
            try {
                const { data, success, msg } = await Convert(filePath)
                if (success && data) {
                    id = data.key
                    title = data.fileName
                    captions = data.subtitles
                } else {
                    throw new Error(msg || 'convert failed')
                }
            } catch (error) {
                $message.error(error.message || 'convert failed')
            }

            this.updateAndInsertTab({
                id: id,
                title: title,
                filePath: filePath,
                black: false,
                icon: 'file',
                originalSubtileId: '',
                language: 'Original',
                stream: false,
                streamStatus: '',
                captions: captions,
                switchTab: true,
            })
        },

        async editSubtitle(subtitleId, fileName, language) {
            let captions
            try {
                const { data, success, msg } = await GetCaptions(subtitleId)
                if (success && data) {
                    captions = data
                } else {
                    throw new Error(msg || 'get failed')
                }
            } catch (error) {
                $message.error(error.message || 'get failed')
            }

            this.updateAndInsertTab({
                id: subtitleId,
                title: fileName,
                filePath: '',
                black: false,
                icon: 'file',
                originalSubtileId: '',
                language: language,
                stream: false,
                streamStatus: '',
                captions: captions,
                switchTab: true,
            })
        },

        translateSubtitle(originalSubtileId, title, language) {
            const now = BigInt(Date.now()) * BigInt(1000000) + BigInt(Math.floor(performance.now() * 1000)) % BigInt(1000000);
            const id = `translated_${language.toLowerCase()}_${now}`;

            this.updateAndInsertTab({
                id: id,
                title: title + "_" + language,
                filePath: '',
                blank: false,
                icon: 'file',
                originalSubtileId: originalSubtileId,
                language: language,
                stream: true,
                captions: '',
                translationState: {
                    streamStatus: 'streaming',
                    translationStatus: 'pending',
                    translationProgress: 0
                },
                switchTab: true,
            })

            // emit
            emitter.emit('newTranslateSubtitle', this.latestTab)
        },

        openBlankTab() {
            this.updateAndInsertTab(
                {
                    id: 'blank',
                    title: 'Welcome',
                    blank: true,
                }
            )
        },

        async closeTab(id) {
            const tab = this.tabs.find(tab => tab.id === id);
            if (tab) {
                $dialog.warning(i18nGlobal.t('dialogue.close_confirm', { title: tab.title }), () => {
                    this.removeTabById(id)
                })
                if (tab.stream) {
                    try {
                        const { success, msg } = await CancelTranslation(id)
                        if (success) {
                            $message.success(i18nGlobal.t('ai.cancel_success'))
                        } else {
                            $message.error(msg)
                        }
                    } catch (error) {
                        $message.error(i18nGlobal.t('ai.cancel_failed'))
                    }

                } else {
                    console.warn(`Tab with id ${id} not found`);
                }
            }
        },

        removeTab(tabIndex) {
            const len = size(this.tabs)
            // ignore remove last blank tab
            if (len === 1 && this.tabs[0].blank) {
                return null
            }

            if (tabIndex < 0 || tabIndex >= len) {
                return null
            }
            const removed = this.tabs.splice(tabIndex, 1)

            // update select index if removed index equal current selected
            if (this.tabs.length > 0) {
                this._setActivatedIndex(0, false)
            } else {
                this._setActivatedIndex(-1, false)
            }

            return size(removed) > 0 ? removed[0] : null
        },

        removeTabById(id) {
            const idx = findIndex(this.tabs, { id: id })
            if (idx !== -1) {
                this.removeTab(idx)
            }
        },

        switchTab(tabId) {
            const tabIndex = findIndex(this.tabs, { id: tabId })
            if (tabIndex === -1) {
                return
            }
            this._setActivatedIndex(tabIndex)
        },

        removeAllTab() {
            this.tabs = []
        },

        updateTitle(id, title) {
            this.updateAndInsertTab({
                id: id,
                title: title,
                switchTab: false,
            })
        },

        updateTranslation(id, content, status, progress, message) {
            const tab = this.tabs.find(tab => tab.id === id)
            if (tab) {
                if (content && content.length > 0) {
                    this.updateAndInsertTab({
                        id: id,
                        captions: (tab.captions || '') + content,
                        translationStatus: status,
                        translationProgress: progress,
                        actionDescription: message
                    });
                } else {
                    tab.updateTranslationState({
                        id: id,
                        translationStatus: status,
                        translationProgress: progress,
                        actionDescription: message
                    })
                }
 
            }
        },

        async formatCaptions(id, newCaptions) {
            try {
                const { data, success, msg } = await UpdateCaptions(id, newCaptions)
                if (success) {
                    this.updateAndInsertTab({
                        id: id,
                        captions: data,
                    });
                    $message.success("format success")
                } else {
                    throw new Error(msg)
                }
            } catch (error) {
                $message.error("format failed: " + error)
            }
        },

        updateAndInsertTab({
            id,
            title,
            filePath,
            blank,
            icon,
            originalSubtileId,
            language,
            stream,
            captions,
            translationState,
            switchTab
        }) {
            let tabIndex = findIndex(this.tabs, { id: id })
            if (tabIndex === -1) {
                const tabItem = new SuperTabItem({
                    id,
                    title,
                    filePath,
                    blank,
                    icon,
                    originalSubtileId,
                    language,
                    stream,
                    captions,
                })
                if (!blank && translationState) {
                    tabItem.updateTranslationState(translationState)
                }
                this.tabs.push(tabItem)
                tabIndex = this.tabs.length - 1
            } else {
                const tab = this.tabs[tabIndex]
                if (id !== undefined) tab.id = id
                if (title !== undefined) tab.title = title
                if (filePath !== undefined) tab.filePath = filePath
                if (blank !== undefined) tab.blank = blank
                if (icon !== undefined) tab.icon = icon
                if (originalSubtileId !== undefined) tab.originalSubtileId = originalSubtileId
                if (language !== undefined) tab.language = language
                if (stream !== undefined) tab.stream = stream
                if (captions !== undefined) tab.captions = captions
                if (translationState !== undefined) {
                    tab.updateTranslationState(translationState)
                }
            }
            if (switchTab) {
                this._setActivatedIndex(tabIndex)
            }
        },

        // add new method to get translation state
        getTabTranslationState(id) {
            const tab = this.tabs.find(tab => tab.id === id)
            return tab ? tab.getTranslationState() : null
        },

        updateTabTranslationState(id, newState) {
            const tab = this.tabs.find(tab => tab.id === id)
            if (tab) {
                tab.updateTranslationState(newState)
            }
        },
    },
})

export default useSuperTabStore