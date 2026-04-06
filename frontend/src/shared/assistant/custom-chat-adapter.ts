import type {
  ChatModelAdapter,
  ChatModelRunResult,
  ThreadMessage,
} from "@assistant-ui/react";

import { resolveRemoteThreadId } from "./thread-identities";
import {
  applyGatewayEvent,
  applyStreamEvent,
  createStreamParserState,
  type ChatEvent,
} from "./stream-parser";
import { messageBus } from "@/shared/message";
import { useSettingsStore } from "@/shared/store/settings";
import { useChatRuntimeStore } from "@/shared/store/chat-runtime";
import { requestGateway, startGateway, subscribeGatewayEvents, type GatewayEvent } from "@/shared/realtime";

export type UIMessagePart = {
  type: string;
  parentId?: string;
  text?: string;
  state?: string;
  toolCallId?: string;
  toolName?: string;
  input?: unknown;
  output?: unknown;
  errorText?: string;
  data?: unknown;
};

export type UIMessage = {
  id: string;
  role: "system" | "user" | "assistant";
  text?: string;
  content?: string;
  parts: UIMessagePart[];
};

const requestThreadAbort = async (threadId: string, reason: string) => {
  if (!threadId) {
    return;
  }
  try {
    await requestGateway("runtime.abort", {
      sessionId: threadId,
      reason,
    });
  } catch {
    // Best effort: cancel should not throw if abort endpoint is temporarily unreachable.
  }
};

const shouldAbortThreadForSignal = (signal: AbortSignal) => {
  const reason = (signal as AbortSignal & { reason?: unknown }).reason;
  if (!reason || typeof reason !== "object") {
    return false;
  }
  const meta = reason as Record<string, unknown>;
  if (typeof meta.detach === "boolean") {
    return meta.detach === false;
  }
  return false;
};

const toToolPayload = (tools?: Record<string, any>) => {
  if (!tools) {
    return undefined;
  }
  const entries = Object.entries(tools).filter(([, tool]) => {
    if (!tool || typeof tool !== "object") {
      return false;
    }
    if (tool.disabled) {
      return false;
    }
    return tool.type !== "backend";
  });
  if (entries.length === 0) {
    return undefined;
  }
  return Object.fromEntries(
    entries.map(([name, tool]) => [
      name,
      {
        description: tool.description,
        parameters: tool.parameters,
        inputSchema: tool.inputSchema,
      },
    ])
  );
};

const normalizeThreadId = (threadId: string) => {
  const trimmed = threadId.trim();
  if (!trimmed) {
    return "";
  }
  return resolveRemoteThreadId(trimmed) || trimmed;
};

const parseMaybeJSON = (value: unknown): unknown => {
  if (typeof value !== "string") {
    return value;
  }
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  if (!(trimmed.startsWith("{") || trimmed.startsWith("["))) {
    return value;
  }
  try {
    return JSON.parse(trimmed);
  } catch {
    return value;
  }
};

const toStructuredPart = (part: unknown): UIMessagePart | null => {
  if (!part || typeof part !== "object") {
    return null;
  }
  const typed = part as Record<string, unknown>;
  const type = typeof typed.type === "string" ? typed.type : "";
  if (!type) {
    return null;
  }
  if (type === "image") {
    const image = typeof typed.image === "string" ? typed.image : "";
    if (!image) {
      return null;
    }
    return {
      type,
      data: {
        image,
        filename: typeof typed.filename === "string" ? typed.filename : "",
      },
    };
  }
  if (type === "file") {
    const data = typeof typed.data === "string" ? typed.data : "";
    const mimeType = typeof typed.mimeType === "string" ? typed.mimeType : "";
    if (!data) {
      return null;
    }
    return {
      type,
      data: {
        filename: typeof typed.filename === "string" ? typed.filename : "",
        data,
        mimeType,
      },
    };
  }
  if (type === "audio") {
    return {
      type,
      data: {
        audio: typed.audio ?? null,
      },
    };
  }
  return null;
};

