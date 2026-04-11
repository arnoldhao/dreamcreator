import * as React from "react";
import JASSUB from "jassub";
import workerUrl from "./jassub.worker.ts?worker&url";
import wasmUrl from "jassub/dist/wasm/jassub-worker.wasm?url";
import modernWasmUrl from "jassub/dist/wasm/jassub-worker-modern.wasm?url";

import { exportFontFamilies } from "@/shared/query/system";

type UseJassubPreviewParams = {
  debugLabel?: string;
  assContent: string;
  referencedFontFamilies?: string[];
  canvas: HTMLCanvasElement | null;
  video?: HTMLVideoElement | null;
  currentTimeSeconds?: number;
  enabled?: boolean;
  requireVideo?: boolean;
};

const DEFAULT_STATIC_PREVIEW_TIME_SECONDS = 1;
const JASSUB_DEBUG_PREFIX = "[jassub-preview]";
const JASSUB_VIDEO_COLOR_SPACE_MAP = {
  rgb: "RGB",
  bt709: "BT709",
  bt470bg: "BT601",
  smpte170m: "BT601",
} as const;

let hasPatchedJassubColorSpaceProbe = false;

type JassubVideoColorSpace =
  (typeof JASSUB_VIDEO_COLOR_SPACE_MAP)[keyof typeof JASSUB_VIDEO_COLOR_SPACE_MAP];
type PatchedJassubRenderer = {
  _setColorSpace: (colorSpace: JassubVideoColorSpace) => Promise<void>;
};
type PatchedJassubInstance = JASSUB & {
  _video?: HTMLVideoElement | null;
  renderer: PatchedJassubRenderer;
  ready: Promise<unknown>;
};
type PatchedJassubPrototype = typeof JASSUB.prototype & {
  _updateColorSpace?: (this: PatchedJassubInstance) => Promise<void>;
};

installJassubMainThreadPatches();

