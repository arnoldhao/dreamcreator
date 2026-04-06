import type {
  ChatModelRunResult,
  MessageStatus,
  ThreadAssistantMessagePart,
  ToolCallMessagePart,
} from "@assistant-ui/react";
import type { ReadonlyJSONObject } from "assistant-stream/utils";

export type ChatEvent = {
  type?: string;
  id?: string;
  messageId?: string;
  errorText?: string;
  data?: unknown;
  messageMetadata?: Record<string, unknown>;
  delta?: string;
  toolCallId?: string;
  toolName?: string;
  inputTextDelta?: string;
  output?: unknown;
};

export type AgentEvent = {
  event?: string;
  runId?: string;
  threadId?: string;
  messageId?: string;
  step?: number;
  delta?: string;
  toolCallId?: string;
  toolName?: string;
  toolArgs?: unknown;
  toolArgsDelta?: string;
  toolOutput?: unknown;
  errorText?: string;
  finishReason?: string;
  metadata?: Record<string, unknown>;
  contextTokens?: {
    promptTokens?: number;
    totalTokens?: number;
    contextLimitTokens?: number;
    warnTokens?: number;
    hardTokens?: number;
  };
};

type ToolState = {
  toolCallId: string;
  toolName: string;
  argsText: string;
  args?: unknown;
  result?: unknown;
  isError?: boolean;
  interrupt?: ToolCallMessagePart["interrupt"];
};

type SourceState = {
  id: string;
  url: string;
  title?: string;
};

type TextTimelineBlock = {
  id: string;
  type: "text";
  text: string;
};

type ReasoningTimelineBlock = {
  id: string;
  type: "reasoning";
  text: string;
};

type ToolTimelineBlock = {
  id: string;
  type: "tool";
  key: string;
  state: ToolState;
};

type SourceTimelineBlock = {
  id: string;
  type: "source";
  key: string;
  state: SourceState;
};

type TimelineBlock = TextTimelineBlock | ReasoningTimelineBlock | ToolTimelineBlock | SourceTimelineBlock;

export type StreamParserState = {
  blocks: TimelineBlock[];
  reasoningBlockIndex: number | null;
  toolBlockIndices: Map<string, number>;
  pendingSources: Map<string, SourceState>;
  pendingSourceOrder: string[];
  sourcesFlushed: boolean;
  blockCounter: number;
};

export type StreamParserUpdate = {
  done?: boolean;
  runId?: string;
  agentEvent?: AgentEvent;
  content?: ChatModelRunResult["content"];
  status?: MessageStatus;
  errorText?: string;
  dataEventName?: string;
  dataEventPayload?: unknown;
};

const toReadonlyJSONObject = (value: unknown): ReadonlyJSONObject => {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as ReadonlyJSONObject;
  }
  return {} as ReadonlyJSONObject;
};

const nextBlockID = (state: StreamParserState) => {
  state.blockCounter += 1;
  return `timeline-${state.blockCounter}`;
};

const buildToolPart = (state: ToolState, parentId: string): ToolCallMessagePart => {
  const args = toReadonlyJSONObject(state.args);
  const argsText = state.argsText || JSON.stringify(args);
  return {
    type: "tool-call",
    toolCallId: state.toolCallId,
    toolName: state.toolName,
    args,
    argsText,
    result: state.result,
    isError: state.isError,
    interrupt: state.interrupt,
    parentId,
  };
};

const buildContent = (blocks: TimelineBlock[]): ThreadAssistantMessagePart[] => {
  const content: ThreadAssistantMessagePart[] = [];
  for (const block of blocks) {
    if (block.type === "text") {
      if (!block.text) {
        continue;
      }
      content.push({
        type: "text",
        text: block.text,
        parentId: block.id,
      });
      continue;
    }
    if (block.type === "reasoning") {
      if (!block.text) {
        continue;
      }
      content.push({
        type: "reasoning",
        text: block.text,
        parentId: block.id,
      });
      continue;
    }
    if (block.type === "source") {
      if (!block.state.url) {
        continue;
      }
      content.push({
        type: "source",
        sourceType: "url",
        id: block.state.id || block.state.url,
        url: block.state.url,
        ...(block.state.title ? { title: block.state.title } : {}),
        parentId: block.id,
      });
      continue;
    }
    content.push(buildToolPart(block.state, block.id));
  }
  return content;
};

