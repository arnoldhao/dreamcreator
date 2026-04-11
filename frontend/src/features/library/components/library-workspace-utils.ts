import type {
  LibraryBilingualStyleDTO,
  LibraryFileDTO,
  LibraryMonoStyleDTO,
  LibrarySubtitleExportPresetDTO,
  SubtitleExportConfig,
  WorkspaceSubtitleTrackDTO,
  WorkspaceTaskSummaryDTO,
} from "@/shared/contracts/library";

import type {
  LibraryWorkspaceEditor,
  LibraryWorkspaceGuidelineProfileId,
  LibraryWorkspacePersistedState,
} from "../model/workspaceStore";
import type { LibraryWorkspaceTarget } from "../model/types";
import { normalizeWorkspaceQaCheckSettings } from "../model/workspaceQa";
import {
  buildDefaultSubtitleExportAssTitle,
  buildDefaultSubtitleExportEventName,
  buildDefaultSubtitleExportLibraryName,
  buildDefaultSubtitleExportProjectName,
  DEFAULT_FCPXML_START_TIMECODE_SECONDS,
  normalizeFCPXMLFrameDuration,
  normalizeSubtitleExportFormat,
  normalizeSubtitleExportMediaStrategy,
  resolveFCPXMLFrameDurationFromFrameRate,
  resolveITTEffectiveFrameRate,
  resolveITTFrameTimingFromFrameRate,
} from "../utils/subtitleStyles";
import type { WorkspaceSubtitleRow } from "./workspace/types";

const WORKSPACE_MONO_STYLE_DRAFT_ID = "workspace-mono-current";
const WORKSPACE_BILINGUAL_STYLE_DRAFT_ID = "workspace-bilingual-current";

function normalizeWorkspacePersistedEditorValue(
  value: unknown,
): LibraryWorkspaceEditor {
  return value === "subtitle" ? "subtitle" : "video";
}

function normalizeWorkspacePersistedDisplayModeValue(value: unknown) {
  if (value === "dual" || value === "bilingual") {
    return "bilingual";
  }
  return "mono";
}

function normalizeWorkspacePersistedGuidelineProfileId(
  value: unknown,
): LibraryWorkspaceGuidelineProfileId {
  switch (value) {
    case "bbc":
    case "ade":
      return value;
    default:
      return "netflix";
  }
}

function normalizeWorkspacePersistedString(value: unknown) {
  return typeof value == "string" ? value.trim() : "";
}

function normalizeWorkspaceTargetEditor(
  target: LibraryWorkspaceTarget,
): LibraryWorkspaceEditor {
  return target.openMode === "subtitle" ||
    normalizeWorkspacePersistedString(target.fileType) == "subtitle"
    ? "subtitle"
    : "video";
}

function normalizeLibraryFileKind(value: unknown) {
  return normalizeWorkspacePersistedString(value).toLowerCase();
}

function resolveWorkspaceTargetLinkedFileIds(
  target: LibraryWorkspaceTarget,
  libraryFiles: LibraryFileDTO[],
) {
  const fileId = normalizeWorkspacePersistedString(target.fileId);
  const targetFile = libraryFiles.find((file) => file.id == fileId);
  const targetLibraryId =
    targetFile?.libraryId?.trim() ||
    normalizeWorkspacePersistedString(target.libraryId);
  const rootFileId = targetFile?.lineage.rootFileId?.trim() || fileId;
  const siblings = libraryFiles.filter((file) => {
    if (file.state.deleted) {
      return false;
    }
    if (targetLibraryId && file.libraryId != targetLibraryId) {
      return false;
    }
    const siblingRootId = file.lineage.rootFileId?.trim() || file.id;
    return siblingRootId == rootFileId;
  });
  const derivedVideoFileId =
    siblings.find((file) => {
      const kind = normalizeLibraryFileKind(file.kind);
      return kind == "video" || kind == "audio" || kind == "transcode";
    })?.id ?? undefined;
  const derivedSubtitleFileId =
    siblings.find((file) => normalizeLibraryFileKind(file.kind) == "subtitle")
      ?.id ?? undefined;
  return {
    videoFileId:
      typeof target.videoAssetId == "string"
        ? target.videoAssetId.trim()
        : derivedVideoFileId,
    subtitleFileId:
      typeof target.subtitleAssetId == "string"
        ? target.subtitleAssetId.trim()
        : derivedSubtitleFileId,
  };
}