export function useJassubPreview({
  debugLabel = "default",
  assContent,
  referencedFontFamilies = [],
  canvas,
  video = null,
  currentTimeSeconds = DEFAULT_STATIC_PREVIEW_TIME_SECONDS,
  enabled = true,
  requireVideo = false,
}: UseJassubPreviewParams) {
  const instanceRef = React.useRef<JASSUB | null>(null);
  const boundCanvasRef = React.useRef<HTMLCanvasElement | null>(null);
  const boundVideoRef = React.useRef<HTMLVideoElement | null>(null);
  const loadedFontFamiliesRef = React.useRef<Set<string>>(new Set());
  // Keep exported font bytes scoped to this preview hook so the cache is
  // released when the preview itself is disabled or unmounted.
  const fontAssetCacheRef = React.useRef<Map<string, Promise<Uint8Array[]>>>(
    new Map(),
  );
  const fallbackFontFamily = React.useMemo(
    () => resolvePlatformJassubFallbackFontFamily(),
    [],
  );
  const assDeclaredFontFamilies = React.useMemo(
    () => extractASSDeclaredFontFamilies(assContent),
    [assContent],
  );

  const normalizedFontFamilies = React.useMemo(
    () =>
      dedupeFontFamilies([
        ...referencedFontFamilies,
        ...assDeclaredFontFamilies,
        ...(fallbackFontFamily ? [fallbackFontFamily] : []),
      ]),
    [assDeclaredFontFamilies, fallbackFontFamily, referencedFontFamilies],
  );
  const fontFamilyKey = React.useMemo(
    () => normalizedFontFamilies.map(normalizeFontFamilyKey).join("\u0000"),
    [normalizedFontFamilies],
  );
  const normalizedASSContent = React.useMemo(
    () => assContent.trim(),
    [assContent],
  );

  React.useEffect(() => {
    return () => {
      const instance = instanceRef.current;
      debugLog(debugLabel, "cleanup requested", {
        hasInstance: Boolean(instance),
        canvas: describeCanvas(boundCanvasRef.current),
        video: describeVideo(boundVideoRef.current),
      });
      instanceRef.current = null;
      boundCanvasRef.current = null;
      boundVideoRef.current = null;
      loadedFontFamiliesRef.current = new Set();
      fontAssetCacheRef.current.clear();
      if (!instance) {
        return;
      }
      void instance.destroy().catch((error) => {
        debugError(debugLabel, "destroy failed during cleanup", error);
      });
    };
  }, [debugLabel]);

  React.useEffect(() => {
    const activeCanvas = canvas;
    const activeVideo = video ?? null;
    const missingRequiredVideo = requireVideo && !activeVideo;
    if (
      !enabled ||
      !activeCanvas ||
      !normalizedASSContent ||
      missingRequiredVideo
    ) {
      debugLog(debugLabel, "preview sync skipped", {
        enabled,
        requireVideo,
        missingRequiredVideo,
        hasCanvas: Boolean(activeCanvas),
        hasVideo: Boolean(activeVideo),
        assLength: normalizedASSContent.length,
        fontFamilies: normalizedFontFamilies,
        canvas: describeCanvas(activeCanvas),
        video: describeVideo(activeVideo),
      });
      const existing = instanceRef.current;
      instanceRef.current = null;
      boundCanvasRef.current = null;
      boundVideoRef.current = null;
      loadedFontFamiliesRef.current = new Set();
      fontAssetCacheRef.current.clear();
      if (existing) {
        debugLog(
          debugLabel,
          "destroying existing instance because prerequisites are missing",
        );
        void existing.destroy().catch((error) => {
          debugError(
            debugLabel,
            "destroy failed while disabling preview",
            error,
          );
        });
      }
      return;
    }

    let cancelled = false;

    const syncPreview = async () => {
      debugLog(debugLabel, "preview sync start", {
        assLength: normalizedASSContent.length,
        fontFamilies: normalizedFontFamilies,
        canvas: describeCanvas(activeCanvas),
        video: describeVideo(activeVideo),
        currentTimeSeconds,
      });

      const fontAssetsByFamily = await ensureFontAssets(
        debugLabel,
        normalizedFontFamilies,
        fontAssetCacheRef.current,
      );
      if (cancelled) {
        debugWarn(debugLabel, "preview sync cancelled after font load");
        return;
      }
      debugLog(
        debugLabel,
        "font assets ready",
        summarizeFontAssets(fontAssetsByFamily),
      );

      let instance = instanceRef.current;
      const shouldRecreate =
        !instance ||
        boundCanvasRef.current !== activeCanvas ||
        boundVideoRef.current !== activeVideo;

      if (shouldRecreate) {
        if (instance) {
          debugLog(
            debugLabel,
            "destroying stale JASSUB instance before recreate",
          );
          await instance.destroy().catch((error) => {
            debugError(debugLabel, "destroy failed before recreate", error);
          });
        }
        const defaultFont = resolveDefaultFontFamily(
          fontAssetsByFamily,
          normalizedFontFamilies,
        );
        debugLog(debugLabel, "creating JASSUB instance", {
          defaultFont,
          fontAssetCount: flattenFontAssets(fontAssetsByFamily).length,
          canvas: describeCanvas(activeCanvas),
          video: describeVideo(activeVideo),
        });
        instance = new JASSUB({
          canvas: activeCanvas,
          video: activeVideo ?? undefined,
          subContent: normalizedASSContent,
          workerUrl,
          wasmUrl,
          modernWasmUrl,
          fonts: flattenFontAssets(fontAssetsByFamily),
          defaultFont,
          queryFonts: false,
          debug: false,
        });
        const internalWorker = (instance as unknown as { _worker?: Worker })
          ._worker;
        if (internalWorker) {
          internalWorker.addEventListener("error", (event) => {
            debugError(debugLabel, "JASSUB worker error event", event);
          });
          internalWorker.addEventListener("messageerror", (event) => {
            debugError(debugLabel, "JASSUB worker messageerror event", event);
          });
        } else {
          debugWarn(
            debugLabel,
            "JASSUB internal worker handle unavailable for debug logging",
          );
        }
        instanceRef.current = instance;
        boundCanvasRef.current = activeCanvas;
        boundVideoRef.current = activeVideo;
        loadedFontFamiliesRef.current = new Set(
          normalizedFontFamilies
            .filter(
              (family) => (fontAssetsByFamily.get(family) ?? []).length > 0,
            )
            .map(normalizeFontFamilyKey),
        );
        const readyTimeout = window.setTimeout(() => {
          debugWarn(debugLabel, "JASSUB ready is still pending after timeout", {
            canvas: describeCanvas(activeCanvas),
            video: describeVideo(activeVideo),
            assLength: normalizedASSContent.length,
            fontFamilies: normalizedFontFamilies,
          });
        }, 4000);
        try {
          await instance.ready;
        } finally {
          window.clearTimeout(readyTimeout);
        }
        debugLog(debugLabel, "JASSUB instance ready", {
          canvas: describeCanvas(activeCanvas),
          video: describeVideo(activeVideo),
          loadedFontFamilies: [...loadedFontFamiliesRef.current],
        });
      } else {
        const activeInstance = instance;
        if (!activeInstance) {
          debugWarn(
            debugLabel,
            "active JASSUB instance disappeared before reuse",
          );
          return;
        }
        const missingFamilies = normalizedFontFamilies.filter(
          (family) =>
            !loadedFontFamiliesRef.current.has(normalizeFontFamilyKey(family)),
        );
        if (missingFamilies.length > 0) {
          const missingAssets = flattenFontAssets(
            missingFamilies.map((family) => ({
              family,
              assets: fontAssetsByFamily.get(family) ?? [],
            })),
          );
          if (missingAssets.length > 0) {
            debugLog(debugLabel, "adding missing fonts to existing instance", {
              missingFamilies,
              assetCount: missingAssets.length,
            });
            await activeInstance.ready;
            await activeInstance.renderer.addFonts(missingAssets);
          }
          for (const family of missingFamilies) {
            if ((fontAssetsByFamily.get(family) ?? []).length > 0) {
              loadedFontFamiliesRef.current.add(normalizeFontFamilyKey(family));
              continue;
            }
            debugWarn(
              debugLabel,
              "font family resolved without usable assets",
              {
                family,
              },
            );
          }
        }
        await activeInstance.ready;
        debugLog(debugLabel, "updating track on existing instance", {
          loadedFontFamilies: [...loadedFontFamiliesRef.current],
          assLength: normalizedASSContent.length,
        });
        await activeInstance.renderer.setTrack(normalizedASSContent);
        instance = activeInstance;
      }

      if (cancelled || !instance) {
        debugWarn(debugLabel, "preview sync cancelled before final draw");
        return;
      }

      if (!activeVideo) {
        await resizeStaticPreview(
          debugLabel,
          instance,
          activeCanvas,
          currentTimeSeconds,
        );
      } else {
        debugLog(debugLabel, "video-backed preview ready", {
          canvas: describeCanvas(activeCanvas),
          video: describeVideo(activeVideo),
        });
      }
    };

    void syncPreview().catch((error) => {
      debugError(debugLabel, "preview sync failed", error, {
        assLength: normalizedASSContent.length,
        fontFamilies: normalizedFontFamilies,
        canvas: describeCanvas(activeCanvas),
        video: describeVideo(activeVideo),
      });
    });

    return () => {
      cancelled = true;
    };
  }, [
    canvas,
    currentTimeSeconds,
    debugLabel,
    enabled,
    fontFamilyKey,
    normalizedASSContent,
    normalizedFontFamilies,
    requireVideo,
    video,
  ]);

  React.useEffect(() => {
    if (!enabled || !canvas || video || requireVideo) {
      return;
    }
    const instance = instanceRef.current;
    if (!instance) {
      return;
    }

    let frame = 0;
    const redraw = () => {
      const activeInstance = instanceRef.current;
      if (
        !activeInstance ||
        boundCanvasRef.current !== canvas ||
        boundVideoRef.current
      ) {
        debugWarn(
          debugLabel,
          "static preview redraw skipped because instance binding changed",
          {
            hasInstance: Boolean(activeInstance),
            canvasMatches: boundCanvasRef.current === canvas,
            hasBoundVideo: Boolean(boundVideoRef.current),
          },
        );
        return;
      }
      void resizeStaticPreview(
        debugLabel,
        activeInstance,
        canvas,
        currentTimeSeconds,
      );
    };
    const scheduleRedraw = () => {
      if (frame) {
        cancelAnimationFrame(frame);
      }
      frame = requestAnimationFrame(redraw);
    };

    scheduleRedraw();
    const observer = new ResizeObserver(scheduleRedraw);
    observer.observe(canvas);
    return () => {
      observer.disconnect();
      if (frame) {
        cancelAnimationFrame(frame);
      }
    };
  }, [canvas, currentTimeSeconds, debugLabel, enabled, requireVideo, video]);
}

