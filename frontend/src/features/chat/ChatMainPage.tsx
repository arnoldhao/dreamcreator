import * as React from "react";
import { code as streamdownCode } from "@streamdown/code";
import { math as streamdownMath } from "@streamdown/math";
import {
  ActionBarPrimitive,
  BranchPickerPrimitive,
  ComposerPrimitive,
  ErrorPrimitive,
  INTERNAL,
  MessagePartPrimitive,
  MessagePrimitive,
  ThreadPrimitive,
  useAssistantApi,
  useAssistantState,
} from "@assistant-ui/react";
import { ChatStreamdownTextPrimitive } from "./lib/chat-streamdown";
import {
  BellRing,
  Bot,
  CalendarClock,
  Check,
  Cpu,
  ChevronsDown,
  ChevronDown,
  Copy,
  Pencil,
  RotateCcw,
} from "lucide-react";

import { UserMessageAttachments } from "@/components/assistant-ui/attachment";
import { ReasoningOutline } from "@/components/assistant-ui/reasoning";
import { SourceOutline, SourcesOutline, resolveSourceHost } from "@/components/assistant-ui/sources";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { normalizeMarkdown } from "@/shared/markdown/normalize";
import { chatFeatureFlags } from "@/shared/assistant/feature-flags";
import { useThreadStore } from "@/shared/store/threads";
import { Button } from "@/shared/ui/button";
import { ChatWelcome } from "./components/ChatWelcome";
import { ComposerBar, useComposerRunConfig, useRuntimeSelections } from "./components/ComposerBar";
import { ToolUIFallbackCard } from "./tool-ui/fallback";
import { isNonCollapsibleToolUI, ToolUIRegistry } from "./tool-ui/registry";
import { CHAT_CODE_LANGUAGES, UniversalCodeSyntaxHighlighter } from "./lib/streamdown-code";

type ChatGroupKey = "reasoning" | "tool-call" | "tool-call-inline" | "source" | undefined;

type ChatPartGroup = {
  groupKey: ChatGroupKey;
  indices: number[];
};

const SmoothStreamdownTextPrimitive =
  INTERNAL.withSmoothContextProvider(ChatStreamdownTextPrimitive);

type ChatTimelinePart = {
  type?: string;
  parentId?: string;
  toolName?: string;
};

type ChatSourcePart = {
  type?: string;
  id?: string;
  title?: string;
  url?: string;
};

type ChatSystemNoticePayload = {
  origin: string;
  source: string;
  kind: string;
  reason: string;
  runId: string;
};

const parseMaybeJSONData = (value: unknown): unknown => {
  if (typeof value !== "string") {
    return value;
  }
  const trimmed = value.trim();
  if (!(trimmed.startsWith("{") || trimmed.startsWith("["))) {
    return value;
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    return value;
  }
};

const parseSystemNoticePayload = (data: unknown): ChatSystemNoticePayload => {
  if (!data || typeof data !== "object" || Array.isArray(data)) {
    return {
      origin: "",
      source: "",
      kind: "",
      reason: "",
      runId: "",
    };
  }
  const payload = data as Record<string, unknown>;
  return {
    origin: typeof payload.origin === "string" ? payload.origin.trim().toLowerCase() : "",
    source: typeof payload.source === "string" ? payload.source.trim().toLowerCase() : "",
    kind: typeof payload.kind === "string" ? payload.kind.trim().toLowerCase() : "",
    reason: typeof payload.reason === "string" ? payload.reason.trim() : "",
    runId: typeof payload.runId === "string" ? payload.runId.trim() : "",
  };
};

