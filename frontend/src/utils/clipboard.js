export async function copyText(text, t = null) {
  try {
    await navigator.clipboard.writeText(String(text || ''))
    try { t && $message?.success?.(t('message.copy_success')) } catch {}
    return true
  } catch (e) {
    try { t && $message?.error?.(t('message.copy_failed')) } catch {}
    return false
  }
}

