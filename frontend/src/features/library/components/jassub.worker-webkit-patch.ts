const JASSUB_WORKER_NOISE_PATTERNS = [/^JASSUB: fontselect:/]

function installJassubWorkerLogFilter() {
  const originalDebug = console.debug.bind(console)

  console.debug = (...args: unknown[]) => {
    const firstArg = args[0]
    if (
      typeof firstArg === "string" &&
      JASSUB_WORKER_NOISE_PATTERNS.some((pattern) => pattern.test(firstArg))
    ) {
      return
    }
    originalDebug(...args)
  }
}

function shouldDisableWebGL2ForCurrentRuntime() {
  const userAgent = navigator.userAgent || ""
  const isAppleWebKit = /AppleWebKit/i.test(userAgent)
  const isChromiumLike = /(Chrome|Chromium|CriOS|Edg|OPR)/i.test(userAgent)
  return isAppleWebKit && !isChromiumLike
}

function patchOffscreenCanvasForWebKitWebGL2Fallback() {
  if (
    typeof OffscreenCanvas === "undefined" ||
    typeof OffscreenCanvas.prototype.getContext !== "function"
  ) {
    return
  }

  if (!shouldDisableWebGL2ForCurrentRuntime()) {
    return
  }

  const originalGetContext = OffscreenCanvas.prototype.getContext as (
    this: OffscreenCanvas,
    contextId: OffscreenRenderingContextId,
    options?: unknown,
  ) => OffscreenRenderingContext | null

  OffscreenCanvas.prototype.getContext = function patchedGetContext(
    this: OffscreenCanvas,
    contextId: OffscreenRenderingContextId,
    options?: unknown,
  ) {
    if (contextId === "webgl2") {
      return null
    }
    return originalGetContext.call(this, contextId, options)
  } as typeof OffscreenCanvas.prototype.getContext
}

patchOffscreenCanvasForWebKitWebGL2Fallback()
installJassubWorkerLogFilter()
