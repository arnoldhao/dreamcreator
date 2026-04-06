import { create } from "zustand";
import { persist } from "zustand/middleware";

import type { SetupNavItemId } from "./nav";

interface SetupCenterState {
  open: boolean;
  focusItemId: SetupNavItemId | null;
  aiDeferred: boolean;
  dependencyDeferred: boolean;
  skippedItemIds: string[];
  setOpen: (open: boolean) => void;
  setFocusItem: (itemId: SetupNavItemId | null) => void;
  deferAi: () => void;
  clearAiDeferred: () => void;
  deferDependencies: () => void;
  clearDependencyDeferred: () => void;
  skipItem: (itemId: string) => void;
  clearSkippedItem: (itemId: string) => void;
}

export const useSetupCenterStore = create<SetupCenterState>()(
  persist(
    (set) => ({
      open: false,
      focusItemId: null,
      aiDeferred: false,
      dependencyDeferred: false,
      skippedItemIds: [],
      setOpen: (open) => set({ open }),
      setFocusItem: (focusItemId) => set({ focusItemId }),
      deferAi: () => set({ aiDeferred: true }),
      clearAiDeferred: () => set({ aiDeferred: false }),
      deferDependencies: () => set({ dependencyDeferred: true }),
      clearDependencyDeferred: () => set({ dependencyDeferred: false }),
      skipItem: (itemId) =>
        set((state) => ({
          skippedItemIds: state.skippedItemIds.includes(itemId)
            ? state.skippedItemIds
            : [...state.skippedItemIds, itemId],
        })),
      clearSkippedItem: (itemId) =>
        set((state) => {
          if (!state.skippedItemIds.includes(itemId)) {
            return state;
          }
          return {
            skippedItemIds: state.skippedItemIds.filter((current) => current !== itemId),
          };
        }),
    }),
    {
      name: "setup-center-storage",
      partialize: (state) => ({
        aiDeferred: state.aiDeferred,
        dependencyDeferred: state.dependencyDeferred,
        skippedItemIds: state.skippedItemIds,
      }),
    }
  )
);

export function useSetupCenter() {
  const open = useSetupCenterStore((state) => state.open);
  const focusItemId = useSetupCenterStore((state) => state.focusItemId);
  const aiDeferred = useSetupCenterStore((state) => state.aiDeferred);
  const dependencyDeferred = useSetupCenterStore((state) => state.dependencyDeferred);
  const skippedItemIds = useSetupCenterStore((state) => state.skippedItemIds);
  const setOpen = useSetupCenterStore((state) => state.setOpen);
  const setFocusItem = useSetupCenterStore((state) => state.setFocusItem);
  const deferAi = useSetupCenterStore((state) => state.deferAi);
  const clearAiDeferred = useSetupCenterStore((state) => state.clearAiDeferred);
  const deferDependencies = useSetupCenterStore((state) => state.deferDependencies);
  const clearDependencyDeferred = useSetupCenterStore((state) => state.clearDependencyDeferred);
  const skipItem = useSetupCenterStore((state) => state.skipItem);
  const clearSkippedItem = useSetupCenterStore((state) => state.clearSkippedItem);

  return {
    open,
    focusItemId,
    aiDeferred,
    dependencyDeferred,
    skippedItemIds,
    setOpen,
    setFocusItem,
    openDialog: () => {
      setFocusItem(null);
      setOpen(true);
    },
    openDialogForItem: (itemId: SetupNavItemId) => {
      setFocusItem(itemId);
      setOpen(true);
    },
    closeDialog: () => {
      setFocusItem(null);
      setOpen(false);
    },
    deferAi,
    clearAiDeferred,
    deferDependencies,
    clearDependencyDeferred,
    skipItem,
    clearSkippedItem,
    clearFocusItem: () => setFocusItem(null),
  };
}