export const toUIMessage = (message: ThreadMessage): UIMessage | null => {
  const role = message.role;
  if (role !== "user" && role !== "assistant" && role !== "system") {
    return null;
  }
  const parts: UIMessagePart[] = [];
  for (const part of message.content ?? []) {
    if (!part) {
      continue;
    }
    if (part.type === "text") {
      parts.push({ type: "text", text: part.text, parentId: part.parentId });
      continue;
    }
    if (part.type === "reasoning") {
      parts.push({ type: "reasoning", text: part.text, parentId: part.parentId });
      continue;
    }
    if (part.type === "source") {
      const url = typeof part.url === "string" ? part.url.trim() : "";
      if (!url) {
        continue;
      }
      parts.push({
        type: "source",
        parentId: part.parentId,
        data: {
          sourceType: "url",
          id: typeof part.id === "string" && part.id.trim() ? part.id.trim() : url,
          url,
          title: typeof part.title === "string" ? part.title : "",
        },
      });
      continue;
    }
    if (part.type === "tool-call") {
      const args = part.args ?? {};
      const state = part.isError
        ? "output-error"
        : part.result !== undefined
        ? "output-available"
        : "input-available";
      parts.push({
        type: "tool-call",
        parentId: part.parentId,
        toolCallId: part.toolCallId,
        toolName: part.toolName,
        state,
        input: args,
        output: part.result,
        errorText: part.isError ? JSON.stringify(part.result ?? "tool error") : undefined,
      });
      continue;
    }
    if (part.type === "data") {
      parts.push({
        type: "data",
        data: {
          name: part.name,
          data: parseMaybeJSON(part.data),
        },
      });
      continue;
    }
    if (part.type === "image" || part.type === "file" || part.type === "audio") {
      const structured = toStructuredPart(part);
      if (structured) {
        parts.push(structured);
      }
      continue;
    }
  }
  if (role === "user") {
    for (const attachment of message.attachments ?? []) {
      for (const attachmentPart of attachment.content ?? []) {
        if (attachmentPart.type === "text") {
          parts.push({ type: "text", text: attachmentPart.text });
          continue;
        }
        const structured = toStructuredPart(attachmentPart);
        if (structured) {
          if (structured.type === "file" && typeof structured.data === "object" && structured.data) {
            const data = structured.data as Record<string, unknown>;
            if (!data.filename && attachment.name) {
              data.filename = attachment.name;
            }
            structured.data = data;
          }
          parts.push(structured);
        }
      }
    }
  }
  return {
    id: message.id,
    role,
    parts,
  };
};

export const toUIMessages = (messages: readonly ThreadMessage[]) =>
  messages.map(toUIMessage).filter(Boolean) as UIMessage[];

const normalizeChatErrorText = (error: unknown) => {
  const raw =
    typeof error === "string"
      ? error
      : error instanceof Error
      ? error.message
      : String(error ?? "");
  const trimmed = raw.trim() || "chat request failed";
  const compact = trimmed.replace(/\s+/g, " ");
  const lower = compact.toLowerCase();
  if (lower.includes("context deadline exceeded") || lower.includes("client.timeout")) {
    return `Request timed out while waiting for model response. ${compact}`;
  }
  return compact;
};

const shouldUseGatewayStream = () => {
  const state = useSettingsStore.getState();
  if (state.isLoading) {
    return true;
  }
  const settings = state.settings;
  if (!settings?.gateway) {
    return false;
  }
  return Boolean(settings.gateway.controlPlaneEnabled);
};

const buildRunErrorResult = (error: unknown): ChatModelRunResult => ({
  status: {
    type: "incomplete",
    reason: "error",
    error: normalizeChatErrorText(error),
  },
});

type AsyncQueue<T> = {
  push: (value: T) => void;
  close: () => void;
  next: () => Promise<IteratorResult<T>>;
};

