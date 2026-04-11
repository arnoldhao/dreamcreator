import type { ColumnDef } from "@tanstack/react-table"
import { AlertTriangle, CheckCircle2, Clock, Loader2 } from "lucide-react"

import { Badge } from "@/shared/ui/badge"
import { cn } from "@/lib/utils"

import type { LibraryFileRow } from "../model/types"
import { useTimeSyncedSpinDelay } from "../utils/progress-display"
import { formatBytes } from "../utils/format"
import { formatRelativeTime } from "../utils/time"
import { resolveFileIcon } from "../utils/fileIcons"
import { LibraryCellTooltip } from "./LibraryCellTooltip"

type Translator = (key: string) => string

type FileColumnOptions = {
  onOpenWorkspace?: (file: LibraryFileRow) => void
  onPreviewImage?: (file: LibraryFileRow) => void
  onOpenPath?: (path: string) => void
  onCreateTranscode?: (file: LibraryFileRow) => void | Promise<void>
  onCreateSubtitleTranslate?: (file: LibraryFileRow) => void | Promise<void>
  onRenameFile?: (id: string, name: string) => void | Promise<void>
  onDeleteFile?: (id: string, deleteFiles: boolean) => void | Promise<void>
  onOpenTaskDialog?: (taskId: string) => void
  language?: string
  t: Translator
}

const STATUS_META: Record<
  string,
  { labelKey: string; defaultLabel: string; className: string; Icon: typeof Clock }
> = {
  existence: {
    labelKey: "library.status.existence",
    defaultLabel: "Available",
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-100",
    Icon: CheckCircle2,
  },
  "non-existence": {
    labelKey: "library.status.nonExistence",
    defaultLabel: "Missing",
    className: "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-100",
    Icon: AlertTriangle,
  },
  deleted: {
    labelKey: "library.status.deleted",
    defaultLabel: "Deleted",
    className: "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-100",
    Icon: AlertTriangle,
  },
  ready: {
    labelKey: "library.status.ready",
    defaultLabel: "Ready",
    className: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/50 dark:text-emerald-100",
    Icon: CheckCircle2,
  },
  running: {
    labelKey: "library.status.running",
    defaultLabel: "In progress",
    className: "bg-blue-100 text-blue-800 dark:bg-blue-900/50 dark:text-blue-100",
    Icon: Loader2,
  },
  failed: {
    labelKey: "library.status.failed",
    defaultLabel: "Failed",
    className: "bg-red-100 text-red-800 dark:bg-red-900/50 dark:text-red-100",
    Icon: AlertTriangle,
  },
}

function FileStatusBadge({ status, t, phaseKey = "" }: { status: string; t: Translator; phaseKey?: string }) {
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

export function getFileColumns({
  onOpenWorkspace,
  onPreviewImage,
  onOpenPath,
  onCreateTranscode,
  onCreateSubtitleTranslate,
  onRenameFile,
  onDeleteFile,
  onOpenTaskDialog,
  language,
  t,
}: FileColumnOptions): ColumnDef<LibraryFileRow>[] {
  const tooltips = {
    openWorkspace: t("library.resources.fileInfoTooltip"),
    previewImage: t("library.tooltips.previewImage"),
    openTaskDialog: t("library.tooltips.openTaskDialog"),
  }
  return [
    {
      id: "name",
      accessorFn: (row) => row.displayLabel || row.name,
      header: t("library.columns.file"),
      meta: { label: t("library.columns.file") },
      enableHiding: false,
      cell: ({ row }) => {
        const file = row.original
        const FileIcon = resolveFileIcon({ fileType: file.fileType, path: file.path, name: file.name })
        const isThumbnail = normalizeFileType(file.fileType) === "thumbnail"
        const isDeleted = file.status === "deleted"
        const primaryLabel = file.displayLabel?.trim() || file.name
        const tooltipLabel = file.displayLabel && file.displayLabel !== file.name
          ? file.name
          : isThumbnail
            ? tooltips.previewImage
            : tooltips.openWorkspace
        if (isDeleted) {
          return (
            <div className="flex w-full min-w-0 items-center gap-2 text-left text-sm font-medium text-muted-foreground">
              <FileIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span className="min-w-0 truncate">{primaryLabel}</span>
            </div>
          )
        }
        return (
          <LibraryCellTooltip label={tooltipLabel}>
            <button
              type="button"
              className="flex w-full min-w-0 items-center gap-2 text-left text-sm font-medium text-foreground hover:text-foreground/80"
              onClick={() => {
                if (isThumbnail) {
                  onPreviewImage?.(file)
                  return
                }
                onOpenWorkspace?.(file)
              }}
            >
              <FileIcon className="h-4 w-4 shrink-0 text-muted-foreground" />
              <span className="min-w-0 truncate">{primaryLabel}</span>
            </button>
          </LibraryCellTooltip>
        )
      },
    },
    {
      id: "source",
      accessorKey: "typeLabel",
      header: t("library.columns.source"),
      meta: { label: t("library.columns.source") },
      cell: ({ row }) => (
        <span className="inline-flex">
          <Badge variant="subtle">{row.original.typeLabel}</Badge>
        </span>
      ),
    },
    {
      id: "status",
      accessorKey: "status",
      header: t("library.columns.status"),
      meta: { label: t("library.columns.status") },
      cell: ({ row }) => <FileStatusBadge status={row.original.status} t={t} phaseKey={row.original.id} />,
    },
    {
      id: "size",
      accessorKey: "sizeBytes",
      header: t("library.columns.size"),
      meta: { label: t("library.columns.size") },
      cell: ({ row }) => (
        <span className="text-sm whitespace-nowrap">{formatBytes(row.original.sizeBytes)}</span>
      ),
    },
    {
      id: "task",
      accessorKey: "taskName",
      header: t("library.columns.task"),
      meta: { label: t("library.columns.task") },
      cell: ({ row }) => (
        <LibraryCellTooltip label={tooltips.openTaskDialog}>
          {row.original.taskId && onOpenTaskDialog ? (
            <button
              type="button"
              className="block max-w-full truncate text-left text-sm text-foreground hover:text-foreground/80"
              onClick={() => onOpenTaskDialog(row.original.taskId ?? "")}
            >
              {row.original.taskName ?? "-"}
            </button>
          ) : (
            <span className="block max-w-full truncate text-sm text-muted-foreground">
              {row.original.taskName ?? "-"}
            </span>
          )}
        </LibraryCellTooltip>
      ),
    },
    {
      id: "fileFormat",
      accessorFn: (row) => resolveFileFormat(row),
      header: t("library.columns.fileFormat"),
      meta: { label: t("library.columns.fileFormat") },
      cell: ({ row }) => {
        const format = resolveFileFormat(row.original)
        if (!format) {
          return (
            <span className="text-sm text-muted-foreground">-</span>
          )
        }
        return (
          <span className="inline-flex">
            <Badge variant="subtle">{format}</Badge>
          </span>
        )
      },
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
  ]
}

function normalizeFileType(value?: string) {
  return value?.trim().toLowerCase() ?? ""
}

function resolveFileFormat(file: LibraryFileRow) {
  const explicit = file.format?.trim()
  if (explicit) {
    return explicit.toUpperCase()
  }
  const byPath = extractExtension(file.path)
  if (byPath) {
    return byPath.toUpperCase()
  }
  const byName = extractExtension(file.name)
  if (byName) {
    return byName.toUpperCase()
  }
  return ""
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
  return baseName.slice(dotIndex + 1).trim()
}
