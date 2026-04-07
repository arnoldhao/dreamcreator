import * as React from "react";
import { Loader2 } from "lucide-react";

import {
  setPendingGatewayTarget,
  setPendingSettingsSection,
  type PendingGatewayTarget,
} from "@/app/settings/sectionStorage";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { useExternalTools } from "@/shared/query/externalTools";
import { useShowSettingsWindow } from "@/shared/query/settings";
import type { Assistant } from "@/shared/store/assistant";
import { Button } from "@/shared/ui/button";
import { SetupCardSection, SetupPageCard } from "@/features/setup/cards";
import { SetupStatusIcon } from "@/features/setup/cards";
import { FULL_MODE_REQUIRED_TOOLS } from "@/features/setup/constants";
import {
  SetupAgentModelCard,
  SetupProviderCard,
  SetupToolCard,
} from "@/features/setup/inline-cards";
import { buildSetupNavStatusEntries, getToolItemId } from "@/features/setup/nav";
import { resolveToolDependencyIssues } from "@/features/setup/readiness";
import { useSetupCenter } from "@/features/setup/store";
import { useSetupStatus } from "@/features/setup/useSetupStatus";

import { useChatReadiness } from "../lib/readiness";
import type { ModelGroup } from "../types";

type ChatWelcomeProps = {
  assistant: Assistant | null;
  modelGroups: ModelGroup[];
  loading: boolean;
  children: React.ReactNode;
};

function WelcomeStatusPill({
  title,
  description,
  actionLabel,
  onAction,
  className,
}: {
  title: string;
  description: string;
  actionLabel?: string;
  onAction?: () => void;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "w-full rounded-[1.75rem] border border-amber-300/45 bg-background/90 px-3.5 py-3 shadow-[0_14px_30px_-24px_rgba(180,83,9,0.9)] backdrop-blur supports-[backdrop-filter]:bg-background/78 sm:w-auto sm:min-w-[30rem] sm:max-w-[42rem]",
        className
      )}
    >
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex min-w-0 items-start gap-3">
          <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-amber-500/10">
            <SetupStatusIcon status="pending" />
          </div>
          <div className="min-w-0 space-y-0.5">
            <div className="text-sm font-medium text-foreground">{title}</div>
            <div className="text-xs leading-5 text-muted-foreground">{description}</div>
          </div>
        </div>
        {actionLabel && onAction ? (
          <Button
            type="button"
            size="compact"
            variant="outline"
            className="shrink-0 rounded-full"
            onClick={onAction}
          >
            {actionLabel}
          </Button>
        ) : null}
      </div>
    </div>
  );
}

