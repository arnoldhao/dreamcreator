import type { GatewayToolMethodSpec } from "@/shared/store/gatewayTools";

const isRecord = (value: unknown): value is Record<string, unknown> =>
  Boolean(value) && typeof value === "object" && !Array.isArray(value);

export const toPrettyJSON = (value: unknown) => {
  try {
    return JSON.stringify(value, null, 2);
  } catch {
    return String(value ?? "");
  }
};

const parseActionsFromSchema = (schemaJson?: string) => {
  if (!schemaJson || !schemaJson.trim()) {
    return [] as string[];
  }
  try {
    const parsed = JSON.parse(schemaJson) as unknown;
    if (!isRecord(parsed)) {
      return [] as string[];
    }
    const properties = isRecord(parsed.properties) ? (parsed.properties as Record<string, unknown>) : undefined;
    if (!properties) {
      return [] as string[];
    }
    const actionDef = isRecord(properties.action) ? (properties.action as Record<string, unknown>) : undefined;
    const methodDef = isRecord(properties.method) ? (properties.method as Record<string, unknown>) : undefined;
    const collect = (candidate: Record<string, unknown> | undefined) => {
      if (!candidate || !Array.isArray(candidate.enum)) {
        return [] as string[];
      }
      return candidate.enum
        .map((item) => (typeof item === "string" ? item.trim() : ""))
        .filter((item) => item !== "");
    };
    const values = [...collect(actionDef), ...collect(methodDef)];
    const seen = new Set<string>();
    return values.filter((item) => {
      if (seen.has(item)) {
        return false;
      }
      seen.add(item);
      return true;
    });
  } catch {
    return [] as string[];
  }
};

const normalizeMethodSpec = (candidate: GatewayToolMethodSpec): GatewayToolMethodSpec | null => {
  const name = candidate.name?.trim?.();
  if (!name) {
    return null;
  }
  return {
    name,
    inputSchema: candidate.inputSchema,
    outputSchema: candidate.outputSchema,
    inputExample: candidate.inputExample,
    outputExample: candidate.outputExample,
  };
};

export const parseGatewayToolMethods = (methods?: GatewayToolMethodSpec[], schemaJson?: string) => {
  const normalized = Array.isArray(methods)
    ? methods
        .map(normalizeMethodSpec)
        .filter((item): item is GatewayToolMethodSpec => item !== null)
    : [];
  if (normalized.length > 0) {
    return normalized;
  }
  return parseActionsFromSchema(schemaJson).map((name) => ({ name }));
};

export const findGatewayToolMethod = (methods: GatewayToolMethodSpec[], name: string) => {
  return methods.find((item) => item.name === name) ?? null;
};

export const buildToolInputExample = (toolId: string, method: GatewayToolMethodSpec | null, action: string) => {
  if (method?.inputExample !== undefined) {
    return method.inputExample;
  }
  const resolvedAction = action || method?.name || toolId;
  if (!resolvedAction) {
    return {};
  }
  if (resolvedAction === toolId) {
    return {
      tool: toolId,
      args: {},
    };
  }
  return {
    tool: toolId,
    action: resolvedAction,
  };
};

export const buildToolOutputExample = (toolId: string, method: GatewayToolMethodSpec | null, action: string) => {
  if (method?.outputExample !== undefined) {
    return method.outputExample;
  }
  const resolvedAction = action || method?.name || toolId;
  if (!resolvedAction) {
    return {};
  }
  if (resolvedAction === toolId) {
    return {
      ok: true,
      result: {},
    };
  }
  return {
    ok: true,
    action: resolvedAction,
    result: {},
  };
};

export const buildGatewayInputExample = (method: GatewayToolMethodSpec | null, action: string) => {
  return buildToolInputExample("gateway", method, action);
};

export const buildGatewayOutputExample = (method: GatewayToolMethodSpec | null, action: string) => {
  return buildToolOutputExample("gateway", method, action);
};
