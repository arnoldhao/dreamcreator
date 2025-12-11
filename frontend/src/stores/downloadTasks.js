import { defineStore } from 'pinia'
import { WS_NAMESPACE, WS_RESPONSE_EVENT } from '@/consts/websockets'
import WebSocketService from '@/services/websocket'

export const useDtStore = defineStore('downtasks', {
  state: () => ({
    taskProgressMap: {}, // 任务进度映射
    subtitleProgressMap: {}, // 字幕任务进度（保留最近一次，用于刷新后回填）
    progressCallbacks: [], // 进度回调函数
    signalCallbacks: [], // 信号回调函数
    stageCallbacks: [], // 阶段事件回调
    installingCallbacks: [], // 安装回调函数
    cookieSyncCallbacks: [], // cookie 同步回调函数
    analysisCallbacks: [], // 原因分析事件回调
    subtitleProgressCallbacks: [], // 字幕进度回调函数
    subtitleChatCallbacks: [],     // 字幕翻译 LLM 对话回调
    wsInited: false, // WebSocket 监听是否已初始化（避免重复注册）
  }),
  actions: {
    // 初始化 WebSocket 事件处理
    init() {
      if (this.wsInited) return
      WebSocketService.addListener(WS_NAMESPACE.DOWNTASKS, this.handleDowntasksCallback)
      WebSocketService.addListener(WS_NAMESPACE.SUBTITLES, this.handleSubtitleCallback)
      this.wsInited = true
    },
    // 清理 WebSocket 事件处理
    cleanup() {
      if (!this.wsInited) return
      WebSocketService.removeListener(WS_NAMESPACE.DOWNTASKS, this.handleDowntasksCallback)
      // 修正命名空间，确保与 init 对应
      WebSocketService.removeListener(WS_NAMESPACE.SUBTITLES, this.handleSubtitleCallback)
      this.wsInited = false
    },
    // 处理 WebSocket 回调
    handleDowntasksCallback(data) {
      switch (data.event) {
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_PROGRESS:
          this.handleProgress(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_SIGNAL:
          this.handleSignal(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_INSTALLING:
          // 安装进度事件：
          // 1) 始终通知依赖安装回调（Settings/Dependency 用）
          this.handleInstalling(data.data)
          // 2) 仅当是下载任务侧的“刷新信号”才广播给通用 signal 回调，避免依赖安装也触发内容页频繁刷新
          try {
            const payload = data?.data || {}
            const id = String(payload.id || payload.ID || '')
            const isDependency = id.startsWith('dep-') || typeof payload.type === 'string' && ['yt-dlp','ffmpeg','deno'].includes(String(payload.type).toLowerCase())
            const shouldRefresh = payload.refresh === true // 后端 DTSignal: json:"refresh" -> JS: payload.refresh
            if (shouldRefresh && !isDependency) {
              this.handleSignal(payload)
            }
          } catch (e) {
            // 忽略保护性错误，不影响主流程
          }
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_COOKIE_SYNC:
          this.handleCookieSync(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_STAGE:
          this.handleStage(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_DOWNTASKS_ANALYSIS:
          this.handleAnalysis(data.data)
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
    // 阶段事件（无强制百分比）
    handleStage(innerData) {
      // persist per-task stage status for list/cards
      try {
        const id = innerData?.id
        if (id) {
          const kind = String(innerData.kind || '').toLowerCase()
          const action = String(innerData.action || '').toLowerCase()
          if (!this.taskProgressMap[id]) this.taskProgressMap[id] = {}
          if (!this.taskProgressMap[id]._stages) this.taskProgressMap[id]._stages = { video: 'idle', subtitle: 'idle', merge: 'idle', finalize: 'idle' }
          if (['video','subtitle','merge','finalize'].includes(kind)) {
            this.taskProgressMap[id]._stages[kind] = (action === 'complete') ? 'done' : (action === 'error' ? 'error' : 'working')
          }
        }
      } catch (e) { /* ignore */ }
      // callbacks
      this.stageCallbacks.forEach((callback) => {
        try {
          callback(innerData)
        } catch (error) {
          console.error('Stage callback error:', error)
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
    // 处理分析事件
    handleAnalysis(data) {
      this.analysisCallbacks.forEach(callback => {
        try { callback(data) } catch (error) { console.error('Analysis callback error:', error) }
      })
    },
    // SUBTITLE
    handleSubtitleCallback(data) {
      switch (data.event) {
        case WS_RESPONSE_EVENT.EVENT_SUBTITLE_PROGRESS:
          this.handleSubtitleProgress(data.data)
          break
        case WS_RESPONSE_EVENT.EVENT_SUBTITLE_CHAT:
          this.handleSubtitleChat(data.data)
          break
        default:
          console.warn('Unknown event:', data.event)
      }
    },
    // 处理字幕进度事件
    handleSubtitleProgress(data) {
      // 持久化最近一次字幕任务进度，便于页面主动刷新后仍可显示进度
      try {
        const id = data?.id
        if (id) this.subtitleProgressMap[id] = data
      } catch {}
      this.subtitleProgressCallbacks.forEach((callback) => {
        try {
          callback(data)
        } catch (error) {
          console.error('Subtitle progress callback error:', error)
        }
      })
    },
    // 处理字幕 LLM 对话事件
    handleSubtitleChat(data) {
      this.subtitleChatCallbacks.forEach((callback) => {
        try {
          callback(data)
        } catch (error) {
          console.error('Subtitle talk callback error:', error)
        }
      })
    },
    // 注册字幕进度回调
    registerSubtitleProgressCallback(callback) {
      this.subtitleProgressCallbacks.push(callback)
    },
    // 取消注册字幕进度回调
    unregisterSubtitleProgressCallback(callback) {
      this.subtitleProgressCallbacks = this.subtitleProgressCallbacks.filter((cb) => cb !== callback)
    },
    // 注册/取消 字幕 LLM 对话回调
    registerSubtitleChatCallback(callback) {
      this.subtitleChatCallbacks.push(callback)
    },
    unregisterSubtitleChatCallback(callback) {
      this.subtitleChatCallbacks = this.subtitleChatCallbacks.filter((cb) => cb !== callback)
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
    // 注册/取消 阶段回调
    registerStageCallback(callback) { this.stageCallbacks.push(callback) },
    unregisterStageCallback(callback) { this.stageCallbacks = this.stageCallbacks.filter((cb) => cb !== callback) },
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
    ,
    // 注册/取消 原因分析回调
    registerAnalysisCallback(callback) { this.analysisCallbacks.push(callback) },
    unregisterAnalysisCallback(callback) { this.analysisCallbacks = this.analysisCallbacks.filter(cb => cb !== callback) }
  }
})