export function ChatWelcome({
  assistant,
  modelGroups,
  loading,
  children,
}: ChatWelcomeProps) {
  const { t } = useI18n();
  const showSettingsWindow = useShowSettingsWindow();
  const readiness = useChatReadiness({ assistant, modelGroups, loading });
  const setupStatus = useSetupStatus();
  const { skippedItemIds } = useSetupCenter();
  const externalToolsQuery = useExternalTools();
  const requiredTools = FULL_MODE_REQUIRED_TOOLS;

  const missingRequiredToolNames = React.useMemo(
    () =>
      resolveToolDependencyIssues(requiredTools, externalToolsQuery.data ?? []).map((item) => item.name),
    [externalToolsQuery.data, requiredTools]
  );
  const navStatusEntries = React.useMemo(
    () =>
      buildSetupNavStatusEntries({
        languageReady: true,
        proxyReady: true,
        providersReady: setupStatus.providersReady,
        gatewayEnabled: setupStatus.gatewayEnabled,
        agentModelReady: setupStatus.agentModelReady,
        hasChosenProductMode: true,
        requiredTools,
        missingRequiredToolNames,
        skippedItemIds,
      }),
    [
      missingRequiredToolNames,
      requiredTools,
      setupStatus.agentModelReady,
      setupStatus.gatewayEnabled,
      setupStatus.providersReady,
      skippedItemIds,
    ]
  );
  const navStatusMap = React.useMemo(
    () => new Map(navStatusEntries.map((entry) => [entry.id, entry.status])),
    [navStatusEntries]
  );

  const checking = readiness.isChecking || setupStatus.checking;
  const assistantSelectionIssue = readiness.issues.includes("assistant.selection");
  const showAssistantPill = !checking && assistantSelectionIssue;
  const providerCardStatus = navStatusMap.get("ai.provider") ?? "pending";
  const agentModelCardStatus = navStatusMap.get("ai.agentModel") ?? "pending";
  const visibleToolNames = React.useMemo(
    () =>
      requiredTools.filter((name) => (navStatusMap.get(getToolItemId(name)) ?? "pending") !== "ready"),
    [navStatusMap, requiredTools]
  );
  const showAiSection = providerCardStatus !== "ready" || agentModelCardStatus !== "ready";
  const showDependencySection = visibleToolNames.length > 0;
  const showSetupPanel = checking || showAiSection || showDependencySection;
  const showBlockedPill = !checking && showSetupPanel;
  const showBottomPills = showBlockedPill || showAssistantPill;

  const openSettings = React.useCallback(
    (section: "gateway" | "provider" | "external-tools", target?: PendingGatewayTarget) => {
      if (target) {
        setPendingGatewayTarget(target);
      }
      setPendingSettingsSection(section);
      showSettingsWindow.mutate();
    },
    [showSettingsWindow]
  );

  return (
    <div className="flex h-full min-h-full w-full flex-1 justify-center px-3 py-6">
      <div className="grid h-full min-h-full w-full max-w-[56rem] grid-rows-[minmax(0,1fr)_auto_minmax(0,1fr)] gap-4">
        <div className="min-h-0 overflow-y-auto">
          {showSetupPanel ? (
            <div className="mx-auto flex min-h-full w-full max-w-[46rem] flex-col justify-end">
              {checking ? (
                <SetupPageCard
                  title={t("chat.welcome.entry.checkingTitle")}
                  headerRight={<Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />}
                  className="w-full animate-in fade-in-0 slide-in-from-bottom-4 duration-300"
                >
                  <SetupCardSection className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Loader2 className="h-3.5 w-3.5 animate-spin" />
                    <span>{t("chat.welcome.entry.checkingDescription")}</span>
                  </SetupCardSection>
                </SetupPageCard>
              ) : (
                <div className="flex flex-col gap-5 pb-4 animate-in fade-in-0 slide-in-from-bottom-4 duration-300">
                  {showAiSection ? (
                    <section className="space-y-4">
                      <div className="px-1 text-lg font-medium text-muted-foreground">
                        {t("setupCenter.steps.ai.title")}
                      </div>
                      <div className="space-y-4">
                        {providerCardStatus !== "ready" ? (
                          <SetupProviderCard status={providerCardStatus} />
                        ) : null}
                        {agentModelCardStatus !== "ready" ? (
                          <SetupAgentModelCard status={agentModelCardStatus} />
                        ) : null}
                      </div>
                    </section>
                  ) : null}

                  {showDependencySection ? (
                    <section className="space-y-4">
                      <div className="px-1 text-lg font-medium text-muted-foreground">
                        {t("setupCenter.steps.dependencies.title")}
                      </div>
                      <div className="space-y-4">
                        {visibleToolNames.map((name) => (
                          <SetupToolCard
                            key={name}
                            name={name}
                            status={navStatusMap.get(getToolItemId(name)) ?? "pending"}
                          />
                        ))}
                      </div>
                    </section>
                  ) : null}
                </div>
              )}
            </div>
          ) : null}
        </div>

        <div className="mx-auto flex w-full max-w-[46rem] items-center">
          <div className="w-full animate-in fade-in-0 slide-in-from-bottom-2 duration-300">
            {children}
          </div>
        </div>

        <div className="flex min-h-0 items-end">
          {showBottomPills ? (
            <div className="mx-auto flex w-full max-w-[46rem] justify-center animate-in fade-in-0 slide-in-from-bottom-2 duration-300">
              <div className="flex w-full flex-col items-center gap-2">
                {showBlockedPill ? (
                  <WelcomeStatusPill
                    title={t("chat.welcome.entry.blockedTitle")}
                    description={t("chat.welcome.entry.blockedDescription")}
                  />
                ) : null}

                {showAssistantPill ? (
                  <WelcomeStatusPill
                    title={t("chat.welcome.entry.assistantPill.title")}
                    description={t("chat.welcome.entry.assistantPill.description")}
                    actionLabel={t("chat.welcome.entry.assistantPill.action")}
                    onAction={() =>
                      openSettings("gateway", { view: "assistant", panelTab: "assistant" })
                    }
                  />
                ) : null}
              </div>
            </div>
          ) : (
            <div aria-hidden className="w-full" />
          )}
        </div>
      </div>
    </div>
  );
}
