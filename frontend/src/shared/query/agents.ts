import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

export interface Agent {
  id: string;
  name: string;
  description: string;
  enabled: boolean;
  threadId: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateAgentRequest {
  name: string;
  description: string;
}

export interface UpdateAgentRequest {
  id: string;
  name?: string;
  description?: string;
  enabled?: boolean;
}

export interface DeleteAgentRequest {
  id: string;
}


export interface AgentRun {
  id: string;
  threadId: string;
  assistantMessageId: string;
  userMessageId: string;
  agentId: string;
  status: string;
  contentPartial: string;
  createdAt: string;
  updatedAt: string;
}

export interface AgentRunEvent {
  id: number;
  runId: string;
  eventType: string;
  payloadJson: string;
  createdAt: string;
}

export const agentsQueryKey = (includeDisabled: boolean) => ["agents", includeDisabled];

export function useAgents(includeDisabled = true) {
  return useQuery({
    queryKey: agentsQueryKey(includeDisabled),
    queryFn: async (): Promise<Agent[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.ListAgents",
        includeDisabled
      );
      return (result as Agent[]) ?? [];
    },
  });
}

export function useCreateAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: CreateAgentRequest): Promise<Agent> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.CreateAgent",
        request
      );
      return result as Agent;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });
}

export function useUpdateAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: UpdateAgentRequest): Promise<Agent> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.UpdateAgent",
        request
      );
      return result as Agent;
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });
}

export function useDeleteAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (request: DeleteAgentRequest) => {
      await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.DeleteAgent",
        request
      );
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });
}

export function useAgentRuns(agentId: string, limit = 50, enabled = true) {
  return useQuery({
    queryKey: ["agentRuns", agentId, limit],
    queryFn: async (): Promise<AgentRun[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.ListAgentRuns",
        { agentId, limit }
      );
      return (result as AgentRun[]) ?? [];
    },
    enabled: enabled && Boolean(agentId),
  });
}

export function useAgentRunEvents(runId: string, afterId = 0, limit = 200, enabled = true) {
  return useQuery({
    queryKey: ["agentRunEvents", runId, afterId, limit],
    queryFn: async (): Promise<AgentRunEvent[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.AgentHandler.ListAgentRunEvents",
        { runId, afterId, limit }
      );
      return (result as AgentRunEvent[]) ?? [];
    },
    enabled: enabled && Boolean(runId),
  });
}
