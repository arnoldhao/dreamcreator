import { WS_NAMESPACE, WS_REQUEST_EVENT, WS_RESPONSE_EVENT } from '@/consts/websockets'
import { EMITTER_EVENTS } from '@/consts/emitter'
import WebSocketService from '@/services/websocket'
import emitter from '@/utils/eventBus'
import useCommonStore from '@/stores/common'
import { i18nGlobal } from "@/utils/i18n.js";

export function useProxyEventHandlers() {
    const commonStore = useCommonStore()

    function handleCallback(data) {
        switch (data.event) {
            case WS_RESPONSE_EVENT.EVENT_PROXY_TEST_RESULT:
                handleProxyResult(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_PROXY_TEST_RESULT_ERROR:
                $message.error(data.data.error)
                break
            case WS_RESPONSE_EVENT.EVENT_PROXY_TEST_RESULT_COMPLETED:
                $message.success(i18nGlobal.t('preferences.proxy.test_completed'))
                break
            case WS_RESPONSE_EVENT.EVENT_PROXY_TEST_RESULT_CANCELED:
                $message.success(i18nGlobal.t('preferences.proxy.test_canceled'))
                break
            default:
                console.warn('Unknown event:', data.event)
        }
    }

    function handleProxyResult(innerData) {
        commonStore.addTestProxySitesResult(innerData)
    }

    function testProxy() {
        WebSocketService.send(WS_NAMESPACE.PROXY, WS_REQUEST_EVENT.EVENT_PROXY_TEST, null)
    }

    function initProxyEventHandlers() {
        // add listener
        WebSocketService.addListener(WS_NAMESPACE.PROXY, (data) => handleCallback(data))
        // emit listen event
        emitter.on(EMITTER_EVENTS.TEST_PROXY, () => {
            testProxy()
        })
    }

    return {
        initProxyEventHandlers
    }
}