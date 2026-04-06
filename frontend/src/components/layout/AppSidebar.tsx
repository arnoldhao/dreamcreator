import * as React from "react";
import { createPortal } from "react-dom";
import { System } from "@wailsio/runtime";
import {
  Archive,
  ArchiveRestore,
  ArrowUpCircle,
  Bell,
  ChevronsUpDown,
  Library,
  MessageCircle,
  Pencil,
  Search,
  Sparkles,
  Settings,
  Trash2,
  Wrench,
} from "lucide-react";
import {
  ThreadListItemPrimitive,
  ThreadListPrimitive,
  ThreadListItemRuntimeProvider,
  useAssistantApi,
  useAssistantState,
  type ThreadListItemState,
} from "@assistant-ui/react";

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/shared/ui/sidebar";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { MAIN_SIDEBAR_FOOTER } from "@/app/routes/main";
import type { MainRouteId } from "@/app/routes/main";
import type { SettingsRouteId } from "@/app/routes/settings";
import { ProductModeDialog } from "@/features/product-mode/ProductModeDialog";
import { SetupCenterDialog, SetupStatusSlot, useSetupCenter, useSetupStatus } from "@/features/setup";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import { useCurrentUserProfile } from "@/shared/query/system";
import { cn } from "@/lib/utils";
import { useAssistantUiMode } from "@/shared/store/assistantUi";
import { useSettingsStore } from "@/shared/store/settings";
import { useThreadStore } from "@/shared/store/threads";
import { UserAvatar, resolveUserDisplayName, resolveUserSubtitle } from "@/shared/ui/user-avatar";

type IconComponent = React.ComponentType<React.SVGProps<SVGSVGElement>>;

export interface SidebarNavItem {
  id: MainRouteId;
  label: string;
  icon: IconComponent;
}

export interface SidebarFooterConfig {
  menu: typeof MAIN_SIDEBAR_FOOTER.menu;
  updateAction: typeof MAIN_SIDEBAR_FOOTER.updateAction;
}

export interface SidebarConfig {
  navItems: SidebarNavItem[];
  footer: SidebarFooterConfig;
}

const DEFAULT_SIDEBAR_CONFIG: SidebarConfig = {
  navItems: [
    { id: "chat", label: "Chat", icon: MessageCircle },
    { id: "library", label: "Library", icon: Library },
  ],
  footer: MAIN_SIDEBAR_FOOTER,
};

export interface AppSidebarProps {
  activeWindow: "main" | "settings";
  reserveMacTrafficLightsGap?: boolean;
  onOpenSettings?: () => void;
  onOpenNotifications?: () => void;
  onToggleNoticePanel?: () => void;
  headerNavActions?: React.ReactNode;
  activeMainRoute?: MainRouteId;
  onSelectMainRoute?: (route: MainRouteId) => void;
  onSelectSettingsRoute?: (route: SettingsRouteId) => void;
  onSelectThread?: (threadId: string) => void;
  highlightThreadActive?: boolean;
  showThreadList?: boolean;
  isAppUpdateAvailable?: boolean;
  isExternalToolsUpdateAvailable?: boolean;
  noticeUnreadCount?: number;
  isNoticePanelOpen?: boolean;
  config?: SidebarConfig;
}

const formatWorkspaceLabel = (name?: string, assistantId?: string) => {
  const trimmedName = name?.trim();
  if (trimmedName) {
    return trimmedName;
  }
  const trimmedId = assistantId?.trim() ?? "";
  if (!trimmedId) {
    return "";
  }
  if (trimmedId.length <= 12) {
    return trimmedId;
  }
  return `${trimmedId.slice(0, 6)}…${trimmedId.slice(-4)}`;
};

type ThreadListItemRowProps = {
  onSelectThread?: (threadId: string) => void;
  highlightActive?: boolean;
};

type ThreadContextMenuProps = {
  anchor: { x: number; y: number } | null;
  archived: boolean;
  onClose: () => void;
  onRename: () => void;
  onArchiveToggle: () => void;
  onDelete: () => void;
  labels: {
    rename: string;
    archive: string;
    unarchive: string;
    delete: string;
  };
};

