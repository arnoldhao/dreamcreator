import * as React from "react";
import { Events } from "@wailsio/runtime";
import {
  Archive,
  ArrowRight,
  Bell,
  BellRing,
  CheckCheck,
  ChevronRight,
  Clock3,
  Inbox,
} from "lucide-react";

import type { Notice } from "@/shared/contracts/notice";
import {
  formatNoticeText,
  noticeSeverityToIntent,
  resolveNoticeBody,
  resolveNoticeSummary,
  resolveNoticeTitle,
} from "@/shared/contracts/notice";
import {
  useArchiveNotice,
  useMarkAllNoticesRead,
  useMarkNoticeRead,
  useNotices,
} from "@/shared/query/notices";
import { useI18n } from "@/shared/i18n";
import { cn } from "@/lib/utils";
import {
  Sheet,
  SheetContent,
  SheetTitle,
} from "@/shared/ui/sheet";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { PanelCard } from "@/shared/ui/dashboard";
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs";

type NoticeTab = "active" | "unread" | "archived";

const CATEGORY_LABEL_KEYS: Record<Notice["category"], string> = {
  heartbeat: "notifications.categories.heartbeat",
  cron: "notifications.categories.cron",
  subagent: "notifications.categories.subagent",
  exec: "notifications.categories.exec",
  gateway: "notifications.categories.gateway",
  update: "notifications.categories.update",
};

const NOTIFICATION_TABS_LIST_CLASSNAME = "grid h-8 grid-cols-3 p-0.5";
const NOTIFICATION_TABS_TRIGGER_CLASSNAME = "h-7 gap-1.5 px-2 py-0 text-xs leading-none";
const PANEL_NOTICE_BATCH = 12;
const PAGE_NOTICE_BATCH = 24;

const formatNoticeTime = (value: string, t: (key: string) => string) => {
  const parsed = Date.parse(value);
  if (!Number.isFinite(parsed)) {
    return t("common.notAvailable");
  }
  const diffMs = Date.now() - parsed;
  if (diffMs < 60_000) {
    return t("common.justNow");
  }
  return new Date(parsed).toLocaleString();
};

function parseNoticeStructuredBody(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return null;
  }
  const candidate = extractJsonCandidate(trimmed);
  if (!candidate) {
    return null;
  }
  try {
    const parsed = JSON.parse(candidate) as unknown;
    if (typeof parsed === "string") {
      return null;
    }
    if (!parsed || typeof parsed !== "object") {
      return null;
    }
    return {
      summary: resolveStructuredSummary(parsed),
      pretty: JSON.stringify(parsed, null, 2),
    };
  } catch {
    return null;
  }
}

function extractJsonCandidate(value: string) {
  const trimmed = value.trim();
  if ((trimmed.startsWith("{") && trimmed.endsWith("}")) || (trimmed.startsWith("[") && trimmed.endsWith("]"))) {
    return trimmed;
  }
  if (trimmed.startsWith("```") && trimmed.endsWith("```")) {
    const lines = trimmed.split("\n");
    if (lines.length >= 3) {
      const body = lines.slice(1, -1).join("\n").trim();
      if ((body.startsWith("{") && body.endsWith("}")) || (body.startsWith("[") && body.endsWith("]"))) {
        return body;
      }
    }
  }
  return "";
}

function resolveStructuredSummary(value: unknown) {
  if (!value || typeof value !== "object") {
    return "";
  }
  const map = value as Record<string, unknown>;
  for (const key of ["detail", "message", "summary", "error", "reason"]) {
    const text = map[key];
    if (typeof text === "string" && text.trim()) {
      return text.trim();
    }
  }
  return "";
}

const runNoticeAction = (notice: Notice) => {
  const action = notice.action;
  if (!action || !action.type) {
    return;
  }
  switch (action.type) {
    case "open_thread":
      if (action.target) {
        void Events.Emit("chat:navigate", action.target);
      }
      return;
    case "open_route":
      void Events.Emit("main:navigate", action.target || "notifications");
      return;
    default:
      if (action.target) {
        void Events.Emit("main:navigate", action.target);
      }
    }
};

