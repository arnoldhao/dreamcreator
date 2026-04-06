import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";
import { CheckCircle2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";

import type { RunSummaryCardProps } from "../types";

const RELATIVE_UNITS: Array<{ unit: Intl.RelativeTimeFormatUnit; seconds: number }> = [
  { unit: "year", seconds: 60 * 60 * 24 * 365 },
  { unit: "month", seconds: 60 * 60 * 24 * 30 },
  { unit: "week", seconds: 60 * 60 * 24 * 7 },
  { unit: "day", seconds: 60 * 60 * 24 },
  { unit: "hour", seconds: 60 * 60 },
  { unit: "minute", seconds: 60 },
  { unit: "second", seconds: 1 },
];

function formatRelativeTime(value?: string, language = "en", justNowLabel = "just now"): string {
  if (!value) {
    return "-";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return "-";
  }
  const diffSeconds = Math.round((date.getTime() - Date.now()) / 1000);
  const absSeconds = Math.abs(diffSeconds);
  if (absSeconds < 5) {
    return justNowLabel;
  }
  const formatter = new Intl.RelativeTimeFormat(language, { numeric: "auto" });
  for (const { unit, seconds } of RELATIVE_UNITS) {
    if (absSeconds >= seconds || unit === "second") {
      return formatter.format(Math.round(diffSeconds / seconds), unit);
    }
  }
  return formatter.format(diffSeconds, "second");
}

export function RunSummaryCard({
  t,
  runSummaries,
  selectedRunId,
  setSelectedRunId,
  formatDateTime,
  statusLabelClass,
  formatRunStatus,
  emptyText,
}: RunSummaryCardProps) {
  const { language } = useI18n();
  const justNowLabel = t("common.justNow");

  return (
    <div className="rounded-lg bg-card outline outline-1 outline-border">
      <div className="p-2">
        {runSummaries.length === 0 ? (
          <div className="px-2 py-1 text-xs text-muted-foreground">{emptyText}</div>
        ) : (
          <>
            <Table className="text-xs table-fixed w-full">
              <colgroup>
                <col style={{ width: "6%" }} />
                <col style={{ width: "40%" }} />
                <col style={{ width: "12%" }} />
                <col style={{ width: "8%" }} />
                <col style={{ width: "14%" }} />
                <col style={{ width: "20%" }} />
              </colgroup>
              <TableHeader className="app-table-dense-head [&_tr]:border-b">
                <TableRow>
                  <TableHead className="text-center whitespace-nowrap">{t("settings.debug.trace.table.current")}</TableHead>
                  <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.runId")}</TableHead>
                  <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.status")}</TableHead>
                  <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.events")}</TableHead>
                  <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.lastEvent")}</TableHead>
                  <TableHead className="whitespace-nowrap">{t("settings.debug.trace.table.lastAt")}</TableHead>
                </TableRow>
              </TableHeader>
            </Table>

              <div className="max-h-[26rem] overflow-y-auto overflow-x-hidden">
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "6%" }} />
                    <col style={{ width: "40%" }} />
                    <col style={{ width: "12%" }} />
                    <col style={{ width: "8%" }} />
                    <col style={{ width: "14%" }} />
                    <col style={{ width: "20%" }} />
                  </colgroup>
                <TableBody>
                  {runSummaries.map((item) => (
                    <TableRow
                      key={item.runId}
                      onClick={() => setSelectedRunId(selectedRunId === item.runId ? "all" : item.runId)}
                      className={cn(
                        "cursor-pointer border-0",
                        selectedRunId === item.runId
                          ? "[&>td]:bg-primary/10 [&>td]:text-primary [&>td]:font-medium [&>td:first-child]:rounded-l-md [&>td:last-child]:rounded-r-md"
                          : ""
                      )}
                    >
                      <TableCell className="whitespace-nowrap text-center">
                        {selectedRunId === item.runId ? <CheckCircle2 className="mx-auto h-3.5 w-3.5 text-primary" /> : null}
                      </TableCell>
                      <TableCell className="whitespace-nowrap font-mono text-[11px]" title={item.runId}>
                        {item.runId}
                      </TableCell>
                      <TableCell className="whitespace-nowrap">
                        <span className={cn("rounded-full px-2 py-0.5 text-[11px]", statusLabelClass(item.status))}>
                          {formatRunStatus(item.status)}
                        </span>
                      </TableCell>
                      <TableCell className="whitespace-nowrap font-mono text-[11px]">{item.eventCount}</TableCell>
                      <TableCell className="truncate whitespace-nowrap text-[11px]" title={item.lastEvent || "-"}>
                        {item.lastEvent || "-"}
                      </TableCell>
                      <TableCell
                        className="whitespace-nowrap text-[11px] text-muted-foreground"
                        title={formatDateTime(item.lastAt)}
                      >
                        {formatRelativeTime(item.lastAt, language, justNowLabel)}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
