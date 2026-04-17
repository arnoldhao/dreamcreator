import { useCallback, useMemo } from "react";

import { useSettings, useUpdateSettings } from "@/shared/query/settings";

export type DebugModeLevel = "off" | "basic" | "full";

function normalizeDebugMode(value: string | undefined, recordPrompt: boolean | undefined): DebugModeLevel {
  switch (value) {
    case "off":
    case "basic":
    case "full":
      return value;
    default:
      return recordPrompt ? "full" : "off";
  }
}

export function useDebugMode() {
  const settingsQuery = useSettings();
  const updateSettings = useUpdateSettings();

  const mode = useMemo(
    () =>
      normalizeDebugMode(
        settingsQuery.data?.gateway.runtime.debugMode,
        settingsQuery.data?.gateway.runtime.recordPrompt
      ),
    [settingsQuery.data?.gateway.runtime.debugMode, settingsQuery.data?.gateway.runtime.recordPrompt]
  );

  const setMode = useCallback(
    (value: DebugModeLevel) => {
      if (value === mode) {
        return;
      }
      updateSettings.mutate({
        gateway: {
          runtime: {
            debugMode: value,
            recordPrompt: value === "full",
          },
        },
      });
    },
    [mode, updateSettings]
  );

  return {
    mode,
    setMode,
    enabled: mode !== "off",
    isPending: settingsQuery.isLoading || updateSettings.isPending,
  };
}