function resolvePersistedWorkspaceFileID(
  candidate: string,
  files: LibraryFileDTO[],
) {
  const normalized = candidate.trim();
  if (normalized && files.some((file) => file.id == normalized)) {
    return normalized;
  }
  return files[0]?.id ?? "";
}

function resolvePersistedComparisonSubtitleFileID(
  candidate: string,
  files: LibraryFileDTO[],
  activeSubtitleFileId: string,
) {
  const normalized = candidate.trim();
  if (
    normalized &&
    normalized != activeSubtitleFileId &&
    files.some((file) => file.id == normalized)
  ) {
    return normalized;
  }
  return files.find((file) => file.id != activeSubtitleFileId)?.id ?? "";
}

function cloneWorkspaceMonoStyle(
  style: LibraryMonoStyleDTO | null | undefined,
): LibraryMonoStyleDTO | undefined {
  if (!style) {
    return undefined;
  }
  return JSON.parse(JSON.stringify(style)) as LibraryMonoStyleDTO;
}

function cloneWorkspaceLingualStyle(
  style: LibraryBilingualStyleDTO | null | undefined,
): LibraryBilingualStyleDTO | undefined {
  if (!style) {
    return undefined;
  }
  return JSON.parse(JSON.stringify(style)) as LibraryBilingualStyleDTO;
}

export function createWorkspaceMonoStyleDraft(
  style: LibraryMonoStyleDTO | null | undefined,
): LibraryMonoStyleDTO | undefined {
  const draft = cloneWorkspaceMonoStyle(style);
  if (!draft) {
    return undefined;
  }
  return {
    ...draft,
    id: WORKSPACE_MONO_STYLE_DRAFT_ID,
    name: "",
    builtIn: false,
    sourceAssStyleName: undefined,
  };
}

export function createWorkspaceLingualStyleDraft(
  style: LibraryBilingualStyleDTO | null | undefined,
): LibraryBilingualStyleDTO | undefined {
  const draft = cloneWorkspaceLingualStyle(style);
  if (!draft) {
    return undefined;
  }
  return {
    ...draft,
    id: WORKSPACE_BILINGUAL_STYLE_DRAFT_ID,
    name: "",
    builtIn: false,
    primary: {
      ...draft.primary,
      sourceMonoStyleID: undefined,
      sourceMonoStyleName: undefined,
    },
    secondary: {
      ...draft.secondary,
      sourceMonoStyleID: undefined,
      sourceMonoStyleName: undefined,
    },
  };
}

function normalizeWorkspacePersistedMonoStyle(
  value: unknown,
): LibraryMonoStyleDTO | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  return createWorkspaceMonoStyleDraft(value as LibraryMonoStyleDTO);
}

function normalizeWorkspacePersistedLingualStyle(
  value: unknown,
): LibraryBilingualStyleDTO | undefined {
  if (!value || typeof value !== "object") {
    return undefined;
  }
  return createWorkspaceLingualStyleDraft(value as LibraryBilingualStyleDTO);
}

