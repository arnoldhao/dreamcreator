import type { LibraryFileDTO } from "@/shared/contracts/library"

import type { LibraryWorkspaceTarget } from "../model/types"

function normalizeLibraryKind(value?: string) {
  return value?.trim().toLowerCase() ?? ""
}

function resolveFileFamilyRootId(file: LibraryFileDTO) {
  return file.lineage.rootFileId?.trim() || file.id
}

export function canOpenLibraryWorkspaceFile(file?: LibraryFileDTO | null) {
  if (!file || file.state.deleted) {
    return false
  }
  const kind = normalizeLibraryKind(file.kind)
  return kind === "video" || kind === "audio" || kind === "subtitle" || kind === "transcode"
}

export function buildWorkspaceTargetFromLibraryFile(
  file: LibraryFileDTO,
  libraryFiles: LibraryFileDTO[],
): LibraryWorkspaceTarget | null {
  if (!canOpenLibraryWorkspaceFile(file)) {
    return null
  }
  const rootFileId = resolveFileFamilyRootId(file)
  const siblings = libraryFiles.filter((candidate) => {
    if (candidate.libraryId !== file.libraryId || candidate.state.deleted) {
      return false
    }
    return resolveFileFamilyRootId(candidate) === rootFileId
  })
  const linkedVideo = siblings.find((candidate) => {
    const kind = normalizeLibraryKind(candidate.kind)
    return kind === "video" || kind === "audio" || kind === "transcode"
  })
  const linkedSubtitle = siblings.find((candidate) => normalizeLibraryKind(candidate.kind) === "subtitle")
  const kind = normalizeLibraryKind(file.kind)
  return {
    libraryId: file.libraryId,
    fileId: file.id,
    name: file.name,
    fileType: file.kind,
    path: file.storage.localPath,
    openMode: kind === "subtitle" ? "subtitle" : "video",
    videoAssetId: kind === "subtitle" ? linkedVideo?.id ?? "" : undefined,
    videoPath: kind === "subtitle" ? linkedVideo?.storage.localPath ?? "" : undefined,
    videoName: kind === "subtitle" ? linkedVideo?.name ?? "" : undefined,
    subtitleAssetId: kind === "subtitle" ? undefined : linkedSubtitle?.id ?? "",
    subtitlePath: kind === "subtitle" ? undefined : linkedSubtitle?.storage.localPath ?? "",
    subtitleName: kind === "subtitle" ? undefined : linkedSubtitle?.name ?? "",
  }
}
