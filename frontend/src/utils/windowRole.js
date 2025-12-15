// Lightweight helper to decide which "role" the current window plays and,
// for the Settings window, which initial sub-page should be shown.

export function getWindowBootstrap() {
  try {
    const search = window.location.search || ''
    const params = new URLSearchParams(search)
    const role = params.get('window') === 'settings' ? 'settings' : 'main'
    const settingsPage = params.get('page') || ''
    return { role, settingsPage }
  } catch {
    return { role: 'main', settingsPage: '' }
  }
}

