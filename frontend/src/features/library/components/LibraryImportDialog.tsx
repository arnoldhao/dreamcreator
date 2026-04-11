import { Captions, FilePlus2, Loader2, Video } from "lucide-react"

import { Button } from "@/shared/ui/button"
import {
  DashboardDialogBody,
  DashboardDialogFooter,
  DashboardDialogHeader,
} from "@/shared/ui/dashboard-dialog"
import { DialogDescription, DialogTitle } from "@/shared/ui/dialog"
import { Input } from "@/shared/ui/input"
import { Select } from "@/shared/ui/select"

import {
  WorkspaceDialogFormRow,
  WorkspaceDialogHeaderCard,
  WorkspaceDialogItemsCard,
  WorkspaceDialogMetricsCard,
  WorkspaceDialogSectionBadge,
  WorkspaceDialogSectionCard,
  WorkspaceDialogSummaryRow,
} from "./workspace/WorkspaceDashboardDialog"
import { extractExtensionFromPath, getPathBaseName, stripPathExtension } from "../utils/resourceHelpers"

type Translator = (key: string) => string

export type LibraryImportDialogProps = {
  kind: "video" | "subtitle"
  filePath: string
  importTargetMode: "new" | "existing"
  currentLibraryLabel?: string
  titleValue: string
  onTitleChange: (value: string) => void
  onModeChange: (value: "new" | "existing") => void
  onClose: () => void
  onSelectFile: () => void
  onSubmit: () => void
  submitting: boolean
  canSubmit: boolean
  t: Translator
}

export function LibraryImportDialog(props: LibraryImportDialogProps) {
  const Icon = props.kind === "video" ? Video : Captions
  const hasFile = Boolean(props.filePath.trim())
  const fileName = getPathBaseName(props.filePath) || "-"
  const fileFormat = extractExtensionFromPath(props.filePath).toUpperCase() || "-"
  const currentLibraryLabel = props.currentLibraryLabel?.trim() ?? ""
  const titlePreview = props.titleValue.trim() || stripPathExtension(getPathBaseName(props.filePath))
  const targetPreview =
    props.importTargetMode === "existing"
      ? currentLibraryLabel || props.t("library.import.targetCurrentEmpty")
      : props.t("library.import.targetNew")

  return (
    <>
      <DashboardDialogHeader>
        <DialogTitle>
          {props.kind === "video"
            ? props.t("library.actions.importVideo")
            : props.t("library.actions.importSubtitle")}
        </DialogTitle>
        <DialogDescription>
          {props.kind === "video"
            ? props.t("library.import.videoDialogDescription")
            : props.t("library.import.subtitleDialogDescription")}
        </DialogDescription>
      </DashboardDialogHeader>

      <DashboardDialogBody className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {!hasFile ? (
          <div className="flex min-h-[280px] items-center justify-center">
            <Button size="compact" onClick={props.onSelectFile}>
              <FilePlus2 className="h-4 w-4" />
              {props.t("library.import.selectFile")}
            </Button>
          </div>
        ) : (
          <>
            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(220px,248px)]">
              <WorkspaceDialogHeaderCard
                title={props.t("library.task.summary")}
                badge={
                  <WorkspaceDialogSectionBadge>
                    <Icon className="h-3.5 w-3.5" />
                    {props.kind === "video"
                      ? props.t("library.actions.importVideo")
                      : props.t("library.actions.importSubtitle")}
                  </WorkspaceDialogSectionBadge>
                }
              >
                <div className="space-y-2">
                  <WorkspaceDialogSummaryRow label={props.t("library.import.fileName")} value={fileName} />
                  <WorkspaceDialogSummaryRow label={props.t("library.import.fileFormat")} value={fileFormat} />
                  <WorkspaceDialogSummaryRow label={props.t("library.import.targetMode")} value={targetPreview} />
                </div>
              </WorkspaceDialogHeaderCard>

              <WorkspaceDialogHeaderCard
                title={props.t("library.task.overview")}
                badge={<WorkspaceDialogSectionBadge>{props.t("library.import.dialog.ready")}</WorkspaceDialogSectionBadge>}
              >
                <WorkspaceDialogMetricsCard
                  columns={2}
                  items={[
                    {
                      label: props.t("library.import.dialog.metrics.file"),
                      value: props.t("library.import.dialog.selected"),
                    },
                    {
                      label: props.t("library.import.dialog.metrics.target"),
                      value:
                        props.importTargetMode === "existing"
                          ? props.t("library.import.targetCurrentLabel")
                          : props.t("library.import.targetNew"),
                    },
                    {
                      label: props.t("library.import.dialog.metrics.title"),
                      value: titlePreview || "-",
                    },
                    {
                      label: props.t("library.import.dialog.metrics.naming"),
                      value: props.titleValue.trim()
                        ? props.t("library.import.dialog.customTitle")
                        : props.t("library.import.dialog.autoTitle"),
                    },
                  ]}
                />
              </WorkspaceDialogHeaderCard>
            </div>

            <ImportTargetCard
              importTargetMode={props.importTargetMode}
              currentLibraryLabel={currentLibraryLabel}
              onModeChange={props.onModeChange}
              t={props.t}
            />

            <ImportFileCard kind={props.kind} path={props.filePath} t={props.t} />

            <WorkspaceDialogSectionCard
              title={props.t("library.import.dialog.namingTitle")}
              description={props.t("library.import.dialog.namingDescription")}
            >
              <WorkspaceDialogFormRow
                label={props.t("library.tools.optionalTitle")}
                description={props.t("library.import.dialog.namingHint")}
                control={
                  <Input
                    value={props.titleValue}
                    onChange={(event) => props.onTitleChange(event.target.value)}
                    placeholder={props.t("library.tools.optionalTitle")}
                    className="h-8 text-xs"
                  />
                }
              />
            </WorkspaceDialogSectionCard>
          </>
        )}
      </DashboardDialogBody>

      <DashboardDialogFooter className="sm:justify-end">
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center">
          <Button variant="outline" size="compact" onClick={props.onClose}>
            {props.t("common.close")}
          </Button>
          {hasFile ? (
            <Button variant="outline" size="compact" onClick={props.onSelectFile}>
              <FilePlus2 className="h-4 w-4" />
              {props.t("library.import.reselect")}
            </Button>
          ) : null}
          <Button size="compact" onClick={props.onSubmit} disabled={props.submitting || !props.canSubmit}>
            {props.submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Icon className="h-4 w-4" />}
            {props.t("library.tools.import")}
          </Button>
        </div>
      </DashboardDialogFooter>
    </>
  )
}

