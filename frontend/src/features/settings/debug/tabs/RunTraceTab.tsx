import { Fragment, useState } from "react";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { TabsContent } from "@/shared/ui/tabs";

import type { RunTraceTabProps } from "../types";
import { RunSummaryCard } from "./RunSummaryCard";

export function RunTraceTab({
  t,
  selectedRunId,
  setSelectedRunId,
  runSummaries,
  filteredRunEvents,
  runEventsLoading,
  runEventsError,
  formatDateTime,
  statusLabelClass,
  formatRunStatus,
}: RunTraceTabProps) {
  const [expandedPayloadKey, setExpandedPayloadKey] = useState<string | null>(null);

  const togglePayload = (key: string) => {
    setExpandedPayloadKey((current) => (current === key ? null : key));
  };

  return (
    <TabsContent value="trace" className="mt-0 space-y-3">
      <RunSummaryCard
        t={t}
        runSummaries={runSummaries}
        selectedRunId={selectedRunId}
        setSelectedRunId={setSelectedRunId}
        formatDateTime={formatDateTime}
        statusLabelClass={statusLabelClass}
        formatRunStatus={formatRunStatus}
        emptyText={t("settings.debug.trace.empty")}
      />

      <div className="rounded-lg bg-card outline outline-1 outline-border">
        <div className="p-2">
          {runEventsLoading ? (
            <div className="px-2 py-1 text-xs">{t("common.loading")}</div>
          ) : runEventsError ? (
            <div className="px-2 py-1 text-xs text-destructive">{t("settings.debug.trace.error")}</div>
          ) : filteredRunEvents.length === 0 ? (
            <div className="px-2 py-1 text-xs text-muted-foreground">{t("settings.debug.trace.empty")}</div>
          ) : (
            <>
              <Table className="text-xs table-fixed w-full">
                <colgroup>
                  <col style={{ width: "28%" }} />
                  <col style={{ width: "14%" }} />
                  <col style={{ width: "34%" }} />
                  <col style={{ width: "24%" }} />
                </colgroup>
                <TableHeader className="app-table-dense-head [&_tr]:border-b">
                  <TableRow>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.time")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.id")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.event")}</TableHead>
                    <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.tool")}</TableHead>
                  </TableRow>
                </TableHeader>
              </Table>

              <div className="max-h-[28rem] overflow-y-auto overflow-x-hidden">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "28%" }} />
                    <col style={{ width: "14%" }} />
                    <col style={{ width: "34%" }} />
                    <col style={{ width: "24%" }} />
                  </colgroup>
                  <TableBody>
                    {filteredRunEvents.map((item) => {
                      const rowKey = `${item.row.id}-${item.row.runId}`;
                      const isExpanded = expandedPayloadKey === rowKey;
                      const payloadText = item.rawRecord ? JSON.stringify(item.rawRecord, null, 2) : item.row.payloadJson || "-";
                      return (
                        <Fragment key={rowKey}>
                          <TableRow
                            className="cursor-pointer"
                            onClick={() => togglePayload(rowKey)}
                            title={t("settings.debug.trace.table.viewPayload")}
                          >
                            <TableCell className="truncate whitespace-nowrap font-mono text-[11px] text-muted-foreground" title={formatDateTime(item.row.createdAt)}>
                              {formatDateTime(item.row.createdAt)}
                            </TableCell>
                            <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={`#${item.row.id}`}>
                              #{item.row.id}
                            </TableCell>
                            <TableCell className="truncate whitespace-nowrap text-[11px]" title={item.summary || "-"}>
                              {item.summary || "-"}
                            </TableCell>
                            <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={item.toolName || "-"}>
                              {item.toolName || "-"}
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
                    })}
                  </TableBody>
                </Table>
              </div>
            </>
          )}
        </div>
      </div>
    </TabsContent>
  );
}
