import { describe, expect, test } from "bun:test";

import { applyGatewayEvent, applyStreamEvent, createStreamParserState } from "./stream-parser";

describe("stream parser", () => {
  test("builds tool calls incrementally and flushes sources on finish", () => {
    const state = createStreamParserState();

    let update = applyStreamEvent(state, {
      type: "tool-input-start",
      toolCallId: "call_1",
      toolName: "web.search",
    });

    expect(update.content?.[0]).toMatchObject({
      type: "tool-call",
      toolCallId: "call_1",
      toolName: "web.search",
    });

    update = applyStreamEvent(state, {
      type: "tool-input-delta",
      toolCallId: "call_1",
      toolName: "web.search",
      inputTextDelta: '{"q":"dreamcreator"}',
    });

    expect(update.content?.[0]).toMatchObject({
      type: "tool-call",
      argsText: '{"q":"dreamcreator"}',
      args: { q: "dreamcreator" },
    });

    update = applyStreamEvent(state, {
      type: "tool-output-available",
      toolCallId: "call_1",
      toolName: "web.search",
      output: {
        results: [{ url: "https://example.com/docs", title: "DreamCreator Docs" }],
      },
    });

    expect(update.content?.some((part) => part.type === "source")).toBe(false);

    update = applyStreamEvent(state, {
      type: "finish",
    });

    expect(update.done).toBe(true);
    expect(update.content?.find((part) => part.type === "source")).toMatchObject({
      type: "source",
      id: "call_1-1",
      url: "https://example.com/docs",
      title: "DreamCreator Docs",
    });
  });

  test("tracks approval interrupts through request and resolution", () => {
    const state = createStreamParserState();

    applyStreamEvent(state, {
      type: "tool-input-start",
      toolCallId: "call_2",
      toolName: "shell_command",
    });
    applyStreamEvent(state, {
      type: "tool-input-delta",
      toolCallId: "call_2",
      toolName: "shell_command",
      inputTextDelta: '{"command":"echo hi"}',
    });

    let update = applyGatewayEvent(state, "exec.approval.requested", {
      id: "approval_1",
      toolCallId: "call_2",
      toolName: "shell_command",
      args: '{"command":"echo hi"}',
      status: "pending",
    });

    expect(update.status).toEqual({
      type: "requires-action",
      reason: "interrupt",
    });
    expect(update.content?.find((part) => part.type === "tool-call")).toMatchObject({
      type: "tool-call",
      toolCallId: "call_2",
      interrupt: {
        type: "human",
      },
    });

    update = applyGatewayEvent(state, "exec.approval.resolved", {
      id: "approval_1",
    });

    expect(update.status).toEqual({
      type: "running",
    });
    const resolvedTool = update.content?.find((part) => part.type === "tool-call");
    expect(resolvedTool).toMatchObject({
      type: "tool-call",
      toolCallId: "call_2",
    });
    expect(resolvedTool?.interrupt).toBeUndefined();
  });

  test("extracts run ids from start and runtime events", () => {
    const state = createStreamParserState();

    let update = applyStreamEvent(state, {
      type: "start",
      messageMetadata: { runId: "run_123" },
    });
    expect(update.runId).toBe("run_123");

    update = applyStreamEvent(state, {
      type: "data-runtime-interrupt",
      data: JSON.stringify({ runId: "run_456", reason: "approval" }),
    });

    expect(update).toMatchObject({
      runId: "run_456",
      dataEventName: "runtime-interrupt",
      dataEventPayload: {
        runId: "run_456",
        reason: "approval",
      },
    });
  });
});
