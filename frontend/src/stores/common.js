import { defineStore } from 'pinia'

const useCommonStore = defineStore('common', {
  state: () => ({
    testProxySites: []
  }),
  actions: {
    addTestProxySitesResult(sites) {
      try {
        // If the input is a JSON string, parse it first
        const parsedSites = typeof sites === 'string' ? JSON.parse(sites) : sites
        
        // Ensure it's an array
        const sitesArray = Array.isArray(parsedSites) ? parsedSites : [parsedSites]
        
        this.testProxySites.push(...sitesArray)
      } catch (error) {
        console.error('Failed to parse sites data:', error)
      }
    },
    emptyTestProxySites() {
      this.testProxySites = []
    }
  }
})

export default useCommonStore