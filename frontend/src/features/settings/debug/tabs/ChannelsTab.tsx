import { RefreshCw } from "lucide-react";

import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { TabsContent } from "@/shared/ui/tabs";

import type { ChannelsTabProps, TranslateFn } from "../types";

function formatChannelNote(note: string, t: TranslateFn) {
  switch (note) {
    case "debug_not_supported":
      return t("settings.debug.channels.notes.debugNotSupported");
    default:
      return note;
  }
}

function formatAccountNote(note: string, t: TranslateFn) {
  switch (note) {
    case "missing_bot_token":
      return t("settings.debug.channels.notes.missingBotToken");
    case "not_running":
      return t("settings.debug.channels.notes.notRunning");
    case "pairing_required":
      return t("settings.debug.channels.notes.pairingRequired");
    case "group_allowlist_empty":
      return t("settings.debug.channels.notes.groupAllowlistEmpty");
    default:
      return note;
  }
}

export function ChannelsTab({ t, isLoading, hasError, isFetching, data, refetch, formatDateTime }: ChannelsTabProps) {
  return (
    <TabsContent value="channels" className="mt-0 space-y-2">
      <Card className="w-full border bg-card">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <CardTitle className="text-sm font-medium leading-none tracking-normal">
              {t("settings.debug.channels.title")}
            </CardTitle>
            <Button
              variant="outline"
              size="compactIcon"
              onClick={refetch}
              disabled={isFetching}
              aria-label={t("common.refresh")}
              title={t("common.refresh")}
            >
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent size="compact" className="space-y-3 pt-0 text-sm text-muted-foreground">
          {isLoading ? (
            <div>{t("settings.debug.channels.loading")}</div>
          ) : hasError ? (
            <div className="text-destructive">{t("settings.debug.channels.error")}</div>
          ) : (data?.length ?? 0) === 0 ? (
            <div>{t("settings.debug.channels.empty")}</div>
          ) : (
            <div className="space-y-3">
              {(data ?? []).map((channel) => {
                const channelLabel = channel.displayName || channel.channelId;
                const accountList = channel.accounts ?? [];
                return (
                  <div key={channel.channelId} className="rounded-md border border-border/60 bg-muted/10 p-3">
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="text-sm font-medium text-foreground">
                        {channelLabel}
                        <span className="ml-2 text-xs text-muted-foreground">{channel.channelId}</span>
                      </div>
                      <div className="flex items-center gap-2 text-xs text-muted-foreground">
                        <span className="rounded-full bg-muted px-2 py-0.5">
                          {channel.state || t("settings.debug.channels.stateUnknown")}
                        </span>
                        <span>
                          {channel.enabled
                            ? t("settings.debug.channels.enabled")
                            : t("settings.debug.channels.disabled")}
                        </span>
                      </div>
                    </div>
                    <div className="mt-2 grid gap-2 text-xs text-muted-foreground sm:grid-cols-2">
                      <div>
                        {t("settings.debug.channels.updatedAt")}：
                        <span className="text-foreground/80"> {formatDateTime(channel.updatedAt)}</span>
                      </div>
                      <div>
                        {t("settings.debug.channels.kind")}：
                        <span className="text-foreground/80"> {channel.kind || "-"}</span>
                      </div>
                    </div>
                    {channel.lastError ? (
                      <div className="mt-2 text-xs text-destructive">
                        {t("settings.debug.channels.lastError")}： {channel.lastError}
                      </div>
                    ) : null}
                    {channel.notes && channel.notes.length > 0 ? (
                      <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                        {channel.notes.map((note, index) => (
                          <span key={`${channel.channelId}-note-${index}`} className="rounded-full bg-muted px-2 py-0.5">
                            {formatChannelNote(note, t)}
                          </span>
                        ))}
                      </div>
                    ) : null}

                    <div className="mt-3 space-y-2">
                      {accountList.length === 0 ? (
                        <div className="text-xs text-muted-foreground">{t("settings.debug.channels.account.empty")}</div>
                      ) : (
                        accountList.map((account, index) => (
                          <div
                            key={`${channel.channelId}-${account.accountId || "account"}-${index}`}
                            className="rounded-md border border-border/50 bg-card/60 p-2"
                          >
                            <div className="flex flex-wrap items-center justify-between gap-2 text-xs">
                              <div className="font-medium text-foreground">
                                {account.accountId || t("settings.debug.channels.account.unknown")}
                              </div>
                              <div className="flex items-center gap-2 text-muted-foreground">
                                <span>
                                  {account.running
                                    ? t("settings.debug.channels.account.running")
                                    : t("settings.debug.channels.account.stopped")}
                                </span>
                                {account.mode ? <span className="rounded-full bg-muted px-2 py-0.5">{account.mode}</span> : null}
                              </div>
                            </div>
                            {account.notes && account.notes.length > 0 ? (
                              <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted-foreground">
                                {account.notes.map((note, noteIndex) => (
                                  <span
                                    key={`${channel.channelId}-${account.accountId || "account"}-note-${noteIndex}`}
                                    className="rounded-full bg-muted px-2 py-0.5"
                                  >
                                    {formatAccountNote(note, t)}
                                  </span>
                                ))}
                              </div>
                            ) : null}
                          </div>
                        ))
                      )}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>
    </TabsContent>
  );
}
