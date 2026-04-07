import * as React from "react";
import { CheckCircle2, CircleAlert, PauseCircle } from "lucide-react";

import { cn } from "@/lib/utils";
import { Button } from "@/shared/ui/button";
import { CardContent } from "@/shared/ui/card";
import { PanelCard } from "@/shared/ui/dashboard";
import { Separator } from "@/shared/ui/separator";
import {
  SETTINGS_COMPACT_ROW_CLASS,
  SETTINGS_COMPACT_ROW_CONTENT_CLASS,
  SETTINGS_COMPACT_ROW_LABEL_CLASS,
  SETTINGS_COMPACT_SEPARATOR_CLASS,
} from "@/shared/ui/settings-layout";

import type { SetupNavStatus } from "./nav";

export function SetupPageCard({
  title,
  description,
  headerRight,
  className,
  titleClassName,
  children,
}: {
  title: string;
  description?: string;
  headerRight?: React.ReactNode;
  className?: string;
  titleClassName?: string;
  children: React.ReactNode;
}) {
  return (
    <PanelCard tone="solid" className={cn("overflow-hidden text-muted-foreground", className)}>
      <div
        className={cn(
          SETTINGS_COMPACT_ROW_CLASS,
          "flex justify-between gap-4",
          description ? "items-start" : "items-center"
        )}
      >
        <div className="min-w-0 space-y-1">
          <div className={cn("text-sm font-semibold text-muted-foreground", titleClassName)}>{title}</div>
          {description ? <p className="text-xs text-muted-foreground">{description}</p> : null}
        </div>
        {headerRight ? <div className="shrink-0">{headerRight}</div> : null}
      </div>
      <SetupCardSeparator />
      <CardContent className="p-0">{children}</CardContent>
    </PanelCard>
  );
}

export function SetupCardRows({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn("flex flex-col", className)}>{children}</div>;
}

export function SetupCardSection({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return <div className={cn(SETTINGS_COMPACT_ROW_CLASS, className)}>{children}</div>;
}

export function SetupCardSeparator() {
  return <Separator className={cn(SETTINGS_COMPACT_SEPARATOR_CLASS, "w-auto")} />;
}

export function SetupCardRow({
  label,
  children,
  contentClassName,
}: {
  label: React.ReactNode;
  children: React.ReactNode;
  contentClassName?: string;
}) {
  return (
    <div className={cn(SETTINGS_COMPACT_ROW_CLASS, "flex items-center justify-between gap-4")}>
      <div className={cn(SETTINGS_COMPACT_ROW_LABEL_CLASS, "min-w-0")}>{label}</div>
      <div
        className={cn(
          SETTINGS_COMPACT_ROW_CONTENT_CLASS,
          "ml-auto flex min-w-0 items-center justify-end gap-2",
          contentClassName
        )}
      >
        {children}
      </div>
    </div>
  );
}

export function SetupCardValue({ children }: { children: React.ReactNode }) {
  return (
    <SetupValueBadge className="max-w-[16rem]" align="end">
      {children}
    </SetupValueBadge>
  );
}

export function SetupValueBadge({
  children,
  className,
  align = "end",
}: {
  children: React.ReactNode;
  className?: string;
  align?: "start" | "end";
}) {
  return (
    <span
      className={cn(
        "inline-flex max-w-full items-center rounded-md border border-border/70 bg-background/80 px-1.5 py-0.5 text-xs font-medium text-muted-foreground whitespace-nowrap",
        align === "end" ? "justify-end text-right" : "justify-start text-left",
        "truncate",
        className
      )}
    >
      {children}
    </span>
  );
}

export function SetupStatusIcon({
  status,
  className,
}: {
  status: SetupNavStatus;
  className?: string;
}) {
  if (status === "ready") {
    return <CheckCircle2 className={cn("h-4 w-4 text-emerald-600", className)} />;
  }
  if (status === "skipped") {
    return <PauseCircle className={cn("h-4 w-4 text-muted-foreground", className)} />;
  }
  return <CircleAlert className={cn("h-4 w-4 text-amber-600", className)} />;
}

export function SetupCardStatusHeader({
  status,
  showSkip,
  skipLabel,
  onSkip,
}: {
  status: SetupNavStatus;
  showSkip?: boolean;
  skipLabel: string;
  onSkip?: () => void;
}) {
  return (
    <div className="flex items-center gap-2">
      <SetupStatusIcon status={status} />
      {showSkip ? (
        <Button type="button" size="compact" variant="outline" onClick={onSkip}>
          <PauseCircle className="h-3.5 w-3.5" />
          {skipLabel}
        </Button>
      ) : null}
    </div>
  );
}

export function InlineNotice({
  children,
  tone = "info",
  className,
}: {
  children: React.ReactNode;
  tone?: "info" | "warning";
  className?: string;
}) {
  return (
    <div
      className={cn(
        "rounded-lg border px-3 py-2 text-xs",
        tone === "warning"
          ? "border-amber-300/45 bg-amber-500/10 text-amber-800"
          : "border-primary/30 bg-primary/5 text-muted-foreground",
        className
      )}
    >
      {children}
    </div>
  );
}
