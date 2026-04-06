"use client";

import * as React from "react";
import { DASHBOARD_FIELD_SURFACE_CLASS } from "@/shared/ui/dashboard";
import { cn } from "./_adapter";
import type { ApprovalCardProps, ApprovalDecision } from "./schema";
import { ActionButtons } from "../shared/action-buttons";
import { type Action } from "../shared/schema";

import { AlertTriangle, icons, Check, X, Wrench } from "lucide-react";

type LucideIcon = React.ComponentType<{ className?: string }>;

function getLucideIcon(name: string): LucideIcon | null {
  const pascalName = name
    .split("-")
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join("");

  const Icon = icons[pascalName as keyof typeof icons];
  return Icon ?? null;
}

interface ApprovalCardReceiptProps {
  id: string;
  title: string;
  choice: ApprovalDecision;
  actionLabel?: string;
  className?: string;
}

function ApprovalCardReceipt({
  id,
  title,
  choice,
  actionLabel,
  className,
}: ApprovalCardReceiptProps) {
  const isApproved = choice === "approved";
  const displayLabel = actionLabel ?? (isApproved ? "Approved" : "Denied");

  return (
    <div
      className={cn(
        "flex w-full min-w-0 max-w-full flex-col",
        "text-xs text-foreground",
        "motion-safe:animate-in motion-safe:fade-in motion-safe:blur-in-sm motion-safe:zoom-in-95 motion-safe:duration-300 motion-safe:ease-out motion-safe:fill-mode-both",
        className,
      )}
      data-slot="approval-card"
      data-tool-ui-id={id}
      data-receipt="true"
      role="status"
      aria-label={displayLabel}
    >
      <div
        className={cn(
          "flex w-full items-center gap-2 px-2.5 py-2",
          DASHBOARD_FIELD_SURFACE_CLASS,
        )}
      >
        <span
          className={cn(
            "flex size-7 shrink-0 items-center justify-center rounded-full bg-muted",
            isApproved ? "text-primary" : "text-muted-foreground",
          )}
        >
          {isApproved ? <Check className="size-4" /> : <X className="size-4" />}
        </span>
        <div className="flex flex-col">
          <span className="text-xs font-medium leading-snug">{displayLabel}</span>
          <span className="text-xs text-muted-foreground leading-snug">{title}</span>
        </div>
      </div>
    </div>
  );
}

export function ApprovalCard({
  id,
  title,
  description,
  icon,
  metadata,
  variant,
  confirmLabel,
  cancelLabel,
  className,
  choice,
  onConfirm,
  onCancel,
}: ApprovalCardProps) {
  const resolvedVariant = variant ?? "default";
  const resolvedConfirmLabel = confirmLabel ?? "Approve";
  const resolvedCancelLabel = cancelLabel ?? "Deny";
  const Icon = icon ? getLucideIcon(icon) : null;
  const summaryText = [title, description, ...(metadata ?? []).map((item) => `${item.key}: ${item.value}`)]
    .map((value) => value?.trim())
    .filter(Boolean)
    .join(" · ");

  const handleAction = React.useCallback(
    async (actionId: string) => {
      if (actionId === "confirm") {
        await onConfirm?.();
      } else if (actionId === "cancel") {
        await onCancel?.();
      }
    },
    [onConfirm, onCancel],
  );

  const handleKeyDown = React.useCallback(
    (event: React.KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault();
        onCancel?.();
      }
    },
    [onCancel],
  );

  const isDestructive = resolvedVariant === "destructive";

  const actions: Action[] = [
    {
      id: "cancel",
      label: resolvedCancelLabel,
      variant: "ghost",
    },
    {
      id: "confirm",
      label: resolvedConfirmLabel,
      variant: isDestructive ? "destructive" : "default",
    },
  ];

  const viewKey = choice ? `receipt-${choice}` : "interactive";

  return (
    <div key={viewKey} className="contents">
      {choice ? (
        <ApprovalCardReceipt
          id={id}
          title={title}
          choice={choice}
          className={className}
        />
      ) : (
        <article
          className={cn(
            "flex w-full min-w-0 max-w-full flex-col gap-2",
            "text-xs text-foreground",
            className,
          )}
          data-slot="approval-card"
          data-tool-ui-id={id}
          role="dialog"
          aria-labelledby={`${id}-title`}
          onKeyDown={handleKeyDown}
        >
          <div
            className={cn(
              "flex w-full min-w-0 items-center gap-2 px-2.5 py-2",
              DASHBOARD_FIELD_SURFACE_CLASS,
            )}
            title={summaryText || undefined}
          >
            {Icon ? (
              <Icon className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            ) : (
              <Wrench className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            )}
            <h2
              id={`${id}-title`}
              className="min-w-0 flex-1 truncate text-xs font-medium leading-snug"
            >
              {summaryText || title}
            </h2>
            <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-orange-500" aria-hidden />
          </div>
          <div className="@container/actions flex justify-end pt-0.5">
            <ActionButtons actions={actions} onAction={handleAction} className="ml-auto w-fit" />
          </div>
        </article>
      )}
    </div>
  );
}
