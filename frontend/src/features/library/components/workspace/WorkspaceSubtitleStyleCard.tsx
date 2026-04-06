import * as React from "react"
import { ChevronDown } from "lucide-react"

import { useFontFamilies } from "@/hooks/useFontFamilies"
import { cn } from "@/lib/utils"
import { useI18n } from "@/shared/i18n"
import type { AssStyleSpecDTO, LibraryBilingualStyleDTO, LibraryMonoStyleDTO } from "@/shared/contracts/library"
import { Button } from "@/shared/ui/button"
import { DASHBOARD_DIALOG_FIELD_SURFACE_CLASS } from "@/shared/ui/dashboard-dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { Input } from "@/shared/ui/input"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { ToggleGroup, ToggleGroupItem } from "@/shared/ui/toggle-group"

import { createMonoSnapshotFromStyle } from "../../utils/subtitleStylePresets"
import type { WorkspaceDisplayMode, WorkspaceSelectOption } from "./types"

type WorkspaceSubtitleStyleCardProps = {
  displayMode: WorkspaceDisplayMode
  monoStyle: LibraryMonoStyleDTO | null
  lingualStyle: LibraryBilingualStyleDTO | null
  monoStyles: LibraryMonoStyleDTO[]
  monoStyleOptions: WorkspaceSelectOption[]
  lingualStyleOptions: WorkspaceSelectOption[]
  onMonoStyleChange: (style: LibraryMonoStyleDTO) => void
  onLingualStyleChange: (style: LibraryBilingualStyleDTO) => void
  onApplyTemplate: (kind: "mono" | "lingual", styleId: string) => void
  onSaveAs: (kind: "mono" | "lingual", name: string) => void
}

export function WorkspaceSubtitleStyleCard({
  displayMode,
  monoStyle,
  lingualStyle,
  monoStyles,
  monoStyleOptions,
  lingualStyleOptions,
  onMonoStyleChange,
  onLingualStyleChange,
  onApplyTemplate,
  onSaveAs,
}: WorkspaceSubtitleStyleCardProps) {
  const { t } = useI18n()
  const [saveAsName, setSaveAsName] = React.useState("")
  const activeKind = displayMode === "dual" ? "lingual" : "mono"
  const activeOptions = activeKind === "mono" ? monoStyleOptions : lingualStyleOptions

  const handleApplyTemplate = React.useCallback(
    (kind: "mono" | "lingual", styleId: string) => {
      if (!styleId.trim()) {
        return
      }
      const confirmed = window.confirm(
        t("library.config.subtitleStyles.applyTemplateConfirm"),
      )
      if (!confirmed) {
        return
      }
      onApplyTemplate(kind, styleId)
    },
    [onApplyTemplate, t],
  )

  return (
    <div className="flex h-full min-h-0 flex-col overflow-hidden">
      <div className="min-h-0 flex-1 overflow-hidden border-y border-border/60">
        <div className="h-full min-h-0 overflow-y-auto px-3 py-3">
          <div className="space-y-3">
            {displayMode === "dual" && lingualStyle ? (
              <WorkspaceBilingualStyleEditor
                draft={lingualStyle}
                monoStyles={monoStyles}
                onChange={onLingualStyleChange}
              />
            ) : null}
            {displayMode !== "dual" && monoStyle ? (
              <WorkspaceAssStyleEditor
                title={t("library.config.subtitleStyles.monoStyleSectionTitle")}
                style={monoStyle.style}
                onChange={(nextStyle) => onMonoStyleChange({ ...monoStyle, style: nextStyle })}
              />
            ) : null}
          </div>
        </div>
      </div>

      <div className="flex shrink-0 items-center gap-2 p-3">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="compact" className="flex-1 justify-center gap-1.5">
              <span>{t("library.config.subtitleStyles.switchStyle")}</span>
              <ChevronDown className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-64">
            {activeOptions.length === 0 ? (
              <DropdownMenuItem disabled>{t("library.config.subtitleStyles.emptyStateTitle")}</DropdownMenuItem>
            ) : (
              activeOptions.map((option) => (
                <DropdownMenuItem key={`${activeKind}-${option.value}`} onClick={() => handleApplyTemplate(activeKind, option.value)}>
                  {option.label}
                </DropdownMenuItem>
              ))
            )}
          </DropdownMenuContent>
        </DropdownMenu>

        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button type="button" variant="outline" size="compact" className="flex-1 justify-center gap-1.5">
              <span>{t("library.config.subtitleStyles.saveAs")}</span>
              <ChevronDown className="h-3.5 w-3.5" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end" className="w-72 p-2">
            <div className="space-y-2">
              <Input
                value={saveAsName}
                onChange={(event) => setSaveAsName(event.target.value)}
                placeholder={t(activeKind === "mono"
                    ? "library.config.subtitleStyles.monoStyleNamePlaceholder"
                    : "library.config.subtitleStyles.bilingualStyleNamePlaceholder")}
                className="h-8 text-xs md:text-xs"
              />
              <Button
                type="button"
                variant="outline"
                size="compact"
                className="w-full justify-center"
                onClick={() => {
                  const nextName = saveAsName.trim()
                  if (!nextName) {
                    return
                  }
                  onSaveAs(activeKind, nextName)
                  setSaveAsName("")
                }}
                disabled={!saveAsName.trim()}
              >
                {t("library.config.subtitleStyles.saveAsStyle")}
              </Button>
            </div>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
  )
}