function installJassubMainThreadPatches() {
  if (hasPatchedJassubColorSpaceProbe) {
    return;
  }
  hasPatchedJassubColorSpaceProbe = true;

  const prototype = JASSUB.prototype as PatchedJassubPrototype;
  if (typeof prototype._updateColorSpace !== "function") {
    return;
  }

  prototype._updateColorSpace = async function patchedUpdateColorSpace(
    this: PatchedJassubInstance,
  ) {
    await this.ready;
    const activeVideo = this._video;
    if (
      !activeVideo ||
      typeof activeVideo.requestVideoFrameCallback !== "function" ||
      typeof VideoFrame === "undefined"
    ) {
      return;
    }

    activeVideo.requestVideoFrameCallback(async () => {
      let frame: VideoFrame | null = null;
      try {
        frame = new VideoFrame(activeVideo);
        const matrix = frame.colorSpace.matrix?.toLowerCase() ?? "rgb";
        const mappedColorSpace =
          JASSUB_VIDEO_COLOR_SPACE_MAP[
            matrix as keyof typeof JASSUB_VIDEO_COLOR_SPACE_MAP
          ];
        if (!mappedColorSpace) {
          return;
        }
        await this.renderer._setColorSpace(mappedColorSpace);
      } catch (error) {
        if (isSecurityError(error)) {
          return;
        }
        console.warn(error);
      } finally {
        frame?.close();
      }
    });
  };
}