const parseThreadTime = (value?: string) => {
  if (!value) {
    return 0;
  }
  const parsed = Date.parse(value);
  return Number.isFinite(parsed) ? parsed : 0;
};

function ThreadContextMenu({
  anchor,
  archived,
  onClose,
  onRename,
  onArchiveToggle,
  onDelete,
  labels,
}: ThreadContextMenuProps) {
  const menuRef = React.useRef<HTMLDivElement | null>(null);
  const [position, setPosition] = React.useState<{ x: number; y: number } | null>(anchor);

  React.useEffect(() => {
    setPosition(anchor);
  }, [anchor]);

  React.useLayoutEffect(() => {
    if (!anchor || !menuRef.current) {
      return;
    }
    const rect = menuRef.current.getBoundingClientRect();
    const nextX = Math.max(8, Math.min(anchor.x, window.innerWidth - rect.width - 8));
    const nextY = Math.max(8, Math.min(anchor.y, window.innerHeight - rect.height - 8));
    if (nextX !== position?.x || nextY !== position?.y) {
      setPosition({ x: nextX, y: nextY });
    }
  }, [anchor, position?.x, position?.y]);

  React.useEffect(() => {
    if (!anchor) {
      return;
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (menuRef.current?.contains(event.target as Node)) {
        return;
      }
      onClose();
    };
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose();
      }
    };
    const handleViewportChange = () => onClose();
    document.addEventListener("pointerdown", handlePointerDown);
    window.addEventListener("keydown", handleKeyDown);
    window.addEventListener("resize", handleViewportChange);
    window.addEventListener("blur", handleViewportChange);
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown);
      window.removeEventListener("keydown", handleKeyDown);
      window.removeEventListener("resize", handleViewportChange);
      window.removeEventListener("blur", handleViewportChange);
    };
  }, [anchor, onClose]);

  if (!anchor || !position) {
    return null;
  }

  const itemClassName =
    "flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm outline-none transition-colors hover:bg-accent hover:text-accent-foreground";

  return createPortal(
    <div
      ref={menuRef}
      role="menu"
      className={cn(
        "fixed z-[120] min-w-[10rem] rounded-md border bg-popover p-1 text-popover-foreground shadow-md",
        "animate-in fade-in-0 zoom-in-95"
      )}
      style={{ left: position.x, top: position.y }}
      onContextMenu={(event) => event.preventDefault()}
    >
      <button type="button" role="menuitem" className={itemClassName} onClick={onRename}>
        <Pencil className="h-4 w-4" />
        <span>{labels.rename}</span>
      </button>
      <div className="-mx-1 my-1 h-px bg-border" />
      <button type="button" role="menuitem" className={itemClassName} onClick={onArchiveToggle}>
        {archived ? <ArchiveRestore className="h-4 w-4" /> : <Archive className="h-4 w-4" />}
        <span>{archived ? labels.unarchive : labels.archive}</span>
      </button>
      <button
        type="button"
        role="menuitem"
        className={cn(itemClassName, "text-destructive hover:text-destructive")}
        onClick={onDelete}
      >
        <Trash2 className="h-4 w-4" />
        <span>{labels.delete}</span>
      </button>
    </div>,
    document.body
  );
}

