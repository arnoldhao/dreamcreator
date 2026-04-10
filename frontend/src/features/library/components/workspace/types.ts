import type { WorkspaceQaCheckId } from "../../model/workspaceQa"

export type WorkspaceDisplayMode = "mono" | "bilingual"
export type WorkspaceDensity = "comfortable" | "compact"
export type WorkspaceFilter = "all" | "needs-review" | "edited" | "current-window"
export type WorkspaceQaFilter = "all" | "issues" | "warnings" | "errors" | "clean"
export type WorkspaceCueIssueSeverity = "error" | "warning" | "info"
export type WorkspaceGuidelineProfileId = "netflix" | "bbc" | "ade"
export type WorkspaceLanguageTaskMode = "translate" | "proofread" | "qa" | "restore"

export type WorkspaceCueIssue = {
  code: string
  label: string
  severity: WorkspaceCueIssueSeverity
  sourceLabel: string
  detailLabel: string
  reason: string
}

export type WorkspaceReviewDecision = "undecided" | "accept" | "reject"

export type WorkspaceReviewSuggestion = {
  cueIndex: number
  kind: "proofread" | "qa" | string
  sourceLabel: string
  detailLabel: string
  reason: string
  originalText: string
  suggestedText: string
  severity: WorkspaceCueIssueSeverity
}

export type WorkspaceSelectOption = {
  value: string
  label: string
  hint?: string
}

export type WorkspaceConstraintOption = WorkspaceSelectOption & {
  description?: string
  badge?: string
}

export type WorkspaceSubtitleRow = {
  id: string
  index: number
  start: string
  end: string
  startMs: number
  endMs: number
  durationMs: number
  sourceText: string
}

export type WorkspaceSubtitleMetrics = {
  cps: number
  cpl: number
  characters: number
  lineCount: number
}

export type WorkspaceResolvedSubtitleRow = WorkspaceSubtitleRow & {
  durationLabel: string
  translationText: string
  qaIssues: WorkspaceCueIssue[]
  status: "ready" | "edited" | "review"
  edited: boolean
  metrics: WorkspaceSubtitleMetrics
  reviewSuggestion?: WorkspaceReviewSuggestion
  reviewDecision?: WorkspaceReviewDecision
}

export type WorkspaceSubtitleTrackOption = WorkspaceSelectOption & {
  language: string
}

export type WorkspaceVideoOption = WorkspaceSelectOption

export type WorkspaceGuidelineOption = {
  value: WorkspaceGuidelineProfileId
  label: string
  hint: string
}

export type WorkspaceImportNormalizationOptions = {
  normalizeLineBreaks: boolean
  trimWhitespace: boolean
  removeBlankLines: boolean
  repairEncoding: boolean
}

export type WorkspaceProofreadOptions = {
  spelling: boolean
  punctuation: boolean
  terminology: boolean
}

export type WorkspaceQaSummary = {
  flaggedCueCount: number
  errorCount: number
  warningCount: number
  issueCounts: Record<WorkspaceQaCheckId, number>
}
