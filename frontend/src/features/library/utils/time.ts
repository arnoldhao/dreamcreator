import { t } from "@/shared/i18n"

const RELATIVE_UNITS: Array<{ unit: Intl.RelativeTimeFormatUnit; seconds: number }> = [
  { unit: "year", seconds: 60 * 60 * 24 * 365 },
  { unit: "month", seconds: 60 * 60 * 24 * 30 },
  { unit: "week", seconds: 60 * 60 * 24 * 7 },
  { unit: "day", seconds: 60 * 60 * 24 },
  { unit: "hour", seconds: 60 * 60 },
  { unit: "minute", seconds: 60 },
  { unit: "second", seconds: 1 },
]

export function formatRelativeTime(value?: string, language = "en"): string {
  if (!value) {
    return "-"
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return "-"
  }
  const now = new Date()
  const diffSeconds = Math.round((date.getTime() - now.getTime()) / 1000)
  const absSeconds = Math.abs(diffSeconds)
  if (absSeconds < 5) {
    return t("library.time.justNow", language as "en" | "zh-CN")
  }
  const formatter = new Intl.RelativeTimeFormat(language, { numeric: "auto" })
  for (const { unit, seconds } of RELATIVE_UNITS) {
    if (absSeconds >= seconds || unit === "second") {
      return formatter.format(Math.round(diffSeconds / seconds), unit)
    }
  }
  return formatter.format(diffSeconds, "second")
}

export function toISOFromTimestamp(value?: number | null): string | undefined {
  if (!value || value <= 0) {
    return undefined
  }
  const millis = value > 1_000_000_000_000 ? value : value * 1000
  const date = new Date(millis)
  if (Number.isNaN(date.getTime())) {
    return undefined
  }
  return date.toISOString()
}

export function formatDuration(start?: string, end?: string): string {
  if (!start) {
    return "-"
  }
  const startDate = new Date(start)
  if (Number.isNaN(startDate.getTime())) {
    return "-"
  }
  const endDate = end ? new Date(end) : new Date()
  if (Number.isNaN(endDate.getTime())) {
    return "-"
  }
  let totalSeconds = Math.max(0, Math.round((endDate.getTime() - startDate.getTime()) / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  totalSeconds -= hours * 3600
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds - minutes * 60
  const parts: string[] = []
  if (hours > 0) {
    parts.push(`${hours}h`)
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`)
  }
  if (parts.length === 0 || seconds > 0) {
    parts.push(`${seconds}s`)
  }
  return parts.join(" ")
}
