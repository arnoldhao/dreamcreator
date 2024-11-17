import { WS_EVENTS, WS_REQUEST_EVENTS } from '@/consts/wsEvents'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import useCommonStore from '@/stores/common'

export function useCommonEventHandlers() {
    const activeListeners = new Map()
    const commonStore = useCommonStore()

    function setupEventListener(id) {
        WebSocketService.connect(id)
            .then(() => {
                WebSocketService.addListener(id,
                    WS_EVENTS.TEST_PROXY_RESULT,
                    (data) => handleTestProxyResult(id, data));
            })
            .then(() => {
                WebSocketService.send(id, {
                    event: WS_REQUEST_EVENTS.REQUEST_TEST_PROXY,
                });
            })
            .catch((error) => {
                console.error('Error in setupEventListener:', error);
            });
    }

    function handleTestProxyResult(id, data) {
        commonStore.addTestProxySitesResult(data)

        // if data.done is true, then remove the event listener
        if (data.done === true) {
            removeTestProxyEventListener(id)
        }
    }

    function removeTestProxyEventListener(id) {
        WebSocketService.removeListener(id, WS_EVENTS.TEST_PROXY_RESULT);
    }

    function removeAllEventListeners() {
        activeListeners.forEach((_, id) => {
            removeEventListener(id)
        })
    }

    function initCommonEventHandlers() {
        // test proxy request
        emitter.on(EMITTER_EVENTS.TEST_PROXY, (info) => {
            setupEventListener(info.id)
        })

        // no need to remove event listeners
        // onUnmounted(() => {
        //     removeAllEventListeners()
        //     emitter.off(WS_EVENTS.TEST_PROXY_RESULT)
        // })
    }

    return {
        initCommonEventHandlers
    }
}