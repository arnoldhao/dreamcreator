import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { ConnectionStatus, RealtimeEvent } from "./types";

const MAX_PER_TOPIC = 100;

export interface RealtimeState {
  status: ConnectionStatus;
  url: string;
  topics: string[];
  messages: Record<string, RealtimeEvent[]>;
  metrics: {
    reconnects: number;
    replayEvents: number;
    resyncRequired: number;
    duplicateDrops: number;
    lastStatusChangeAt: number;
  };
  setStatus: (status: ConnectionStatus, url?: string) => void;
  registerTopic: (topic: string) => void;
  pushMessage: (event: RealtimeEvent) => void;
  recordMetric: (kind: "reconnect" | "replay" | "resync-required" | "duplicate-drop") => void;
  clearMessages: (topic?: string) => void;
}

const defaultRealtimeMetrics = () => ({
  reconnects: 0,
  replayEvents: 0,
  resyncRequired: 0,
  duplicateDrops: 0,
  lastStatusChangeAt: 0,
});

export const useRealtimeStore = create<RealtimeState>()(
  persist(
    (set, get) => ({
      status: "disconnected",
      url: "",
      topics: [],
      messages: {},
      metrics: defaultRealtimeMetrics(),
      setStatus: (status, url) =>
        set((state) => ({
          status,
          url: url ?? state.url,
          metrics: {
            ...state.metrics,
            lastStatusChangeAt: Date.now(),
          },
        })),
      registerTopic: (topic) =>
        set((state) =>
          state.topics.includes(topic)
            ? state
            : {
                topics: [...state.topics, topic],
              }
        ),
      pushMessage: (event) =>
        set((state) => {
          const existing = state.messages[event.topic] ?? [];
          const next = [...existing, event].slice(-MAX_PER_TOPIC);
          return {
            messages: {
              ...state.messages,
              [event.topic]: next,
            },
          };
        }),
      recordMetric: (kind) =>
        set((state) => {
          const metrics = { ...state.metrics };
          if (kind === "reconnect") {
            metrics.reconnects += 1;
          } else if (kind === "replay") {
            metrics.replayEvents += 1;
          } else if (kind === "resync-required") {
            metrics.resyncRequired += 1;
          } else if (kind === "duplicate-drop") {
            metrics.duplicateDrops += 1;
          }
          return { metrics };
        }),
      clearMessages: (topic) =>
        set((state) => {
          if (!topic) {
            return { messages: {} };
          }
          const { [topic]: _removed, ...rest } = state.messages;
          return { messages: rest };
        }),
    }),
    {
      name: "realtime-messages",
      version: 2,
      partialize: (state) => ({
        topics: state.topics,
        url: state.url,
        metrics: state.metrics,
      }),
      migrate: (persistedState) => {
        const state = (persistedState ?? {}) as Partial<RealtimeState>;
        return {
          status: "disconnected" as const,
          url: typeof state.url === "string" ? state.url : "",
          topics: Array.isArray(state.topics) ? state.topics.filter((topic): topic is string => typeof topic === "string") : [],
          messages: {},
          metrics: {
            ...defaultRealtimeMetrics(),
            ...(state.metrics ?? {}),
          },
        };
      },
    }
  )
);