const parseDataPayload = (value: unknown): unknown => {
  if (typeof value === "string") {
    try {
      return JSON.parse(value);
    } catch {
      return value;
    }
  }
  return value;
};

const parseJsonObject = (value: string): unknown => {
  try {
    return JSON.parse(value);
  } catch {
    return undefined;
  }
};

const looksLikeHTTPURL = (value: string): boolean => {
  const trimmed = value.trim();
  if (!trimmed) {
    return false;
  }
  try {
    const parsed = new URL(trimmed);
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
};

const asRecord = (value: unknown): Record<string, unknown> | null => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  return value as Record<string, unknown>;
};

const firstString = (record: Record<string, unknown>, keys: string[]): string => {
  for (const key of keys) {
    const value = record[key];
    if (typeof value !== "string") {
      continue;
    }
    const trimmed = value.trim();
    if (trimmed) {
      return trimmed;
    }
  }
  return "";
};

const normalizeSourceKey = (url: string) => url.trim().toLowerCase();

const buildSourceID = (toolCallId: string, toolName: string, index: number) => {
  const base = toolCallId.trim() || toolName.trim() || "source";
  return `${base}-${Math.max(1, index)}`;
};

const extractSourceFromRecord = (
  record: Record<string, unknown>,
  toolCallId: string,
  toolName: string,
  index: number
): SourceState | null => {
  const url =
    firstString(record, ["url", "href", "link", "sourceUrl", "sourceURL", "uri"]) || "";
  if (!looksLikeHTTPURL(url)) {
    return null;
  }
  const id = firstString(record, ["id", "sourceId", "sourceID", "ref"]) || buildSourceID(toolCallId, toolName, index);
  const title = firstString(record, ["title", "name", "label"]);
  return {
    id,
    url,
    ...(title ? { title } : {}),
  };
};

const extractSourcesFromToolOutput = (
  value: unknown,
  toolCallId: string,
  toolName: string
): SourceState[] => {
  const result: SourceState[] = [];
  const seen = new Set<string>();
  const queue: Array<{ value: unknown; depth: number }> = [{ value, depth: 0 }];

  while (queue.length > 0) {
    const current = queue.shift();
    if (!current || current.depth > 5 || current.value == null) {
      continue;
    }
    if (typeof current.value === "string") {
      const trimmed = current.value.trim();
      if (!looksLikeHTTPURL(trimmed)) {
        continue;
      }
      const key = normalizeSourceKey(trimmed);
      if (seen.has(key)) {
        continue;
      }
      seen.add(key);
      result.push({
        id: buildSourceID(toolCallId, toolName, result.length + 1),
        url: trimmed,
      });
      continue;
    }
    if (Array.isArray(current.value)) {
      for (const item of current.value) {
        queue.push({ value: item, depth: current.depth + 1 });
      }
      continue;
    }
    const record = asRecord(current.value);
    if (!record) {
      continue;
    }
    const source = extractSourceFromRecord(record, toolCallId, toolName, result.length + 1);
    if (source) {
      const key = normalizeSourceKey(source.url);
      if (!seen.has(key)) {
        seen.add(key);
        result.push(source);
      }
    }
    for (const key of ["results", "sources", "items", "documents", "data", "value"]) {
      if (key in record) {
        queue.push({ value: record[key], depth: current.depth + 1 });
      }
    }
  }

  return result;
};

const queueSourcesFromToolOutput = (
  state: StreamParserState,
  toolOutput: unknown,
  toolCallId: string,
  toolName: string
) => {
  const sources = extractSourcesFromToolOutput(toolOutput, toolCallId, toolName);
  if (sources.length === 0) {
    return false;
  }
  let updated = false;
  for (const source of sources) {
    const key = normalizeSourceKey(source.url);
    if (!key || state.pendingSources.has(key)) {
      continue;
    }
    state.pendingSources.set(key, source);
    state.pendingSourceOrder.push(key);
    updated = true;
  }
  return updated;
};

