import * as React from "react"
import { Browser, Call } from "@wailsio/runtime"
import {
  AlertTriangle,
  Ban,
  CheckCircle2,
  Clock,
  Copy,
  FolderOpen,
  Globe,
  Loader2,
  RotateCcw,
  Square,
  Trash2,
  XCircle,
} from "lucide-react"

import defaultThumbnail from "@/shared/assets/default-thumbnail.png"
import { setPendingSettingsSection } from "@/app/settings/sectionStorage"
import { useI18n } from "@/shared/i18n"
import { messageBus } from "@/shared/message"
import {
  useCancelOperation,
  useCheckYtdlpOperationFailure,
  useDeleteOperation,
  useGetLibrary,
  useGetOperation,
  useGetWorkspaceProject,
  useOpenFileLocation,
  useOpenLibraryPath,
  useResumeOperation,
  useRetryYtdlpOperation,
} from "@/shared/query/library"
import { useShowSettingsWindow } from "@/shared/query/settings"
import type {
  CheckYtdlpOperationFailureItem,
  CheckYtdlpOperationFailureResponse,
  LibraryFileDTO,
  LibraryOperationDTO,
  OperationRequestPreviewDTO,
} from "@/shared/contracts/library"
import { useLibraryRealtimeStore } from "@/shared/store/libraryRealtime"
import { useTaskDialogStore } from "@/shared/store/taskDialog"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
} from "@/shared/ui/dashboard-dialog"
import { Dialog, DialogDescription, DialogTitle } from "@/shared/ui/dialog"
import { Separator } from "@/shared/ui/separator"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { cn } from "@/lib/utils"

import { openLibraryWorkspace } from "../model/workspaceStore"
import { resolveFileIcon } from "../utils/fileIcons"
import { formatBytes } from "../utils/format"
import { formatTemplate } from "../utils/i18n"
import { useSmoothedProgressSpeed, useTimeSyncedSpinDelay } from "../utils/progress-display"
import { translateLibraryProgressDetail, translateLibraryProgressLabel } from "../utils/progress"
import { formatDuration, formatRelativeTime } from "../utils/time"
import { LibraryCellTooltip } from "./LibraryCellTooltip"
import { LibraryTaskIcon } from "./LibraryTaskIcon"

const STATUS_META: Record<
  string,
  { labelKey: string; defaultLabel: string; className: string; Icon: typeof Clock }
> = {
  queued: {
    labelKey: "library.status.queued",
    defaultLabel: "Queued",
    className: "bg-slate-100 text-slate-700 dark:bg-slate-900/50 dark:text-slate-200",
    Icon: Clock,
  },
  running: {
    labelKey: "library.status.running",
    defaultLabel: "In progress",
    className: "bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-100",
    Icon: Loader2,
  },
  succeeded: {
    labelKey: "library.status.succeeded",
    defaultLabel: "Done",
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-100",
    Icon: CheckCircle2,
  },
  failed: {
    labelKey: "library.status.failed",
    defaultLabel: "Failed",
    className: "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-100",
    Icon: AlertTriangle,
  },
  canceled: {
    labelKey: "library.status.canceled",
    defaultLabel: "Canceled",
    className: "bg-amber-100 text-amber-800 dark:bg-amber-900/50 dark:text-amber-100",
    Icon: Ban,
  },
}

type TaskDialogTab = "status" | "outputs" | "library" | "info"
type TaskInfoItem = { key: string; label: string; value: React.ReactNode; stacked?: boolean }

type OperationFileItem = {
  id: string
  name: string
  displayLabel?: string
  path?: string
  fileType: string
  format?: string
  isPrimary?: boolean
  deleted?: boolean
  sourceType?: string
  sizeBytes?: number
}

type TaskImagePreview = {
  name: string
  path: string
}

type SubtitleTaskOutputSummary = {
  status?: string
  mode?: string
  targetLanguage?: string
  language?: string
  chunkCount?: number
  cueCount?: number
  completedChunkCount?: number
  failedChunkCount?: number
  resumedFromCheckpoint?: boolean
  glossaryProfileIds?: string[]
  referenceTrackFileIds?: string[]
  promptProfileIds?: string[]
  inlinePromptHash?: string
  passes?: string[]
}

const EMPTY_OPERATION_REQUEST: OperationRequestPreviewDTO = {}

