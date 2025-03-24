import { defineStore } from 'pinia'

export const useLoggerStore = defineStore('logger', {
    state: () => ({}),
    actions: {
        debug(...args) {
            console.debug(...args)
            if (window.runtime?.LogDebug) {
                window.runtime.LogDebug(typeof args[0] === 'string' ? args[0] : JSON.stringify(args))
            }
        },
        info(...args) {
            console.info(...args)
            if (window.runtime?.LogInfo) {
                window.runtime.LogInfo(typeof args[0] === 'string' ? args[0] : JSON.stringify(args))
            }
        },
        warn(...args) {
            console.warn(...args)
            if (window.runtime?.LogWarning) {
                window.runtime.LogWarning(typeof args[0] === 'string' ? args[0] : JSON.stringify(args))
            }
        },
        error(...args) {
            console.error(...args)
            if (window.runtime?.LogError) {
                window.runtime.LogError(typeof args[0] === 'string' ? args[0] : JSON.stringify(args))
            }
        }
    }
})