import { WS_NAMESPACE, WS_REQUEST_EVENT, WS_RESPONSE_EVENT } from '@/consts/websockets'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import useDownloadStore from '@/stores/download'

export function useDownloadEventHandlers() {
    const downloadStore = useDownloadStore()

    function handleCallback(data) {
        switch (data.event) {
            case WS_RESPONSE_EVENT.EVENT_DOWNLOAD_PROGRESS:
                handleDownloadUpdate(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_DOWNLOAD_COMPLETED:
                handleDownloadUpdate(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_DOWNLOAD_ERROR:
                handleDownloadUpdate(data.data)
                break
            default:
                console.warn('Unknown event:', data.event)
        }
    }

    function downloadStart(info) {
        WebSocketService.send(WS_NAMESPACE.DOWNLOAD, WS_REQUEST_EVENT.EVENT_DOWNLOAD_START, info)
    }

    function handleDownloadUpdate(innerData) {
        const newData = {
            ...innerData
        }
        downloadStore.setStreamData(newData)
    }

    function initDownloadEventHandlers() {
        // add listener
        WebSocketService.addListener(WS_NAMESPACE.DOWNLOAD, (data) => handleCallback(data))
        // emit listen event
        emitter.on(EMITTER_EVENTS.DOWNLOAD_START, (info) => {
            downloadStart(info)
        })
    }

    return {
        initDownloadEventHandlers
    }
}