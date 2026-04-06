import * as React from "react";

import { cn } from "@/lib/utils";

import { Card, CardDescription, CardTitle } from "./card";

export const DASHBOARD_PANEL_CARD_CLASS = "app-surface-panel app-motion-surface";
export const DASHBOARD_PANEL_SOLID_CARD_CLASS = "app-surface-panel-solid app-motion-surface";
export const DASHBOARD_INSET_CARD_CLASS = "app-surface-inset app-motion-surface";

export const DASHBOARD_PANEL_SURFACE_CLASS = "app-surface-panel";
export const DASHBOARD_PANEL_SOLID_SURFACE_CLASS = "app-surface-panel-solid";
export const DASHBOARD_INSET_SURFACE_CLASS = "app-surface-inset";
export const DASHBOARD_SOFT_SURFACE_CLASS = "app-surface-soft";
export const DASHBOARD_FIELD_SURFACE_CLASS = "app-surface-field";
export const DASHBOARD_CHART_SURFACE_CLASS = "h-full min-h-[220px] rounded-lg border border-border/70 bg-background/60 p-3";
export const DASHBOARD_RECORD_ITEM_CLASS = "rounded-lg border border-border/70 bg-background/70 px-3 py-2";
export const DASHBOARD_EMPTY_PANEL_CLASS =
  "flex h-full items-center justify-center rounded-lg border border-dashed text-xs text-muted-foreground";
export const DASHBOARD_CONTROL_GROUP_CLASS =
  "inline-flex overflow-visible rounded-lg border border-border/70 bg-background/80";
export const DASHBOARD_WORKSPACE_SHELL_SURFACE_CLASS = "app-surface-workspace";
export const DASHBOARD_WORKSPACE_META_BAR_CLASS =
  "border-t border-border/70 bg-muted/[0.12] text-xs text-muted-foreground";

type PanelCardTone = "panel" | "solid" | "inset";

export interface PanelCardProps extends React.ComponentPropsWithoutRef<typeof Card> {
  tone?: PanelCardTone;
}

export const panelCardToneClassName = (tone: PanelCardTone = "panel") =>
  tone === "solid"
    ? DASHBOARD_PANEL_SOLID_CARD_CLASS
    : tone === "inset"
      ? DASHBOARD_INSET_CARD_CLASS
      : DASHBOARD_PANEL_CARD_CLASS;

export const PanelCard = React.forwardRef<HTMLDivElement, PanelCardProps>(
  ({ tone = "panel", className, ...props }, ref) => (
    <Card ref={ref} className={cn(panelCardToneClassName(tone), className)} {...props} />
  ),
);
PanelCard.displayName = "PanelCard";

export interface MetricCardProps {
  title: React.ReactNode;
  value: React.ReactNode;
  description?: React.ReactNode;
  icon?: React.ComponentType<{ className?: string }>;
  className?: string;
  valueClassName?: string;
  bodyClassName?: string;
  minHeightClassName?: string;
}

export function MetricCard({
  title,
  value,
  description,
  icon: Icon,
  className,
  valueClassName,
  bodyClassName,
  minHeightClassName = "min-h-[112px]",
}: MetricCardProps) {
  return (
    <PanelCard tone="solid" className={className}>
      <div
        className={cn(
          "flex h-full flex-col justify-between p-4",
          minHeightClassName,
          bodyClassName,
        )}
      >
        <div className="flex items-start justify-between gap-3">
          <CardDescription className="truncate text-sm font-medium text-foreground">
            {title}
          </CardDescription>
          {Icon ? (
            <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center text-muted-foreground">
              <Icon className="h-4 w-4" />
            </span>
          ) : null}
        </div>
        <CardTitle
          className={cn(
            "truncate text-left whitespace-nowrap text-3xl font-semibold tracking-[-0.02em]",
            valueClassName,
          )}
        >
          {value}
        </CardTitle>
        {description ? (
          <CardDescription className="truncate text-xs text-muted-foreground">
            {description}
          </CardDescription>
        ) : null}
      </div>
    </PanelCard>
  );
}
