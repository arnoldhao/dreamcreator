import type { AssStyleSpecDTO } from "@/shared/contracts/library"

export type FontCatalogFace = {
  name: string
  fullName?: string
  postScriptName?: string
  weight?: number
  italic?: boolean
}

export type FontCatalogFamily = {
  family: string
  faces: FontCatalogFace[]
}

type FontStyleIdentity = Pick<
  AssStyleSpecDTO,
  "fontname" | "fontFace" | "fontWeight" | "fontPostScriptName"
> &
  Partial<Pick<AssStyleSpecDTO, "bold" | "italic">>

export function normalizeFontCatalog(input: unknown): FontCatalogFamily[] {
  if (!Array.isArray(input)) {
    return []
  }
  return input
    .map((item) => normalizeFontCatalogFamily(item))
    .filter((item): item is FontCatalogFamily => item !== null)
    .sort((left, right) => left.family.localeCompare(right.family))
}

export function normalizeFontCatalogFamily(input: unknown): FontCatalogFamily | null {
  if (!input || typeof input !== "object") {
    return null
  }
  const record = input as Record<string, unknown>
  const family = typeof record.family === "string" ? record.family.trim() : ""
  if (!family) {
    return null
  }
  const faces = Array.isArray(record.faces)
    ? record.faces
        .map((item) => normalizeFontCatalogFace(item))
        .filter((item): item is FontCatalogFace => item !== null)
    : []
  return {
    family,
    faces: faces.length > 0 ? faces : [buildFallbackFontFace({ fontname: family })],
  }
}

export function normalizeFontCatalogFace(input: unknown): FontCatalogFace | null {
  if (!input || typeof input !== "object") {
    return null
  }
  const record = input as Record<string, unknown>
  const name = typeof record.name === "string" ? record.name.trim() : ""
  if (!name) {
    return null
  }
  const fullName = typeof record.fullName === "string" ? record.fullName.trim() : ""
  const postScriptName =
    typeof record.postScriptName === "string" ? record.postScriptName.trim() : ""
  const weight = Number.isFinite(record.weight) ? Number(record.weight) : deriveFontWeightFromLabel(name)
  const italic =
    typeof record.italic === "boolean" ? record.italic : deriveFontItalicFromLabel(name)
  return {
    name,
    fullName: fullName || undefined,
    postScriptName: postScriptName || undefined,
    weight,
    italic,
  }
}

export function resolveAssStyleFontWeight(style: AssStyleSpecDTO | null | undefined): number {
  if (style?.fontWeight && Number.isFinite(style.fontWeight) && style.fontWeight > 0) {
    return style.fontWeight
  }
  if (style?.fontFace?.trim()) {
    return deriveFontWeightFromLabel(style.fontFace)
  }
  return style?.bold ? 700 : 400
}

export function resolveAssStyleFontItalic(style: AssStyleSpecDTO | null | undefined): boolean {
  if (!style) {
    return false
  }
  if (style.italic) {
    return true
  }
  return deriveFontItalicFromLabel(style.fontFace)
}

export function resolveAssStyleFontFace(style: AssStyleSpecDTO | null | undefined): string {
  const trimmed = style?.fontFace?.trim()
  if (trimmed) {
    return trimmed
  }
  const weight = resolveAssStyleFontWeight(style)
  if (resolveAssStyleFontItalic(style)) {
    return weight >= 700 ? "Bold Italic" : "Italic"
  }
  return weight >= 700 ? "Bold" : "Regular"
}

export function resolveFontCatalogFamily(
  catalog: FontCatalogFamily[],
  familyName: string | null | undefined,
): FontCatalogFamily | null {
  const trimmed = familyName?.trim() ?? ""
  if (!trimmed) {
    return null
  }
  const normalized = normalizeFontCatalogKey(trimmed)
  for (const family of catalog) {
    if (normalizeFontCatalogKey(family.family) === normalized) {
      return family
    }
  }
  return null
}

export function resolveFontCatalogFaces(
  catalog: FontCatalogFamily[],
  familyName: string | null | undefined,
  style?: AssStyleSpecDTO | null,
): FontCatalogFace[] {
  const family = resolveFontCatalogFamily(catalog, familyName)
  if (!family) {
    return [buildFallbackFontFace(style ?? { fontname: familyName?.trim() || "Arial" })]
  }
  if (family.faces.length > 0) {
    return family.faces
  }
  return [buildFallbackFontFace(style ?? { fontname: family.family })]
}

export function pickFontCatalogFace(
  family: FontCatalogFamily | null,
  style: AssStyleSpecDTO | null | undefined,
): FontCatalogFace | null {
  const faces = family?.faces ?? []
  if (faces.length === 0) {
    return null
  }
  const postScriptName = style?.fontPostScriptName?.trim().toLowerCase() ?? ""
  if (postScriptName) {
    const match = faces.find(
      (face) => face.postScriptName?.trim().toLowerCase() === postScriptName,
    )
    if (match) {
      return match
    }
  }
  const faceName = resolveAssStyleFontFace(style).trim().toLowerCase()
  if (faceName) {
    const match = faces.find((face) => face.name.trim().toLowerCase() === faceName)
    if (match) {
      return match
    }
  }
  const weight = resolveAssStyleFontWeight(style)
  const italic = resolveAssStyleFontItalic(style)
  const weightedMatch = faces.find(
    (face) => (face.weight ?? deriveFontWeightFromLabel(face.name)) === weight && Boolean(face.italic) === italic,
  )
  if (weightedMatch) {
    return weightedMatch
  }
  const regularMatch =
    faces.find((face) => face.name.trim().toLowerCase() === "regular") ??
    faces.find((face) => (face.weight ?? deriveFontWeightFromLabel(face.name)) === 400 && !face.italic)
  return regularMatch ?? faces[0]
}

