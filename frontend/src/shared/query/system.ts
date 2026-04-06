import { useQuery } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

export interface CurrentUserProfile {
  username: string;
  displayName: string;
  initials?: string;
  avatarPath?: string;
  avatarBase64?: string;
  avatarMime?: string;
}

export const CURRENT_USER_PROFILE_QUERY_KEY = ["system", "current-user-profile"];

export function useCurrentUserProfile() {
  return useQuery({
    queryKey: CURRENT_USER_PROFILE_QUERY_KEY,
    queryFn: async (): Promise<CurrentUserProfile> => {
      const result = await Call.ByName("dreamcreator/internal/presentation/wails.SystemHandler.GetCurrentUserProfile");
      return normalizeCurrentUserProfile(result as Partial<CurrentUserProfile> | null | undefined);
    },
    staleTime: Infinity,
    refetchInterval: 60 * 60 * 1_000,
    retry: false,
  });
}

function normalizeCurrentUserProfile(raw: Partial<CurrentUserProfile> | null | undefined): CurrentUserProfile {
  const anyRaw = (raw ?? {}) as Record<string, unknown>;
  return {
    username: stringOrEmpty(raw?.username) || stringOrEmpty(anyRaw.Username),
    displayName: stringOrEmpty(raw?.displayName) || stringOrEmpty(anyRaw.DisplayName),
    initials: stringOrEmpty(raw?.initials) || stringOrEmpty(anyRaw.Initials) || undefined,
    avatarPath: stringOrEmpty(raw?.avatarPath) || stringOrEmpty(anyRaw.AvatarPath) || undefined,
    avatarBase64: stringOrEmpty(raw?.avatarBase64) || stringOrEmpty(anyRaw.AvatarBase64) || undefined,
    avatarMime: stringOrEmpty(raw?.avatarMime) || stringOrEmpty(anyRaw.AvatarMime) || undefined,
  };
}

function stringOrEmpty(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
