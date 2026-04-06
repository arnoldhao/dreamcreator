import * as React from "react"

import { cn } from "@/lib/utils"

import { DialogContent, DialogFooter, DialogHeader } from "./dialog"

export const DASHBOARD_DIALOG_CONTENT_CLASS = "border-border/70 bg-background p-4 shadow-2xl sm:rounded-xl"
export const DASHBOARD_DIALOG_SOFT_SURFACE_CLASS = "rounded-xl border border-border/70 bg-card"
export const DASHBOARD_DIALOG_INSET_SURFACE_CLASS = "rounded-xl border border-border/70 bg-card"
export const DASHBOARD_DIALOG_FIELD_SURFACE_CLASS = "rounded-lg border border-border/60 bg-card"

export type DashboardDialogSize = "compact" | "standard" | "detail" | "flow" | "workspace"

const dashboardDialogSizeClassName = (size: DashboardDialogSize = "compact") =>
  size === "workspace"
    ? "max-w-5xl"
    : size === "flow"
      ? "max-w-4xl"
      : size === "detail"
        ? "max-w-3xl"
        : size === "standard"
          ? "max-w-2xl"
          : "max-w-lg"

export interface DashboardDialogContentProps extends React.ComponentPropsWithoutRef<typeof DialogContent> {
  size?: DashboardDialogSize
}

export const DashboardDialogContent = React.forwardRef<
  React.ElementRef<typeof DialogContent>,
  DashboardDialogContentProps
>(({ className, size = "compact", ...props }, ref) => (
  <DialogContent
    ref={ref}
    className={cn(DASHBOARD_DIALOG_CONTENT_CLASS, dashboardDialogSizeClassName(size), className)}
    {...props}
  />
))
DashboardDialogContent.displayName = "DashboardDialogContent"

export function DashboardDialogHeader({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <DialogHeader className={cn("gap-1 text-left sm:text-left", className)} {...props} />
}

export function DashboardDialogBody({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("min-h-0 space-y-4", className)} {...props} />
}

export function DashboardDialogSection({
  className,
  tone = "soft",
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { tone?: "soft" | "inset" | "field" }) {
  return (
    <div
      className={cn(
        "min-w-0 p-4",
        tone === "field"
          ? DASHBOARD_DIALOG_FIELD_SURFACE_CLASS
          : tone === "inset"
            ? DASHBOARD_DIALOG_INSET_SURFACE_CLASS
            : DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
        className,
      )}
      {...props}
    />
  )
}

export function DashboardDialogFooter({
  className,
  ...props
}: React.HTMLAttributes<HTMLDivElement>) {
  return <DialogFooter className={cn("pt-4 sm:gap-2 sm:space-x-0", className)} {...props} />
}
