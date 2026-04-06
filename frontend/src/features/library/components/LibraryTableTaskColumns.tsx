import type { ColumnDef } from "@tanstack/react-table"
import { AlertTriangle, Ban, CheckCircle2, Clock, Loader2 } from "lucide-react"

import { cn } from "@/lib/utils"

import type { LibraryTaskRow } from "../model/types"
import { useTimeSyncedSpinDelay } from "../utils/progress-display"
import { formatBytes } from "../utils/format"
import { formatTemplate } from "../utils/i18n"
import { formatRelativeTime } from "../utils/time"
import { LibraryProgressBadge } from "./LibraryProgressBadge"
import { LibraryRowMenu } from "./LibraryRowMenu"
import { LibraryCellTooltip } from "./LibraryCellTooltip"
import { LibraryTaskIcon } from "./LibraryTaskIcon"

type Translator = (key: string) => string

type TaskColumnOptions = {
  onRenameTask?: (id: string, name: string) => void | Promise<void>
  onDeleteTask?: (id: string, deleteFiles: boolean) => void | Promise<void>
  onOpenTaskDialog?: (taskId: string) => void
  onOpenLibrary?: (libraryId: string) => void
  language?: string
  t: Translator
}

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

function TaskStatusBadge({ status, t, phaseKey = "" }: { status: string; t: Translator; phaseKey?: string }) {
  const meta = STATUS_META[status]
  const Icon = meta?.Icon ?? Clock
  const label = meta ? t(meta.labelKey) : status || t("library.status.unknown")
  const spinDelay = useTimeSyncedSpinDelay(phaseKey)
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
        meta?.className ?? "bg-muted text-muted-foreground"
      )}
    >
      <Icon
        className={cn("h-3 w-3", status === "running" ? "animate-spin" : "")}
        style={status === "running" ? { animationDelay: spinDelay } : undefined}
      />
      {label}
    </span>
  )
}

