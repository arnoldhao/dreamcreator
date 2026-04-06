import { create } from "zustand";
import { persist } from "zustand/middleware";

export type DebugModeLevel = "off" | "basic" | "full";

interface DebugState {
  mode: DebugModeLevel;
  setMode: (value: DebugModeLevel) => void;
}

export const useDebugStore = create<DebugState>()(
  persist(
    (set) => ({
      mode: "off",
      setMode: (value) => set({ mode: value }),
    }),
    {
      name: "debug-mode",
    }
  )
);

export function useDebugMode() {
  const mode = useDebugStore((state) => state.mode);
  const setMode = useDebugStore((state) => state.setMode);
  return {
    mode,
    setMode,
    enabled: mode !== "off",
  };
}
