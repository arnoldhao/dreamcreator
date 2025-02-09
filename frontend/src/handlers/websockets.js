import WebSocketService from '@/services/websocket'

export function WSON() {
    function connect() {
        WebSocketService.connect()
    }

    return {
        connect
    }
}