import * as React from "react"

import type { AssStyleSpecDTO } from "@/shared/contracts/library"
import { useI18n } from "@/shared/i18n"
import { cn } from "@/lib/utils"
import { Button } from "@/shared/ui/button"
import { Input } from "@/shared/ui/input"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog"
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty"

import { formatAssColorWithRgb, parseAssColor } from "./SubtitleStylePresetManagerShared"

export function InlineTypographyButtons(props: {
  value: AssStyleSpecDTO
  disabled?: boolean
  onChange: (patch: Partial<AssStyleSpecDTO>) => void
}) {
  const { t } = useI18n()

  const items = [
    {
      key: "bold" as const,
      label: "B",
      active: props.value.bold,
      title: t("library.config.subtitleStyles.bold"),
    },
    {
      key: "italic" as const,
      label: "I",
      active: props.value.italic,
      title: t("library.config.subtitleStyles.italic"),
    },
    {
      key: "underline" as const,
      label: "U",
      active: props.value.underline,
      title: t("library.config.subtitleStyles.underline"),
    },
    {
      key: "strikeOut" as const,
      label: "S",
      active: props.value.strikeOut,
      title: t("library.config.subtitleStyles.strikeOut"),
    },
  ]

  return (
    <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background">
      {items.map((item, index) => (
        <Button
          key={item.key}
          type="button"
          variant="ghost"
          size="compact"
          title={item.title}
          disabled={props.disabled}
          className={cn(
            "h-8 rounded-none border-0 px-3 font-semibold",
            index > 0 ? "border-l border-border/60" : "",
            item.active ? "bg-background text-foreground" : "text-muted-foreground",
          )}
          onClick={() => props.onChange({ [item.key]: !item.active })}
        >
          {item.label}
        </Button>
      ))}
    </div>
  )
}

export function SubtitleStyleEmptyState(props: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  title: string
  description: string
}) {
  const Icon = props.icon

  return (
    <div className="flex min-h-0 flex-1 items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center">
      <Empty className="max-w-lg py-8">
        <EmptyHeader>
          <EmptyMedia className="flex h-14 w-14 items-center justify-center rounded-full border border-border/70 bg-background/80 text-muted-foreground">
            <Icon className="h-6 w-6" />
          </EmptyMedia>
          <EmptyTitle>{props.title}</EmptyTitle>
          <EmptyDescription>{props.description}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  )
}

export function EditorGroupCard(props: {
  title?: string
  children: React.ReactNode
}) {
  return (
    <div className={cn("space-y-2 rounded-xl px-3 py-3", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
      {props.title ? <div className="text-xs font-semibold tracking-[0.04em] text-foreground/85">{props.title}</div> : null}
      <div className="space-y-2">{props.children}</div>
    </div>
  )
}

export function EditorRow(props: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="grid items-center gap-2 sm:grid-cols-[84px_minmax(0,1fr)]">
      <div className="text-[11px] text-muted-foreground">{props.label}</div>
      <div className="min-w-0">{props.children}</div>
    </div>
  )
}

export function InfoItem(props: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[84px_minmax(0,1fr)] items-start gap-2">
      <div className="text-[11px] text-muted-foreground">{props.label}</div>
      <div className="break-all text-xs text-foreground">{props.value || "-"}</div>
    </div>
  )
}

export function NativeSelect(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={cn(
        "flex h-8 w-full rounded-lg border border-border/70 bg-background px-2.5 text-xs outline-none transition-colors",
        "focus:border-ring focus:ring-2 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-60",
        props.className,
      )}
    />
  )
}

export function NumberInput(props: {
  value: number
  integer?: boolean
  disabled?: boolean
  onChange: (value: number) => void
}) {
  return (
    <Input
      type="number"
      step={props.integer ? 1 : "any"}
      className="h-8 text-xs md:text-xs"
      value={Number.isFinite(props.value) ? props.value : 0}
      disabled={props.disabled}
      onChange={(event) => {
        const next = Number(event.target.value)
        if (!Number.isFinite(next)) {
          return
        }
        props.onChange(props.integer ? Math.round(next) : next)
      }}
    />
  )
}

export function AssColorCompactField(props: {
  value: string
  disabled?: boolean
  onChange: (value: string) => void
}) {
  const parsed = parseAssColor(props.value)
  const swatchColor = parsed?.rgb ?? "#ffffff"

  return (
    <div className="flex h-8 items-center gap-2 rounded-lg border border-border/70 bg-background px-2.5">
      <div className="flex h-5 w-5 items-center justify-center rounded-full border border-border/70">
        <span className="h-full w-full rounded-full" style={{ backgroundColor: swatchColor }} />
      </div>
      <Input
        value={props.value}
        disabled={props.disabled}
        onChange={(event) => props.onChange(event.target.value)}
        className="h-7 border-0 bg-transparent px-0 font-mono text-xs shadow-none focus-visible:ring-0 md:text-xs"
      />
      <input
        type="color"
        value={swatchColor}
        disabled={props.disabled}
        onChange={(event) => props.onChange(formatAssColorWithRgb(event.target.value, props.value))}
        className="h-5 w-5 rounded-full border border-border/70 bg-transparent p-0 disabled:cursor-not-allowed"
      />
    </div>
  )
}