const flushPendingSources = (state: StreamParserState): boolean => {
  if (state.sourcesFlushed || state.pendingSourceOrder.length === 0) {
    return false;
  }
  state.sourcesFlushed = true;
  const parentID = nextBlockID(state);
  for (const key of state.pendingSourceOrder) {
    const source = state.pendingSources.get(key);
    if (!source) {
      continue;
    }
    state.blocks.push({
      id: parentID,
      type: "source",
      key,
      state: source,
    });
  }
  return true;
};

const parseRunIdFromPayload = (value: unknown): string => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return "";
  }
  const record = value as Record<string, unknown>;
  const runId = record.runId;
  if (typeof runId === "string") {
    return runId.trim();
  }
  return "";
};

const parseRunIdFromMetadata = (value: unknown) => {
  if (!value || typeof value !== "object") {
    return "";
  }
  const record = value as Record<string, unknown>;
  const runId = record.runId;
  if (typeof runId === "string") {
    return runId.trim();
  }
  return "";
};

const parseAgentEvent = (value: unknown): AgentEvent | null => {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return null;
  }
  const record = value as Record<string, unknown>;
  const eventName = typeof record.event === "string" ? record.event.trim() : "";
  if (!eventName) {
    return null;
  }
  return record as AgentEvent;
};

const appendTextDelta = (state: StreamParserState, delta: string): boolean => {
  if (!delta) {
    return false;
  }
  const last = state.blocks[state.blocks.length - 1];
  if (last && last.type === "text") {
    last.text += delta;
    return true;
  }
  state.blocks.push({
    id: nextBlockID(state),
    type: "text",
    text: delta,
  });
  return true;
};

const appendReasoningDelta = (state: StreamParserState, delta: string): boolean => {
  if (!delta) {
    return false;
  }
  if (state.reasoningBlockIndex != null) {
    const block = state.blocks[state.reasoningBlockIndex];
    if (block && block.type === "reasoning") {
      block.text += delta;
      return true;
    }
    state.reasoningBlockIndex = null;
  }
  const block: ReasoningTimelineBlock = {
    id: nextBlockID(state),
    type: "reasoning",
    text: delta,
  };
  state.blocks.push(block);
  state.reasoningBlockIndex = state.blocks.length - 1;
  return true;
};

const resolveToolKey = (toolCallId: string, toolName: string): string => {
  const callID = toolCallId.trim();
  if (callID) {
    return callID;
  }
  return toolName.trim();
};

const upsertToolState = (
  state: StreamParserState,
  key: string,
  patch: (Omit<Partial<ToolState>, "interrupt"> & Pick<ToolState, "toolCallId" | "toolName">) & {
    interrupt?: ToolCallMessagePart["interrupt"] | null;
  }
): boolean => {
  const normalizedKey = key.trim();
  if (!normalizedKey) {
    return false;
  }

  const existingIndex = state.toolBlockIndices.get(normalizedKey);
  if (existingIndex == null) {
    const blockID = nextBlockID(state);
    const nextState: ToolState = {
      toolCallId: patch.toolCallId || normalizedKey,
      toolName: patch.toolName,
      argsText: patch.argsText ?? "",
      args: patch.args ?? {},
      result: patch.result,
      isError: patch.isError,
      interrupt: patch.interrupt ?? undefined,
    };
    state.blocks.push({
      id: blockID,
      type: "tool",
      key: normalizedKey,
      state: nextState,
    });
    state.toolBlockIndices.set(normalizedKey, state.blocks.length - 1);
    return true;
  }

  const block = state.blocks[existingIndex];
  if (!block || block.type !== "tool") {
    return false;
  }
  const nextInterrupt = patch.interrupt === null
    ? undefined
    : patch.interrupt === undefined
      ? block.state.interrupt
      : patch.interrupt;
  block.state = {
    ...block.state,
    ...patch,
    toolCallId: patch.toolCallId || block.state.toolCallId || normalizedKey,
    toolName: patch.toolName || block.state.toolName,
    interrupt: nextInterrupt,
  };
  return true;
};

