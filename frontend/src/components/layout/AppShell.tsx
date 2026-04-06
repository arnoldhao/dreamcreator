import { useEffect, useState, type CSSProperties, type ReactNode } from "react";
import { Events, System, Window } from "@wailsio/runtime";

import { SquarePen } from "lucide-react";

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip";
import {
  SidebarInset,
  SidebarProvider,
  SidebarTrigger,
  useSidebar,
} from "@/shared/ui/sidebar";

import { Button } from "@/shared/ui/button";
import { AppSidebar } from "@/components/layout/AppSidebar";
import { TitleBar } from "@/components/layout/TitleBar";
import { useI18n } from "@/shared/i18n";
import { MessageHost } from "@/shared/message/MessageHost";
import { cn } from "@/lib/utils";

export interface AppShellProps {
  activeWindow: "main" | "settings";
  title: string;
  titleContent?: ReactNode;
  children: ReactNode;
  titleActions?: ReactNode;
  extraActions?: ReactNode;
  onOpenSettings?: () => void;
  onOpenNotifications?: () => void;
  sidebar?: ReactNode;
  sidebarConfig?: import("@/components/layout/AppSidebar").SidebarConfig;
  sidebarWidth?: string;
  showTitleSeparator?: boolean;
  showTitleBarBorder?: boolean;
  boldTitle?: boolean;
  hideTitleBar?: boolean;
  contentScrollable?: boolean;
  contentClassName?: string;
  activeMainRoute?: import("@/app/routes/main").MainRouteId;
  onSelectMainRoute?: (route: import("@/app/routes/main").MainRouteId) => void;
  onSelectSettingsRoute?: (route: import("@/app/routes/settings").SettingsRouteId) => void;
  onNewThread?: () => void;
  onSelectThread?: (threadId: string) => void;
  highlightThreadActive?: boolean;
  showThreadList?: boolean;
  isAppUpdateAvailable?: boolean;
  isExternalToolsUpdateAvailable?: boolean;
  noticeUnreadCount?: number;
  isNoticePanelOpen?: boolean;
  onToggleNoticePanel?: () => void;
}

