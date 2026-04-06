import { useEffect, useMemo, useRef, useState, type CSSProperties } from "react";
import { Events } from "@wailsio/runtime";
import { useAssistantApi, useAssistantState } from "@assistant-ui/react";

import {
  CalendarClock,
  ChevronDown,
  ChevronUp,
	FolderOpen,
	Library as LibraryIcon,
	MessageCircle,
	Trash2,
} from "lucide-react";

import { MAIN_SIDEBAR_FOOTER, type MainRouteId } from "@/app/routes/main";
import type { SettingsRouteId } from "@/app/routes/settings";
import { isSettingsSection, setPendingSettingsSection } from "@/app/settings/sectionStorage";
import { ChatMainPage } from "@/features/chat";
import { CronPage } from "@/features/cron";
import { LibraryPage } from "@/features/library";
import { NotificationsPage, NotificationsPanel } from "@/features/notifications/NotificationsCenter";
import { useShowSettingsWindow } from "@/shared/query/settings";
import { useExternalTools, useExternalToolUpdates } from "@/shared/query/externalTools";
import { useNoticeUnreadCount } from "@/shared/query/notices";
import { useOpenAssistantWorkspaceDirectory } from "@/shared/query/workspace";
import { resolvePersistedThreadId } from "@/shared/assistant/thread-identities";
import { useUpdateStore } from "@/shared/store/update";
import { AppShell } from "@/components/layout/AppShell";
import type { SidebarConfig } from "@/components/layout/AppSidebar";
import { PageHeader } from "@/shared/ui/PageHeader";
import { Button } from "@/shared/ui/button";
import { Separator } from "@/shared/ui/separator";
import { messageBus } from "@/shared/message";
import { useI18n } from "@/shared/i18n";
import { TaskDialog } from "@/features/library/components/TaskDialog";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import { useThreadStore } from "@/shared/store/threads";

type MainViewId = MainRouteId | "notifications";

const MAIN_PAGES: Record<MainViewId, { title: string; description: string; view: JSX.Element }> = {
  library: {
    title: "Library",
    description: "Manage assets and references in one place.",
    view: <LibraryPage />,
  },
  cron: {
    title: "Cron",
    description: "Scheduler jobs, run history, and execution status.",
    view: <CronPage />,
  },
  chat: {
    title: "Chat",
    description: "Conversations and collaboration.",
    view: <ChatMainPage />,
  },
  notifications: {
    title: "",
    description: "",
    view: <NotificationsPage />,
  },
};

type ActiveTarget =
  | { type: "page"; id: MainViewId }
  | { type: "thread"; id: string };

const normalizeToolVersion = (version?: string, toolName?: string) => {
  let value = (version ?? "").trim();
  if (!value) {
    return "";
  }
  value = value.replace(/^v/i, "");
  if (toolName?.toLowerCase() === "ffmpeg") {
    value = value.replace(/^n-/i, "");
    value = value.replace(/-tessus$/i, "");
  }
  return value;
};