const applyAgentEvent = (
  state: StreamParserState,
  event: AgentEvent
): StreamParserUpdate => {
  const eventName = (event.event ?? "").trim();
  if (!eventName) {
    return {};
  }
  const runId = (event.runId ?? "").trim() || undefined;
  if (eventName === "run_error") {
    return {
      runId,
      agentEvent: event,
      errorText: event.errorText || "request failed",
    };
  }
  if (eventName === "run_end" || eventName === "run_abort") {
    const sourceFlushed = flushPendingSources(state);
    return {
      runId,
      agentEvent: event,
      done: true,
      ...(sourceFlushed ? { content: buildContent(state.blocks) } : {}),
    };
  }
  if (eventName === "run_start" || eventName === "context_snapshot") {
    return {
      runId,
      agentEvent: event,
    };
  }

  let updated = false;
  if (eventName === "text_delta") {
    updated = appendTextDelta(state, event.delta ?? "");
  }
  if (eventName === "reasoning_delta") {
    updated = appendReasoningDelta(state, event.delta ?? "");
  }
  if (eventName === "tool_call_start") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        argsText: previousToolState?.argsText ?? "",
        args: previousToolState?.args ?? {},
        result: previousToolState?.result,
        isError: previousToolState?.isError,
        interrupt: previousToolState?.interrupt,
      });
    }
  }
  if (eventName === "tool_call_delta") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key && event.toolArgsDelta) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      const nextArgsText = `${previousToolState?.argsText ?? ""}${event.toolArgsDelta}`;
      const parsed = parseJsonObject(nextArgsText);
      updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        argsText: nextArgsText,
        args: parsed ?? previousToolState?.args ?? {},
        result: previousToolState?.result,
        isError: previousToolState?.isError,
        interrupt: previousToolState?.interrupt,
      });
    }
  }
  if (eventName === "tool_call_ready") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      const args = event.toolArgs ?? previousToolState?.args ?? {};
      const argsText = typeof args === "string" ? args : JSON.stringify(args);
      updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        args,
        argsText,
        result: previousToolState?.result,
        isError: previousToolState?.isError,
        interrupt: previousToolState?.interrupt,
      });
    }
  }
  if (eventName === "tool_result") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        argsText: previousToolState?.argsText ?? "",
        args: previousToolState?.args ?? {},
        result: event.toolOutput,
        isError: false,
        interrupt: null,
      });
      if (queueSourcesFromToolOutput(state, event.toolOutput, toolCallId || key, toolName)) {
        updated = true;
      }
    }
  }
  if (eventName === "tool_error") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        argsText: previousToolState?.argsText ?? "",
        args: previousToolState?.args ?? {},
        result: { error: event.errorText || "tool failed" },
        isError: true,
        interrupt: null,
      });
    }
  }

  if (!updated) {
    return {
      runId,
      agentEvent: event,
    };
  }
  return {
    runId,
    agentEvent: event,
    content: buildContent(state.blocks),
  };
};

type ExecApprovalEventPayload = {
  id: string;
  toolCallId: string;
  toolName: string;
  action: string;
  args: string;
  status: string;
  decision: string;
  reason: string;
};

const normalizeComparableText = (value: string) =>
  value.replace(/\s+/g, " ").trim().toLowerCase();

const parseExecApprovalEventPayload = (value: unknown): ExecApprovalEventPayload | null => {
  const record = asRecord(value);
  if (!record) {
    return null;
  }
  const id = firstString(record, ["id", "approvalId", "approvalID"]);
  if (!id) {
    return null;
  }
  return {
    id,
    toolCallId: firstString(record, ["toolCallId", "tool_call_id", "callId"]),
    toolName: firstString(record, ["toolName", "tool", "name"]),
    action: firstString(record, ["action", "title"]),
    args: firstString(record, ["args", "input", "request"]),
    status: firstString(record, ["status"]),
    decision: firstString(record, ["decision"]),
    reason: firstString(record, ["reason"]),
  };
};

const parseApprovalArgs = (argsText: string): unknown => {
  const trimmed = argsText.trim();
  if (!trimmed) {
    return {};
  }
  const parsed = parseJsonObject(trimmed);
  if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
    return parsed;
  }
  return {};
};

const findToolKeyByApprovalID = (state: StreamParserState, approvalID: string): string => {
  const normalizedID = approvalID.trim();
  if (!normalizedID) {
    return "";
  }
  for (let index = state.blocks.length - 1; index >= 0; index -= 1) {
    const block = state.blocks[index];
    if (!block || block.type !== "tool") {
      continue;
    }
    const payload = asRecord(block.state.interrupt?.payload);
    const payloadID = payload ? firstString(payload, ["id", "approvalId", "approvalID"]) : "";
    if (payloadID === normalizedID) {
      return block.key;
    }
  }
  return "";
};

