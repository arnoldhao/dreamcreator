import { WS_NAMESPACE, WS_REQUEST_EVENT, WS_RESPONSE_EVENT } from '@/consts/websockets'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import { useOllamaStore } from '@/stores/ollama'
import { i18nGlobal } from "@/utils/i18n.js";

export function useOllamaEventHandlers() {
    const ollamaStore = useOllamaStore()

    function handleCallback(data) {
        switch (data.event) {
            case WS_RESPONSE_EVENT.EVENT_OLLAMA_PULL_UPDATE:
                handleOllamaPullModel(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_OLLAMA_PULL_COMPLETED:
                handleOllamaPullModel(data.data)
                $message.success(i18nGlobal.t('ollama.pull_model_completed'))
                break
            case WS_RESPONSE_EVENT.EVENT_OLLAMA_PULL_CANCELED:
                handleOllamaPullModel(data.data)
                $message.success(i18nGlobal.t('ollama.pull_model_canceled'))
                break
            case WS_RESPONSE_EVENT.EVENT_OLLAMA_PULL_ERROR:
                handleOllamaPullModel(data.data)
                $message.error(i18nGlobal.t('ollama.pull_model_failed', { error: data.data.error }))
                break
            default:
                console.warn('Unknown event:', data.event)
        }
    }

    function handleOllamaPullModel(innerData) {
        const progress = calculateProgress(innerData.completed, innerData.total);

        ollamaStore.addOrUpdateDownload({
            id: innerData.id,
            status: innerData.status,
            digest: innerData.digest,
            total: innerData.total,
            completed: innerData.completed,
            progress: progress,
        })
    }

    function pullModel(info) {
        WebSocketService.send(WS_NAMESPACE.OLLAMA, WS_REQUEST_EVENT.EVENT_OLLAMA_PULL, {
            id: info.id,
            model: info.model,
        })
    }

    function calculateProgress(completed, total) {
        if (typeof completed !== 'number' || typeof total !== 'number' || total <= 0) {
            return 0;
        }
        return Math.min(100, Math.max(0, Math.round((completed / total) * 100)));
    }

    function initOllamaEventHandlers() {
        // add listener
        WebSocketService.addListener(WS_NAMESPACE.OLLAMA, (data) => handleCallback(data))
        // emit listen event
        emitter.on(EMITTER_EVENTS.OLLAMA_PULL_MODEL, (info) => {
            pullModel(info)
        })
    }

    return {
        initOllamaEventHandlers
    }
}