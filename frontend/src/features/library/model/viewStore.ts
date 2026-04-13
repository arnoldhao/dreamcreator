import { create } from "zustand"
import { persist } from "zustand/middleware"
import type { VisibilityState } from "@tanstack/react-table"

import type { LibraryPageTab, LibraryViewMode } from "./types"

type ColumnVisibilityByView = {
  task: VisibilityState
  file: VisibilityState
}

type LibraryViewState = {
  pageTab: LibraryPageTab
  viewMode: LibraryViewMode
  columnVisibility: ColumnVisibilityByView
  rowsPerPage: number
  setPageTab: (tab: LibraryPageTab) => void
  setViewMode: (mode: LibraryViewMode) => void
  setColumnVisibility: (mode: LibraryViewMode, visibility: VisibilityState) => void
  setRowsPerPage: (rows: number) => void
}

const defaultColumnVisibility: ColumnVisibilityByView = {
  task: {
    platform: false,
    uploader: false,
    publishTime: false,
  },
  file: {},
}

export const useLibraryViewStore = create<LibraryViewState>()(
  persist(
    (set) => ({
      pageTab: "overview",
      viewMode: "task",
      columnVisibility: defaultColumnVisibility,
      rowsPerPage: 20,
      setPageTab: (pageTab) => set({ pageTab }),
      setViewMode: (mode) => set({ viewMode: mode }),
      setColumnVisibility: (mode, visibility) =>
        set((state) => ({
          columnVisibility: { ...state.columnVisibility, [mode]: visibility },
        })),
      setRowsPerPage: (rows) => set({ rowsPerPage: rows }),
    }),
    {
      name: "library-view-store",
    }
  )
)