const createAsyncQueue = <T,>(): AsyncQueue<T> => {
  const values: T[] = [];
  let resolver: ((value: IteratorResult<T>) => void) | null = null;
  let closed = false;

  return {
    push: (value) => {
      if (closed) {
        return;
      }
      if (resolver) {
        resolver({ value, done: false });
        resolver = null;
        return;
      }
      values.push(value);
    },
    close: () => {
      if (closed) {
        return;
      }
      closed = true;
      if (resolver) {
        resolver({ value: undefined as T, done: true });
        resolver = null;
      }
    },
    next: () => {
      if (values.length > 0) {
        return Promise.resolve({ value: values.shift() as T, done: false });
      }
      if (closed) {
        return Promise.resolve({ value: undefined as T, done: true });
      }
      return new Promise<IteratorResult<T>>((resolve) => {
        resolver = resolve;
      });
    },
  };
};

const normalizeGatewayPayload = (payload: unknown): ChatEvent | null => {
  if (!payload) {
    return null;
  }
  if (typeof payload === "string") {
    try {
      return JSON.parse(payload) as ChatEvent;
    } catch {
      return null;
    }
  }
  if (typeof payload === "object") {
    return payload as ChatEvent;
  }
  return null;
};

const createGatewayEventQueue = (threadId: string, abortSignal: AbortSignal) => {
  const queue = createAsyncQueue<GatewayEvent>();
  let stopped = false;
  const unsubscribe = subscribeGatewayEvents((event) => {
    if (stopped) {
      return;
    }
    if (event.sessionId && event.sessionId !== threadId) {
      return;
    }
    queue.push(event);
  });
  const onAbort = () => stop();
  const stop = () => {
    if (stopped) {
      return;
    }
    stopped = true;
    abortSignal.removeEventListener("abort", onAbort);
    unsubscribe();
    queue.close();
  };
  abortSignal.addEventListener("abort", onAbort, { once: true });
  return { queue, stop };
};

async function* streamChatUpdatesViaGateway(
  queue: AsyncQueue<GatewayEvent>,
  threadId: string
): AsyncGenerator<ChatModelRunResult> {
  const parserState = createStreamParserState();
  let activeRunId = "";

  while (true) {
    const { value, done } = await queue.next();
    if (done) {
      break;
    }
    const gatewayUpdate = applyGatewayEvent(parserState, value.event, value.payload);
    if (gatewayUpdate.errorText) {
      yield buildRunErrorResult(gatewayUpdate.errorText);
      break;
    }
    if (gatewayUpdate.content || gatewayUpdate.status) {
      yield {
        ...(gatewayUpdate.content ? { content: gatewayUpdate.content } : {}),
        ...(gatewayUpdate.status ? { status: gatewayUpdate.status } : {}),
      };
    }
    if (gatewayUpdate.done) {
      break;
    }
    if (gatewayUpdate.content || gatewayUpdate.status) {
      continue;
    }
    const payload = normalizeGatewayPayload(value.payload);
    if (!payload || typeof payload.type !== "string") {
      continue;
    }
    const update = applyStreamEvent(parserState, payload);
    if (update.runId && !activeRunId) {
      activeRunId = update.runId;
    }
    if (activeRunId && update.runId && update.runId !== activeRunId) {
      continue;
    }
    if (update.agentEvent?.event === "context_snapshot" && update.agentEvent.contextTokens) {
      const tokens = update.agentEvent.contextTokens;
      useChatRuntimeStore.getState().setContextTokens(threadId, {
        promptTokens: tokens.promptTokens ?? 0,
        totalTokens: tokens.totalTokens ?? 0,
        contextWindowTokens: tokens.contextLimitTokens ?? undefined,
        warnTokens: tokens.warnTokens ?? undefined,
        hardTokens: tokens.hardTokens ?? undefined,
        contextFresh: true,
        updatedAt: Date.now(),
      });
    }
    if (update.errorText) {
      yield buildRunErrorResult(update.errorText);
      break;
    }
    if (update.content || update.status) {
      yield {
        ...(update.content ? { content: update.content } : {}),
        ...(update.status ? { status: update.status } : {}),
      };
    }
    if (update.done) {
      break;
    }
  }
}

