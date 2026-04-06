import { create } from "zustand";

export type UpdateKind = "app" | "dependency" | "plugin";
export type UpdateStatus =
  | "idle"
  | "checking"
  | "no_update"
  | "available"
  | "downloading"
  | "installing"
  | "ready_to_restart"
  | "error";

export interface UpdateInfo {
  kind: UpdateKind;
  currentVersion: string;
  latestVersion: string;
  changelog: string;
  downloadURL: string;
  checkedAt?: string;
  status: UpdateStatus;
  progress: number;
  message?: string;
}

export interface UpdateStore {
  info: UpdateInfo;
  setInfo: (info: UpdateInfo) => void;
}

const defaultInfo: UpdateInfo = {
  kind: "app",
  currentVersion: "",
  latestVersion: "",
  changelog: "",
  downloadURL: "",
  status: "idle",
  progress: 0,
  message: "",
};

export const useUpdateStore = create<UpdateStore>((set) => ({
  info: defaultInfo,
  setInfo: (info) => set({ info }),
}));

export function normalizeUpdateInfo(raw: Partial<UpdateInfo> | null | undefined): UpdateInfo {
  if (!raw) {
    return defaultInfo;
  }
  const anyRaw = raw as any;
  return {
    kind: (raw.kind as UpdateKind) ?? (anyRaw.Kind as UpdateKind) ?? "app",
    currentVersion: raw.currentVersion ?? anyRaw.CurrentVersion ?? "",
    latestVersion: raw.latestVersion ?? anyRaw.LatestVersion ?? "",
    changelog: raw.changelog ?? anyRaw.Changelog ?? "",
    downloadURL: raw.downloadURL ?? anyRaw.DownloadURL ?? "",
    checkedAt: raw.checkedAt ?? anyRaw.CheckedAt,
    status: (raw.status as UpdateStatus) ?? (anyRaw.Status as UpdateStatus) ?? "idle",
    progress: typeof raw.progress === "number" ? raw.progress : typeof anyRaw.Progress === "number" ? anyRaw.Progress : 0,
    message: raw.message ?? anyRaw.Message ?? "",
  };
}