async function resizeStaticPreview(
  debugLabel: string,
  instance: JASSUB,
  canvas: HTMLCanvasElement,
  currentTimeSeconds: number,
) {
  const cssWidth = Math.max(
    1,
    Math.round(canvas.clientWidth || canvas.width || 1),
  );
  const cssHeight = Math.max(
    1,
    Math.round(canvas.clientHeight || canvas.height || 1),
  );
  const devicePixelRatio = Math.max(1, window.devicePixelRatio || 1);
  const width = Math.max(1, Math.round(cssWidth * devicePixelRatio));
  const height = Math.max(1, Math.round(cssHeight * devicePixelRatio));
  debugLog(debugLabel, "static preview resize/draw", {
    cssWidth,
    cssHeight,
    devicePixelRatio,
    width,
    height,
    currentTimeSeconds,
    canvas: describeCanvas(canvas),
  });
  await instance.ready;
  await instance.resize(true, width, height);
  await instance.renderer._draw(Math.max(0, currentTimeSeconds), true);
}

async function ensureFontAssets(
  debugLabel: string,
  families: string[],
  cache: Map<string, Promise<Uint8Array[]>>,
) {
  const deduped = dedupeFontFamilies(families);
  if (deduped.length === 0) {
    debugLog(debugLabel, "no referenced font families provided");
    return new Map<string, Uint8Array[]>();
  }

  const missingFamilies = deduped.filter(
    (family) => !cache.has(normalizeFontFamilyKey(family)),
  );
  if (missingFamilies.length > 0) {
    debugLog(debugLabel, "requesting missing font families", {
      missingFamilies,
      cachedFamilies: deduped.filter(
        (family) => !missingFamilies.includes(family),
      ),
    });
    const batchPromise = exportFontFamilies(missingFamilies).then((result) => {
      debugLog(
        debugLabel,
        "font family export completed",
        result.map((item) => ({
          family: item.family,
          assetCount: item.assets.length,
          files: item.assets.map((asset) => asset.fileName),
        })),
      );
      return result;
    });
    for (const family of missingFamilies) {
      const familyKey = normalizeFontFamilyKey(family);
      cache.set(
        familyKey,
        batchPromise
          .then((result) => {
            const match = result.find(
              (item) => normalizeFontFamilyKey(item.family) === familyKey,
            );
            if (!match) {
              return [];
            }
            const decodedAssets = match.assets
              .map((asset) => decodeBase64(asset.contentBase64))
              .filter((value) => value.byteLength > 0);
            if (decodedAssets.length === 0) {
              debugWarn(
                debugLabel,
                "font family export returned no usable assets",
                {
                  family,
                },
              );
            }
            return decodedAssets;
          })
          .catch((error) => {
            cache.delete(familyKey);
            throw error;
          }),
      );
    }
  }

  const result = new Map<string, Uint8Array[]>();
  for (const family of deduped) {
    const cached = cache.get(normalizeFontFamilyKey(family));
    if (!cached) {
      result.set(family, []);
      continue;
    }
    result.set(family, await cached);
  }
  debugLog(
    debugLabel,
    "resolved font asset cache",
    summarizeFontAssets(result),
  );
  return result;
}

function flattenFontAssets(
  entries:
    | Array<{ family: string; assets: Uint8Array[] }>
    | Map<string, Uint8Array[]>,
) {
  if (entries instanceof Map) {
    return [...entries.values()].flatMap((assets) => assets);
  }
  return entries.flatMap((entry) => entry.assets);
}

