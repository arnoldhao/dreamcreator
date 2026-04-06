import { Badge } from "@/shared/ui/badge"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip"
import { useI18n } from "@/shared/i18n"

import type { LibraryProgress } from "../model/types"
import { useSmoothedProgressSpeed } from "../utils/progress-display"
import { translateLibraryProgressDetail, translateLibraryProgressLabel } from "../utils/progress"

type LibraryProgressBadgeProps = {
  progress?: LibraryProgress | null
  status?: string
  tooltipLabel?: string
  disableTooltip?: boolean
}

export function LibraryProgressBadge({ progress, status, tooltipLabel, disableTooltip }: LibraryProgressBadgeProps) {
  const { t } = useI18n()
  const displaySpeed = useSmoothedProgressSpeed(
    progress?.speed,
    progress?.updatedAt || `${progress?.label ?? ""}|${progress?.percent ?? ""}|${progress?.detail ?? ""}`,
    { enabled: progress ? status === "running" : false },
  )
  if (!progress) {
    return <span className="text-sm text-muted-foreground">-</span>
  }
  const normalizedLabel = translateLibraryProgressLabel(progress.label, t)
  const parts: string[] = [normalizedLabel]
  if (progress.percent !== undefined && progress.percent !== null) {
    parts.push(`${Math.round(progress.percent)}%`)
  }
  if (displaySpeed) {
    parts.push(displaySpeed)
  }
  const label = parts.join(" · ")
  const content = (
    <span className="inline-flex">
      <Badge variant="subtle" className="gap-1">
        {label}
      </Badge>
    </span>
  )
  if (disableTooltip) {
    return content
  }
  const tooltipContent = progress.detail ? translateLibraryProgressDetail(progress.detail, t) : tooltipLabel
  if (!tooltipContent) {
    return content
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>{content}</TooltipTrigger>
      <TooltipContent>{tooltipContent}</TooltipContent>
    </Tooltip>
  )
}
