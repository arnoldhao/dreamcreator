import { describe, expect, test } from "bun:test"

import type { LibraryFileDTO } from "@/shared/contracts/library"

import { resolveLibraryWorkspaceStateFromOpenTarget } from "../components/library-workspace-utils"
import { DEFAULT_WORKSPACE_QA_CHECK_SETTINGS } from "../model/workspaceQa"
import type { LibraryWorkspacePersistedState } from "../model/workspaceStore"
import { buildWorkspaceTargetFromLibraryFile } from "../utils/workspaceTargets"

function buildLibraryFile(id: string, kind: string, rootFileId?: string): LibraryFileDTO {
  const extension = kind === "subtitle" ? "srt" : "mp4"
  return {
    id,
    libraryId: "lib-1",
    kind,
    name: `${id}.${extension}`,
    displayLabel: id,
    storage: {
      mode: "local_path",
      localPath: `/tmp/${id}.${extension}`,
    },
    origin: {
      kind: "import",
    },
    lineage: {
      rootFileId: rootFileId ?? id,
    },
    state: {
      status: "ready",
      deleted: false,
      archived: false,
    },
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
  }
}

function buildBaseWorkspaceState(): LibraryWorkspacePersistedState {
  return {
    libraryId: "lib-1",
    activeEditor: "video",
    activeVideoFileId: "video-1",
    activeSubtitleFileId: "subtitle-1",
    displayMode: "mono",
    comparisonSubtitleFileId: "subtitle-2",
    guidelineProfileId: "netflix",
    qaCheckSettings: DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
  }
}

describe("buildWorkspaceTargetFromLibraryFile", () => {
  test("links subtitle and video files from the same resource family", () => {
    const files = [
      buildLibraryFile("video-1", "video", "root-1"),
      buildLibraryFile("subtitle-1", "subtitle", "root-1"),
    ]

    const target = buildWorkspaceTargetFromLibraryFile(files[1], files)

    expect(target?.openMode).toBe("subtitle")
    expect(target?.fileId).toBe("subtitle-1")
    expect(target?.videoAssetId).toBe("video-1")
  })
})

describe("resolveLibraryWorkspaceStateFromOpenTarget", () => {
  test("keeps explicit subtitle open intent ahead of restored workspace state", () => {
    const files = [
      buildLibraryFile("video-1", "video", "root-1"),
      buildLibraryFile("subtitle-1", "subtitle", "root-1"),
      buildLibraryFile("video-2", "video", "root-2"),
      buildLibraryFile("subtitle-2", "subtitle", "root-2"),
    ]
    const target = buildWorkspaceTargetFromLibraryFile(files[3], files)

    const nextState = resolveLibraryWorkspaceStateFromOpenTarget(
      target!,
      buildBaseWorkspaceState(),
      {
        libraryId: "lib-1",
        libraryFiles: files,
        videoFiles: files.filter((file) => file.kind !== "subtitle"),
        subtitleFiles: files.filter((file) => file.kind === "subtitle"),
      },
    )

    expect(nextState.activeEditor).toBe("subtitle")
    expect(nextState.activeSubtitleFileId).toBe("subtitle-2")
    expect(nextState.activeVideoFileId).toBe("video-2")
    expect(nextState.comparisonSubtitleFileId).toBe("subtitle-1")
  })

  test("clears the previous subtitle track when the opened video has no linked subtitle", () => {
    const files = [
      buildLibraryFile("video-1", "video", "root-1"),
      buildLibraryFile("subtitle-1", "subtitle", "root-1"),
      buildLibraryFile("video-3", "video", "root-3"),
    ]
    const target = buildWorkspaceTargetFromLibraryFile(files[2], files)

    const nextState = resolveLibraryWorkspaceStateFromOpenTarget(
      target!,
      buildBaseWorkspaceState(),
      {
        libraryId: "lib-1",
        libraryFiles: files,
        videoFiles: files.filter((file) => file.kind !== "subtitle"),
        subtitleFiles: files.filter((file) => file.kind === "subtitle"),
      },
    )

    expect(nextState.activeEditor).toBe("video")
    expect(nextState.activeVideoFileId).toBe("video-3")
    expect(nextState.activeSubtitleFileId).toBe("")
  })
})
