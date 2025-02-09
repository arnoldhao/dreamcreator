// src/handlers/EventHandler.js
import { EventsOn, EventsOff } from 'wailsjs/runtime'
import { ref } from 'vue'
import useDownloadStore from '@/stores/download'

export const EventTypes = {
  DOWNLOAD_SINGLE: 'download.single'
}

export function useEventStoreHandler() {
  const downloadStore = useDownloadStore()
  const isInitialized = ref(false)

  function cleanupEvents() {
    EventsOff(EventTypes.DOWNLOAD_BEGIN)
    isInitialized.value = false
  }

  function initEventHandlers() {
    if (isInitialized.value) return

    try {
      // 设置前端的事件监听
      EventsOn(EventTypes.DOWNLOAD_SINGLE, ({ taskId, status }) => {
        downloadStore.setInstantData()
      })

      isInitialized.value = true
    } catch (error) {
      console.error('Failed to initialize events:', error)
      throw error
    }
  }

  return {
    initEventHandlers,
    cleanupEvents
  }
}