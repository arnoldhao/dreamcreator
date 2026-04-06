import * as React from "react";
import { makeAssistantToolUI } from "@assistant-ui/react";

import { ApprovalCard } from "@/components/tool-ui/approval-card";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { requestGateway } from "@/shared/realtime";

export const EXEC_APPROVAL_TOOL_NAME = "exec_approval";

type ExecApprovalArgs = {
  id?: string;
  toolName?: string;
  action?: string;
  args?: string;
};

type ExecApprovalResult = {
  decision?: string;
  reason?: string;
};

const normalizeDecision = (value: unknown): "approve" | "deny" | "" => {
  if (typeof value !== "string") {
    return "";
  }
  const normalized = value.trim().toLowerCase();
  if (normalized === "approve" || normalized === "approved") {
    return "approve";
  }
  if (normalized === "deny" || normalized === "denied" || normalized === "reject" || normalized === "rejected") {
    return "deny";
  }
  return "";
};

export const ExecApprovalToolUI = makeAssistantToolUI<ExecApprovalArgs, ExecApprovalResult>({
  toolName: EXEC_APPROVAL_TOOL_NAME,
  render: ({ args, result }) => {
    const { t } = useI18n();
    const approvalID = typeof args?.id === "string" ? args.id.trim() : "";
    const actionName = typeof args?.action === "string" ? args.action.trim() : "";
    const toolName = typeof args?.toolName === "string" ? args.toolName.trim() : "";
    const toolLabel = toolName
      ? t(`settings.tools.builtin.${toolName}.name`)
      : t("chat.tools.unknown");
    const decision = normalizeDecision(result?.decision);

    const resolveApproval = React.useCallback(
      async (nextDecision: "approve" | "deny") => {
        if (!approvalID) {
          return;
        }
        try {
          await requestGateway("exec.approval.resolve", {
            id: approvalID,
            decision: nextDecision,
            reason: nextDecision === "approve" ? "approved by aui" : "denied by aui",
          });
        } catch (error) {
          messageBus.publishToast({
            intent: "danger",
            title: t("chat.tools.approvalTool.resolveError"),
            description: error instanceof Error ? error.message : String(error ?? ""),
            source: "gateway",
          });
        }
      },
      [approvalID, t]
    );

    const isResolved = decision !== "";
    const choice = decision === "approve" ? "approved" : decision === "deny" ? "denied" : undefined;

    if (!approvalID && !isResolved) {
      return (
        <div className="py-1">
          <div className="text-xs text-destructive">
            {t("chat.tools.approvalTool.error")}
          </div>
        </div>
      );
    }

    const title = actionName || t("chat.tools.approvalTool.title");
    const description = `${t("chat.tools.approvalTool.tool")}: ${toolLabel}`;

    return (
      <ApprovalCard
        id={`exec-approval-${approvalID || "resolved"}`}
        title={title}
        description={description}
        icon="shield-check"
        confirmLabel={t("chat.tools.approvalTool.approveAction")}
        cancelLabel={t("chat.tools.approvalTool.denyAction")}
        choice={choice}
        onConfirm={!isResolved ? () => resolveApproval("approve") : undefined}
        onCancel={!isResolved ? () => resolveApproval("deny") : undefined}
      />
    );
  },
});
