import { useEffect, useMemo, useState, type ReactNode } from "react";

import { Button } from "@/shared/ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs";

import type { PromptTabProps } from "../types";

const PROMPT_VIEW_VALUES = ["system", "history", "user", "assistant", "tool", "meta"] as const;
type PromptViewValue = (typeof PROMPT_VIEW_VALUES)[number];
const DEFAULT_META_LIST_COUNT = 6;

function PromptContentCard(props: { children: ReactNode; className?: string }) {
  return (
    <div className={`flex h-full min-h-0 flex-col rounded-lg border bg-card ${props.className ?? ""}`}>
      {props.children}
    </div>
  );
}

export function PromptTab({
  t,
  selectedRunId,
  runEventsLoading,
  runEventsError,
  selectedPromptRun,
  formatDateTime,
}: PromptTabProps) {
  const [activeView, setActiveView] = useState<PromptViewValue>("system");
  const [toolsExpanded, setToolsExpanded] = useState(false);
  const [skillsExpanded, setSkillsExpanded] = useState(false);

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

  useEffect(() => {
    setActiveView("system");
    setToolsExpanded(false);
    setSkillsExpanded(false);
  }, [selectedPromptRun?.runId]);

  const renderCompactGridList = (items: string[]) => {
    if (items.length === 0) {
      return <div className="text-sm text-muted-foreground">{t("settings.debug.prompt.listEmpty")}</div>;
    }

    return (
      <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <div
            key={item}
            className="min-w-0 rounded border border-border/60 bg-background px-2 py-1.5 text-[11px] text-foreground"
            title={item}
          >
            <span className="block truncate font-mono">{item}</span>
          </div>
        ))}
      </div>
    );
  };

  const renderMetaCollectionSection = (
    title: string,
    items: string[],
    expanded: boolean,
    onToggle: () => void
  ) => {
    if (items.length === 0) {
      return null;
    }

    const visibleItems = expanded ? items : items.slice(0, DEFAULT_META_LIST_COUNT);
    const canToggle = items.length > DEFAULT_META_LIST_COUNT;

    return (
      <section className="space-y-2">
        <div className="flex items-center justify-between gap-3">
          <div className="text-xs font-medium text-muted-foreground">
            {title} ({items.length})
          </div>
          {canToggle ? (
            <Button size="compact" variant="ghost" className="h-7 px-2 text-[11px]" onClick={onToggle}>
              {expanded ? t("settings.debug.prompt.collapseList") : t("settings.debug.prompt.expandList")}
            </Button>
          ) : null}
        </div>
        {renderCompactGridList(visibleItems)}
      </section>
    );
  };

  const renderMessageList = (
    items: Array<{ key: string; role: string; content: string; reasoning: string; toolCallId: string }>,
    emptyText: string
  ) => {
    if (promptMessages.length === 0) {
      return <div className="text-sm text-muted-foreground">{t("settings.debug.prompt.messages.unavailable")}</div>;
    }
    if (items.length === 0) {
      return <div className="text-sm text-muted-foreground">{emptyText}</div>;
    }

    return (
      <div className="space-y-2">
        {items.map((item) => (
          <div key={item.key} className="rounded-md border border-border/60 bg-muted/20 p-3">
            <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
              <span className="font-mono uppercase">{item.role || "unknown"}</span>
              {item.toolCallId ? (
                <span className="max-w-[70%] truncate font-mono" title={item.toolCallId}>
                  {t("settings.debug.prompt.messages.toolCallId")}: {item.toolCallId}
                </span>
              ) : null}
            </div>
            {item.content ? (
              <pre className="mt-2 overflow-x-hidden whitespace-pre-wrap break-all rounded-md border border-border/50 bg-background p-3 text-[11px] text-foreground">
                {item.content}
              </pre>
            ) : null}
            {item.reasoning ? (
              <div className="mt-2">
                <div className="text-[11px] text-muted-foreground">{t("settings.debug.prompt.messages.reasoning")}</div>
                <pre className="mt-1 overflow-x-hidden whitespace-pre-wrap break-all rounded-md border border-border/50 bg-background p-3 text-[11px] text-foreground">
                  {item.reasoning}
                </pre>
              </div>
            ) : null}
          </div>
        ))}
      </div>
    );
  };

  const renderEmptyState = (content: ReactNode) => (
    <TabsContent value="prompt" className="mt-0 flex h-full min-h-0 flex-1 flex-col">
      <div className="flex min-h-0 flex-1 items-center justify-center rounded-lg border border-border/70 bg-background/70 px-4 text-sm text-muted-foreground">
        {content}
      </div>
    </TabsContent>
  );

  if (runEventsLoading) {
    return renderEmptyState(t("common.loading"));
  }

  if (runEventsError) {
    return renderEmptyState(<span className="text-destructive">{t("settings.debug.trace.error")}</span>);
  }

  if (!selectedPromptRun) {
    return renderEmptyState(
      selectedRunId === "all" ? t("settings.debug.prompt.empty") : t("settings.debug.prompt.emptyForRun")
    );
  }

  return (
    <TabsContent value="prompt" className="mt-0 flex h-full min-h-0 flex-1 flex-col">
      <Tabs
        value={activeView}
        onValueChange={(value) => setActiveView(value as PromptViewValue)}
        className="flex h-full min-h-0 flex-1 flex-col gap-3"
      >
        <div className="flex min-w-0 flex-nowrap items-center overflow-x-auto pb-1 -mb-1">
          <TabsList className="h-auto shrink-0 rounded-lg bg-muted/60 p-1">
            <TabsTrigger value="system" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.system")}
            </TabsTrigger>
            <TabsTrigger value="history" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.history")}
            </TabsTrigger>
            <TabsTrigger value="user" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.user")}
            </TabsTrigger>
            <TabsTrigger value="assistant" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.assistant")}
            </TabsTrigger>
            <TabsTrigger value="tool" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.tool")}
            </TabsTrigger>
            <TabsTrigger value="meta" className="min-w-0 data-[state=active]:shadow-none">
              {t("settings.debug.prompt.view.meta")}
            </TabsTrigger>
          </TabsList>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          <TabsContent value="system" className="mt-0 h-full">
            <PromptContentCard>
              <div className="min-h-0 flex-1 overflow-auto p-3">
                <pre className="min-h-full overflow-x-hidden whitespace-pre-wrap break-all rounded-md border bg-background p-3 text-xs text-foreground">
                  {selectedPromptRun.payload.prompt || t("settings.debug.prompt.systemEmpty")}
                </pre>
              </div>
            </PromptContentCard>
          </TabsContent>

          <TabsContent value="history" className="mt-0 h-full">
            <PromptContentCard>
              <div className="min-h-0 flex-1 overflow-auto p-3 text-xs">
                {renderMessageList(historyMessages, t("settings.debug.prompt.messages.empty"))}
              </div>
            </PromptContentCard>
          </TabsContent>

          <TabsContent value="user" className="mt-0 h-full">
            <PromptContentCard>
              <div className="min-h-0 flex-1 overflow-auto p-3 text-xs">
                {renderMessageList(userMessages, t("settings.debug.prompt.messages.empty"))}
              </div>
            </PromptContentCard>
          </TabsContent>

          <TabsContent value="assistant" className="mt-0 h-full">
            <PromptContentCard>
              <div className="min-h-0 flex-1 overflow-auto p-3 text-xs">
                {renderMessageList(assistantMessages, t("settings.debug.prompt.messages.empty"))}
              </div>
            </PromptContentCard>
          </TabsContent>

          <TabsContent value="tool" className="mt-0 h-full">
            <PromptContentCard>
              <div className="min-h-0 flex-1 overflow-auto p-3 text-xs">
                {renderMessageList(toolMessages, t("settings.debug.prompt.messages.empty"))}
              </div>
            </PromptContentCard>
          </TabsContent>

          <TabsContent value="meta" className="mt-0 h-full">
            <PromptContentCard>
              <div className="flex h-full min-h-0 flex-col gap-3 overflow-auto p-3">
                <div className="overflow-hidden rounded-md border border-border/60 bg-background/60">
                  <div className="divide-y divide-border/70">
                    <div className="flex items-center justify-between gap-3 px-4 py-3 text-xs">
                      <span className="text-muted-foreground">{t("settings.debug.prompt.meta.runId")}</span>
                      <span className="max-w-[70%] break-all text-right font-mono text-foreground">
                        {selectedPromptRun.runId}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-3 px-4 py-3 text-xs">
                      <span className="text-muted-foreground">{t("settings.debug.prompt.meta.mode")}</span>
                      <span className="max-w-[70%] truncate text-right text-foreground">
                        {selectedPromptRun.payload.mode || "-"}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-3 px-4 py-3 text-xs">
                      <span className="text-muted-foreground">{t("settings.debug.prompt.meta.generatedAt")}</span>
                      <span className="max-w-[70%] truncate text-right text-foreground">
                        {formatDateTime(selectedPromptRun.payload.report?.generatedAt)}
                      </span>
                    </div>
                    <div className="flex items-center justify-between gap-3 px-4 py-3 text-xs">
                      <span className="text-muted-foreground">{t("settings.debug.prompt.meta.promptChars")}</span>
                      <span className="font-mono text-foreground">
                        {selectedPromptRun.payload.promptChars ?? selectedPromptRun.payload.prompt?.length ?? 0}
                      </span>
                    </div>
                  </div>
                </div>

                {toolItems.length === 0 && skillItems.length === 0 ? (
                  <div className="text-sm text-muted-foreground">{t("settings.debug.prompt.meta.emptyCollections")}</div>
                ) : (
                  <div className="grid gap-4 lg:grid-cols-2">
                    {renderMetaCollectionSection(
                      t("settings.debug.prompt.tools"),
                      toolItems,
                      toolsExpanded,
                      () => setToolsExpanded((current) => !current)
                    )}
                    {renderMetaCollectionSection(
                      t("settings.debug.prompt.skills"),
                      skillItems,
                      skillsExpanded,
                      () => setSkillsExpanded((current) => !current)
                    )}
                  </div>
                )}
              </div>
            </PromptContentCard>
          </TabsContent>
        </div>
      </Tabs>
    </TabsContent>
  );
}
