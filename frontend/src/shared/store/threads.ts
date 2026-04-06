import { create } from "zustand";

export type ThreadStatus = "regular" | "archived";
export type ThreadTitleChangedBy = "user" | "summary";

export interface ThreadSummary {
  id: string;
  assistantId: string;
  title: string;
  titleIsDefault?: boolean;
  titleChangedBy?: ThreadTitleChangedBy;
  status: ThreadStatus;
  createdAt: string;
  updatedAt: string;
  lastInteractiveAt: string;
  deletedAt: string;
  purgeAfter: string;
  workspaceName?: string;
}

interface ThreadStoreState {
  threads: Record<string, ThreadSummary>;
  setThreads: (threads: ThreadSummary[]) => void;
  upsertThread: (thread: ThreadSummary) => void;
  patchThread: (id: string, patch: Partial<ThreadSummary>) => void;
  removeThread: (id: string) => void;
}

export const useThreadStore = create<ThreadStoreState>((set) => ({
  threads: {},
  setThreads: (threads) =>
    set(() => ({
      threads: threads.reduce<Record<string, ThreadSummary>>((acc, thread) => {
        acc[thread.id] = thread;
        return acc;
      }, {}),
    })),
  upsertThread: (thread) =>
    set((state) => ({
      threads: {
        ...state.threads,
        [thread.id]: thread,
      },
    })),
  patchThread: (id, patch) =>
    set((state) => {
      const current = state.threads[id];
      if (!current) {
        return state;
      }
      return {
        threads: {
          ...state.threads,
          [id]: {
            ...current,
            ...patch,
          },
        },
      };
    }),
  removeThread: (id) =>
    set((state) => {
      if (!state.threads[id]) {
        return state;
      }
      const next = { ...state.threads };
      delete next[id];
      return { threads: next };
    }),
}));