export function TaskDialog() {
  const { t, language } = useI18n()
  const unknownErrorText = t("library.errors.unknown")
  const open = useTaskDialogStore((state) => state.open)
  const operationId = useTaskDialogStore((state) => state.operationId)
  const closeTaskDialog = useTaskDialogStore((state) => state.closeTaskDialog)
  const realtimeOperations = useLibraryRealtimeStore((state) => state.operations)

  const liveOperation = React.useMemo(
    () => realtimeOperations.find((item) => item.id === operationId),
    [operationId, realtimeOperations],
  )
  const operationQuery = useGetOperation(operationId ?? "")
  const operation = liveOperation ?? operationQuery.data
  const libraryQuery = useGetLibrary(operation?.libraryId ?? "")
  const workspaceProjectQuery = useGetWorkspaceProject(
    operation?.libraryId ?? "",
    open && Boolean(operation?.libraryId),
  )

  const checkFailure = useCheckYtdlpOperationFailure()
  const retryOperation = useRetryYtdlpOperation()
  const openFileLocation = useOpenFileLocation()
  const openLibraryPath = useOpenLibraryPath()
  const cancelOperation = useCancelOperation()
  const resumeOperation = useResumeOperation()
  const deleteOperation = useDeleteOperation()
  const showSettingsWindow = useShowSettingsWindow()

  const [httpBaseUrl, setHttpBaseUrl] = React.useState("")
  const [activeTab, setActiveTab] = React.useState<TaskDialogTab>("outputs")
  const [failureCheckOpen, setFailureCheckOpen] = React.useState(false)
  const [failureCheckResult, setFailureCheckResult] = React.useState<CheckYtdlpOperationFailureResponse | null>(null)
  const [failureCheckError, setFailureCheckError] = React.useState("")
  const [imagePreview, setImagePreview] = React.useState<TaskImagePreview | null>(null)

  React.useEffect(() => {
    let active = true
    Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.HTTPBaseURL")
      .then((url) => {
        if (!active) {
          return
        }
        const resolved = typeof url === "string" ? url : String(url ?? "")
        setHttpBaseUrl(resolved)
      })
      .catch(() => {
        if (active) {
          setHttpBaseUrl("")
        }
      })
    return () => {
      active = false
    }
  }, [])

  React.useEffect(() => {
    if (!open) {
      setFailureCheckOpen(false)
      setFailureCheckResult(null)
      setFailureCheckError("")
      setImagePreview(null)
      return
    }
    setActiveTab(operation?.status === "failed" ? "status" : "outputs")
  }, [open, operation?.status, operationId])

  const filesById = React.useMemo(() => {
    const map = new Map<string, LibraryFileDTO>()
    for (const item of libraryQuery.data?.files ?? []) {
      map.set(item.id, item)
    }
    return map
  }, [libraryQuery.data?.files])
  const workspaceTrackLabelByFileId = React.useMemo(() => {
    const map = new Map<string, string>()
    for (const track of workspaceProjectQuery.data?.videoTracks ?? []) {
      const fileId = track.file?.id?.trim() ?? ""
      const label = track.display?.label?.trim() ?? ""
      if (fileId && label) {
        map.set(fileId, label)
      }
    }
    for (const track of workspaceProjectQuery.data?.subtitleTracks ?? []) {
      const fileId = track.file?.id?.trim() ?? ""
      const label = track.display?.label?.trim() ?? ""
      if (fileId && label) {
        map.set(fileId, label)
      }
    }
    return map
  }, [workspaceProjectQuery.data?.subtitleTracks, workspaceProjectQuery.data?.videoTracks])

  const input = operation?.request ?? EMPTY_OPERATION_REQUEST
  const outputFiles = React.useMemo(() => {
    if (!operation?.outputFiles) {
      return [] as OperationFileItem[]
    }
    return operation.outputFiles.map((output) => {
      const file = filesById.get(output.fileId)
      return {
        id: output.fileId,
        name: file?.name || output.fileId,
        displayLabel: resolveTaskDialogFileDisplayLabel(
          file,
          workspaceTrackLabelByFileId,
          output.fileId,
        ),
        path: file?.storage.localPath,
        fileType: file?.kind || output.kind,
        format: output.format || file?.media?.format,
        isPrimary: output.isPrimary,
        deleted: output.deleted,
        sourceType: file?.origin.kind,
        sizeBytes: output.sizeBytes ?? file?.media?.sizeBytes,
      }
    })
  }, [filesById, operation?.outputFiles, workspaceTrackLabelByFileId])

  const libraryFiles = React.useMemo(() => {
    return (libraryQuery.data?.files ?? []).map((file) => ({
      id: file.id,
      name: file.name,
      displayLabel: resolveTaskDialogFileDisplayLabel(
        file,
        workspaceTrackLabelByFileId,
        file.id,
      ),
      path: file.storage.localPath,
      fileType: file.kind,
      format: file.media?.format,
      deleted: file.state.deleted,
      sourceType: file.origin.kind,
      sizeBytes: file.media?.sizeBytes,
    }))
  }, [libraryQuery.data?.files, workspaceTrackLabelByFileId])

  const folderOutputPath = React.useMemo(() => {
    const preferred =
      outputFiles.find((file) => file.isPrimary && !file.deleted && file.path) ??
      outputFiles.find((file) => !file.deleted && file.path)
    return preferred?.path ?? ""
  }, [outputFiles])

  const thumbnailFile = React.useMemo(
    () => outputFiles.find((file) => file.fileType === "thumbnail" && !file.deleted && file.path),
    [outputFiles],
  )
  const localThumbnailUrl = buildAssetPreviewUrl(httpBaseUrl, thumbnailFile?.path ?? "")
  const coverUrl = localThumbnailUrl || normalizeThumbnailUrl(input.thumbnailUrl ?? "")
  const imagePreviewUrl = React.useMemo(
    () => buildAssetPreviewUrl(httpBaseUrl, imagePreview?.path ?? ""),
    [httpBaseUrl, imagePreview?.path],
  )
  const [coverFailed, setCoverFailed] = React.useState(false)
  React.useEffect(() => {
    setCoverFailed(false)
  }, [coverUrl])

  const dialogTitle = operation?.displayName || operation?.id || t("library.task.title")
  const subtitleTaskOutput = React.useMemo(
    () => parseSubtitleTaskOutput(operation),
    [operation],
  )
  const sourceLabel = operation ? resolveSourceLabel(operation, t) : "-"
  const platformLabel = operation?.meta.platform || input.extractor || "-"
  const uploaderLabel = operation?.meta.uploader || input.author || "-"
  const sourceDomainLabel = operation?.sourceDomain?.trim() ?? ""
  const outputCountLabel = operation ? formatOutputCountValue(operation, outputFiles.length) : "-"
  const outputSizeLabel = operation ? formatOutputSizeValue(operation, outputFiles) : "-"
  const durationLabel = operation ? formatOperationDurationValue(operation) : "-"
  const publishTimeLabel = operation?.meta.publishTime ? formatRelativeTime(operation.meta.publishTime, language) : "-"
  const createdTimeLabel = operation?.createdAt ? formatRelativeTime(operation.createdAt, language) : "-"
  const requestURL = input.url?.trim() ?? ""
  const requestID = operation?.correlation.requestId?.trim() ?? ""
  const runID = operation?.correlation.runId?.trim() ?? ""
  const parentOperationID = operation?.correlation.parentOperationId?.trim() ?? ""
  const subtitleTargetLanguage =
    subtitleTaskOutput?.targetLanguage?.trim() || subtitleTaskOutput?.language?.trim() || ""
  const subtitleConstraintSummary = React.useMemo(
    () => formatSubtitleConstraintSummary(operation?.kind, subtitleTaskOutput, t),
    [operation?.kind, subtitleTaskOutput, t],
  )

  const infoItems = React.useMemo<TaskInfoItem[]>(() => {
    if (!operation) {
      return []
    }
    const items: TaskInfoItem[] = [
      {
        key: "source",
        label: t("library.columns.source"),
        value: <span className="text-xs text-muted-foreground">{sourceLabel}</span>,
      },
      {
        key: "platform",
        label: t("library.columns.platform"),
        value: <span className="text-xs text-muted-foreground">{platformLabel}</span>,
      },
      {
        key: "uploader",
        label: t("library.columns.uploader"),
        value: <span className="text-xs text-muted-foreground">{uploaderLabel}</span>,
      },
      {
        key: "duration",
        label: t("library.columns.duration"),
        value: <span className="text-xs text-muted-foreground">{durationLabel}</span>,
      },
      {
        key: "publish-time",
        label: t("library.columns.publishTime"),
        value: <span className="text-xs text-muted-foreground">{publishTimeLabel}</span>,
      },
      {
        key: "create-time",
        label: t("library.columns.createTime"),
        value: <span className="text-xs text-muted-foreground">{createdTimeLabel}</span>,
      },
    ]
    if (subtitleTargetLanguage) {
      items.push({
        key: "target-language",
        label: t("library.config.languageAssets.targetLanguage"),
        value: <span className="text-xs text-muted-foreground">{subtitleTargetLanguage}</span>,
      })
    }
    if (subtitleConstraintSummary) {
      items.push({
        key: "constraints",
        label: t("library.task.constraints"),
        value: (
          <ExpandableTextBlock text={subtitleConstraintSummary} collapsedChars={96} inline align="end" />
        ),
      })
    }
    if (requestURL) {
      items.push({
        key: "source-url",
        label: t("library.task.requestUrl"),
        value: <TaskDetailSourceURLValue value={requestURL} />,
      })
    }

    const correlationGroupItems: TaskDetailMetaItem[] = []
    if (requestID) {
      correlationGroupItems.push({
        key: "request-id",
        label: t("library.task.requestId"),
        value: (
          <span className="block truncate text-right font-mono text-[11px] text-muted-foreground" title={requestID}>
            {requestID}
          </span>
        ),
      })
    }
    if (runID) {
      correlationGroupItems.push({
        key: "run-id",
        label: t("library.task.runId"),
        value: (
          <span className="block truncate text-right font-mono text-[11px] text-muted-foreground" title={runID}>
            {runID}
          </span>
        ),
      })
    }
    if (parentOperationID) {
      correlationGroupItems.push({
        key: "parent-operation",
        label: t("library.task.parentOperation"),
        value: (
          <span
            className="block truncate text-right font-mono text-[11px] text-muted-foreground"
            title={parentOperationID}
          >
            {parentOperationID}
          </span>
        ),
      })
    }
    if (correlationGroupItems.length > 0) {
      items.push({
        key: "correlation",
        label: t("library.task.correlation"),
        value: <TaskDetailMetaGroup items={correlationGroupItems} />,
        stacked: true,
      })
    }
    return items
  }, [
    createdTimeLabel,
    durationLabel,
    operation,
    parentOperationID,
    platformLabel,
    publishTimeLabel,
    requestID,
    requestURL,
    runID,
    sourceLabel,
    subtitleConstraintSummary,
    subtitleTargetLanguage,
    t,
    uploaderLabel,
  ])
  const canCancelSubtitleTask = Boolean(
    operation?.id &&
      isResumableSubtitleTask(operation?.kind) &&
      (operation?.status === "queued" || operation?.status === "running"),
  )
  const canResumeSubtitleTask = Boolean(
    operation?.id &&
      isResumableSubtitleTask(operation?.kind) &&
      (operation?.status === "failed" || operation?.status === "canceled"),
  )

  const handleOpenFailureCheck = React.useCallback(async () => {
    if (!operation?.id) {
      return
    }
    setFailureCheckError("")
    setFailureCheckResult(null)
    setFailureCheckOpen(true)
    try {
      const result = await checkFailure.mutateAsync({ operationId: operation.id })
      setFailureCheckResult(result)
    } catch (error) {
      setFailureCheckError(resolveErrorMessage(error, unknownErrorText))
    }
  }, [checkFailure, operation?.id, unknownErrorText])

  const statusItems = React.useMemo<TaskInfoItem[]>(() => {
    if (!operation) {
      return []
    }
    const items: TaskInfoItem[] = [
      {
        key: "status",
        label: t("library.columns.status"),
        value: <TaskDialogStatusBadge status={operation.status} phaseKey={operation.id} t={t} />,
      },
      {
        key: "progress",
        label: t("library.columns.progress"),
        value: <TaskProgressValue operation={operation} inline />,
      },
    ]
    if (subtitleTaskOutput?.chunkCount) {
      items.push({
        key: "chunks",
        label: t("library.task.subtitleChunks"),
        value: <span className="text-xs text-muted-foreground">{formatChunkProgress(subtitleTaskOutput)}</span>,
      })
      if (subtitleTaskOutput.failedChunkCount) {
        items.push({
          key: "chunk-failed",
          label: t("library.task.failedChunks"),
          value: <span className="text-xs text-muted-foreground">{subtitleTaskOutput.failedChunkCount}</span>,
        })
      }
      if (subtitleTaskOutput.resumedFromCheckpoint) {
        items.push({
          key: "chunk-resume",
          label: t("library.task.checkpoint"),
          value: <span className="text-xs text-muted-foreground">{t("library.task.resumed")}</span>,
        })
      }
    }
    if (operation.errorCode) {
      items.push({
        key: "error-code",
        label: t("library.task.errorCode"),
        value: <span className="font-mono text-[11px] text-muted-foreground">{operation.errorCode}</span>,
      })
    }
    if (operation.errorMessage && operation.status !== "failed") {
      items.push({
        key: "error-message",
        label: t("library.task.errorMessage"),
        value: <TaskDetailErrorValue value={operation.errorMessage} />,
      })
    }
    if (operation.status === "failed") {
      items.push({
        key: "error",
        label: t("library.task.error"),
        value: <TaskDetailErrorValue value={operation.errorMessage?.trim() || t("library.task.failedUnknown")} />,
      })
    }
    if (operation.status === "failed" && operation.kind === "download") {
      items.push({
        key: "failed-actions",
        label: t("library.task.actions"),
        value: (
          <div className="flex min-w-0 justify-end">
            <Button size="compact" variant="outline" className="h-7 text-xs" onClick={() => void handleOpenFailureCheck()}>
              {t("library.task.failedCheck")}
            </Button>
          </div>
        ),
      })
    }
    return items
  }, [handleOpenFailureCheck, operation, subtitleTaskOutput, t])

  const handleRetryOperation = React.useCallback(async () => {
    if (!operation?.id) {
      return
    }
    setFailureCheckError("")
    try {
      const next = await retryOperation.mutateAsync({ operationId: operation.id })
      setFailureCheckOpen(false)
      closeTaskDialog()
      useTaskDialogStore.getState().openTaskDialog(next.id)
    } catch (error) {
      setFailureCheckError(resolveErrorMessage(error, unknownErrorText))
    }
  }, [closeTaskDialog, operation?.id, retryOperation, unknownErrorText])

  const handleOpenSettings = React.useCallback(
    (section: "external-tools" | "general") => {
      setPendingSettingsSection(section)
      showSettingsWindow.mutate()
      setFailureCheckOpen(false)
    },
    [showSettingsWindow],
  )

  const handleOpenWebpage = React.useCallback(() => {
    const url = input.url?.trim() ?? ""
    if (!url) {
      return
    }
    Browser.OpenURL(url).catch(() => {})
  }, [input.url])

  const handleOpenFolder = React.useCallback(async () => {
    if (!folderOutputPath) {
      return
    }
    try {
      await openLibraryPath.mutateAsync({ path: folderOutputPath })
    } catch {
      // ignore open folder errors
    }
  }, [folderOutputPath, openLibraryPath])

  const handleOpenFile = React.useCallback(
    async (file: OperationFileItem) => {
      const libraryId = operation?.libraryId || libraryQuery.data?.id || ""
      if (isWorkspaceFileType(file.fileType) && libraryId) {
        openLibraryWorkspace({
          libraryId,
          fileId: file.id,
          name: file.name,
          fileType: file.fileType,
          path: file.path,
          openMode: normalizeWorkspaceMode(file.fileType),
        })
        closeTaskDialog()
        return
      }
      if (file.id) {
        try {
          await openFileLocation.mutateAsync({ fileId: file.id })
          return
        } catch {
          // fall through to path open when file lookup fails
        }
      }
      if (file.path) {
        await openLibraryPath.mutateAsync({ path: file.path })
      }
    },
    [closeTaskDialog, libraryQuery.data?.id, openFileLocation, openLibraryPath, operation?.libraryId],
  )

  const handlePreviewFile = React.useCallback((file: OperationFileItem) => {
    setImagePreview({
      name: resolveOperationFilePrimaryLabel(file),
      path: file.path ?? "",
    })
  }, [])

  const handleDelete = React.useCallback(() => {
    if (!operationId) {
      return
    }
    messageBus.publishDialog({
      intent: "danger",
      destructive: true,
      title: t("library.task.deleteTitle"),
      description: formatTemplate(
        t("library.task.deleteDialogDescription"),
        { name: dialogTitle || t("library.rowMenu.renameFallback") }
      ),
      confirmLabel: t("library.task.deleteConfirm"),
      cancelLabel: t("library.rowMenu.deleteCancel"),
      onConfirm: async () => {
        try {
          await deleteOperation.mutateAsync({ operationId, cascadeFiles: false })
          closeTaskDialog()
          messageBus.publishToast({
            intent: "success",
            title: t("library.task.deleteSuccessTitle"),
            description: t("library.task.deleteSuccess"),
          })
        } catch (error) {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.task.deleteFailedTitle"),
            description: resolveErrorMessage(error, unknownErrorText),
          })
        }
      },
    })
  }, [closeTaskDialog, deleteOperation, dialogTitle, operationId, t, unknownErrorText])

  const handleCancelSubtitleTask = React.useCallback(async () => {
    if (!operation?.id) {
      return
    }
    try {
      await cancelOperation.mutateAsync({ operationId: operation.id })
      messageBus.publishToast({
        intent: "success",
        title: t("library.task.cancelSuccess"),
        description: t("library.task.cancelSuccessDescription"),
      })
      setActiveTab("status")
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.task.cancelFailed"),
        description: resolveErrorMessage(error, unknownErrorText),
      })
    }
  }, [cancelOperation, operation?.id, t, unknownErrorText])

  const handleResumeSubtitleTask = React.useCallback(async () => {
    if (!operation?.id) {
      return
    }
    try {
      await resumeOperation.mutateAsync({ operationId: operation.id })
      messageBus.publishToast({
        intent: "success",
        title: t("library.task.resumeSuccess"),
        description: t("library.task.resumeSuccessDescription"),
      })
      setActiveTab("status")
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.task.resumeFailed"),
        description: resolveErrorMessage(error, unknownErrorText),
      })
    }
  }, [operation?.id, resumeOperation, t, unknownErrorText])

  return (
    <>
      <Dialog open={open} onOpenChange={(nextOpen) => (!nextOpen ? closeTaskDialog() : undefined)}>
        <DashboardDialogContent
          size="detail"
          className="flex max-h-[80vh] min-h-0 w-full flex-col gap-4 text-xs"
          onOpenAutoFocus={(event) => event.preventDefault()}
        >
          <DashboardDialogHeader>
            <DialogTitle className="min-w-0 truncate">{dialogTitle}</DialogTitle>
            <DialogDescription className="sr-only">
              {t("library.task.dialogDescription")}
            </DialogDescription>
          </DashboardDialogHeader>

          <div className="flex min-h-0 flex-1 flex-col gap-3">
            {operationQuery.isLoading && !operation ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" />
                {t("library.task.loading")}
              </div>
            ) : null}

            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(220px,248px)]">
              <TaskDialogHeaderCard
                title={t("library.task.summary")}
                badge={
                  operation ? (
                    <Badge variant="outline" className="h-5 rounded-md px-2 text-xs font-medium">
                      {sourceLabel}
                    </Badge>
                  ) : null
                }
              >
                <div className="grid gap-3 sm:grid-cols-[128px_minmax(0,1fr)]">
                  <div className="h-[72px] w-[128px] overflow-hidden rounded-md bg-muted">
                    {coverUrl && !coverFailed ? (
                      <img
                        src={coverUrl}
                        alt={dialogTitle}
                        className="h-full w-full object-cover"
                        onError={() => setCoverFailed(true)}
                      />
                    ) : (
                      <img src={defaultThumbnail} alt="thumbnail" className="h-full w-full object-contain" />
                    )}
                  </div>
                  <div className="min-w-0 space-y-2">
                    <TaskDialogSummaryRow label={t("library.columns.platform")} value={platformLabel} />
                    <TaskDialogSummaryRow label={t("library.columns.uploader")} value={uploaderLabel} />
                    <TaskDialogSummaryRow label={t("library.columns.source")} value={sourceLabel} />
                  </div>
                </div>
                <div className={cn("mt-3 flex items-center gap-2 px-3 py-2", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
                  {sourceDomainLabel ? (
                    <>
                      <LibraryTaskIcon
                        taskType={operation?.kind ?? "download"}
                        sourceDomain={operation?.sourceDomain}
                        sourceIcon={operation?.sourceIcon}
                        className="h-4 w-4 shrink-0"
                      />
                      <span className="min-w-0 flex-1 truncate text-xs text-muted-foreground">{sourceDomainLabel}</span>
                    </>
                  ) : null}
                  <div className="ml-auto flex items-center gap-2">
                    {requestURL ? (
                      <LibraryCellTooltip label={t("library.task.openSourceWebpage")}>
                        <Button
                          type="button"
                          variant="ghost"
                          size="compactIcon"
                          className="h-6 w-6"
                          onClick={handleOpenWebpage}
                        >
                          <Globe className="h-3.5 w-3.5" />
                        </Button>
                      </LibraryCellTooltip>
                    ) : null}
                    <LibraryCellTooltip label={t("library.tooltips.openFolder")}>
                      <Button
                        type="button"
                        variant="ghost"
                        size="compactIcon"
                        className="h-6 w-6"
                        onClick={handleOpenFolder}
                        disabled={!folderOutputPath}
                      >
                        <FolderOpen className="h-3.5 w-3.5" />
                      </Button>
                    </LibraryCellTooltip>
                  </div>
                </div>
              </TaskDialogHeaderCard>

              <TaskDialogHeaderCard
                title={t("library.task.overview")}
                badge={operation ? <TaskDialogStatusBadge status={operation.status} phaseKey={operation.id} t={t} /> : null}
              >
                <TaskDialogMetricsCard
                  columns={2}
                  items={[
                    { label: t("library.columns.outputCount"), value: outputCountLabel },
                    { label: t("library.columns.outputSize"), value: outputSizeLabel },
                    { label: t("library.columns.createTime"), value: createdTimeLabel },
                    { label: t("library.columns.duration"), value: durationLabel },
                  ]}
                />
              </TaskDialogHeaderCard>
            </div>

            <Tabs
              value={activeTab}
              onValueChange={(value) => setActiveTab(value as TaskDialogTab)}
              className="flex min-h-0 flex-1 flex-col gap-3"
            >
              <div className="flex justify-center">
                <TabsList>
                  <TabsTrigger value="status">
                    {t("library.task.tabStatus")}
                  </TabsTrigger>
                  <TabsTrigger value="outputs">
                    {t("library.task.tabOutputs")}
                  </TabsTrigger>
                  <TabsTrigger value="library">
                    {t("library.task.tabLibrary")}
                  </TabsTrigger>
                  <TabsTrigger value="info">
                    {t("library.task.tabInfo")}
                  </TabsTrigger>
                </TabsList>
              </div>

              <div className="flex min-h-0 flex-1 flex-col overflow-hidden">
                <TabsContent value="status" className="mt-0 flex min-h-0 flex-1 flex-col data-[state=inactive]:hidden">
                  <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
                    {operation ? <TaskDetailItemsCard items={statusItems} className="flex-1" /> : null}
                  </div>
                </TabsContent>

                <TabsContent value="outputs" className="mt-0 flex min-h-0 flex-1 flex-col data-[state=inactive]:hidden">
                  <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
                    <Card className="flex min-h-0 flex-1 flex-col border-border/70">
                      <div className="flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto">
                        {outputFiles.length === 0 ? (
                          <div className="px-3 py-2 text-xs text-muted-foreground">{t("library.table.noOutputs")}</div>
                        ) : (
                          outputFiles.map((file, index) => {
                            const Icon = resolveFileIcon({ fileType: file.fileType, path: file.path, name: file.name })
                            const isImage = isImageOperationFile(file)
                            return (
                              <React.Fragment key={file.id}>
                                {index > 0 ? <Separator /> : null}
                                <LibraryCellTooltip label={isImage ? t("library.tooltips.previewImage") : t("library.resources.fileInfoTooltip")}>
                                  <button
                                    type="button"
                                    className={cn(
                                      "flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left transition-colors",
                                      file.deleted ? "cursor-not-allowed text-muted-foreground line-through" : "hover:bg-muted/60",
                                    )}
                                    onClick={() => {
                                      if (isImage) {
                                        handlePreviewFile(file)
                                        return
                                      }
                                      void handleOpenFile(file)
                                    }}
                                    disabled={file.deleted}
                                  >
                                    <Icon className="h-4 w-4 shrink-0" />
                                    <span className="min-w-0 flex-1 truncate">
                                      {resolveOperationFilePrimaryLabel(file)}
                                    </span>
                                    {resolveOutputFormat(file) ? (
                                      <Badge variant="outline" className="text-xs font-medium">{resolveOutputFormat(file)}</Badge>
                                    ) : null}
                                    <span className="text-xs text-muted-foreground">{formatBytes(file.sizeBytes)}</span>
                                  </button>
                                </LibraryCellTooltip>
                              </React.Fragment>
                            )
                          })
                        )}
                      </div>
                    </Card>
                  </div>
                </TabsContent>

                <TabsContent value="library" className="mt-0 flex min-h-0 flex-1 flex-col data-[state=inactive]:hidden">
                  <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
                    <Card className="flex min-h-0 flex-1 flex-col border-border/70">
                      <div className="flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto">
                        {libraryFiles.length === 0 ? (
                          <div className="px-3 py-2 text-xs text-muted-foreground">{t("library.task.tabLibraryEmpty")}</div>
                        ) : (
                          libraryFiles.map((file, index) => {
                            const Icon = resolveFileIcon({ fileType: file.fileType, path: file.path, name: file.name })
                            const isImage = isImageOperationFile(file)
                            return (
                              <React.Fragment key={file.id}>
                                {index > 0 ? <Separator /> : null}
                                <LibraryCellTooltip label={isImage ? t("library.tooltips.previewImage") : t("library.resources.fileInfoTooltip")}>
                                  <button
                                    type="button"
                                    className={cn(
                                      "flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left transition-colors",
                                      file.deleted ? "cursor-not-allowed text-muted-foreground line-through" : "hover:bg-muted/60",
                                    )}
                                    onClick={() => {
                                      if (isImage) {
                                        handlePreviewFile(file)
                                        return
                                      }
                                      void handleOpenFile(file)
                                    }}
                                    disabled={file.deleted}
                                  >
                                    <Icon className="h-4 w-4 shrink-0" />
                                    <span className="min-w-0 flex-1 truncate">
                                      {resolveOperationFilePrimaryLabel(file)}
                                    </span>
                                    {resolveOutputFormat(file) ? (
                                      <Badge variant="outline" className="text-xs font-medium">{resolveOutputFormat(file)}</Badge>
                                    ) : null}
                                    <span className="text-xs text-muted-foreground">{formatBytes(file.sizeBytes)}</span>
                                    <span className="max-w-[40%] shrink-0 truncate text-xs text-muted-foreground">
                                      {resolveLibrarySourceLabel(file, t)}
                                    </span>
                                  </button>
                                </LibraryCellTooltip>
                              </React.Fragment>
                            )
                          })
                        )}
                      </div>
                    </Card>
                  </div>
                </TabsContent>

                <TabsContent value="info" className="mt-0 flex min-h-0 flex-1 flex-col data-[state=inactive]:hidden">
                  <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden pr-1">
                    <TaskDetailItemsCard items={infoItems} className="flex-1" emptyLabel="-" />
                  </div>
                </TabsContent>
              </div>
            </Tabs>
          </div>

          <DashboardDialogFooter className="sm:justify-between">
            <Button
              variant="destructive"
              size="compact"
              onClick={() => void handleDelete()}
              disabled={deleteOperation.isPending || !operationId}
            >
              <Trash2 className="h-4 w-4" />
              {t("library.task.delete")}
            </Button>
            <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center">
              <Button variant="outline" size="compact" onClick={closeTaskDialog}>
                {t("library.tools.dependencyClose")}
              </Button>
              {canResumeSubtitleTask ? (
                <Button
                  variant="outline"
                  size="compact"
                  onClick={() => void handleResumeSubtitleTask()}
                  disabled={resumeOperation.isPending || cancelOperation.isPending}
                >
                  {resumeOperation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <RotateCcw className="mr-2 h-4 w-4" />}
                  {t("library.task.resume")}
                </Button>
              ) : null}
              {canCancelSubtitleTask ? (
                <Button
                  variant="outline"
                  size="compact"
                  onClick={() => void handleCancelSubtitleTask()}
                  disabled={cancelOperation.isPending || resumeOperation.isPending}
                >
                  {cancelOperation.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Square className="mr-2 h-4 w-4" />}
                  {t("library.task.cancel")}
                </Button>
              ) : null}
            </div>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>

      <Dialog open={failureCheckOpen} onOpenChange={setFailureCheckOpen}>
        <DashboardDialogContent size="compact">
          <DashboardDialogHeader>
            <DialogTitle>{t("library.task.failedCheckTitle")}</DialogTitle>
            <DialogDescription className="sr-only">
              {t("library.task.failedCheckDescription")}
            </DialogDescription>
          </DashboardDialogHeader>
          {checkFailure.isPending ? (
            <div className="flex items-center gap-2 text-xs text-muted-foreground">
              <Loader2 className="h-4 w-4 animate-spin" />
              {t("library.task.failedChecking")}
            </div>
          ) : null}
          {failureCheckError ? <div className="text-xs text-destructive">{failureCheckError}</div> : null}
          {failureCheckResult ? (
            <Card className="overflow-hidden border-border/70">
              <div className="flex flex-col">
                {failureCheckResult.items.map((item, index) => (
                  <React.Fragment key={item.id}>
                    {index > 0 ? <Separator /> : null}
                    <div className="min-w-0 px-3 py-2">
                      <FailureCheckCard item={item} onOpenSettings={handleOpenSettings} t={t} />
                    </div>
                  </React.Fragment>
                ))}
              </div>
            </Card>
          ) : null}
          <DashboardDialogFooter>
            <Button variant="ghost" size="compact" onClick={() => setFailureCheckOpen(false)}>
              {t("library.tools.dependencyClose")}
            </Button>
            <Button size="compact" onClick={() => void handleRetryOperation()} disabled={!failureCheckResult?.canRetry || retryOperation.isPending}>
              {retryOperation.isPending ? <Loader2 className="mr-2 h-3.5 w-3.5 animate-spin" /> : null}
              {t("library.task.failedRetry")}
            </Button>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>

      <Dialog open={imagePreview !== null} onOpenChange={(nextOpen) => (!nextOpen ? setImagePreview(null) : undefined)}>
        <DashboardDialogContent size="workspace" className="flex max-h-[88vh] w-full flex-col overflow-hidden">
          <DashboardDialogHeader>
            <DialogTitle>{imagePreview?.name || t("library.preview.imageTitle")}</DialogTitle>
            <DialogDescription>{imagePreview?.path || "-"}</DialogDescription>
          </DashboardDialogHeader>
          <div className={cn("min-h-0 flex-1 overflow-auto p-3", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
            {imagePreviewUrl ? (
              <img
                src={imagePreviewUrl}
                alt={imagePreview?.name ?? ""}
                className="mx-auto block h-auto max-h-[72vh] w-auto max-w-full object-contain"
              />
            ) : (
              <div className="flex min-h-[240px] items-center justify-center text-xs text-muted-foreground">
                {t("library.preview.imageUnavailable")}
              </div>
            )}
          </div>
        </DashboardDialogContent>
      </Dialog>
    </>
  )
}

function TaskDialogHeaderCard({
  title,
  badge,
  className,
  children,
}: {
  title: string
  badge?: React.ReactNode
  className?: string
  children: React.ReactNode
}) {
  return (
    <section className={cn("flex min-w-0 flex-col overflow-hidden px-4 py-3", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS, className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 text-sm font-semibold text-foreground">{title}</div>
        {badge ? <div className="min-w-0 shrink">{badge}</div> : null}
      </div>
      <div className="mt-1.5 min-h-0 min-w-0 flex-1">{children}</div>
    </section>
  )
}

function TaskDialogSummaryRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 text-xs leading-5 text-muted-foreground">
      <span>{label}</span>
      <span className="min-w-0 text-right font-medium text-foreground">{value}</span>
    </div>
  )
}

function TaskDialogMetricsCard({
  items,
  columns = 3,
}: {
  items: Array<{ label: string; value: React.ReactNode }>
  columns?: 2 | 3
}) {
  const columnCount = columns === 2 ? 2 : 3
  return (
    <div
      className={cn(
        "grid min-w-0 overflow-hidden",
        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
        columns === 2 ? "grid-cols-2" : "grid-cols-3",
      )}
    >
      {items.map((item, index) => (
        <div
          key={`${item.label}-${index}`}
          className={cn(
            "min-w-0 px-2.5 py-2.5 sm:px-3",
            index % columnCount !== 0 && "border-l border-border/70",
            index >= columnCount && "border-t border-border/70",
          )}
        >
          <div className="overflow-hidden text-xs uppercase leading-tight tracking-[0.04em] text-muted-foreground">
            {item.label}
          </div>
          <div className="mt-1 min-h-5 min-w-0 truncate text-sm font-semibold text-foreground">{item.value}</div>
        </div>
      ))}
    </div>
  )
}

type TaskDetailMetaItem = {
  key: string
  label: string
  value: React.ReactNode
}

function TaskDetailCopyValue({
  value,
  label,
  successDescription,
  failureDescription,
  tone = "muted",
}: {
  value: string
  label: string
  successDescription: string
  failureDescription: string
  tone?: "muted" | "destructive"
}) {
  return (
    <div className="ml-auto flex w-full max-w-[16rem] min-w-0 items-center justify-end gap-1.5">
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-xs",
          tone === "destructive" ? "text-destructive" : "text-muted-foreground",
        )}
        title={value}
      >
        {value}
      </span>
      <TaskDetailCopyButton
        value={value}
        label={label}
        successDescription={successDescription}
        failureDescription={failureDescription}
      />
    </div>
  )
}