export const createChatModelAdapter = (_httpBaseUrl: string): ChatModelAdapter => {
  return {
    async *run({ messages, runConfig, abortSignal, context }) {
      const custom = runConfig?.custom ?? {};
      const threadId =
        typeof custom.threadId === "string" ? custom.threadId : "";
      const resolvedThreadId = normalizeThreadId(threadId);
      if (!resolvedThreadId) {
        throw new Error("thread id is required");
      }
      let abortRequested = false;
      const requestAbortOnce = () => {
        if (abortRequested) {
          return;
        }
        abortRequested = true;
        void requestThreadAbort(resolvedThreadId, "user cancelled");
      };
      const onAbort = () => {
        if (shouldAbortThreadForSignal(abortSignal)) {
          requestAbortOnce();
        }
      };
      abortSignal.addEventListener("abort", onAbort, { once: true });

      const runtimeDefaults = useChatRuntimeStore.getState();
      const assistantIdRaw =
        typeof custom.assistantId === "string" ? custom.assistantId : runtimeDefaults.assistantId ?? "";
      const assistantId = assistantIdRaw.trim();
      const runtimeRequest: Record<string, unknown> = {
        sessionId: resolvedThreadId,
        sessionKey: resolvedThreadId,
        agentId: typeof custom.agentId === "string" ? custom.agentId : "",
        assistantId,
        input: {
          messages: toUIMessages(messages),
          replaceHistory: Boolean(custom.replaceHistory),
        },
        tools: {
          mode: typeof custom.toolsAuto === "boolean" && !custom.toolsAuto ? "off" : "auto",
          allowList: Array.isArray(custom.toolsList) ? custom.toolsList : [],
        },
        metadata: {
          channel: "aui",
          usageSource: "dialogue",
          queueMode: typeof custom.queueMode === "string" ? custom.queueMode : "",
        },
      };
      if (context?.system) {
        runtimeRequest.metadata = {
          ...(runtimeRequest.metadata as Record<string, unknown>),
          system: context.system,
        };
      }
      const toolsPayload = toToolPayload(context?.tools ?? undefined);
      if (toolsPayload) {
        runtimeRequest.metadata = {
          ...(runtimeRequest.metadata as Record<string, unknown>),
          frontendTools: toolsPayload,
        };
      }
      const callSettings = context?.callSettings ?? {};
      if (callSettings.temperature != null) {
        runtimeRequest.metadata = {
          ...(runtimeRequest.metadata as Record<string, unknown>),
          temperature: callSettings.temperature,
        };
      }
      if (callSettings.maxTokens != null) {
        runtimeRequest.metadata = {
          ...(runtimeRequest.metadata as Record<string, unknown>),
          maxTokens: callSettings.maxTokens,
        };
      }

      const gatewayEnabled = shouldUseGatewayStream();
      if (!gatewayEnabled) {
        messageBus.publishToast({
          title: "Gateway disabled",
          description: "Gateway control plane is disabled. Enable it in Settings to send messages.",
          intent: "warning",
        });
        yield buildRunErrorResult("gateway control plane disabled");
        return;
      }

      let gatewayQueue: ReturnType<typeof createGatewayEventQueue> | null = null;
      try {
        await startGateway();
        gatewayQueue = createGatewayEventQueue(resolvedThreadId, abortSignal);
      } catch (error) {
        throw new Error(`gateway connect failed: ${String(error ?? "")}`);
      }

      try {
        if (abortSignal.aborted) {
          if (shouldAbortThreadForSignal(abortSignal)) {
            requestAbortOnce();
          }
          return;
        }
        await requestGateway("runtime.run", runtimeRequest);
        if (!gatewayQueue) {
          yield buildRunErrorResult("gateway stream unavailable");
          return;
        }
        yield* streamChatUpdatesViaGateway(gatewayQueue.queue, resolvedThreadId);
        gatewayQueue.stop();
        return;
      } catch (error) {
        if (error instanceof Error && error.name === "AbortError") {
          throw error;
        }
        gatewayQueue?.stop();
        yield buildRunErrorResult(error);
      } finally {
        abortSignal.removeEventListener("abort", onAbort);
      }
    },
  };
};