const findPendingToolKeyForApproval = (
  state: StreamParserState,
  toolName: string,
  argsText: string
): string => {
  const normalizedToolName = normalizeComparableText(toolName);
  const normalizedArgs = normalizeComparableText(argsText);
  for (let index = state.blocks.length - 1; index >= 0; index -= 1) {
    const block = state.blocks[index];
    if (!block || block.type !== "tool") {
      continue;
    }
    if (block.state.result !== undefined) {
      continue;
    }
    if (normalizedToolName) {
      const currentToolName = normalizeComparableText(block.state.toolName || "");
      if (currentToolName !== normalizedToolName) {
        continue;
      }
    }
    if (normalizedArgs && block.state.argsText) {
      const currentArgs = normalizeComparableText(block.state.argsText);
      if (currentArgs && currentArgs !== normalizedArgs) {
        continue;
      }
    }
    return block.key;
  }
  return "";
};

const resolveToolKeyForApproval = (
  state: StreamParserState,
  approval: ExecApprovalEventPayload
): string => {
  const byApprovalID = findToolKeyByApprovalID(state, approval.id);
  if (byApprovalID) {
    return byApprovalID;
  }
  if (approval.toolCallId) {
    const byCallID = resolveToolKey(approval.toolCallId, approval.toolName);
    if (byCallID) {
      return byCallID;
    }
  }
  return findPendingToolKeyForApproval(state, approval.toolName, approval.args);
};

const applyApprovalRequestedEvent = (
  state: StreamParserState,
  payload: ExecApprovalEventPayload
): StreamParserUpdate => {
  const fallbackKey = resolveToolKey(payload.toolCallId || payload.id, payload.toolName);
  const key = resolveToolKeyForApproval(state, payload) || fallbackKey;
  if (!key) {
    return {};
  }
  const existing = state.toolBlockIndices.has(key)
    ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
    : null;
  const previousToolState =
    existing && existing.type === "tool" ? existing.state : undefined;
  const argsText = payload.args || previousToolState?.argsText || "{}";
  const args = previousToolState?.args ?? parseApprovalArgs(argsText);
  const updated = upsertToolState(state, key, {
    toolCallId: payload.toolCallId || previousToolState?.toolCallId || key,
    toolName: payload.toolName || previousToolState?.toolName || "",
    argsText,
    args,
    result: previousToolState?.result,
    isError: previousToolState?.isError,
    interrupt: {
      type: "human",
      payload: {
        id: payload.id,
        toolCallId: payload.toolCallId,
        toolName: payload.toolName,
        action: payload.action,
        args: payload.args,
        status: payload.status || "pending",
      },
    },
  });
  if (!updated) {
    return {};
  }
  return {
    content: buildContent(state.blocks),
    status: { type: "requires-action", reason: "interrupt" },
  };
};

const applyApprovalResolvedEvent = (
  state: StreamParserState,
  payload: ExecApprovalEventPayload
): StreamParserUpdate => {
  const key = resolveToolKeyForApproval(state, payload);
  if (!key) {
    return {};
  }
  const existing = state.toolBlockIndices.has(key)
    ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
    : null;
  const previousToolState =
    existing && existing.type === "tool" ? existing.state : undefined;
  if (!previousToolState) {
    return {};
  }
  const updated = upsertToolState(state, key, {
    toolCallId: previousToolState.toolCallId || key,
    toolName: previousToolState.toolName,
    argsText: previousToolState.argsText,
    args: previousToolState.args,
    result: previousToolState.result,
    isError: previousToolState.isError,
    interrupt: null,
  });
  if (!updated) {
    return {};
  }
  return {
    content: buildContent(state.blocks),
    status: { type: "running" },
  };
};

