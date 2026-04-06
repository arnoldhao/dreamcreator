import { requestGateway } from "@/shared/realtime";

export async function gatewayRequest<T = any>(method: string, params?: unknown, timeoutMs?: number): Promise<T> {
  const result = await requestGateway(method, params, timeoutMs);
  return result as T;
}
