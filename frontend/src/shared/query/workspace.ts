import { useMutation, useQuery } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

export type AssistantWorkspaceDirectory = {
  assistantId: string;
  workspaceId?: string;
  rootPath?: string;
};

export const assistantWorkspaceDirectoryKey = (assistantId: string) => [
  "assistant-workspace-directory",
  assistantId,
];

export function useAssistantWorkspaceDirectory(assistantId: string | null) {
  return useQuery({
    queryKey: assistantId ? assistantWorkspaceDirectoryKey(assistantId) : ["assistant-workspace-directory", "empty"],
    queryFn: async (): Promise<AssistantWorkspaceDirectory> => {
      if (!assistantId) {
        throw new Error("assistant id is required");
      }
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.WorkspaceHandler.GetAssistantWorkspaceDirectory",
        assistantId
      );
      return (result as AssistantWorkspaceDirectory) ?? { assistantId };
    },
    enabled: Boolean(assistantId),
    staleTime: 0,
    refetchOnWindowFocus: false,
  });
}

export function useOpenAssistantWorkspaceDirectory() {
  return useMutation({
    mutationFn: async (assistantId: string) => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.WorkspaceHandler.OpenAssistantWorkspaceDirectory",
        assistantId
      );
    },
  });
}