function WorkspaceBilingualStyleEditor(props: {
  draft: LibraryBilingualStyleDTO
  monoStyles: LibraryMonoStyleDTO[]
  onChange: (next: LibraryBilingualStyleDTO) => void
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
              props.onChange({
                ...props.draft,
                layout: { ...props.draft.layout, gap: value },
              })
            }
          />
        </EditorRow>

        <EditorRow label={t("library.config.subtitleStyles.blockAnchorLabel")}>
          <NativeSelect
            value={String(props.draft.layout.blockAnchor)}
            onChange={(event) =>
              props.onChange({
                ...props.draft,
                layout: { ...props.draft.layout, blockAnchor: Number(event.target.value) },
              })
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
              props.onChange({
                ...props.draft,
                primary: createMonoSnapshotFromStyle(source),
              })
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
              props.onChange({
                ...props.draft,
                secondary: createMonoSnapshotFromStyle(source),
              })
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
          <WorkspaceAssStyleEditor
            title={t("library.config.subtitleStyles.primaryStyleTitle")}
            style={props.draft.primary.style}
            onChange={(nextStyle) =>
              props.onChange({
                ...props.draft,
                primary: { ...props.draft.primary, style: nextStyle },
              })
            }
          />
        </TabsContent>

        <TabsContent value="secondary" className="mt-0">
          <WorkspaceAssStyleEditor
            title={t("library.config.subtitleStyles.secondaryStyleTitle")}
            style={props.draft.secondary.style}
            onChange={(nextStyle) =>
              props.onChange({
                ...props.draft,
                secondary: { ...props.draft.secondary, style: nextStyle },
              })
            }
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function WorkspaceAssStyleEditor(props: {
  title: string
  style: AssStyleSpecDTO
  onChange: (value: AssStyleSpecDTO) => void
}) {
  const { t } = useI18n()
  const { data: fontFamilies = [], isLoading: fontFamiliesLoading } = useFontFamilies()
  const alignmentOptions = React.useMemo(() => resolveAlignmentOptions(t), [t])
  const borderStyleOptions = React.useMemo(() => resolveBorderStyleOptions(t), [t])

  const fontOptions = React.useMemo(() => {
    const options = new Set(fontFamilies.map((family) => family.trim()).filter(Boolean))
    if (props.style.fontname.trim()) {
      options.add(props.style.fontname.trim())
    }
    return [...options].sort((left, right) => left.localeCompare(right))
  }, [fontFamilies, props.style.fontname])

  const updateStyle = React.useCallback(
    (patch: Partial<AssStyleSpecDTO>) => {
      props.onChange(
        normalizeAssStyleForEditor({
          ...props.style,
          ...patch,
        }),
      )
    },
    [props],
  )

  return (
    <div className="space-y-3">
      <EditorGroupCard title={props.title}>
        <EditorRow label={t("library.config.subtitleStyles.fontFamily")}>
          <NativeSelect
            value={props.style.fontname}
            disabled={fontFamiliesLoading}
            onChange={(event) => updateStyle({ fontname: event.target.value })}
          >
            {fontOptions.map((family) => (
              <option key={family} value={family}>
                {family}
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
            onChange={(patch) => updateStyle(patch)}
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
    <ToggleGroup
      type="multiple"
      value={items.filter((item) => item.active).map((item) => item.key)}
      onValueChange={(nextValues) => {
        const selected = new Set(nextValues)
        items.forEach((item) => {
          const active = selected.has(item.key)
          if (active !== item.active) {
            props.onChange({ [item.key]: active })
          }
        })
      }}
      variant="outline"
      size="sm"
      className="rounded-lg border border-border/70 bg-background p-0.5"
    >
      {items.map((item) => (
        <ToggleGroupItem
          key={item.key}
          value={item.key}
          title={item.title}
          aria-label={item.title}
          className={cn("h-7 min-w-8 rounded-md px-2 font-semibold", item.active ? "text-foreground" : "text-muted-foreground")}
        >
          {item.label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
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

function normalizeAssStyleForEditor(style: AssStyleSpecDTO): AssStyleSpecDTO {
  return {
    ...style,
    fontsize: Math.max(1, Math.round(style.fontsize || 0)),
    marginL: Math.round(style.marginL || 0),
    marginR: Math.round(style.marginR || 0),
    marginV: Math.round(style.marginV || 0),
    encoding: Math.round(style.encoding || 0),
  }
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
