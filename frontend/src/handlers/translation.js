import { WS_NAMESPACE, WS_REQUEST_EVENT, WS_RESPONSE_EVENT } from '@/consts/websockets'
import useSuperTabStore from "@/stores/supertab.js"
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import { EMITTER_EVENTS } from '@/consts/emitter'
import { i18nGlobal } from "@/utils/i18n.js";
import { IsTranslationProcessing } from 'wailsjs/go/trans/WorkQueue'

export function useTranslationEventHandlers() {
    const tabStore = useSuperTabStore()

    function handleCallback(data) {
        switch (data.event) {
            case WS_RESPONSE_EVENT.EVENT_TRANSLATION_PROGRESS:
                handleTranslationUpdate(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_TRANSLATION_COMPLETED:
                handleTranslationUpdate(data.data)
                $message.success(i18nGlobal.t('translation.translation_completed'))
                break
            case WS_RESPONSE_EVENT.EVENT_TRANSLATION_CANCELED:
                handleTranslationUpdate(data.data)
                $message.success(i18nGlobal.t('translation.translation_cancelled'))
                break
            case WS_RESPONSE_EVENT.EVENT_TRANSLATION_ERROR:
                handleTranslationUpdate(data.data)
                $message.error(i18nGlobal.t('translation.translation_failed', { error: data.data.error }))
                break
            default:
                console.warn('Unknown event:', data.event)
        }
    }

    function translateStart(info) {
        WebSocketService.send(WS_NAMESPACE.TRANSLATION, WS_REQUEST_EVENT.EVENT_TRANSLATION_START, {
            id: info.id,
            originalSubtileId: info.originalSubtileId,
            language: info.language,
        })
    }

    function handleTranslationUpdate(innerData) {
        tabStore.updateTranslation(innerData.id, innerData.content, innerData.status, formatNumber(innerData.progress), innerData.message)
        if (innerData.status === 'completed' || innerData.status === 'error' || innerData.status === 'canceled') {
            tabStore.streamDone(innerData.id)
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

    const tabQueue = []
    let isProcessing = false

    async function processQueue() {
        if (isProcessing || tabQueue.length === 0) return

        isProcessing = true
        const newTab = tabQueue.shift()

        // check if tab is processing
        const { success } = await IsTranslationProcessing(newTab.id)
        if (success) {
            tabQueue.unshift(newTab)
            $message.info(i18nGlobal.t('translation.translation_processing'))
            return
        } else {
            const currentStreamStatus = newTab.getTranslationState().streamStatus
            if (newTab.stream && currentStreamStatus === 'streaming') {
                translateStart(newTab)
            }
        }


        isProcessing = false
        processQueue()
    }
    function initTranslateEventHandlers() {
        // add listener
        WebSocketService.addListener(WS_NAMESPACE.TRANSLATION, (data) => handleCallback(data))
        // emit listen event
        emitter.on(EMITTER_EVENTS.TRANSLATE_SUBTITLE, (newTab) => {
            if (newTab) {
                tabQueue.push(newTab)
                processQueue()
            }
        })
    }

    return {
        initTranslateEventHandlers
    }
}