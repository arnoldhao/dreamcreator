import { create } from "zustand"

import type {
  LibraryDTO,
  LibraryFileDTO,
  LibraryHistoryRecordDTO,
  LibraryOperationDTO,
  OperationListItemDTO,
  WorkspaceStateRecordDTO,
} from "@/shared/contracts/library"

type LibraryRealtimeState = {
  hasLiveUpdates: boolean
  operations: LibraryOperationDTO[]
  files: LibraryFileDTO[]
  histories: LibraryHistoryRecordDTO[]
  workspaceHeads: Record<string, WorkspaceStateRecordDTO>
  markLiveUpdates: () => void
  reset: () => void
  replaceLibrary: (library: LibraryDTO) => void
  removeLibrary: (libraryId: string) => void
  upsertOperation: (operation: LibraryOperationDTO) => void
  deleteOperation: (operationId: string) => void
  upsertFile: (file: LibraryFileDTO) => void
  deleteFile: (fileId: string) => void
  upsertHistory: (history: LibraryHistoryRecordDTO) => void
  replaceWorkspaceHead: (workspace: WorkspaceStateRecordDTO) => void
}

function upsertById<T extends { id: string }>(items: T[], next: T): T[] {
  const index = items.findIndex((item) => item.id === next.id)
  if (index === -1) {
    return [next, ...items]
  }
  const updated = [...items]
  updated[index] = { ...updated[index], ...next }
  return updated
}

function upsertByRecordId(items: LibraryHistoryRecordDTO[], next: LibraryHistoryRecordDTO) {
  const index = items.findIndex((item) => item.recordId === next.recordId)
  if (index === -1) {
    return [next, ...items]
  }
  const updated = [...items]
  updated[index] = { ...updated[index], ...next }
  return updated
}

function mergeFiles(existing: LibraryFileDTO[], incoming: LibraryFileDTO[]) {
  return incoming.reduce((accumulator, item) => upsertById(accumulator, item), existing)
}

function mergeHistory(existing: LibraryHistoryRecordDTO[], incoming: LibraryHistoryRecordDTO[]) {
  return incoming.reduce((accumulator, item) => upsertByRecordId(accumulator, item), existing)
}

export function toOperationListItem(operation: LibraryOperationDTO): OperationListItemDTO {
  return {
    operationId: operation.id,
    libraryId: operation.libraryId,
    name: operation.displayName,
    kind: operation.kind,
    status: operation.status,
    domain: operation.sourceDomain,
    sourceIcon: operation.sourceIcon,
    platform: operation.meta.platform,
    uploader: operation.meta.uploader,
    publishTime: operation.meta.publishTime,
    progress: operation.progress,
    outputFiles: operation.outputFiles,
    metrics: operation.metrics,
    startedAt: operation.startedAt,
    finishedAt: operation.finishedAt,
    createdAt: operation.createdAt,
  }
}

export const useLibraryRealtimeStore = create<LibraryRealtimeState>((set) => ({
  hasLiveUpdates: false,
  operations: [],
  files: [],
  histories: [],
  workspaceHeads: {},
  markLiveUpdates: () => set({ hasLiveUpdates: true }),
  reset: () => set({ hasLiveUpdates: false, operations: [], files: [], histories: [], workspaceHeads: {} }),
  replaceLibrary: (library) =>
    set((state) => ({
      hasLiveUpdates: true,
      files: mergeFiles(
        state.files.filter((item) => item.libraryId !== library.id),
        library.files ?? [],
      ),
      histories: mergeHistory(
        state.histories.filter((item) => item.libraryId !== library.id),
        library.records.history ?? [],
      ),
      workspaceHeads: library.records.workspaceStateHead
        ? { ...state.workspaceHeads, [library.id]: library.records.workspaceStateHead }
        : state.workspaceHeads,
    })),
  removeLibrary: (libraryId) =>
    set((state) => {
      const nextWorkspaceHeads = { ...state.workspaceHeads }
      delete nextWorkspaceHeads[libraryId]
      return {
        hasLiveUpdates: true,
        operations: state.operations.filter((item) => item.libraryId !== libraryId),
        files: state.files.filter((item) => item.libraryId !== libraryId),
        histories: state.histories.filter((item) => item.libraryId !== libraryId),
        workspaceHeads: nextWorkspaceHeads,
      }
    }),
  upsertOperation: (operation) =>
    set((state) => ({
      hasLiveUpdates: true,
      operations: upsertById(state.operations, operation),
    })),
  deleteOperation: (operationId) =>
    set((state) => ({
      operations: state.operations.filter((item) => item.id !== operationId),
    })),
  upsertFile: (file) =>
    set((state) => ({
      hasLiveUpdates: true,
      files: upsertById(state.files, file),
    })),
  deleteFile: (fileId) =>
    set((state) => ({
      files: state.files.filter((item) => item.id !== fileId),
    })),
  upsertHistory: (history) =>
    set((state) => ({
      hasLiveUpdates: true,
      histories: upsertByRecordId(state.histories, history),
    })),
  replaceWorkspaceHead: (workspace) =>
    set((state) => ({
      hasLiveUpdates: true,
      workspaceHeads: { ...state.workspaceHeads, [workspace.libraryId]: workspace },
    })),
}))
