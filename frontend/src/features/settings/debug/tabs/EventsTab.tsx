import { Fragment, useState } from "react";

import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { TabsContent } from "@/shared/ui/tabs";

import type { EventsTabProps } from "../types";

export function EventsTab({
  t,
  gatewayEventsForThread,
  gatewayFilteredEvents,
  selectedGatewayEvent,
  setSelectedGatewayEvent,
  gatewayEventOptions,
  formatRuntimeTime,
}: EventsTabProps) {
  const [expandedGatewayEventKey, setExpandedGatewayEventKey] = useState<string | null>(null);

  const toggleGatewayPayload = (key: string) => {
    setExpandedGatewayEventKey((current) => (current === key ? null : key));
  };

  return (
    <TabsContent value="events" className="mt-0">
      <Card className="w-full border bg-card">
        <CardHeader size="compact" className="space-y-3">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <CardTitle className="text-sm font-medium leading-none tracking-normal">
              {t("settings.debug.gateway.events")}
            </CardTitle>
            <div className="flex items-center gap-2">
              <Select
                value={selectedGatewayEvent}
                onChange={(event) => setSelectedGatewayEvent(event.target.value)}
                className="min-w-[220px]"
              >
                <option value="all">{t("settings.debug.gateway.filterAll")}</option>
                {gatewayEventOptions.map(([eventName, count]) => (
                  <option key={eventName} value={eventName}>
                    {eventName} ({count})
                  </option>
                ))}
              </Select>
              <span className="text-xs text-muted-foreground">
                {t("settings.debug.gateway.eventsCount")}: {gatewayFilteredEvents.length}/{gatewayEventsForThread.length}
              </span>
            </div>
          </div>
        </CardHeader>
        <CardContent size="compact" className="space-y-2 pt-0">
          <div className="rounded-lg bg-card outline outline-1 outline-border">
            <div className="p-2">
              <Table className="text-xs table-fixed w-full">
                <colgroup>
                  <col style={{ width: "24%" }} />
                  <col style={{ width: "28%" }} />
                  <col style={{ width: "24%" }} />
                  <col style={{ width: "24%" }} />
                </colgroup>
                <TableHeader className="app-table-dense-head [&_tr]:border-b">
                  <TableRow>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.gateway.table.time")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.gateway.table.event")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.gateway.table.run")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.gateway.table.session")}</TableHead>
                  </TableRow>
                </TableHeader>
              </Table>

              <div className="max-h-64 overflow-y-auto overflow-x-hidden">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "24%" }} />
                    <col style={{ width: "28%" }} />
                    <col style={{ width: "24%" }} />
                    <col style={{ width: "24%" }} />
                  </colgroup>
                  <TableBody>
                    {gatewayFilteredEvents.length === 0 ? (
                      <TableRow>
                        <TableCell colSpan={4} className="px-2 py-1 text-muted-foreground">
                          {t("settings.debug.gateway.eventsEmpty")}
                        </TableCell>
                      </TableRow>
                    ) : (
                      gatewayFilteredEvents.slice(-120).reverse().map((event) => {
                        const isExpanded = expandedGatewayEventKey === event.__key;
                        const sessionText = event.sessionDisplayId || "-";
                        const payloadText = event.payload
                          ? JSON.stringify(event.payload, null, 2)
                          : t("settings.debug.gateway.noPayload");
                        return (
                          <Fragment key={event.__key}>
                            <TableRow
                              className="cursor-pointer"
                              onClick={() => toggleGatewayPayload(event.__key)}
                              title={t("settings.debug.gateway.table.viewPayload")}
                            >
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px] text-muted-foreground" title={formatRuntimeTime(event.timestamp)}>
                                {formatRuntimeTime(event.timestamp)}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={event.event || "-"}>
                                {event.event || "-"}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={event.runId || "-"}>
                                {event.runId || "-"}
                              </TableCell>
                              <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={sessionText}>
                                {sessionText}
                              </TableCell>
                            </TableRow>
                            {isExpanded ? (
                              <TableRow>
                                <TableCell colSpan={4} className="border-0 px-2 pb-2">
                                  <pre className="max-h-56 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted/20 p-2 text-[11px] text-foreground">
                                    {payloadText}
                                  </pre>
                                </TableCell>
                              </TableRow>
                            ) : null}
                          </Fragment>
                        );
                      })
                    )}
                  </TableBody>
                </Table>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>
    </TabsContent>
  );
}
