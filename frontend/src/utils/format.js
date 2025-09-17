// Shared formatting helpers for durations, sizes and dates

export function formatDuration(seconds, t = null) {
  if (seconds == null) return t ? t('common.unknown_duration') : ''
  const s = Math.max(0, Math.floor(Number(seconds)))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const r = s % 60
  return [h, m, r].map((n) => String(n).padStart(2, '0')).join(':')
}

export function formatFileSize(bytes, t = null) {
  if (bytes == null) return t ? t('common.unknown_size') : ''
  const u = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = Number(bytes)
  while (v >= 1024 && i < u.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(1)} ${u[i]}`
}

export function formatDate(ts) {
  if (ts == null) return ''
  let ms = ts
  if (typeof ts === 'number') {
    ms = ts < 1e12 ? ts * 1000 : ts
  } else if (typeof ts === 'string' && /^\d+(\.\d+)?$/.test(ts)) {
    const num = Number(ts)
    ms = num < 1e12 ? num * 1000 : num
  }
  try { return new Date(ms).toLocaleString() } catch { return '' }
}

export function formatEstimated(eta) {
  if (eta == null) return ''
  const s = String(eta)
  return s.includes('.') ? s.split('.')[0] : s
}

