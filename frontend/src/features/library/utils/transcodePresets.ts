import type {
  TranscodePreset,
  TranscodePresetOutputType,
  TranscodeScaleMode,
} from "@/shared/store/transcodePresets";

export type MediaMetadata = {
  width?: number | null;
  height?: number | null;
  videoCodec?: string | null;
  audioCodec?: string | null;
  channels?: number | null;
  bitrate?: number | null;
  durationMs?: number | null;
};

export type PresetAvailability = {
  preset: TranscodePreset;
  available: boolean;
  reasonKey?:
    | "noVideoStream"
    | "noAudioStream"
    | "containerVideoCodec"
    | "containerAudioCodec"
    | "resolutionTooSmall";
};

type ContainerCompat = {
  videoCodecs: string[];
  audioCodecs: string[];
};

const CONTAINER_COMPAT: Record<string, ContainerCompat> = {
  mp4: { videoCodecs: ["h264", "h265"], audioCodecs: ["aac", "mp3", "copy"] },
  mov: { videoCodecs: ["h264", "h265"], audioCodecs: ["aac", "mp3", "copy"] },
  mkv: {
    videoCodecs: ["h264", "h265", "vp9", "copy"],
    audioCodecs: ["aac", "mp3", "opus", "copy"],
  },
  webm: { videoCodecs: ["vp9"], audioCodecs: ["opus", "copy"] },
  mp3: { videoCodecs: [], audioCodecs: ["mp3"] },
  m4a: { videoCodecs: [], audioCodecs: ["aac", "mp3", "copy"] },
  ogg: { videoCodecs: [], audioCodecs: ["opus", "copy"] },
  flac: { videoCodecs: [], audioCodecs: ["flac", "copy"] },
  wav: { videoCodecs: [], audioCodecs: ["pcm", "copy"] },
};

const SCALE_TARGETS: Record<
  Exclude<TranscodeScaleMode, "custom" | "original">,
  { width: number; height: number }
> = {
  "2160p": { width: 3840, height: 2160 },
  "1080p": { width: 1920, height: 1080 },
  "720p": { width: 1280, height: 720 },
  "480p": { width: 854, height: 480 },
};

const DEFAULT_FFMPEG_PRESET = "slow";
const DEFAULT_H264_CRF = 18;
const DEFAULT_H265_CRF = 20;
const DEFAULT_VP9_CRF = 20;

type BuiltinVideoPresetSeries = {
  idPrefix: string;
  namePrefix: string;
  container: string;
  videoCodec: string;
  audioCodec: string;
  crf: number;
};

type BuiltinVideoScaleSpec = {
  idSuffix: string;
  nameSuffix: string;
  scale: TranscodeScaleMode;
};

type BuiltinAudioPresetSpec = {
  id: string;
  name: string;
  container: string;
  audioCodec: string;
  bitrate?: number;
};

const BUILTIN_VIDEO_PRESET_SERIES: BuiltinVideoPresetSeries[] = [
  {
    idPrefix: "builtin-video-h264-mp4",
    namePrefix: "H.264 MP4",
    container: "mp4",
    videoCodec: "h264",
    audioCodec: "aac",
    crf: DEFAULT_H264_CRF,
  },
  {
    idPrefix: "builtin-video-h265-mp4",
    namePrefix: "H.265 MP4",
    container: "mp4",
    videoCodec: "h265",
    audioCodec: "aac",
    crf: DEFAULT_H265_CRF,
  },
  {
    idPrefix: "builtin-video-h264-mov",
    namePrefix: "H.264 MOV",
    container: "mov",
    videoCodec: "h264",
    audioCodec: "aac",
    crf: DEFAULT_H264_CRF,
  },
  {
    idPrefix: "builtin-video-h265-mov",
    namePrefix: "H.265 MOV",
    container: "mov",
    videoCodec: "h265",
    audioCodec: "aac",
    crf: DEFAULT_H265_CRF,
  },
  {
    idPrefix: "builtin-video-h264-mkv",
    namePrefix: "H.264 MKV",
    container: "mkv",
    videoCodec: "h264",
    audioCodec: "aac",
    crf: DEFAULT_H264_CRF,
  },
  {
    idPrefix: "builtin-video-h265-mkv",
    namePrefix: "H.265 MKV",
    container: "mkv",
    videoCodec: "h265",
    audioCodec: "aac",
    crf: DEFAULT_H265_CRF,
  },
  {
    idPrefix: "builtin-video-vp9-mkv",
    namePrefix: "VP9 MKV",
    container: "mkv",
    videoCodec: "vp9",
    audioCodec: "opus",
    crf: DEFAULT_VP9_CRF,
  },
  {
    idPrefix: "builtin-video-vp9-webm",
    namePrefix: "VP9 WebM",
    container: "webm",
    videoCodec: "vp9",
    audioCodec: "opus",
    crf: DEFAULT_VP9_CRF,
  },
];

