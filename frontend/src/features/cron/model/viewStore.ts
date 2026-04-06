import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { VisibilityState } from "@tanstack/react-table";

export type CronTab = "overview" | "list" | "records";
export type JobsViewMode = "table" | "cards";
export type RunsViewMode = "table" | "cards";

type CronColumnVisibilityByTab = {
  list: VisibilityState;
  records: VisibilityState;
};

type CronViewState = {
  activeTab: CronTab;
  jobsViewMode: JobsViewMode;
  runsViewMode: RunsViewMode;
  jobsRowsPerPage: number;
  runsRowsPerPage: number;
  columnVisibility: CronColumnVisibilityByTab;
  setActiveTab: (tab: CronTab | string) => void;
  setJobsViewMode: (mode: JobsViewMode) => void;
  setRunsViewMode: (mode: RunsViewMode) => void;
  setJobsRowsPerPage: (value: number) => void;
  setRunsRowsPerPage: (value: number) => void;
  setColumnVisibility: (tab: Exclude<CronTab, "overview">, visibility: VisibilityState) => void;
};

const defaultColumnVisibility: CronColumnVisibilityByTab = {
  list: {
    id: false,
    description: false,
    assistantId: false,
    sessionTarget: false,
    wakeMode: false,
    payload: false,
    delivery: false,
    sourceChannel: false,
    runningAt: false,
    lastError: false,
    lastDuration: false,
    consecutiveErrors: false,
    scheduleErrors: false,
    lastDeliveryStatus: false,
    lastDeliveryError: false,
    deleteAfterRun: false,
    createdAt: false,
    updatedAt: false,
  },
  records: {
    runId: false,
    stage: true,
    ended: true,
    deliveryStatus: true,
    summary: true,
    usage: false,
    model: false,
    provider: false,
    sessionKey: false,
    deliveryError: false,
    error: false,
  },
};

const normalizeCronTab = (tab: unknown): CronTab => {
  const normalized = typeof tab === "string" ? tab.trim().toLowerCase() : "";
  if (normalized === "overview" || normalized === "list" || normalized === "records") {
    return normalized;
  }
  return "overview";
};

export const useCronViewStore = create<CronViewState>()(
  persist(
    (set) => ({
      activeTab: "overview",
      jobsViewMode: "table",
      runsViewMode: "table",
      jobsRowsPerPage: 20,
      runsRowsPerPage: 20,
      columnVisibility: defaultColumnVisibility,
      setActiveTab: (tab) => set({ activeTab: normalizeCronTab(tab) }),
      setJobsViewMode: (mode) => set({ jobsViewMode: mode }),
      setRunsViewMode: (mode) => set({ runsViewMode: mode }),
      setJobsRowsPerPage: (value) => set({ jobsRowsPerPage: value }),
      setRunsRowsPerPage: (value) => set({ runsRowsPerPage: value }),
      setColumnVisibility: (tab, visibility) =>
        set((state) => ({
          columnVisibility: {
            ...state.columnVisibility,
            [tab]: visibility,
          },
        })),
    }),
    {
      name: "cron-view-store",
      onRehydrateStorage: () => (state) => {
        if (!state) {
          return;
        }
        state.setActiveTab(state.activeTab);
      },
    }
  )
);
