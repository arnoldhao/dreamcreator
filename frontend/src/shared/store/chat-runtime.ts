import { create } from "zustand";

type ChatRuntimeDefaults = {
  assistantId?: string;
};

export type ContextTokenSnapshot = {
  promptTokens: number;
  totalTokens: number;
  contextWindowTokens?: number;
  warnTokens?: number;
  hardTokens?: number;
  contextFresh?: boolean;
  updatedAt?: number;
};

export type RuntimeUsageSnapshot = {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
  updatedAt?: number;
};

type ChatRuntimeState = {
  assistantId: string;
  contextTokens: Record<string, ContextTokenSnapshot>;
  runUsage: Record<string, RuntimeUsageSnapshot>;
  setAssistantId: (assistantId: string) => void;
  setContextTokens: (threadId: string, snapshot: ContextTokenSnapshot) => void;
  clearContextTokens: (threadId: string) => void;
  setRunUsage: (threadId: string, snapshot: RuntimeUsageSnapshot) => void;
  clearRunUsage: (threadId: string) => void;
  applyDefaults: (defaults: ChatRuntimeDefaults) => void;
};

const normalize = (value: string | undefined) => (value ?? "").trim();

export const useChatRuntimeStore = create<ChatRuntimeState>((set) => ({
  assistantId: "",
  contextTokens: {},
  runUsage: {},
  setAssistantId: (assistantId) =>
    set(() => ({
      assistantId: normalize(assistantId),
    })),
  setContextTokens: (threadId, snapshot) =>
    set((state) => {
      const id = normalize(threadId);
      if (!id) {
        return state;
      }
      return {
        contextTokens: {
          ...state.contextTokens,
          [id]: snapshot,
        },
      };
    }),
  clearContextTokens: (threadId) =>
    set((state) => {
      const id = normalize(threadId);
      if (!id || !state.contextTokens[id]) {
        return state;
      }
      const next = { ...state.contextTokens };
      delete next[id];
      return { contextTokens: next };
    }),
  setRunUsage: (threadId, snapshot) =>
    set((state) => {
      const id = normalize(threadId);
      if (!id) {
        return state;
      }
      return {
        runUsage: {
          ...state.runUsage,
          [id]: snapshot,
        },
      };
    }),
  clearRunUsage: (threadId) =>
    set((state) => {
      const id = normalize(threadId);
      if (!id || !state.runUsage[id]) {
        return state;
      }
      const next = { ...state.runUsage };
      delete next[id];
      return { runUsage: next };
    }),
  applyDefaults: ({ assistantId }) =>
    set((state) => ({
      assistantId: state.assistantId || normalize(assistantId),
    })),
}));
