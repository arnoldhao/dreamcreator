import * as React from "react"
import { FolderOpen, Languages, MoreHorizontal, PencilLine, RefreshCw, Trash2 } from "lucide-react"

import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
  DashboardDialogSection,
} from "@/shared/ui/dashboard-dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { Dialog, DialogDescription, DialogTitle } from "@/shared/ui/dialog"
import { Input } from "@/shared/ui/input"
import { Separator } from "@/shared/ui/separator"
import { Switch } from "@/shared/ui/switch"
import { useI18n } from "@/shared/i18n"
import { cn } from "@/lib/utils"

import { resolveFileIcon } from "../utils/fileIcons"

type DeleteFileItem = {
  id?: string
  name?: string
  path?: string
  fileType?: string
  format?: string
}

type LibraryRowMenuProps = {
  itemName: string
  onCreateTranscode?: () => void | Promise<void>
  onCreateSubtitleTranslate?: () => void | Promise<void>
  onOpenFolder?: () => void
  onRename?: (name: string) => void | Promise<void>
  onDelete?: (options: { deleteFiles: boolean }) => void | Promise<void>
  showDeleteFilesToggle?: boolean
  deleteFileItems?: DeleteFileItem[]
  defaultDeleteFiles?: boolean
  deleteTitle?: string
  deleteDescription?: string
  deleteImpactDescription?: string
  deleteFilesLabel?: string
  deleteFilesTitle?: string
  deleteFilesEmpty?: string
  deleteCancelLabel?: string
  deleteConfirmLabel?: string
}