function ThreadListItemRow({
  onSelectThread,
  highlightActive = true,
}: ThreadListItemRowProps) {
  const api = useAssistantApi();
  const { t } = useI18n();
  const item = useAssistantState(({ threadListItem }) => threadListItem);
  const [isEditing, setIsEditing] = React.useState(false);
  const [draft, setDraft] = React.useState("");
  const [contextMenuAnchor, setContextMenuAnchor] = React.useState<{ x: number; y: number } | null>(null);
  const rowRef = React.useRef<HTMLDivElement | null>(null);
  const threadId = (item.remoteId ?? item.id ?? "").trim();

  const threadMeta = useThreadStore((state) => state.threads[threadId]);
  const currentTitle = (threadMeta?.title ?? item.title ?? "").trim();
  const displayTitle = currentTitle || t("sidebar.thread.untitled");

  React.useEffect(() => {
    if (isEditing) {
      setDraft(currentTitle);
    }
  }, [currentTitle, isEditing]);

  if (threadId && !threadMeta) {
    return null;
  }

  const threadStatus = (threadMeta?.status ?? item.status ?? "new").toLowerCase();
  if (threadStatus === "archived" || threadStatus === "deleted" || threadStatus === "new") {
    return null;
  }
  const canManageThread = threadStatus === "regular" || threadStatus === "archived";
  const activeClasses = highlightActive
    ? "data-[active=true]:bg-sidebar-primary/10 data-[active=true]:font-medium data-[active=true]:text-sidebar-primary"
    : "";

  const handleSelectThread = () => {
    if (!threadId) {
      return;
    }
    onSelectThread?.(threadId);
  };

  const handleCommitRename = () => {
    const trimmed = draft.trim();
    setIsEditing(false);
    if (!trimmed || trimmed === currentTitle) {
      return;
    }
    api.threadListItem().rename(trimmed);
  };

  const handleDeleteThread = React.useCallback(async () => {
    try {
      await api.threadListItem().delete();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title: t("sidebar.thread.deleteError"),
        description: message,
        intent: "warning",
      });
    }
  }, [api, t]);

  const handleArchiveToggle = React.useCallback(async () => {
    try {
      if (threadStatus === "archived") {
        await api.threadListItem().unarchive();
      } else {
        await api.threadListItem().archive();
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      messageBus.publishToast({
        title:
          threadStatus === "archived"
            ? t("sidebar.thread.unarchiveError")
            : t("sidebar.thread.archiveError"),
        description: message,
        intent: "warning",
      });
    }
  }, [api, t, threadStatus]);

  const openContextMenu = React.useCallback((x: number, y: number) => {
    setContextMenuAnchor({ x, y });
  }, []);

  const handleContextMenu = (event: React.MouseEvent) => {
    if (!canManageThread || isEditing) {
      return;
    }
    event.preventDefault();
    openContextMenu(event.clientX, event.clientY);
  };

  const handleMouseDown = (event: React.MouseEvent) => {
    if (event.button !== 2 || isEditing) {
      return;
    }
    event.preventDefault();
  };

  const handleMenuKeyDown = (event: React.KeyboardEvent) => {
    if (isEditing || !canManageThread) {
      return;
    }
    if (event.key !== "ContextMenu" && !(event.shiftKey && event.key === "F10")) {
      return;
    }
    event.preventDefault();
    const rect = rowRef.current?.getBoundingClientRect();
    openContextMenu(rect?.left ?? 24, rect?.bottom ?? 24);
  };

  return (
    <ThreadListItemPrimitive.Root
      ref={rowRef}
      className={cn(
        "flex w-full select-none items-center rounded-md px-2 py-1 text-left text-sm",
        "hover:bg-sidebar-accent hover:text-sidebar-accent-foreground",
        activeClasses
      )}
      onContextMenu={handleContextMenu}
      onMouseDown={handleMouseDown}
      onKeyDown={handleMenuKeyDown}
    >
      {isEditing ? (
        <Input
          autoFocus
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={handleCommitRename}
          onKeyDown={(event) => {
            if (event.key === "Enter") {
              event.preventDefault();
              event.currentTarget.blur();
            }
            if (event.key === "Escape") {
              event.preventDefault();
              setIsEditing(false);
            }
          }}
          size="compact"
          className="h-6 w-full select-text border-0 bg-transparent px-0 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
        />
      ) : (
        <ThreadListItemPrimitive.Trigger
          className="flex min-w-0 flex-1 items-center gap-2 text-left"
          onClick={handleSelectThread}
          >
            <div className="min-w-0 flex-1">
              <div className="truncate">{displayTitle}</div>
            </div>
          </ThreadListItemPrimitive.Trigger>
      )}
      <ThreadContextMenu
        anchor={contextMenuAnchor}
        archived={threadStatus === "archived"}
        onClose={() => setContextMenuAnchor(null)}
        onRename={() => {
          setContextMenuAnchor(null);
          setIsEditing(true);
        }}
        onArchiveToggle={() => {
          setContextMenuAnchor(null);
          void handleArchiveToggle();
        }}
        onDelete={() => {
          setContextMenuAnchor(null);
          void handleDeleteThread();
        }}
        labels={{
          rename: t("sidebar.thread.rename"),
          archive: t("sidebar.thread.archive"),
          unarchive: t("sidebar.thread.unarchive"),
          delete: t("sidebar.thread.delete"),
        }}
      />
    </ThreadListItemPrimitive.Root>
  );
}