function ImportTargetCard(props: {
  importTargetMode: "new" | "existing"
  currentLibraryLabel?: string
  onModeChange: (value: "new" | "existing") => void
  t: Translator
}) {
  const currentLibraryLabel = props.currentLibraryLabel?.trim() ?? ""

  return (
    <WorkspaceDialogSectionCard
      title={props.t("library.import.dialog.targetTitle")}
      description={props.t("library.import.dialog.targetDescription")}
      badge={<WorkspaceDialogSectionBadge>{props.t("library.workspace.dialogs.required")}</WorkspaceDialogSectionBadge>}
    >
      <div className="space-y-3">
        <WorkspaceDialogFormRow
          label={props.t("library.import.targetMode")}
          description={props.t("library.import.dialog.targetHint")}
          control={
            <Select
              value={props.importTargetMode}
              onChange={(event) => props.onModeChange(event.target.value === "existing" ? "existing" : "new")}
              className="h-8 w-full border-border/70 bg-background/80"
            >
              <option value="new">{props.t("library.import.targetNew")}</option>
              <option value="existing" disabled={!currentLibraryLabel}>
                {props.t("library.import.targetCurrent")}
              </option>
            </Select>
          }
        />
        {props.importTargetMode === "existing" ? (
          <WorkspaceDialogItemsCard
            items={[
              {
                key: "current-library",
                label: props.t("library.import.targetCurrentLabel"),
                value: (
                  <span
                    className="block truncate text-xs text-muted-foreground"
                    title={currentLibraryLabel || props.t("library.import.targetCurrentEmpty")}
                  >
                    {currentLibraryLabel || props.t("library.import.targetCurrentEmpty")}
                  </span>
                ),
              },
            ]}
          />
        ) : null}
      </div>
    </WorkspaceDialogSectionCard>
  )
}

function ImportFileCard(props: { kind: "video" | "subtitle"; path: string; t: Translator }) {
  const formatLabel = extractExtensionFromPath(props.path).toUpperCase() || "-"
  const fileName = getPathBaseName(props.path) || "-"

  return (
    <WorkspaceDialogSectionCard
      title={
        props.kind === "video"
          ? props.t("library.actions.importVideo")
          : props.t("library.actions.importSubtitle")
      }
      description={props.t("library.import.dialog.fileDescription")}
    >
      <WorkspaceDialogItemsCard
        items={[
          {
            key: "file-name",
            label: props.t("library.import.fileName"),
            value: (
              <span className="block truncate text-xs text-muted-foreground" title={fileName}>
                {fileName}
              </span>
            ),
          },
          {
            key: "file-format",
            label: props.t("library.import.fileFormat"),
            value: <span className="text-xs text-muted-foreground">{formatLabel}</span>,
          },
          {
            key: "file-path",
            label: props.t("library.import.filePath"),
            value: (
              <span className="block truncate text-xs text-muted-foreground" title={props.path || "-"}>
                {props.path || "-"}
              </span>
            ),
          },
        ]}
      />
    </WorkspaceDialogSectionCard>
  )
}