export function LibraryRowMenu({
  itemName,
  onCreateTranscode,
  onCreateSubtitleTranslate,
  onOpenFolder,
  onRename,
  onDelete,
  showDeleteFilesToggle,
  deleteFileItems,
  defaultDeleteFiles = false,
  deleteTitle,
  deleteDescription,
  deleteImpactDescription,
  deleteFilesLabel,
  deleteFilesTitle,
  deleteFilesEmpty,
  deleteCancelLabel,
  deleteConfirmLabel,
}: LibraryRowMenuProps) {
  const { t } = useI18n()
  const [renameOpen, setRenameOpen] = React.useState(false)
  const [deleteOpen, setDeleteOpen] = React.useState(false)
  const [draftName, setDraftName] = React.useState(itemName)
  const [isRenaming, setIsRenaming] = React.useState(false)
  const [isDeleting, setIsDeleting] = React.useState(false)
  const [deleteFiles, setDeleteFiles] = React.useState(defaultDeleteFiles)

  React.useEffect(() => {
    if (renameOpen) {
      setDraftName(itemName)
    }
  }, [itemName, renameOpen])

  React.useEffect(() => {
    if (deleteOpen) {
      setDeleteFiles(defaultDeleteFiles)
    }
    return
  }, [defaultDeleteFiles, deleteOpen])

  const resolvedDeleteFileItems = React.useMemo(
    () =>
      (deleteFileItems ?? []).filter(
        (item): item is DeleteFileItem => Boolean(item && (item.name || item.path))
      ),
    [deleteFileItems]
  )

  const hasActions = Boolean(onRename || onDelete || onOpenFolder)
  const canCreateTranscode = Boolean(onCreateTranscode)
  const canCreateSubtitleTranslate = Boolean(onCreateSubtitleTranslate)
  const canRename = Boolean(onRename)
  const canDelete = Boolean(onDelete)
  const canOpenFolder = Boolean(onOpenFolder)
  const hasPrimaryActions = canOpenFolder || canCreateTranscode || canCreateSubtitleTranslate

  const handleRenameConfirm = async () => {
    if (!onRename) {
      return
    }
    const trimmed = draftName.trim()
    if (!trimmed) {
      return
    }
    setIsRenaming(true)
    try {
      await onRename(trimmed)
      setRenameOpen(false)
    } catch {
      // Keep dialog open for retry.
    } finally {
      setIsRenaming(false)
    }
  }

  const handleDeleteConfirm = async () => {
    if (!onDelete) {
      return
    }
    setIsDeleting(true)
    try {
      await onDelete({ deleteFiles })
      setDeleteOpen(false)
    } catch {
      // Keep dialog open for retry.
    } finally {
      setIsDeleting(false)
    }
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            aria-label={t("library.rowMenu.actions")}
          >
            <MoreHorizontal className="h-4 w-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {canOpenFolder ? (
            <DropdownMenuItem disabled={!canOpenFolder} onClick={onOpenFolder}>
              <FolderOpen className="h-4 w-4" />
              <span>{t("library.rowMenu.openFolder")}</span>
            </DropdownMenuItem>
          ) : null}
          {canCreateTranscode ? (
            <DropdownMenuItem onClick={() => void onCreateTranscode?.()}>
              <RefreshCw className="h-4 w-4" />
              <span>{t("library.actions.newTranscode")}</span>
            </DropdownMenuItem>
          ) : null}
          {canCreateSubtitleTranslate ? (
            <DropdownMenuItem onClick={() => void onCreateSubtitleTranslate?.()}>
              <Languages className="h-4 w-4" />
              <span>{t("library.actions.newSubtitleTranslate")}</span>
            </DropdownMenuItem>
          ) : null}
          {hasPrimaryActions && (canRename || canDelete) ? <DropdownMenuSeparator /> : null}
          <DropdownMenuItem disabled={!canRename} onClick={() => setRenameOpen(true)}>
            <PencilLine className="h-4 w-4" />
            <span>{t("library.rowMenu.editName")}</span>
          </DropdownMenuItem>
          <DropdownMenuItem
            disabled={!canDelete || !hasActions}
            onClick={() => setDeleteOpen(true)}
            className="text-xs text-destructive focus:text-destructive"
          >
            <Trash2 className="h-4 w-4 text-destructive" />
            <span>{t("library.rowMenu.delete")}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>

      <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
        <DashboardDialogContent size="compact">
          <DashboardDialogHeader>
            <DialogTitle>{t("library.rowMenu.renameTitle")}</DialogTitle>
            <DialogDescription>
              {t("library.rowMenu.renameDescription").replace(
                "{name}",
                itemName || t("library.rowMenu.renameFallback")
              )}
            </DialogDescription>
          </DashboardDialogHeader>
          <DashboardDialogSection tone="field" className="space-y-2">
            <div className="text-xs text-muted-foreground">
              {t("library.rowMenu.renamePlaceholder")}
            </div>
            <Input
              value={draftName}
              onChange={(event) => setDraftName(event.target.value)}
              placeholder={t("library.rowMenu.renamePlaceholder")}
            />
          </DashboardDialogSection>
          <DashboardDialogFooter>
            <Button variant="ghost" size="compact" onClick={() => setRenameOpen(false)}>
              {t("library.rowMenu.renameCancel")}
            </Button>
            <Button
              size="compact"
              onClick={handleRenameConfirm}
              disabled={!canRename || !draftName.trim() || isRenaming}
            >
              {t("library.rowMenu.renameConfirm")}
            </Button>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DashboardDialogContent size="standard" className="flex max-h-[80vh] w-full flex-col gap-4 overflow-hidden">
          <DashboardDialogHeader>
            <DialogTitle>{deleteTitle ?? t("library.rowMenu.deleteTitle")}</DialogTitle>
            <DialogDescription className="text-xs">
              {(deleteDescription ??
                t("library.rowMenu.deleteDescription"))
                .replace("{name}", itemName || t("library.rowMenu.renameFallback"))}
            </DialogDescription>
            {deleteImpactDescription ? <div className="text-xs text-muted-foreground">{deleteImpactDescription}</div> : null}
          </DashboardDialogHeader>
          <div className="min-h-0 flex-1 overflow-y-auto pr-1">
            {showDeleteFilesToggle ? (
              <div className="space-y-2">
                <DashboardDialogSection tone="field" className="flex items-center justify-between px-3 py-2">
                  <span className="text-xs text-foreground">
                    {deleteFilesLabel ?? t("library.rowMenu.deleteFilesLabel")}
                  </span>
                  <Switch checked={deleteFiles} onCheckedChange={setDeleteFiles} />
                </DashboardDialogSection>
                {deleteFiles ? (
                  <>
                    <div className="text-xs font-medium text-muted-foreground">
                      {deleteFilesTitle ?? t("library.rowMenu.deleteFilesTitle")}
                    </div>
                    <DashboardDialogSection tone="inset" className="overflow-hidden p-0">
                      {resolvedDeleteFileItems.length > 0 ? (
                        <div className="max-h-36 overflow-x-hidden overflow-y-auto">
                          <div className="flex flex-col">
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
                                      index > 0 && "rounded-none border-0 border-t border-border/60 bg-transparent",
                                    )}
                                  >
                                    <FileIcon className="h-4 w-4 shrink-0" />
                                    <span className="min-w-0 flex-1 truncate">{label}</span>
                                    {formatLabel ? (
                                      <Badge
                                        variant="outline"
                                        className="ml-auto shrink-0 text-xs font-medium"
                                      >
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
                  </>
                ) : null}
              </div>
            ) : null}
          </div>
          <DashboardDialogFooter>
            <Button
              variant="ghost"
              size="compact"
              className="text-xs"
              onClick={() => setDeleteOpen(false)}
            >
              {deleteCancelLabel ?? t("library.rowMenu.deleteCancel")}
            </Button>
            <Button
              variant="destructive"
              size="compact"
              className="text-xs"
              onClick={handleDeleteConfirm}
              disabled={!canDelete || isDeleting}
            >
              {deleteConfirmLabel ?? t("library.rowMenu.deleteConfirm")}
            </Button>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>
    </>
  )
}

function resolveDeleteItemFormat(item: DeleteFileItem) {
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