function SystemNoticeDataPart({ data }: { name: string; data: unknown }) {
  const { t } = useI18n();
  const payload = React.useMemo(() => parseSystemNoticePayload(data), [data]);

  let icon = <BellRing className="h-3 w-3" />;
  let label = t("chat.message.systemNotice.system");

  if (payload.kind === "cron_reminder" || payload.source === "cron") {
    icon = <CalendarClock className="h-3 w-3" />;
    label = t("chat.message.systemNotice.cron");
  } else if (payload.kind === "exec_result" || payload.source === "exec") {
    icon = <Cpu className="h-3 w-3" />;
    label = t("chat.message.systemNotice.exec");
  } else if (payload.kind === "subagent_result" || payload.source === "subagent") {
    icon = <Bot className="h-3 w-3" />;
    label = t("chat.message.systemNotice.subagent");
  }

  const tooltip = payload.reason || payload.runId || undefined;

  return (
    <div
      className="mb-1 inline-flex max-w-full items-center gap-1.5 rounded-full border border-border/60 bg-muted/40 px-2 py-0.5 text-[10px] font-medium text-muted-foreground"
      title={tooltip}
    >
      {icon}
      <span className="truncate">{label}</span>
    </div>
  );
}

function GenericDataPart({ name, data }: { name: string; data: unknown }) {
  const parsed = React.useMemo(() => parseMaybeJSONData(data), [data]);
  const label = name.trim() || "data";
  const detail = React.useMemo(() => {
    if (parsed == null) {
      return "";
    }
    if (typeof parsed === "string") {
      return parsed.trim();
    }
    try {
      return JSON.stringify(parsed, null, 2);
    } catch {
      return String(parsed);
    }
  }, [parsed]);
  if (!detail) {
    return null;
  }
  return (
    <details className="rounded-lg border border-border/60 bg-muted/30 p-2">
      <summary className="cursor-pointer text-[11px] font-medium text-muted-foreground">
        {label}
      </summary>
      <pre className="mt-2 max-h-40 overflow-auto whitespace-pre-wrap break-words text-[11px] text-foreground [overflow-wrap:anywhere]">
        {detail}
      </pre>
    </details>
  );
}

const normalizeParentID = (part: ChatTimelinePart | undefined) => {
  const parentId = part?.parentId;
  if (typeof parentId !== "string") {
    return "";
  }
  return parentId.trim();
};

const resolvePartGroupKey = (
  parts: readonly ChatTimelinePart[],
  indices: number[]
): ChatGroupKey => {
  const first = parts[indices[0]];
  const partType = first?.type;
  if (partType === "reasoning") {
    return "reasoning";
  }
  if (partType === "tool-call") {
    const hasCustomToolUI = indices.some((index) =>
      isNonCollapsibleToolUI(parts[index]?.toolName)
    );
    return hasCustomToolUI ? "tool-call-inline" : "tool-call";
  }
  if (partType === "source") {
    return "source";
  }
  return undefined;
};

const isToolFallbackPart = (part: ChatTimelinePart | undefined) => {
  if (!part || part.type !== "tool-call") {
    return false;
  }
  return !isNonCollapsibleToolUI(part.toolName);
};

const groupChatMessageParts = (parts: readonly ChatTimelinePart[]): ChatPartGroup[] => {
  const groups: ChatPartGroup[] = [];
  for (let i = 0; i < parts.length; ) {
    const currentType = parts[i]?.type;
    if (currentType === "source") {
      const indices: number[] = [];
      let cursor = i;
      while (cursor < parts.length && parts[cursor]?.type === "source") {
        indices.push(cursor);
        cursor += 1;
      }
      groups.push({ groupKey: resolvePartGroupKey(parts, indices), indices });
      i = cursor;
      continue;
    }

    if (currentType === "tool-call") {
      if (!isToolFallbackPart(parts[i])) {
        groups.push({ groupKey: resolvePartGroupKey(parts, [i]), indices: [i] });
        i += 1;
        continue;
      }

      const indices: number[] = [];
      let cursor = i;
      while (cursor < parts.length && isToolFallbackPart(parts[cursor])) {
        indices.push(cursor);
        cursor += 1;
      }
      groups.push({
        groupKey: resolvePartGroupKey(parts, indices),
        indices,
      });
      i = cursor;
      continue;
    }

    const parentId = normalizeParentID(parts[i]);
    if (parentId) {
      const indices: number[] = [];
      let cursor = i;
      while (cursor < parts.length && normalizeParentID(parts[cursor]) === parentId) {
        indices.push(cursor);
        cursor += 1;
      }
      groups.push({
        groupKey: resolvePartGroupKey(parts, indices),
        indices,
      });
      i = cursor;
      continue;
    }

    if (currentType === "reasoning") {
      const indices: number[] = [];
      let cursor = i;
      while (
        cursor < parts.length &&
        !normalizeParentID(parts[cursor]) &&
        parts[cursor]?.type === "reasoning"
      ) {
        indices.push(cursor);
        cursor += 1;
      }
      groups.push({ groupKey: resolvePartGroupKey(parts, indices), indices });
      i = cursor;
      continue;
    }
    groups.push({ groupKey: undefined, indices: [i] });
    i += 1;
  }
  return groups;
};

