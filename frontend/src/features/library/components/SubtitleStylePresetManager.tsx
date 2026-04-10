import * as React from "react"
import { Call } from "@wailsio/runtime"
import {
  Captions,
  Check,
  ChevronDown,
  Download,
  FileUp,
  Plus,
  RotateCcw,
  Save,
} from "lucide-react"

import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"
import { messageBus } from "@/shared/message"
import { useSelectDirectory } from "@/shared/query/settings"
import type {
  ExportSubtitleStylePresetRequest,
  ExportSubtitleStylePresetResult,
  LibraryBilingualStyleDTO,
  LibraryModuleConfigDTO,
  LibraryMonoStyleDTO,
  ParseSubtitleStyleImportRequest,
  ParseSubtitleStyleImportResult,
} from "@/shared/contracts/library"
import { Button } from "@/shared/ui/button"
import {
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { Input } from "@/shared/ui/input"

import {
  applyAspectRatioBaseResolution,
  cloneBilingualStyle,
  cloneMonoStyle,
  createEmptyBilingualStyle,
  createEmptyMonoStyle,
  createMonoSnapshotFromStyle,
  createSubtitleStylePresetID,
  resolveBilingualStyles,
  resolveMonoStyles,
  type SubtitleStylePresetKind,
  type SubtitleStylePresetSelection,
} from "../utils/subtitleStylePresets"
import { resolveSubtitleStyleDefaults } from "../utils/subtitleStyles"
import {
  EditorGroupCard,
  EditorRow,
  SubtitleStyleEmptyState,
} from "./SubtitleStylePresetManagerControls"
import {
  BilingualStyleEditor,
  MonoStyleEditor,
} from "./SubtitleStylePresetManagerEditors"
import {
  AllStyleCardsPane,
  CreateStylePane,
  PresetCompositeHeader,
  PreviewPane,
} from "./SubtitleStylePresetManagerPanes"
import type {
  LeftPaneView,
} from "./SubtitleStylePresetManagerShared"
import {
  createInitialCreateStyleDraft,
  normalizeBilingualStyleForEditor,
  normalizeCreateStyleDraft,
  normalizeImportResult,
  normalizeMonoStyleForEditor,
  normalizeSnapshotBase,
  resolveSelectionItem,
  signatureOf,
} from "./SubtitleStylePresetManagerShared"

type SubtitleStylePresetManagerProps = {
  value: LibraryModuleConfigDTO
  onChange: (next: LibraryModuleConfigDTO) => void
  onRequestPersist?: () => void
  onToolbarActionsChange?: (actions: React.ReactNode | null) => void
}

export function SubtitleStylePresetManager({
  value,
  onChange,
  onRequestPersist,
  onToolbarActionsChange,
}: SubtitleStylePresetManagerProps) {
  const { t } = useI18n()
  const selectDirectory = useSelectDirectory()
  const fileInputRef = React.useRef<HTMLInputElement | null>(null)

  const monoStyles = React.useMemo(() => resolveMonoStyles(value), [value])
  const bilingualStyles = React.useMemo(() => resolveBilingualStyles(value), [value])
  const fontMappings = React.useMemo(
    () => value.subtitleStyles.fonts ?? [],
    [value.subtitleStyles.fonts],
  )
  const subtitleStyleDefaults = React.useMemo(
    () => resolveSubtitleStyleDefaults(value),
    [value],
  )
  const defaultMonoStyle = React.useMemo(
    () =>
      monoStyles.find((item) => item.id === subtitleStyleDefaults.monoStyleId.trim()) ??
      monoStyles[0] ??
      null,
    [monoStyles, subtitleStyleDefaults.monoStyleId],
  )
  const defaultBilingualStyle = React.useMemo(
    () =>
      bilingualStyles.find((item) => item.id === subtitleStyleDefaults.bilingualStyleId.trim()) ??
      bilingualStyles[0] ??
      null,
    [bilingualStyles, subtitleStyleDefaults.bilingualStyleId],
  )

  const [selection, setSelection] = React.useState<SubtitleStylePresetSelection>(null)
  const [draftKind, setDraftKind] = React.useState<SubtitleStylePresetKind | null>(null)
  const [monoDraft, setMonoDraft] = React.useState<LibraryMonoStyleDTO | null>(null)
  const [bilingualDraft, setBilingualDraft] = React.useState<LibraryBilingualStyleDTO | null>(null)
  const [draftBaseSignature, setDraftBaseSignature] = React.useState("")
  const [leftPaneView, setLeftPaneView] = React.useState<LeftPaneView>("preview")
  const [createPaneMode, setCreatePaneMode] = React.useState<"form" | "import">("form")
  const [createDraft, setCreateDraft] = React.useState(() => createInitialCreateStyleDraft(monoStyles))
  const [importDraft, setImportDraft] = React.useState<{
    kind: "ass"
    fileName: string
    detectedRatio?: string
    normalizedPlayResX?: number
    normalizedPlayResY?: number
    styles: LibraryMonoStyleDTO[]
  } | {
    kind: "dcssp"
    fileName: string
    result: ParseSubtitleStyleImportResult
  } | null>(null)
  const [importingFile, setImportingFile] = React.useState(false)
  const [exporting, setExporting] = React.useState(false)

  const currentDraft = draftKind === "mono" ? monoDraft : draftKind === "bilingual" ? bilingualDraft : null
  const currentDraftSignature = React.useMemo(() => signatureOf(currentDraft), [currentDraft])
  const hasDraft = Boolean(currentDraft && draftKind)
  const isDirty = Boolean(hasDraft && currentDraftSignature !== draftBaseSignature)
  const selectionItem = React.useMemo(
    () => resolveSelectionItem(selection, monoStyles, bilingualStyles),
    [bilingualStyles, monoStyles, selection],
  )
  const canExportSelected = Boolean(selection && selectionItem && !isDirty)
  const isSelectedDefault = React.useMemo(() => {
    if (!selection || !selectionItem) {
      return false
    }
    if (selection.kind === "mono") {
      return subtitleStyleDefaults.monoStyleId.trim() === selectionItem.id
    }
    return subtitleStyleDefaults.bilingualStyleId.trim() === selectionItem.id
  }, [
    selection,
    selectionItem,
    subtitleStyleDefaults.bilingualStyleId,
    subtitleStyleDefaults.monoStyleId,
  ])

  React.useEffect(() => {
    setCreateDraft((current) => normalizeCreateStyleDraft(current, monoStyles))
  }, [monoStyles])

  const updateValue = React.useCallback(
    (patch: Partial<LibraryModuleConfigDTO["subtitleStyles"]>) => {
      onChange({
        ...value,
        subtitleStyles: {
          ...value.subtitleStyles,
          ...patch,
        },
      })
    },
    [onChange, value],
  )

  const beginMonoEdit = React.useCallback((style: LibraryMonoStyleDTO) => {
    const nextDraft = normalizeMonoStyleForEditor(cloneMonoStyle(style))
    setSelection({ kind: "mono", id: style.id })
    setDraftKind("mono")
    setMonoDraft(nextDraft)
    setBilingualDraft(null)
    setDraftBaseSignature(signatureOf(nextDraft))
  }, [])

  const beginBilingualEdit = React.useCallback((style: LibraryBilingualStyleDTO) => {
    const nextDraft = normalizeBilingualStyleForEditor(cloneBilingualStyle(style))
    setSelection({ kind: "bilingual", id: style.id })
    setDraftKind("bilingual")
    setMonoDraft(null)
    setBilingualDraft(nextDraft)
    setDraftBaseSignature(signatureOf(nextDraft))
  }, [])

  const clearDraft = React.useCallback(() => {
    setSelection(null)
    setDraftKind(null)
    setMonoDraft(null)
    setBilingualDraft(null)
    setDraftBaseSignature("")
  }, [])

  React.useEffect(() => {
    if (!selection && !hasDraft) {
      if (defaultMonoStyle) {
        beginMonoEdit(defaultMonoStyle)
      } else if (defaultBilingualStyle) {
        beginBilingualEdit(defaultBilingualStyle)
      }
      return
    }

    if (selection && !selectionItem) {
      if (defaultMonoStyle) {
        beginMonoEdit(defaultMonoStyle)
      } else if (defaultBilingualStyle) {
        beginBilingualEdit(defaultBilingualStyle)
      } else {
        clearDraft()
      }
    }
  }, [
    beginBilingualEdit,
    beginMonoEdit,
    clearDraft,
    defaultBilingualStyle,
    defaultMonoStyle,
    hasDraft,
    selection,
    selectionItem,
  ])

  const guardDirtyBeforeSwitch = React.useCallback(
    (nextSelection: SubtitleStylePresetSelection) => {
      if (!isDirty) {
        return true
      }
      const isSame = nextSelection?.kind === selection?.kind && nextSelection?.id === selection?.id
      if (isSame) {
        return true
      }
      messageBus.publishToast({
        intent: "warning",
        title: t("library.config.subtitleStyles.unsavedTitle"),
        description: t("library.config.subtitleStyles.unsavedDescription"),
      })
      return false
    },
    [isDirty, selection?.id, selection?.kind, t],
  )

  const selectStyle = React.useCallback(
    (nextSelection: SubtitleStylePresetSelection) => {
      if (!nextSelection) {
        return
      }
      if (!guardDirtyBeforeSwitch(nextSelection)) {
        return
      }
      if (nextSelection.kind === "mono") {
        const next = monoStyles.find((item) => item.id === nextSelection.id)
        if (next) {
          beginMonoEdit(next)
          setLeftPaneView("preview")
        }
        return
      }
      const next = bilingualStyles.find((item) => item.id === nextSelection.id)
      if (next) {
        beginBilingualEdit(next)
        setLeftPaneView("preview")
      }
    },
    [beginBilingualEdit, beginMonoEdit, bilingualStyles, guardDirtyBeforeSwitch, monoStyles],
  )

  const resetDraft = React.useCallback(() => {
    if (!selection) {
      clearDraft()
      return
    }
    const latest = resolveSelectionItem(selection, monoStyles, bilingualStyles)
    if (!latest) {
      clearDraft()
      return
    }
    if (selection.kind === "mono") {
      beginMonoEdit(latest as LibraryMonoStyleDTO)
      return
    }
    beginBilingualEdit(latest as LibraryBilingualStyleDTO)
  }, [beginBilingualEdit, beginMonoEdit, bilingualStyles, clearDraft, monoStyles, selection])

  const saveDraft = React.useCallback(() => {
    if (!currentDraft || !draftKind || !selection) {
      return
    }

    if (draftKind === "mono") {
      const nextDraft = normalizeMonoStyleForEditor(cloneMonoStyle(currentDraft as LibraryMonoStyleDTO))
      updateValue({
        monoStyles: monoStyles.map((item) => (item.id === nextDraft.id ? nextDraft : item)),
      })
      setMonoDraft(nextDraft)
      setDraftBaseSignature(signatureOf(nextDraft))
      onRequestPersist?.()
      return
    }

    const nextDraft = normalizeBilingualStyleForEditor(cloneBilingualStyle(currentDraft as LibraryBilingualStyleDTO))
    updateValue({
      bilingualStyles: bilingualStyles.map((item) => (item.id === nextDraft.id ? nextDraft : item)),
    })
    setBilingualDraft(nextDraft)
    setDraftBaseSignature(signatureOf(nextDraft))
    onRequestPersist?.()
  }, [
    bilingualStyles,
    currentDraft,
    draftKind,
    monoStyles,
    onRequestPersist,
    selection,
    updateValue,
  ])

  const selectedExportInfo = React.useMemo(() => {
    if (!selection || !selectionItem) {
      return null
    }

    if (selection.kind === "mono") {
      const style = selectionItem as LibraryMonoStyleDTO
      return {
        type: "mono" as const,
        id: style.id,
        name: style.name,
        baseAspectRatio: style.baseAspectRatio,
        basePlayResX: style.basePlayResX,
        basePlayResY: style.basePlayResY,
        mono: style,
        bilingual: null,
      }
    }

    const style = selectionItem as LibraryBilingualStyleDTO
    return {
      type: "bilingual" as const,
      id: style.id,
      name: style.name,
      baseAspectRatio: style.baseAspectRatio,
      basePlayResX: style.basePlayResX,
      basePlayResY: style.basePlayResY,
      mono: null,
      bilingual: style,
    }
  }, [selection, selectionItem])

  const handleConfirmExport = React.useCallback(async () => {
    if (!selectedExportInfo) {
      return
    }

    setExporting(true)
    try {
      const directoryPath = await selectDirectory.mutateAsync({
        title: t("library.config.subtitleStyles.selectExportFolder"),
      })
      if (!directoryPath.trim()) {
        setExporting(false)
        return
      }

      const request: ExportSubtitleStylePresetRequest =
        selectedExportInfo.type === "mono"
          ? {
              directoryPath,
              type: "mono",
              mono: selectedExportInfo.mono as LibraryMonoStyleDTO,
            }
          : {
              directoryPath,
              type: "bilingual",
              bilingual: selectedExportInfo.bilingual as LibraryBilingualStyleDTO,
            }

      const result = (await Call.ByName(
        "dreamcreator/internal/presentation/wails.LibraryHandler.ExportSubtitleStylePreset",
        request,
      )) as ExportSubtitleStylePresetResult

      messageBus.publishToast({
        intent: "success",
        title: t("library.config.subtitleStyles.exportSucceededTitle"),
        description: result.exportPath || result.fileName,
      })
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.config.subtitleStyles.exportFailedTitle"),
        description: error instanceof Error ? error.message : String(error),
      })
    } finally {
      setExporting(false)
    }
  }, [selectDirectory, selectedExportInfo, t])

  const handleSetSelectedDefault = React.useCallback(() => {
    if (!selection || !selectionItem || isDirty) {
      return
    }
    updateValue({
      defaults: {
        ...value.subtitleStyles.defaults,
        ...(selection.kind === "mono"
          ? { monoStyleId: selectionItem.id }
          : { bilingualStyleId: selectionItem.id }),
      },
    })
    onRequestPersist?.()
  }, [
    isDirty,
    onRequestPersist,
    selection,
    selectionItem,
    updateValue,
    value.subtitleStyles.defaults,
  ])

  const openImportFilePicker = React.useCallback(() => {
    setLeftPaneView("create")
    setCreatePaneMode("import")
    fileInputRef.current?.click()
  }, [])

  const handleFileImport = React.useCallback(
    (file: File) => {
      setImportingFile(true)
      void file
        .text()
        .then((content) =>
          Call.ByName(
            "dreamcreator/internal/presentation/wails.LibraryHandler.ParseSubtitleStyleImport",
            {
              content,
              format: file.name.split(".").pop() ?? "",
              filename: file.name,
            } satisfies ParseSubtitleStyleImportRequest,
          ),
        )
        .then((result) => {
          const payload = result as ParseSubtitleStyleImportResult
          const importFormat = (payload.importFormat ?? "").toLowerCase()

          if (importFormat === "dcssp") {
            setImportDraft({
              kind: "dcssp",
              fileName: file.name,
              result: normalizeImportResult(payload),
            })
            setLeftPaneView("create")
            setCreatePaneMode("import")
            return
          }

          setImportDraft({
            kind: "ass",
            fileName: file.name,
            detectedRatio: payload.detectedRatio,
            normalizedPlayResX: payload.normalizedPlayResX,
            normalizedPlayResY: payload.normalizedPlayResY,
            styles: (payload.monoStyles ?? []).map((item) => normalizeMonoStyleForEditor(cloneMonoStyle(item))),
          })
          setLeftPaneView("create")
          setCreatePaneMode("import")
        })
        .catch((error) => {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.config.subtitleStyles.importFailedTitle"),
            description: error instanceof Error ? error.message : String(error),
          })
        })
        .finally(() => {
          setImportingFile(false)
        })
    },
    [t],
  )

  const appendImportedMonoStyles = React.useCallback(
    (styles: LibraryMonoStyleDTO[]) => {
      const nextStyles = styles.map((item) =>
        normalizeMonoStyleForEditor({ ...cloneMonoStyle(item), builtIn: false }),
      )
      if (nextStyles.length === 0) {
        return
      }
      const merged = [...monoStyles, ...nextStyles]
      updateValue({ monoStyles: merged })
      beginMonoEdit(nextStyles[0])
      setLeftPaneView("preview")
      setImportDraft(null)
      onRequestPersist?.()
    },
    [beginMonoEdit, monoStyles, onRequestPersist, updateValue],
  )

  const appendImportedBilingualStyle = React.useCallback(
    (style: LibraryBilingualStyleDTO) => {
      const nextStyle = normalizeBilingualStyleForEditor({
        ...cloneBilingualStyle(style),
        builtIn: false,
      })
      const merged = [...bilingualStyles, nextStyle]
      updateValue({ bilingualStyles: merged })
      beginBilingualEdit(nextStyle)
      setLeftPaneView("preview")
      setImportDraft(null)
      onRequestPersist?.()
    },
    [beginBilingualEdit, bilingualStyles, onRequestPersist, updateValue],
  )

  const handleCreateStyle = React.useCallback(() => {
    const safeDraft = normalizeCreateStyleDraft(createDraft, monoStyles)
    const resolvedName = safeDraft.name.trim()
    const { basePlayResX, basePlayResY } = applyAspectRatioBaseResolution(safeDraft.aspectRatio)

    if (safeDraft.kind === "mono") {
      const template = monoStyles.find((item) => item.id === safeDraft.monoTemplateID)
      const nextStyle = template ? cloneMonoStyle(template) : createEmptyMonoStyle()
      nextStyle.id = createSubtitleStylePresetID("mono")
      nextStyle.name = resolvedName || t("library.config.subtitleStyles.typeMono")
      nextStyle.builtIn = false
      nextStyle.baseAspectRatio = safeDraft.aspectRatio
      nextStyle.basePlayResX = basePlayResX
      nextStyle.basePlayResY = basePlayResY
      nextStyle.style = normalizeMonoStyleForEditor(nextStyle).style

      updateValue({ monoStyles: [...monoStyles, nextStyle] })
      beginMonoEdit(nextStyle)
      setCreateDraft((current) => ({ ...current, name: "" }))
      setLeftPaneView("preview")
      onRequestPersist?.()
      return
    }

    if (monoStyles.length === 0) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.config.subtitleStyles.bilingualRequiresMonoTitle"),
        description: t("library.config.subtitleStyles.bilingualRequiresMonoDescription"),
      })
      return
    }

    const primarySource = monoStyles.find((item) => item.id === safeDraft.bilingualPrimaryTemplateID) ?? monoStyles[0]
    const secondarySource = monoStyles.find((item) => item.id === safeDraft.bilingualSecondaryTemplateID) ?? monoStyles[0]

    const nextStyle = createEmptyBilingualStyle(monoStyles)
    nextStyle.id = createSubtitleStylePresetID("bilingual")
    nextStyle.name = resolvedName || t("library.config.subtitleStyles.typeBilingual")
    nextStyle.builtIn = false
    nextStyle.baseAspectRatio = safeDraft.aspectRatio
    nextStyle.basePlayResX = basePlayResX
    nextStyle.basePlayResY = basePlayResY
    nextStyle.primary = normalizeSnapshotBase(createMonoSnapshotFromStyle(primarySource), safeDraft.aspectRatio, basePlayResX, basePlayResY)
    nextStyle.secondary = normalizeSnapshotBase(createMonoSnapshotFromStyle(secondarySource), safeDraft.aspectRatio, basePlayResX, basePlayResY)
    nextStyle.primary.style = normalizeBilingualStyleForEditor(nextStyle).primary.style
    nextStyle.secondary.style = normalizeBilingualStyleForEditor(nextStyle).secondary.style

    updateValue({ bilingualStyles: [...bilingualStyles, nextStyle] })
    beginBilingualEdit(nextStyle)
    setCreateDraft((current) => ({ ...current, name: "" }))
    setLeftPaneView("preview")
    onRequestPersist?.()
  }, [
    beginBilingualEdit,
    beginMonoEdit,
    bilingualStyles,
    createDraft,
    monoStyles,
    onRequestPersist,
    t,
    updateValue,
  ])

  const toolbarActions = React.useMemo(
    () => (
      <>
        <input
          ref={fileInputRef}
          type="file"
          accept=".ass,.ssa,.dcssp,application/json"
          className="hidden"
          onChange={(event) => {
            const file = event.target.files?.[0]
            event.target.value = ""
            if (!file) {
              return
            }
            handleFileImport(file)
          }}
        />
        <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <Button
                type="button"
                variant="ghost"
                size="compact"
                className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                disabled={importingFile || exporting}
              >
                <Plus className="h-3.5 w-3.5" />
                {t("library.config.subtitleStyles.newPreset")}
                <ChevronDown className="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="start" className="w-52">
              <DropdownMenuItem
                onClick={() => {
                  setLeftPaneView("create")
                  setCreatePaneMode("form")
                }}
              >
                <Plus className="mr-2 h-3.5 w-3.5" />
                {t("library.config.subtitleStyles.createPreset")}
              </DropdownMenuItem>
              <DropdownMenuItem onClick={openImportFilePicker}>
                <FileUp className="mr-2 h-3.5 w-3.5" />
                {t("library.config.subtitleStyles.importPreset")}
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button
            type="button"
            variant="ghost"
            size="compact"
            className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
            disabled={!selectionItem || isDirty || isSelectedDefault}
            onClick={handleSetSelectedDefault}
          >
            <Check className="h-3.5 w-3.5" />
            {t("library.config.subtitleStyles.setDefault")}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="compact"
            className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
            disabled={!canExportSelected || exporting}
            onClick={() => void handleConfirmExport()}
          >
            <Download className="h-3.5 w-3.5" />
            {exporting
              ? t("common.exporting")
              : t("library.config.subtitleStyles.exportPreset")}
          </Button>
        </div>
      </>
    ),
    [
      canExportSelected,
      exporting,
      handleConfirmExport,
      handleFileImport,
      handleSetSelectedDefault,
      importingFile,
      isDirty,
      isSelectedDefault,
      openImportFilePicker,
      selectionItem,
      t,
    ],
  )

  React.useEffect(() => {
    onToolbarActionsChange?.(toolbarActions)
    return () => {
      onToolbarActionsChange?.(null)
    }
  }, [onToolbarActionsChange, toolbarActions])

  return (
    <div className="grid h-full min-h-0 w-full gap-4 xl:grid-cols-[minmax(0,1fr)_320px]">
      <section className="min-h-0 min-w-0 overflow-hidden">
        {leftPaneView === "all" ? (
          <AllStyleCardsPane
            monoStyles={monoStyles}
            bilingualStyles={bilingualStyles}
            fontMappings={fontMappings}
            defaults={subtitleStyleDefaults}
            selection={selection}
            onSelect={(next) => selectStyle(next)}
          />
        ) : leftPaneView === "create" ? (
          <CreateStylePane
            mode={createPaneMode}
            onModeChange={setCreatePaneMode}
            createDraft={createDraft}
            monoStyles={monoStyles}
            importingFile={importingFile}
            importDraft={importDraft}
            onCreateDraftChange={setCreateDraft}
            onCreate={handleCreateStyle}
            onOpenImport={openImportFilePicker}
            onApplyImportMono={appendImportedMonoStyles}
            onApplyImportBilingual={appendImportedBilingualStyle}
            onImportDraftChange={setImportDraft}
          />
        ) : (
          <PreviewPane
            draftKind={draftKind}
            monoDraft={monoDraft}
            bilingualDraft={bilingualDraft}
            monoStyles={monoStyles}
            fontMappings={fontMappings}
          />
        )}
      </section>

      <aside className="min-h-0 w-full xl:w-[320px]">
        <div className={cn("flex h-full min-h-0 flex-col overflow-hidden", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
          <PresetCompositeHeader
            selection={selection}
            monoStyles={monoStyles}
            bilingualStyles={bilingualStyles}
            fontMappings={fontMappings}
            defaults={subtitleStyleDefaults}
            onSelect={(next) => selectStyle(next)}
            onShowAll={() => {
              if (isDirty) {
                messageBus.publishToast({
                  intent: "warning",
                  title: t("library.config.subtitleStyles.unsavedTitle"),
                  description: t("library.config.subtitleStyles.unsavedDescription"),
                })
                return
              }
              setLeftPaneView("all")
            }}
          />

          <div className="min-h-0 flex-1 overflow-hidden border-y border-border/60">
            {currentDraft && draftKind ? (
              <div className="h-full min-h-0 overflow-y-auto px-3 py-3">
                <div className="space-y-3">
                  <EditorGroupCard>
                    <EditorRow label={t("library.config.subtitleStyles.presetName")}>
                      <Input
                        value={currentDraft.name}
                        className="h-8 text-xs md:text-xs"
                        onChange={(event) => {
                          const nextName = event.target.value
                          if (draftKind === "mono") {
                            setMonoDraft((current) => (current ? { ...current, name: nextName } : current))
                          } else {
                            setBilingualDraft((current) => (current ? { ...current, name: nextName } : current))
                          }
                        }}
                      />
                    </EditorRow>
                  </EditorGroupCard>

                  {draftKind === "mono" && monoDraft ? (
                    <MonoStyleEditor
                      draft={monoDraft}
                      onChange={setMonoDraft}
                    />
                  ) : null}

                  {draftKind === "bilingual" && bilingualDraft ? (
                    <BilingualStyleEditor
                      draft={bilingualDraft}
                      monoStyles={monoStyles}
                      onChange={setBilingualDraft}
                    />
                  ) : null}
                </div>
              </div>
            ) : (
              <div className="flex h-full items-center justify-center p-4">
                <SubtitleStyleEmptyState
                  icon={Captions}
                  title={t("library.config.subtitleStyles.emptyDetailTitle")}
                  description={t("library.config.subtitleStyles.emptyDetailDescription")}
                />
              </div>
            )}
          </div>

          <div className="flex items-center gap-2 p-3">
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="flex-1 gap-1.5"
              disabled={!isDirty}
              onClick={resetDraft}
            >
              <RotateCcw className="h-3.5 w-3.5" />
              {t("common.undo")}
            </Button>
            <Button
              type="button"
              size="compact"
              className="flex-1 gap-1.5"
              disabled={!isDirty || !currentDraft}
              onClick={saveDraft}
            >
              <Save className="h-3.5 w-3.5" />
              {t("common.save")}
            </Button>
          </div>
        </div>
      </aside>
    </div>
  )
}