function TaskDetailSourceURLValue({ value }: { value: string }) {
  const { t } = useI18n()

  return (
    <TaskDetailCopyValue
      value={value}
      label={t("library.task.copySourceUrl")}
      successDescription={t("library.task.copySourceUrlSuccess")}
      failureDescription={t("library.task.copySourceUrlFailed")}
    />
  )
}

function TaskDetailErrorValue({ value }: { value: string }) {
  const { t } = useI18n()

  return (
    <TaskDetailCopyValue
      value={value}
      label={t("library.task.copyError")}
      successDescription={t("library.task.copyErrorSuccess")}
      failureDescription={t("library.task.copyErrorFailed")}
      tone="destructive"
    />
  )
}

function TaskDetailCopyButton({
  value,
  label,
  successDescription,
  failureDescription,
}: {
  value: string
  label: string
  successDescription: string
  failureDescription: string
}) {
  const handleCopy = React.useCallback(async () => {
    if (typeof navigator === "undefined" || !navigator.clipboard) {
      messageBus.publishToast({
        intent: "danger",
        title: label,
        description: failureDescription,
      })
      return
    }
    try {
      await navigator.clipboard.writeText(value)
      messageBus.publishToast({
        intent: "success",
        title: label,
        description: successDescription,
      })
    } catch {
      messageBus.publishToast({
        intent: "danger",
        title: label,
        description: failureDescription,
      })
    }
  }, [failureDescription, label, successDescription, value])

  return (
    <LibraryCellTooltip label={label}>
        <Button
          type="button"
          variant="ghost"
          size="compactIcon"
          className="h-6 w-6 shrink-0"
          onClick={() => void handleCopy()}
          aria-label={label}
        >
          <Copy className="h-3.5 w-3.5" />
        </Button>
      </LibraryCellTooltip>
  )
}