export function resolveRecommendedAudioBitrateKbps(
  audioCodec?: string,
): number | undefined {
  switch ((audioCodec ?? "").trim().toLowerCase()) {
    case "aac":
      return 256;
    case "mp3":
      return 320;
    case "opus":
      return 192;
    default:
      return undefined;
  }
}

const BUILTIN_VIDEO_SCALE_SPECS: BuiltinVideoScaleSpec[] = [
  { idSuffix: "original", nameSuffix: "Original", scale: "original" },
  { idSuffix: "2160p", nameSuffix: "2160p", scale: "2160p" },
  { idSuffix: "1080p", nameSuffix: "1080p", scale: "1080p" },
  { idSuffix: "720p", nameSuffix: "720p", scale: "720p" },
  { idSuffix: "480p", nameSuffix: "480p", scale: "480p" },
];

const BUILTIN_AUDIO_PRESET_SPECS: BuiltinAudioPresetSpec[] = [
  {
    id: "builtin-audio-mp3-128k",
    name: "MP3 128k",
    container: "mp3",
    audioCodec: "mp3",
    bitrate: 128,
  },
  {
    id: "builtin-audio-mp3-192k",
    name: "MP3 192k",
    container: "mp3",
    audioCodec: "mp3",
    bitrate: 192,
  },
  {
    id: "builtin-audio-mp3-256k",
    name: "MP3 256k",
    container: "mp3",
    audioCodec: "mp3",
    bitrate: 256,
  },
  {
    id: "builtin-audio-mp3-320k",
    name: "MP3 320k",
    container: "mp3",
    audioCodec: "mp3",
    bitrate: 320,
  },
  {
    id: "builtin-audio-aac-m4a-128k",
    name: "AAC M4A 128k",
    container: "m4a",
    audioCodec: "aac",
    bitrate: 128,
  },
  {
    id: "builtin-audio-aac-m4a-192k",
    name: "AAC M4A 192k",
    container: "m4a",
    audioCodec: "aac",
    bitrate: 192,
  },
  {
    id: "builtin-audio-aac-m4a-256k",
    name: "AAC M4A 256k",
    container: "m4a",
    audioCodec: "aac",
    bitrate: 256,
  },
  {
    id: "builtin-audio-opus-ogg-96k",
    name: "Opus OGG 96k",
    container: "ogg",
    audioCodec: "opus",
    bitrate: 96,
  },
  {
    id: "builtin-audio-opus-ogg-128k",
    name: "Opus OGG 128k",
    container: "ogg",
    audioCodec: "opus",
    bitrate: 128,
  },
  {
    id: "builtin-audio-opus-ogg-192k",
    name: "Opus OGG 192k",
    container: "ogg",
    audioCodec: "opus",
    bitrate: 192,
  },
  {
    id: "builtin-audio-flac-lossless",
    name: "FLAC Lossless",
    container: "flac",
    audioCodec: "flac",
  },
  {
    id: "builtin-audio-wav-pcm",
    name: "WAV PCM 16-bit",
    container: "wav",
    audioCodec: "pcm",
  },
];

