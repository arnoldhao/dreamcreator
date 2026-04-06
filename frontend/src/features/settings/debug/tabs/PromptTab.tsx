import { useEffect, useMemo, useState } from "react";
import { ChevronDown, ChevronUp } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { SETTINGS_ROW_CLASS, SettingsListCard, SettingsSeparator } from "@/shared/ui/settings-layout";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";

import type { PromptTabProps } from "../types";
import { RunSummaryCard } from "./RunSummaryCard";

const PROMPT_VIEW_VALUES = ["system", "history", "user", "assistant", "tool"] as const;
type PromptViewValue = (typeof PROMPT_VIEW_VALUES)[number];

export function PromptTab({
  t,
  selectedRunId,
  setSelectedRunId,
  runSummaries,
  runEventsLoading,
  runEventsError,
  selectedPromptRun,
  formatDateTime,
  statusLabelClass,
  formatRunStatus,
}: PromptTabProps) {
  const [toolsExpanded, setToolsExpanded] = useState(false);
  const [skillsExpanded, setSkillsExpanded] = useState(false);
  const [activeView, setActiveView] = useState<PromptViewValue>("system");
  const toolItems = useMemo(() => selectedPromptRun?.payload.tools ?? [], [selectedPromptRun]);
  const skillItems = useMemo(() => selectedPromptRun?.payload.skills ?? [], [selectedPromptRun]);
  const promptMessages = useMemo(() => {
    const source = Array.isArray(selectedPromptRun?.payload.messages) ? selectedPromptRun?.payload.messages ?? [] : [];
    return source
      .map((item, index) => ({
        key: `msg-${index}`,
        role: String(item?.role ?? "").trim().toLowerCase(),
        content: String(item?.content ?? ""),
        reasoning: String(item?.reasoning ?? ""),
        toolCallId: String(item?.toolCallId ?? "").trim(),
      }))
      .filter((item) => item.role || item.content || item.reasoning || item.toolCallId);
  }, [selectedPromptRun]);
  const userMessages = useMemo(() => promptMessages.filter((item) => item.role === "user"), [promptMessages]);
  const assistantMessages = useMemo(() => promptMessages.filter((item) => item.role === "assistant"), [promptMessages]);
  const toolMessages = useMemo(() => promptMessages.filter((item) => item.role === "tool"), [promptMessages]);
  const historyMessages = useMemo(() => promptMessages.filter((item) => item.role !== "system"), [promptMessages]);
  const metaRowClassName = `${SETTINGS_ROW_CLASS} py-1.5`;
  const renderCompactGridList = (items: string[]) => (
    <div className="max-h-32 overflow-y-auto">
      <div className="grid grid-cols-2 gap-1.5 sm:grid-cols-4 xl:grid-cols-8">
        {items.map((item) => (
          <div
            key={item}
            className="min-w-0 rounded border border-border/60 bg-card px-2 py-1 text-[11px] text-foreground"
            title={item}
          >
            <span className="block truncate font-mono">{item}</span>
          </div>
        ))}
      </div>
    </div>
  );

  useEffect(() => {
    setToolsExpanded(false);
    setSkillsExpanded(false);
    setActiveView("system");
  }, [selectedPromptRun?.runId]);

  const renderMessageList = (
    items: Array<{ key: string; role: string; content: string; reasoning: string; toolCallId: string }>,
    emptyText: string
  ) => {
    if (items.length === 0) {
      return <div className="text-sm text-muted-foreground">{emptyText}</div>;
    }
    return (
      <div className="space-y-2">
        {items.map((item) => (
          <div key={item.key} className="rounded-md border border-border/60 bg-muted/20 p-2">
            <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
              <span className="font-mono uppercase">{item.role || "unknown"}</span>
              {item.toolCallId ? (
                <span className="max-w-[70%] truncate font-mono" title={item.toolCallId}>
                  {t("settings.debug.prompt.messages.toolCallId")}: {item.toolCallId}
                </span>
              ) : null}
            </div>
            {item.content ? (
              <pre className="mt-2 max-h-40 overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-all rounded-md border border-border/50 bg-card p-2 text-[11px] text-foreground">
                {item.content}
              </pre>
            ) : null}
            {item.reasoning ? (
              <div className="mt-2">
                <div className="text-[11px] text-muted-foreground">
                  {t("settings.debug.prompt.messages.reasoning")}
                </div>
                <pre className="mt-1 max-h-32 overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-all rounded-md border border-border/50 bg-card p-2 text-[11px] text-foreground">
                  {item.reasoning}
                </pre>
              </div>
            ) : null}
          </div>
        ))}
      </div>
    );
  };

  return (
    <TabsContent value="prompt" className="mt-0 space-y-3">
      <RunSummaryCard
        t={t}
        runSummaries={runSummaries}
        selectedRunId={selectedRunId}
        setSelectedRunId={setSelectedRunId}
        formatDateTime={formatDateTime}
        statusLabelClass={statusLabelClass}
        formatRunStatus={formatRunStatus}
        emptyText={t("settings.debug.trace.empty")}
      />

      {runEventsLoading ? (
        <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">{t("common.loading")}</div>
      ) : runEventsError ? (
        <div className="rounded-lg border bg-card p-4 text-sm text-destructive">
          {t("settings.debug.trace.error")}
        </div>
      ) : !selectedPromptRun ? (
        <div className="rounded-lg border bg-card p-4 text-sm text-muted-foreground">
          {selectedRunId === "all"
            ? t("settings.debug.prompt.empty")
            : t("settings.debug.prompt.emptyForRun")}
        </div>
      ) : (
        <>
          <SettingsListCard contentClassName="text-xs">
            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.meta.runId")}</div>
              <div className="max-w-[70%] break-all text-right font-mono text-[11px] text-foreground" title={selectedPromptRun.runId}>
                {selectedPromptRun.runId}
              </div>
            </div>
            <SettingsSeparator />

            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.meta.mode")}</div>
              <div className="max-w-[70%] truncate text-right text-foreground" title={selectedPromptRun.payload.mode || "-"}>
                {selectedPromptRun.payload.mode || "-"}
              </div>
            </div>
            <SettingsSeparator />

            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.meta.generatedAt")}</div>
              <div
                className="max-w-[70%] truncate text-right text-foreground"
                title={formatDateTime(selectedPromptRun.payload.report?.generatedAt)}
              >
                {formatDateTime(selectedPromptRun.payload.report?.generatedAt)}
              </div>
            </div>
            <SettingsSeparator />

            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.meta.promptChars")}</div>
              <div className="text-right font-mono text-foreground">
                {selectedPromptRun.payload.promptChars ?? selectedPromptRun.payload.prompt?.length ?? 0}
              </div>
            </div>
            <SettingsSeparator />

            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.tools")}</div>
              <Button
                type="button"
                variant="ghost"
                size="compact"
                className="h-auto px-2 py-0.5 text-xs"
                onClick={() => setToolsExpanded((current) => !current)}
                aria-label={
                  toolsExpanded
                    ? t("settings.debug.prompt.collapseList")
                    : t("settings.debug.prompt.expandList")
                }
              >
                <span className="font-mono">{toolItems.length}</span>
                {toolsExpanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
              </Button>
            </div>
            {toolsExpanded ? (
              <div className="mt-1.5 rounded-md border border-border/60 bg-muted/20 p-2">
                {toolItems.length === 0 ? (
                  <div className="text-[11px] text-muted-foreground">{t("settings.debug.prompt.listEmpty")}</div>
                ) : (
                  renderCompactGridList(toolItems)
                )}
              </div>
            ) : null}
            <SettingsSeparator />

            <div className={metaRowClassName}>
              <div className="text-sm font-medium text-muted-foreground">{t("settings.debug.prompt.skills")}</div>
              <Button
                type="button"
                variant="ghost"
                size="compact"
                className="h-auto px-2 py-0.5 text-xs"
                onClick={() => setSkillsExpanded((current) => !current)}
                aria-label={
                  skillsExpanded
                    ? t("settings.debug.prompt.collapseList")
                    : t("settings.debug.prompt.expandList")
                }
              >
                <span className="font-mono">{skillItems.length}</span>
                {skillsExpanded ? <ChevronUp className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
              </Button>
            </div>
            {skillsExpanded ? (
              <div className="mt-1.5 rounded-md border border-border/60 bg-muted/20 p-2">
                {skillItems.length === 0 ? (
                  <div className="text-[11px] text-muted-foreground">{t("settings.debug.prompt.listEmpty")}</div>
                ) : (
                  renderCompactGridList(skillItems)
                )}
              </div>
            ) : null}
          </SettingsListCard>

          <Tabs
            value={activeView}
            onValueChange={(value) => setActiveView(value as PromptViewValue)}
            className="space-y-3"
          >
            <TabsList>
              <TabsTrigger value="system">
                {t("settings.debug.prompt.view.system")}
              </TabsTrigger>
              <TabsTrigger value="history">
                {t("settings.debug.prompt.view.history")}
              </TabsTrigger>
              <TabsTrigger value="user">
                {t("settings.debug.prompt.view.user")}
              </TabsTrigger>
              <TabsTrigger value="assistant">
                {t("settings.debug.prompt.view.assistant")}
              </TabsTrigger>
              <TabsTrigger value="tool">
                {t("settings.debug.prompt.view.tool")}
              </TabsTrigger>
            </TabsList>

            <TabsContent value="system" className="mt-0">
              <div className="rounded-lg border bg-card">
                <div className="border-b border-border/70 px-3 py-2 text-sm font-medium text-muted-foreground">
                  {t("settings.debug.prompt.system")}
                </div>
                <div className="p-3">
                  <pre className="max-h-[32rem] overflow-y-auto overflow-x-hidden whitespace-pre-wrap break-all rounded-md border bg-card p-3 text-xs text-foreground">
                    {selectedPromptRun.payload.prompt || t("settings.debug.prompt.systemEmpty")}
                  </pre>
                </div>
              </div>
            </TabsContent>

            <TabsContent value="history" className="mt-0">
              <div className="rounded-lg border bg-card">
                <div className="border-b border-border/70 px-3 py-2 text-sm font-medium text-muted-foreground">
                  {t("settings.debug.prompt.view.history")}
                </div>
                <div className="max-h-[32rem] overflow-y-auto overflow-x-hidden p-3 text-xs">
                  {promptMessages.length === 0
                    ? (
                        <div className="text-muted-foreground">
                          {t("settings.debug.prompt.messages.unavailable")}
                        </div>
                      )
                    : renderMessageList(
                        historyMessages,
                        t("settings.debug.prompt.messages.empty")
                      )}
                </div>
              </div>
            </TabsContent>

            <TabsContent value="user" className="mt-0">
              <div className="rounded-lg border bg-card">
                <div className="border-b border-border/70 px-3 py-2 text-sm font-medium text-muted-foreground">
                  {t("settings.debug.prompt.view.user")}
                </div>
                <div className="max-h-[32rem] overflow-y-auto overflow-x-hidden p-3 text-xs">
                  {promptMessages.length === 0
                    ? (
                        <div className="text-muted-foreground">
                          {t("settings.debug.prompt.messages.unavailable")}
                        </div>
                      )
                    : renderMessageList(
                        userMessages,
                        t("settings.debug.prompt.messages.empty")
                      )}
                </div>
              </div>
            </TabsContent>

            <TabsContent value="assistant" className="mt-0">
              <div className="rounded-lg border bg-card">
                <div className="border-b border-border/70 px-3 py-2 text-sm font-medium text-muted-foreground">
                  {t("settings.debug.prompt.view.assistant")}
                </div>
                <div className="max-h-[32rem] overflow-y-auto overflow-x-hidden p-3 text-xs">
                  {promptMessages.length === 0
                    ? (
                        <div className="text-muted-foreground">
                          {t("settings.debug.prompt.messages.unavailable")}
                        </div>
                      )
                    : renderMessageList(
                        assistantMessages,
                        t("settings.debug.prompt.messages.empty")
                      )}
                </div>
              </div>
            </TabsContent>

            <TabsContent value="tool" className="mt-0">
              <div className="rounded-lg border bg-card">
                <div className="border-b border-border/70 px-3 py-2 text-sm font-medium text-muted-foreground">
                  {t("settings.debug.prompt.view.tool")}
                </div>
                <div className="max-h-[32rem] overflow-y-auto overflow-x-hidden p-3 text-xs">
                  {promptMessages.length === 0
                    ? (
                        <div className="text-muted-foreground">
                          {t("settings.debug.prompt.messages.unavailable")}
                        </div>
                      )
                    : renderMessageList(
                        toolMessages,
                        t("settings.debug.prompt.messages.empty")
                      )}
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </>
      )}
    </TabsContent>
  );
}