type ThreadListItemByIdProps = ThreadListItemRowProps & {
  threadId: string;
};

type ThreadListItemRuntime = React.ComponentProps<
  typeof ThreadListItemRuntimeProvider
>["runtime"];

function ThreadListItemById({
  threadId,
  onSelectThread,
  highlightActive,
}: ThreadListItemByIdProps) {
  const api = useAssistantApi();
  const runtime = React.useMemo(() => {
    if (!threadId) {
      return null;
    }
    try {
      const client = api.threads().item({ id: threadId });
      const runtimeGetter = (
        client as unknown as { __internal_getRuntime?: () => ThreadListItemRuntime | null }
      ).__internal_getRuntime;
      return runtimeGetter?.() ?? null;
    } catch {
      return null;
    }
  }, [api, threadId]);

  if (!runtime) {
    return null;
  }

  return (
    <ThreadListItemRuntimeProvider runtime={runtime}>
      <ThreadListItemRow
        onSelectThread={onSelectThread}
        highlightActive={highlightActive}
      />
    </ThreadListItemRuntimeProvider>
  );
}

export function AppSidebar({
  reserveMacTrafficLightsGap,
  onOpenSettings,
  onOpenNotifications,
  onToggleNoticePanel,
  headerNavActions,
  activeMainRoute,
  onSelectMainRoute,
  onSelectSettingsRoute,
  onSelectThread,
  highlightThreadActive = true,
  showThreadList = true,
  isAppUpdateAvailable,
  isExternalToolsUpdateAvailable,
  noticeUnreadCount = 0,
  isNoticePanelOpen = false,
  config = DEFAULT_SIDEBAR_CONFIG,
}: AppSidebarProps) {
  const isMac = System.IsMac();
  const shouldReserveMacTrafficLightsGap = reserveMacTrafficLightsGap ?? true;
  const { t } = useI18n();
  const { enabled: assistantUiEnabled, setEnabled: setAssistantUiEnabled } = useAssistantUiMode();
  const settingsLoading = useSettingsStore((state) => state.isLoading);
  const { open: isSetupCenterOpen, setOpen: setSetupCenterOpen } = useSetupCenter();
  const currentUserProfileQuery = useCurrentUserProfile();
  const [search, setSearch] = React.useState("");
  const deferredSearch = React.useDeferredValue(search);
  const [isProductModeOpen, setIsProductModeOpen] = React.useState(false);
  const loadMoreRef = React.useRef<HTMLDivElement | null>(null);
  const setupAutoOpenInitializedRef = React.useRef(false);
  const [renderLimit, setRenderLimit] = React.useState(200);

  const hasUpdateMenu = Boolean(isAppUpdateAvailable || isExternalToolsUpdateAvailable);
  const normalizedSearch = deferredSearch.trim().toLowerCase();
  const currentUserProfile = currentUserProfileQuery.data;
  const currentUserName = resolveUserDisplayName(currentUserProfile);
  const currentProductModeLabel = t(assistantUiEnabled ? "productMode.options.full.title" : "productMode.options.download.title");
  const currentUserSubtitle = resolveUserSubtitle(currentUserProfile) || currentProductModeLabel;
  const unreadNoticeCount = Math.max(0, Math.trunc(noticeUnreadCount));
  const setupStatus = useSetupStatus(unreadNoticeCount);
  const footerDropdownItemClassName =
    "w-full gap-2 whitespace-nowrap rounded-lg py-2 pr-3 pl-3 text-sm outline-none";
  const threadMeta = useThreadStore((state) => state.threads);
  const threadItems = useAssistantState(
    ({ threads }) =>
      threads.threadItems as
        | ReadonlyArray<ThreadListItemState>
        | Record<string, ThreadListItemState>
        | undefined
  );
  const threadIds = useAssistantState(({ threads }) => threads.threadIds);

  React.useEffect(() => {
    if (settingsLoading || setupStatus.checking || setupAutoOpenInitializedRef.current) {
      return;
    }
    setupAutoOpenInitializedRef.current = true;
    if (setupStatus.shouldAutoOpen && !isSetupCenterOpen) {
      setSetupCenterOpen(true);
    }
  }, [
    isSetupCenterOpen,
    setSetupCenterOpen,
    settingsLoading,
    setupStatus.checking,
    setupStatus.shouldAutoOpen,
  ]);

  const threadItemMap = React.useMemo(() => {
    if (!threadItems) {
      return {} as Record<string, ThreadListItemState>;
    }
    if (Array.isArray(threadItems)) {
      const map: Record<string, ThreadListItemState> = {};
      for (const item of threadItems) {
        const id = (item?.id ?? "").trim();
        const remoteId = (item?.remoteId ?? "").trim();
        if (id) {
          map[id] = item;
        }
        if (remoteId) {
          map[remoteId] = item;
        }
      }
      return map;
    }
    return threadItems;
  }, [threadItems]);

  const threadListEntries = React.useMemo(() => {
    return (threadIds ?? [])
      .map((clientId) => {
        const item = threadItemMap[clientId];
        const threadId = (item?.remoteId ?? item?.id ?? "").trim();
        if (!item || !threadId) {
          return null;
        }
        return { clientId, threadId };
      })
      .filter((entry): entry is { clientId: string; threadId: string } => Boolean(entry));
  }, [threadIds, threadItemMap]);

  const sortedThreadListEntries = React.useMemo(() => {
    return [...threadListEntries].sort((left, right) => {
      const leftMeta = threadMeta[left.threadId];
      const rightMeta = threadMeta[right.threadId];
      const rightUpdatedAt = parseThreadTime(rightMeta?.updatedAt || threadItemMap[right.clientId]?.updatedAt);
      const leftUpdatedAt = parseThreadTime(leftMeta?.updatedAt || threadItemMap[left.clientId]?.updatedAt);
      if (rightUpdatedAt !== leftUpdatedAt) {
        return rightUpdatedAt - leftUpdatedAt;
      }
      const rightCreatedAt = parseThreadTime(rightMeta?.createdAt || threadItemMap[right.clientId]?.createdAt);
      const leftCreatedAt = parseThreadTime(leftMeta?.createdAt || threadItemMap[left.clientId]?.createdAt);
      if (rightCreatedAt !== leftCreatedAt) {
        return rightCreatedAt - leftCreatedAt;
      }
      return left.clientId.localeCompare(right.clientId);
    });
  }, [threadItemMap, threadListEntries, threadMeta]);

  const threadSearchIndex = React.useMemo(() => {
    const index = new Map<string, string>();
    sortedThreadListEntries.forEach(({ clientId, threadId }) => {
      const item = threadItemMap[clientId];
      const meta = threadMeta[threadId];
      if (!meta) {
        return;
      }
      const title = (meta.title ?? item?.title ?? "").toLowerCase();
      const workspace = formatWorkspaceLabel(meta.workspaceName, meta.assistantId).toLowerCase();
      index.set(
        clientId,
        [title, workspace, threadId.toLowerCase()].filter(Boolean).join(" ")
      );
    });
    return index;
  }, [sortedThreadListEntries, threadItemMap, threadMeta]);

  const visibleThreadClientIds = React.useMemo(() => {
    return sortedThreadListEntries
      .filter(({ clientId, threadId }) => {
        const item = threadItemMap[clientId];
        const meta = threadMeta[threadId];
        const status = (meta?.status ?? item?.status ?? "new").toLowerCase();
        if (status === "archived" || status === "deleted" || status === "new") {
          return false;
        }
        if (!meta || meta.deletedAt) {
          return false;
        }
        if (!normalizedSearch) {
          return true;
        }
        return (threadSearchIndex.get(clientId) ?? "").includes(normalizedSearch);
      })
      .map(({ clientId }) => clientId);
  }, [normalizedSearch, sortedThreadListEntries, threadItemMap, threadMeta, threadSearchIndex]);

  const visibleThreadCount = visibleThreadClientIds.length;
  const renderedThreadClientIds = React.useMemo(
    () => visibleThreadClientIds.slice(0, renderLimit),
    [renderLimit, visibleThreadClientIds]
  );

  React.useEffect(() => {
    setRenderLimit(200);
  }, [normalizedSearch]);

  React.useEffect(() => {
    setRenderLimit((current) => Math.min(Math.max(current, 200), Math.max(visibleThreadClientIds.length, 200)));
  }, [visibleThreadClientIds.length]);

  React.useEffect(() => {
    if (renderLimit >= visibleThreadClientIds.length) {
      return;
    }
    const node = loadMoreRef.current;
    if (!node) {
      return;
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (!entries.some((entry) => entry.isIntersecting)) {
          return;
        }
        setRenderLimit((current) => Math.min(current + 200, visibleThreadClientIds.length));
      },
      { root: null, rootMargin: "320px" }
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [renderLimit, visibleThreadClientIds.length]);

  const handleSelectRoute = (routeId: MainRouteId) => {
    if (onSelectMainRoute) {
      onSelectMainRoute(routeId);
    }
  };

  const handleSelectSettings = (routeId?: SettingsRouteId) => {
    if (routeId) {
      onSelectSettingsRoute?.(routeId);
    }
    onOpenSettings?.();
  };

  const updateMenuItems = React.useMemo(
    () =>
      [
        isAppUpdateAvailable
          ? {
              key: "app-update",
              label: t("sidebar.footer.menu.appUpdate"),
              Icon: ArrowUpCircle,
              iconClassName: "text-primary",
              onSelect: () => handleSelectSettings("about"),
            }
          : null,
        isExternalToolsUpdateAvailable
          ? {
              key: "external-tools-update",
              label: t("sidebar.footer.menu.externalToolsUpdate"),
              Icon: Wrench,
              iconClassName: "text-primary",
              onSelect: () => handleSelectSettings("external-tools"),
            }
          : null,
      ].filter((item): item is NonNullable<typeof item> => Boolean(item)),
    [isAppUpdateAvailable, isExternalToolsUpdateAvailable, t]
  );

  const handleSelectProductMode = (nextEnabled: boolean) => {
    setAssistantUiEnabled(nextEnabled);
  };

  return (
    <Sidebar
      variant="floating"
      side="left"
      className="pb-[6px] [&_div[data-sidebar=sidebar]]:!rounded-[var(--app-sidebar-radius)]"
    >
      {/* 浮动 sidebar 外层有 1px 边框，向上偏移 1px 让 navActions 与 TitleBar 垂直对齐 */}
      <SidebarHeader
        className="-mt-px h-[var(--app-sidebar-title-height)] justify-center gap-0"
        style={
          {
            "--wails-draggable": "drag",
            paddingTop: "var(--app-sidebar-padding)",
            paddingBottom: "var(--app-sidebar-padding)",
            paddingRight: "var(--app-sidebar-padding)",
            paddingLeft: isMac && shouldReserveMacTrafficLightsGap
              ? "var(--app-macos-traffic-lights-gap-sidebar)"
              : "var(--app-sidebar-padding)",
          } as React.CSSProperties
        }
      >
        <div className="flex w-full items-center gap-2">
          <div
            className="flex items-center gap-2"
            style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
          >
            {headerNavActions}
          </div>
          <div className="flex-1" />
        </div>
      </SidebarHeader>

      {showThreadList ? (
        <SidebarHeader
          className="gap-0"
          style={
            {
              paddingTop: 0,
              paddingBottom: "var(--app-sidebar-padding)",
              paddingRight: "var(--app-sidebar-padding)",
              paddingLeft: "var(--app-sidebar-padding)",
            } as React.CSSProperties
          }
        >
          <div className="space-y-[3px]">
            <div className="flex h-8 items-center gap-2 rounded-md border border-border/80 bg-card px-2">
              <Search className="h-4 w-4 text-muted-foreground" />
              <Input
                placeholder={t("sidebar.search.placeholder")}
                value={search}
                onChange={(event) => setSearch(event.target.value)}
                size="compact"
                className="border-0 bg-transparent shadow-none focus-visible:ring-0 focus-visible:ring-offset-0"
              />
            </div>
          </div>
        </SidebarHeader>
      ) : null}

      <SidebarContent className="pb-0">
        <SidebarGroup className="px-[var(--app-sidebar-padding)] pt-[3px] pb-[var(--app-sidebar-padding)]">
          <SidebarMenu className="gap-[3px]">
            {config.navItems.map((route) => (
              <SidebarMenuItem key={route.id}>
                <SidebarMenuButton
                  isActive={activeMainRoute === route.id}
                  onClick={() => handleSelectRoute(route.id)}
                  className="justify-start"
                >
                  <route.icon className="h-4 w-4" />
                  <span className="truncate">{t(`sidebar.nav.${route.id}`)}</span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            ))}
          </SidebarMenu>
        </SidebarGroup>

        {showThreadList ? (
          <SidebarGroup className="px-[var(--app-sidebar-padding)] py-[var(--app-sidebar-padding)]">
            <ThreadListPrimitive.Root className="space-y-1.5">
              {visibleThreadCount === 0 ? (
                <div className="px-2 py-2 text-xs text-muted-foreground">
                  {t("sidebar.thread.empty")}
                </div>
              ) : (
                renderedThreadClientIds.map((threadId) => (
                  <ThreadListItemById
                    key={threadId}
                    threadId={threadId}
                    onSelectThread={onSelectThread}
                    highlightActive={highlightThreadActive}
                  />
                ))
              )}
              {renderedThreadClientIds.length < visibleThreadClientIds.length ? (
                <div ref={loadMoreRef} className="h-2 w-full" />
              ) : null}
            </ThreadListPrimitive.Root>
          </SidebarGroup>
        ) : null}
      </SidebarContent>

      <SidebarFooter
        className="pt-0"
        style={{ "--wails-draggable": "no-drag" } as React.CSSProperties}
      >
        <div className="mb-3">
          <SetupStatusSlot
            unreadNoticeCount={unreadNoticeCount}
            isNoticePanelOpen={isNoticePanelOpen}
            onOpenSetup={() => setSetupCenterOpen(true)}
            onToggleNoticePanel={onToggleNoticePanel}
          />
        </div>
        <DropdownMenu>
          <SidebarMenu>
            <SidebarMenuItem>
              <DropdownMenuTrigger asChild>
                <SidebarMenuButton
                  size="lg"
                  className={cn(
                    "justify-start gap-3 rounded-xl border-0 px-3 py-2.5 shadow-none focus-visible:ring-0 focus-visible:ring-offset-0",
                    "bg-sidebar-accent/55 hover:bg-sidebar-accent"
                  )}
                  aria-label={t("sidebar.footer.menu.open")}
                >
                  <UserAvatar profile={currentUserProfile} className="h-8 w-8 rounded-xl" fallbackClassName="text-[10px]" />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-semibold leading-tight text-foreground">{currentUserName}</div>
                    <div className="truncate text-xs leading-tight text-muted-foreground">{currentUserSubtitle}</div>
                  </div>
                  <div className="flex items-center gap-2">
                    {hasUpdateMenu ? <span className="h-2 w-2 rounded-full bg-primary" /> : null}
                    <ChevronsUpDown className="h-3.5 w-3.5 text-muted-foreground" />
                  </div>
                </SidebarMenuButton>
              </DropdownMenuTrigger>
            </SidebarMenuItem>
          </SidebarMenu>
          <DropdownMenuContent
            side="top"
            align="start"
            sideOffset={8}
            className="w-[var(--radix-dropdown-menu-trigger-width)] min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[var(--radix-dropdown-menu-trigger-width)] rounded-xl bg-popover/95 p-1.5 shadow-lg backdrop-blur-sm"
          >
            <div className="flex items-center gap-3 rounded-lg px-3 py-2">
              <UserAvatar profile={currentUserProfile} className="h-8 w-8 rounded-xl" fallbackClassName="text-[10px]" />
              <div className="min-w-0 flex-1">
                <div className="truncate text-sm font-medium text-foreground">{currentUserName}</div>
                <div className="truncate text-xs text-muted-foreground">
                    {resolveUserSubtitle(currentUserProfile) || t("productMode.profileHint")}
                </div>
              </div>
            </div>

            <DropdownMenuSeparator />

            <div className="p-1">
              <DropdownMenuItem
                className={cn(
                  footerDropdownItemClassName,
                  "border border-border/80 bg-accent/45 shadow-sm",
                  "hover:bg-accent focus:bg-accent"
                )}
                onSelect={() => setIsProductModeOpen(true)}
              >
                <div className="flex h-4 w-4 shrink-0 items-center justify-center text-muted-foreground">
                  <Sparkles className="h-4 w-4 text-muted-foreground" />
                </div>
                <span className="min-w-0 flex-1 truncate font-medium text-foreground">
                  {t("sidebar.footer.menu.productMode")}
                </span>
                <span className="max-w-[42%] shrink truncate rounded-md border border-border/70 bg-background/80 px-1.5 py-0.5 text-right text-[11px] font-medium text-muted-foreground">
                  {currentProductModeLabel}
                </span>
              </DropdownMenuItem>

              <DropdownMenuItem
                className={footerDropdownItemClassName}
                onSelect={() => onOpenNotifications?.()}
              >
                <div className="flex h-4 w-4 shrink-0 items-center justify-center text-muted-foreground">
                  <Bell className="h-4 w-4 text-muted-foreground" />
                </div>
                <span className="min-w-0 flex-1 truncate font-medium text-foreground">
                  {t("sidebar.footer.menu.notifications")}
                </span>
                <span
                  className={cn(
                    "min-w-[1.25rem] rounded-md px-1.5 py-0.5 text-center text-[11px] font-semibold",
                    unreadNoticeCount > 0
                      ? "bg-primary/15 text-primary"
                      : "bg-muted/60 text-muted-foreground"
                  )}
                >
                  {unreadNoticeCount}
                </span>
              </DropdownMenuItem>

              <DropdownMenuItem
                className={footerDropdownItemClassName}
                onSelect={() => handleSelectSettings("general")}
              >
                <div className="flex h-4 w-4 shrink-0 items-center justify-center text-muted-foreground">
                  <Settings className="h-4 w-4 text-muted-foreground" />
                </div>
                <span className="truncate font-medium text-foreground">
                  {t("sidebar.footer.menu.settings")}
                </span>
              </DropdownMenuItem>

              {updateMenuItems.map((item) => (
                <DropdownMenuItem
                  key={item.key}
                  className={footerDropdownItemClassName}
                  onSelect={item.onSelect}
                >
                  <div className={cn("flex h-4 w-4 shrink-0 items-center justify-center", item.iconClassName)}>
                    <item.Icon className={cn("h-4 w-4", item.iconClassName)} />
                  </div>
                  <span className="truncate font-medium text-foreground">
                    {item.label}
                  </span>
                </DropdownMenuItem>
              ))}
            </div>
          </DropdownMenuContent>
        </DropdownMenu>

        <ProductModeDialog
          open={isProductModeOpen}
          onOpenChange={setIsProductModeOpen}
          requireSelection={false}
          enabled={assistantUiEnabled}
          profile={currentUserProfile}
          onSelectMode={handleSelectProductMode}
        />
        <SetupCenterDialog open={isSetupCenterOpen} onOpenChange={setSetupCenterOpen} />
      </SidebarFooter>
    </Sidebar>
  );
}
