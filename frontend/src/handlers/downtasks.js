import { defineStore } from 'pinia'
import { WS_NAMESPACE, WS_RESPONSE_EVENT } from '@/consts/websockets'
import WebSocketService from '@/services/websocket'

export const useDtStore = defineStore('downtasks', {
  state: () => ({
    taskProgressMap: {}, // 任务进度映射
    progressCallbacks: [], // 进度回调函数
    signalCallbacks: [], // 信号回调函数
    installingCallbacks: [], // 安装回调函数
    cookieSyncCallbacks: [] // cookie 同步回调函数
  }),
  actions: {
    // 初始化 WebSocket 事件处理
    init() {
      WebSocketService.addListener(WS_NAMESPACE.DOWNTASKS, this.handleCallback)
    },
    // 清理 WebSocket 事件处理
    cleanup() {
      WebSocketService.removeListener(WS_NAMESPACE.DOWNTASKS, this.handleCallback)
    },
    // 处理 WebSocket 回调
    handleCallback(data) {
      switch (data.event) {
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_PROGRESS:
          this.handleProgress(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_SIGNAL:
          this.handleSignal(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_INSTALLING:
          this.handleInstalling(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_COOKIE_SYNC:
          this.handleCookieSync(data.data)
          break
        default:
          console.warn('Unknown event:', data.event)
      }
    },
    // 处理进度事件
    handleProgress(innerData) {
      if (innerData && innerData.id) {
        this.taskProgressMap[innerData.id] = innerData
        this.progressCallbacks.forEach((callback) => {
          try {
            callback(innerData)
          } catch (error) {
            console.error('Progress callback error:', error)
          }
        })
      }
    },
    // 处理信号事件
    handleSignal(innerData) {
      this.signalCallbacks.forEach((callback) => {
        try {
          callback(innerData)
        } catch (error) {
          console.error('Signal callback error:', error)
        }
      })
    },
    // 处理安装事件
    handleInstalling(innerData) {
      this.installingCallbacks.forEach((callback) => {
        try {
          callback(innerData)
        } catch (error) {
          console.error('Installing callback error:', error)
        }
      })
    },
    // 处理 cookie 同步事件
    handleCookieSync(data) {
      this.cookieSyncCallbacks.forEach(callback => {
        try {
          callback(data)
        } catch (error) {
          console.error('Cookie sync callback error:', error)
        }
      })
    },
    // 注册进度回调
    registerProgressCallback(callback) {
      this.progressCallbacks.push(callback)
    },
    // 取消注册进度回调
    unregisterProgressCallback(callback) {
      this.progressCallbacks = this.progressCallbacks.filter((cb) => cb !== callback)
    },
    // 注册信号回调
    registerSignalCallback(callback) {
      this.signalCallbacks.push(callback)
    },
    // 取消注册信号回调
    unregisterSignalCallback(callback) {
      this.signalCallbacks = this.signalCallbacks.filter((cb) => cb !== callback)
    },
    // 注册安装回调
    registerInstallingCallback(callback) {
      this.installingCallbacks.push(callback)
    },
    // 取消注册安装回调
    unregisterInstallingCallback(callback) {
      this.installingCallbacks = this.installingCallbacks.filter((cb) => cb !== callback)
    },
    // 注册 cookie 同步回调
    registerCookieSyncCallback(callback) {
      this.cookieSyncCallbacks.push(callback)
    }, 
    // 移除 cookie 同步回调
    removeCookieSyncCallback(callback) {
      const index = this.cookieSyncCallbacks.indexOf(callback)
      if (index > -1) {
        this.cookieSyncCallbacks.splice(index, 1)
      }
    }
  }
})