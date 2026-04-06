import { create } from "zustand"
import { persist } from "zustand/middleware"

import type { LibraryBilingualStyleDTO, LibraryMonoStyleDTO } from "@/shared/contracts/library"
import type { LibraryWorkspaceTarget } from "./types"
import {
  DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
  normalizeWorkspaceQaCheckSettings,
  type WorkspaceQaCheckId,
  type WorkspaceQaCheckSettings,
} from "./workspaceQa"

export type LibraryWorkspaceEditor = "video" | "subtitle"
export type LibraryWorkspaceDisplayMode = "single" | "dual"
export type LibraryWorkspaceGuidelineProfileId = "netflix" | "bbc" | "ade"
export type LibraryWorkspacePersistedState = {
  libraryId: string
  activeEditor: LibraryWorkspaceEditor
  activeVideoFileId: string
  activeSubtitleFileId: string
  displayMode: LibraryWorkspaceDisplayMode
  comparisonSubtitleFileId: string
  guidelineProfileId: LibraryWorkspaceGuidelineProfileId
  qaCheckSettings: WorkspaceQaCheckSettings
  subtitleMonoStyle?: LibraryMonoStyleDTO
  subtitleLingualStyle?: LibraryBilingualStyleDTO
  subtitleStyleSidebarOpen?: boolean
}

type LibraryWorkspaceState = {
  libraryId: string
  activeEditor: LibraryWorkspaceEditor
  activeVideoFileId: string
  activeSubtitleFileId: string
  displayMode: LibraryWorkspaceDisplayMode
  comparisonSubtitleFileId: string
  guidelineProfileId: LibraryWorkspaceGuidelineProfileId
  qaCheckSettings: WorkspaceQaCheckSettings
  openRevision: number
  openFile: (target: LibraryWorkspaceTarget) => void
  setLibraryId: (libraryId: string) => void
  setActiveEditor: (editor: LibraryWorkspaceEditor) => void
  setActiveVideoFileId: (fileId: string) => void
  setActiveSubtitleFileId: (fileId: string) => void
  setDisplayMode: (value: LibraryWorkspaceDisplayMode) => void
  setComparisonSubtitleFileId: (value: string) => void
  setGuidelineProfileId: (value: LibraryWorkspaceGuidelineProfileId) => void
  setQaCheckEnabled: (id: WorkspaceQaCheckId, enabled: boolean) => void
  applyPersistedState: (state: LibraryWorkspacePersistedState) => void
  clear: () => void
}

function normalizeWorkspaceEditor(target: LibraryWorkspaceTarget): LibraryWorkspaceEditor {
  if (target.openMode === "subtitle") {
    return "subtitle"
  }
  return normalizeWorkspaceFileType(target.fileType) === "subtitle" ? "subtitle" : "video"
}

function normalizeWorkspaceFileType(value: string) {
  return value.trim().toLowerCase()
}

function normalizeWorkspaceDisplayMode(value: unknown): LibraryWorkspaceDisplayMode {
  if (value === "dual" || value === "bilingual") {
    return "dual"
  }
  return "single"
}

function normalizeWorkspaceGuidelineProfileId(value: unknown): LibraryWorkspaceGuidelineProfileId {
  switch (value) {
    case "bbc":
    case "ade":
      return value
    default:
      return "netflix"
  }
}

export const useLibraryWorkspaceStore = create<LibraryWorkspaceState>()(
  persist(
    (set) => ({
      libraryId: "",
      activeEditor: "video",
      activeVideoFileId: "",
      activeSubtitleFileId: "",
      displayMode: "single",
      comparisonSubtitleFileId: "",
      guidelineProfileId: "netflix",
      qaCheckSettings: DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
      openRevision: 0,
      openFile: (target) =>
        set((state) => {
          const editor = normalizeWorkspaceEditor(target)
          return {
            libraryId: target.libraryId?.trim() ?? state.libraryId,
            activeEditor: editor,
            activeVideoFileId:
              editor === "video" ? target.fileId : target.videoAssetId?.trim() ?? state.activeVideoFileId,
            activeSubtitleFileId:
              editor === "subtitle"
                ? target.fileId
                : target.subtitleAssetId?.trim() ?? state.activeSubtitleFileId,
            openRevision: state.openRevision + 1,
          }
        }),
      setLibraryId: (libraryId) => set({ libraryId: libraryId.trim() }),
      setActiveEditor: (editor) => set({ activeEditor: editor }),
      setActiveVideoFileId: (fileId) => set({ activeVideoFileId: fileId.trim() }),
      setActiveSubtitleFileId: (fileId) => set({ activeSubtitleFileId: fileId.trim() }),
      setDisplayMode: (value) => set({ displayMode: value }),
      setComparisonSubtitleFileId: (value) => set({ comparisonSubtitleFileId: value.trim() }),
      setGuidelineProfileId: (value) => set({ guidelineProfileId: value }),
      setQaCheckEnabled: (id, enabled) =>
        set((state) => ({
          qaCheckSettings: {
            ...state.qaCheckSettings,
            [id]: enabled,
          },
        })),
      applyPersistedState: (state) =>
        set({
          libraryId: state.libraryId.trim(),
          activeEditor: state.activeEditor,
          activeVideoFileId: state.activeVideoFileId.trim(),
          activeSubtitleFileId: state.activeSubtitleFileId.trim(),
          displayMode: normalizeWorkspaceDisplayMode(state.displayMode),
          comparisonSubtitleFileId: state.comparisonSubtitleFileId.trim(),
          guidelineProfileId: normalizeWorkspaceGuidelineProfileId(state.guidelineProfileId),
          qaCheckSettings: normalizeWorkspaceQaCheckSettings(state.qaCheckSettings),
        }),
      clear: () =>
        set({
          libraryId: "",
          activeEditor: "video",
          activeVideoFileId: "",
          activeSubtitleFileId: "",
          displayMode: "single",
          comparisonSubtitleFileId: "",
          guidelineProfileId: "netflix",
          qaCheckSettings: DEFAULT_WORKSPACE_QA_CHECK_SETTINGS,
          openRevision: 0,
        }),
    }),
    {
      name: "library-workspace-store",
      partialize: (state) => ({
        libraryId: state.libraryId,
        activeEditor: state.activeEditor,
        activeVideoFileId: state.activeVideoFileId,
        activeSubtitleFileId: state.activeSubtitleFileId,
        displayMode: state.displayMode,
        comparisonSubtitleFileId: state.comparisonSubtitleFileId,
        guidelineProfileId: state.guidelineProfileId,
        qaCheckSettings: state.qaCheckSettings,
      }),
      merge: (persisted, current) => ({
        ...current,
        ...(persisted as Partial<LibraryWorkspaceState>),
        displayMode: normalizeWorkspaceDisplayMode((persisted as Partial<LibraryWorkspaceState>)?.displayMode),
        guidelineProfileId: normalizeWorkspaceGuidelineProfileId(
          (persisted as Partial<LibraryWorkspaceState>)?.guidelineProfileId,
        ),
        qaCheckSettings: normalizeWorkspaceQaCheckSettings(
          (persisted as Partial<LibraryWorkspaceState>)?.qaCheckSettings,
        ),
        openRevision: 0,
      }),
    },
  ),
)

export function openLibraryWorkspace(target: LibraryWorkspaceTarget) {
  useLibraryWorkspaceStore.getState().openFile(target)
}
