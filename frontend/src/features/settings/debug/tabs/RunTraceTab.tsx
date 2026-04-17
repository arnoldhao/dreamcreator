import { useEffect, useMemo, useState } from "react";

import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { TabsContent } from "@/shared/ui/tabs";

import type { RunTraceTabProps } from "../types";
import { ContextTableFooter } from "./ContextTabControls";
import { DebugDetailSheet, type DebugDetailField } from "./DebugDetailSheet";
import { formatRelativeDebugTime } from "../utils/time";

function formatJsonPayload(value: unknown) {
  if (typeof value === "string") {
    const trimmed = value.trim();
    if (!trimmed) {
      return "-";
    }
    try {
      return JSON.stringify(JSON.parse(trimmed), null, 2);
    } catch {
      return trimmed;
    }
  }
  if (value == null) {
    return "-";
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value);
  }
}

export function RunTraceTab({
  t,
  filteredRunEvents,
  runEventsLoading,
  runEventsError,
  formatDateTime,
}: RunTraceTabProps) {
  const { language } = useI18n();
  const [selectedPayloadKey, setSelectedPayloadKey] = useState("");
  const [rowsPerPage, setRowsPerPage] = useState(20);
  const [pageIndex, setPageIndex] = useState(0);

  const pageCount = Math.max(1, Math.ceil(filteredRunEvents.length / rowsPerPage));

  const paginatedRunEvents = useMemo(() => {
    const start = pageIndex * rowsPerPage;
    return filteredRunEvents.slice(start, start + rowsPerPage);
  }, [filteredRunEvents, pageIndex, rowsPerPage]);

  useEffect(() => {
    setPageIndex((current) => Math.min(current, pageCount - 1));
  }, [pageCount]);

  useEffect(() => {
    const visibleKeys = new Set(paginatedRunEvents.map((item) => `${item.row.id}-${item.row.runId}`));
    setSelectedPayloadKey((current) => (current && visibleKeys.has(current) ? current : ""));
  }, [paginatedRunEvents]);

  const selectedRunEvent = useMemo(
    () => paginatedRunEvents.find((item) => `${item.row.id}-${item.row.runId}` === selectedPayloadKey) ?? null,
    [paginatedRunEvents, selectedPayloadKey]
  );

  const firstAt =
    filteredRunEvents.length > 0
      ? formatRelativeDebugTime(filteredRunEvents[filteredRunEvents.length - 1]?.row.createdAt, language, t("common.justNow"))
      : "-";
  const lastAt =
    filteredRunEvents.length > 0
      ? formatRelativeDebugTime(filteredRunEvents[0]?.row.createdAt, language, t("common.justNow"))
      : "-";

  const detailFields = useMemo<DebugDetailField[]>(
    () =>
      !selectedRunEvent
        ? []
        : [
            {
              label: t("settings.debug.trace.table.time"),
              value: formatDateTime(selectedRunEvent.row.createdAt),
            },
            {
              label: t("settings.debug.trace.table.id"),
              value: `#${selectedRunEvent.row.id}`,
            },
            {
              label: t("settings.debug.trace.table.event"),
              value: selectedRunEvent.summary || "-",
            },
            {
              label: t("settings.debug.trace.table.tool"),
              value: selectedRunEvent.toolName || "-",
            },
            {
              label: t("settings.debug.trace.table.runId"),
              value: selectedRunEvent.runId || "-",
              valueClassName: "break-all",
            },
            {
              label: t("settings.debug.realtime.table.type"),
              value: selectedRunEvent.chatEventType || "-",
            },
            {
              label: t("settings.debug.prompt.messages.toolCallId"),
              value: selectedRunEvent.toolCallId || "-",
              valueClassName: "break-all",
            },
          ],
    [formatDateTime, selectedRunEvent, t]
  );

  const selectedPayloadText = useMemo(
    () => formatJsonPayload(selectedRunEvent?.rawRecord ?? selectedRunEvent?.row.payloadJson),
    [selectedRunEvent]
  );

  return (
    <>
      <TabsContent value="trace" className="mt-0 flex h-full min-h-0 flex-1 flex-col gap-3">
        <div className="min-h-0 flex-1 overflow-hidden rounded-lg border border-border/70 bg-background/70">
          <div className="relative h-full overflow-auto">
            <Table className="table-fixed min-w-[760px]">
              <TableHeader className="app-table-dense-head sticky top-0 z-10 bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/80">
                <TableRow>
                  <TableHead className="w-[180px]">{t("settings.debug.trace.table.time")}</TableHead>
                  <TableHead className="w-[88px]">{t("settings.debug.trace.table.id")}</TableHead>
                  <TableHead className="w-[220px]">{t("settings.debug.trace.table.event")}</TableHead>
                  <TableHead>{t("settings.debug.trace.table.tool")}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runEventsLoading ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-10 text-center text-sm text-muted-foreground">
                      {t("common.loading")}
                    </TableCell>
                  </TableRow>
                ) : runEventsError ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-10 text-center text-sm text-destructive">
                      {t("settings.debug.trace.error")}
                    </TableCell>
                  </TableRow>
                ) : paginatedRunEvents.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={4} className="py-10 text-center text-sm text-muted-foreground">
                      {t("settings.debug.trace.empty")}
                    </TableCell>
                  </TableRow>
                ) : (
                  paginatedRunEvents.map((item) => {
                    const rowKey = `${item.row.id}-${item.row.runId}`;
                    const active = selectedRunEvent
                      ? `${selectedRunEvent.row.id}-${selectedRunEvent.row.runId}` === rowKey
                      : false;

                    return (
                      <TableRow
                        key={rowKey}
                        className={cn("cursor-pointer hover:bg-muted/20", active && "bg-muted/40")}
                        onClick={() => setSelectedPayloadKey(rowKey)}
                        title={t("settings.debug.trace.table.viewPayload")}
                      >
                        <TableCell
                          className="font-mono text-[11px] text-muted-foreground"
                          title={formatDateTime(item.row.createdAt)}
                        >
                          {formatDateTime(item.row.createdAt)}
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={`#${item.row.id}`}>
                          #{item.row.id}
                        </TableCell>
                        <TableCell className="text-[11px]" title={item.summary || "-"}>
                          <div className="truncate">{item.summary || "-"}</div>
                        </TableCell>
                        <TableCell className="font-mono text-[11px]" title={item.toolName || "-"}>
                          <div className="truncate">{item.toolName || "-"}</div>
                        </TableCell>
                      </TableRow>
                    );
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </div>

        <ContextTableFooter
          t={t}
          stats={[
            { label: t("settings.debug.contextPanel.footer.eventCount"), value: filteredRunEvents.length },
            { label: t("settings.debug.contextPanel.footer.firstAt"), value: firstAt },
            { label: t("settings.debug.contextPanel.footer.lastAt"), value: lastAt },
          ]}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(value) => {
            setRowsPerPage(value);
            setPageIndex(0);
          }}
          pageIndex={pageIndex}
          pageCount={pageCount}
          onPrevPage={() => setPageIndex((current) => Math.max(0, current - 1))}
          onNextPage={() => setPageIndex((current) => Math.min(pageCount - 1, current + 1))}
        />
      </TabsContent>

      <DebugDetailSheet
        open={Boolean(selectedRunEvent)}
        onOpenChange={(open) => (!open ? setSelectedPayloadKey("") : undefined)}
        title={selectedRunEvent?.summary || `#${selectedRunEvent?.row.id ?? ""}`}
        description={t("settings.debug.trace.detail.description")}
        headerMeta={
          selectedRunEvent ? (
            <span className="truncate text-[11px] text-muted-foreground">
              {selectedRunEvent.runId || selectedRunEvent.toolName || formatDateTime(selectedRunEvent.row.createdAt)}
            </span>
          ) : null
        }
        fields={detailFields}
      >
        <div className="overflow-hidden rounded-md border">
          <div className="space-y-2 px-4 py-3">
            <div className="text-[11px] font-medium text-muted-foreground">{t("settings.debug.trace.detail.payload")}</div>
            <pre className="max-h-64 overflow-auto rounded-md bg-muted/20 p-3 font-mono text-[11px] whitespace-pre-wrap break-words text-foreground">
              {selectedPayloadText}
            </pre>
          </div>
        </div>
      </DebugDetailSheet>
    </>
  );
}
