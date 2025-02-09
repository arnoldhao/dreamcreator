import { defineStore } from 'pinia'
import { ListDownloaded} from 'wailsjs/go/api/DownloadAPI'

const useDownloadStore = defineStore('download', {
    state: () => ({
        streamData: [],
        downloadData: [],
        instantData: []
    }),
    actions: {
        setStreamData(data) {
            const index = this.streamData.findIndex(item => item.id === data.id)
            if (index !== -1) {
                this.streamData[index] = { ...this.streamData[index], ...data }
            } else {
                this.streamData.push(data)
            }
            this.updateInstantData()
        },

        updateInstantData() {
            if (!Array.isArray(this.downloadData)) {
                console.warn('downloadData 不是数组:', this.downloadData)
                this.downloadData = []
                return
            }
            
            this.instantData = this.downloadData.map(downloadItem => {
                const streamItem = this.streamData.find(s => s.id === downloadItem.taskId)
                return streamItem ? { ...downloadItem, ...streamItem } : downloadItem
            })
        },

        async setInstantData() {
            try {
                const { data, success, msg } = await ListDownloaded()
                if (!success) {
                    throw new Error(msg)
                }
                
                // If data is a string, try to parse it
                let parsedData
                if (typeof data === 'string') {
                    try {
                        parsedData = JSON.parse(data)
                    } catch (e) {
                        console.error('Failed to parse data:', e)
                        parsedData = []
                    }
                } else {
                    parsedData = data
                }
                
                // Ensure the final result is an array
                this.downloadData = Array.isArray(parsedData) ? parsedData : []
                this.updateInstantData()
            } catch (error) {
                console.error('Failed to get download data:', error)
                throw error
            }
        },

        clearStreamData(id) {
            if (id) {
                this.streamData = this.streamData.filter(item => item.id !== id)
            } else {
                this.streamData = []
            }
            this.updateInstantData()
        }
    }
})

export default useDownloadStore