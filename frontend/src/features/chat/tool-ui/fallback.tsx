import * as React from "react";
import type { ToolCallMessagePart, ToolCallMessagePartStatus } from "@assistant-ui/react";

import { ToolFallback } from "@/components/assistant-ui/tool-fallback";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { requestGateway } from "@/shared/realtime";
import { Button } from "@/shared/ui/button";
import { DASHBOARD_PANEL_SURFACE_CLASS } from "@/shared/ui/dashboard";

export type ToolUIFallbackCardProps = {
  toolName?: string;
  toolCallId?: string;
  args?: unknown;
  argsText?: string;
  result?: unknown;
  isError?: boolean;
  status?: ToolCallMessagePartStatus;
  interrupt?: ToolCallMessagePart["interrupt"];
  addResult?: (result: unknown) => void;
  resume?: (payload: unknown) => void;
};

const resolveArgsText = (argsText: string | undefined, args: unknown): string => {
  const explicit = (argsText ?? "").trim();
  if (explicit) {
    return explicit;
  }
  if (typeof args === "string") {
    return args;
  }
  if (args !== undefined) {
    try {
      return JSON.stringify(args, null, 2);
    } catch {
      return String(args);
    }
  }
  return "{}";
};

const resolveStatus = (
  status: ToolCallMessagePartStatus | undefined,
  isError: boolean | undefined,
  result: unknown
): ToolCallMessagePartStatus => {
  if (status) {
    return status;
  }
  if (isError) {
    return { type: "incomplete", reason: "error" };
  }
  if (result === undefined) {
    return { type: "running" };
  }
  return { type: "complete" };
};

type ApprovalInterruptPayload = {
  id: string;
  toolName: string;
  action: string;
  args: string;
  status: string;
  decision: string;
};

const firstString = (record: Record<string, unknown>, keys: string[]): string => {
  for (const key of keys) {
    const value = record[key];
    if (typeof value !== "string") {
      continue;
    }
    const trimmed = value.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return "";
};

const normalizeDecision = (value: string): "approve" | "deny" | "" => {
  const normalized = value.trim().toLowerCase();
  if (normalized === "approve" || normalized === "approved") {
    return "approve";
  }
  if (normalized === "deny" || normalized === "denied" || normalized === "reject" || normalized === "rejected") {
    return "deny";
  }
  return "";
};

const resolveApprovalInterrupt = (
  interrupt: ToolCallMessagePart["interrupt"] | undefined
): ApprovalInterruptPayload | null => {
  if (!interrupt || interrupt.type !== "human") {
    return null;
  }
  const payload = interrupt.payload;
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    return null;
  }
  const record = payload as Record<string, unknown>;
  const id = firstString(record, ["id", "approvalId", "approvalID"]);
  if (!id) {
    return null;
  }
  return {
    id,
    toolName: firstString(record, ["toolName", "tool", "name"]),
    action: firstString(record, ["action", "title"]),
    args: firstString(record, ["args", "input", "request"]),
    status: firstString(record, ["status"]),
    decision: firstString(record, ["decision"]),
  };
};

export function ToolUIFallbackCard({
  toolName,
  toolCallId,
  args,
  argsText,
  result,
  isError,
  status,
  interrupt,
}: ToolUIFallbackCardProps) {
  const { t } = useI18n();
  const resolvedToolName = (toolName ?? "").trim() || "tool";
  const resolvedStatus = resolveStatus(status, isError, result);
  const approval = React.useMemo(() => resolveApprovalInterrupt(interrupt), [interrupt]);
  const approvalID = approval?.id ?? "";
  const approvalDecision = normalizeDecision(approval?.decision ?? "");
  const requiresApprovalAction =
    resolvedStatus.type === "requires-action" && approvalID !== "" && approvalDecision === "";
  const isCancelled =
    resolvedStatus.type === "incomplete" && resolvedStatus.reason === "cancelled";

  const [open, setOpen] = React.useState(false);
  const [pendingDecision, setPendingDecision] = React.useState<"approve" | "deny" | "">("");

  React.useEffect(() => {
    if (!requiresApprovalAction) {
      setPendingDecision("");
    }
  }, [requiresApprovalAction, approvalID]);

  const resolveApproval = React.useCallback(
    async (decision: "approve" | "deny") => {
      if (!approvalID || pendingDecision) {
        return;
      }
      setPendingDecision(decision);
      try {
        await requestGateway("exec.approval.resolve", {
          id: approvalID,
          decision,
          reason: decision === "approve" ? "approved by aui fallback" : "denied by aui fallback",
        });
      } catch (error) {
        setPendingDecision("");
        messageBus.publishToast({
          intent: "danger",
          title: t("chat.tools.approvalTool.resolveError"),
          description: error instanceof Error ? error.message : String(error ?? ""),
          source: "gateway",
        });
      }
    },
    [approvalID, pendingDecision, t]
  );

  return (
    <ToolFallback.Root
      className={cn(
        DASHBOARD_PANEL_SURFACE_CLASS,
        isCancelled && "border-muted-foreground/30 bg-muted/30"
      )}
      open={open}
      onOpenChange={setOpen}
    >
      <ToolFallback.Trigger toolName={resolvedToolName} status={resolvedStatus} />
      {requiresApprovalAction ? (
        <div className="flex items-center justify-end gap-2 px-4 pt-2">
          <Button
            type="button"
            size="sm"
            variant="default"
            disabled={pendingDecision !== ""}
            onClick={() => {
              void resolveApproval("approve");
            }}
          >
            {t("chat.tools.approvalTool.approveAction")}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="outline"
            disabled={pendingDecision !== ""}
            onClick={() => {
              void resolveApproval("deny");
            }}
          >
            {t("chat.tools.approvalTool.denyAction")}
          </Button>
        </div>
      ) : null}
      <ToolFallback.Content>
        <ToolFallback.Error status={resolvedStatus} />
        <ToolFallback.Args
          argsText={resolveArgsText(argsText, args)}
          className={cn(isCancelled && "opacity-60")}
        />
        {!isCancelled && <ToolFallback.Result result={result} />}
      </ToolFallback.Content>
    </ToolFallback.Root>
  );
}
