import type {
  AssStyleSpecDTO,
  LibraryBilingualLayoutDTO,
  LibraryBilingualStyleDTO,
  LibraryModuleConfigDTO,
  LibraryMonoStyleDTO,
  LibraryMonoStyleSnapshotDTO,
} from "@/shared/contracts/library"
import { t } from "@/shared/i18n"

export type SubtitleStylePresetKind = "mono" | "bilingual"

export type SubtitleStylePresetSelection =
  | {
      kind: SubtitleStylePresetKind
      id: string
    }
  | null

export const SUBTITLE_STYLE_ASPECT_RATIO_OPTIONS = [
  { value: "16:9", label: "16:9 · 1920×1080" },
  { value: "16:10", label: "16:10 · 1920×1200" },
  { value: "4:3", label: "4:3 · 1440×1080" },
  { value: "1:1", label: "1:1 · 1080×1080" },
  { value: "9:16", label: "9:16 · 1080×1920" },
] as const

export const BILINGUAL_BLOCK_ANCHOR_OPTIONS = [
  { value: 1, label: "Bottom Left" },
  { value: 2, label: "Bottom Center" },
  { value: 3, label: "Bottom Right" },
  { value: 4, label: "Middle Left" },
  { value: 5, label: "Middle Center" },
  { value: 6, label: "Middle Right" },
  { value: 7, label: "Top Left" },
  { value: 8, label: "Top Center" },
  { value: 9, label: "Top Right" },
] as const

export function resolveMonoStyles(config?: LibraryModuleConfigDTO) {
  return config?.subtitleStyles?.monoStyles ?? []
}

export function resolveBilingualStyles(config?: LibraryModuleConfigDTO) {
  return config?.subtitleStyles?.bilingualStyles ?? []
}

export function createEmptyAssStyleSpec(): AssStyleSpecDTO {
  return {
    fontname: "Arial",
    fontsize: 48,
    primaryColour: "&H00FFFFFF",
    secondaryColour: "&H00FFFFFF",
    outlineColour: "&H00111111",
    backColour: "&HFF111111",
    bold: false,
    italic: false,
    underline: false,
    strikeOut: false,
    scaleX: 100,
    scaleY: 100,
    spacing: 0,
    angle: 0,
    borderStyle: 1,
    outline: 2,
    shadow: 0.8,
    alignment: 2,
    marginL: 72,
    marginR: 72,
    marginV: 56,
    encoding: 1,
  }
}

export function createEmptyMonoStyle(): LibraryMonoStyleDTO {
  return {
    id: createSubtitleStylePresetID("mono"),
    name: t("library.config.subtitleStyles.newMonoStyle"),
    builtIn: false,
    basePlayResX: 1920,
    basePlayResY: 1080,
    baseAspectRatio: "16:9",
    style: createEmptyAssStyleSpec(),
  }
}

export function createMonoSnapshotFromStyle(style: LibraryMonoStyleDTO): LibraryMonoStyleSnapshotDTO {
  return {
    sourceMonoStyleID: style.id,
    sourceMonoStyleName: style.name,
    name: style.name,
    basePlayResX: style.basePlayResX,
    basePlayResY: style.basePlayResY,
    baseAspectRatio: style.baseAspectRatio,
    style: cloneAssStyleSpec(style.style),
  }
}

export function createEmptyBilingualLayout(): LibraryBilingualLayoutDTO {
  return {
    gap: 24,
    blockAnchor: 2,
  }
}

export function createEmptyBilingualStyle(monoStyles: LibraryMonoStyleDTO[] = []): LibraryBilingualStyleDTO {
  const primarySource = monoStyles[0] ?? createEmptyMonoStyle()
  const secondarySource = monoStyles[1] ?? monoStyles[0] ?? createEmptyMonoStyle()
  return {
    id: createSubtitleStylePresetID("bilingual"),
    name: t("library.config.subtitleStyles.newBilingualStyle"),
    builtIn: false,
    basePlayResX: primarySource.basePlayResX,
    basePlayResY: primarySource.basePlayResY,
    baseAspectRatio: primarySource.baseAspectRatio,
    primary: createMonoSnapshotFromStyle(primarySource),
    secondary: createMonoSnapshotFromStyle(secondarySource),
    layout: createEmptyBilingualLayout(),
  }
}

export function cloneAssStyleSpec(style: AssStyleSpecDTO): AssStyleSpecDTO {
  return { ...style }
}

export function cloneMonoStyle(style: LibraryMonoStyleDTO): LibraryMonoStyleDTO {
  return {
    ...style,
    style: cloneAssStyleSpec(style.style),
  }
}

export function cloneMonoStyleSnapshot(style: LibraryMonoStyleSnapshotDTO): LibraryMonoStyleSnapshotDTO {
  return {
    ...style,
    style: cloneAssStyleSpec(style.style),
  }
}

export function cloneBilingualStyle(style: LibraryBilingualStyleDTO): LibraryBilingualStyleDTO {
  return {
    ...style,
    primary: cloneMonoStyleSnapshot(style.primary),
    secondary: cloneMonoStyleSnapshot(style.secondary),
    layout: { ...style.layout },
  }
}

export function applyAspectRatioBaseResolution(aspectRatio: string) {
  switch (aspectRatio) {
    case "16:10":
      return { basePlayResX: 1920, basePlayResY: 1200 }
    case "4:3":
      return { basePlayResX: 1440, basePlayResY: 1080 }
    case "1:1":
      return { basePlayResX: 1080, basePlayResY: 1080 }
    case "9:16":
      return { basePlayResX: 1080, basePlayResY: 1920 }
    default:
      return { basePlayResX: 1920, basePlayResY: 1080 }
  }
}

export function createSubtitleStylePresetID(prefix: SubtitleStylePresetKind) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`
}
