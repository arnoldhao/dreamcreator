import { defineStore } from 'pinia'

export const useOllamaStore = defineStore('ollama', {
  state: () => ({
    downloads: []
  }),
  actions: {
    addOrUpdateDownload(downloadInfo) {
      const index = this.downloads.findIndex(d => d.id === downloadInfo.id)
      if (index !== -1) {
        // update existing download
        this.downloads[index] = { ...this.downloads[index], ...downloadInfo }
      } else {
        // add new download
        this.downloads.push(downloadInfo)
      }
    },
    removeDownload(id) {
      const index = this.downloads.findIndex(d => d.id === id)
      if (index !== -1) {
        this.downloads.splice(index, 1)
      }
    }
  },
  getters: {
    isAnyDownloading: (state) => state.downloads.length > 0,
    getDownloadById: (state) => (id) => state.downloads.find(d => d.id === id)
  }
})