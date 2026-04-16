import { useEffect, useMemo, useState } from "react";

import { cn } from "@/lib/utils";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { TabsContent } from "@/shared/ui/tabs";

import type { EventsTabProps } from "../types";
import { DebugDetailSheet, type DebugDetailField } from "./DebugDetailSheet";

function formatJsonPayload(value: unknown, emptyText: string) {
  if (value == null) {
    return emptyText;
  }
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return emptyText;
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return trimmed;
    }
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function EventsTab({
  t,
  gatewayFilteredEvents,
  formatDateTime,
  formatRuntimeTime,
}: EventsTabProps) {
  const [selectedGatewayEventKey, setSelectedGatewayEventKey] = useState("");

  useEffect(() => {
    const visibleKeys = new Set(gatewayFilteredEvents.map((item) => item.__key));
    setSelectedGatewayEventKey((current) => (current && visibleKeys.has(current) ? current : ""));
  }, [gatewayFilteredEvents]);

  const selectedGatewayEvent = useMemo(
    () => gatewayFilteredEvents.find((item) => item.__key === selectedGatewayEventKey) ?? null,
    [gatewayFilteredEvents, selectedGatewayEventKey]
  );

  const detailFields = useMemo<DebugDetailField[]>(
    () =>
      !selectedGatewayEvent
        ? []
        : [
            {
              label: t("settings.debug.gateway.table.time"),
              value: formatDateTime(selectedGatewayEvent.timestamp),
            },
            {
              label: t("settings.debug.gateway.table.event"),
              value: selectedGatewayEvent.event || t("settings.debug.gateway.eventUnknown"),
            },
            {
              label: t("settings.debug.gateway.table.run"),
              value: selectedGatewayEvent.runId || "-",
            },
            {
              label: t("settings.debug.gateway.table.session"),
              value: selectedGatewayEvent.sessionDisplayId || "-",
              valueClassName: "break-all",
            },
          ],
    [formatDateTime, selectedGatewayEvent, t]
  );

  const selectedPayloadText = useMemo(
    () => formatJsonPayload(selectedGatewayEvent?.payload, t("settings.debug.gateway.noPayload")),
    [selectedGatewayEvent?.payload, t]
  );

  return (
    <>
      <TabsContent value="events" className="mt-0 flex h-full min-h-0 flex-1 flex-col">
        <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-border/70 bg-background/70">
          <div className="relative h-full overflow-auto">
            <Table className="table-fixed min-w-[760px]">
              <TableHeader className="app-table-dense-head sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
                <TableRow>
                  <TableHead className="w-[160px]">{t("settings.debug.gateway.table.time")}</TableHead>
                  <TableHead className="w-[200px]">{t("settings.debug.gateway.table.event")}</TableHead>
                  <TableHead className="w-[160px]">{t("settings.debug.gateway.table.run")}</TableHead>
                  <TableHead>{t("settings.debug.gateway.table.session")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {gatewayFilteredEvents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-10 text-center text-sm text-muted-foreground">
                      {t("settings.debug.gateway.eventsEmpty")}
                    </TableCell>
                  </TableRow>
                ) : (
                  gatewayFilteredEvents.map((event) => {
                    const sessionText = event.sessionDisplayId || "-";
                    const active = selectedGatewayEvent?.__key === event.__key;

                    return (
                      <TableRow
                        key={event.__key}
                        className={cn("cursor-pointer hover:bg-muted/20", active && "bg-muted/40")}
                        onClick={() => setSelectedGatewayEventKey(event.__key)}
                        title={t("settings.debug.gateway.table.viewPayload")}
                      >
                        <TableCell
                          className="font-mono text-[11px] text-muted-foreground"
                          title={formatRuntimeTime(event.timestamp)}
                        >
                          <div className="truncate">{formatRuntimeTime(event.timestamp)}</div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={event.event || "-"}>
                          <div className="truncate">{event.event || "-"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={event.runId || "-"}>
                          <div className="truncate">{event.runId || "-"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={sessionText}>
                          <div className="truncate">{sessionText}</div>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>
      </TabsContent>

      <DebugDetailSheet
        open={Boolean(selectedGatewayEvent)}
        onOpenChange={(open) => (!open ? setSelectedGatewayEventKey("") : undefined)}
        title={selectedGatewayEvent?.event || t("settings.debug.gateway.eventUnknown")}
        description={t("settings.debug.gateway.detail.description")}
        headerMeta={
          selectedGatewayEvent ? (
            <span className="truncate text-[11px] text-muted-foreground">
              {selectedGatewayEvent.runId || selectedGatewayEvent.sessionDisplayId || formatRuntimeTime(selectedGatewayEvent.timestamp)}
            </span>
          ) : null
        }
        fields={detailFields}
      >
        <div className="overflow-hidden rounded-md border">
          <div className="space-y-2 px-4 py-3">
            <div className="text-[11px] font-medium text-muted-foreground">
              {t("settings.debug.gateway.detail.payload")}
            </div>
            <pre className="max-h-64 overflow-auto rounded-md bg-muted/20 p-3 font-mono text-[11px] whitespace-pre-wrap break-words text-foreground">
              {selectedPayloadText}
            </pre>
          </div>
        </div>
      </DebugDetailSheet>
    </>
  );
}