const useAutoOpenDetails = (active: boolean) => {
  const [open, setOpen] = React.useState(active);
  const wasActive = React.useRef(active);
  React.useEffect(() => {
    if (active && !wasActive.current) {
      setOpen(true);
    }
    wasActive.current = active;
  }, [active]);
  return { open, setOpen };
};

function ReasoningGroupPanel({
  children,
  indices,
}: React.PropsWithChildren<{ indices: number[] }>) {
  const { t } = useI18n();
  const reasoningMessageStatus = useAssistantState(({ message }) => {
    if (message.role !== "assistant") {
      return "complete";
    }
    return (message.status?.type ?? "complete").toLowerCase();
  });
  const isReasoningStreaming = useAssistantState(({ message }) => {
    if (message.role !== "assistant" || message.status?.type !== "running") {
      return false;
    }
    let lastReasoningIndex = -1;
    for (let index = message.parts.length - 1; index >= 0; index -= 1) {
      if (message.parts[index]?.type === "reasoning") {
        lastReasoningIndex = index;
        break;
      }
    }
    if (lastReasoningIndex < 0) {
      return false;
    }
    const start = indices[0] ?? 0;
    const end = indices[indices.length - 1] ?? start;
    return lastReasoningIndex >= start && lastReasoningIndex <= end;
  });
  const reasoningStatusType = isReasoningStreaming
    ? "running"
    : reasoningMessageStatus === "requires-action"
      ? "requires-action"
      : reasoningMessageStatus === "incomplete"
        ? "incomplete"
        : "complete";
  const reasoningStatusLabel = reasoningStatusType === "running"
    ? t("chat.tools.state.calling")
    : reasoningStatusType === "requires-action"
      ? t("chat.tools.state.actionRequired")
      : reasoningStatusType === "incomplete"
        ? t("chat.tools.state.failed")
        : t("chat.tools.state.completed");
  const hasLeadingGap = (indices[0] ?? 0) > 0;

  return (
    <ReasoningOutline
      title={t("chat.message.reasoning")}
      status={reasoningStatusType}
      statusLabel={reasoningStatusLabel}
      className={hasLeadingGap ? "mt-2" : undefined}
    >
      {children}
    </ReasoningOutline>
  );
}

function ToolGroupPanel({
  children,
  indices,
}: React.PropsWithChildren<{ indices: number[] }>) {
  const hasLeadingGap = (indices[0] ?? 0) > 0;
  return (
    <div className={cn("min-w-0 space-y-2", hasLeadingGap ? "mt-2" : "")}>
      {children}
    </div>
  );
}

function SourceGroupPanel({
  children,
  indices,
}: React.PropsWithChildren<{ indices: number[] }>) {
  const { t } = useI18n();
  const { open, setOpen } = useAutoOpenDetails(false);
  const hasLeadingGap = (indices[0] ?? 0) > 0;
  const messageParts = useAssistantState(({ message }) => message.parts);
  const sourcePreviews = React.useMemo(() => {
    const items: Array<{ key: string; title: string; host: string }> = [];
    for (const index of indices) {
      const part = messageParts[index] as ChatSourcePart | undefined;
      if (!part || part.type !== "source") {
        continue;
      }
      const url = typeof part.url === "string" ? part.url.trim() : "";
      if (!url) {
        continue;
      }
      const title = typeof part.title === "string" ? part.title.trim() : "";
      const id = typeof part.id === "string" ? part.id.trim() : "";
      const host = resolveSourceHost(url);
      items.push({
        key: id || `${url}:${index}`,
        title,
        host,
      });
    }
    return items;
  }, [indices, messageParts]);

  return (
    <SourcesOutline
      title={t("chat.message.sources")}
      count={indices.length}
      previews={sourcePreviews}
      open={open}
      onToggle={setOpen}
      className={hasLeadingGap ? "mt-2" : undefined}
    >
      {children}
    </SourcesOutline>
  );
}