function buildBuiltinVideoPresets(): TranscodePreset[] {
  return BUILTIN_VIDEO_PRESET_SERIES.flatMap((series) =>
    BUILTIN_VIDEO_SCALE_SPECS.map((scale) => ({
      id: `${series.idPrefix}-${scale.idSuffix}`,
      name: `${series.namePrefix} ${scale.nameSuffix}`,
      outputType: "video" as const,
      container: series.container,
      videoCodec: series.videoCodec,
      audioCodec: series.audioCodec,
      qualityMode: "crf" as const,
      crf: series.crf,
      audioBitrateKbps: resolveRecommendedAudioBitrateKbps(series.audioCodec),
      scale: scale.scale,
      ffmpegPreset: DEFAULT_FFMPEG_PRESET,
      requiresVideo: true,
      isBuiltin: true,
    })),
  );
}

function buildBuiltinAudioPresets(): TranscodePreset[] {
  return BUILTIN_AUDIO_PRESET_SPECS.map((spec) => ({
    id: spec.id,
    name: spec.name,
    outputType: "audio" as const,
    container: spec.container,
    audioCodec: spec.audioCodec,
    audioBitrateKbps: spec.bitrate,
    requiresAudio: true,
    isBuiltin: true,
  }));
}

export const BUILTIN_PRESETS: TranscodePreset[] = [
  ...buildBuiltinVideoPresets(),
  ...buildBuiltinAudioPresets(),
];

export function buildPresetSummary(
  preset: TranscodePreset,
  t: (key: string) => string,
) {
  const parts: string[] = [];
  if (preset.container) {
    parts.push(preset.container.toUpperCase());
  }
  if (preset.outputType === "video") {
    if (preset.videoCodec) {
      parts.push(codecLabel(preset.videoCodec));
    }
    if (preset.scale && preset.scale !== "original") {
      parts.push(preset.scale.toUpperCase());
    } else {
      parts.push(t("library.workspace.transcode.summary.original"));
    }
    if (preset.qualityMode === "bitrate" && preset.bitrateKbps) {
      parts.push(
        t("library.workspace.transcode.summary.kbps").replace(
          "{value}",
          String(preset.bitrateKbps),
        ),
      );
    } else if (preset.crf) {
      parts.push(
        t("library.workspace.transcode.summary.crf").replace(
          "{value}",
          String(preset.crf),
        ),
      );
    }
  }
  if (preset.audioCodec) {
    parts.push(audioCodecLabel(preset.audioCodec, t));
    if (preset.audioBitrateKbps && preset.audioCodec !== "copy") {
      parts.push(
        t("library.workspace.transcode.summary.kbps").replace(
          "{value}",
          String(preset.audioBitrateKbps),
        ),
      );
    }
  }
  return parts.join(" · ");
}

export function resolvePresetName(
  preset: TranscodePreset,
  t: (key: string) => string,
) {
  return preset.name;
}

export function getPresetAvailability(
  presets: TranscodePreset[],
  metadata: MediaMetadata,
): PresetAvailability[] {
  const hasVideo = Boolean(
    metadata.videoCodec || metadata.width || metadata.height,
  );
  const hasAudio = Boolean(metadata.audioCodec || metadata.channels);
  const videoKnown = Boolean(
    metadata.videoCodec || metadata.width || metadata.height,
  );
  const audioKnown = Boolean(metadata.audioCodec || metadata.channels);
  const inputWidth = metadata.width ?? 0;
  const inputHeight = metadata.height ?? 0;
  const inputShortSide =
    inputWidth > 0 && inputHeight > 0 ? Math.min(inputWidth, inputHeight) : 0;

  return presets.map((preset) => {
    const outputType = preset.outputType;
    if (outputType === "video") {
      if (videoKnown && !hasVideo) {
        return { preset, available: false, reasonKey: "noVideoStream" };
      }
      if (preset.requiresAudio && audioKnown && !hasAudio) {
        return { preset, available: false, reasonKey: "noAudioStream" };
      }
      const compat = resolveContainerCompat(preset.container);
      if (
        preset.videoCodec &&
        compat &&
        !compat.videoCodecs.includes(preset.videoCodec)
      ) {
        return { preset, available: false, reasonKey: "containerVideoCodec" };
      }
      if (
        preset.audioCodec &&
        compat &&
        !compat.audioCodecs.includes(preset.audioCodec)
      ) {
        return { preset, available: false, reasonKey: "containerAudioCodec" };
      }
      const target = resolveScaleTarget(
        preset.scale,
        preset.width,
        preset.height,
      );
      if (target && inputShortSide > 0 && !preset.allowUpscale) {
        const targetShortSide = Math.min(target.width, target.height);
        if (inputShortSide < targetShortSide) {
          return { preset, available: false, reasonKey: "resolutionTooSmall" };
        }
      }
      return { preset, available: true };
    }
    if (outputType === "audio") {
      if (audioKnown && !hasAudio) {
        return { preset, available: false, reasonKey: "noAudioStream" };
      }
      const compat = resolveContainerCompat(preset.container);
      if (
        preset.audioCodec &&
        compat &&
        !compat.audioCodecs.includes(preset.audioCodec)
      ) {
        return { preset, available: false, reasonKey: "containerAudioCodec" };
      }
      return { preset, available: true };
    }
    return { preset, available: true };
  });
}