function AppShellLayout({
  activeWindow,
  title,
  titleContent,
  children,
  titleActions,
  extraActions,
  onOpenSettings,
  onOpenNotifications,
  sidebar,
  sidebarConfig,
  sidebarWidth,
  showTitleSeparator,
  showTitleBarBorder,
  boldTitle,
  hideTitleBar,
  contentScrollable = true,
  contentClassName,
  activeMainRoute,
  onSelectMainRoute,
  onSelectSettingsRoute,
  onNewThread,
  onSelectThread,
  highlightThreadActive,
  showThreadList,
  isAppUpdateAvailable,
  isExternalToolsUpdateAvailable,
  noticeUnreadCount,
  isNoticePanelOpen,
  onToggleNoticePanel,
}: AppShellProps) {
  const { isMobile, state } = useSidebar();
  const { t } = useI18n();
  const isMac = System.IsMac();
  const [isFullscreen, setIsFullscreen] = useState(false);

  const isSidebarExpandedOnDesktop = !isMobile && state === "expanded";
  const navActionLabel = isSidebarExpandedOnDesktop
    ? t("sidebar.close")
    : t("sidebar.open");
  const newThreadLabel = t("sidebar.thread.new");

  useEffect(() => {
    if (!isMac) {
      setIsFullscreen(false);
      return;
    }

    let cancelled = false;

    const syncFullscreenState = async () => {
      try {
        const fullscreen = await Window.IsFullscreen();
        if (!cancelled) {
          setIsFullscreen(Boolean(fullscreen));
        }
      } catch {
        if (!cancelled) {
          setIsFullscreen(false);
        }
      }
    };

    void syncFullscreenState();

    const offWindowFullscreen = Events.On(Events.Types.Common.WindowFullscreen, () => {
      setIsFullscreen(true);
    });
    const offWindowUnFullscreen = Events.On(Events.Types.Common.WindowUnFullscreen, () => {
      setIsFullscreen(false);
    });

    return () => {
      cancelled = true;
      offWindowFullscreen();
      offWindowUnFullscreen();
    };
  }, [isMac]);

  const navActions =
    activeWindow === "main" ? (
      <TooltipProvider delayDuration={0}>
        <div className="flex items-center gap-2">
          <Tooltip>
            <TooltipTrigger asChild>
              <SidebarTrigger
                className="wails-no-drag"
                aria-label={navActionLabel}
              />
            </TooltipTrigger>
            <TooltipContent side="bottom">{navActionLabel}</TooltipContent>
          </Tooltip>
          {onNewThread ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-[var(--app-nav-action-size)] w-[var(--app-nav-action-size)] wails-no-drag"
                  onClick={onNewThread}
                  aria-label={newThreadLabel}
                >
                  <SquarePen className="h-[var(--app-nav-icon-size)] w-[var(--app-nav-icon-size)]" />
                </Button>
              </TooltipTrigger>
              <TooltipContent side="bottom">{newThreadLabel}</TooltipContent>
            </Tooltip>
          ) : null}
        </div>
      </TooltipProvider>
    ) : null;

  const sidebarHeaderNavActions = isSidebarExpandedOnDesktop ? navActions : null;
  const titleBarNavActions = isSidebarExpandedOnDesktop ? null : navActions;

  const shouldReserveMacTrafficLightsGap = activeWindow === "main" && isMac && !isFullscreen;
  const reserveTitleBarMacTrafficLightsGap = shouldReserveMacTrafficLightsGap && !isSidebarExpandedOnDesktop;

  return (
    <>
      {sidebar ?? (
        <AppSidebar
          activeWindow={activeWindow}
          reserveMacTrafficLightsGap={activeWindow === "main" ? shouldReserveMacTrafficLightsGap : undefined}
          onOpenSettings={onOpenSettings}
          onOpenNotifications={onOpenNotifications}
          headerNavActions={sidebarHeaderNavActions}
          activeMainRoute={activeMainRoute}
          onSelectMainRoute={onSelectMainRoute}
          onSelectSettingsRoute={onSelectSettingsRoute}
          onSelectThread={onSelectThread}
          highlightThreadActive={highlightThreadActive}
          showThreadList={showThreadList}
          isAppUpdateAvailable={isAppUpdateAvailable}
          isExternalToolsUpdateAvailable={isExternalToolsUpdateAvailable}
          noticeUnreadCount={noticeUnreadCount}
          isNoticePanelOpen={isNoticePanelOpen}
          onToggleNoticePanel={onToggleNoticePanel}
          config={sidebarConfig}
        />
      )}

      <SidebarInset>
        {hideTitleBar ? null : (
          <TitleBar
            title={title}
            titleContent={titleContent}
            navActions={titleBarNavActions}
            titleActions={
              boldTitle && title ? (
                <div className="flex items-center font-semibold text-foreground [&>svg]:size-[var(--app-titlebar-leading-icon-size)]">
                  {titleActions ?? null}
                </div>
              ) : (
                titleActions
              )
            }
            extraActions={extraActions}
            reserveMacTrafficLightsGap={reserveTitleBarMacTrafficLightsGap}
            showTitleSeparator={showTitleSeparator}
            showBottomBorder={showTitleBarBorder}
            boldTitle={boldTitle}
          />
        )}
        <MessageHost />
        <div
          className={cn(
            "flex min-h-0 flex-1 flex-col bg-transparent",
            contentScrollable ? "overflow-auto" : "overflow-hidden",
            contentClassName
          )}
        >
          {children}
        </div>
      </SidebarInset>
    </>
  );
}

export function AppShell(props: AppShellProps) {
  const lockSidebarOpen = props.activeWindow === "settings";
  const sidebarWidth = props.sidebarWidth ?? "16rem";
  const showTitleSeparator = props.showTitleSeparator ?? true;
  const showTitleBarBorder = props.showTitleBarBorder ?? true;
  const boldTitle = props.boldTitle ?? false;

  return (
    <SidebarProvider
      style={{ "--sidebar-width": sidebarWidth } as CSSProperties}
      open={lockSidebarOpen ? true : undefined}
      onOpenChange={lockSidebarOpen ? () => undefined : undefined}
    >
      <AppShellLayout
        {...props}
        showTitleSeparator={showTitleSeparator}
        showTitleBarBorder={showTitleBarBorder}
        boldTitle={boldTitle}
      />
    </SidebarProvider>
  );
}
