import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import type { Notice, NoticeListRequest, NoticeSurface } from "@/shared/contracts/notice";

export const noticeListKey = (request: NoticeListRequest = {}) => ["notices", "list", request] as const;
export const noticeUnreadKey = (surface: NoticeSurface) => ["notices", "unread", surface] as const;

export function useNotices(request: NoticeListRequest = {}) {
  return useQuery({
    queryKey: noticeListKey(request),
    queryFn: async (): Promise<Notice[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.NoticeHandler.List",
        request,
      );
      return (result as Notice[]) ?? [];
    },
    staleTime: 5_000,
  });
}

export function useNoticeUnreadCount(surface: NoticeSurface) {
  return useQuery({
    queryKey: noticeUnreadKey(surface),
    queryFn: async (): Promise<number> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.NoticeHandler.UnreadCount",
        surface,
      );
      return typeof result === "number" ? result : Number(result ?? 0);
    },
    staleTime: 5_000,
  });
}

export function useMarkNoticeRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: { ids: string[]; read?: boolean }) => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.NoticeHandler.MarkRead",
        {
          ids: request.ids,
          read: request.read ?? true,
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
    },
  });
}

export function useMarkAllNoticesRead() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (surface: NoticeSurface) => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.NoticeHandler.MarkAllRead",
        surface,
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
    },
  });
}

export function useArchiveNotice() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: { ids: string[]; archived?: boolean }) => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.NoticeHandler.Archive",
        {
          ids: request.ids,
          archived: request.archived ?? true,
        },
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notices"] });
    },
  });
}
