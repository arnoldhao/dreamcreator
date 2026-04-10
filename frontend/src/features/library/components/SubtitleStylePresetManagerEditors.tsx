import * as React from "react"

import { useFontCatalog } from "@/hooks/useFontCatalog"
import {
  applyFontCatalogFaceToStyle,
  applyFontFamilyToStyle,
  resolveAssStyleFontFace,
  resolveFontCatalogFaces,
  resolveFontCatalogFamily,
  toggleAssStyleBold,
  toggleAssStyleItalic,
} from "@/shared/fonts/fontCatalog"
import { useI18n } from "@/shared/i18n"
import type {
  AssStyleSpecDTO,
  LibraryBilingualStyleDTO,
  LibraryMonoStyleDTO,
} from "@/shared/contracts/library"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui/tabs"

import { createMonoSnapshotFromStyle } from "../utils/subtitleStylePresets"

import {
  AssColorCompactField,
  EditorGroupCard,
  EditorRow,
  InlineTypographyButtons,
  NativeSelect,
  NumberInput,
} from "./SubtitleStylePresetManagerControls"
import {
  normalizeAssStyleForEditor,
  resolveAlignmentOptions,
  resolveBorderStyleOptions,
} from "./SubtitleStylePresetManagerShared"

export function MonoStyleEditor(props: {
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

export function BilingualStyleEditor(props: {
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