export function MainApp() {
  const { t } = useI18n();
  const { enabled: assistantUiEnabled } = useAssistantUiMode();
  const api = useAssistantApi();
  const activeThread = useAssistantState(({ threadListItem }) => threadListItem);
  const messageCount = useAssistantState(({ thread }) => thread.messages.length);
  const activeThreadStatus = useAssistantState(({ threadListItem }) => threadListItem.status);
  const activeThreadResolvedId = useAssistantState(({ threadListItem }) =>
    resolvePersistedThreadId(threadListItem.remoteId, threadListItem.id)
  );
  const threadMeta = useThreadStore((state) =>
    state.threads[resolvePersistedThreadId(activeThread.remoteId, activeThread.id)]
  );
  const assistantWorkspaceId = threadMeta?.assistantId ?? "";
  const showSettingsWindow = useShowSettingsWindow();
  const updateInfo = useUpdateStore((state) => state.info);
  const externalTools = useExternalTools();
  const externalToolUpdates = useExternalToolUpdates();
  const noticeUnreadCount = useNoticeUnreadCount("center");
  const openAssistantWorkspace = useOpenAssistantWorkspaceDirectory();

  const [activeTarget, setActiveTarget] = useState<ActiveTarget>(() =>
    assistantUiEnabled ? { type: "page", id: "chat" } : { type: "page", id: "library" }
  );
  const [isChatMenuOpen, setIsChatMenuOpen] = useState(false);
  const chatMenuRef = useRef<HTMLDivElement | null>(null);
  const pendingNewThreadIdRef = useRef<string | null>(null);
  const [isNewChatPending, setIsNewChatPending] = useState(false);
  const [isNoticePanelOpen, setIsNoticePanelOpen] = useState(false);
  const visibleActiveTarget = assistantUiEnabled
    ? activeTarget
    : activeTarget.type === "page" && activeTarget.id === "notifications"
      ? activeTarget
      : ({ type: "page", id: "library" } as const);
  const activeRoute = visibleActiveTarget.type === "page" ? visibleActiveTarget.id : "chat";
  const activePage = useMemo(() => MAIN_PAGES[activeRoute], [activeRoute]);

  useEffect(() => {
    if (!isChatMenuOpen) {
      return;
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (!chatMenuRef.current || chatMenuRef.current.contains(event.target as Node)) {
        return;
      }
      setIsChatMenuOpen(false);
    };
    document.addEventListener("pointerdown", handlePointerDown);
    return () => document.removeEventListener("pointerdown", handlePointerDown);
  }, [isChatMenuOpen]);

  useEffect(() => {
    if (activeRoute !== "chat" && isChatMenuOpen) {
      setIsChatMenuOpen(false);
    }
  }, [activeRoute, isChatMenuOpen]);

  useEffect(() => {
    if (!assistantUiEnabled) {
      return;
    }
    const offNavigate = Events.On("chat:navigate", (event: any) => {
      const threadId = event?.data ?? event ?? "";
      setIsNoticePanelOpen(false);
      if (threadId) {
        setActiveTarget({ type: "thread", id: threadId });
        void api.threads().switchToThread(threadId);
        return;
      }
      setActiveTarget({ type: "page", id: "chat" });
      void api.threads().switchToNewThread();
    });
    return () => {
      offNavigate();
    };
  }, [api, assistantUiEnabled]);

  useEffect(() => {
    const offNavigate = Events.On("main:navigate", (event: any) => {
      const nextTarget = typeof event?.data === "string" ? event.data : event;
      switch (nextTarget) {
        case "notifications":
          setIsNoticePanelOpen(false);
          setActiveTarget({ type: "page", id: "notifications" });
          return;
        case "library":
        case "cron":
          setIsNoticePanelOpen(false);
          setActiveTarget({ type: "page", id: nextTarget });
          return;
        case "chat":
          setIsNoticePanelOpen(false);
          setActiveTarget({ type: "page", id: "chat" });
          return;
        default:
          return;
      }
    });
    return () => {
      offNavigate();
    };
  }, []);

  const handleNewThread = () => {
    if (!assistantUiEnabled) {
      return;
    }
    setIsNoticePanelOpen(false);
    pendingNewThreadIdRef.current = activeThread.id;
    setIsNewChatPending(true);
    setActiveTarget({ type: "page", id: "chat" });
    void api.threads().switchToNewThread();
  };

  const handleSelectMainRoute = (route: MainRouteId) => {
    if (!assistantUiEnabled && route !== "library") {
      setActiveTarget({ type: "page", id: "library" });
      return;
    }
    setIsNoticePanelOpen(false);
    if (route === "chat") {
      handleNewThread();
      return;
    }
    setActiveTarget({ type: "page", id: route });
  };

  const handleSelectThread = (threadId: string) => {
    if (!assistantUiEnabled || !threadId) {
      return;
    }
    setIsNoticePanelOpen(false);
    setIsNewChatPending(false);
    pendingNewThreadIdRef.current = null;
    setActiveTarget({ type: "thread", id: threadId });
  };

  const handleSelectSettingsRoute = (route: SettingsRouteId) => {
    if (isSettingsSection(route)) {
      setPendingSettingsSection(route);
    }
    showSettingsWindow.mutate();
  };

  const handleOpenNotifications = () => {
    setIsNoticePanelOpen(false);
    setActiveTarget({ type: "page", id: "notifications" });
  };

  const handleToggleNoticePanel = () => {
    setIsNoticePanelOpen((open) => !open);
  };

  useEffect(() => {
    if (!assistantUiEnabled) {
      return;
    }
    if (activeTarget.type === "page" && activeTarget.id !== "chat") {
      return;
    }
    if (isNewChatPending) {
      const pendingId = pendingNewThreadIdRef.current;
      if (pendingId && activeThread.id === pendingId && activeThreadStatus !== "new") {
        return;
      }
      if (activeThreadStatus === "new") {
        return;
      }
      if (activeThreadResolvedId) {
        setActiveTarget({ type: "thread", id: activeThreadResolvedId });
        setIsNewChatPending(false);
        pendingNewThreadIdRef.current = null;
      }
      return;
    }
    if (activeThreadStatus === "new") {
      if (!(activeTarget.type === "page" && activeTarget.id === "chat")) {
        setActiveTarget({ type: "page", id: "chat" });
      }
      return;
    }
    if (activeThreadResolvedId && (activeTarget.type !== "thread" || activeTarget.id !== activeThreadResolvedId)) {
      setActiveTarget({ type: "thread", id: activeThreadResolvedId });
    }
  }, [activeTarget, activeThread.id, activeThreadResolvedId, activeThreadStatus, assistantUiEnabled, isNewChatPending]);

  useEffect(() => {
    if (
      !assistantUiEnabled &&
      (activeTarget.type !== "page" || (activeTarget.id !== "library" && activeTarget.id !== "notifications"))
    ) {
      setActiveTarget({ type: "page", id: "library" });
    }
  }, [activeTarget, assistantUiEnabled]);

  const handleOpenWorkspace = () => {
    if (!assistantWorkspaceId || openAssistantWorkspace.isPending) {
      return;
    }
    openAssistantWorkspace.mutate(assistantWorkspaceId);
  };

  const handleDeleteThread = () => {
    messageBus.publishDialog({
      intent: "danger",
      title: "Delete thread",
      description: "This will remove the thread and its messages.",
      confirmLabel: "Delete",
      cancelLabel: "Cancel",
      onConfirm: () => {
        const threadItem = api.threadListItem();
        threadItem.detach();
        void threadItem.delete();
      },
    });
  };

  const sidebarConfig = useMemo<SidebarConfig>(
    () => ({
      navItems: assistantUiEnabled
        ? [
            { id: "chat", label: "Chat", icon: MessageCircle },
            { id: "library", label: "Library", icon: LibraryIcon },
            { id: "cron", label: "Cron", icon: CalendarClock },
          ]
        : [{ id: "library", label: "Library", icon: LibraryIcon }],
      footer: MAIN_SIDEBAR_FOOTER,
    }),
    [assistantUiEnabled]
  );

  const defaultThreadTitle = t("chat.header.untitled");
  const activeThreadTitle =
    threadMeta?.title?.trim() || activeThread.title?.trim() || defaultThreadTitle;
  const activeTitle =
    activeRoute === "chat"
      ? activeTarget.type === "thread"
        ? activeThreadTitle
        : t("sidebar.nav.chat")
      : activeRoute === "notifications"
        ? t("notifications.page.title")
      : t(`sidebar.nav.${activeRoute}`);

  const chatSubtitle = t("chat.header.messageCount");
  const resolvedChatSubtitle = chatSubtitle
    .replace("{count}", String(messageCount))
    .replace("{suffix}", t("chat.header.messageSuffix"));

  const isDashboardWorkspacePage =
    activeRoute === "library" || activeRoute === "cron" || activeRoute === "notifications";
  const isMainFamilyPage = isDashboardWorkspacePage || activeRoute === "chat";
  const showTitleBarBorder = !isMainFamilyPage;

  const titleContent =
    activeRoute === "chat" ? (
      <div className="flex min-w-0 items-center gap-3">
        <div className="flex min-w-0 items-center gap-1">
          <div className="min-w-0 flex-1 truncate font-display text-sm font-semibold text-foreground">
            {activeTitle}
          </div>
          <div
            className="relative flex items-center"
            ref={chatMenuRef}
            style={{ "--wails-draggable": "no-drag" } as CSSProperties}
          >
            <Button
              type="button"
              variant="ghost"
              size="compactIcon"
              className="h-6 w-6 shrink-0 border-0 focus-visible:ring-0 focus-visible:ring-offset-0"
              onClick={() => setIsChatMenuOpen((open) => !open)}
              aria-label={t("chat.header.menu")}
            >
              {isChatMenuOpen ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
            </Button>
            {isChatMenuOpen ? (
              <div className="absolute left-0 top-full z-50 mt-2 w-[220px] rounded-md border bg-popover p-1 text-sm shadow-md">
                <div className="flex items-center gap-2 px-2 py-1.5 text-muted-foreground">
                  <MessageCircle className="h-4 w-4" />
                  <span className="truncate">{resolvedChatSubtitle}</span>
                </div>
                <Separator className="my-1" />
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left hover:bg-accent hover:text-accent-foreground disabled:pointer-events-none disabled:opacity-50"
                  onClick={() => {
                    handleOpenWorkspace();
                    setIsChatMenuOpen(false);
                  }}
                  disabled={!assistantWorkspaceId}
                >
                  <FolderOpen className="h-4 w-4" />
                  <span>{t("chat.header.openWorkspace")}</span>
                </button>
                <button
                  type="button"
                  className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-left text-destructive hover:bg-accent/60 hover:text-destructive"
                  onClick={() => {
                    handleDeleteThread();
                    setIsChatMenuOpen(false);
                  }}
                >
                  <Trash2 className="h-4 w-4" />
                  <span>{t("sidebar.thread.delete")}</span>
                </button>
              </div>
            ) : null}
          </div>
        </div>
      </div>
    ) : activeRoute === "cron" ? (
      <div className="min-w-0 truncate pl-1 font-display text-lg font-bold text-foreground">
        {activeTitle}
      </div>
    ) : undefined;

  const isAppUpdateAvailable =
    updateInfo.status === "available" ||
    updateInfo.status === "downloading" ||
    updateInfo.status === "installing" ||
    updateInfo.status === "ready_to_restart";

  const isExternalToolsUpdateAvailable = useMemo(() => {
    const tools = externalTools.data ?? [];
    const updates = externalToolUpdates.data ?? [];
    if (tools.length === 0 || updates.length === 0) {
      return false;
    }
    const updateMap = new Map<string, (typeof updates)[number]>();
    updates.forEach((item) => {
      if (item?.name) {
        updateMap.set(item.name, item);
      }
    });
    return tools.some((tool) => {
      if (tool.status !== "installed") {
        return false;
      }
      const update = updateMap.get(tool.name);
      const current = normalizeToolVersion(tool.version, tool.name);
      const latest = normalizeToolVersion(update?.latestVersion, tool.name);
      return Boolean(current && latest && current !== latest);
    });
  }, [externalTools.data, externalToolUpdates.data]);

  const activeView = activeRoute === "chat" ? <ChatMainPage /> : activePage.view;

  return (
    <AppShell
      activeWindow="main"
      title={activeRoute === "chat" ? "" : activeTitle}
      titleContent={titleContent}
      onOpenSettings={() => showSettingsWindow.mutate()}
      activeMainRoute={
        visibleActiveTarget.type === "page" && visibleActiveTarget.id !== "notifications"
          ? visibleActiveTarget.id
          : undefined
      }
      onSelectMainRoute={handleSelectMainRoute}
      onSelectSettingsRoute={handleSelectSettingsRoute}
      onOpenNotifications={handleOpenNotifications}
      noticeUnreadCount={noticeUnreadCount.data ?? 0}
      isNoticePanelOpen={isNoticePanelOpen}
      onToggleNoticePanel={handleToggleNoticePanel}
      onSelectThread={assistantUiEnabled ? handleSelectThread : undefined}
      highlightThreadActive={assistantUiEnabled && visibleActiveTarget.type === "thread"}
      showThreadList={assistantUiEnabled}
      isAppUpdateAvailable={isAppUpdateAvailable}
      isExternalToolsUpdateAvailable={isExternalToolsUpdateAvailable}
      showTitleBarBorder={showTitleBarBorder}
      contentScrollable={!isMainFamilyPage}
      contentClassName={
        activeRoute === "chat"
          ? "px-[var(--app-sidebar-padding)] pb-[var(--app-sidebar-padding)] pt-1"
          : isMainFamilyPage
            ? "px-5 pb-5 pt-1"
            : undefined
      }
      sidebarConfig={sidebarConfig}
      onNewThread={assistantUiEnabled ? handleNewThread : undefined}
    >
      {activeRoute === "chat" ? (
        <div className="flex h-full min-h-0 flex-1 flex-col">{activeView}</div>
      ) : (
        <div
          className={
            isDashboardWorkspacePage
              ? "flex w-full min-h-0 flex-1 flex-col space-y-6"
              : "mx-auto w-full max-w-4xl space-y-6"
          }
        >
          {isDashboardWorkspacePage ? null : (
            <PageHeader title={activePage.title} description={activePage.description} />
          )}
          {activeView}
        </div>
      )}
      <TaskDialog />
      {isNoticePanelOpen ? (
        <NotificationsPanel
          onViewAll={handleOpenNotifications}
          onClose={() => setIsNoticePanelOpen(false)}
        />
      ) : null}
    </AppShell>
  );
}
