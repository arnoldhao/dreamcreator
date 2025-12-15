import { defineStore } from 'pinia'
import { useDtStore } from '@/stores/downloadTasks'
import { ListLLMTasks, DeleteLLMTask } from 'bindings/dreamcreator/backend/api/subtitlesapi'

export const useSubtitleTasksStore = defineStore('subtitleTasks', {
  state: () => ({
    tasksMap: {}, // id -> task
    order: [],    // recent first
    taskProjectMap: {}, // taskId -> projectId
    inited: false,
  }),
  getters: {
    tasks(state) {
      return state.order.map(id => state.tasksMap[id]).filter(Boolean)
    }
  },
  actions: {
    init() {
      if (this.inited) return
      const dt = useDtStore()
      dt.init?.()
      dt.registerSubtitleProgressCallback(this.onSubtitleProgress)
      // also react to generic refresh signals from backend to keep names up to date
      dt.registerSignalCallback(this.onDtSignal)
      this.loadAll().catch(() => {})
      this.inited = true
    },
    cleanup() {
      if (!this.inited) return
      const dt = useDtStore()
      dt.unregisterSubtitleProgressCallback(this.onSubtitleProgress)
      dt.unregisterSignalCallback?.(this.onDtSignal)
      this.inited = false
    },
    onSubtitleProgress(payload) {
      try {
        if (!payload || !payload.id) return
        const id = payload.id
        const prev = this.tasksMap[id]
        if (prev) {
          Object.assign(prev, payload)
        } else {
          this.tasksMap[id] = { ...payload }
        }
        if (!this.order.includes(id)) this.order.unshift(id)
      } catch (e) { console.error('subtitleTasks.onSubtitleProgress error:', e) }
    },
    onDtSignal(payload) {
      try {
        if (payload && payload.refresh) {
          // refresh full list to update names/derived fields
          this.loadAll().catch(() => {})
        }
      } catch (e) { /* ignore */ }
    },
    async loadAll() {
      try {
        const resp = await ListLLMTasks()
        if (resp?.success) {
          const raw = resp.data
          const arr = Array.isArray(raw) ? raw : JSON.parse(raw || '[]')
          const order = []
          const projMap = {}
          // 合并策略：先更新基本字段，再回填最近一次的进度（来自 dtStore.subtitleProgressMap）
          const dt = useDtStore()
          const lastProgress = (dt && dt.subtitleProgressMap) ? dt.subtitleProgressMap : {}
          for (const t of (arr || [])) {
            if (!t || !t.id) continue
            const id = t.id
            if (this.tasksMap[id]) Object.assign(this.tasksMap[id], t)
            else this.tasksMap[id] = { ...t }
            // 回填最近一次进度，避免刷新后进度清零直到下次WS
            try {
              const prog = lastProgress[id]
              if (prog && typeof prog === 'object') {
                Object.assign(this.tasksMap[id], prog)
              }
            } catch {}
            order.push(id)
            if (t.project_id) projMap[id] = t.project_id
          }
          this.order = order
          this.taskProjectMap = { ...this.taskProjectMap, ...projMap }
        }
      } catch (e) {
        console.error('ListLLMTasks failed:', e)
      }
    },
    async deleteTask(taskId) {
      if (!taskId) return false
      try {
        const r = await DeleteLLMTask(taskId)
        if (r?.success) {
          delete this.tasksMap[taskId]
          this.order = this.order.filter(id => id !== taskId)
          delete this.taskProjectMap[taskId]
          return true
        }
      } catch (e) { console.error('DeleteLLMTask failed:', e) }
      return false
    },
  }
})