export function resolveMatchingFontCatalogFace(
  family: FontCatalogFamily | null,
  weight: number,
  italic: boolean,
): FontCatalogFace | null {
  const faces = family?.faces ?? []
  if (faces.length === 0) {
    return null
  }
  const exact = faces.find(
    (face) => (face.weight ?? deriveFontWeightFromLabel(face.name)) === weight && Boolean(face.italic) === italic,
  )
  if (exact) {
    return exact
  }
  const sameWeight = faces.find((face) => (face.weight ?? deriveFontWeightFromLabel(face.name)) === weight)
  if (sameWeight) {
    return sameWeight
  }
  const regular =
    faces.find((face) => face.name.trim().toLowerCase() === "regular") ??
    faces.find((face) => (face.weight ?? deriveFontWeightFromLabel(face.name)) === 400 && !face.italic)
  return regular ?? faces[0]
}

export function buildFallbackFontFace(style: AssStyleSpecDTO | FontStyleIdentity): FontCatalogFace {
  const name = resolveAssStyleFontFace(style as AssStyleSpecDTO)
  const weight = resolveAssStyleFontWeight(style as AssStyleSpecDTO)
  return {
    name,
    postScriptName: style.fontPostScriptName?.trim() || undefined,
    weight,
    italic: resolveAssStyleFontItalic(style as AssStyleSpecDTO),
  }
}

export function applyFontCatalogFaceToStyle(
  style: AssStyleSpecDTO,
  familyName: string,
  face: FontCatalogFace,
): AssStyleSpecDTO {
  const weight = face.weight ?? deriveFontWeightFromLabel(face.name)
  const italic = typeof face.italic === "boolean" ? face.italic : deriveFontItalicFromLabel(face.name)
  return {
    ...style,
    fontname: familyName,
    fontFace: face.name,
    fontWeight: weight,
    fontPostScriptName: face.postScriptName?.trim() || "",
    bold: weight >= 700,
    italic,
  }
}

export function applyFontFamilyToStyle(
  style: AssStyleSpecDTO,
  family: FontCatalogFamily | null,
  familyName: string,
): AssStyleSpecDTO {
  if (!family) {
    return {
      ...style,
      fontname: familyName,
      fontPostScriptName: "",
      fontFace: resolveAssStyleFontFace(style),
      fontWeight: resolveAssStyleFontWeight(style),
    }
  }
  const selectedFace = pickFontCatalogFace(family, style) ?? buildFallbackFontFace(style)
  return applyFontCatalogFaceToStyle(style, family.family, selectedFace)
}

export function applyFontWeightToStyle(
  style: AssStyleSpecDTO,
  family: FontCatalogFamily | null,
  weight: number,
  italic: boolean,
): AssStyleSpecDTO {
  const nextFace = resolveMatchingFontCatalogFace(family, weight, italic)
  if (family && nextFace) {
    return applyFontCatalogFaceToStyle(style, family.family, nextFace)
  }
  return {
    ...style,
    fontWeight: weight,
    fontFace: italic ? (weight >= 700 ? "Bold Italic" : "Italic") : weight >= 700 ? "Bold" : "Regular",
    fontPostScriptName: "",
    bold: weight >= 700,
    italic,
  }
}

export function toggleAssStyleBold(style: AssStyleSpecDTO, family: FontCatalogFamily | null): AssStyleSpecDTO {
  const nextBold = !style.bold
  const italic = resolveAssStyleFontItalic(style)
  return applyFontWeightToStyle(style, family, nextBold ? 700 : 400, italic)
}

export function toggleAssStyleItalic(style: AssStyleSpecDTO, family: FontCatalogFamily | null): AssStyleSpecDTO {
  const nextItalic = !resolveAssStyleFontItalic(style)
  const weight = resolveAssStyleFontWeight(style)
  return applyFontWeightToStyle(style, family, weight, nextItalic)
}

export function deriveFontWeightFromLabel(value: string | null | undefined): number {
  const normalized = (value ?? "").trim().toLowerCase()
  switch (true) {
    case normalized.includes("thin") || normalized.includes("hairline"):
      return 100
    case normalized.includes("ultralight") || normalized.includes("extra light") || normalized.includes("extralight"):
      return 200
    case normalized.includes("light"):
      return 300
    case normalized.includes("semibold") || normalized.includes("demibold") || normalized.includes("demi bold") || normalized.includes("demi"):
      return 600
    case normalized.includes("extrabold") || normalized.includes("extra bold") || normalized.includes("ultrabold") || normalized.includes("ultra bold"):
      return 800
    case normalized.includes("black") || normalized.includes("heavy"):
      return 900
    case normalized.includes("medium"):
      return 500
    case normalized.includes("bold"):
      return 700
    default:
      return 400
  }
}

export function deriveFontItalicFromLabel(value: string | null | undefined): boolean {
  const normalized = (value ?? "").trim().toLowerCase()
  return normalized.includes("italic") || normalized.includes("oblique")
}

export function normalizeFontCatalogKey(value: string | null | undefined): string {
  return (value ?? "")
    .trim()
    .replace(/^['"]+|['"]+$/g, "")
    .replace(/\s+/g, " ")
    .toLowerCase()
}
