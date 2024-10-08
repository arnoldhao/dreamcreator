import { WS_EVENTS, WS_REQUEST_EVENTS } from '@/consts/wsEvents'
import useSuperTabStore from "@/stores/supertab.js"
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'

export function useTranslationEventHandlers() {
    const tabStore = useSuperTabStore()

    const activeListeners = new Map()
    function setupEventListener(id, originalSubtileId, language) {
        WebSocketService.connect(id)
            .then(() => {
                WebSocketService.addListener(id,
                    WS_EVENTS.TRANSLATION_UPDATE,
                    (data) => handleTranslationUpdate(id, data));
            })
            .then(() => {
                WebSocketService.send(id, {
                    event: WS_REQUEST_EVENTS.REQUEST_TRANSLATION_START,
                    translate: {
                        id: id,
                        originalSubtileId: originalSubtileId,
                        language: language,
                    }
                });
            })
            .catch((error) => {
                console.error('Error in setupEventListener:', error);
            });
    }

    function handleTranslationUpdate(id, data) {
        tabStore.updateTranslation(id, data.content, data.status, formatNumber(data.progress), data.message)
        if (data.status === 'completed' || data.status === 'error' || data.status === 'canceled') {
            removeEventListener(id)
        }
    }

    function formatNumber(num) {
        if (typeof num !== 'number') {
            num = parseFloat(num);
        }

        if (isNaN(num)) {
            return '0.00';
        }
        return num.toFixed(2);
    }

    function removeEventListener(id) {
        WebSocketService.removeListener(id, WS_EVENTS.TRANSLATION_UPDATE);
    }

    function removeAllEventListeners() {
        activeListeners.forEach((_, id) => {
            removeEventListener(id)
        })
    }

    const tabQueue = []
    let isProcessing = false

    function processQueue() {
        if (isProcessing || tabQueue.length === 0) return

        isProcessing = true
        const newTab = tabQueue.shift()

        const currentStreamStatus = newTab.getTranslationState().streamStatus
        if (newTab.stream && currentStreamStatus === 'streaming') {
            if (!activeListeners.has(newTab.id)) {
                setupEventListener(newTab.id, newTab.originalSubtileId, newTab.language)
            }
        } else {
            if (activeListeners.has(newTab.id)) {
                removeEventListener(newTab.id)
            }
        }

        isProcessing = false
        processQueue()
    }
    function initEventHandlers() {
        emitter.on('newTranslateSubtitle', (newTab) => {
            if (newTab) {
                tabQueue.push(newTab)
                processQueue()
            }
        })

        // no need to remove event listeners
        // onUnmounted(() => {
        //     removeAllEventListeners()
        //     emitter.off('newTranslateSubtitle')
        // })
    }

    return {
        initEventHandlers
    }
}