function TaskDetailMetaGroup({ items }: { items: TaskDetailMetaItem[] }) {
  return (
    <div className="space-y-2">
      {items.map((item) => (
        <div
          key={item.key}
          className="grid min-w-0 grid-cols-[max-content_minmax(0,1fr)] items-center gap-3 overflow-hidden text-xs"
        >
          <span className="whitespace-nowrap text-muted-foreground">{item.label}</span>
          <div className="min-w-0 overflow-hidden text-right text-foreground">{item.value}</div>
        </div>
      ))}
    </div>
  )
}

function TaskDetailItemsCard({
  items,
  className,
  emptyLabel,
}: {
  items: TaskInfoItem[]
  className?: string
  emptyLabel?: string
}) {
  return (
    <Card className={cn("flex min-h-0 flex-col border-border/70", className)}>
      <div className="flex min-h-0 flex-1 flex-col overflow-x-hidden overflow-y-auto">
        {items.length === 0 ? (
          <div className="px-3 py-2 text-xs text-muted-foreground">{emptyLabel ?? "-"}</div>
        ) : (
          items.map((item, index) => (
            <React.Fragment key={item.key}>
              {index > 0 ? <Separator /> : null}
              {item.stacked ? (
                <div className="min-w-0 overflow-hidden px-3 py-2">
                  <div className="text-xs text-muted-foreground">{item.label}</div>
                  <div className="mt-2 border-t border-border/60 pt-2 text-foreground">{item.value}</div>
                </div>
              ) : (
                <div className="grid min-h-11 min-w-0 grid-cols-[max-content_minmax(0,1fr)] items-center gap-3 overflow-hidden px-3 py-2">
                  <span className="shrink-0 whitespace-nowrap text-muted-foreground">{item.label}</span>
                  <div className="flex min-w-0 flex-1 items-center justify-end text-right text-foreground">
                    {item.value}
                  </div>
                </div>
              )}
            </React.Fragment>
          ))
        )}
      </div>
    </Card>
  )
}