function SourcePart({
  title,
  url,
}: {
  title?: string;
  url: string;
}) {
  return <SourceOutline title={title} url={url} />;
}

function MessagePartsGroupPanel({
  groupKey,
  indices,
  children,
}: React.PropsWithChildren<{ groupKey: string | undefined; indices: number[] }>) {
  if (!groupKey) {
    return <>{children}</>;
  }
  if (groupKey === "reasoning") {
    return <ReasoningGroupPanel indices={indices}>{children}</ReasoningGroupPanel>;
  }
  if (groupKey === "tool-call") {
    return <ToolGroupPanel indices={indices}>{children}</ToolGroupPanel>;
  }
  if (groupKey === "tool-call-inline") {
    return <>{children}</>;
  }
  if (groupKey === "source") {
    return <SourceGroupPanel indices={indices}>{children}</SourceGroupPanel>;
  }
  return <>{children}</>;
}

function StreamingIndicator({ className }: { className?: string }) {
  return (
    <span className={cn("inline-flex items-end gap-1 align-middle", className)} aria-hidden>
      {[0, 1, 2].map((dotIndex) => (
        <span
          key={dotIndex}
          className="typing-dot-wave h-2 w-2 rounded-full bg-foreground/80"
          style={{
            animationDelay: `${dotIndex * 120}ms`,
          }}
        />
      ))}
    </span>
  );
}

function MessagePartsBody() {
  const streamdownPlugins = React.useMemo(
    () => ({ code: streamdownCode, math: streamdownMath }),
    []
  );
  const streamdownComponents = React.useMemo(
    () => ({ SyntaxHighlighter: UniversalCodeSyntaxHighlighter }),
    []
  );

  return (
    <MessagePrimitive.Unstable_PartsGrouped
      groupingFunction={
        chatFeatureFlags.explainModeEnabled
          ? groupChatMessageParts
          : (parts) => parts.map((_, index) => ({ groupKey: undefined, indices: [index] }))
      }
      components={{
        Text: () => (
          <div className="min-w-0 break-words leading-relaxed [overflow-wrap:anywhere]">
            <SmoothStreamdownTextPrimitive
              className="chat-markdown"
              components={streamdownComponents}
              componentsByLanguage={CHAT_CODE_LANGUAGES}
              plugins={streamdownPlugins}
              preprocess={normalizeMarkdown}
            />
          </div>
        ),
        Reasoning: () => (
          <p className="min-w-0 whitespace-pre-wrap break-words text-xs leading-relaxed text-muted-foreground [overflow-wrap:anywhere]">
            <MessagePartPrimitive.Text />
          </p>
        ),
        Image: () => <MessagePartPrimitive.Image className="max-h-[24rem] rounded-lg border border-border/60" />,
        Source: ({ title, url }) => <SourcePart title={title} url={url} />,
        data: {
          by_name: {
            system_notice: SystemNoticeDataPart,
          },
          Fallback: GenericDataPart,
        },
        Group: MessagePartsGroupPanel,
        tools: {
          Fallback: ({ toolName, toolCallId, args, argsText, result, isError, status, interrupt, addResult, resume }) => (
            <ToolUIFallbackCard
              toolName={toolName}
              toolCallId={toolCallId}
              args={args}
              argsText={argsText}
              result={result}
              isError={isError}
              status={status}
              interrupt={interrupt}
              addResult={addResult}
              resume={resume}
            />
          ),
        },
      }}
    />
  );
}