export function normalizePresetForOutput(
  preset: TranscodePreset,
  outputType: TranscodePresetOutputType,
): TranscodePreset {
  if (outputType === "audio") {
    return {
      ...preset,
      outputType: "audio",
      container: preset.container || "mp3",
      audioCodec: preset.audioCodec || "mp3",
      videoCodec: undefined,
      qualityMode: undefined,
      crf: undefined,
      bitrateKbps: undefined,
      scale: "original",
      width: undefined,
      height: undefined,
      requiresVideo: false,
      requiresAudio: true,
      audioBitrateKbps:
        preset.audioBitrateKbps ??
        resolveRecommendedAudioBitrateKbps(preset.audioCodec || "mp3"),
    };
  }
  const resolvedVideoCodec = preset.videoCodec || "h264";
  return {
    ...preset,
    outputType: "video",
    container: preset.container || "mp4",
    videoCodec: resolvedVideoCodec,
    qualityMode: preset.qualityMode || "crf",
    crf:
      preset.crf ??
      (resolvedVideoCodec === "h265"
        ? DEFAULT_H265_CRF
        : resolvedVideoCodec === "vp9"
          ? DEFAULT_VP9_CRF
          : DEFAULT_H264_CRF),
    audioBitrateKbps:
      preset.audioBitrateKbps ??
      resolveRecommendedAudioBitrateKbps(preset.audioCodec || "aac"),
    ffmpegPreset: preset.ffmpegPreset || DEFAULT_FFMPEG_PRESET,
    requiresVideo: true,
  };
}

function resolveScaleTarget(
  scale?: TranscodeScaleMode,
  width?: number,
  height?: number,
) {
  if (!scale || scale === "original") {
    return null;
  }
  if (scale === "custom") {
    if (width && height) {
      return { width, height };
    }
    return null;
  }
  return SCALE_TARGETS[scale] ?? null;
}

function resolveContainerCompat(container?: string) {
  if (!container) {
    return null;
  }
  return CONTAINER_COMPAT[container.toLowerCase()] ?? null;
}

export function getSupportedVideoCodecs(container?: string) {
  return resolveContainerCompat(container)?.videoCodecs ?? [];
}

export function getSupportedAudioCodecs(container?: string) {
  return resolveContainerCompat(container)?.audioCodecs ?? [];
}

function codecLabel(value: string) {
  switch (value.toLowerCase()) {
    case "h264":
      return "H.264";
    case "h265":
      return "H.265";
    case "vp9":
      return "VP9";
    default:
      return value.toUpperCase();
  }
}

function audioCodecLabel(
  value: string,
  t: (key: string) => string,
) {
  switch (value.toLowerCase()) {
    case "aac":
      return "AAC";
    case "mp3":
      return "MP3";
    case "opus":
      return "Opus";
    case "flac":
      return "FLAC";
    case "pcm":
      return "PCM";
    case "copy":
      return t("library.workspace.transcode.summary.copy");
    default:
      return value.toUpperCase();
  }
}