export function parseLibraryWorkspacePersistedState(
  raw: string,
): Partial<LibraryWorkspacePersistedState> | null {
  const trimmed = raw.trim();
  if (!trimmed) {
    return null;
  }
  try {
    const parsed = JSON.parse(trimmed) as Record<string, unknown>;
    if (!parsed || typeof parsed != "object") {
      return null;
    }
    return {
      libraryId: normalizeWorkspacePersistedString(parsed.libraryId),
      activeEditor: normalizeWorkspacePersistedEditorValue(parsed.activeEditor),
      activeVideoFileId: normalizeWorkspacePersistedString(
        parsed.activeVideoFileId,
      ),
      activeSubtitleFileId: normalizeWorkspacePersistedString(
        parsed.activeSubtitleFileId,
      ),
      displayMode: normalizeWorkspacePersistedDisplayModeValue(
        parsed.displayMode,
      ),
      comparisonSubtitleFileId: normalizeWorkspacePersistedString(
        parsed.comparisonSubtitleFileId,
      ),
      guidelineProfileId: normalizeWorkspacePersistedGuidelineProfileId(
        parsed.guidelineProfileId,
      ),
      qaCheckSettings: normalizeWorkspaceQaCheckSettings(parsed.qaCheckSettings),
      subtitleMonoStyle: normalizeWorkspacePersistedMonoStyle(
        parsed.subtitleMonoStyle,
      ),
      subtitleLingualStyle: normalizeWorkspacePersistedLingualStyle(
        parsed.subtitleLingualStyle,
      ),
      subtitleStyleSidebarOpen: Boolean(parsed.subtitleStyleSidebarOpen),
    };
  } catch {
    return null;
  }
}

export function resolveLibraryWorkspacePersistedState(
  candidate: Partial<LibraryWorkspacePersistedState> | null | undefined,
  options: {
    libraryId: string;
    videoFiles: LibraryFileDTO[];
    subtitleFiles: LibraryFileDTO[];
    defaultSubtitleMonoStyle?: LibraryMonoStyleDTO;
    defaultSubtitleLingualStyle?: LibraryBilingualStyleDTO;
  },
): LibraryWorkspacePersistedState {
  const activeVideoFileId = resolvePersistedWorkspaceFileID(
    candidate?.activeVideoFileId ?? "",
    options.videoFiles,
  );
  const activeSubtitleFileId = resolvePersistedWorkspaceFileID(
    candidate?.activeSubtitleFileId ?? "",
    options.subtitleFiles,
  );
  let activeEditor = normalizeWorkspacePersistedEditorValue(
    candidate?.activeEditor,
  );
  if (activeEditor == "video" && !activeVideoFileId && activeSubtitleFileId) {
    activeEditor = "subtitle";
  } else if (
    activeEditor == "subtitle" &&
    !activeSubtitleFileId &&
    activeVideoFileId
  ) {
    activeEditor = "video";
  }
  return {
    libraryId: options.libraryId.trim(),
    activeEditor,
    activeVideoFileId,
    activeSubtitleFileId,
    displayMode: normalizeWorkspacePersistedDisplayModeValue(
      candidate?.displayMode,
    ),
    comparisonSubtitleFileId: resolvePersistedComparisonSubtitleFileID(
      candidate?.comparisonSubtitleFileId ?? "",
      options.subtitleFiles,
      activeSubtitleFileId,
    ),
    guidelineProfileId: normalizeWorkspacePersistedGuidelineProfileId(
      candidate?.guidelineProfileId,
    ),
    qaCheckSettings: normalizeWorkspaceQaCheckSettings(
      candidate?.qaCheckSettings,
    ),
    subtitleMonoStyle:
      createWorkspaceMonoStyleDraft(candidate?.subtitleMonoStyle) ??
      createWorkspaceMonoStyleDraft(options.defaultSubtitleMonoStyle),
    subtitleLingualStyle:
      createWorkspaceLingualStyleDraft(candidate?.subtitleLingualStyle) ??
      createWorkspaceLingualStyleDraft(options.defaultSubtitleLingualStyle),
    subtitleStyleSidebarOpen: Boolean(candidate?.subtitleStyleSidebarOpen),
  };
}