function resolveDefaultFontFamily(
  fontAssetsByFamily: Map<string, Uint8Array[]>,
  fontFamilies: string[],
) {
  for (const family of fontFamilies) {
    if ((fontAssetsByFamily.get(family) ?? []).length > 0) {
      return family;
    }
  }
  return undefined;
}

function decodeBase64(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return new Uint8Array();
  }
  const binary = window.atob(trimmed);
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function isSecurityError(error: unknown) {
  return error instanceof DOMException && error.name === "SecurityError";
}

function dedupeFontFamilies(families: string[]) {
  const seen = new Set<string>();
  const result: string[] = [];
  for (const family of families) {
    const trimmed = family.trim();
    if (!trimmed) {
      continue;
    }
    const key = normalizeFontFamilyKey(trimmed);
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    result.push(trimmed);
  }
  return result;
}

function normalizeFontFamilyKey(value: string) {
  return value.trim().toLowerCase();
}

function resolvePlatformJassubFallbackFontFamily() {
  const userAgent = navigator.userAgent || "";
  const platform = navigator.platform || "";
  if (/Windows/i.test(userAgent) || /Win/i.test(platform)) {
    return "Segoe UI";
  }
  return "Helvetica";
}

function extractASSDeclaredFontFamilies(assContent: string) {
  const trimmed = assContent.trim();
  if (!trimmed) {
    return [];
  }

  const result: string[] = [];
  const lines = trimmed.split(/\r?\n/);
  let formatColumns: string[] | null = null;

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (!line) {
      continue;
    }

    const metadataMatch = line.match(
      /^;\s*DCStyle\.[^.]+\.FontFamily:\s*(.+)\s*$/i,
    );
    if (metadataMatch?.[1]) {
      result.push(metadataMatch[1].trim());
      continue;
    }

    if (/^format\s*:/i.test(line)) {
      formatColumns = line
        .slice(line.indexOf(":") + 1)
        .split(",")
        .map((column) => column.trim().toLowerCase());
      continue;
    }

    if (!/^style\s*:/i.test(line)) {
      continue;
    }

    const values = line
      .slice(line.indexOf(":") + 1)
      .split(",")
      .map((value) => value.trim());
    const fontNameIndex = formatColumns?.indexOf("fontname") ?? 1;
    const fontFamily = values[fontNameIndex] ?? "";
    if (fontFamily) {
      result.push(fontFamily);
    }
  }

  return dedupeFontFamilies(result);
}

function debugLog(debugLabel: string, message: string, payload?: unknown) {
  void debugLabel;
  void message;
  void payload;
}

function debugWarn(debugLabel: string, message: string, payload?: unknown) {
  void debugLabel;
  void message;
  void payload;
}

function debugError(
  debugLabel: string,
  message: string,
  error: unknown,
  payload?: unknown,
) {
  if (typeof payload === "undefined") {
    console.error(`${JASSUB_DEBUG_PREFIX}[${debugLabel}] ${message}`, error);
    return;
  }
  console.error(`${JASSUB_DEBUG_PREFIX}[${debugLabel}] ${message}`, {
    error,
    ...toObjectPayload(payload),
  });
}

function describeCanvas(canvas: HTMLCanvasElement | null | undefined) {
  if (!canvas) {
    return null;
  }
  return {
    width: canvas.width,
    height: canvas.height,
    clientWidth: canvas.clientWidth,
    clientHeight: canvas.clientHeight,
  };
}

function describeVideo(video: HTMLVideoElement | null | undefined) {
  if (!video) {
    return null;
  }
  return {
    readyState: video.readyState,
    paused: video.paused,
    currentTime: video.currentTime,
    videoWidth: video.videoWidth,
    videoHeight: video.videoHeight,
    clientWidth: video.clientWidth,
    clientHeight: video.clientHeight,
    currentSrc: video.currentSrc || video.src || "",
  };
}

function summarizeFontAssets(fontAssetsByFamily: Map<string, Uint8Array[]>) {
  return [...fontAssetsByFamily.entries()].map(([family, assets]) => ({
    family,
    assetCount: assets.length,
    assetByteLengths: assets.map((asset) => asset.byteLength),
  }));
}

function toObjectPayload(payload: unknown) {
  if (!payload || typeof payload !== "object" || Array.isArray(payload)) {
    return { payload };
  }
  return payload as Record<string, unknown>;
}
