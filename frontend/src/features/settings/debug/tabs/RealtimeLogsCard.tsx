import { Fragment, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/shared/ui/table";

import type { RealtimeLogsCardProps } from "../types";

export function RealtimeLogsCard({
  t,
  logLevel,
  setLogLevel,
  logsLoading,
  logsError,
  logRecords,
  formatRuntimeTime,
}: RealtimeLogsCardProps) {
  const [expandedLogKey, setExpandedLogKey] = useState<string | null>(null);

  const toggleLogPayload = (key: string) => {
    setExpandedLogKey((current) => (current === key ? null : key));
  };

  return (
    <Card className="w-full border bg-card">
      <CardHeader size="compact" className="space-y-3">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <CardTitle className="text-sm font-medium leading-none tracking-normal">
            {t("settings.debug.diagnostics.logs")}
          </CardTitle>
          <div className="flex items-center gap-2">
            <Select value={logLevel} onChange={(event) => setLogLevel(event.target.value)}>
              <option value="debug">debug</option>
              <option value="info">info</option>
              <option value="warn">warn</option>
              <option value="error">error</option>
            </Select>
            <span className="text-xs text-muted-foreground">
              {t("settings.debug.diagnostics.logsCount")}: {logRecords.length}
            </span>
          </div>
        </div>
      </CardHeader>
      <CardContent size="compact" className="pt-0">
        <div className="rounded-lg bg-card outline outline-1 outline-border">
          <div className="p-2">
            {logsLoading ? (
              <div className="px-2 py-1 text-xs">{t("common.loading")}</div>
            ) : logsError ? (
              <div className="px-2 py-1 text-xs text-destructive">
                {t("settings.debug.diagnostics.logsError")}
              </div>
            ) : (
              <>
                <Table className="text-xs table-fixed w-full">
                  <colgroup>
                    <col style={{ width: "24%" }} />
                    <col style={{ width: "12%" }} />
                    <col style={{ width: "26%" }} />
                    <col style={{ width: "38%" }} />
                  </colgroup>
                  <TableHeader className="app-table-dense-head [&_tr]:border-b">
                    <TableRow>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.diagnostics.logsTable.time")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.diagnostics.logsTable.level")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.diagnostics.logsTable.source")}</TableHead>
                      <TableHead className="whitespace-nowrap">{t("settings.debug.diagnostics.logsTable.message")}</TableHead>
                    </TableRow>
                  </TableHeader>
                </Table>

                <div className="max-h-64 overflow-y-auto overflow-x-hidden">
                  <Table className="text-xs table-fixed w-full">
                    <colgroup>
                      <col style={{ width: "24%" }} />
                      <col style={{ width: "12%" }} />
                      <col style={{ width: "26%" }} />
                      <col style={{ width: "38%" }} />
                    </colgroup>
                    <TableBody>
                      {logRecords.length === 0 ? (
                        <TableRow>
                          <TableCell colSpan={4} className="px-2 py-1 text-muted-foreground">
                            {t("settings.debug.diagnostics.logsEmpty")}
                          </TableCell>
                        </TableRow>
                      ) : (
                        logRecords.map((record, index) => {
                          const rowKey = `${record.ts}-${record.level}-${record.component || "component"}-${index}`;
                          const isExpanded = expandedLogKey === rowKey;
                          const payloadText = JSON.stringify(
                            {
                              ts: record.ts,
                              level: record.level,
                              component: record.component,
                              message: record.message,
                              ...(record.fields ? { fields: record.fields } : {}),
                            },
                            null,
                            2
                          );
                          return (
                            <Fragment key={rowKey}>
                              <TableRow
                                className="cursor-pointer"
                                onClick={() => toggleLogPayload(rowKey)}
                                title={t("settings.debug.diagnostics.logsTable.viewPayload")}
                              >
                                <TableCell
                                  className="truncate whitespace-nowrap font-mono text-[11px] text-muted-foreground"
                                  title={formatRuntimeTime(record.ts)}
                                >
                                  {formatRuntimeTime(record.ts)}
                                </TableCell>
                                <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={record.level || "-"}>
                                  {record.level || "-"}
                                </TableCell>
                                <TableCell className="truncate whitespace-nowrap font-mono text-[11px]" title={record.component || "-"}>
                                  {record.component || "-"}
                                </TableCell>
                                <TableCell className="truncate whitespace-nowrap text-[11px]" title={record.message || "-"}>
                                  {record.message || "-"}
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
              </>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