export function resolveLibraryWorkspaceStateFromOpenTarget(
  target: LibraryWorkspaceTarget,
  candidate: Partial<LibraryWorkspacePersistedState> | null | undefined,
  options: {
    libraryId: string;
    libraryFiles: LibraryFileDTO[];
    videoFiles: LibraryFileDTO[];
    subtitleFiles: LibraryFileDTO[];
    defaultSubtitleMonoStyle?: LibraryMonoStyleDTO;
    defaultSubtitleLingualStyle?: LibraryBilingualStyleDTO;
  },
): LibraryWorkspacePersistedState {
  const baseState = resolveLibraryWorkspacePersistedState(candidate, options);
  const targetFileId = normalizeWorkspacePersistedString(target.fileId);
  const activeEditor = normalizeWorkspaceTargetEditor(target);
  const linkedFileIds = resolveWorkspaceTargetLinkedFileIds(
    target,
    options.libraryFiles,
  );
  const activeVideoFileId =
    activeEditor == "video"
      ? targetFileId || baseState.activeVideoFileId
      : linkedFileIds.videoFileId ?? baseState.activeVideoFileId;
  const activeSubtitleFileId =
    activeEditor == "subtitle"
      ? targetFileId || baseState.activeSubtitleFileId
      : linkedFileIds.subtitleFileId ?? baseState.activeSubtitleFileId;
  return {
    ...baseState,
    activeEditor,
    activeVideoFileId,
    activeSubtitleFileId,
    comparisonSubtitleFileId: resolvePersistedComparisonSubtitleFileID(
      baseState.comparisonSubtitleFileId,
      options.subtitleFiles,
      activeSubtitleFileId,
    ),
  };
}

export function rowsEqual(
  left: WorkspaceSubtitleRow[],
  right: WorkspaceSubtitleRow[],
) {
  if (left.length != right.length) {
    return false;
  }
  return left.every((row, index) => {
    const other = right[index];
    return (
      Boolean(other) &&
      row.id == other.id &&
      row.start == other.start &&
      row.end == other.end &&
      row.sourceText == other.sourceText
    );
  });
}

export function resolveErrorMessage(error: unknown, fallback: string) {
  if (error instanceof Error && error.message.trim()) {
    return error.message.trim();
  }
  if (typeof error == "string" && error.trim()) {
    return error.trim();
  }
  return fallback;
}

export function stripFileExtension(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return "";
  }
  const lastSlash = Math.max(
    trimmed.lastIndexOf("/"),
    trimmed.lastIndexOf("\\"),
  );
  const lastDot = trimmed.lastIndexOf(".");
  if (lastDot <= lastSlash || lastDot === -1) {
    return trimmed;
  }
  return trimmed.slice(0, lastDot);
}

export function resolveDirectoryName(path: string) {
  const trimmed = path.trim();
  if (!trimmed) {
    return "";
  }
  const lastSlash = Math.max(
    trimmed.lastIndexOf("/"),
    trimmed.lastIndexOf("\\"),
  );
  return lastSlash > 0 ? trimmed.slice(0, lastSlash) : "";
}

export function parseAssistantModelRef(value: string) {
  const trimmed = value.trim();
  if (!trimmed) {
    return { providerId: "", modelName: "" };
  }
  const slashIndex = trimmed.indexOf("/");
  if (slashIndex > 0) {
    return {
      providerId: trimmed.slice(0, slashIndex).trim(),
      modelName: trimmed.slice(slashIndex + 1).trim(),
    };
  }
  const colonIndex = trimmed.indexOf(":");
  if (colonIndex > 0) {
    return {
      providerId: trimmed.slice(0, colonIndex).trim(),
      modelName: trimmed.slice(colonIndex + 1).trim(),
    };
  }
  return { providerId: "", modelName: trimmed };
}

export function toggleSelectedId(
  currentValues: string[],
  value: string,
  checked: boolean,
) {
  const normalized = value.trim();
  if (!normalized) {
    return currentValues;
  }
  if (checked) {
    if (currentValues.includes(normalized)) {
      return currentValues;
    }
    return [...currentValues, normalized];
  }
  return currentValues.filter((item) => item !== normalized);
}

