import { Download, FileText, FileVideo, Film, Sparkles } from "lucide-react"
import type { SimpleIcon } from "simple-icons"
import {
  siBilibili,
  siDailymotion,
  siFacebook,
  siInstagram,
  siNiconico,
  siReddit,
  siSoundcloud,
  siTiktok,
  siTwitch,
  siVimeo,
  siX,
  siYoutube,
} from "simple-icons"

import { cn } from "@/lib/utils"

type LibraryTaskIconProps = {
  taskType: string
  sourceDomain?: string | null
  sourceIcon?: string | null
  className?: string
}

const SITE_ICON_MAP: Record<string, SimpleIcon> = {
  "youtube.com": siYoutube,
  "youtu.be": siYoutube,
  "bilibili.com": siBilibili,
  "x.com": siX,
  "twitter.com": siX,
  "tiktok.com": siTiktok,
  "vimeo.com": siVimeo,
  "twitch.tv": siTwitch,
  "instagram.com": siInstagram,
  "facebook.com": siFacebook,
  "reddit.com": siReddit,
  "soundcloud.com": siSoundcloud,
  "dailymotion.com": siDailymotion,
  "nicovideo.jp": siNiconico,
  "niconico.jp": siNiconico,
}

const TASK_ICON: Record<string, typeof Download> = {
  "yt-dlp": Download,
  download: Download,
  subtitle: FileText,
  subtitle_translate: FileText,
  subtitle_proofread: Sparkles,
  subtitle_qa_review: Sparkles,
  "import-video": FileVideo,
  transcode: Film,
}

export function LibraryTaskIcon({ taskType, sourceDomain, sourceIcon, className }: LibraryTaskIconProps) {
  if (taskType === "yt-dlp" || taskType === "download") {
    const domainKey = normalizeDomain(sourceDomain)
    const icon = domainKey ? SITE_ICON_MAP[domainKey] : undefined
    if (icon) {
      return <BrandIcon icon={icon} className={className} />
    }
    if (sourceIcon) {
      return (
        <img
          src={sourceIcon}
          alt={domainKey || "site"}
          className={cn("h-4 w-4 shrink-0 rounded-sm", className)}
        />
      )
    }
  }
  const FallbackIcon = TASK_ICON[taskType] ?? Download
  return <FallbackIcon className={cn("h-4 w-4 shrink-0 text-muted-foreground", className)} />
}

type BrandIconProps = {
  icon: SimpleIcon
  className?: string
}

function BrandIcon({ icon, className }: BrandIconProps) {
  return (
    <svg
      viewBox="0 0 24 24"
      role="img"
      aria-label={icon.title}
      className={cn("h-4 w-4 shrink-0", className)}
      style={{ color: `#${icon.hex}` }}
    >
      <path d={icon.path} fill="currentColor" />
    </svg>
  )
}

function normalizeDomain(domain?: string | null) {
  if (!domain) {
    return ""
  }
  return domain.trim().toLowerCase()
}
