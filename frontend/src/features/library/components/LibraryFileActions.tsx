import * as React from "react"
import { createPortal } from "react-dom"
import { FolderOpen, Trash2 } from "lucide-react"

import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
  DashboardDialogSection,
} from "@/shared/ui/dashboard-dialog"
import { Dialog, DialogDescription, DialogTitle } from "@/shared/ui/dialog"
import { Separator } from "@/shared/ui/separator"
import { Switch } from "@/shared/ui/switch"
import { useI18n } from "@/shared/i18n"
import { cn } from "@/lib/utils"

import { resolveFileIcon } from "../utils/fileIcons"

export type FileDeleteItem = {
  id?: string
  name?: string
  path?: string
  fileType?: string
  format?: string
}

type LibraryFileContextMenuProps = {
  anchor: { x: number; y: number } | null
  onClose: () => void
  onOpenFolder?: () => void
  onDelete: () => void
  openFolderLabel: string
  deleteLabel: string
}

type LibraryFileDeleteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  itemName: string
  deleteFileItems?: FileDeleteItem[]
  defaultDeleteFiles?: boolean
  deleteTitle: string
  deleteDescription: string
  deleteImpactDescription?: string
  deleteFilesLabel: string
  deleteFilesTitle: string
  deleteFilesEmpty?: string
  deleteFilesEnabledText?: string
  deleteFilesDisabledText?: string
  deleteCancelLabel?: string
  deleteConfirmLabel?: string
  onDelete: (options: { deleteFiles: boolean }) => void | Promise<void>
}

export function LibraryFileContextMenu({
  anchor,
  onClose,
  onOpenFolder,
  onDelete,
  openFolderLabel,
  deleteLabel,
}: LibraryFileContextMenuProps) {
  const menuRef = React.useRef<HTMLDivElement | null>(null)
  const [position, setPosition] = React.useState<{ x: number; y: number } | null>(anchor)

  React.useEffect(() => {
    setPosition(anchor)
  }, [anchor])

  React.useLayoutEffect(() => {
    if (!anchor || !menuRef.current) {
      return
    }
    const rect = menuRef.current.getBoundingClientRect()
    const nextX = Math.max(8, Math.min(anchor.x, window.innerWidth - rect.width - 8))
    const nextY = Math.max(8, Math.min(anchor.y, window.innerHeight - rect.height - 8))
    if (nextX !== position?.x || nextY !== position?.y) {
      setPosition({ x: nextX, y: nextY })
    }
  }, [anchor, position?.x, position?.y])

  React.useEffect(() => {
    if (!anchor) {
      return
    }
    const handlePointerDown = (event: PointerEvent) => {
      if (menuRef.current?.contains(event.target as Node)) {
        return
      }
      onClose()
    }
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        onClose()
      }
    }
    const handleViewportChange = () => onClose()
    document.addEventListener("pointerdown", handlePointerDown)
    window.addEventListener("keydown", handleKeyDown)
    window.addEventListener("resize", handleViewportChange)
    window.addEventListener("blur", handleViewportChange)
    return () => {
      document.removeEventListener("pointerdown", handlePointerDown)
      window.removeEventListener("keydown", handleKeyDown)
      window.removeEventListener("resize", handleViewportChange)
      window.removeEventListener("blur", handleViewportChange)
    }
  }, [anchor, onClose])

  if (!anchor || !position) {
    return null
  }

  const itemClassName =
    "app-menu-item app-motion-color flex w-full items-center text-left text-sm outline-none hover:bg-accent hover:text-accent-foreground"

  return createPortal(
    <div
      ref={menuRef}
      role="menu"
      className={cn(
        "app-menu-content app-motion-surface fixed z-[120] w-max min-w-fit text-sm",
        "animate-in fade-in-0 zoom-in-95"
      )}
      style={{ left: position.x, top: position.y }}
      onContextMenu={(event) => event.preventDefault()}
    >
      <button
        type="button"
        role="menuitem"
        className={cn(itemClassName, !onOpenFolder && "pointer-events-none opacity-50")}
        onClick={onOpenFolder}
        disabled={!onOpenFolder}
      >
        <FolderOpen className="h-4 w-4" />
        <span>{openFolderLabel}</span>
      </button>
      <div className="app-menu-separator" />
      <button
        type="button"
        role="menuitem"
        className={cn(itemClassName, "text-destructive hover:text-destructive")}
        onClick={onDelete}
      >
        <Trash2 className="h-4 w-4" />
        <span>{deleteLabel}</span>
      </button>
    </div>,
    document.body
  )
}

