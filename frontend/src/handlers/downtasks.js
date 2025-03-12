import { WS_NAMESPACE, WS_RESPONSE_EVENT } from '@/consts/websockets'
import WebSocketService from '@/services/websocket'
import { ref } from 'vue'

// 创建一个响应式的任务进度映射，用于存储每个任务的最新进度
const taskProgressMap = ref({})

export function useDt() {
    // 注册回调函数
    const progressCallbacks = []
    const singleCallbacks = []

    function handleCallback(data) {
        switch (data.event) {
            case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_PROGRESS:
                handleProgress(data.data)
                break
            case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_SINGLE:
                handleSingle(data.data)
                break
            default:
                console.warn('Unknown event:', data.event)
        }
    }

    function handleProgress(innerData) {
        // 更新任务进度映射
        if (innerData && innerData.id) {
            taskProgressMap.value[innerData.id] = innerData
            
            // 调用所有注册的进度回调函数
            progressCallbacks.forEach(callback => {
                try {
                    callback(innerData)
                } catch (error) {
                    console.error('Error in progress callback:', error)
                }
            })
        }
    }

    function handleSingle(innerData) {
        // 调用所有注册的单任务回调函数
        singleCallbacks.forEach(callback => {
            try {
                callback(innerData)
            } catch (error) {
                console.error('Error in single callback:', error)
            }
        })
    }

    function initDt() {
        // add listener
        WebSocketService.addListener(WS_NAMESPACE.DOWNTASKS, (data) => handleCallback(data))
    }
    
    // 注册进度更新回调
    function onProgress(callback) {
        if (typeof callback === 'function') {
            progressCallbacks.push(callback)
        }
        return () => {
            const index = progressCallbacks.indexOf(callback)
            if (index !== -1) {
                progressCallbacks.splice(index, 1)
            }
        }
    }
    
    // 注册单任务更新回调
    function onSingle(callback) {
        if (typeof callback === 'function') {
            singleCallbacks.push(callback)
        }
        return () => {
            const index = singleCallbacks.indexOf(callback)
            if (index !== -1) {
                singleCallbacks.splice(index, 1)
            }
        }
    }

    return {
        initDt,
        onProgress,
        onSingle,
        taskProgressMap
    }
}