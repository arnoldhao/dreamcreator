import type { PropsWithChildren } from "react";
import { AlertTriangle, Brain, CheckCircle2, ChevronDown, Loader2, XCircle } from "lucide-react";

import { cn } from "@/lib/utils";

type ReasoningStatus = "running" | "requires-action" | "incomplete" | "complete";

type ReasoningOutlineProps = PropsWithChildren<{
  title: string;
  statusLabel: string;
  status: ReasoningStatus;
  className?: string;
  contentClassName?: string;
}>;

const resolveStatusAppearance = (status: ReasoningStatus) => {
  if (status === "running") {
    return {
      Icon: Loader2,
      iconClassName: "text-amber-500",
      spin: true,
    };
  }
  if (status === "requires-action") {
    return {
      Icon: AlertTriangle,
      iconClassName: "text-orange-500",
      spin: false,
    };
  }
  if (status === "incomplete") {
    return {
      Icon: XCircle,
      iconClassName: "text-destructive",
      spin: false,
    };
  }
  return {
    Icon: CheckCircle2,
    iconClassName: "text-emerald-600 dark:text-emerald-400",
    spin: false,
  };
};

export function ReasoningOutline({
  title,
  statusLabel,
  status,
  className,
  contentClassName,
  children,
}: ReasoningOutlineProps) {
  const { Icon, iconClassName, spin } = resolveStatusAppearance(status);

  return (
    <details className={cn("group min-w-0 overflow-hidden rounded-lg border border-border/60 bg-background/60 p-2.5", className)}>
      <summary className="flex min-w-0 cursor-pointer list-none items-center gap-1.5 text-xs text-muted-foreground [&::-webkit-details-marker]:hidden">
        <ChevronDown className="h-3.5 w-3.5 shrink-0 transition-transform duration-150 group-open:rotate-180" />
        <Brain className="h-3.5 w-3.5 shrink-0" />
        <span className="truncate">{title}</span>
        <span className={cn("ml-auto shrink-0", iconClassName)} title={statusLabel} aria-label={statusLabel}>
          <Icon className={cn("h-3.5 w-3.5", spin && "animate-spin")} />
          <span className="sr-only">{statusLabel}</span>
        </span>
      </summary>
      <div className={cn("mt-2 min-w-0 space-y-2 border-l border-border/50 pl-2", contentClassName)}>{children}</div>
    </details>
  );
}