function NoticeRow({
  notice,
  compact = false,
  onOpen,
}: {
  notice: Notice;
  compact?: boolean;
  onOpen?: (notice: Notice) => void;
}) {
  const { t } = useI18n();
  const markRead = useMarkNoticeRead();
  const archive = useArchiveNotice();

  const title = resolveNoticeTitle(notice, t);
  const summary = resolveNoticeSummary(notice, t);
  const body = resolveNoticeBody(notice, t);
  const structuredBody = parseNoticeStructuredBody(body);
  const timestamp = formatNoticeTime(notice.lastOccurredAt || notice.createdAt, t);
  const categoryLabel = t(CATEGORY_LABEL_KEYS[notice.category] ?? "notifications.categories.heartbeat");
  const intent = noticeSeverityToIntent(notice.severity);

  const handleOpen = () => {
    if (notice.status === "unread") {
      markRead.mutate({ ids: [notice.id], read: true });
    }
    runNoticeAction(notice);
    onOpen?.(notice);
  };

  const showAction = Boolean(notice.action?.type && notice.action?.labelKey);

  return (
    <div
      className={cn(
        "group rounded-xl border border-border/70 bg-background/70 p-3 app-motion-surface",
        compact ? "space-y-2" : "space-y-3",
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 space-y-2">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={intent === "danger" ? "destructive" : "secondary"} className="rounded-md">
              {categoryLabel}
            </Badge>
            {notice.status === "unread" ? (
              <Badge variant="secondary" className="rounded-md">
                {t("notifications.status.unread")}
              </Badge>
            ) : null}
            {notice.occurrenceCount > 1 ? (
              <Badge variant="outline" className="rounded-md">
                ×{notice.occurrenceCount}
              </Badge>
            ) : null}
          </div>
          <div className="space-y-1">
            <div className="text-sm font-semibold text-foreground">{title}</div>
            <div className="text-sm text-muted-foreground">{summary}</div>
            {body && body !== summary ? (
              structuredBody ? (
                <details className="rounded-md border border-border/60 bg-muted/20 p-2 text-xs text-muted-foreground">
                  <summary className="cursor-pointer select-none font-medium text-foreground/85">
                    {structuredBody.summary || "Structured details"}
                  </summary>
                  <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-5 text-muted-foreground">
                    {structuredBody.pretty}
                  </pre>
                </details>
              ) : (
                <div className="whitespace-pre-wrap text-xs leading-5 text-muted-foreground">{body}</div>
              )
            ) : null}
          </div>
        </div>
        <div className="flex items-center gap-1">
          {notice.status !== "archived" ? (
            <Button
              type="button"
              size="compactIcon"
              variant="ghost"
              className="h-8 w-8"
              onClick={(event) => {
                event.stopPropagation();
                archive.mutate({ ids: [notice.id], archived: true });
              }}
              aria-label={t("notifications.actions.archive")}
            >
              <Archive className="h-4 w-4" />
            </Button>
          ) : null}
        </div>
      </div>
      <div className="flex items-center justify-between gap-3">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Clock3 className="h-3.5 w-3.5" />
          <span>{timestamp}</span>
        </div>
        <div className="flex items-center gap-2">
          {showAction ? (
            <Button type="button" size="compact" variant="secondary" onClick={handleOpen}>
              {t(notice.action.labelKey)}
              <ChevronRight className="ml-1 h-3.5 w-3.5" />
            </Button>
          ) : notice.status === "unread" ? (
            <Button
              type="button"
              size="compact"
              variant="secondary"
              onClick={() => markRead.mutate({ ids: [notice.id], read: true })}
            >
              {t("notifications.actions.markRead")}
            </Button>
          ) : null}
        </div>
      </div>
    </div>
  );
}

function useNoticeList(tab: NoticeTab, limit: number) {
  return useNotices({
    statuses:
      tab === "unread"
        ? ["unread"]
        : tab === "archived"
          ? ["archived"]
          : ["unread", "read"],
    surface: "center",
    limit,
  });
}

function useNoticeFeed(tab: NoticeTab, visibleLimit: number) {
  const query = useNoticeList(tab, visibleLimit + 1);
  const rawItems = query.data ?? [];
  return {
    ...query,
    items: rawItems.slice(0, visibleLimit),
    hasMore: rawItems.length > visibleLimit,
  };
}

export function NotificationsPanel({
  onViewAll,
  onClose,
}: {
  onViewAll: () => void;
  onClose: () => void;
}) {
  const { t } = useI18n();
  const [tab, setTab] = React.useState<NoticeTab>("active");
  const [visibleLimit, setVisibleLimit] = React.useState(PANEL_NOTICE_BATCH);
  const noticesQuery = useNoticeFeed(tab, visibleLimit);
  const markAllRead = useMarkAllNoticesRead();
  const notices = noticesQuery.items;

  React.useEffect(() => {
    setVisibleLimit(PANEL_NOTICE_BATCH);
  }, [tab]);

  return (
    <Sheet open onOpenChange={(open) => (!open ? onClose() : undefined)}>
      <SheetContent side="right" showCloseButton={false} className="w-[380px] gap-0 p-4 sm:max-w-[380px]">
        <PanelCard tone="solid" className="flex min-h-0 w-full flex-1 flex-col">
          <div className="border-b border-border/70 px-4 py-3">
            <div className="space-y-2">
              <div className="min-w-0">
                <SheetTitle className="text-base font-semibold text-foreground">
                  {t("notifications.center.title")}
                </SheetTitle>
              </div>
              <Tabs value={tab} onValueChange={(value) => setTab(value as NoticeTab)} className="w-full">
                <TabsList className={cn(NOTIFICATION_TABS_LIST_CLASSNAME, "w-full")}>
                  <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="active">
                    <Inbox className="h-3.5 w-3.5" />
                    {t("notifications.tabs.active")}
                  </TabsTrigger>
                  <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="unread">
                    <BellRing className="h-3.5 w-3.5" />
                    {t("notifications.tabs.unread")}
                  </TabsTrigger>
                  <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="archived">
                    <Archive className="h-3.5 w-3.5" />
                    {t("notifications.tabs.archived")}
                  </TabsTrigger>
                </TabsList>
              </Tabs>
            </div>
          </div>
          <div className="flex items-center justify-between gap-3 px-4 pt-3">
            <Button type="button" size="compact" variant="ghost" onClick={() => markAllRead.mutate("center")}>
              <CheckCheck className="mr-1.5 h-4 w-4" />
              {t("notifications.actions.markAllRead")}
            </Button>
            <Button type="button" size="compact" variant="ghost" onClick={onViewAll}>
              {t("notifications.actions.viewAll")}
              <ArrowRight className="ml-1.5 h-4 w-4" />
            </Button>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto px-4 py-4">
            {notices.length === 0 ? (
              <div className="flex h-full min-h-[220px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/70 bg-background/40 text-center">
                <Bell className="h-6 w-6 text-muted-foreground" />
                <div className="space-y-1">
                  <div className="text-sm font-medium text-foreground">{t("notifications.empty.title")}</div>
                  <div className="max-w-[220px] text-xs leading-5 text-muted-foreground">
                    {t("notifications.empty.description")}
                  </div>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                {notices.map((notice) => (
                  <NoticeRow key={notice.id} notice={notice} compact onOpen={() => onClose()} />
                ))}
                <div className="flex items-center justify-between gap-3 pt-1">
                  <div className="text-xs text-muted-foreground">
                    {formatNoticeText(t("notifications.list.loadedCount"), {
                      count: String(notices.length),
                    })}
                  </div>
                  {noticesQuery.hasMore ? (
                    <Button
                      type="button"
                      size="compact"
                      variant="ghost"
                      onClick={() => setVisibleLimit((current) => current + PANEL_NOTICE_BATCH)}
                    >
                      {t("notifications.actions.loadMore")}
                    </Button>
                  ) : null}
                </div>
              </div>
            )}
          </div>
        </PanelCard>
      </SheetContent>
    </Sheet>
  );
}

export function NotificationsPage() {
  const { t } = useI18n();
  const [tab, setTab] = React.useState<NoticeTab>("active");
  const [visibleLimit, setVisibleLimit] = React.useState(PAGE_NOTICE_BATCH);
  const noticesQuery = useNoticeFeed(tab, visibleLimit);
  const markAllRead = useMarkAllNoticesRead();
  const notices = noticesQuery.items;

  React.useEffect(() => {
    setVisibleLimit(PAGE_NOTICE_BATCH);
  }, [tab]);

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className="grid shrink-0 grid-cols-[auto_minmax(0,1fr)] items-center gap-3">
        <Tabs value={tab} onValueChange={(value) => setTab(value as NoticeTab)} className="min-w-0 w-auto">
          <TabsList className={NOTIFICATION_TABS_LIST_CLASSNAME}>
            <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="active">
              <Inbox className="h-3.5 w-3.5" />
              {t("notifications.tabs.active")}
            </TabsTrigger>
            <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="unread">
              <BellRing className="h-3.5 w-3.5" />
              {t("notifications.tabs.unread")}
            </TabsTrigger>
            <TabsTrigger className={NOTIFICATION_TABS_TRIGGER_CLASSNAME} value="archived">
              <Archive className="h-3.5 w-3.5" />
              {t("notifications.tabs.archived")}
            </TabsTrigger>
          </TabsList>
        </Tabs>

        <div className="flex min-w-0 flex-nowrap items-center justify-end gap-2 overflow-x-auto pb-1 -mb-1">
          <Button type="button" size="compact" variant="secondary" onClick={() => markAllRead.mutate("center")}>
            <CheckCheck className="mr-1.5 h-4 w-4" />
            {t("notifications.actions.markAllRead")}
          </Button>
        </div>
      </div>

      <PanelCard tone="solid" className="flex min-h-0 flex-1 flex-col">
        <div className="min-h-0 flex-1 overflow-y-auto px-5 py-5">
          {notices.length === 0 ? (
            <div className="flex h-full min-h-[320px] flex-col items-center justify-center gap-3 rounded-xl border border-dashed border-border/70 bg-background/30 text-center">
              <Bell className="h-7 w-7 text-muted-foreground" />
              <div className="space-y-1">
                <div className="text-sm font-medium text-foreground">{t("notifications.empty.title")}</div>
                <div className="max-w-[260px] text-sm text-muted-foreground">
                  {t("notifications.empty.description")}
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              {notices.map((notice) => (
                <NoticeRow key={notice.id} notice={notice} />
              ))}
              <div className="flex items-center justify-between gap-3 border-t border-border/60 pt-4">
                <div className="text-sm text-muted-foreground">
                  {formatNoticeText(t("notifications.list.loadedCount"), {
                    count: String(notices.length),
                  })}
                </div>
                {noticesQuery.hasMore ? (
                  <Button
                    type="button"
                    size="compact"
                    variant="secondary"
                    onClick={() => setVisibleLimit((current) => current + PAGE_NOTICE_BATCH)}
                  >
                    {t("notifications.actions.loadMore")}
                  </Button>
                ) : null}
              </div>
            </div>
          )}
        </div>
      </PanelCard>
    </div>
  );
}