function MessageCopyButton({
  ariaLabel,
}: {
  ariaLabel: string;
}) {
  const api = useAssistantApi();
  const { t } = useI18n();
  const hasCopyableContent = useAssistantState(({ message }) => {
    const isRunningAssistant = message.role === "assistant" && message.status?.type === "running";
    if (isRunningAssistant) {
      return false;
    }
    return message.parts.some((part) => part.type === "text" && part.text.length > 0);
  });
  const [copied, setCopied] = React.useState(false);

  React.useEffect(() => {
    if (!copied) {
      return;
    }
    const timer = window.setTimeout(() => {
      setCopied(false);
    }, 3000);
    return () => window.clearTimeout(timer);
  }, [copied]);

  const handleCopy = React.useCallback(() => {
    if (!hasCopyableContent || copied) {
      return;
    }
    const value = api.message().getCopyText();
    if (!value) {
      return;
    }
    void navigator.clipboard.writeText(value).then(() => {
      setCopied(true);
    });
  }, [api, copied, hasCopyableContent]);

  const copiedLabel = t("chat.actions.copied");
  const buttonLabel = copied ? copiedLabel : ariaLabel;

  return (
    <Button
      type="button"
      variant="ghost"
      size="compactIcon"
      className={cn(
        "h-7 w-7 rounded-md",
        copied
          ? "border border-emerald-500/25 bg-emerald-500/10 text-emerald-700 hover:bg-emerald-500/10 hover:text-emerald-700 dark:text-emerald-400 disabled:opacity-100"
          : "text-muted-foreground hover:text-foreground"
      )}
      aria-label={buttonLabel}
      title={buttonLabel}
      data-copied={copied ? "true" : undefined}
      disabled={copied || !hasCopyableContent}
      onClick={handleCopy}
    >
      {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
    </Button>
  );
}

function UserTextPart() {
  return (
    <MessagePartPrimitive.Text
      component="div"
      className="min-w-0 whitespace-pre-wrap break-words leading-relaxed [overflow-wrap:anywhere]"
    />
  );
}

function UserMessage() {
  const { t } = useI18n();
  const partCount = useAssistantState(({ message }) => message.parts.length);
  const hasAttachments = useAssistantState(
    ({ message }) => message.role === "user" && (message.attachments?.length ?? 0) > 0
  );
  const hasContent = partCount > 0 || hasAttachments;
  if (!hasContent) {
    return null;
  }

  return (
    <MessagePrimitive.Root className="w-full px-3 pb-8">
      <div className="relative ml-auto min-w-0 w-fit max-w-[80%]">
        <div className="space-y-2 rounded-2xl border border-border/70 bg-muted/45 px-3 py-2 text-sm text-foreground shadow-sm">
          {partCount > 0 ? (
            partCount === 1 ? (
              <MessagePrimitive.PartByIndex
                index={0}
                components={{
                  Text: UserTextPart,
                  Image: () => (
                    <MessagePartPrimitive.Image className="max-h-[24rem] rounded-md border border-border/60" />
                  ),
                }}
              />
            ) : (
              <MessagePrimitive.Content
                components={{
                  Text: UserTextPart,
                  Image: () => (
                    <MessagePartPrimitive.Image className="max-h-[24rem] rounded-md border border-border/60" />
                  ),
                }}
              />
            )
          ) : null}
          <UserMessageAttachments />
        </div>
        <ActionBarPrimitive.Root
          autohide="not-last"
          className="absolute -bottom-8 right-1 flex items-center justify-end gap-1"
        >
          <MessageCopyButton ariaLabel={t("chat.actions.copy")} />
          <ActionBarPrimitive.Edit asChild>
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className="h-7 w-7 rounded-md text-muted-foreground hover:text-foreground"
              aria-label={t("chat.actions.edit")}
              title={t("chat.actions.edit")}
            >
              <Pencil className="h-3.5 w-3.5" />
            </Button>
          </ActionBarPrimitive.Edit>
        </ActionBarPrimitive.Root>
      </div>
    </MessagePrimitive.Root>
  );
}

function AssistantMessage() {
  const { t } = useI18n();
  const hasContent = useAssistantState(({ message }) => message.parts.length > 0);
  const hasAssistantError = useAssistantState(
    ({ message }) => message.role === "assistant" && message.status?.type === "incomplete"
  );
  const assistantErrorText = useAssistantState(({ message }) => {
    if (message.role !== "assistant") {
      return "";
    }
    const status = message.status;
    if (!status || status.type !== "incomplete" || status.reason !== "error") {
      return "";
    }
    const value = status.error;
    if (typeof value === "string") {
      return value.trim();
    }
    if (value == null) {
      return "";
    }
    if (typeof value === "object") {
      try {
        return JSON.stringify(value);
      } catch {
        return String(value);
      }
    }
    return String(value);
  });
  if (!hasContent && !hasAssistantError) {
    return null;
  }

  return (
    <MessagePrimitive.Root className="w-full px-3">
      <ThreadPrimitive.ViewportSlack>
        <div className="mr-auto min-w-0 w-full max-w-3xl space-y-2 py-1 text-sm text-foreground">
          <MessagePartsBody />
          <MessagePrimitive.Error>
            <ErrorPrimitive.Root className="mt-2 text-xs text-destructive">
              <ErrorPrimitive.Message>
                {assistantErrorText || t("chat.message.error")}
              </ErrorPrimitive.Message>
            </ErrorPrimitive.Root>
          </MessagePrimitive.Error>
          <MessagePrimitive.If assistant>
            <div className="mt-1 flex items-center justify-between gap-2">
              <ActionBarPrimitive.Root hideWhenRunning autohide="not-last" className="flex items-center gap-1">
                <ActionBarPrimitive.Reload asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="compactIcon"
                    className="h-6 w-6 text-muted-foreground hover:text-foreground"
                    aria-label={t("chat.actions.retry")}
                  >
                    <RotateCcw className="h-3.5 w-3.5" />
                  </Button>
                </ActionBarPrimitive.Reload>
                <MessageCopyButton ariaLabel={t("chat.actions.copy")} />
              </ActionBarPrimitive.Root>
              <div className="flex items-center gap-2">
                <BranchPickerPrimitive.Root
                  hideWhenSingleBranch
                  className="flex items-center gap-1 text-[11px] text-muted-foreground"
                >
                  <BranchPickerPrimitive.Previous asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="compactIcon"
                      className="h-6 w-6 text-muted-foreground hover:text-foreground"
                      aria-label={t("chat.actions.previousResponse")}
                    >
                      <ChevronDown className="h-3.5 w-3.5 rotate-90" />
                    </Button>
                  </BranchPickerPrimitive.Previous>
                  <span className="text-[11px]">
                    <BranchPickerPrimitive.Number />/<BranchPickerPrimitive.Count />
                  </span>
                  <BranchPickerPrimitive.Next asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="compactIcon"
                      className="h-6 w-6 text-muted-foreground hover:text-foreground"
                      aria-label={t("chat.actions.nextResponse")}
                    >
                      <ChevronDown className="h-3.5 w-3.5 -rotate-90" />
                    </Button>
                  </BranchPickerPrimitive.Next>
                </BranchPickerPrimitive.Root>
              </div>
            </div>
          </MessagePrimitive.If>
        </div>
      </ThreadPrimitive.ViewportSlack>
    </MessagePrimitive.Root>
  );
}

function UserEditComposer() {
  const { t } = useI18n();
  return (
    <div className="w-full px-3">
      <ComposerPrimitive.Root className="ml-auto w-full max-w-[80%] space-y-2 rounded-2xl border border-border/70 bg-card px-3 py-2 shadow-sm">
        <ComposerPrimitive.If editing>
          <div className="text-[11px] text-muted-foreground">
            {t("chat.actions.edit")}
          </div>
        </ComposerPrimitive.If>
        <ComposerPrimitive.Input
          className="min-h-[40px] w-full resize-none bg-transparent text-sm focus-visible:outline-none"
        />
        <div className="flex items-center justify-end gap-2">
          <ComposerPrimitive.Cancel asChild>
            <Button type="button" variant="outline" size="sm">
              {t("chat.actions.closeEdit")}
            </Button>
          </ComposerPrimitive.Cancel>
          <ComposerPrimitive.Send asChild>
            <Button type="submit" variant="secondary" size="sm">
              {t("chat.composer.send")}
            </Button>
          </ComposerPrimitive.Send>
        </div>
      </ComposerPrimitive.Root>
    </div>
  );
}

export function ChatMainPage() {
  const api = useAssistantApi();
  const runtime = useRuntimeSelections();

  const activeThreadId = useAssistantState(({ threadListItem }) => threadListItem.id);
  const activeThreadRemoteId = useAssistantState(({ threadListItem }) => (threadListItem.remoteId ?? "").trim());
  const activeThreadStatus = useAssistantState(({ threadListItem }) => (threadListItem.status ?? "new").toLowerCase());
  const messageCount = useAssistantState(({ thread }) => thread.messages.length);
  const isThreadRunning = useAssistantState(({ thread }) => thread.isRunning);
  const threadMeta = useThreadStore((state) => state.threads[activeThreadRemoteId || activeThreadId]);
  const threadAssistant = runtime.assistants.find(
    (assistant) => assistant.id === (threadMeta?.assistantId?.trim() ?? "")
  ) ?? null;
  const entryAssistant =
    activeThreadStatus === "new" ? runtime.selectedAssistant : threadAssistant ?? runtime.selectedAssistant;
  const isEntryMode = messageCount === 0 && !isThreadRunning;

  useComposerRunConfig();

  React.useEffect(() => {
    if (activeThreadId) {
      return;
    }
    void api.threads().switchToNewThread();
  }, [api, activeThreadId]);

  return (
    <div className="relative flex min-h-0 flex-1 flex-col">
      <ToolUIRegistry />
      <ThreadPrimitive.ViewportProvider>
        <ThreadPrimitive.Root className="flex min-h-0 flex-1 flex-col">
          <ThreadPrimitive.Viewport
            className={cn(
              "flex min-h-0 flex-1 flex-col gap-4 py-[var(--app-sidebar-padding)]",
              isEntryMode ? "overflow-hidden" : "overflow-auto"
            )}
          >
            <ThreadPrimitive.Empty>
              <ChatWelcome
                assistant={entryAssistant}
                modelGroups={runtime.modelGroups}
                loading={runtime.assistantsLoading || runtime.providersLoading}
              >
                <ComposerBar
                  assistants={runtime.assistants}
                  assistantId={runtime.assistantId}
                  selectedAssistant={runtime.selectedAssistant}
                  setAssistantId={runtime.setAssistantId}
                  modelGroups={runtime.modelGroups}
                  agentProviderId={runtime.agentProviderId}
                  agentModelName={runtime.agentModelName}
                  loading={runtime.assistantsLoading || runtime.providersLoading}
                />
              </ChatWelcome>
            </ThreadPrimitive.Empty>
            <ThreadPrimitive.Messages
              components={{ UserMessage, AssistantMessage, UserEditComposer }}
            />
            {!isEntryMode && isThreadRunning ? (
              <div className="mr-auto max-w-3xl px-3">
                <StreamingIndicator />
              </div>
            ) : null}
            {!isEntryMode ? (
              <ThreadPrimitive.ScrollToBottom asChild>
                <Button
                  type="button"
                  size="icon"
                  variant="outline"
                  className="sticky bottom-4 ml-auto mr-1 aspect-square h-8 w-8 min-h-8 min-w-8 max-h-8 max-w-8 shrink-0 rounded-full p-0 shadow-sm"
                  aria-label="scroll to latest"
                >
                  <ChevronsDown className="h-4 w-4" />
                </Button>
              </ThreadPrimitive.ScrollToBottom>
            ) : null}
          </ThreadPrimitive.Viewport>
          {!isEntryMode ? (
            <div className="shrink-0 bg-background/72 pt-[var(--app-sidebar-padding)] animate-in fade-in-0 slide-in-from-bottom-3 duration-300">
              <ComposerBar
                assistants={runtime.assistants}
                assistantId={runtime.assistantId}
                selectedAssistant={runtime.selectedAssistant}
                setAssistantId={runtime.setAssistantId}
                modelGroups={runtime.modelGroups}
                agentProviderId={runtime.agentProviderId}
                agentModelName={runtime.agentModelName}
                loading={runtime.assistantsLoading || runtime.providersLoading}
              />
            </div>
          ) : null}
        </ThreadPrimitive.Root>
      </ThreadPrimitive.ViewportProvider>
    </div>
  );
}
