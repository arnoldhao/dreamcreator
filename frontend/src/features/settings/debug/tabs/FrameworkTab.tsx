import { Bell, Send, ShieldAlert, Sparkles } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { SettingsCompactListCard, SettingsCompactRow, SettingsCompactSeparator } from "@/shared/ui/settings-layout";
import { TabsContent } from "@/shared/ui/tabs";

import type { FrameworkTabProps } from "../types";
import { RealtimeLogsCard } from "./RealtimeLogsCard";

export function FrameworkTab({
  t,
  logLevel,
  setLogLevel,
  logsLoading,
  logsError,
  logRecords,
  formatRuntimeTime,
  showToastPreview,
  showNotificationPreview,
  showDialogPreview,
  sendOsNotification,
  publishBackendDebug,
}: FrameworkTabProps) {
  const rowButtonsClassName = "flex flex-wrap items-center justify-end gap-2";

  return (
    <TabsContent value="framework" className="mt-0 space-y-3">
      <RealtimeLogsCard
        t={t}
        logLevel={logLevel}
        setLogLevel={setLogLevel}
        logsLoading={logsLoading}
        logsError={logsError}
        logRecords={logRecords}
        formatRuntimeTime={formatRuntimeTime}
      />

      <SettingsCompactListCard>
        <SettingsCompactRow label={t("settings.debug.framework.rows.message")}>
          <div className={rowButtonsClassName}>
            <Button variant="outline" size="compact" onClick={showToastPreview}>
              <Sparkles className="mr-1 h-4 w-4" />
              {t("settings.debug.framework.actions.toast")}
            </Button>
            <Button variant="outline" size="compact" onClick={showNotificationPreview}>
              <Bell className="mr-1 h-4 w-4" />
              {t("settings.debug.framework.actions.notification")}
            </Button>
            <Button size="compact" variant="outline" onClick={sendOsNotification}>
              <Bell className="mr-1 h-4 w-4" />
              {t("settings.debug.framework.actions.sendOS")}
            </Button>
          </div>
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.debug.framework.rows.dialog")}>
          <div className={rowButtonsClassName}>
            <Button variant="destructive" size="compact" onClick={showDialogPreview}>
              <ShieldAlert className="mr-1 h-4 w-4" />
              {t("settings.debug.framework.actions.dialog")}
            </Button>
          </div>
        </SettingsCompactRow>
        <SettingsCompactSeparator />
        <SettingsCompactRow label={t("settings.debug.framework.rows.event")}>
          <div className={rowButtonsClassName}>
            <Button size="compact" variant="outline" onClick={publishBackendDebug}>
              <Send className="mr-1 h-4 w-4" />
              {t("settings.debug.framework.actions.publish")}
            </Button>
          </div>
        </SettingsCompactRow>
      </SettingsCompactListCard>
    </TabsContent>
  );
}