export function LibraryFileDeleteDialog({
  open,
  onOpenChange,
  itemName,
  deleteFileItems,
  defaultDeleteFiles = false,
  deleteTitle,
  deleteDescription,
  deleteImpactDescription,
  deleteFilesLabel,
  deleteFilesTitle,
  deleteFilesEmpty,
  deleteFilesEnabledText,
  deleteFilesDisabledText,
  deleteCancelLabel,
  deleteConfirmLabel,
  onDelete,
}: LibraryFileDeleteDialogProps) {
  const { t } = useI18n()
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [deleteFiles, setDeleteFiles] = React.useState(defaultDeleteFiles)

  React.useEffect(() => {
    if (open) {
      setDeleteFiles(defaultDeleteFiles)
    }
  }, [defaultDeleteFiles, open])

  const resolvedDeleteFileItems = React.useMemo(
    () =>
      (deleteFileItems ?? []).filter(
        (item): item is FileDeleteItem => Boolean(item && (item.name || item.path))
      ),
    [deleteFileItems]
  )
  const showDeleteFilesToggle = resolvedDeleteFileItems.length > 0

  const handleDeleteConfirm = async () => {
    setIsDeleting(true)
    try {
      await onDelete({ deleteFiles })
      onOpenChange(false)
    } catch {
      // Keep dialog open for retry.
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DashboardDialogContent size="standard" className="flex max-h-[80vh] w-full flex-col gap-4 overflow-hidden">
        <DashboardDialogHeader>
          <DialogTitle>{deleteTitle}</DialogTitle>
          <DialogDescription className="text-xs">
            {deleteDescription || itemName || t("library.rowMenu.renameFallback")}
          </DialogDescription>
          {deleteImpactDescription ? <div className="text-xs text-muted-foreground">{deleteImpactDescription}</div> : null}
        </DashboardDialogHeader>
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          {showDeleteFilesToggle ? (
            <div className="space-y-4">
              <DashboardDialogSection tone="field" className="space-y-3">
                <div className="flex items-start justify-between gap-3">
                  <div className="space-y-1">
                    <div className="text-sm font-medium">{deleteFilesLabel}</div>
                    <div className="text-xs text-muted-foreground">
                      {deleteFiles
                        ? deleteFilesEnabledText ?? t("library.file.deleteFilesEnabled")
                        : deleteFilesDisabledText ?? t("library.file.deleteFilesDisabled")}
                    </div>
                  </div>
                  <Switch checked={deleteFiles} onCheckedChange={setDeleteFiles} />
                </div>
              </DashboardDialogSection>
              {deleteFiles ? (
                <DashboardDialogSection tone="field" className="space-y-3">
                  <div className="text-xs font-medium text-muted-foreground">{deleteFilesTitle}</div>
                  {resolvedDeleteFileItems.length > 0 ? (
                    <div className="overflow-hidden rounded-lg border border-border/70">
                      <div className="divide-y divide-border/60">
                        {resolvedDeleteFileItems.map((item, index) => {
                          const FileIcon = resolveFileIcon({
                            fileType: item.fileType,
                            path: item.path,
                            name: item.name ?? "",
                          })
                          const label = item.name?.trim() || item.path?.trim() || "-"
                          const formatLabel = resolveDeleteItemFormat(item)
                          return (
                            <React.Fragment key={`${label}-${index}`}>
                              {index > 0 ? <Separator /> : null}
                              <div
                                className={cn(
                                  "flex w-full min-w-0 items-center gap-2 px-3 py-2 text-left text-xs",
                                  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                                  index > 0 && "rounded-none border-0 border-t border-border/60 bg-transparent"
                                )}
                              >
                                <FileIcon className="h-4 w-4 shrink-0" />
                                <span className="min-w-0 flex-1 truncate">{label}</span>
                                {formatLabel ? (
                                  <Badge variant="outline" className="ml-auto shrink-0 text-xs font-medium">
                                    {formatLabel}
                                  </Badge>
                                ) : null}
                              </div>
                            </React.Fragment>
                          )
                        })}
                      </div>
                    </div>
                  ) : (
                    <div className="px-3 py-3 text-xs text-muted-foreground">
                      {deleteFilesEmpty ?? t("library.rowMenu.deleteFilesEmpty")}
                    </div>
                  )}
                </DashboardDialogSection>
              ) : null}
            </div>
          ) : null}
        </div>
        <DashboardDialogFooter>
          <Button variant="ghost" size="compact" className="text-xs" onClick={() => onOpenChange(false)}>
            {deleteCancelLabel ?? t("library.rowMenu.deleteCancel")}
          </Button>
          <Button
            variant="destructive"
            size="compact"
            className="text-xs"
            onClick={handleDeleteConfirm}
            disabled={isDeleting}
          >
            {deleteConfirmLabel ?? t("library.rowMenu.deleteConfirm")}
          </Button>
        </DashboardDialogFooter>
      </DashboardDialogContent>
    </Dialog>
  )
}

function resolveDeleteItemFormat(item: FileDeleteItem) {
  const explicit = item.format?.trim()
  if (explicit) {
    return explicit.toUpperCase()
  }
  const byPath = extractExtension(item.path)
  if (byPath) {
    return byPath.toUpperCase()
  }
  const byName = extractExtension(item.name)
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
