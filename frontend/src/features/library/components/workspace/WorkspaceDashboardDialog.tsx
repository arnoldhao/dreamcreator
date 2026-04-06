import * as React from "react"

import { Badge } from "@/shared/ui/badge"
import { Card } from "@/shared/ui/card"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog"
import { Separator } from "@/shared/ui/separator"
import { cn } from "@/lib/utils"

export const WORKSPACE_DIALOG_CONTROL_WIDTH_CLASS = "w-full md:w-[248px]"

export type WorkspaceDialogInfoItem = {
  key: string
  label: string
  value: React.ReactNode
  stacked?: boolean
}

export function WorkspaceDialogSectionBadge({
  children,
  className,
}: {
  children: React.ReactNode
  className?: string
}) {
  return (
    <Badge
      variant="outline"
      className={cn("inline-flex h-5 items-center gap-1.5 rounded-md px-2 text-xs font-medium uppercase tracking-[0.08em]", className)}
    >
      {children}
    </Badge>
  )
}

export function WorkspaceDialogHeaderCard({
  title,
  badge,
  className,
  children,
}: {
  title: string
  badge?: React.ReactNode
  className?: string
  children: React.ReactNode
}) {
  return (
    <section
      className={cn("flex min-w-0 flex-col overflow-hidden px-4 py-3", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS, className)}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 text-sm font-semibold text-foreground">{title}</div>
        {badge ? <div className="min-w-0 shrink">{badge}</div> : null}
      </div>
      <div className="mt-1.5 min-h-0 min-w-0 flex-1">{children}</div>
    </section>
  )
}

export function WorkspaceDialogSectionCard({
  title,
  description,
  badge,
  className,
  children,
}: {
  title: string
  description?: string
  badge?: React.ReactNode
  className?: string
  children: React.ReactNode
}) {
  return (
    <section className={cn("min-w-0 overflow-hidden p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS, className)}>
      <div className={cn("mb-3 flex items-start justify-between gap-3", !description && "items-center")}>
        <div className="min-w-0 space-y-1">
          <div className="text-sm font-semibold text-foreground">{title}</div>
          {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
        </div>
        {badge ? <div className="shrink-0">{badge}</div> : null}
      </div>
      {children}
    </section>
  )
}

export function WorkspaceDialogSummaryRow({
  label,
  value,
}: {
  label: string
  value: React.ReactNode
}) {
  return (
    <div className="flex items-start justify-between gap-3 text-xs leading-5 text-muted-foreground">
      <span className="shrink-0 whitespace-nowrap">{label}</span>
      <span className="min-w-0 text-right font-medium text-foreground">{value}</span>
    </div>
  )
}

export function WorkspaceDialogMetricsCard({
  items,
  columns = 2,
}: {
  items: Array<{ label: string; value: React.ReactNode }>
  columns?: 2 | 3
}) {
  const columnCount = columns === 2 ? 2 : 3
  return (
    <div
      className={cn(
        "grid min-w-0 overflow-hidden",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        columns === 2 ? "grid-cols-2" : "grid-cols-3",
      )}
    >
      {items.map((item, index) => (
        <div
          key={`${item.label}-${index}`}
          className={cn(
            "min-w-0 px-2.5 py-2.5 sm:px-3",
            index % columnCount !== 0 && "border-l border-border/70",
            index >= columnCount && "border-t border-border/70",
          )}
        >
          <div className="overflow-hidden text-xs uppercase leading-tight tracking-[0.04em] text-muted-foreground">
            {item.label}
          </div>
          <div className="mt-1 min-h-5 min-w-0 truncate text-sm font-semibold text-foreground">{item.value}</div>
        </div>
      ))}
    </div>
  )
}

export function WorkspaceDialogItemsCard({
  items,
  className,
  emptyLabel,
}: {
  items: WorkspaceDialogInfoItem[]
  className?: string
  emptyLabel?: string
}) {
  return (
    <Card className={cn("flex min-h-0 flex-col border-border/70", className)}>
      <div className="flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto">
        {items.length === 0 ? (
          <div className="px-3 py-2 text-xs text-muted-foreground">{emptyLabel ?? "-"}</div>
        ) : (
          items.map((item, index) => (
            <React.Fragment key={item.key}>
              {index > 0 ? <Separator /> : null}
              {item.stacked ? (
                <div className="min-w-0 overflow-hidden px-3 py-2">
                  <div className="text-xs text-muted-foreground">{item.label}</div>
                  <div className="mt-2 border-t border-border/60 pt-2 text-foreground">{item.value}</div>
                </div>
              ) : (
                <div className="grid min-h-11 min-w-0 grid-cols-[max-content_minmax(0,1fr)] items-center gap-3 overflow-hidden px-3 py-2">
                  <span className="shrink-0 whitespace-nowrap text-xs text-muted-foreground">{item.label}</span>
                  <div className="flex min-w-0 flex-1 items-center justify-end text-right text-foreground">
                    {item.value}
                  </div>
                </div>
              )}
            </React.Fragment>
          ))
        )}
      </div>
    </Card>
  )
}

export function WorkspaceDialogFormRow({
  label,
  description,
  control,
  className,
  controlClassName,
  alignTop = false,
}: {
  label: string
  description?: string
  control: React.ReactNode
  className?: string
  controlClassName?: string
  alignTop?: boolean
}) {
  return (
    <div
      className={cn(
        "grid gap-3 px-3 py-2.5 md:grid-cols-[minmax(0,1fr)_248px]",
        !alignTop && "md:items-center",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        className,
      )}
    >
      <div className="min-w-0 space-y-1">
        <div className="text-xs text-foreground">{label}</div>
        {description ? <div className="text-xs leading-5 text-muted-foreground">{description}</div> : null}
      </div>
      <div className={cn("min-w-0 md:justify-self-end", WORKSPACE_DIALOG_CONTROL_WIDTH_CLASS, controlClassName)}>
        {control}
      </div>
    </div>
  )
}
