import { onUnmounted } from 'vue'
import { WS_EVENTS, WS_REQUEST_EVENTS } from '@/consts/wsEvents'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import { useOllamaStore } from '@/stores/ollama'

export function useOllamaEventHandlers() {
    const activeListeners = new Map()
    const ollamaStore = useOllamaStore()

    function setupEventListener(id,model) {
        WebSocketService.connect(id)
            .then(() => {
                WebSocketService.addListener(id,
                    WS_EVENTS.OLLAMA_PULL_UPDATE,
                    (data) => handleOllamaPullModel(id, data));
            })
            .then(() => {
                WebSocketService.send(id, {
                    event: WS_REQUEST_EVENTS.REQUEST_OLLAMA_PULL,
                    ollama: {
                        id: id,
                        model: model,
                    }
                });
            })
            .catch((error) => {
                console.error('Error in setupEventListener:', error);
            });
    }

    function handleOllamaPullModel(id, data) {
        const progress = calculateProgress(data.completed, data.total);
        
        ollamaStore.addOrUpdateDownload({
            id: id,
            status: data.status,
            digest: data.digest,
            total: data.total,
            completed: data.completed,
            progress: progress,
        })

        // if data.status is success, then remove the event listener
        if (data.status === 'success') {
            removeEventListener(id)
        }
    }

    function calculateProgress(completed, total) {
        if (typeof completed !== 'number' || typeof total !== 'number' || total <= 0) {
            return 0;
        }
        return Math.min(100, Math.max(0, Math.round((completed / total) * 100)));
    }

    function removeEventListener(id) {
        WebSocketService.removeListener(id, WS_EVENTS.OLLAMA_PULL_UPDATE);
    }

    function removeAllEventListeners() {
        activeListeners.forEach((_, id) => {
            removeEventListener(id)
        })
    }

    function initOllamaEventHandlers() {
        emitter.on(EMITTER_EVENTS.OLLAMA_PULL_MODEL, (info) => {
            setupEventListener(info.id,info.model)
        })

        // no need to remove event listeners
        // onUnmounted(() => {
        //     removeAllEventListeners()
        //     emitter.off(WS_EVENTS.OLLAMA_PULL_UPDATE)
        // })
    }

    return {
        initOllamaEventHandlers
    }
}