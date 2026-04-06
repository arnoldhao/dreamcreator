import { t } from "@/shared/i18n"

export type WorkspaceQaCheckId = "text" | "timing" | "layout" | "cps" | "cpl"
export type WorkspaceQaMetricColumn = "cps" | "cpl"
export type WorkspaceQaCheckSettings = Record<WorkspaceQaCheckId, boolean>
export type WorkspaceQaCheckDefinition = {
  id: WorkspaceQaCheckId
  label: string
  description: string
  metricColumn?: WorkspaceQaMetricColumn
}

export const DEFAULT_WORKSPACE_QA_CHECK_SETTINGS: WorkspaceQaCheckSettings = {
  text: true,
  timing: true,
  layout: true,
  cps: true,
  cpl: true,
}

export const WORKSPACE_QA_CHECK_ORDER: WorkspaceQaCheckId[] = ["text", "timing", "layout", "cps", "cpl"]

export function normalizeWorkspaceQaCheckSettings(value: unknown): WorkspaceQaCheckSettings {
  const candidate = value && typeof value === "object" ? (value as Partial<Record<WorkspaceQaCheckId, unknown>>) : {}
  return {
    text: candidate.text === false ? false : true,
    timing: candidate.timing === false ? false : true,
    layout: candidate.layout === false ? false : true,
    cps: candidate.cps === false ? false : true,
    cpl: candidate.cpl === false ? false : true,
  }
}

export function resolveWorkspaceQaCheckDefinitions(): WorkspaceQaCheckDefinition[] {
  return [
    {
      id: "text",
      label: t("library.workspace.dialogs.languageTask.qaChecks.text"),
      description: t("library.workspace.dialogs.languageTask.qaChecks.textDescription"),
    },
    {
      id: "timing",
      label: t("library.workspace.dialogs.languageTask.qaChecks.timing"),
      description: t("library.workspace.dialogs.languageTask.qaChecks.timingDescription"),
    },
    {
      id: "layout",
      label: t("library.workspace.dialogs.languageTask.qaChecks.layout"),
      description: t("library.workspace.dialogs.languageTask.qaChecks.layoutDescription"),
    },
    {
      id: "cps",
      label: t("library.workspace.dialogs.languageTask.qaChecks.cps"),
      description: t("library.workspace.dialogs.languageTask.qaChecks.cpsDescription"),
      metricColumn: "cps",
    },
    {
      id: "cpl",
      label: t("library.workspace.dialogs.languageTask.qaChecks.cpl"),
      description: t("library.workspace.dialogs.languageTask.qaChecks.cplDescription"),
      metricColumn: "cpl",
    },
  ]
}

export function resolveWorkspaceQaIssueCheckId(code: string): WorkspaceQaCheckId | null {
  const normalized = code.trim().toLowerCase()
  switch (normalized) {
    case "empty-text":
    case "empty_text":
    case "empty_content":
      return "text"
    case "fast-cue":
    case "tight-gap":
    case "invalid_timing":
    case "non_positive_duration":
    case "overlap":
      return "timing"
    case "too-many-lines":
      return "layout"
    case "cps-error":
    case "cps-warning":
      return "cps"
    case "cpl-error":
    case "cpl-warning":
      return "cpl"
    default:
      return null
  }
}

export function isWorkspaceQaCheckEnabled(settings: WorkspaceQaCheckSettings, id: WorkspaceQaCheckId) {
  return settings[id] !== false
}

export function hasWorkspaceQaMetricColumn(
  settings: WorkspaceQaCheckSettings,
  metricColumn: WorkspaceQaMetricColumn,
) {
  const definition = resolveWorkspaceQaCheckDefinitions().find((item) => item.metricColumn === metricColumn)
  return definition ? isWorkspaceQaCheckEnabled(settings, definition.id) : false
}
