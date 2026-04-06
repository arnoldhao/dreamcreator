import { useQuery } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";

import type { GatewayToolSpec } from "@/shared/store/gatewayTools";

export const gatewayToolsKey = ["gateway", "tools"];

export function useGatewayTools(enabled = true) {
  return useQuery({
    queryKey: gatewayToolsKey,
    queryFn: async (): Promise<GatewayToolSpec[]> => {
      const result = await Call.ByName(
        "dreamcreator/internal/presentation/wails.ToolsHandler.ListTools"
      );
      return (result as GatewayToolSpec[]) ?? [];
    },
    enabled,
    staleTime: 5_000,
  });
}
