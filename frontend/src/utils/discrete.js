import usePreferencesStore from 'stores/preferences.js'
import { setupMacUI } from '@/utils/message.js'

/**
 * setup discrete api and bind global component (like dialog, message, alert) to window
 * @return {Promise<void>}
 */
export async function setupDiscreteApi() {
  const prefStore = usePreferencesStore()
  
  // 绑定全局消息/通知/对话框为自定义 macOS 风格实现
  setupMacUI()
}
