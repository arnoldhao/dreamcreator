import { describe, expect, test } from "bun:test"

import type { WorkspaceResolvedSubtitleRow } from "../components/workspace/types"
import { buildSubtitleRows, parseCueTime, resolveCurrentRow } from "../components/workspace/utils"

function buildRow(id: string, startMs: number, endMs: number, index: number): WorkspaceResolvedSubtitleRow {
  return {
    id,
    index,
    start: "",
    end: "",
    startMs,
    endMs,
    durationMs: endMs - startMs,
    sourceText: id,
    durationLabel: "",
    translationText: "",
    qaIssues: [],
    status: "ready",
    edited: false,
    metrics: {
      cps: 0,
      cpl: 0,
      characters: 0,
      lineCount: 0,
    },
  }
}

describe("resolveCurrentRow", () => {
  const rows = [
    buildRow("cue-1", 0, 1000, 1),
    buildRow("cue-2", 1400, 2200, 2),
    buildRow("cue-3", 2200, 3000, 3),
  ]

  test("returns the active row, next row in a gap, and last row after playback ends", () => {
    expect(resolveCurrentRow(rows, -10)?.id).toBe("cue-1")
    expect(resolveCurrentRow(rows, 500)?.id).toBe("cue-1")
    expect(resolveCurrentRow(rows, 1000)?.id).toBe("cue-2")
    expect(resolveCurrentRow(rows, 1800)?.id).toBe("cue-2")
    expect(resolveCurrentRow(rows, 2200)?.id).toBe("cue-3")
    expect(resolveCurrentRow(rows, 4000)?.id).toBe("cue-3")
  })

  test("returns null when there are no rows", () => {
    expect(resolveCurrentRow([], 500)).toBeNull()
  })
})

describe("subtitle time parsing", () => {
  test("parses SMPTE-style ITT cue times well enough for workspace duration math", () => {
    expect(parseCueTime("00:00:30:22")).toBe(30_733)
    expect(parseCueTime("00:00:34:22")).toBe(34_733)
  })

  test("buildSubtitleRows computes duration from normalized or SMPTE-like cue times", () => {
    const rows = buildSubtitleRows({
      format: "itt",
      cues: [
        {
          index: 1,
          start: "00:00:30:22",
          end: "00:00:34:22",
          text: "Hello",
        },
      ],
    })

    expect(rows).toHaveLength(1)
    expect(rows[0]?.durationMs).toBe(4000)
  })
})
