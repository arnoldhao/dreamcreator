import * as React from "react"

const INVALID_PROGRESS_SPEEDS = new Set([
  "",
  "-",
  "--",
  "unknown",
  "n/a",
  "na",
  "null",
  "none",
])

function hashSeed(seed: string, durationMs: number) {
  if (!seed) {
    return 0
  }
  let hash = 0
  for (let index = 0; index < seed.length; index += 1) {
    hash = (hash * 31 + seed.charCodeAt(index)) >>> 0
  }
  return hash % durationMs
}

export function useTimeSyncedSpinDelay(seed: string, durationMs = 1200) {
  return React.useMemo(() => {
    const phase = (Date.now() + hashSeed(seed, durationMs)) % durationMs
    return `-${phase}ms`
  }, [durationMs, seed])
}

export function normalizeProgressSpeed(value?: string | null) {
  if (typeof value !== "string") {
    return null
  }
  const trimmed = value.trim()
  if (!trimmed) {
    return null
  }
  return INVALID_PROGRESS_SPEEDS.has(trimmed.toLowerCase()) ? null : trimmed
}

type SmoothedProgressSpeedOptions = {
  enabled?: boolean
  invalidSampleLimit?: number
}

type ProgressSpeedState = {
  lastValidSpeed: string | null
  invalidSamples: number
}

export function useSmoothedProgressSpeed(
  rawSpeed?: string | null,
  sampleKey = "",
  options: SmoothedProgressSpeedOptions = {},
) {
  const enabled = options.enabled ?? true
  const invalidSampleLimit = options.invalidSampleLimit ?? 3
  const normalizedSpeed = normalizeProgressSpeed(rawSpeed)
  const [state, setState] = React.useState<ProgressSpeedState>(() => ({
    lastValidSpeed: normalizedSpeed,
    invalidSamples: normalizedSpeed ? 0 : invalidSampleLimit,
  }))

  React.useEffect(() => {
    setState((current) => {
      if (!enabled) {
        if (!normalizedSpeed && current.lastValidSpeed === null && current.invalidSamples === invalidSampleLimit) {
          return current
        }
        return {
          lastValidSpeed: normalizedSpeed,
          invalidSamples: normalizedSpeed ? 0 : invalidSampleLimit,
        }
      }

      if (normalizedSpeed) {
        if (current.lastValidSpeed === normalizedSpeed && current.invalidSamples === 0) {
          return current
        }
        return {
          lastValidSpeed: normalizedSpeed,
          invalidSamples: 0,
        }
      }

      if (!current.lastValidSpeed) {
        if (current.invalidSamples === invalidSampleLimit) {
          return current
        }
        return {
          lastValidSpeed: null,
          invalidSamples: invalidSampleLimit,
        }
      }

      const nextInvalidSamples = current.invalidSamples + 1
      if (nextInvalidSamples >= invalidSampleLimit) {
        return {
          lastValidSpeed: null,
          invalidSamples: nextInvalidSamples,
        }
      }

      return {
        lastValidSpeed: current.lastValidSpeed,
        invalidSamples: nextInvalidSamples,
      }
    })
  }, [enabled, invalidSampleLimit, normalizedSpeed, sampleKey])

  if (!enabled) {
    return normalizedSpeed
  }

  return normalizedSpeed ?? (state.invalidSamples < invalidSampleLimit ? state.lastValidSpeed : null)
}
