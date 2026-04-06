import * as React from "react";

import { useI18n } from "@/shared/i18n";
import { useGatewayHealth, useGatewayLogsTail, useGatewayStatus } from "@/shared/query/diagnostics";
import { Button } from "@/shared/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/shared/ui/card";
import { Select } from "@/shared/ui/select";
import { Separator } from "@/shared/ui/separator";

const LEVEL_OPTIONS = [
  { id: "debug", label: "Debug" },
  { id: "info", label: "Info" },
  { id: "warn", label: "Warn" },
  { id: "error", label: "Error" },
];

const formatUptime = (seconds: number) => {
  if (!Number.isFinite(seconds) || seconds <= 0) {
    return "0s";
  }
  const mins = Math.floor(seconds / 60);
  const hrs = Math.floor(mins / 60);
  if (hrs > 0) {
    return `${hrs}h ${mins % 60}m`;
  }
  if (mins > 0) {
    return `${mins}m ${Math.floor(seconds % 60)}s`;
  }
  return `${Math.floor(seconds)}s`;
};

export function GatewayPage() {
  const { t } = useI18n();
  const [level, setLevel] = React.useState<string>("info");
  const healthQuery = useGatewayHealth();
  const statusQuery = useGatewayStatus();
  const logsQuery = useGatewayLogsTail({ level, limit: 200 });

  const refresh = () => {
    void healthQuery.refetch();
    void statusQuery.refetch();
    void logsQuery.refetch();
  };

  const health = healthQuery.data;
  const status = statusQuery.data;
  const logs = logsQuery.data?.records ?? [];

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-lg font-semibold">{t("app.settings.title.gateway")}</div>
          <div className="text-sm text-muted-foreground">
            {t("gateway.page.subtitle")}
          </div>
        </div>
        <Button variant="outline" size="compact" onClick={refresh}>
          {t("common.refresh")}
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("gateway.health.overall")}</CardDescription>
            <CardTitle className="text-2xl">{health?.overall || "-"}</CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            {health?.updatedAt ? `Updated ${new Date(health.updatedAt).toLocaleTimeString()}` : "-"}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("gateway.status.uptime")}</CardDescription>
            <CardTitle className="text-2xl">{formatUptime(status?.uptimeSec ?? 0)}</CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            {status?.appVersion ? `v${status.appVersion}` : "-"}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("gateway.status.activeRuns")}</CardDescription>
            <CardTitle className="text-2xl">{status?.activeRuns ?? 0}</CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            {t("gateway.status.sessions")}: {status?.activeSessions ?? 0}
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-2">
            <CardDescription>{t("gateway.status.queueDepth")}</CardDescription>
            <CardTitle className="text-2xl">{status?.queueDepth ?? 0}</CardTitle>
          </CardHeader>
          <CardContent className="text-xs text-muted-foreground">
            {t("gateway.status.nodes")}: {status?.connectedNodes ?? 0}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">{t("gateway.health.components")}</CardTitle>
          <CardDescription className="text-xs">
            {health?.components?.length ? t("gateway.health.componentHint") : "-"}
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3 text-sm text-muted-foreground">
          {health?.components?.length ? (
            health.components.map((component) => (
              <div key={component.name} className="flex flex-wrap items-center justify-between gap-3">
                <div className="font-medium text-foreground">{component.name}</div>
                <div className="flex items-center gap-3 text-xs">
                  <span>{component.status}</span>
                  {component.latencyMs ? <span>{component.latencyMs}ms</span> : null}
                  {component.detail ? <span className="text-muted-foreground">{component.detail}</span> : null}
                </div>
              </div>
            ))
          ) : (
            <div>{t("gateway.health.empty")}</div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="space-y-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <CardTitle className="text-base">{t("gateway.logs.title")}</CardTitle>
              <CardDescription className="text-xs">
                {t("gateway.logs.subtitle")}
              </CardDescription>
            </div>
            <Select
              value={level}
              onChange={(event) => setLevel(event.target.value)}
              className="w-32"
            >
              {LEVEL_OPTIONS.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.label}
                </option>
              ))}
            </Select>
          </div>
          <Separator />
        </CardHeader>
        <CardContent className="space-y-2 text-xs text-muted-foreground">
          {logs.length === 0 ? (
            <div>{t("gateway.logs.empty")}</div>
          ) : (
            logs.map((record, index) => (
              <div key={`${record.ts}-${index}`} className="flex flex-wrap items-start gap-2">
                <span className="min-w-[90px] text-muted-foreground/80">{record.ts || "-"}</span>
                <span className="uppercase">{record.level || "-"}</span>
                {record.component ? <span className="text-muted-foreground/70">{record.component}</span> : null}
                <span className="flex-1 text-foreground">{record.message}</span>
              </div>
            ))
          )}
        </CardContent>
      </Card>
    </div>
  );
}
