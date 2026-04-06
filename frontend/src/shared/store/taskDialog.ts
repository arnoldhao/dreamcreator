import { create } from "zustand"

type TaskDialogState = {
  open: boolean
  operationId: string | null
  openTaskDialog: (operationId: string) => void
  closeTaskDialog: () => void
}

export const useTaskDialogStore = create<TaskDialogState>((set) => ({
  open: false,
  operationId: null,
  openTaskDialog: (operationId) => set({ open: true, operationId }),
  closeTaskDialog: () => set({ open: false, operationId: null }),
}))

export const openTaskDialog = (operationId: string) => {
  useTaskDialogStore.getState().openTaskDialog(operationId)
}

export const closeTaskDialog = () => {
  useTaskDialogStore.getState().closeTaskDialog()
}