export const applyGatewayEvent = (
  state: StreamParserState,
  eventName: string,
  payload: unknown
): StreamParserUpdate => {
  const normalizedEventName = eventName.trim().toLowerCase();
  if (normalizedEventName !== "exec.approval.requested" && normalizedEventName !== "exec.approval.resolved") {
    return {};
  }
  const parsedPayload = parseExecApprovalEventPayload(payload);
  if (!parsedPayload) {
    return {};
  }
  if (normalizedEventName === "exec.approval.requested") {
    return applyApprovalRequestedEvent(state, parsedPayload);
  }
  return applyApprovalResolvedEvent(state, parsedPayload);
};

export const createStreamParserState = (): StreamParserState => ({
  blocks: [],
  reasoningBlockIndex: null,
  toolBlockIndices: new Map<string, number>(),
  pendingSources: new Map<string, SourceState>(),
  pendingSourceOrder: [],
  sourcesFlushed: false,
  blockCounter: 0,
});

export const applyStreamEvent = (
  state: StreamParserState,
  event: ChatEvent
): StreamParserUpdate => {
  const type = (event.type ?? "").trim();
  if (!type) {
    return {};
  }
  if (type === "start") {
    const runId = parseRunIdFromMetadata(event.messageMetadata);
    return runId ? { runId } : {};
  }
  if (type === "finish") {
    const sourceFlushed = flushPendingSources(state);
    return {
      done: true,
      ...(sourceFlushed ? { content: buildContent(state.blocks) } : {}),
    };
  }
  if (type === "error") {
    return {
      errorText: event.errorText || "request failed",
    };
  }
  if (type === "text-delta") {
    if (appendTextDelta(state, event.delta ?? "")) {
      return {
        content: buildContent(state.blocks),
      };
    }
  }
  if (type === "tool-input-start") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      if (
        upsertToolState(state, key, {
          toolCallId: toolCallId || key,
          toolName: toolName || previousToolState?.toolName || "",
          argsText: previousToolState?.argsText ?? "",
          args: previousToolState?.args ?? {},
          result: previousToolState?.result,
          isError: previousToolState?.isError,
          interrupt: previousToolState?.interrupt,
        })
      ) {
        return {
          content: buildContent(state.blocks),
        };
      }
    }
  }
  if (type === "tool-input-delta") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key && event.inputTextDelta) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      const nextArgsText = `${previousToolState?.argsText ?? ""}${event.inputTextDelta}`;
      const parsed = parseJsonObject(nextArgsText);
      if (
        upsertToolState(state, key, {
          toolCallId: toolCallId || key,
          toolName: toolName || previousToolState?.toolName || "",
          argsText: nextArgsText,
          args: parsed ?? previousToolState?.args ?? {},
          result: previousToolState?.result,
          isError: previousToolState?.isError,
          interrupt: previousToolState?.interrupt,
        })
      ) {
        return {
          content: buildContent(state.blocks),
        };
      }
    }
  }
  if (type === "tool-output-available") {
    const toolCallId = (event.toolCallId ?? "").trim();
    const toolName = (event.toolName ?? "").trim();
    const key = resolveToolKey(toolCallId, toolName);
    if (key) {
      const previous = state.toolBlockIndices.has(key)
        ? state.blocks[state.toolBlockIndices.get(key) ?? -1]
        : null;
      const previousToolState =
        previous && previous.type === "tool" ? previous.state : undefined;
      const updated = upsertToolState(state, key, {
        toolCallId: toolCallId || key,
        toolName: toolName || previousToolState?.toolName || "",
        argsText: previousToolState?.argsText ?? "",
        args: previousToolState?.args ?? {},
        result: event.output,
        isError: false,
        interrupt: null,
      });
      const queued = queueSourcesFromToolOutput(state, event.output, toolCallId || key, toolName);
      if (updated || queued) {
        return {
          content: buildContent(state.blocks),
        };
      }
    }
  }
  if (type === "data-runtime-interrupt" || type === "data-runtime-resume") {
    const payload = parseDataPayload(event.data);
    const runId = parseRunIdFromPayload(payload);
    return {
      runId: runId || undefined,
      dataEventName: type.replace("data-", ""),
      dataEventPayload: payload,
    };
  }
  if (type === "data-agent-event") {
    const payload = parseDataPayload(event.data);
    const agentEvent = parseAgentEvent(payload);
    if (!agentEvent) {
      return {};
    }
    return applyAgentEvent(state, agentEvent);
  }
  return {};
};