export function createWorkspacePromptProfileId(prefix: string) {
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function derivePromptProfileName(
  rawName: string,
  prompt: string,
  fallbackPrefix: string,
) {
  const named = rawName.trim();
  if (named) {
    return named;
  }
  const compact = prompt.replace(/\s+/g, " ").trim();
  if (!compact) {
    return fallbackPrefix;
  }
  const base =
    compact.length > 28 ? `${compact.slice(0, 28).trim()}...` : compact;
  return `${fallbackPrefix}: ${base}`;
}

function positiveNumber(value: number | undefined, fallback: number) {
  return typeof value === "number" && Number.isFinite(value) && value > 0
    ? value
    : fallback;
}

export function resolveSubtitleExportPresetFormat(
  profile: LibrarySubtitleExportPresetDTO,
) {
  return normalizeSubtitleExportFormat(
    profile.format ?? profile.targetFormat ?? "srt",
  );
}

type SubtitleExportMediaHint = {
  width: number;
  height: number;
  frameRate: number;
  source: "video" | "subtitle" | "root-subtitle" | "default";
};

type SubtitleExportPresetSelection = {
  preset: LibrarySubtitleExportPresetDTO;
  reason: string;
};

function resolveExportMediaFromFile(file: LibraryFileDTO | null | undefined) {
  if (!file?.media) {
    return null;
  }
  const width = positiveNumber(file.media.width, 0);
  const height = positiveNumber(file.media.height, 0);
  const frameRate = positiveNumber(file.media.frameRate, 0);
  if (width <= 0 && height <= 0 && frameRate <= 0) {
    return null;
  }
  return { width, height, frameRate };
}

export function resolveSubtitleExportMediaHint(
  videoFile: LibraryFileDTO | null,
  subtitleFile: LibraryFileDTO | null,
  files: LibraryFileDTO[],
): SubtitleExportMediaHint {
  const videoMedia = resolveExportMediaFromFile(videoFile);
  if (videoMedia) {
    return {
      width: positiveNumber(videoMedia.width, 1920),
      height: positiveNumber(videoMedia.height, 1080),
      frameRate: positiveNumber(videoMedia.frameRate, 30),
      source: "video",
    };
  }

  const subtitleMedia = resolveExportMediaFromFile(subtitleFile);
  if (subtitleMedia) {
    return {
      width: positiveNumber(subtitleMedia.width, 1920),
      height: positiveNumber(subtitleMedia.height, 1080),
      frameRate: positiveNumber(subtitleMedia.frameRate, 30),
      source: "subtitle",
    };
  }

  const rootFileID = subtitleFile?.lineage.rootFileId?.trim() ?? "";
  if (rootFileID) {
    const rootFile =
      files.find((candidate) => candidate.id === rootFileID) ?? null;
    const rootMedia = resolveExportMediaFromFile(rootFile);
    if (rootMedia) {
      return {
        width: positiveNumber(rootMedia.width, 1920),
        height: positiveNumber(rootMedia.height, 1080),
        frameRate: positiveNumber(rootMedia.frameRate, 30),
        source: "root-subtitle",
      };
    }
  }

  return {
    width: 1920,
    height: 1080,
    frameRate: 30,
    source: "default",
  };
}

export function buildDefaultSubtitleExportConfig(
  libraryName: string,
  mediaHint: SubtitleExportMediaHint,
): SubtitleExportConfig {
  const width = positiveNumber(mediaHint.width, 1920);
  const height = positiveNumber(mediaHint.height, 1080);
  const frameRate = positiveNumber(mediaHint.frameRate, 30);
  const ittTiming = resolveITTFrameTimingFromFrameRate(frameRate);
  const frameDuration = resolveFCPXMLFrameDurationFromFrameRate(frameRate);
  const projectName = buildDefaultSubtitleExportProjectName(libraryName);
  const assTitle = buildDefaultSubtitleExportAssTitle(libraryName);
  const fcpxmlLibraryName = buildDefaultSubtitleExportLibraryName(libraryName);
  const fcpxmlEventName = buildDefaultSubtitleExportEventName(libraryName);
  return {
    srt: {
      encoding: "utf-8",
    },
    vtt: {
      kind: "subtitles",
      language: "en-US",
    },
    ass: {
      playResX: width,
      playResY: height,
      title: assTitle,
    },
    itt: {
      frameRate: ittTiming.frameRate,
      frameRateMultiplier: ittTiming.frameRateMultiplier,
      language: "en-US",
    },
    fcpxml: {
      frameDuration,
      width,
      height,
      colorSpace: "1-1-1 (Rec. 709)",
      version: "1.11",
      projectName,
      libraryName: fcpxmlLibraryName,
      eventName: fcpxmlEventName,
      defaultLane: 1,
      startTimecodeSeconds: DEFAULT_FCPXML_START_TIMECODE_SECONDS,
    },
  };
}

function pickPresetDimensions(preset: LibrarySubtitleExportPresetDTO) {
  const format = resolveSubtitleExportPresetFormat(preset);
  switch (format) {
    case "ass":
      return {
        width: positiveNumber(preset.config?.ass?.playResX, 0),
        height: positiveNumber(preset.config?.ass?.playResY, 0),
        frameRate: 0,
        frameDuration: "",
      };
    case "itt": {
      const configuredITTFrameRate = positiveNumber(
        preset.config?.itt?.frameRate,
        0,
      );
      return {
        width: 0,
        height: 0,
        frameRate:
          configuredITTFrameRate > 0
            ? resolveITTEffectiveFrameRate(
                configuredITTFrameRate,
                preset.config?.itt?.frameRateMultiplier,
              )
            : 0,
        frameDuration: "",
      };
    }
    case "fcpxml":
      return {
        width: positiveNumber(preset.config?.fcpxml?.width, 0),
        height: positiveNumber(preset.config?.fcpxml?.height, 0),
        frameRate: 0,
        frameDuration: (preset.config?.fcpxml?.frameDuration ?? "").trim(),
      };
    default:
      return {
        width: 0,
        height: 0,
        frameRate: 0,
        frameDuration: "",
      };
  }
}

function presetMatchScore(
  preset: LibrarySubtitleExportPresetDTO,
  fallbackFormat: string,
  mediaHint: SubtitleExportMediaHint,
) {
  const normalizedFallbackFormat =
    normalizeSubtitleExportFormat(fallbackFormat);
  const normalizedFormat = resolveSubtitleExportPresetFormat(preset);
  const dimensions = pickPresetDimensions(preset);
  let score = normalizedFormat === normalizedFallbackFormat ? 15 : 0;
  let metrics = 0;

  if (dimensions.width > 0 && dimensions.height > 0) {
    const widthRatio = Math.log2(
      Math.max(mediaHint.width, 1) / dimensions.width,
    );
    const heightRatio = Math.log2(
      Math.max(mediaHint.height, 1) / dimensions.height,
    );
    const resolutionDistance = Math.abs(widthRatio) + Math.abs(heightRatio);
    score += Math.max(0, 120 - resolutionDistance * 100);
    metrics += 1;
  }

  if (dimensions.frameRate > 0) {
    const fpsDistance =
      Math.abs(mediaHint.frameRate - dimensions.frameRate) / 30;
    score += Math.max(0, 90 - fpsDistance * 100);
    metrics += 1;
  }

  if (dimensions.frameDuration) {
    const mediaFrameDuration = resolveFCPXMLFrameDurationFromFrameRate(
      mediaHint.frameRate,
    );
    const profileFrameDuration = normalizeFCPXMLFrameDuration(
      dimensions.frameDuration,
    );
    score += profileFrameDuration === mediaFrameDuration ? 90 : 10;
    metrics += 1;
  }

  return { score, metrics };
}

function resolvePresetSelectionReason(
  source: SubtitleExportMediaHint["source"],
  presetName: string,
  mediaStrategy: "auto" | "fixed",
) {
  if (mediaStrategy === "fixed") {
    return `Applied fixed preset -> ${presetName}`;
  }
  switch (source) {
    case "video":
      return `Auto matched from video metadata -> ${presetName}`;
    case "subtitle":
      return `Auto matched from subtitle metadata -> ${presetName}`;
    case "root-subtitle":
      return `Auto matched from source subtitle lineage metadata -> ${presetName}`;
    default:
      return `Fallback to default preset -> ${presetName}`;
  }
}

export function buildWorkspaceTaskProgressLabel(
  task: WorkspaceTaskSummaryDTO | null | undefined,
  runningLabel: string,
) {
  if (!task) {
    return runningLabel;
  }
  if (
    typeof task.current === "number" &&
    typeof task.total === "number" &&
    task.total > 0
  ) {
    return `${runningLabel} ${task.current}/${task.total}`;
  }
  return runningLabel;
}

export function buildWorkspaceTranslateTaskLabel(
  tasks: WorkspaceTaskSummaryDTO[] | undefined,
  t: (key: string) => string,
) {
  const items = tasks ?? [];
  if (items.length === 0) {
    return t("library.workspace.actions.translate");
  }
  if (items.length > 1) {
    return t("library.workspace.header.translateRunningCount")
      .replace("{count}", String(items.length));
  }
  return buildWorkspaceTaskProgressLabel(
    items[0],
    t("library.workspace.header.translateRunning"),
  );
}

export function trackSupportsDualDisplay(
  track: WorkspaceSubtitleTrackDTO | null | undefined,
) {
  const blockedActions = track?.pendingReview?.blockedActions ?? [];
  return (
    blockedActions.includes("proofread") === false &&
    blockedActions.includes("qa") === false
  );
}

export function mergeSubtitleExportConfig(
  base: SubtitleExportConfig,
  override?: SubtitleExportConfig,
): SubtitleExportConfig {
  if (!override) {
    return base;
  }
  const mergeString = (
    baseValue: string | undefined,
    overrideValue: string | undefined,
  ) => (overrideValue && overrideValue.trim() ? overrideValue : baseValue);
  const mergeNumber = (
    baseValue: number | undefined,
    overrideValue: number | undefined,
  ) =>
    typeof overrideValue === "number" &&
    Number.isFinite(overrideValue) &&
    overrideValue > 0
      ? overrideValue
      : baseValue;
  const mergeNonNegativeNumber = (
    baseValue: number | undefined,
    overrideValue: number | undefined,
  ) =>
    typeof overrideValue === "number" &&
    Number.isFinite(overrideValue) &&
    overrideValue >= 0
      ? overrideValue
      : baseValue;
  return {
    ...base,
    srt: {
      ...(base.srt ?? {}),
      encoding: mergeString(base.srt?.encoding, override.srt?.encoding),
    },
    vtt: {
      ...(base.vtt ?? {}),
      kind: mergeString(base.vtt?.kind, override.vtt?.kind),
      language: mergeString(base.vtt?.language, override.vtt?.language),
    },
    ass: {
      ...(base.ass ?? {}),
      playResX: mergeNumber(base.ass?.playResX, override.ass?.playResX),
      playResY: mergeNumber(base.ass?.playResY, override.ass?.playResY),
      title: mergeString(base.ass?.title, override.ass?.title),
    },
    itt: {
      ...(base.itt ?? {}),
      frameRate: mergeNumber(base.itt?.frameRate, override.itt?.frameRate),
      frameRateMultiplier: mergeString(
        base.itt?.frameRateMultiplier,
        override.itt?.frameRateMultiplier,
      ),
      language: mergeString(base.itt?.language, override.itt?.language),
    },
    fcpxml: {
      ...(base.fcpxml ?? {}),
      frameDuration: mergeString(
        base.fcpxml?.frameDuration,
        override.fcpxml?.frameDuration,
      ),
      width: mergeNumber(base.fcpxml?.width, override.fcpxml?.width),
      height: mergeNumber(base.fcpxml?.height, override.fcpxml?.height),
      colorSpace: mergeString(
        base.fcpxml?.colorSpace,
        override.fcpxml?.colorSpace,
      ),
      version: mergeString(base.fcpxml?.version, override.fcpxml?.version),
      libraryName: mergeString(
        base.fcpxml?.libraryName,
        override.fcpxml?.libraryName,
      ),
      eventName: mergeString(
        base.fcpxml?.eventName,
        override.fcpxml?.eventName,
      ),
      projectName: mergeString(
        base.fcpxml?.projectName,
        override.fcpxml?.projectName,
      ),
      defaultLane: mergeNumber(
        base.fcpxml?.defaultLane,
        override.fcpxml?.defaultLane,
      ),
      startTimecodeSeconds: mergeNonNegativeNumber(
        base.fcpxml?.startTimecodeSeconds,
        override.fcpxml?.startTimecodeSeconds,
      ),
    },
  };
}

export function resolveSubtitleExportPresetOverrideConfig(
  preset: LibrarySubtitleExportPresetDTO | null | undefined,
): SubtitleExportConfig | undefined {
  if (!preset?.config) {
    return undefined;
  }
  const strategy = normalizeSubtitleExportMediaStrategy(
    preset.mediaStrategy ?? "",
  );
  if (strategy !== "auto") {
    return preset.config;
  }
  return {
    ...preset.config,
    ass: preset.config.ass
      ? {
          ...preset.config.ass,
          playResX: 0,
          playResY: 0,
        }
      : preset.config.ass,
    itt: preset.config.itt
      ? {
          ...preset.config.itt,
          frameRate: 0,
          frameRateMultiplier: "",
        }
      : preset.config.itt,
    fcpxml: preset.config.fcpxml
      ? {
          ...preset.config.fcpxml,
          width: 0,
          height: 0,
          frameDuration: "",
        }
      : preset.config.fcpxml,
  };
}

export function resolveSubtitleExportPresetSelection(
  presets: LibrarySubtitleExportPresetDTO[],
  defaults: { subtitleExportPresetId?: string },
  fallbackFormat: string,
  mediaHint: SubtitleExportMediaHint,
): SubtitleExportPresetSelection | null {
  if (presets.length === 0) {
    return null;
  }
  const normalizedFallbackFormat =
    normalizeSubtitleExportFormat(fallbackFormat);
  const sameFormatPresets = presets.filter(
    (preset) =>
      resolveSubtitleExportPresetFormat(preset) === normalizedFallbackFormat,
  );
  if (sameFormatPresets.length === 0) {
    return null;
  }
  const defaultPreset = sameFormatPresets.find(
    (preset) => preset.id === defaults.subtitleExportPresetId?.trim(),
  );
  if (defaultPreset) {
    const strategy = normalizeSubtitleExportMediaStrategy(
      defaultPreset.mediaStrategy ?? "",
    );
    return {
      preset: defaultPreset,
      reason: resolvePresetSelectionReason(
        mediaHint.source,
        defaultPreset.name || defaultPreset.id,
        strategy,
      ),
    };
  }

  const autoPresets = sameFormatPresets.filter(
    (preset) =>
      normalizeSubtitleExportMediaStrategy(preset.mediaStrategy ?? "") ===
      "auto",
  );
  if (autoPresets.length > 0) {
    const preferredAutoPreset =
      autoPresets.find((preset) =>
        (preset.id ?? "").toLowerCase().includes("auto"),
      ) ?? autoPresets[0];
    return {
      preset: preferredAutoPreset,
      reason: resolvePresetSelectionReason(
        mediaHint.source,
        preferredAutoPreset.name || preferredAutoPreset.id,
        "auto",
      ),
    };
  }

  const fixedPresets = sameFormatPresets.filter(
    (preset) =>
      normalizeSubtitleExportMediaStrategy(preset.mediaStrategy ?? "") ===
      "fixed",
  );
  const scored = fixedPresets
    .map((preset) => {
      const score = presetMatchScore(preset, fallbackFormat, mediaHint);
      return { preset, ...score };
    })
    .filter((candidate) => candidate.metrics > 0)
    .sort((left, right) => right.score - left.score);
  if (scored.length > 0) {
    return {
      preset: scored[0].preset,
      reason: resolvePresetSelectionReason(
        mediaHint.source,
        scored[0].preset.name || scored[0].preset.id,
        "fixed",
      ),
    };
  }
  const fallbackPreset = fixedPresets[0] ?? sameFormatPresets[0];
  const fallbackStrategy = normalizeSubtitleExportMediaStrategy(
    fallbackPreset.mediaStrategy ?? "",
  );
  return {
    preset: fallbackPreset,
    reason: resolvePresetSelectionReason(
      mediaHint.source,
      fallbackPreset.name || fallbackPreset.id,
      fallbackStrategy,
    ),
  };
}
