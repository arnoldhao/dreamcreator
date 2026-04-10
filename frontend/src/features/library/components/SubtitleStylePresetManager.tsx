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

import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui/empty"
import { useFontCatalog } from "@/hooks/useFontCatalog"
import { cn } from "@/lib/utils"
import {
  applyFontCatalogFaceToStyle,
  applyFontFamilyToStyle,
  resolveAssStyleFontFace,
  resolveAssStyleFontItalic,
  resolveAssStyleFontWeight,
  resolveFontCatalogFaces,
  resolveFontCatalogFamily,
  toggleAssStyleBold,
  toggleAssStyleItalic,
} from "@/shared/fonts/fontCatalog"
import { useI18n } from "@/shared/i18n"
import { messageBus } from "@/shared/message"
import { useSelectDirectory } from "@/shared/query/settings"
import type {
  AssStyleSpecDTO,
  ExportSubtitleStylePresetRequest,
  ExportSubtitleStylePresetResult,
  LibraryBilingualStyleDTO,
  LibraryModuleConfigDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
  ParseSubtitleStyleImportRequest,
  ParseSubtitleStyleImportResult,
} from "@/shared/contracts/library"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { Input } from "@/shared/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"

import { SubtitleStylePresetPreview } from "./SubtitleStylePresetPreview"
import {
  SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS,
  applyAspectRatioBaseResolution,
  cloneAssStyleSpec,
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
import { buildSubtitleStyleNamePreviewStyle } from "../utils/subtitleStyleNamePreview"
import { resolveSubtitleStyleDefaults } from "../utils/subtitleStyles"

type SubtitleStylePresetManagerProps = {
  value: LibraryModuleConfigDTO
  onChange: (next: LibraryModuleConfigDTO) => void
  onRequestPersist?: () => void
  onToolbarActionsChange?: (actions: React.ReactNode | null) => void
}

type LeftPaneView = "preview" | "all" | "create"
type CreatePaneMode = "form" | "import"

type ImportDraftState =
  | {
      kind: "ass"
      fileName: string
      detectedRatio?: string
      normalizedPlayResX?: number
      normalizedPlayResY?: number
      styles: LibraryMonoStyleDTO[]
    }
  | {
      kind: "dcssp"
      fileName: string
      result: ParseSubtitleStyleImportResult
    }
  | null

type CreateStyleDraft = {
  kind: SubtitleStylePresetKind
  name: string
  aspectRatio: string
  monoTemplateID: string
  bilingualPrimaryTemplateID: string
  bilingualSecondaryTemplateID: string
}

function signatureOf(value: unknown) {
  return JSON.stringify(value)
}

function resolveSelectionItem(
  selection: SubtitleStylePresetSelection,
  monoStyles: LibraryMonoStyleDTO[],
  bilingualStyles: LibraryBilingualStyleDTO[],
) {
  if (!selection) {
    return null
  }
  if (selection.kind === "mono") {
    return monoStyles.find((item) => item.id === selection.id) ?? null
  }
  return bilingualStyles.find((item) => item.id === selection.id) ?? null
}

function isDefaultMonoStyle(
  defaults: { monoStyleId?: string },
  style: LibraryMonoStyleDTO,
) {
  return (defaults.monoStyleId ?? "").trim() === style.id
}

function isDefaultBilingualStyle(
  defaults: { bilingualStyleId?: string },
  style: LibraryBilingualStyleDTO,
) {
  return (defaults.bilingualStyleId ?? "").trim() === style.id
}

function createInitialCreateStyleDraft(monoStyles: LibraryMonoStyleDTO[]): CreateStyleDraft {
  const firstMono = monoStyles[0]?.id ?? ""
  const secondMono = monoStyles[1]?.id ?? firstMono
  return {
    kind: "mono",
    name: "",
    aspectRatio: "16:9",
    monoTemplateID: firstMono,
    bilingualPrimaryTemplateID: firstMono,
    bilingualSecondaryTemplateID: secondMono,
  }
}

function normalizeCreateStyleDraft(current: CreateStyleDraft, monoStyles: LibraryMonoStyleDTO[]): CreateStyleDraft {
  const next = { ...current }
  const monoIDs = new Set(monoStyles.map((item) => item.id))
  const firstMono = monoStyles[0]?.id ?? ""
  const secondMono = monoStyles[1]?.id ?? firstMono

  if (!monoIDs.has(next.monoTemplateID)) {
    next.monoTemplateID = firstMono
  }
  if (!monoIDs.has(next.bilingualPrimaryTemplateID)) {
    next.bilingualPrimaryTemplateID = firstMono
  }
  if (!monoIDs.has(next.bilingualSecondaryTemplateID)) {
    next.bilingualSecondaryTemplateID = secondMono
  }

  if (!SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS.some((option) => option.value === next.aspectRatio)) {
    next.aspectRatio = "16:9"
  }

  return next
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
  const [createPaneMode, setCreatePaneMode] = React.useState<CreatePaneMode>("form")
  const [createDraft, setCreateDraft] = React.useState<CreateStyleDraft>(() => createInitialCreateStyleDraft(monoStyles))
  const [importDraft, setImportDraft] = React.useState<ImportDraftState>(null)
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
      nextStyle.style = normalizeAssStyleForEditor(nextStyle.style)

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
    nextStyle.primary.style = normalizeAssStyleForEditor(nextStyle.primary.style)
    nextStyle.secondary.style = normalizeAssStyleForEditor(nextStyle.secondary.style)

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
            fontMappings={value.subtitleStyles.fonts ?? []}
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

function PreviewPane(props: {
  draftKind: SubtitleStylePresetKind | null
  monoDraft: LibraryMonoStyleDTO | null
  bilingualDraft: LibraryBilingualStyleDTO | null
  monoStyles: LibraryMonoStyleDTO[]
  fontMappings: LibraryModuleConfigDTO["subtitleStyles"]["fonts"]
}) {
  const { t } = useI18n()
  const [previewResolution, setPreviewResolution] = React.useState<{ width: number; height: number } | null>(null)

  if (!props.draftKind || (!props.monoDraft && !props.bilingualDraft)) {
    return (
      <div className={cn("flex h-full min-h-0 w-full flex-col overflow-hidden p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
        <SubtitleStyleEmptyState
          icon={Captions}
          title={t("library.config.subtitleStyles.emptyDetailTitle")}
          description={t("library.config.subtitleStyles.emptyDetailDescription")}
        />
      </div>
    )
  }

  const styleForInfo = props.draftKind === "mono" ? props.monoDraft?.style : props.bilingualDraft?.primary.style
  const nameForInfo = props.draftKind === "mono" ? props.monoDraft?.name : props.bilingualDraft?.name
  const ratioForInfo = props.draftKind === "mono" ? props.monoDraft?.baseAspectRatio : props.bilingualDraft?.baseAspectRatio
  const resX = props.draftKind === "mono" ? props.monoDraft?.basePlayResX : props.bilingualDraft?.basePlayResX
  const resY = props.draftKind === "mono" ? props.monoDraft?.basePlayResY : props.bilingualDraft?.basePlayResY
  const primarySourceName =
    props.bilingualDraft?.primary.sourceMonoStyleID
      ? props.monoStyles.find((style) => style.id === props.bilingualDraft?.primary.sourceMonoStyleID)?.name ?? "-"
      : "-"
  const secondarySourceName =
    props.bilingualDraft?.secondary.sourceMonoStyleID
      ? props.monoStyles.find((style) => style.id === props.bilingualDraft?.secondary.sourceMonoStyleID)?.name ?? "-"
      : "-"

  return (
    <div className="flex h-full min-h-0 flex-col gap-4">
      <div className={cn("p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
        <SubtitleStylePresetPreview
          kind={props.draftKind}
          mono={props.monoDraft}
          bilingual={props.bilingualDraft}
          fontMappings={props.fontMappings}
          onPreviewSizeChange={setPreviewResolution}
        />
      </div>

      <div className={cn("min-h-0 flex-1 overflow-y-auto p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
        <div className="space-y-3">
          <div className="text-sm font-semibold text-foreground">
            {t("library.config.subtitleStyles.previewInfoTitle")}
          </div>

          <div className="grid gap-x-4 gap-y-2 md:grid-cols-2">
            <InfoItem
              label={t("library.config.subtitleStyles.typeLabel")}
              value={
                props.draftKind === "mono"
                  ? t("library.config.subtitleStyles.monoSection")
                  : t("library.config.subtitleStyles.bilingualSection")
              }
            />
            <InfoItem label={t("library.config.subtitleStyles.nameLabel")} value={nameForInfo || "-"} />
            <InfoItem label={t("library.config.subtitleStyles.aspectRatioLabel")} value={ratioForInfo || "-"} />
            <InfoItem label={t("library.config.subtitleStyles.resolutionLabel")} value={resX && resY ? `${resX}×${resY}` : "-"} />
            <InfoItem
              label={t("library.config.subtitleStyles.currentResolutionLabel")}
              value={previewResolution ? `${previewResolution.width}×${previewResolution.height}` : "-"}
            />
            <InfoItem
              label={t("library.config.subtitleStyles.fontFamily")}
              value={styleForInfo?.fontname || "-"}
            />
            <InfoItem
              label={t("library.config.subtitleStyles.fontSize")}
              value={styleForInfo ? String(Math.round(styleForInfo.fontsize)) : "-"}
            />
            <InfoItem
              label={t("library.config.subtitleStyles.styleFlagsLabel")}
              value={styleForInfo ? formatStyleFlags(styleForInfo, t) : "-"}
            />

            {props.draftKind === "bilingual" && props.bilingualDraft ? (
              <>
                <InfoItem
                  label={t("library.config.subtitleStyles.primarySourceTitle")}
                  value={primarySourceName}
                />
                <InfoItem
                  label={t("library.config.subtitleStyles.secondarySourceTitle")}
                  value={secondarySourceName}
                />
              </>
            ) : null}
          </div>
        </div>
      </div>
    </div>
  )
}

function AllStyleCardsPane(props: {
  monoStyles: LibraryMonoStyleDTO[]
  bilingualStyles: LibraryBilingualStyleDTO[]
  fontMappings: LibrarySubtitleStyleFontDTO[]
  defaults: { monoStyleId?: string; bilingualStyleId?: string }
  selection: SubtitleStylePresetSelection
  onSelect: (selection: SubtitleStylePresetSelection) => void
}) {
  const { t } = useI18n()
  const totalCount = props.monoStyles.length + props.bilingualStyles.length
  const adaptiveGridStyle = React.useMemo(
    () => ({
      gridTemplateColumns: "repeat(auto-fill, minmax(min(100%, 15.5rem), 1fr))",
    }),
    [],
  )

  return (
    <div className={cn("flex h-full min-h-0 w-full flex-col overflow-hidden p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
      <div className="mb-3 flex items-center justify-between">
        <div className="text-sm font-semibold text-foreground">
          {t("library.config.subtitleStyles.allStylesTitle")}
        </div>
        <Badge variant="outline" className="text-[10px]">
          {totalCount}
        </Badge>
      </div>

      {totalCount === 0 ? (
        <SubtitleStyleEmptyState
          icon={Captions}
          title={t("library.config.subtitleStyles.emptyStateTitle")}
          description={t("library.config.subtitleStyles.emptyStateDescription")}
        />
      ) : (
        <div className="min-h-0 flex-1 overflow-y-auto pr-1">
          <div className="space-y-4">
            {props.monoStyles.length > 0 ? (
              <section className="space-y-2">
                <div className="text-xs font-semibold text-foreground/80">
                  {t("library.config.subtitleStyles.monoSection")}
                </div>
                <div className="grid gap-3" style={adaptiveGridStyle}>
                  {props.monoStyles.map((style) => {
                    const selected = props.selection?.kind === "mono" && props.selection.id === style.id
                    const isDefault = isDefaultMonoStyle(props.defaults, style)
                    const styleSpec = resolveStyleForRendering("mono", style)
                    return (
                      <PresetSummaryCard
                        key={`mono-${style.id}`}
                        name={style.name}
                        selected={selected}
                        styleSpec={styleSpec}
                        fontMappings={props.fontMappings}
                        kindLabel={t("library.config.subtitleStyles.monoSection")}
                        builtIn={style.builtIn === true}
                        isDefault={isDefault}
                        aspectRatio={style.baseAspectRatio}
                        builtInLabel={t("library.config.subtitleStyles.builtinBadge")}
                        defaultLabel={t("library.config.subtitleStyles.defaultBadge")}
                        onClick={() => props.onSelect({ kind: "mono", id: style.id })}
                      />
                    )
                  })}
                </div>
              </section>
            ) : null}

            {props.bilingualStyles.length > 0 ? (
              <section className="space-y-2">
                <div className="text-xs font-semibold text-foreground/80">
                  {t("library.config.subtitleStyles.bilingualSection")}
                </div>
                <div className="grid gap-3" style={adaptiveGridStyle}>
                  {props.bilingualStyles.map((style) => {
                    const selected = props.selection?.kind === "bilingual" && props.selection.id === style.id
                    const isDefault = isDefaultBilingualStyle(props.defaults, style)
                    const styleSpec = resolveStyleForRendering("bilingual", style)
                    return (
                      <PresetSummaryCard
                        key={`bilingual-${style.id}`}
                        name={style.name}
                        selected={selected}
                        styleSpec={styleSpec}
                        fontMappings={props.fontMappings}
                        kindLabel={t("library.config.subtitleStyles.bilingualSection")}
                        builtIn={style.builtIn === true}
                        isDefault={isDefault}
                        aspectRatio={style.baseAspectRatio}
                        builtInLabel={t("library.config.subtitleStyles.builtinBadge")}
                        defaultLabel={t("library.config.subtitleStyles.defaultBadge")}
                        onClick={() => props.onSelect({ kind: "bilingual", id: style.id })}
                      />
                    )
                  })}
                </div>
              </section>
            ) : null}
          </div>
        </div>
      )}
    </div>
  )
}

function PresetSummaryCard(props: {
  name: string
  selected: boolean
  styleSpec: AssStyleSpecDTO
  fontMappings: LibrarySubtitleStyleFontDTO[]
  kindLabel: string
  builtIn: boolean
  isDefault: boolean
  aspectRatio?: string
  builtInLabel: string
  defaultLabel: string
  onClick: () => void
}) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      className={cn(
        "flex min-h-[84px] w-full min-w-0 flex-col rounded-xl border p-3 text-left transition-colors",
        props.selected
          ? "border-border/80 bg-background/95 shadow-sm"
          : "border-border/40 bg-background/70 hover:border-border/70 hover:bg-background/90",
      )}
    >
      <div
        className="truncate text-sm"
        style={buildSubtitleStyleNamePreviewStyle(props.styleSpec, props.fontMappings)}
        title={props.name}
      >
        {props.name || "-"}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-1.5 text-[11px] text-muted-foreground">
        <PresetMetaBadge variant="outline">{props.kindLabel}</PresetMetaBadge>
        {props.builtIn ? (
          <PresetMetaBadge variant="secondary">{props.builtInLabel}</PresetMetaBadge>
        ) : null}
        {props.isDefault ? <PresetMetaBadge>{props.defaultLabel}</PresetMetaBadge> : null}
        <span className="shrink-0 whitespace-nowrap rounded-md border border-border/60 bg-background/55 px-2 py-0.5 font-mono text-[10px] leading-4 text-muted-foreground">
          {props.aspectRatio || "-"}
        </span>
      </div>
    </button>
  )
}

function PresetMetaBadge(props: React.ComponentProps<typeof Badge>) {
  const { className, ...rest } = props
  return <Badge className={cn("shrink-0 whitespace-nowrap text-[10px]", className)} {...rest} />
}

function CreateStylePane(props: {
  mode: CreatePaneMode
  onModeChange: (mode: CreatePaneMode) => void
  createDraft: CreateStyleDraft
  monoStyles: LibraryMonoStyleDTO[]
  importingFile: boolean
  importDraft: ImportDraftState
  onCreateDraftChange: React.Dispatch<React.SetStateAction<CreateStyleDraft>>
  onCreate: () => void
  onOpenImport: () => void
  onApplyImportMono: (styles: LibraryMonoStyleDTO[]) => void
  onApplyImportBilingual: (style: LibraryBilingualStyleDTO) => void
  onImportDraftChange: React.Dispatch<React.SetStateAction<ImportDraftState>>
}) {
  const { t } = useI18n()

  return (
    <div className={cn("flex h-full min-h-0 w-full flex-col overflow-hidden p-4", DASHBOARD_DIALOG_SOFT_SURFACE_CLASS)}>
      <div className="min-h-0 flex-1 overflow-y-auto pr-1">
        <div className="mx-auto flex min-h-full w-full max-w-2xl flex-col justify-center gap-3 py-2">
          <div className="flex justify-center">
            <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/70">
              <Button
                type="button"
                variant="ghost"
                size="compact"
                className={cn("rounded-none border-0 px-3 text-xs", props.mode === "form" ? "bg-background" : "")}
                onClick={() => props.onModeChange("form")}
              >
                <Plus className="h-3.5 w-3.5" />
                {t("library.config.subtitleStyles.createPreset")}
              </Button>
              <Button
                type="button"
                variant="ghost"
                size="compact"
                className={cn("rounded-none border-0 border-l border-border/60 px-3 text-xs", props.mode === "import" ? "bg-background" : "")}
                onClick={() => props.onModeChange("import")}
              >
                <FileUp className="h-3.5 w-3.5" />
                {t("library.config.subtitleStyles.importPreset")}
              </Button>
            </div>
          </div>

          {props.mode === "form" ? (
            <EditorGroupCard title={t("library.config.subtitleStyles.createFormTitle")}>
              <EditorRow label={t("library.config.subtitleStyles.presetName")}>
                <Input
                  value={props.createDraft.name}
                  className="h-8 text-xs md:text-xs"
                  onChange={(event) =>
                    props.onCreateDraftChange((current) => ({
                      ...current,
                      name: event.target.value,
                    }))
                  }
                  placeholder={t("library.config.subtitleStyles.nameLabel")}
                />
              </EditorRow>

              <EditorRow label={t("library.config.subtitleStyles.typeLabel")}>
                <Tabs
                  value={props.createDraft.kind}
                  onValueChange={(value) =>
                    props.onCreateDraftChange((current) => ({
                      ...current,
                      kind: value as SubtitleStylePresetKind,
                    }))
                  }
                  className="w-full"
                >
                  <div className="flex justify-end">
                    <TabsList>
                      <TabsTrigger value="mono">
                        {t("library.config.subtitleStyles.monoSection")}
                      </TabsTrigger>
                      <TabsTrigger value="bilingual">
                        {t("library.config.subtitleStyles.bilingualSection")}
                      </TabsTrigger>
                    </TabsList>
                  </div>
                </Tabs>
              </EditorRow>

              <EditorRow label={t("library.config.subtitleStyles.aspectRatioLabel")}>
                <NativeSelect
                  value={props.createDraft.aspectRatio}
                  onChange={(event) =>
                    props.onCreateDraftChange((current) => ({
                      ...current,
                      aspectRatio: event.target.value,
                    }))
                  }
                >
                  {SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>
                      {option.label}
                    </option>
                  ))}
                </NativeSelect>
              </EditorRow>

              {props.createDraft.kind === "mono" ? (
                <EditorRow label={t("library.config.subtitleStyles.copyStyleLabel")}>
                  <NativeSelect
                    value={props.createDraft.monoTemplateID}
                    onChange={(event) =>
                      props.onCreateDraftChange((current) => ({
                        ...current,
                        monoTemplateID: event.target.value,
                      }))
                    }
                  >
                    <option value="">{t("library.config.subtitleStyles.emptyStyleTemplate")}</option>
                    {props.monoStyles.map((style) => (
                      <option key={style.id} value={style.id}>
                        {style.name}
                      </option>
                    ))}
                  </NativeSelect>
                </EditorRow>
              ) : (
                <>
                  <EditorRow label={t("library.config.subtitleStyles.primarySourceTitle")}>
                    <NativeSelect
                      value={props.createDraft.bilingualPrimaryTemplateID}
                      onChange={(event) =>
                        props.onCreateDraftChange((current) => ({
                          ...current,
                          bilingualPrimaryTemplateID: event.target.value,
                        }))
                      }
                    >
                      {props.monoStyles.map((style) => (
                        <option key={style.id} value={style.id}>
                          {style.name}
                        </option>
                      ))}
                    </NativeSelect>
                  </EditorRow>
                  <EditorRow label={t("library.config.subtitleStyles.secondarySourceTitle")}>
                    <NativeSelect
                      value={props.createDraft.bilingualSecondaryTemplateID}
                      onChange={(event) =>
                        props.onCreateDraftChange((current) => ({
                          ...current,
                          bilingualSecondaryTemplateID: event.target.value,
                        }))
                      }
                    >
                      {props.monoStyles.map((style) => (
                        <option key={style.id} value={style.id}>
                          {style.name}
                        </option>
                      ))}
                    </NativeSelect>
                  </EditorRow>
                </>
              )}

              <div className="pt-1">
                <Button
                  type="button"
                  size="compact"
                  className="w-full gap-2"
                  onClick={props.onCreate}
                  disabled={props.createDraft.kind === "bilingual" && props.monoStyles.length === 0}
                >
                  <Check className="h-3.5 w-3.5" />
                  {t("common.create")}
                </Button>
              </div>
            </EditorGroupCard>
          ) : (
            <>
              <EditorGroupCard title={t("library.config.subtitleStyles.importGuideTitle")}>
                <div className="space-y-2 text-xs text-muted-foreground">
                  <div>{t("library.config.subtitleStyles.importGuideText")}</div>
                  <Button type="button" size="compact" variant="outline" className="gap-2" onClick={props.onOpenImport}>
                    <FileUp className="h-3.5 w-3.5" />
                    {props.importingFile ? t("common.loading") : t("library.config.subtitleStyles.importPreset")}
                  </Button>
                </div>
              </EditorGroupCard>

              {props.importDraft?.kind === "ass" ? (
                <EditorGroupCard
                  title={`${props.importDraft.fileName}${
                    props.importDraft.detectedRatio
                      ? ` · ${props.importDraft.detectedRatio} · ${props.importDraft.normalizedPlayResX ?? "-"}×${props.importDraft.normalizedPlayResY ?? "-"}`
                      : ""
                  }`}
                >
                  <div className="space-y-2">
                    {props.importDraft.styles.map((style, index) => (
                      <div key={style.id} className={cn("space-y-2 rounded-lg px-3 py-2", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
                        <div className="text-xs text-muted-foreground">
                          {style.sourceAssStyleName || `${t("library.config.subtitleStyles.importStylePrefix")} ${index + 1}`}
                        </div>
                        <Input
                          value={style.name}
                          className="h-8 text-xs md:text-xs"
                          onChange={(event) =>
                            props.onImportDraftChange((current) =>
                              current?.kind === "ass"
                                ? {
                                    ...current,
                                    styles: current.styles.map((item) =>
                                      item.id === style.id ? { ...item, name: event.target.value } : item,
                                    ),
                                  }
                                : current,
                            )
                          }
                        />
                      </div>
                    ))}
                  </div>

                  <Button
                    type="button"
                    size="compact"
                    className="mt-2 w-full gap-2"
                    onClick={() => props.onApplyImportMono(props.importDraft?.kind === "ass" ? props.importDraft.styles : [])}
                    disabled={props.importDraft.styles.length === 0}
                  >
                    <Check className="h-3.5 w-3.5" />
                    {t("common.import")}
                  </Button>
                </EditorGroupCard>
              ) : null}

              {props.importDraft?.kind === "dcssp" ? (
                <EditorGroupCard title={props.importDraft.fileName}>
                  <div className="space-y-2 text-xs">
                    <InfoItem label={t("library.config.subtitleStyles.importDetailFormatLabel")} value={props.importDraft.result.dcssp?.format ?? "-"} />
                    <InfoItem label={t("library.config.subtitleStyles.importDetailTypeLabel")} value={props.importDraft.result.dcssp?.type ?? "-"} />
                    <InfoItem label={t("library.config.subtitleStyles.importDetailNameLabel")} value={props.importDraft.result.dcssp?.name ?? "-"} />
                    <InfoItem label={t("library.config.subtitleStyles.importDetailAuthorLabel")} value={props.importDraft.result.dcssp?.author ?? "-"} />
                  </div>

                  <Button
                    type="button"
                    size="compact"
                    className="mt-2 w-full gap-2"
                    onClick={() => {
                      if (props.importDraft?.kind !== "dcssp") {
                        return
                      }
                      if (props.importDraft.result.bilingualStyle) {
                        props.onApplyImportBilingual(props.importDraft.result.bilingualStyle)
                        return
                      }
                      props.onApplyImportMono(props.importDraft.result.monoStyles ?? [])
                    }}
                    disabled={
                      !props.importDraft.result.bilingualStyle &&
                      (props.importDraft.result.monoStyles?.length ?? 0) === 0
                    }
                  >
                    <Check className="h-3.5 w-3.5" />
                    {t("common.import")}
                  </Button>
                </EditorGroupCard>
              ) : null}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function PresetCompositeHeader(props: {
  selection: SubtitleStylePresetSelection
  monoStyles: LibraryMonoStyleDTO[]
  bilingualStyles: LibraryBilingualStyleDTO[]
  fontMappings: LibrarySubtitleStyleFontDTO[]
  defaults: { monoStyleId?: string; bilingualStyleId?: string }
  onSelect: (selection: SubtitleStylePresetSelection) => void
  onShowAll: () => void
}) {
  const { t } = useI18n()
  const selectedItem = React.useMemo(
    () => resolveSelectionItem(props.selection, props.monoStyles, props.bilingualStyles),
    [props.bilingualStyles, props.monoStyles, props.selection],
  )
  const selectedKind = props.selection?.kind ?? "mono"
  const selectedStyle = selectedItem
    ? resolveStyleForRendering(selectedKind, selectedItem as LibraryMonoStyleDTO | LibraryBilingualStyleDTO)
    : null
  const selectedIsDefault =
    selectedKind === "mono"
      ? selectedItem
        ? isDefaultMonoStyle(props.defaults, selectedItem as LibraryMonoStyleDTO)
        : false
      : selectedItem
        ? isDefaultBilingualStyle(props.defaults, selectedItem as LibraryBilingualStyleDTO)
        : false
  const selectedIsBuiltIn = (selectedItem as (LibraryMonoStyleDTO | LibraryBilingualStyleDTO | null))?.builtIn === true

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button type="button" className="flex w-full items-center gap-2 p-3 text-left hover:bg-background/60">
          <div className="min-w-0 flex-1 overflow-hidden">
            <div
              className="truncate text-sm"
              style={buildSubtitleStyleNamePreviewStyle(selectedStyle, props.fontMappings)}
              title={(selectedItem as LibraryMonoStyleDTO | LibraryBilingualStyleDTO | null)?.name || ""}
            >
              {(selectedItem as LibraryMonoStyleDTO | LibraryBilingualStyleDTO | null)?.name ||
                t("library.config.subtitleStyles.emptyStateTitle")}
            </div>
            {selectedItem ? (
              <div className="mt-1 flex items-center gap-2">
                {selectedIsBuiltIn ? (
                  <Badge variant="secondary" className="text-[10px]">
                    {t("library.config.subtitleStyles.builtinBadge")}
                  </Badge>
                ) : null}
                {selectedIsDefault ? (
                  <Badge className="text-[10px]">
                    {t("library.config.subtitleStyles.defaultBadge")}
                  </Badge>
                ) : null}
              </div>
            ) : null}
          </div>
          <ChevronDown className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-64">
          <DropdownMenuLabel>{t("library.config.subtitleStyles.monoSection")}</DropdownMenuLabel>
          {props.monoStyles.length === 0 ? (
            <DropdownMenuItem disabled>{t("library.config.subtitleStyles.emptyMono")}</DropdownMenuItem>
          ) : (
            props.monoStyles.map((style) => (
              <DropdownMenuItem key={style.id} onClick={() => props.onSelect({ kind: "mono", id: style.id })}>
                <span
                  className="block min-w-0 truncate"
                  style={buildSubtitleStyleNamePreviewStyle(style.style, props.fontMappings)}
                  title={style.name}
                >
                  {style.name}
                </span>
              </DropdownMenuItem>
            ))
          )}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>{t("library.config.subtitleStyles.bilingualSection")}</DropdownMenuLabel>
          {props.bilingualStyles.length === 0 ? (
            <DropdownMenuItem disabled>
              {t("library.config.subtitleStyles.emptyBilingual")}
            </DropdownMenuItem>
          ) : (
            props.bilingualStyles.map((style) => (
              <DropdownMenuItem key={style.id} onClick={() => props.onSelect({ kind: "bilingual", id: style.id })}>
                <span
                  className="block min-w-0 truncate"
                  style={buildSubtitleStyleNamePreviewStyle(style.primary.style, props.fontMappings)}
                  title={style.name}
                >
                  {style.name}
                </span>
              </DropdownMenuItem>
            ))
          )}

          <DropdownMenuSeparator />
          <DropdownMenuItem onClick={props.onShowAll}>
            {t("library.config.subtitleStyles.allStylesTitle")}
          </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function MonoStyleEditor(props: {
  draft: LibraryMonoStyleDTO
  onChange: React.Dispatch<React.SetStateAction<LibraryMonoStyleDTO | null>>
}) {
  const { t } = useI18n()

  return (
    <AssStyleEditor
      title={t("library.config.subtitleStyles.monoStyleSectionTitle")}
      style={props.draft.style}
      onChange={(nextStyle) =>
        props.onChange((current) => (current ? { ...current, style: nextStyle } : current))
      }
    />
  )
}

function BilingualStyleEditor(props: {
  draft: LibraryBilingualStyleDTO
  monoStyles: LibraryMonoStyleDTO[]
  onChange: React.Dispatch<React.SetStateAction<LibraryBilingualStyleDTO | null>>
}) {
  const { t } = useI18n()
  const [activeLane, setActiveLane] = React.useState<"primary" | "secondary">("primary")
  const alignmentOptions = React.useMemo(() => resolveAlignmentOptions(t), [t])

  return (
    <div className="space-y-3">
      <EditorGroupCard title={t("library.config.subtitleStyles.bilingualMetaSectionTitle")}> 
        <EditorRow label={t("library.config.subtitleStyles.gapLabel")}> 
          <NumberInput
            value={props.draft.layout.gap}
            onChange={(value) =>
              props.onChange((current) =>
                current ? { ...current, layout: { ...current.layout, gap: value } } : current,
              )
            }
          />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.blockAnchorLabel")}> 
          <NativeSelect
            value={String(props.draft.layout.blockAnchor)}
            onChange={(event) =>
              props.onChange((current) =>
                current
                  ? { ...current, layout: { ...current.layout, blockAnchor: Number(event.target.value) } }
                  : current,
              )
            }
          >
            {alignmentOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.primarySourceTitle")}> 
          <NativeSelect
            value={props.draft.primary.sourceMonoStyleID ?? ""}
            onChange={(event) => {
              const source = props.monoStyles.find((item) => item.id === event.target.value)
              if (!source) {
                return
              }
              props.onChange((current) =>
                current
                  ? {
                      ...current,
                      primary: createMonoSnapshotFromStyle(source),
                    }
                  : current,
              )
            }}
          >
            <option value="">{t("library.config.subtitleStyles.selectSourceMonoStyle")}</option>
            {props.monoStyles.map((style) => (
              <option key={style.id} value={style.id}>
                {style.name}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.secondarySourceTitle")}> 
          <NativeSelect
            value={props.draft.secondary.sourceMonoStyleID ?? ""}
            onChange={(event) => {
              const source = props.monoStyles.find((item) => item.id === event.target.value)
              if (!source) {
                return
              }
              props.onChange((current) =>
                current
                  ? {
                      ...current,
                      secondary: createMonoSnapshotFromStyle(source),
                    }
                  : current,
              )
            }}
          >
            <option value="">{t("library.config.subtitleStyles.selectSourceMonoStyle")}</option>
            {props.monoStyles.map((style) => (
              <option key={style.id} value={style.id}>
                {style.name}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>
      </EditorGroupCard>

      <Tabs
        value={activeLane}
        onValueChange={(value) => setActiveLane(value as "primary" | "secondary")}
        className="space-y-3"
      >
        <div className="flex justify-center">
          <TabsList>
            <TabsTrigger value="primary">
              {t("library.config.subtitleStyles.primaryTabLabel")}
            </TabsTrigger>
            <TabsTrigger value="secondary">
              {t("library.config.subtitleStyles.secondaryTabLabel")}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="primary" className="mt-0">
          <AssStyleEditor
            title={t("library.config.subtitleStyles.primaryStyleTitle")}
            style={props.draft.primary.style}
            onChange={(nextStyle) =>
              props.onChange((current) =>
                current ? { ...current, primary: { ...current.primary, style: nextStyle } } : current,
              )
            }
          />
        </TabsContent>

        <TabsContent value="secondary" className="mt-0">
          <AssStyleEditor
            title={t("library.config.subtitleStyles.secondaryStyleTitle")}
            style={props.draft.secondary.style}
            onChange={(nextStyle) =>
              props.onChange((current) =>
                current ? { ...current, secondary: { ...current.secondary, style: nextStyle } } : current,
              )
            }
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function AssStyleEditor(props: {
  title: string
  style: AssStyleSpecDTO
  onChange: (value: AssStyleSpecDTO) => void
}) {
  const { t } = useI18n()
  const { data: fontCatalog = [], isLoading: fontCatalogLoading } = useFontCatalog()
  const alignmentOptions = React.useMemo(() => resolveAlignmentOptions(t), [t])
  const borderStyleOptions = React.useMemo(() => resolveBorderStyleOptions(t), [t])
  const selectedFontFamily = React.useMemo(
    () => resolveFontCatalogFamily(fontCatalog, props.style.fontname),
    [fontCatalog, props.style.fontname],
  )

  const fontOptions = React.useMemo(() => {
    const options = new Set(fontCatalog.map((family) => family.family.trim()).filter(Boolean))
    if (props.style.fontname.trim()) {
      options.add(props.style.fontname.trim())
    }
    return [...options].sort((left, right) => left.localeCompare(right))
  }, [fontCatalog, props.style.fontname])

  const fontFaceOptions = React.useMemo(
    () => resolveFontCatalogFaces(fontCatalog, props.style.fontname, props.style),
    [fontCatalog, props.style],
  )

  const commitStyle = React.useCallback(
    (nextStyle: AssStyleSpecDTO) => {
      props.onChange(normalizeAssStyleForEditor(nextStyle))
    },
    [props],
  )

  const updateStyle = React.useCallback(
    (patch: Partial<AssStyleSpecDTO>) => {
      commitStyle({
        ...props.style,
        ...patch,
      })
    },
    [commitStyle, props.style],
  )

  const selectedFontFaceValue =
    props.style.fontPostScriptName?.trim() || resolveAssStyleFontFace(props.style)

  return (
    <div className="space-y-3">
      <EditorGroupCard title={props.title}>
        <EditorRow label={t("library.config.subtitleStyles.fontFamily")}> 
          <NativeSelect
            value={props.style.fontname}
            disabled={fontCatalogLoading}
            onChange={(event) =>
              commitStyle(
                applyFontFamilyToStyle(
                  props.style,
                  resolveFontCatalogFamily(fontCatalog, event.target.value),
                  event.target.value,
                ),
              )
            }
          >
            {fontOptions.map((family) => (
              <option key={family} value={family}>
                {family}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.fontFace")}>
          <NativeSelect
            value={selectedFontFaceValue}
            disabled={fontCatalogLoading || fontFaceOptions.length === 0}
            onChange={(event) => {
              const nextFace =
                fontFaceOptions.find(
                  (face) => (face.postScriptName?.trim() || face.name) === event.target.value,
                ) ?? fontFaceOptions[0]
              if (!nextFace) {
                return
              }
              commitStyle(
                applyFontCatalogFaceToStyle(
                  props.style,
                  selectedFontFamily?.family ?? props.style.fontname,
                  nextFace,
                ),
              )
            }}
          >
            {fontFaceOptions.map((face) => (
              <option key={face.postScriptName?.trim() || face.name} value={face.postScriptName?.trim() || face.name}>
                {face.name}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.fontSize")}> 
          <NumberInput
            value={props.style.fontsize}
            integer
            onChange={(value) => updateStyle({ fontsize: value })}
          />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.styleFlagsLabel")}> 
          <InlineTypographyButtons
            value={props.style}
            onChange={(patch) => {
              if ("bold" in patch) {
                commitStyle(toggleAssStyleBold(props.style, selectedFontFamily))
                return
              }
              if ("italic" in patch) {
                commitStyle(toggleAssStyleItalic(props.style, selectedFontFamily))
                return
              }
              updateStyle(patch)
            }}
          />
        </EditorRow>
      </EditorGroupCard>

      <EditorGroupCard title={t("library.config.subtitleStyles.sectionColors")}> 
        <EditorRow label={t("library.config.subtitleStyles.primaryColorLabel")}> 
          <AssColorCompactField value={props.style.primaryColour} onChange={(value) => updateStyle({ primaryColour: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.secondaryColorLabel")}> 
          <AssColorCompactField value={props.style.secondaryColour} onChange={(value) => updateStyle({ secondaryColour: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.outlineColor")}> 
          <AssColorCompactField value={props.style.outlineColour} onChange={(value) => updateStyle({ outlineColour: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.backColorLabel")}> 
          <AssColorCompactField value={props.style.backColour} onChange={(value) => updateStyle({ backColour: value })} />
        </EditorRow>
      </EditorGroupCard>

      <EditorGroupCard title={t("library.config.subtitleStyles.sectionRendering")}> 
        <EditorRow label={t("library.config.subtitleStyles.alignment")}> 
          <NativeSelect
            value={String(props.style.alignment)}
            onChange={(event) => updateStyle({ alignment: Number(event.target.value) })}
          >
            {alignmentOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.borderStyleLabel")}> 
          <NativeSelect
            value={String(props.style.borderStyle)}
            onChange={(event) => updateStyle({ borderStyle: Number(event.target.value) })}
          >
            {borderStyleOptions.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </NativeSelect>
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.outlineWidth")}> 
          <NumberInput value={props.style.outline} onChange={(value) => updateStyle({ outline: value })} />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.shadowLabel")}> 
          <NumberInput value={props.style.shadow} onChange={(value) => updateStyle({ shadow: value })} />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.scaleXLabel")}> 
          <NumberInput value={props.style.scaleX} onChange={(value) => updateStyle({ scaleX: value })} />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.scaleYLabel")}> 
          <NumberInput value={props.style.scaleY} onChange={(value) => updateStyle({ scaleY: value })} />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.spacingLabel")}> 
          <NumberInput value={props.style.spacing} onChange={(value) => updateStyle({ spacing: value })} />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.angleLabel")}> 
          <NumberInput value={props.style.angle} onChange={(value) => updateStyle({ angle: value })} />
        </EditorRow>
      </EditorGroupCard>

      <EditorGroupCard title={t("library.config.subtitleStyles.sectionSpacing")}> 
        <EditorRow label={t("library.config.subtitleStyles.marginLLabel")}> 
          <NumberInput integer value={props.style.marginL} onChange={(value) => updateStyle({ marginL: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.marginRLabel")}> 
          <NumberInput integer value={props.style.marginR} onChange={(value) => updateStyle({ marginR: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.marginVLabel")}> 
          <NumberInput integer value={props.style.marginV} onChange={(value) => updateStyle({ marginV: value })} />
        </EditorRow>
        <EditorRow label={t("library.config.subtitleStyles.encodingLabel")}> 
          <NumberInput integer value={props.style.encoding} onChange={(value) => updateStyle({ encoding: value })} />
        </EditorRow>
      </EditorGroupCard>
    </div>
  )
}

function InlineTypographyButtons(props: {
  value: AssStyleSpecDTO
  onChange: (patch: Partial<AssStyleSpecDTO>) => void
}) {
  const { t } = useI18n()

  const items = [
    {
      key: "bold" as const,
      label: "B",
      active: props.value.bold,
      title: t("library.config.subtitleStyles.bold"),
    },
    {
      key: "italic" as const,
      label: "I",
      active: props.value.italic,
      title: t("library.config.subtitleStyles.italic"),
    },
    {
      key: "underline" as const,
      label: "U",
      active: props.value.underline,
      title: t("library.config.subtitleStyles.underline"),
    },
    {
      key: "strikeOut" as const,
      label: "S",
      active: props.value.strikeOut,
      title: t("library.config.subtitleStyles.strikeOut"),
    },
  ]

  return (
    <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background">
      {items.map((item, index) => (
        <Button
          key={item.key}
          type="button"
          variant="ghost"
          size="compact"
          title={item.title}
          className={cn(
            "h-8 rounded-none border-0 px-3 font-semibold",
            index > 0 ? "border-l border-border/60" : "",
            item.active ? "bg-background text-foreground" : "text-muted-foreground",
          )}
          onClick={() => props.onChange({ [item.key]: !item.active })}
        >
          {item.label}
        </Button>
      ))}
    </div>
  )
}

function SubtitleStyleEmptyState(props: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  title: string
  description: string
}) {
  const Icon = props.icon

  return (
    <div className="flex min-h-0 flex-1 items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center">
      <Empty className="max-w-lg py-8">
        <EmptyHeader>
          <EmptyMedia className="flex h-14 w-14 items-center justify-center rounded-full border border-border/70 bg-background/80 text-muted-foreground">
            <Icon className="h-6 w-6" />
          </EmptyMedia>
          <EmptyTitle>{props.title}</EmptyTitle>
          <EmptyDescription>{props.description}</EmptyDescription>
        </EmptyHeader>
      </Empty>
    </div>
  )
}

function EditorGroupCard(props: {
  title?: string
  children: React.ReactNode
}) {
  return (
    <div className={cn("space-y-2 rounded-xl px-3 py-3", DASHBOARD_DIALOG_FIELD_SURFACE_CLASS)}>
      {props.title ? <div className="text-xs font-semibold tracking-[0.04em] text-foreground/85">{props.title}</div> : null}
      <div className="space-y-2">{props.children}</div>
    </div>
  )
}

function EditorRow(props: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="grid items-center gap-2 sm:grid-cols-[84px_minmax(0,1fr)]">
      <div className="text-[11px] text-muted-foreground">{props.label}</div>
      <div className="min-w-0">{props.children}</div>
    </div>
  )
}

function InfoItem(props: { label: string; value: string }) {
  return (
    <div className="grid grid-cols-[84px_minmax(0,1fr)] items-start gap-2">
      <div className="text-[11px] text-muted-foreground">{props.label}</div>
      <div className="text-xs text-foreground break-all">{props.value || "-"}</div>
    </div>
  )
}

function NativeSelect(props: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={cn(
        "flex h-8 w-full rounded-lg border border-border/70 bg-background px-2.5 text-xs outline-none transition-colors",
        "focus:border-ring focus:ring-2 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-60",
        props.className,
      )}
    />
  )
}

function NumberInput(props: {
  value: number
  integer?: boolean
  onChange: (value: number) => void
}) {
  return (
    <Input
      type="number"
      step={props.integer ? 1 : "any"}
      className="h-8 text-xs md:text-xs"
      value={Number.isFinite(props.value) ? props.value : 0}
      onChange={(event) => {
        const next = Number(event.target.value)
        if (!Number.isFinite(next)) {
          return
        }
        props.onChange(props.integer ? Math.round(next) : next)
      }}
    />
  )
}

function AssColorCompactField(props: {
  value: string
  onChange: (value: string) => void
}) {
  const parsed = parseAssColor(props.value)
  const swatchColor = parsed?.rgb ?? "#ffffff"

  return (
    <div className="flex h-8 items-center gap-2 rounded-lg border border-border/70 bg-background px-2.5">
      <div className="flex h-5 w-5 items-center justify-center rounded-full border border-border/70">
        <span className="h-full w-full rounded-full" style={{ backgroundColor: swatchColor }} />
      </div>
      <Input
        value={props.value}
        onChange={(event) => props.onChange(event.target.value)}
        className="h-7 border-0 bg-transparent px-0 font-mono text-xs shadow-none focus-visible:ring-0 md:text-xs"
      />
      <input
        type="color"
        value={swatchColor}
        onChange={(event) => props.onChange(formatAssColorWithRgb(event.target.value, props.value))}
        className="h-5 w-5 cursor-pointer rounded-full border border-border/70 bg-transparent p-0"
      />
    </div>
  )
}

function resolveStyleForRendering(
  kind: SubtitleStylePresetKind,
  item: LibraryMonoStyleDTO | LibraryBilingualStyleDTO,
): AssStyleSpecDTO {
  if (kind === "mono") {
    return (item as LibraryMonoStyleDTO).style
  }
  return (item as LibraryBilingualStyleDTO).primary.style
}

function normalizeAssStyleForEditor(style: AssStyleSpecDTO): AssStyleSpecDTO {
  const fontWeight = resolveAssStyleFontWeight(style)
  return {
    ...style,
    fontname: style.fontname?.trim() || "Arial",
    fontFace: resolveAssStyleFontFace(style),
    fontWeight,
    fontPostScriptName: style.fontPostScriptName?.trim() || "",
    bold: fontWeight >= 700,
    italic: resolveAssStyleFontItalic(style),
    fontsize: Math.max(1, Math.round(style.fontsize || 0)),
    marginL: Math.round(style.marginL || 0),
    marginR: Math.round(style.marginR || 0),
    marginV: Math.round(style.marginV || 0),
    encoding: Math.round(style.encoding || 0),
  }
}

function normalizeMonoStyleForEditor(style: LibraryMonoStyleDTO): LibraryMonoStyleDTO {
  return {
    ...style,
    style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.style)),
  }
}

function normalizeBilingualStyleForEditor(style: LibraryBilingualStyleDTO): LibraryBilingualStyleDTO {
  return {
    ...style,
    primary: {
      ...style.primary,
      style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.primary.style)),
    },
    secondary: {
      ...style.secondary,
      style: normalizeAssStyleForEditor(cloneAssStyleSpec(style.secondary.style)),
    },
    layout: {
      ...style.layout,
      gap: Number.isFinite(style.layout.gap) ? style.layout.gap : 0,
      blockAnchor: Math.round(style.layout.blockAnchor || 2),
    },
  }
}

function normalizeSnapshotBase(
  snapshot: ReturnType<typeof createMonoSnapshotFromStyle>,
  aspectRatio: string,
  basePlayResX: number,
  basePlayResY: number,
) {
  return {
    ...snapshot,
    baseAspectRatio: aspectRatio,
    basePlayResX,
    basePlayResY,
  }
}

function normalizeImportResult(result: ParseSubtitleStyleImportResult): ParseSubtitleStyleImportResult {
  const next: ParseSubtitleStyleImportResult = {
    ...result,
  }

  if (result.monoStyles) {
    next.monoStyles = result.monoStyles.map((style) => normalizeMonoStyleForEditor(cloneMonoStyle(style)))
  }

  if (result.bilingualStyle) {
    next.bilingualStyle = normalizeBilingualStyleForEditor(cloneBilingualStyle(result.bilingualStyle))
  }

  return next
}

function formatStyleFlags(style: AssStyleSpecDTO, t: (key: string) => string) {
  const flags: string[] = []
  if (style.bold) {
    flags.push("B")
  }
  if (style.italic) {
    flags.push("I")
  }
  if (style.underline) {
    flags.push("U")
  }
  if (style.strikeOut) {
    flags.push("S")
  }
  return flags.length > 0 ? flags.join(" / ") : t("library.config.subtitleStyles.styleFlagsNormal")
}

function resolveAlignmentOptions(t: (key: string) => string) {
  return [
    { value: 1, label: t("library.config.subtitleStyles.alignmentBottomLeft") },
    { value: 2, label: t("library.config.subtitleStyles.alignmentBottomCenter") },
    { value: 3, label: t("library.config.subtitleStyles.alignmentBottomRight") },
    { value: 4, label: t("library.config.subtitleStyles.alignmentMiddleLeft") },
    { value: 5, label: t("library.config.subtitleStyles.alignmentMiddleCenter") },
    { value: 6, label: t("library.config.subtitleStyles.alignmentMiddleRight") },
    { value: 7, label: t("library.config.subtitleStyles.alignmentTopLeft") },
    { value: 8, label: t("library.config.subtitleStyles.alignmentTopCenter") },
    { value: 9, label: t("library.config.subtitleStyles.alignmentTopRight") },
  ]
}

function resolveBorderStyleOptions(t: (key: string) => string) {
  return [
    { value: 1, label: t("library.config.subtitleStyles.borderStyleOutline") },
    { value: 3, label: t("library.config.subtitleStyles.borderStyleBox") },
  ]
}

function parseAssColor(value: string) {
  const normalized = value.trim().replace(/^&?H/i, "").replace(/[^0-9a-f]/gi, "").toUpperCase()
  if (normalized.length !== 6 && normalized.length !== 8) {
    return null
  }

  const hex = normalized.length === 6 ? `00${normalized}` : normalized
  const alpha = hex.slice(0, 2)
  const blue = hex.slice(2, 4)
  const green = hex.slice(4, 6)
  const red = hex.slice(6, 8)

  return {
    alpha,
    rgb: `#${red}${green}${blue}`.toLowerCase(),
  }
}

function formatAssColorWithRgb(rgb: string, currentValue: string) {
  const parsed = parseAssColor(currentValue)
  const alpha = parsed?.alpha ?? "00"
  const normalized = rgb.trim().replace(/^#/, "").replace(/[^0-9a-f]/gi, "").toUpperCase()
  if (normalized.length !== 6) {
    return currentValue
  }

  const red = normalized.slice(0, 2)
  const green = normalized.slice(2, 4)
  const blue = normalized.slice(4, 6)
  return `&H${alpha}${blue}${green}${red}`
}
