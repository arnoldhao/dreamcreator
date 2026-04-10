import * as React from "react"
import { Captions, Check, ChevronDown, FileUp, Plus } from "lucide-react"

import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryModuleConfigDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleStyleFontDTO,
} from "@/shared/contracts/library"
import { useI18n } from "@/shared/i18n"
import { cn } from "@/lib/utils"
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
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs"

import { buildSubtitleStyleNamePreviewStyle } from "../utils/subtitleStyleNamePreview"
import {
  SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS,
  type SubtitleStylePresetSelection,
} from "../utils/subtitleStylePresets"
import { SubtitleStylePresetPreview } from "./SubtitleStylePresetPreview"
import {
  EditorGroupCard,
  EditorRow,
  InfoItem,
  NativeSelect,
  SubtitleStyleEmptyState,
} from "./SubtitleStylePresetManagerControls"
import type {
  CreatePaneMode,
  CreateStyleDraft,
  ImportDraftState,
} from "./SubtitleStylePresetManagerShared"
import {
  formatStyleFlags,
  isDefaultBilingualStyle,
  isDefaultMonoStyle,
  resolveSelectionItem,
  resolveStyleForRendering,
} from "./SubtitleStylePresetManagerShared"

export function PreviewPane(props: {
  draftKind: "mono" | "bilingual" | null
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

export function AllStyleCardsPane(props: {
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

export function CreateStylePane(props: {
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
                      kind: value as CreateStyleDraft["kind"],
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

export function PresetCompositeHeader(props: {
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
