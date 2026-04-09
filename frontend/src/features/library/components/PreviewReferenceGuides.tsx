import * as React from "react"

import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"

export type PreviewGuideSize = {
  width: number
  height: number
}

const TITLE_SAFE_INSET_RATIO = 0.1
const RULE_OF_THIRDS_GUIDES = ["33.333%", "66.667%"] as const

export function PreviewViewportBoundary(props: { className?: string }) {
  return <div className={cn("pointer-events-none absolute inset-0 z-20 border border-dashed border-rose-400/75", props.className)} />
}

export function RenderedFrameReferenceGuides(props: { className?: string }) {
  return (
    <div className={cn("pointer-events-none absolute inset-0 z-20", props.className)}>
      <div className="absolute inset-0 border-2 border-cyan-300/85 shadow-[0_0_0_1px_rgba(8,145,178,0.38)_inset]" />
      <div
        className="absolute border border-emerald-300/75"
        style={{
          inset: `${TITLE_SAFE_INSET_RATIO * 100}%`,
          boxShadow: "0 0 0 1px rgba(16, 185, 129, 0.18) inset",
        }}
      />
      <div className="absolute inset-y-0 left-1/2 w-px -translate-x-1/2 bg-white/55" />
      <div className="absolute inset-x-0 top-1/2 h-px -translate-y-1/2 bg-white/55" />
      {RULE_OF_THIRDS_GUIDES.map((offset) => (
        <div
          key={`vertical-${offset}`}
          className="absolute inset-y-0 border-l border-dashed border-amber-200/65"
          style={{ left: offset }}
        />
      ))}
      {RULE_OF_THIRDS_GUIDES.map((offset) => (
        <div
          key={`horizontal-${offset}`}
          className="absolute inset-x-0 border-t border-dashed border-amber-200/65"
          style={{ top: offset }}
        />
      ))}
    </div>
  )
}

export function PreviewGuideLegend(props: {
  renderedSize: PreviewGuideSize
  viewportSize?: PreviewGuideSize | null
  className?: string
}) {
  const { t } = useI18n()

  return (
    <div className={cn("pointer-events-none absolute left-2 top-2 z-20 flex max-w-[min(18rem,calc(100%-1rem))] flex-col gap-1", props.className)}>
      {props.viewportSize ? (
        <GuideBadge tone="rose">
          {t("library.workspace.preview.viewportGuide")} {formatGuideSize(props.viewportSize)}
        </GuideBadge>
      ) : null}
      <GuideBadge tone="cyan">
        {t("library.workspace.preview.renderedVideoGuide")} {formatGuideSize(props.renderedSize)}
      </GuideBadge>
      <GuideBadge tone="emerald">
        {t("library.workspace.preview.titleSafeGuide")}
      </GuideBadge>
      <GuideBadge tone="white">
        {t("library.workspace.preview.centerGuide")}
      </GuideBadge>
      <GuideBadge tone="amber">
        {t("library.workspace.preview.thirdsGuide")}
      </GuideBadge>
    </div>
  )
}

function GuideBadge(props: { children: React.ReactNode; tone: "rose" | "cyan" | "emerald" | "white" | "amber" }) {
  const accentClassName = {
    rose: "bg-rose-300 shadow-[0_0_10px_rgba(251,113,133,0.65)]",
    cyan: "bg-cyan-300 shadow-[0_0_10px_rgba(103,232,249,0.65)]",
    emerald: "bg-emerald-300 shadow-[0_0_10px_rgba(110,231,183,0.65)]",
    white: "bg-white shadow-[0_0_10px_rgba(255,255,255,0.55)]",
    amber: "bg-amber-200 shadow-[0_0_10px_rgba(253,230,138,0.65)]",
  }[props.tone]

  return (
    <div
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md border border-white/18 bg-slate-950/84 px-2 py-1 font-mono text-[10px] font-semibold uppercase tracking-[0.08em] text-white shadow-[0_10px_30px_-16px_rgba(0,0,0,0.9)] backdrop-blur-sm",
      )}
    >
      <span className={cn("h-1.5 w-1.5 shrink-0 rounded-full", accentClassName)} />
      {props.children}
    </div>
  )
}

function formatGuideSize(size: PreviewGuideSize) {
  return `${Math.max(0, Math.round(size.width))} x ${Math.max(0, Math.round(size.height))}`
}
