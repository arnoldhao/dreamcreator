import type { TFunction } from "@/shared/i18n";

export type NoticeKind = "runtime_event" | "system_status" | "product_update";
export type NoticeCategory =
  | "heartbeat"
  | "cron"
  | "subagent"
  | "exec"
  | "gateway"
  | "update";
export type NoticeSeverity = "info" | "success" | "warning" | "error" | "critical";
export type NoticeStatus = "unread" | "read" | "archived";
export type NoticeSurface = "center" | "toast" | "popup" | "os" | "footer";

export interface NoticeSource {
  producer: string;
  sessionKey: string;
  threadId: string;
  runId: string;
  jobId: string;
  channel: string;
  metadata?: Record<string, string>;
}

export interface NoticeAction {
  type: string;
  labelKey: string;
  target: string;
  params?: Record<string, string>;
}

export interface NoticeI18n {
  titleKey: string;
  summaryKey: string;
  bodyKey: string;
  params?: Record<string, string>;
}

export interface Notice {
  id: string;
  kind: NoticeKind;
  category: NoticeCategory;
  code: string;
  severity: NoticeSeverity;
  status: NoticeStatus;
  i18n: NoticeI18n;
  source: NoticeSource;
  action: NoticeAction;
  surfaces: NoticeSurface[];
  dedupKey: string;
  occurrenceCount: number;
  metadata?: Record<string, unknown>;
  createdAt: string;
  updatedAt: string;
  lastOccurredAt: string;
  readAt?: string;
  archivedAt?: string;
  expiresAt?: string;
}

export interface NoticeListRequest {
  statuses?: NoticeStatus[];
  kinds?: NoticeKind[];
  categories?: NoticeCategory[];
  severities?: NoticeSeverity[];
  surface?: NoticeSurface;
  query?: string;
  limit?: number;
}

export const NOTICE_CENTER_SURFACE: NoticeSurface = "center";
export const NOTICE_RUNTIME_I18N_KEYS = [
  "notifications.center.codes.heartbeatPeriodic.title",
  "notifications.center.codes.heartbeatPeriodic.summary",
  "notifications.center.codes.heartbeatPeriodic.body",
  "notifications.center.codes.heartbeatEvent.title",
  "notifications.center.codes.heartbeatEvent.summary",
  "notifications.center.codes.heartbeatEvent.body",
  "notifications.center.codes.heartbeatCron.title",
  "notifications.center.codes.heartbeatCron.summary",
  "notifications.center.codes.heartbeatCron.body",
  "notifications.center.codes.heartbeatExec.title",
  "notifications.center.codes.heartbeatExec.summary",
  "notifications.center.codes.heartbeatExec.body",
  "notifications.center.codes.heartbeatSubagent.title",
  "notifications.center.codes.heartbeatSubagent.summary",
  "notifications.center.codes.heartbeatSubagent.body",
  "notifications.center.codes.heartbeatRuntimeFailed.title",
  "notifications.center.codes.heartbeatRuntimeFailed.summary",
  "notifications.center.codes.heartbeatRuntimeFailed.body",
  "notifications.footer.codes.appUpdate.title",
  "notifications.footer.codes.appUpdate.summary",
  "notifications.footer.codes.appUpdate.body",
  "notifications.footer.codes.externalToolsUpdate.title",
  "notifications.footer.codes.externalToolsUpdate.summary",
  "notifications.footer.codes.externalToolsUpdate.body",
  "notifications.actions.openThread",
  "notifications.actions.openCron",
  "notifications.actions.openNotifications",
  "notifications.actions.openAppUpdates",
  "notifications.actions.openExternalTools",
] as const;

export function noticeSeverityToIntent(severity: NoticeSeverity): "info" | "success" | "warning" | "danger" {
  switch (severity) {
    case "success":
      return "success";
    case "warning":
      return "warning";
    case "error":
    case "critical":
      return "danger";
    default:
      return "info";
  }
}

export function formatNoticeText(template: string, params?: Record<string, string>): string {
  if (!params) {
    return template;
  }
  let output = template;
  Object.entries(params).forEach(([key, value]) => {
    output = output.split(`{${key}}`).join(value ?? "");
  });
  return output;
}

export function resolveNoticeTitle(notice: Notice, t: TFunction): string {
  return formatNoticeText(t(notice.i18n.titleKey), notice.i18n.params);
}

export function resolveNoticeSummary(notice: Notice, t: TFunction): string {
  return formatNoticeText(t(notice.i18n.summaryKey), notice.i18n.params);
}

export function resolveNoticeBody(notice: Notice, t: TFunction): string {
  return formatNoticeText(t(notice.i18n.bodyKey), notice.i18n.params);
}