function FailureCheckCard(props: {
  item: CheckYtdlpOperationFailureItem
  onOpenSettings: (section: "external-tools" | "general") => void
  t: (key: string) => string
}) {
  const label = resolveFailureCheckLabel(props.item.id, props.item.label, props.t)
  const isOK = props.item.status === "ok"
  const statusLabel = isOK
    ? props.t("library.task.failedCheckOk")
    : props.t("library.task.failedCheckFailed")
  const StatusIcon = isOK ? CheckCircle2 : XCircle
  return (
    <div className="flex flex-col gap-2 text-xs">
      <div className="flex items-center justify-between gap-2">
        <span className="truncate text-foreground">{label}</span>
        <span title={statusLabel}>
          <StatusIcon
            className={cn("h-4 w-4 shrink-0", isOK ? "text-emerald-500" : "text-rose-500")}
            aria-label={statusLabel}
          />
        </span>
      </div>
      {props.item.message ? <div className="break-words text-muted-foreground">{resolveFailureCheckMessage(props.item, props.t)}</div> : null}
      {props.item.action && props.item.status === "failed" ? (
        <div className="flex justify-end">
          <Button size="compact" variant="outline" className="h-7 text-xs" onClick={() => props.onOpenSettings(props.item.action === "general" ? "general" : "external-tools")}>
            {props.item.action === "general"
              ? props.t("library.task.failedCheckOpenProxy")
              : props.t("library.task.failedCheckOpenTools")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function ExpandableTextBlock(props: {
  text?: string
  collapsedChars?: number
  tone?: "default" | "destructive"
  inline?: boolean
  align?: "start" | "end"
}) {
  const { t } = useI18n()
  const [expanded, setExpanded] = React.useState(false)
  const text = props.text?.trim() ?? ""
  const collapsedChars = props.collapsedChars ?? 180
  const canExpand = text.length > collapsedChars || text.includes("\n")
  const displayText = !canExpand || expanded ? text : `${text.slice(0, collapsedChars).trimEnd()}…`

  if (!text) {
    return <span className="text-xs text-muted-foreground">-</span>
  }

  return (
    <div
      className={cn(
        "min-w-0",
        props.inline && !expanded
          ? "flex max-w-full items-center justify-end gap-2"
          : cn("space-y-1", props.align === "end" && "text-right"),
      )}
    >
      <div
        className={cn(
          "min-w-0 text-xs",
          props.inline && !expanded
            ? cn("flex-1 truncate", props.align === "end" && "text-right")
            : cn("whitespace-pre-wrap break-words", props.align === "end" && "text-right"),
          props.tone === "destructive" ? "text-destructive" : "text-muted-foreground",
        )}
      >
        {displayText}
      </div>
      {canExpand ? (
        <div className={cn(props.inline && !expanded ? "shrink-0" : props.align === "end" && "flex justify-end")}>
          <Button
            type="button"
            variant="outline"
            size="compact"
            className="h-7 text-xs"
            onClick={() => setExpanded((value) => !value)}
          >
            {expanded ? t("common.collapse") : t("common.expand")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function TaskProgressValue({
  operation,
  inline,
}: {
  operation?: LibraryOperationDTO
  inline?: boolean
}) {
  const { t } = useI18n()
  const progress = operation?.progress
  const displaySpeed = useSmoothedProgressSpeed(
    progress?.speed,
    progress?.updatedAt || `${progress?.stage || progress?.message || ""}|${progress?.percent ?? ""}|${progress?.message ?? ""}`,
    { enabled: operation?.status === "running" },
  )

  if (!progress) {
    return <span className="text-xs text-muted-foreground">-</span>
  }

  const parts = [translateLibraryProgressLabel(progress.stage || progress.message || "", t)]
  if (progress.percent !== undefined && progress.percent !== null) {
    parts.push(`${Math.round(progress.percent)}%`)
  }
  if (displaySpeed) {
    parts.push(displaySpeed)
  }

  const detail = resolveProgressDetail(operation, t)
  const badge = (
    <span className="inline-flex shrink-0">
      <Badge variant="subtle" className="max-w-full gap-1 text-xs font-medium">
        {parts.filter(Boolean).join(" · ")}
      </Badge>
    </span>
  )

  if (inline) {
    return (
      <div className="flex min-w-0 items-center justify-end gap-2">
        {badge}
        {detail ? <ExpandableTextBlock text={detail} inline align="end" /> : null}
      </div>
    )
  }

  return (
    <div className="min-w-0 space-y-1">
      <span className="inline-flex max-w-full">{badge}</span>
      {detail ? <ExpandableTextBlock text={detail} /> : null}
    </div>
  )
}

function TaskDialogStatusBadge({
  status,
  t,
  phaseKey = "",
}: {
  status: string
  t: (key: string) => string
  phaseKey?: string
}) {
  const meta = STATUS_META[status]
  const Icon = meta?.Icon ?? Clock
  const label = meta ? t(meta.labelKey) : status || t("library.status.unknown")
  const spinDelay = useTimeSyncedSpinDelay(phaseKey)
  return (
    <span className={cn("inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium", meta?.className ?? "bg-muted text-muted-foreground")}>
      <Icon
        className={cn("h-3 w-3", status === "running" ? "animate-spin" : "")}
        style={status === "running" ? { animationDelay: spinDelay } : undefined}
      />
      {label}
    </span>
  )
}

function resolveProgressDetail(
  operation: LibraryOperationDTO | undefined,
  t: (key: string) => string,
) {
  const progress = operation?.progress
  if (!progress?.message?.trim()) {
    return ""
  }

  const message = progress.message.trim()
  const stage = progress.stage?.trim() ?? ""
  const status = operation?.status?.trim().toLowerCase() ?? ""
  const errorMessage = operation?.errorMessage?.trim() ?? ""

  if (message === stage || message === errorMessage) {
    return ""
  }
  if (status === "failed" || status === "canceled") {
    return ""
  }
  return translateLibraryProgressDetail(message, t)
}

function formatOutputCountValue(operation: LibraryOperationDTO, outputCount: number) {
  const count = operation.metrics.fileCount ?? outputCount
  return String(count)
}

function formatOutputSizeValue(operation: LibraryOperationDTO, outputFiles: OperationFileItem[]) {
  const metricsSize = operation.metrics.totalSizeBytes
  if (typeof metricsSize === "number" && Number.isFinite(metricsSize) && metricsSize >= 0) {
    return formatBytes(metricsSize)
  }
  const fallbackSize = outputFiles.reduce((total, file) => total + (file.sizeBytes ?? 0), 0)
  return formatBytes(fallbackSize)
}

function formatOperationDurationValue(operation: LibraryOperationDTO) {
  const directDuration = formatDuration(operation.startedAt, operation.finishedAt)
  if (directDuration !== "-") {
    return directDuration
  }
  return formatDurationMilliseconds(operation.metrics.durationMs)
}

function formatDurationMilliseconds(value?: number | null) {
  if (typeof value !== "number" || !Number.isFinite(value) || value < 0) {
    return "-"
  }
  let totalSeconds = Math.max(0, Math.round(value / 1000))
  const hours = Math.floor(totalSeconds / 3600)
  totalSeconds -= hours * 3600
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds - minutes * 60
  const parts: string[] = []
  if (hours > 0) {
    parts.push(`${hours}h`)
  }
  if (minutes > 0) {
    parts.push(`${minutes}m`)
  }
  if (parts.length === 0 || seconds > 0) {
    parts.push(`${seconds}s`)
  }
  return parts.join(" ")
}

function resolveTaskDialogFileDisplayLabel(
  file: Pick<LibraryFileDTO, "id" | "name" | "displayLabel"> | undefined,
  workspaceTrackLabelByFileId: Map<string, string>,
  fallback: string,
) {
  const fileId = file?.id?.trim() ?? ""
  if (fileId) {
    const trackLabel = workspaceTrackLabelByFileId.get(fileId)?.trim() ?? ""
    if (trackLabel) {
      return trackLabel
    }
  }
  const displayLabel = file?.displayLabel?.trim() ?? ""
  if (displayLabel) {
    return displayLabel
  }
  const fileName = file?.name?.trim() ?? ""
  if (fileName) {
    return fileName
  }
  return fallback
}

function resolveOperationFilePrimaryLabel(file: OperationFileItem) {
  const displayLabel = file.displayLabel?.trim() ?? ""
  if (displayLabel) {
    return displayLabel
  }
  const fileName = file.name?.trim() ?? ""
  if (fileName) {
    return fileName
  }
  return file.id
}

function resolveSourceLabel(operation: LibraryOperationDTO, t: (key: string) => string) {
  if (operation.kind === "download") {
    return t("library.jobType.ytdlp")
  }
  if (operation.kind === "transcode") {
    return t("library.jobType.transcode")
  }
  if (operation.kind === "subtitle_translate") {
    return t("library.jobType.subtitleTranslate")
  }
  if (operation.kind === "subtitle_proofread") {
    return t("library.jobType.subtitleProofread")
  }
  if (operation.kind === "subtitle_qa_review") {
    return t("library.jobType.subtitleQaReview")
  }
  return operation.kind
}

function resolveOutputFormat(file: OperationFileItem) {
  const explicit = file.format?.trim()
  if (explicit) {
    return explicit.toUpperCase()
  }
  const target = file.path || file.name
  const dotIndex = target.lastIndexOf(".")
  if (dotIndex <= 0 || dotIndex >= target.length - 1) {
    return ""
  }
  return target.slice(dotIndex + 1).toUpperCase()
}

function resolveLibrarySourceLabel(file: OperationFileItem, t: (key: string) => string) {
  return resolveOriginKindLabel(file.sourceType, t)
}

function resolveOriginKindLabel(kind: string | undefined, t: (key: string) => string) {
  const normalized = kind?.trim().toLowerCase() ?? ""
  switch (normalized) {
    case "download":
      return t("library.task.download")
    case "transcode":
      return t("library.workspace.actions.transcode")
    case "subtitle_translate":
      return t("library.workspace.actions.translate")
    case "subtitle_proofread":
      return t("library.workspace.actions.proofread")
    case "subtitle_qa_review":
      return t("library.workspace.header.qa")
    case "import_video":
      return t("library.actions.importVideo")
    case "import_subtitle":
      return t("library.actions.importSubtitle")
    case "import":
      return t("library.type.import")
    default:
      return kind?.trim() || t("library.source.unknown")
  }
}

function isResumableSubtitleTask(kind: string | undefined) {
  const normalized = kind?.trim().toLowerCase() ?? ""
  return normalized === "subtitle_translate" || normalized === "subtitle_proofread"
}

function parseSubtitleTaskOutput(operation: LibraryOperationDTO | undefined): SubtitleTaskOutputSummary | null {
  if (!operation || !isResumableSubtitleTask(operation.kind) || !operation.outputJson?.trim()) {
    return null
  }
  try {
    const parsed = JSON.parse(operation.outputJson) as SubtitleTaskOutputSummary
    return parsed && typeof parsed === "object" ? parsed : null
  } catch {
    return null
  }
}

function formatChunkProgress(output: SubtitleTaskOutputSummary) {
  const total = output.chunkCount ?? 0
  const completed = output.completedChunkCount ?? 0
  return `${completed} / ${total}`
}

function formatSubtitleConstraintSummary(
  kind: string | undefined,
  output: SubtitleTaskOutputSummary | null,
  t: (key: string) => string,
) {
  if (!output) {
    return ""
  }
  const parts: string[] = []
  if (kind?.trim().toLowerCase() === "subtitle_translate") {
    if (output.glossaryProfileIds?.length) {
      parts.push(`${output.glossaryProfileIds.length} ${t("library.task.glossaryProfiles")}`)
    }
    if (output.referenceTrackFileIds?.length) {
      parts.push(`${output.referenceTrackFileIds.length} ${t("library.task.referenceTracks")}`)
    }
  }
  if (kind?.trim().toLowerCase() === "subtitle_proofread" && output.glossaryProfileIds?.length) {
    parts.push(`${output.glossaryProfileIds.length} ${t("library.task.glossaryProfiles")}`)
  }
  if (output.passes?.length) {
    parts.push(output.passes.join(", "))
  }
  if (output.promptProfileIds?.length) {
    parts.push(`${output.promptProfileIds.length} ${t("library.task.promptProfiles")}`)
  }
  if (output.inlinePromptHash) {
    parts.push(t("library.task.inlinePrompt"))
  }
  return parts.join(" · ")
}

function normalizeThumbnailUrl(value: string) {
  const trimmed = value.trim()
  if (!trimmed) {
    return ""
  }
  switch (trimmed.toLowerCase()) {
    case "na":
    case "n/a":
    case "null":
    case "none":
      return ""
    default:
      return trimmed
  }
}

function buildAssetPreviewUrl(baseUrl: string, path: string) {
  if (!baseUrl || !path) {
    return ""
  }
  const trimmed = baseUrl.replace(/\/+$/, "")
  const previewName = path.replace(/\\/g, "/").split("/").pop()?.trim() || "asset"
  return `${trimmed}/api/library/asset/${encodeURIComponent(previewName)}?path=${encodeURIComponent(path)}`
}

function isImageOperationFile(file: Pick<OperationFileItem, "fileType" | "path" | "name">) {
  const fileType = normalizeFileType(file.fileType)
  if (fileType === "thumbnail" || fileType === "image") {
    return true
  }
  const extension = extractExtension(file.path) || extractExtension(file.name)
  return IMAGE_EXTENSIONS.has(extension)
}

function normalizeFileType(value?: string) {
  return value?.trim().toLowerCase() ?? ""
}

function extractExtension(value?: string) {
  if (!value) {
    return ""
  }
  const normalized = value.replace(/\\/g, "/")
  const baseName = normalized.split("/").pop() ?? ""
  const dotIndex = baseName.lastIndexOf(".")
  if (dotIndex <= 0 || dotIndex === baseName.length - 1) {
    return ""
  }
  return baseName.slice(dotIndex + 1).trim().toLowerCase()
}

function isWorkspaceFileType(fileType: string) {
  const normalized = fileType.trim().toLowerCase()
  return normalized === "video" || normalized === "audio" || normalized === "subtitle" || normalized === "transcode"
}

function normalizeWorkspaceMode(fileType: string): "video" | "subtitle" {
  return fileType.trim().toLowerCase() === "subtitle" ? "subtitle" : "video"
}

function resolveFailureCheckLabel(id: string, fallback: string, t: (key: string) => string) {
  switch (id) {
    case "connectivity":
      return t("library.task.checks.connectivity")
    case "yt-dlp":
      return t("library.task.checks.ytdlp")
    case "ffmpeg":
      return t("library.task.checks.ffmpeg")
    case "bun":
      return t("library.task.checks.bun")
    case "yt-dlp-version":
      return t("library.task.checks.version")
    default:
      return fallback || id
  }
}

function resolveFailureCheckMessage(item: CheckYtdlpOperationFailureItem, t: (key: string) => string) {
  const message = item.message?.trim()
  if (!message) {
    return ""
  }
  const latestMatch = message.match(/^(.+?)\s*\(latest\)$/i)
  if (latestMatch) {
    return formatTemplate(t("library.task.checks.versionLatest"), { version: latestMatch[1].trim() })
  }
  return message
}

const IMAGE_EXTENSIONS = new Set([
  "apng",
  "avif",
  "bmp",
  "gif",
  "heic",
  "heif",
  "jfif",
  "jpeg",
  "jpg",
  "png",
  "svg",
  "tif",
  "tiff",
  "webp",
])

function resolveErrorMessage(error: unknown, fallback = "Unknown error") {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === "string") {
    return error
  }
  return fallback
}
