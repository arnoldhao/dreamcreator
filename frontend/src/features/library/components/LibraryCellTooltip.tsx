import type { ReactElement, ReactNode } from "react"

import { Tooltip, TooltipContent, TooltipTrigger } from "@/shared/ui/tooltip"

type LibraryCellTooltipProps = {
  label?: ReactNode
  children: ReactElement
}

export function LibraryCellTooltip({ label, children }: LibraryCellTooltipProps) {
  if (!label) {
    return children
  }
  return (
    <Tooltip>
      <TooltipTrigger asChild>{children}</TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  )
}
