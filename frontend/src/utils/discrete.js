import usePreferencesStore from 'stores/preferences.js'
import { setupDaisyUI } from '@/utils/daisyMessage.js'

/**
 * setup discrete api and bind global component (like dialog, message, alert) to window
 * @return {Promise<void>}
 */
export async function setupDiscreteApi() {
  const prefStore = usePreferencesStore()
  
  // 使用 DaisyUI 实现替换 Naive UI
  setupDaisyUI()
}