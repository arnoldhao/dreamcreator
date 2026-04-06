import { Bell, ChevronRight, ShieldAlert } from "lucide-react";

import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { useSetupStatus } from "./useSetupStatus";

type SetupStatusSlotProps = {
  unreadNoticeCount: number;
  isNoticePanelOpen: boolean;
  onOpenSetup: () => void;
  onToggleNoticePanel?: () => void;
};

export function SetupStatusSlot({
  unreadNoticeCount,
  isNoticePanelOpen,
  onOpenSetup,
  onToggleNoticePanel,
}: SetupStatusSlotProps) {
  const { t } = useI18n();
  const status = useSetupStatus(unreadNoticeCount);
  const hasSetupAttention = status.blockingIssues.length > 0 || status.recommendedIssues.length > 0;

  if (hasSetupAttention) {
    return (
      <button
        type="button"
        className={cn(
          "inline-flex h-7 w-full items-center justify-between rounded-full border px-2.5 text-xs app-motion-surface",
          status.blockingIssues.length > 0
            ? "border-amber-300/45 bg-amber-500/10 text-amber-900 dark:text-amber-100"
            : "border-sky-300/45 bg-sky-500/10 text-sky-900 dark:text-sky-100"
        )}
        onClick={onOpenSetup}
      >
        <div className="flex min-w-0 items-center gap-2">
          <ShieldAlert className="h-3.5 w-3.5 shrink-0" />
          <span className="truncate font-medium">{t("setupCenter.footer.incompleteTitle")}</span>
        </div>
        <div className="ml-2 flex shrink-0 items-center gap-1.5 font-semibold">
          <span className="tabular-nums">
            {status.configuredChecks}/{status.totalChecks}
          </span>
          <ChevronRight className="h-3.5 w-3.5 opacity-70" />
        </div>
      </button>
    );
  }

  if (unreadNoticeCount > 0) {
    return (
      <button
        type="button"
        aria-label={t("sidebar.footer.menu.notifications")}
        className={cn(
          "inline-flex h-7 w-full items-center justify-between rounded-full border px-2.5 text-xs app-motion-surface",
          isNoticePanelOpen
            ? "border-primary/50 bg-primary/12 text-primary"
            : "border-border/60 bg-sidebar-accent/25 text-muted-foreground hover:text-foreground"
        )}
        onClick={onToggleNoticePanel}
      >
        <div className="flex min-w-0 items-center gap-2">
          <Bell className="h-3.5 w-3.5 shrink-0" />
          <span className="truncate font-medium">{t("setupCenter.footer.noticeLabel")}</span>
        </div>
        <div className="ml-2 flex shrink-0 items-center gap-1.5 font-semibold">
          <span className="tabular-nums">{unreadNoticeCount > 99 ? "99+" : unreadNoticeCount}</span>
          <ChevronRight className="h-3.5 w-3.5 opacity-70" />
        </div>
      </button>
    );
  }

  return null;
}
