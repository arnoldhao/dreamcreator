import type { ReactNode } from "react"

import { cn } from "@/lib/utils"

type WorkspaceMetaItemProps = {
  value: ReactNode
  label?: ReactNode
  className?: string
  valueClassName?: string
  title?: string
}

type WorkspaceMetaFlagProps = {
  children: ReactNode
  className?: string
  title?: string
}

export function WorkspaceMetaItem({
  value,
  label,
  className,
  valueClassName,
  title,
}: WorkspaceMetaItemProps) {
  return (
    <span
      className={cn(
        "inline-flex h-6 min-w-0 max-w-full items-center gap-1.5 rounded-md border border-border/60 bg-background/72 px-2 text-[11px] text-muted-foreground",
        className,
      )}
      title={title}
    >
      {label ? <span className="shrink-0 text-muted-foreground/80">{label}</span> : null}
      <span className={cn("min-w-0 truncate font-medium text-foreground", valueClassName)}>
        {value}
      </span>
    </span>
  )
}

export function WorkspaceMetaFlag({
  children,
  className,
  title,
}: WorkspaceMetaFlagProps) {
  return (
    <span
      className={cn(
        "inline-flex h-6 shrink-0 items-center rounded-md border px-2 text-[11px] font-medium",
        className,
      )}
      title={title}
    >
      {children}
    </span>
  )
}
