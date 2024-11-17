import { WS_EVENTS, WS_REQUEST_EVENTS } from '@/consts/wsEvents'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import useDownloadStore from '@/stores/download'

export function useDownloadEventHandlers() {
    const activeListeners = new Map()
    const downloadStore = useDownloadStore()

    function setupEventListener(id, info) {
        WebSocketService.connect(id)
            .then(() => {
                WebSocketService.addListener(id,
                    WS_EVENTS.DOWNLOAD_UPDATE,
                    (data) => handleDownloadUpdate(id, data));
            })
            .then(() => {
                WebSocketService.send(id, {
                    event: WS_REQUEST_EVENTS.REQUEST_DOWNLOAD,
                    download: {
                        ...info
                    }
                });
            })
            .catch((error) => {
                console.error('Error in setupEventListener:', error);
            });
    }

    function handleDownloadUpdate(id, data) {
        const newData = {
            id: id,
            ...data
        }
        downloadStore.setStreamData(newData)
    }

    function removeDownloadEventListener(id) {
        WebSocketService.removeListener(id, WS_EVENTS.DOWNLOAD_UPDATE);
    }

    function removeAllEventListeners() {
        activeListeners.forEach((_, id) => {
            removeDownloadEventListener(id)
        })
    }

    function initDownloadEventHandlers() {
        emitter.on(EMITTER_EVENTS.DOWNLOAD, (info) => {
            setupEventListener(info.id, info)
        })

        // no need to remove event listeners
        // onUnmounted(() => {
        //     removeAllEventListeners()
        //     emitter.off(WS_EVENTS.DOWNLOAD_UPDATE)
        // })
    }

    return {
        initDownloadEventHandlers
    }
}