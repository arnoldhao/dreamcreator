import * as React from "react";
import { create } from "zustand";
import { persist } from "zustand/middleware";

const ASSISTANT_UI_STORAGE_KEY = "assistant-ui-preferences";

interface AssistantUiState {
  enabled: boolean;
  hasChosenProductMode: boolean;
  setEnabled: (enabled: boolean) => void;
  resetProductModeChoice: () => void;
}

export const useAssistantUiStore = create<AssistantUiState>()(
  persist(
    (set) => ({
      enabled: true,
      hasChosenProductMode: false,
      setEnabled: (enabled) => set({ enabled, hasChosenProductMode: true }),
      resetProductModeChoice: () => set({ hasChosenProductMode: false }),
    }),
    {
      name: ASSISTANT_UI_STORAGE_KEY,
      version: 2,
      migrate: (persistedState, version) => {
        const anyState = (persistedState ?? {}) as Partial<AssistantUiState>;
        if (version < 2) {
          return {
            enabled: typeof anyState.enabled === "boolean" ? anyState.enabled : true,
            hasChosenProductMode: false,
          };
        }
        return {
          enabled: typeof anyState.enabled === "boolean" ? anyState.enabled : true,
          hasChosenProductMode: typeof anyState.hasChosenProductMode === "boolean" ? anyState.hasChosenProductMode : false,
        };
      },
    }
  )
);

export function useAssistantUiMode() {
  const enabled = useAssistantUiStore((state) => state.enabled);
  const hasChosenProductMode = useAssistantUiStore((state) => state.hasChosenProductMode);
  const setEnabled = useAssistantUiStore((state) => state.setEnabled);
  const resetProductModeChoice = useAssistantUiStore((state) => state.resetProductModeChoice);
  return {
    enabled,
    hasChosenProductMode,
    setEnabled,
    resetProductModeChoice,
  };
}

export function useAssistantUiStorageSync() {
  React.useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }
    const handleStorage = (event: StorageEvent) => {
      if (event.key !== ASSISTANT_UI_STORAGE_KEY) {
        return;
      }
      void useAssistantUiStore.persist.rehydrate();
    };
    window.addEventListener("storage", handleStorage);
    return () => window.removeEventListener("storage", handleStorage);
  }, []);
}