export function getTaskColumns({
  onRenameTask,
  onDeleteTask,
  onOpenTaskDialog,
  onOpenLibrary,
  language,
  t,
}: TaskColumnOptions): ColumnDef<LibraryTaskRow>[] {
  const tooltips = {
    openTaskDialog: t("library.tooltips.openTaskDialog"),
    outputCount: t("library.tooltips.outputCount"),
    outputSize: t("library.tooltips.outputSize"),
  }
  return [
    {
      id: "name",
      accessorKey: "name",
      header: t("library.columns.taskName"),
      meta: { label: t("library.columns.taskName") },
      enableHiding: false,
      cell: ({ row }) => {
        const task = row.original
        return (
          <div className="flex w-full min-w-0 items-center gap-2">
            <LibraryCellTooltip label={tooltips.openTaskDialog}>
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-2 text-left text-sm font-medium text-foreground hover:text-foreground/80"
                onClick={() => onOpenTaskDialog?.(task.id)}
              >
                <LibraryTaskIcon
                  taskType={task.taskType}
                  sourceDomain={task.sourceDomain}
                  sourceIcon={task.sourceIcon}
                />
                <span className="min-w-0 truncate">{task.name}</span>
              </button>
            </LibraryCellTooltip>
          </div>
        )
      },
    },
    {
      id: "library",
      accessorKey: "libraryName",
      header: t("library.columns.library"),
      meta: { label: t("library.columns.library") },
      cell: ({ row }) => {
        const libraryId = row.original.libraryId?.trim() ?? ""
        const label = row.original.libraryName ?? row.original.libraryId ?? "-"
        if (!libraryId || !onOpenLibrary) {
          return <span className="text-sm whitespace-nowrap">{label}</span>
        }
        return (
          <button
            type="button"
            className="max-w-full truncate text-left text-sm text-foreground hover:text-foreground/80"
            onClick={() => onOpenLibrary(libraryId)}
          >
            {label}
          </button>
        )
      },
    },
    {
      id: "domain",
      accessorKey: "sourceDomain",
      header: t("library.columns.domain"),
      meta: { label: t("library.columns.domain") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{row.original.sourceDomain ?? "-"}</span>
      ),
    },
    {
      id: "action",
      accessorKey: "taskType",
      header: t("library.columns.type"),
      meta: { label: t("library.columns.type") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{row.original.taskTypeLabel || row.original.taskType || "-"}</span>
      ),
    },
    {
      id: "platform",
      accessorKey: "platform",
      header: t("library.columns.platform"),
      meta: { label: t("library.columns.platform") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{row.original.platform ?? "-"}</span>
      ),
    },
    {
      id: "uploader",
      accessorKey: "uploader",
      header: t("library.columns.uploader"),
      meta: { label: t("library.columns.uploader") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{row.original.uploader ?? "-"}</span>
      ),
    },
    {
      id: "status",
      accessorKey: "status",
      header: t("library.columns.status"),
      meta: { label: t("library.columns.status") },
      cell: ({ row }) => <TaskStatusBadge status={row.original.status} t={t} phaseKey={row.original.id} />,
    },
    {
      id: "progress",
      accessorKey: "progress",
      header: t("library.columns.progress"),
      meta: { label: t("library.columns.progress") },
      cell: ({ row }) => (
        <LibraryProgressBadge progress={row.original.progress} status={row.original.status} disableTooltip />
      ),
    },
    {
      id: "outputCount",
      accessorKey: "outputs",
      header: t("library.columns.outputCount"),
      meta: { label: t("library.columns.outputCount") },
      cell: ({ row }) => {
        const outputs = row.original.outputs
        const outputCount = outputs?.count ?? 0
        const outputSizeLabel = formatOutputSize(outputs?.sizeBytes ?? null, outputs?.deletedCount ?? 0, t)
        const tooltipContent = (
          <div className="flex flex-col gap-0.5">
            <span>{formatTemplate(tooltips.outputCount, { count: outputCount })}</span>
            <span>{formatTemplate(tooltips.outputSize, { size: outputSizeLabel })}</span>
          </div>
        )
        return (
          <LibraryCellTooltip label={tooltipContent}>
            <span className="text-sm whitespace-nowrap">{outputCount}</span>
          </LibraryCellTooltip>
        )
      },
    },
    {
      id: "outputSize",
      accessorKey: "outputs",
      header: t("library.columns.outputSize"),
      meta: { label: t("library.columns.outputSize") },
      cell: ({ row }) => {
        const outputs = row.original.outputs
        const outputCount = outputs?.count ?? 0
        const outputSizeLabel = formatOutputSize(outputs?.sizeBytes ?? null, outputs?.deletedCount ?? 0, t)
        const tooltipContent = (
          <div className="flex flex-col gap-0.5">
            <span>{formatTemplate(tooltips.outputCount, { count: outputCount })}</span>
            <span>{formatTemplate(tooltips.outputSize, { size: outputSizeLabel })}</span>
          </div>
        )
        return (
          <LibraryCellTooltip label={tooltipContent}>
            <span className="text-sm whitespace-nowrap">{outputSizeLabel}</span>
          </LibraryCellTooltip>
        )
      },
    },
    {
      id: "duration",
      accessorKey: "duration",
      header: t("library.columns.duration"),
      meta: { label: t("library.columns.duration") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{row.original.duration ?? "-"}</span>
      ),
    },
    {
      id: "publishTime",
      accessorKey: "publishedAt",
      header: t("library.columns.publishTime"),
      meta: { label: t("library.columns.publishTime") },
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground whitespace-nowrap">
          {formatRelativeTime(row.original.publishedAt, language)}
        </span>
      ),
    },
    {
      id: "startedAt",
      accessorKey: "startedAt",
      header: t("library.columns.startedAt"),
      meta: { label: t("library.columns.startedAt") },
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground whitespace-nowrap">
          {formatRelativeTime(row.original.startedAt, language)}
        </span>
      ),
    },
    {
      id: "createdAt",
      accessorKey: "createdAt",
      header: t("library.columns.createTime"),
      meta: { label: t("library.columns.createTime") },
      cell: ({ row }) => (
        <span className="text-sm text-muted-foreground whitespace-nowrap">
          {formatRelativeTime(row.original.createdAt, language)}
        </span>
      ),
    },
    {
      id: "actions",
      header: "",
      enableHiding: false,
      cell: ({ row }) => {
        const task = row.original
        const candidateFiles = (task.libraryFiles && task.libraryFiles.length > 0 ? task.libraryFiles : task.outputFiles) ?? []
        const deleteFileItems =
          candidateFiles
            ?.filter((file) => !file.deleted)
            .map((file) => ({
              id: file.id,
              name: file.label,
              path: file.path,
              fileType: file.fileType,
              format: undefined,
            }))
            .filter((item) => item.name || item.path) ?? []
        return (
          <div className="flex justify-end">
            <LibraryRowMenu
              itemName={task.name}
              deleteFileItems={deleteFileItems}
              onRename={onRenameTask ? (name) => onRenameTask(task.id, name) : undefined}
              showDeleteFilesToggle={deleteFileItems.length > 0}
              defaultDeleteFiles={false}
              deleteTitle={t("library.task.deleteTitle")}
              deleteDescription={formatTemplate(
                t("library.task.deleteDescription"),
                { name: task.name || t("library.rowMenu.renameFallback") }
              )}
              deleteImpactDescription={
                deleteFileItems.length > 0
                  ? formatTemplate(
                      t("library.task.deleteImpactDescription"),
                      { count: deleteFileItems.length }
                    )
                  : t("library.task.deleteKeepFilesDescription")
              }
              deleteFilesLabel={t("library.task.deleteFilesLabel")}
              deleteFilesTitle={t("library.task.deleteFilesTitle")}
              deleteConfirmLabel={t("library.task.deleteConfirm")}
              onDelete={onDeleteTask ? ({ deleteFiles }) => onDeleteTask(task.id, deleteFiles) : undefined}
            />
          </div>
        )
      },
    },
  ]
}

function formatOutputSize(
  sizeBytes: number | null | undefined,
  deletedCount: number,
  t: Translator
) {
  if (sizeBytes === 0 && deletedCount > 0) {
    return t("library.outputs.zeroSize")
  }
  return formatBytes(sizeBytes)
}
