import * as React from "react";
import { useQueryClient } from "@tanstack/react-query";
import { Call } from "@wailsio/runtime";
import {
  AudioLines,
  Check,
  BookOpen,
  Captions,
  ChevronDown,
  Clapperboard,
  Copy,
  Download,
  Loader2,
  MonitorPlay,
  Database,
  FileText,
  Languages,
  PencilLine,
  Plus,
  RefreshCw,
  Search,
  Settings2,
  SlidersHorizontal,
  Sparkles,
  Trash2,
  Type,
  X,
} from "lucide-react";

import { useFontFamilies } from "@/hooks/useFontFamilies";
import { FONT_CATALOG_QUERY_KEY } from "@/hooks/useFontCatalog";
import { cn } from "@/lib/utils";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import {
  useDeleteTranscodePreset,
  useSaveTranscodePreset,
  useTranscodePresets,
} from "@/shared/query/library";
import { REALTIME_TOPICS, registerTopic } from "@/shared/realtime";
import type {
  LibraryGlossaryProfileDTO,
  LibraryGlossaryTermDTO,
  LibraryModuleConfigDTO,
  LibraryPromptProfileDTO,
  LibraryRemoteFontManifestDTO,
  LibrarySubtitleExportPresetDTO,
  LibrarySubtitleStyleDocumentDTO,
  LibrarySubtitleStyleSourceDTO,
  LibraryTranslateLanguageDTO,
  TranscodePreset,
} from "@/shared/contracts/library";
import { Badge } from "@/shared/ui/badge";
import { Button } from "@/shared/ui/button";
import { DASHBOARD_PANEL_CARD_CLASS } from "@/shared/ui/dashboard";
import {
  DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
} from "@/shared/ui/dashboard-dialog";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu";
import { Input } from "@/shared/ui/input";
import { Select } from "@/shared/ui/select";
import { Switch } from "@/shared/ui/switch";

import { SubtitleStylePresetManager } from "./SubtitleStylePresetManager";
import { FONT_FAMILIES_QUERY_KEY } from "@/hooks/useFontFamilies";
import {
  ConfigDetailPanel,
  ConfigInputField,
  ConfigMasterDetailLayout,
  ConfigNavigationGroup,
  ConfigNavigationItem,
  ConfigNavigationSidebar,
  ConfigNumberField,
  ConfigSectionCard,
  ConfigSelectField,
  ConfigStandardEmptyState,
  ConfigSwitchField,
  ConfigTextarea,
  ConfigTextareaField,
  EmptyConfigState,
  ReadOnlyInfoField,
  TaskRuntimeFields,
} from "./library-config-shared";
import {
  buildBuiltinLanguageItemID,
  buildCustomLanguageItemID,
  buildEditableCardID,
  buildRemoteStyleDocumentSourceRef,
  createEmptyGlossaryProfile,
  createEmptyPromptProfile,
  formatFontList,
  formatRemoteFontSourceNames,
  formatRemoteFontUnavailableCandidates,
  normalizeFontFamilyKey,
  normalizeGlossaryCategory,
  normalizePromptCategory,
  normalizeScopedLanguageValue,
  parseNonNegativeInt,
  parsePositiveInt,
  resolveFontSourceDisplayName,
  resolveFontSourceSummary,
  resolveGlossaryCategoryLabel,
  resolvePromptCategoryLabel,
  resolveScopedLanguageOptions,
  resolveSubtitleStyleDocumentFontCoverage,
  resolveSubtitleStylePresetFontCoverage,
  resolveSubtitleStylePresetReferencedFonts,
  resolveSubtitleStyleSourceKind,
  resolveSubtitleStyleSyncStatusLabel,
  splitAliases,
  summarizeGlossaryProfile,
  summarizeLanguageConfigItem,
  summarizePromptProfile,
  summarizeTaskRuntimeSettings,
  syncSelectedAssetItemId,
  type LanguageAssetTabId,
  type LanguageConfigItemKind,
  type RemoteFontSearchCandidate,
} from "./library-config-utils";
import {
  resolveBilingualStyles,
  resolveMonoStyles,
} from "../utils/subtitleStylePresets";
import {
  BUILTIN_PRESETS,
  buildPresetSummary,
  getSupportedAudioCodecs,
  getSupportedVideoCodecs,
  normalizePresetForOutput,
  resolveRecommendedAudioBitrateKbps,
  resolvePresetName,
} from "../utils/transcodePresets";
import {
  FCPXML_VERSION_OPTIONS,
  FCPXML_FRAME_DURATION_PRESETS,
  ITT_FRAME_RATE_PRESETS,
  createEmptySubtitleExportPreset,
  createEmptySubtitleStyleSource,
  duplicateSubtitleExportPreset,
  duplicateSubtitleStyleDocument,
  ensureBuiltInSubtitleStyleFontSources,
  FCPXML_START_TIMECODE_PRESETS,
  formatSubtitleStyleDocumentFeatureFlag,
  normalizeFCPXMLFrameDuration,
  normalizeFCPXMLStartTimecodeSeconds,
  normalizeFCPXMLVersion,
  normalizeSubtitleExportFormat,
  normalizeSubtitleExportMediaStrategy,
  resolveITTFrameRatePresetValue,
  resolveITTFrameTimingFromPresetValue,
  resolveDefaultBilingualStyle,
  resolveDefaultMonoStyle,
  resolveAssDocumentSummary,
  resolveSubtitleExportPresets,
  resolveSubtitleStyleDefaults,
  resolveSubtitleStyleDocuments,
  resolveSubtitleStyleSources,
  sortSubtitleStyleSources,
} from "../utils/subtitleStyles";
import { formatTemplate } from "../utils/i18n";

function ConditionalPanel({
  active,
  className,
  children,
}: {
  active: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  if (!active) {
    return null;
  }
  return <div className={className}>{children}</div>;
}

type LibraryConfigPageProps = {
  value: LibraryModuleConfigDTO;
  onChange: (next: LibraryModuleConfigDTO) => void;
  onRequestPersist?: () => void;
  onToolbarStateChange?: (state: LibraryConfigToolbarState | null) => void;
  requestedPage?: LibraryConfigPageId | null;
};

export type LibraryConfigToolbarState = {
  actions?: React.ReactNode;
};

export type LibraryConfigPageId =
  | "overview"
  | "task-runtime"
  | "languages"
  | "glossary"
  | "prompts"
  | "subtitle-styles"
  | "font-management"
  | "subtitle-export-presets"
  | "video-export-presets"
  | "remote-sources";

type TaskConfigScopeId = "translate" | "proofread";

type ConfigPageMeta = {
  id: LibraryConfigPageId;
  label: string;
  description: string;
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>;
  badge?: string;
};

type RemoteStyleManifestItem = {
  id: string;
  name: string;
  description?: string;
  version?: string;
  filePath?: string;
  fonts?: string[];
};

type RemoteStyleBrowserState = {
  loading: boolean;
  error: string;
  items: RemoteStyleManifestItem[];
};

type RemoteFontSearchState = {
  loading: boolean;
  error: string;
  candidates: RemoteFontSearchCandidate[];
};

type SyncRemoteFontSourceResult = {
  sourceId: string;
  sourceName: string;
  remoteFontManifest?: LibraryRemoteFontManifestDTO;
  fontCount: number;
  syncStatus: string;
  lastSyncedAt?: string;
  lastError?: string;
};

type RefreshFontCatalogResult = {
  familyCount?: number;
};

type LanguageConfigItem = {
  id: string;
  kind: LanguageConfigItemKind;
  language: LibraryTranslateLanguageDTO;
  rowID: string;
  customIndex: number;
};

const SUBTITLE_EXPORT_FORMAT_ORDER = [
  "srt",
  "vtt",
  "ass",
  "itt",
  "fcpxml",
] as const;
const SUBTITLE_EXPORT_PROFILE_FORMAT_OPTIONS = [
  { value: "srt", label: "SRT" },
  { value: "vtt", label: "VTT" },
  { value: "ass", label: "ASS/SSA" },
  { value: "itt", label: "ITT" },
  { value: "fcpxml", label: "FCPXML" },
] as const;
const GLOSSARY_CATEGORY_ORDER = ["all", "translate", "proofread"] as const;
const PROMPT_CATEGORY_ORDER = [
  "all",
  "translate",
  "proofread",
  "glossary",
] as const;

type VideoExportPresetDraft = Omit<TranscodePreset, "id"> & { id?: string };

const DEFAULT_VIDEO_EXPORT_PRESET_DRAFT: VideoExportPresetDraft = {
  name: "",
  outputType: "video",
  container: "mp4",
  videoCodec: "h264",
  audioCodec: "aac",
  qualityMode: "crf",
  crf: 18,
  audioBitrateKbps: 256,
  scale: "1080p",
  ffmpegPreset: "slow",
  allowUpscale: false,
  requiresVideo: true,
};

const DEFAULT_AUDIO_EXPORT_PRESET_DRAFT: VideoExportPresetDraft = {
  name: "",
  outputType: "audio",
  container: "mp3",
  audioCodec: "mp3",
  audioBitrateKbps: 320,
  allowUpscale: false,
  requiresAudio: true,
};

const VIDEO_PRESET_CONTAINER_OPTIONS = [
  { value: "mp4", label: "MP4" },
  { value: "mov", label: "MOV" },
  { value: "mkv", label: "MKV" },
  { value: "webm", label: "WebM" },
] as const;

const AUDIO_PRESET_CONTAINER_OPTIONS = [
  { value: "mp3", label: "MP3" },
  { value: "m4a", label: "M4A" },
  { value: "ogg", label: "OGG" },
  { value: "flac", label: "FLAC" },
  { value: "wav", label: "WAV" },
] as const;

const VIDEO_CODEC_OPTIONS = [
  { value: "h264", label: "H.264" },
  { value: "h265", label: "H.265 / HEVC" },
  { value: "vp9", label: "VP9" },
] as const;

const VIDEO_AUDIO_CODEC_OPTIONS = [
  { value: "aac", label: "AAC" },
  { value: "mp3", label: "MP3" },
  { value: "opus", label: "Opus" },
  { value: "copy", label: "Copy" },
] as const;

const AUDIO_CODEC_OPTIONS = [
  { value: "mp3", label: "MP3" },
  { value: "aac", label: "AAC" },
  { value: "opus", label: "Opus" },
  { value: "flac", label: "FLAC" },
  { value: "pcm", label: "PCM" },
] as const;

const VIDEO_SCALE_OPTIONS = [
  { value: "original", label: "Original" },
  { value: "2160p", label: "2160p" },
  { value: "1080p", label: "1080p" },
  { value: "720p", label: "720p" },
  { value: "480p", label: "480p" },
  { value: "custom", label: "Custom" },
] as const;

const FFMPEG_SPEED_PRESET_OPTIONS = [
  { value: "ultrafast", label: "ultrafast" },
  { value: "fast", label: "fast" },
  { value: "medium", label: "medium" },
  { value: "slow", label: "slow" },
] as const;

function resolveTranscodeOutputTypeLabel(
  outputType: TranscodePreset["outputType"],
  t: (key: string) => string,
) {
  return outputType === "audio"
    ? t("library.workspace.transcode.outputAudio")
    : t("library.workspace.transcode.outputVideo");
}

function resolveDefaultAudioCodecForContainer(container: string) {
  switch (container.trim().toLowerCase()) {
    case "m4a":
      return "aac";
    case "ogg":
      return "opus";
    case "flac":
      return "flac";
    case "wav":
      return "pcm";
    default:
      return "mp3";
  }
}

function resolveDefaultVideoCRF(videoCodec: string | undefined) {
  switch ((videoCodec ?? "").trim().toLowerCase()) {
    case "h265":
      return 20;
    case "vp9":
      return 20;
    default:
      return 18;
  }
}

function resolveCompactCodecToken(value: string | undefined) {
  switch ((value ?? "").trim().toLowerCase()) {
    case "h264":
      return "H.264";
    case "h265":
      return "H.265";
    case "vp9":
      return "VP9";
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
    default:
      return (value ?? "").toUpperCase();
  }
}

function resolveCompactPresetMeta(
  preset: TranscodePreset,
  t: (key: string) => string,
) {
  const parts = [preset.container.toUpperCase()];
  if (preset.outputType === "video") {
    if (preset.videoCodec) {
      parts.push(resolveCompactCodecToken(preset.videoCodec));
    }
    if (preset.scale && preset.scale !== "original") {
      parts.push(preset.scale.toUpperCase());
    } else {
      parts.push(t("library.workspace.transcode.summary.original"));
    }
  } else if (preset.audioCodec) {
    parts.push(resolveCompactCodecToken(preset.audioCodec));
    if (preset.audioBitrateKbps) {
      parts.push(`${preset.audioBitrateKbps}k`);
    } else {
      parts.push(t("library.config.videoExportPresets.lossless"));
    }
  }
  return parts.join(" · ");
}

function resolveSubtitleExportPresetFormat(
  profile: LibrarySubtitleExportPresetDTO,
) {
  return normalizeSubtitleExportFormat(profile.format ?? "srt");
}

function isBuiltInSubtitleExportPreset(
  profile: LibrarySubtitleExportPresetDTO | null | undefined,
) {
  return (profile?.id ?? "").startsWith("builtin-subtitle-export-preset-");
}

export function LibraryConfigPage({
  value,
  onChange,
  onRequestPersist,
  onToolbarStateChange,
  requestedPage,
}: LibraryConfigPageProps) {
  const { t, language } = useI18n();
  const queryClient = useQueryClient();
  const { data: systemFontFamilies, isLoading: isSystemFontsLoading } =
    useFontFamilies();
  const transcodePresetsQuery = useTranscodePresets();
  const saveTranscodePreset = useSaveTranscodePreset();
  const deleteTranscodePreset = useDeleteTranscodePreset();
  const [activePage, setActivePage] = React.useState<LibraryConfigPageId>(
    requestedPage ?? "overview",
  );
  const [activeTaskRuntimeTask, setActiveTaskRuntimeTask] =
    React.useState<TaskConfigScopeId>("translate");
  const [selectedSubtitleStyleDocumentId, setSelectedSubtitleStyleDocumentId] =
    React.useState("");
  const [selectedSubtitleExportPresetId, setSelectedSubtitleExportPresetId] =
    React.useState("");
  const [selectedVideoExportPresetId, setSelectedVideoExportPresetId] =
    React.useState("");
  const [subtitleStyleDocumentDraft, setSubtitleStyleDocumentDraft] =
    React.useState<LibrarySubtitleStyleDocumentDTO | null>(null);
  const [videoExportPresetDraft, setVideoExportPresetDraft] =
    React.useState<VideoExportPresetDraft | null>(null);
  const [videoExportPresetDraftMode, setVideoExportPresetDraftMode] =
    React.useState<"create" | "edit" | null>(null);
  const [taskRuntimeSearch, setTaskRuntimeSearch] = React.useState("");
  const [languageSearch, setLanguageSearch] = React.useState("");
  const [glossarySearch, setGlossarySearch] = React.useState("");
  const [promptSearch, setPromptSearch] = React.useState("");
  const [videoExportPresetSearch, setVideoExportPresetSearch] =
    React.useState("");
  const [subtitleExportPresetSearch, setSubtitleExportPresetSearch] =
    React.useState("");
  const [subtitleStyleToolbarActions, setSubtitleStyleToolbarActions] =
    React.useState<React.ReactNode | null>(null);
  const [remoteStyleBrowserState, setRemoteStyleBrowserState] = React.useState<
    Record<string, RemoteStyleBrowserState>
  >({});
  const [importingRemoteStyleItems, setImportingRemoteStyleItems] =
    React.useState<Record<string, true>>({});
  const [remoteFontSearchState, setRemoteFontSearchState] = React.useState<
    Record<string, RemoteFontSearchState>
  >({});
  const [repairingFontFamilies, setRepairingFontFamilies] = React.useState<
    Record<string, true>
  >({});
  const [syncingFontSources, setSyncingFontSources] = React.useState<
    Record<string, true>
  >({});
  const [refreshingFontList, setRefreshingFontList] = React.useState(false);
  const customLanguageRowIDCounterRef = React.useRef(0);
  const createCustomLanguageRowID = React.useCallback(
    () =>
      `library-config-language-row-${customLanguageRowIDCounterRef.current++}`,
    [],
  );

  const builtinLanguages = value.translateLanguages.builtin ?? [];
  const customLanguages = value.translateLanguages.custom ?? [];
  const subtitleMonoStyles = React.useMemo(
    () => resolveMonoStyles(value),
    [value],
  );
  const subtitleBilingualStyles = React.useMemo(
    () => resolveBilingualStyles(value),
    [value],
  );
  const subtitlePresetCount =
    subtitleMonoStyles.length + subtitleBilingualStyles.length;
  const subtitleExportLanguageOptions = React.useMemo(() => {
    const seen = new Set<string>();
    const options: Array<{ value: string; label: string }> = [];
    for (const language of [...builtinLanguages, ...customLanguages]) {
      const code = language.code.trim();
      if (!code || seen.has(code)) {
        continue;
      }
      seen.add(code);
      const label = language.label.trim();
      options.push({
        value: code,
        label: label ? `${label} (${code})` : code,
      });
    }
    if (!seen.has("en-US")) {
      options.unshift({
        value: "en-US",
        label: `${t("settings.language.option.en")} (en-US)`,
      });
    }
    return options;
  }, [builtinLanguages, customLanguages, t]);
  const [customLanguageRowIDs, setCustomLanguageRowIDs] = React.useState<
    string[]
  >(() => customLanguages.map(() => createCustomLanguageRowID()));
  const [selectedAssetItemIds, setSelectedAssetItemIds] = React.useState<
    Record<LanguageAssetTabId, string>
  >({
    languages: "",
    glossary: "",
    prompts: "",
  });
  const [editingCards, setEditingCards] = React.useState<Record<string, true>>(
    {},
  );
  const glossaryProfiles = value.languageAssets.glossaryProfiles ?? [];
  const promptProfiles = value.languageAssets.promptProfiles ?? [];
  const subtitleStyleDocuments = React.useMemo(
    () => resolveSubtitleStyleDocuments(value),
    [value],
  );
  const subtitleStyleSources = React.useMemo(
    () => resolveSubtitleStyleSources(value),
    [value],
  );
  const subtitleExportPresets = React.useMemo(
    () => resolveSubtitleExportPresets(value),
    [value],
  );
  const videoExportPresets = React.useMemo(
    () => transcodePresetsQuery.data ?? BUILTIN_PRESETS,
    [transcodePresetsQuery.data],
  );
  const transcodePresetCount = videoExportPresets.length;
  const taskRuntimeItems = React.useMemo(
    () => [
      {
        id: "translate" as const,
        title: t("library.config.taskRuntime.translateTitle"),
        description: summarizeTaskRuntimeSettings(value.taskRuntime.translate),
      },
      {
        id: "proofread" as const,
        title: t("library.config.taskRuntime.proofreadTitle"),
        description: summarizeTaskRuntimeSettings(value.taskRuntime.proofread),
      },
    ],
    [language, t, value.taskRuntime.proofread, value.taskRuntime.translate],
  );
  const normalizedTaskRuntimeSearch = taskRuntimeSearch.trim().toLowerCase();
  const visibleTaskRuntimeItems = React.useMemo(() => {
    if (!normalizedTaskRuntimeSearch) {
      return taskRuntimeItems;
    }
    return taskRuntimeItems.filter((item) =>
      [item.title, item.description]
        .join(" ")
        .toLowerCase()
        .includes(normalizedTaskRuntimeSearch),
    );
  }, [normalizedTaskRuntimeSearch, taskRuntimeItems]);
  const normalizedSubtitleExportPresetSearch = subtitleExportPresetSearch
    .trim()
    .toLowerCase();
  const visibleSubtitleExportPresets = React.useMemo(() => {
    if (!normalizedSubtitleExportPresetSearch) {
      return subtitleExportPresets;
    }
    return subtitleExportPresets.filter((profile) => {
      const searchable = [
        profile.name ?? "",
        resolveSubtitleExportPresetFormat(profile),
        normalizeSubtitleExportMediaStrategy(profile.mediaStrategy ?? ""),
      ]
        .join(" ")
        .toLowerCase();
      return searchable.includes(normalizedSubtitleExportPresetSearch);
    });
  }, [normalizedSubtitleExportPresetSearch, subtitleExportPresets]);
  const groupedSubtitleExportPresets = React.useMemo(() => {
    const groups = new Map<string, LibrarySubtitleExportPresetDTO[]>();
    for (const profile of visibleSubtitleExportPresets) {
      const format = resolveSubtitleExportPresetFormat(profile);
      const items = groups.get(format);
      if (items) {
        items.push(profile);
      } else {
        groups.set(format, [profile]);
      }
    }
    const formatOrder = new Map<string, number>(
      SUBTITLE_EXPORT_FORMAT_ORDER.map((format, index) => [format, index]),
    );
    return Array.from(groups.entries())
      .map(([format, profiles]) => ({ format, profiles }))
      .sort((left, right) => {
        const leftOrder =
          formatOrder.get(left.format) ?? Number.MAX_SAFE_INTEGER;
        const rightOrder =
          formatOrder.get(right.format) ?? Number.MAX_SAFE_INTEGER;
        if (leftOrder !== rightOrder) {
          return leftOrder - rightOrder;
        }
        return left.format.localeCompare(right.format);
      });
  }, [visibleSubtitleExportPresets]);
  const subtitleExportMediaStrategyOptions = React.useMemo(
    () => [
      {
        value: "auto",
        label: t("library.config.subtitleStyles.mediaStrategyAuto"),
      },
      {
        value: "fixed",
        label: t("library.config.subtitleStyles.mediaStrategyFixed"),
      },
    ],
    [t],
  );
  const subtitleExportVttKindOptions = React.useMemo(
    () => [
      {
        value: "subtitles",
        label: t("library.config.subtitleStyles.vttKindSubtitles"),
      },
      {
        value: "captions",
        label: t("library.config.subtitleStyles.vttKindCaptions"),
      },
      {
        value: "descriptions",
        label: t("library.config.subtitleStyles.vttKindDescriptions"),
      },
    ],
    [t],
  );
  const resolveSubtitleExportMediaStrategyLabel = React.useCallback(
    (value: string) =>
      normalizeSubtitleExportMediaStrategy(value) === "fixed"
        ? t("library.config.subtitleStyles.mediaStrategyFixed")
        : t("library.config.subtitleStyles.mediaStrategyAuto"),
    [t],
  );
  const subtitleStyleDefaults = React.useMemo(
    () => resolveSubtitleStyleDefaults(value),
    [value],
  );
  const normalizedSystemFonts = React.useMemo(
    () =>
      new Set(
        (systemFontFamilies ?? [])
          .map((font) => normalizeFontFamilyKey(font))
          .filter(Boolean),
      ),
    [systemFontFamilies],
  );
  const subtitleFontCoverage = React.useMemo(
    () =>
      resolveSubtitleStylePresetFontCoverage(
        subtitleMonoStyles,
        subtitleBilingualStyles,
        normalizedSystemFonts,
        value.subtitleStyles.fonts ?? [],
      ),
    [
      normalizedSystemFonts,
      subtitleBilingualStyles,
      subtitleMonoStyles,
      value.subtitleStyles.fonts,
    ],
  );
  const subtitleReferencedFonts = React.useMemo(
    () =>
      resolveSubtitleStylePresetReferencedFonts(
        subtitleMonoStyles,
        subtitleBilingualStyles,
        normalizedSystemFonts,
        value.subtitleStyles.fonts ?? [],
      ),
    [
      normalizedSystemFonts,
      subtitleBilingualStyles,
      subtitleMonoStyles,
      value.subtitleStyles.fonts,
    ],
  );
  const subtitleStyleDocumentSources = React.useMemo(
    () =>
      subtitleStyleSources.filter(
        (source) => resolveSubtitleStyleSourceKind(source.kind) === "style",
      ),
    [subtitleStyleSources],
  );
  const subtitleStyleFontSources = React.useMemo(
    () =>
      subtitleStyleSources.filter(
        (source) => resolveSubtitleStyleSourceKind(source.kind) === "font",
      ),
    [subtitleStyleSources],
  );
  const selectedSubtitleStyleDocument = React.useMemo(
    () =>
      subtitleStyleDocuments.find(
        (document) => document.id === selectedSubtitleStyleDocumentId,
      ) ??
      subtitleStyleDocuments[0] ??
      null,
    [selectedSubtitleStyleDocumentId, subtitleStyleDocuments],
  );
  const selectedSubtitleExportPreset = React.useMemo(
    () =>
      subtitleExportPresets.find(
        (profile) => profile.id === selectedSubtitleExportPresetId,
      ) ??
      subtitleExportPresets[0] ??
      null,
    [selectedSubtitleExportPresetId, subtitleExportPresets],
  );
  const selectedVideoExportPreset = React.useMemo(
    () =>
      videoExportPresets.find(
        (preset) => preset.id === selectedVideoExportPresetId,
      ) ??
      videoExportPresets[0] ??
      null,
    [selectedVideoExportPresetId, videoExportPresets],
  );
  const languageOptions = React.useMemo(
    () => [...builtinLanguages, ...customLanguages],
    [builtinLanguages, customLanguages],
  );
  const glossaryLanguageScopeOptions = React.useMemo(
    () => resolveScopedLanguageOptions(languageOptions),
    [languageOptions],
  );
  const languageConfigItems = React.useMemo<LanguageConfigItem[]>(
    () => [
      ...builtinLanguages.map((language, index) => ({
        id: buildBuiltinLanguageItemID(language.code || `builtin-${index}`),
        kind: "builtin" as const,
        language,
        rowID: "",
        customIndex: -1,
      })),
      ...customLanguages.map((language, index) => {
        const rowID =
          customLanguageRowIDs[index] ??
          `library-config-language-row-fallback-${index}`;
        return {
          id: buildCustomLanguageItemID(rowID),
          kind: "custom" as const,
          language,
          rowID,
          customIndex: index,
        };
      }),
    ],
    [builtinLanguages, customLanguageRowIDs, customLanguages],
  );
  const normalizedLanguageSearch = languageSearch.trim().toLowerCase();
  const visibleLanguageConfigItems = React.useMemo(() => {
    if (!normalizedLanguageSearch) {
      return languageConfigItems;
    }
    return languageConfigItems.filter((item) =>
      [
        item.language.label ?? "",
        item.language.code ?? "",
        (item.language.aliases ?? []).join(" "),
        item.kind,
      ]
        .join(" ")
        .toLowerCase()
        .includes(normalizedLanguageSearch),
    );
  }, [languageConfigItems, normalizedLanguageSearch]);
  const groupedVisibleLanguageConfigItems = React.useMemo(
    () =>
      [
        {
          id: "builtin",
          title: t("library.config.translateLanguages.builtinBadge"),
          items: visibleLanguageConfigItems.filter(
            (item) => item.kind === "builtin",
          ),
        },
        {
          id: "custom",
          title: t("library.config.translateLanguages.customBadge"),
          items: visibleLanguageConfigItems.filter(
            (item) => item.kind === "custom",
          ),
        },
      ].filter((group) => group.items.length > 0),
    [t, visibleLanguageConfigItems],
  );
  const selectedLanguageItem = React.useMemo(
    () =>
      languageConfigItems.find(
        (item) => item.id === selectedAssetItemIds.languages,
      ) ??
      languageConfigItems[0] ??
      null,
    [languageConfigItems, selectedAssetItemIds.languages],
  );
  const selectedLanguage = selectedLanguageItem?.language ?? null;
  const selectedCustomLanguageIndex =
    selectedLanguageItem?.kind === "custom"
      ? selectedLanguageItem.customIndex
      : -1;
  const selectedLanguageCardID =
    selectedLanguageItem?.kind === "custom" && selectedLanguageItem.rowID
      ? buildEditableCardID("language", selectedLanguageItem.rowID)
      : "";
  const selectedGlossaryProfile = React.useMemo(
    () =>
      glossaryProfiles.find(
        (profile) => profile.id === selectedAssetItemIds.glossary,
      ) ??
      glossaryProfiles[0] ??
      null,
    [glossaryProfiles, selectedAssetItemIds.glossary],
  );
  const normalizedGlossarySearch = glossarySearch.trim().toLowerCase();
  const visibleGlossaryProfiles = React.useMemo(() => {
    if (!normalizedGlossarySearch) {
      return glossaryProfiles;
    }
    return glossaryProfiles.filter((profile) =>
      [
        profile.name ?? "",
        profile.description ?? "",
        resolveGlossaryCategoryLabel(profile.category),
        summarizeGlossaryProfile(profile, languageOptions),
        ...(profile.terms ?? []).flatMap((term) => [
          term.source ?? "",
          term.target ?? "",
          term.note ?? "",
        ]),
      ]
        .join(" ")
        .toLowerCase()
        .includes(normalizedGlossarySearch),
    );
  }, [glossaryProfiles, languageOptions, normalizedGlossarySearch]);
  const groupedVisibleGlossaryProfiles = React.useMemo(() => {
    const groups = new Map<
      string,
      { title: string; profiles: LibraryGlossaryProfileDTO[] }
    >();
    for (const category of GLOSSARY_CATEGORY_ORDER) {
      groups.set(category, {
        title: resolveGlossaryCategoryLabel(category),
        profiles: [],
      });
    }
    for (const profile of visibleGlossaryProfiles) {
      const category = normalizeGlossaryCategory(profile.category);
      const group = groups.get(category);
      if (group) {
        group.profiles.push(profile);
      }
    }
    return GLOSSARY_CATEGORY_ORDER.map((category) => ({
      id: category,
      ...groups.get(category)!,
    })).filter((group) => group.profiles.length > 0);
  }, [language, visibleGlossaryProfiles]);
  const selectedPromptProfile = React.useMemo(
    () =>
      promptProfiles.find(
        (profile) => profile.id === selectedAssetItemIds.prompts,
      ) ??
      promptProfiles[0] ??
      null,
    [promptProfiles, selectedAssetItemIds.prompts],
  );
  const normalizedPromptSearch = promptSearch.trim().toLowerCase();
  const visiblePromptProfiles = React.useMemo(() => {
    if (!normalizedPromptSearch) {
      return promptProfiles;
    }
    return promptProfiles.filter((profile) =>
      [
        profile.name ?? "",
        profile.description ?? "",
        profile.prompt ?? "",
        resolvePromptCategoryLabel(profile.category),
      ]
        .join(" ")
        .toLowerCase()
        .includes(normalizedPromptSearch),
    );
  }, [normalizedPromptSearch, promptProfiles]);
  const groupedVisiblePromptProfiles = React.useMemo(() => {
    const groups = new Map<
      string,
      { title: string; profiles: LibraryPromptProfileDTO[] }
    >();
    for (const category of PROMPT_CATEGORY_ORDER) {
      groups.set(category, {
        title: resolvePromptCategoryLabel(category),
        profiles: [],
      });
    }
    for (const profile of visiblePromptProfiles) {
      const category = normalizePromptCategory(profile.category);
      const group = groups.get(category);
      if (group) {
        group.profiles.push(profile);
      }
    }
    return PROMPT_CATEGORY_ORDER.map((category) => ({
      id: category,
      ...groups.get(category)!,
    })).filter((group) => group.profiles.length > 0);
  }, [language, visiblePromptProfiles]);

  React.useEffect(() => {
    return registerTopic(REALTIME_TOPICS.system.fonts, () => {
      queryClient.invalidateQueries({ queryKey: FONT_FAMILIES_QUERY_KEY });
      queryClient.invalidateQueries({ queryKey: FONT_CATALOG_QUERY_KEY });
    });
  }, [queryClient]);

  React.useEffect(() => {
    setRemoteFontSearchState({});
  }, [subtitleStyleFontSources]);

  const configPages = React.useMemo<ConfigPageMeta[]>(
    () => [
      {
        id: "overview",
        label: t("library.config.pages.overview"),
        description: t("library.config.pages.overviewDescription"),
        icon: MonitorPlay,
      },
      {
        id: "task-runtime",
        label: t("library.config.pages.taskRuntime"),
        description: t("library.config.pages.taskRuntimeDescription"),
        icon: SlidersHorizontal,
      },
      {
        id: "languages",
        label: t("library.config.pages.languages"),
        description: t("library.config.pages.languagesDescription"),
        icon: Languages,
        badge: String(languageConfigItems.length),
      },
      {
        id: "glossary",
        label: t("library.config.pages.glossary"),
        description: t("library.config.pages.glossaryDescription"),
        icon: BookOpen,
        badge: String(glossaryProfiles.length),
      },
      {
        id: "prompts",
        label: t("library.config.pages.prompts"),
        description: t("library.config.pages.promptsDescription"),
        icon: Sparkles,
        badge: String(promptProfiles.length),
      },
      {
        id: "subtitle-styles",
        label: t("library.config.pages.subtitleStyles"),
        description: t("library.config.pages.subtitleStylesDescription"),
        icon: Captions,
        badge: String(subtitlePresetCount),
      },
      {
        id: "font-management",
        label: t("library.config.pages.fontManagement"),
        description: t("library.config.pages.fontManagementDescription"),
        icon: Type,
        badge: String(subtitleFontCoverage.missing.length),
      },
      {
        id: "subtitle-export-presets",
        label: t("library.config.pages.subtitleExportPresets"),
        description: t("library.config.pages.subtitleExportPresetsDescription"),
        icon: Download,
        badge: String(subtitleExportPresets.length),
      },
      {
        id: "video-export-presets",
        label: t("library.config.pages.videoExportPresets"),
        description: t("library.config.pages.videoExportPresetsDescription"),
        icon: FileText,
        badge: String(transcodePresetCount),
      },
      {
        id: "remote-sources",
        label: t("library.config.pages.remoteSources"),
        description: t("library.config.pages.remoteSourcesDescription"),
        icon: Database,
        badge: String(
          subtitleStyleDocumentSources.length + subtitleStyleFontSources.length,
        ),
      },
    ],
    [
      glossaryProfiles.length,
      languageConfigItems.length,
      promptProfiles.length,
      subtitleExportPresets.length,
      transcodePresetCount,
      subtitleFontCoverage.missing.length,
      subtitlePresetCount,
      subtitleStyleDocumentSources.length,
      subtitleStyleFontSources.length,
      t,
    ],
  );

  const updateValue = React.useCallback(
    (patch: Partial<LibraryModuleConfigDTO>) => {
      onChange({
        ...value,
        ...patch,
      });
    },
    [onChange, value],
  );

  const updateCustomLanguages = React.useCallback(
    (nextCustom: LibraryTranslateLanguageDTO[]) => {
      updateValue({
        translateLanguages: {
          ...value.translateLanguages,
          custom: nextCustom,
        },
      });
    },
    [updateValue, value.translateLanguages],
  );

  React.useEffect(() => {
    setCustomLanguageRowIDs((current) => {
      if (current.length === customLanguages.length) {
        return current;
      }
      if (current.length > customLanguages.length) {
        return current.slice(0, customLanguages.length);
      }
      return [
        ...current,
        ...Array.from({ length: customLanguages.length - current.length }, () =>
          createCustomLanguageRowID(),
        ),
      ];
    });
  }, [createCustomLanguageRowID, customLanguages.length]);

  React.useEffect(() => {
    setSelectedAssetItemIds((current) =>
      syncSelectedAssetItemId(
        current,
        "languages",
        languageConfigItems.map((item) => item.id),
      ),
    );
  }, [languageConfigItems]);

  React.useEffect(() => {
    setSelectedAssetItemIds((current) =>
      syncSelectedAssetItemId(
        current,
        "glossary",
        glossaryProfiles.map((profile) => profile.id),
      ),
    );
  }, [glossaryProfiles]);

  React.useEffect(() => {
    setSelectedAssetItemIds((current) =>
      syncSelectedAssetItemId(
        current,
        "prompts",
        promptProfiles.map((profile) => profile.id),
      ),
    );
  }, [promptProfiles]);

  React.useEffect(() => {
    if (requestedPage && requestedPage !== activePage) {
      setActivePage(requestedPage);
    }
  }, [activePage, requestedPage]);

  React.useEffect(() => {
    if (!selectedSubtitleStyleDocumentId) {
      if (subtitleStyleDocuments[0]?.id) {
        setSelectedSubtitleStyleDocumentId(subtitleStyleDocuments[0].id);
      }
      return;
    }
    if (
      !subtitleStyleDocuments.some(
        (document) => document.id === selectedSubtitleStyleDocumentId,
      )
    ) {
      setSelectedSubtitleStyleDocumentId(subtitleStyleDocuments[0]?.id ?? "");
    }
  }, [selectedSubtitleStyleDocumentId, subtitleStyleDocuments]);

  React.useEffect(() => {
    if (!selectedSubtitleExportPresetId) {
      if (subtitleExportPresets[0]?.id) {
        setSelectedSubtitleExportPresetId(subtitleExportPresets[0].id);
      }
      return;
    }
    if (
      !subtitleExportPresets.some(
        (profile) => profile.id === selectedSubtitleExportPresetId,
      )
    ) {
      setSelectedSubtitleExportPresetId(subtitleExportPresets[0]?.id ?? "");
    }
  }, [selectedSubtitleExportPresetId, subtitleExportPresets]);

  React.useEffect(() => {
    if (!selectedVideoExportPresetId) {
      if (videoExportPresets[0]?.id) {
        setSelectedVideoExportPresetId(videoExportPresets[0].id);
      }
      return;
    }
    if (
      !videoExportPresets.some(
        (preset) => preset.id === selectedVideoExportPresetId,
      )
    ) {
      setSelectedVideoExportPresetId(videoExportPresets[0]?.id ?? "");
    }
  }, [selectedVideoExportPresetId, videoExportPresets]);

  React.useEffect(() => {
    if (!subtitleStyleDocumentDraft) {
      return;
    }
    if (
      !subtitleStyleDocuments.some(
        (document) => document.id === subtitleStyleDocumentDraft.id,
      )
    ) {
      setSubtitleStyleDocumentDraft(null);
    }
  }, [subtitleStyleDocumentDraft, subtitleStyleDocuments]);

  const openCardEditor = React.useCallback((cardID: string) => {
    setEditingCards((current) => {
      if (current[cardID]) {
        return current;
      }
      return {
        ...current,
        [cardID]: true,
      };
    });
  }, []);

  const closeCardEditor = React.useCallback((cardID: string) => {
    setEditingCards((current) => {
      if (!current[cardID]) {
        return current;
      }
      const next = { ...current };
      delete next[cardID];
      return next;
    });
  }, []);

  const isCardEditing = React.useCallback(
    (cardID: string) => Boolean(editingCards[cardID]),
    [editingCards],
  );

  const updateLanguageAssets = React.useCallback(
    (patch: Partial<LibraryModuleConfigDTO["languageAssets"]>) => {
      updateValue({
        languageAssets: {
          ...value.languageAssets,
          ...patch,
        },
      });
    },
    [updateValue, value.languageAssets],
  );

  const updateTaskRuntime = React.useCallback(
    (patch: Partial<LibraryModuleConfigDTO["taskRuntime"]>) => {
      updateValue({
        taskRuntime: {
          ...value.taskRuntime,
          ...patch,
        },
      });
    },
    [updateValue, value.taskRuntime],
  );

  const updateSubtitleStyles = React.useCallback(
    (patch: Partial<LibraryModuleConfigDTO["subtitleStyles"]>) => {
      updateValue({
        subtitleStyles: {
          ...value.subtitleStyles,
          ...patch,
        },
      });
    },
    [updateValue, value.subtitleStyles],
  );

  const handleAddLanguage = React.useCallback(() => {
    const rowID = createCustomLanguageRowID();
    setCustomLanguageRowIDs((current) => [...current, rowID]);
    setSelectedAssetItemIds((current) => ({
      ...current,
      languages: buildCustomLanguageItemID(rowID),
    }));
    openCardEditor(buildEditableCardID("language", rowID));
    updateCustomLanguages([
      ...customLanguages,
      { code: "", label: "", aliases: [] },
    ]);
  }, [
    createCustomLanguageRowID,
    customLanguages,
    openCardEditor,
    updateCustomLanguages,
  ]);

  const handleChangeLanguage = React.useCallback(
    (index: number, patch: Partial<LibraryTranslateLanguageDTO>) => {
      updateCustomLanguages(
        customLanguages.map((item, itemIndex) =>
          itemIndex === index
            ? {
                ...item,
                ...patch,
              }
            : item,
        ),
      );
    },
    [customLanguages, updateCustomLanguages],
  );

  const handleDeleteLanguage = React.useCallback(
    (index: number) => {
      const rowID = customLanguageRowIDs[index];
      if (rowID) {
        closeCardEditor(buildEditableCardID("language", rowID));
      }
      setCustomLanguageRowIDs((current) =>
        current.filter((_, itemIndex) => itemIndex !== index),
      );
      updateCustomLanguages(
        customLanguages.filter((_, itemIndex) => itemIndex !== index),
      );
    },
    [
      closeCardEditor,
      customLanguageRowIDs,
      customLanguages,
      updateCustomLanguages,
    ],
  );

  const updateGlossaryProfiles = React.useCallback(
    (nextProfiles: LibraryGlossaryProfileDTO[]) => {
      updateLanguageAssets({ glossaryProfiles: nextProfiles });
    },
    [updateLanguageAssets],
  );

  const updatePromptProfiles = React.useCallback(
    (nextProfiles: LibraryPromptProfileDTO[]) => {
      updateLanguageAssets({ promptProfiles: nextProfiles });
    },
    [updateLanguageAssets],
  );

  const handleAddGlossaryProfile = React.useCallback(() => {
    const profile = createEmptyGlossaryProfile();
    setSelectedAssetItemIds((current) => ({
      ...current,
      glossary: profile.id,
    }));
    openCardEditor(buildEditableCardID("glossary", profile.id));
    updateGlossaryProfiles([...glossaryProfiles, profile]);
  }, [glossaryProfiles, openCardEditor, updateGlossaryProfiles]);

  const handleUpdateGlossaryProfile = React.useCallback(
    (profileId: string, patch: Partial<LibraryGlossaryProfileDTO>) => {
      updateGlossaryProfiles(
        glossaryProfiles.map((profile) =>
          profile.id === profileId ? { ...profile, ...patch } : profile,
        ),
      );
    },
    [glossaryProfiles, updateGlossaryProfiles],
  );

  const handleDeleteGlossaryProfile = React.useCallback(
    (profileId: string) => {
      closeCardEditor(buildEditableCardID("glossary", profileId));
      updateGlossaryProfiles(
        glossaryProfiles.filter((profile) => profile.id !== profileId),
      );
    },
    [closeCardEditor, glossaryProfiles, updateGlossaryProfiles],
  );

  const handleAddGlossaryTerm = React.useCallback(
    (profileId: string) => {
      updateGlossaryProfiles(
        glossaryProfiles.map((profile) =>
          profile.id === profileId
            ? {
                ...profile,
                terms: [
                  ...(profile.terms ?? []),
                  { source: "", target: "", note: "" },
                ],
              }
            : profile,
        ),
      );
    },
    [glossaryProfiles, updateGlossaryProfiles],
  );

  const handleUpdateGlossaryTerm = React.useCallback(
    (
      profileId: string,
      termIndex: number,
      patch: Partial<LibraryGlossaryTermDTO>,
    ) => {
      updateGlossaryProfiles(
        glossaryProfiles.map((profile) => {
          if (profile.id !== profileId) {
            return profile;
          }
          return {
            ...profile,
            terms: (profile.terms ?? []).map((term, currentIndex) =>
              currentIndex === termIndex
                ? {
                    ...term,
                    ...patch,
                  }
                : term,
            ),
          };
        }),
      );
    },
    [glossaryProfiles, updateGlossaryProfiles],
  );

  const handleDeleteGlossaryTerm = React.useCallback(
    (profileId: string, termIndex: number) => {
      updateGlossaryProfiles(
        glossaryProfiles.map((profile) => {
          if (profile.id !== profileId) {
            return profile;
          }
          return {
            ...profile,
            terms: (profile.terms ?? []).filter(
              (_, currentIndex) => currentIndex !== termIndex,
            ),
          };
        }),
      );
    },
    [glossaryProfiles, updateGlossaryProfiles],
  );

  const handleAddPromptProfile = React.useCallback(() => {
    const profile = createEmptyPromptProfile();
    setSelectedAssetItemIds((current) => ({
      ...current,
      prompts: profile.id,
    }));
    openCardEditor(buildEditableCardID("prompt", profile.id));
    updatePromptProfiles([...promptProfiles, profile]);
  }, [openCardEditor, promptProfiles, updatePromptProfiles]);

  const handleUpdatePromptProfile = React.useCallback(
    (profileId: string, patch: Partial<LibraryPromptProfileDTO>) => {
      updatePromptProfiles(
        promptProfiles.map((profile) =>
          profile.id === profileId ? { ...profile, ...patch } : profile,
        ),
      );
    },
    [promptProfiles, updatePromptProfiles],
  );

  const handleDeletePromptProfile = React.useCallback(
    (profileId: string) => {
      closeCardEditor(buildEditableCardID("prompt", profileId));
      updatePromptProfiles(
        promptProfiles.filter((profile) => profile.id !== profileId),
      );
    },
    [closeCardEditor, promptProfiles, updatePromptProfiles],
  );

  const updateSubtitleStyleDocuments = React.useCallback(
    (nextDocuments: LibrarySubtitleStyleDocumentDTO[]) => {
      void nextDocuments;
    },
    [],
  );

  const updateSubtitleExportPresets = React.useCallback(
    (nextProfiles: LibrarySubtitleExportPresetDTO[]) => {
      updateSubtitleStyles({ subtitleExportPresets: nextProfiles });
    },
    [updateSubtitleStyles],
  );

  const updateSubtitleStyleSources = React.useCallback(
    (nextSources: LibrarySubtitleStyleSourceDTO[]) => {
      updateSubtitleStyles({ sources: sortSubtitleStyleSources(nextSources) });
    },
    [updateSubtitleStyles],
  );

  const handleCreateDuplicateSubtitleStyleDocument = React.useCallback(
    (document: LibrarySubtitleStyleDocumentDTO) => {
      const duplicate = duplicateSubtitleStyleDocument(document);
      if (subtitleStyleDocumentDraft?.id) {
        closeCardEditor(
          buildEditableCardID(
            "subtitle-style-document",
            subtitleStyleDocumentDraft.id,
          ),
        );
      }
      setActivePage("subtitle-styles");
      setSelectedSubtitleStyleDocumentId(duplicate.id);
      setSubtitleStyleDocumentDraft(duplicate);
      openCardEditor(
        buildEditableCardID("subtitle-style-document", duplicate.id),
      );
      updateSubtitleStyleDocuments([duplicate, ...subtitleStyleDocuments]);
    },
    [
      closeCardEditor,
      openCardEditor,
      subtitleStyleDocumentDraft,
      subtitleStyleDocuments,
      updateSubtitleStyleDocuments,
    ],
  );

  const handleUpdateSubtitleStyleDocument = React.useCallback(
    (documentId: string, patch: Partial<LibrarySubtitleStyleDocumentDTO>) => {
      updateSubtitleStyleDocuments(
        subtitleStyleDocuments.map((document) =>
          document.id === documentId ? { ...document, ...patch } : document,
        ),
      );
    },
    [subtitleStyleDocuments, updateSubtitleStyleDocuments],
  );

  const handleDeleteSubtitleStyleDocument = React.useCallback(
    (documentId: string) => {
      setSubtitleStyleDocumentDraft((current) =>
        current?.id === documentId ? null : current,
      );
      closeCardEditor(
        buildEditableCardID("subtitle-style-document", documentId),
      );
      updateSubtitleStyleDocuments(
        subtitleStyleDocuments.filter((document) => document.id !== documentId),
      );
    },
    [closeCardEditor, subtitleStyleDocuments, updateSubtitleStyleDocuments],
  );

  const handleStartSubtitleStyleDocumentEdit = React.useCallback(
    (document: LibrarySubtitleStyleDocumentDTO) => {
      setSubtitleStyleDocumentDraft({ ...document });
      openCardEditor(
        buildEditableCardID("subtitle-style-document", document.id),
      );
    },
    [openCardEditor],
  );

  const handleCancelSubtitleStyleDocumentEdit = React.useCallback(
    (documentId: string) => {
      setSubtitleStyleDocumentDraft((current) =>
        current?.id === documentId ? null : current,
      );
      closeCardEditor(
        buildEditableCardID("subtitle-style-document", documentId),
      );
    },
    [closeCardEditor],
  );

  const handleSaveSubtitleStyleDocumentEdit = React.useCallback(
    (documentId: string) => {
      const draft = subtitleStyleDocumentDraft;
      if (!draft || draft.id !== documentId) {
        closeCardEditor(
          buildEditableCardID("subtitle-style-document", documentId),
        );
        return;
      }
      handleUpdateSubtitleStyleDocument(documentId, draft);
      setSubtitleStyleDocumentDraft(null);
      closeCardEditor(
        buildEditableCardID("subtitle-style-document", documentId),
      );
    },
    [
      closeCardEditor,
      handleUpdateSubtitleStyleDocument,
      subtitleStyleDocumentDraft,
    ],
  );

  const handleAddSubtitleExportPreset = React.useCallback(
    (format: string) => {
      const profile = createEmptySubtitleExportPreset(format);
      const profileCardID = buildEditableCardID(
        "subtitle-export-preset",
        profile.id,
      );
      closeCardEditor(profileCardID);
      openCardEditor(profileCardID);
      const normalizedFormat = resolveSubtitleExportPresetFormat(profile);
      const formatLabel =
        SUBTITLE_EXPORT_PROFILE_FORMAT_OPTIONS.find(
          (option) => option.value === normalizedFormat,
        )?.label ?? normalizedFormat.toUpperCase();
      const nextName =
        normalizedFormat === "fcpxml"
          ? `${formatLabel} ${resolveSubtitleExportMediaStrategyLabel("auto")}`
          : `${formatLabel} ${t("library.config.subtitleStyles.exportProfileDefaultName")}`;
      const profileWithName = { ...profile, name: nextName };
      setActivePage("subtitle-export-presets");
      setSelectedSubtitleExportPresetId(profile.id);
      updateSubtitleExportPresets([
        profileWithName,
        ...subtitleExportPresets,
      ]);
    },
    [
      closeCardEditor,
      openCardEditor,
      resolveSubtitleExportMediaStrategyLabel,
      subtitleExportPresets,
      t,
      updateSubtitleExportPresets,
    ],
  );

  const handleUpdateSubtitleExportPreset = React.useCallback(
    (profileId: string, patch: Partial<LibrarySubtitleExportPresetDTO>) => {
      updateSubtitleExportPresets(
        subtitleExportPresets.map((profile) =>
          profile.id === profileId ? { ...profile, ...patch } : profile,
        ),
      );
    },
    [subtitleExportPresets, updateSubtitleExportPresets],
  );

  const handleDuplicateSubtitleExportPreset = React.useCallback(
    (profile: LibrarySubtitleExportPresetDTO) => {
      const duplicate = duplicateSubtitleExportPreset(profile);
      setActivePage("subtitle-export-presets");
      setSelectedSubtitleExportPresetId(duplicate.id);
      updateSubtitleExportPresets([duplicate, ...subtitleExportPresets]);
    },
    [subtitleExportPresets, updateSubtitleExportPresets],
  );

  const handleDeleteSubtitleExportPreset = React.useCallback(
    (profileId: string) => {
      const targetProfile = subtitleExportPresets.find(
        (profile) => profile.id === profileId,
      );
      if (isBuiltInSubtitleExportPreset(targetProfile)) {
        return;
      }
      closeCardEditor(buildEditableCardID("subtitle-export-preset", profileId));
      const nextProfiles = subtitleExportPresets.filter(
        (profile) => profile.id !== profileId,
      );
      const nextDefaultProfileID =
        subtitleStyleDefaults.subtitleExportPresetId === profileId
          ? (nextProfiles[0]?.id ?? "")
          : subtitleStyleDefaults.subtitleExportPresetId;
      updateSubtitleStyles({
        subtitleExportPresets: nextProfiles,
        defaults: {
          ...value.subtitleStyles.defaults,
          subtitleExportPresetId: nextDefaultProfileID,
        },
      });
    },
    [
      closeCardEditor,
      subtitleExportPresets,
      subtitleStyleDefaults.subtitleExportPresetId,
      updateSubtitleStyles,
      value.subtitleStyles.defaults,
    ],
  );

  const handleCreateVideoExportPreset = React.useCallback(
    (outputType: TranscodePreset["outputType"]) => {
      setActivePage("video-export-presets");
      setVideoExportPresetDraftMode("create");
      setVideoExportPresetDraft(
        outputType === "audio"
          ? {
              ...DEFAULT_AUDIO_EXPORT_PRESET_DRAFT,
              name: t("library.config.videoExportPresets.audioDefaultName"),
            }
          : {
              ...DEFAULT_VIDEO_EXPORT_PRESET_DRAFT,
              name: t("library.config.videoExportPresets.videoDefaultName"),
            },
      );
    },
    [t],
  );

  const handleEditVideoExportPreset = React.useCallback(
    (preset: TranscodePreset) => {
      if (preset.isBuiltin) {
        return;
      }
      setActivePage("video-export-presets");
      setSelectedVideoExportPresetId(preset.id);
      setVideoExportPresetDraftMode("edit");
      setVideoExportPresetDraft({ ...preset });
    },
    [],
  );

  const handleDuplicateVideoExportPreset = React.useCallback(
    (preset: TranscodePreset) => {
      setActivePage("video-export-presets");
      setSelectedVideoExportPresetId(preset.id);
      setVideoExportPresetDraftMode("create");
      setVideoExportPresetDraft({
        ...preset,
        id: undefined,
        isBuiltin: false,
        name: `${resolvePresetName(preset, t)} ${t("library.config.videoExportPresets.copySuffix")}`,
      });
    },
    [t],
  );

  const handleCancelVideoExportPresetEdit = React.useCallback(() => {
    setVideoExportPresetDraft(null);
    setVideoExportPresetDraftMode(null);
  }, []);

  const handleSaveVideoExportPreset = React.useCallback(async () => {
    if (!videoExportPresetDraft) {
      return;
    }
    const baseName =
      videoExportPresetDraft.name.trim() ||
      (videoExportPresetDraft.outputType === "audio"
        ? t("library.config.videoExportPresets.audioUntitled")
        : t("library.config.videoExportPresets.videoUntitled"));
    const normalized = normalizePresetForOutput(
      {
        ...videoExportPresetDraft,
        id: videoExportPresetDraft.id ?? "",
        name: baseName,
      } as TranscodePreset,
      videoExportPresetDraft.outputType,
    );
    try {
      const saved = await saveTranscodePreset.mutateAsync({
        ...normalized,
        id:
          videoExportPresetDraft.id && !videoExportPresetDraft.isBuiltin
            ? videoExportPresetDraft.id
            : "",
        name: baseName,
        isBuiltin: false,
      });
      setSelectedVideoExportPresetId(saved.id);
      setVideoExportPresetDraft(null);
      setVideoExportPresetDraftMode(null);
      messageBus.publishToast({
        intent: "success",
        title: t("library.config.videoExportPresets.savedTitle"),
        description: t("library.config.videoExportPresets.savedDescription"),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.config.videoExportPresets.saveFailedTitle"),
        description:
          error instanceof Error
            ? error.message
            : t("library.errors.unknown"),
      });
    }
  }, [saveTranscodePreset, t, videoExportPresetDraft]);

  const handleDeleteVideoExportPreset = React.useCallback(
    async (preset: TranscodePreset) => {
      if (preset.isBuiltin) {
        return;
      }
      try {
        await deleteTranscodePreset.mutateAsync({ id: preset.id });
        if (selectedVideoExportPresetId === preset.id) {
          setSelectedVideoExportPresetId("");
        }
        setVideoExportPresetDraft(null);
        setVideoExportPresetDraftMode(null);
        messageBus.publishToast({
          intent: "success",
          title: t("library.config.videoExportPresets.deletedTitle"),
          description: t("library.config.videoExportPresets.deletedDescription"),
        });
      } catch (error) {
        messageBus.publishToast({
          intent: "danger",
          title: t("library.config.videoExportPresets.deleteFailedTitle"),
          description:
            error instanceof Error
              ? error.message
              : t("library.errors.unknown"),
        });
      }
    },
    [deleteTranscodePreset, selectedVideoExportPresetId, t],
  );

  const handleAddSubtitleStyleSource = React.useCallback(() => {
    const source = createEmptySubtitleStyleSource();
    updateSubtitleStyleSources([...subtitleStyleSources, source]);
    onRequestPersist?.();
  }, [onRequestPersist, subtitleStyleSources, updateSubtitleStyleSources]);

  const handleAddSubtitleStyleFontSources = React.useCallback(() => {
    const nextSources = ensureBuiltInSubtitleStyleFontSources(
      subtitleStyleSources,
    );
    if (nextSources.length === subtitleStyleSources.length) {
      messageBus.publishToast({
        intent: "info",
        title: t("library.config.subtitleStyles.fontSourcesTitle"),
        description: t(
          "library.config.subtitleStyles.fontSourcesAlreadyAvailable",
        ),
      });
      return;
    }
    updateSubtitleStyleSources(nextSources);
    onRequestPersist?.();
  }, [onRequestPersist, subtitleStyleSources, t, updateSubtitleStyleSources]);

  const handleUpdateSubtitleStyleSource = React.useCallback(
    (sourceId: string, patch: Partial<LibrarySubtitleStyleSourceDTO>) => {
      updateSubtitleStyleSources(
        subtitleStyleSources.map((source) => {
          if (source.id !== sourceId) {
            return source;
          }
          if (resolveSubtitleStyleSourceKind(source.kind) !== "font") {
            return { ...source, ...patch };
          }
          return {
            ...source,
            ...patch,
            provider: "fontget",
            owner: "",
            repo: "",
            ref: "",
            manifestPath: "",
          };
        }),
      );
    },
    [subtitleStyleSources, updateSubtitleStyleSources],
  );

  const handleDeleteSubtitleStyleSource = React.useCallback(
    (sourceId: string) => {
      const source = subtitleStyleSources.find(
        (candidate) => candidate.id === sourceId,
      );
      if (source?.builtIn === true) {
        return;
      }
      closeCardEditor(buildEditableCardID("subtitle-style-source", sourceId));
      updateSubtitleStyleSources(
        subtitleStyleSources.filter((source) => source.id !== sourceId),
      );
      onRequestPersist?.();
    },
    [
      closeCardEditor,
      onRequestPersist,
      subtitleStyleSources,
      updateSubtitleStyleSources,
    ],
  );

  const handleUpdateSubtitleStyleDefault = React.useCallback(
    (
      field: "monoStyleId" | "bilingualStyleId" | "subtitleExportPresetId",
      nextValue: string,
    ) => {
      updateSubtitleStyles({
        defaults: {
          ...value.subtitleStyles.defaults,
          [field]: nextValue,
        },
      });
    },
    [updateSubtitleStyles, value.subtitleStyles.defaults],
  );

  const handleBrowseSubtitleStyleSource = React.useCallback(
    async (source: LibrarySubtitleStyleSourceDTO) => {
      setRemoteStyleBrowserState((current) => ({
        ...current,
        [source.id]: {
          loading: true,
          error: "",
          items: current[source.id]?.items ?? [],
        },
      }));
      try {
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.LibraryHandler.BrowseSubtitleStyleRemoteSource",
          { source },
        );
        const items = Array.isArray(result)
          ? (result as RemoteStyleManifestItem[])
          : [];
        setRemoteStyleBrowserState((current) => ({
          ...current,
          [source.id]: {
            loading: false,
            error: "",
            items,
          },
        }));
      } catch (error) {
        const message =
          error instanceof Error ? error.message : String(error ?? "");
        setRemoteStyleBrowserState((current) => ({
          ...current,
          [source.id]: {
            loading: false,
            error: message,
            items: current[source.id]?.items ?? [],
          },
        }));
        messageBus.publishToast({
          intent: "danger",
          title: t("library.config.subtitleStyles.browseSourceFailedTitle"),
          description: message,
        });
      }
    },
    [t],
  );

  const handleImportRemoteSubtitleStyleItem = React.useCallback(
    async (
      source: LibrarySubtitleStyleSourceDTO,
      item: RemoteStyleManifestItem,
    ) => {
      const actionKey = `${source.id}:${item.id}`;
      setImportingRemoteStyleItems((current) => ({
        ...current,
        [actionKey]: true,
      }));
      try {
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.LibraryHandler.ImportSubtitleStyleRemoteSourceItem",
          {
            source,
            itemId: item.id,
          },
        );
        const importedDocument = result as LibrarySubtitleStyleDocumentDTO;
        const existingDocument = subtitleStyleDocuments.find(
          (document) =>
            document.source === "remote" &&
            document.sourceRef === importedDocument.sourceRef,
        );
        const nextDocument = existingDocument
          ? {
              ...existingDocument,
              ...importedDocument,
              id: existingDocument.id,
            }
          : importedDocument;
        updateSubtitleStyleDocuments(
          existingDocument
            ? subtitleStyleDocuments.map((document) =>
                document.id === existingDocument.id ? nextDocument : document,
              )
            : [nextDocument, ...subtitleStyleDocuments],
        );
        if (subtitleStyleDocumentDraft?.id === existingDocument?.id) {
          setSubtitleStyleDocumentDraft(null);
        }
        setSelectedSubtitleStyleDocumentId(nextDocument.id);
        setActivePage("subtitle-styles");
        onRequestPersist?.();
        messageBus.publishToast({
          intent: "success",
          title: t("library.config.subtitleStyles.importSourceItemSuccessTitle"),
          description: existingDocument
            ? t("library.config.subtitleStyles.importSourceItemUpdated")
            : t("library.config.subtitleStyles.importSourceItemCreated"),
        });
      } catch (error) {
        messageBus.publishToast({
          intent: "danger",
          title: t("library.config.subtitleStyles.importSourceItemFailedTitle"),
          description:
            error instanceof Error ? error.message : String(error ?? ""),
        });
      } finally {
        setImportingRemoteStyleItems((current) => {
          const next = { ...current };
          delete next[actionKey];
          return next;
        });
      }
    },
    [
      onRequestPersist,
      subtitleStyleDocumentDraft?.id,
      subtitleStyleDocuments,
      t,
      updateSubtitleStyleDocuments,
    ],
  );

  const handleSearchRemoteFontCandidates = React.useCallback(
    async (family: string) => {
      const normalizedFamily = family.trim();
      if (!normalizedFamily) {
        return;
      }
      const enabledSources = subtitleStyleFontSources.filter(
        (source) => source.enabled === true,
      );
      if (enabledSources.length === 0) {
        setRemoteFontSearchState((current) => ({
          ...current,
          [normalizedFamily]: {
            loading: false,
            error: "",
            candidates: [],
          },
        }));
        return;
      }

      setRemoteFontSearchState((current) => ({
        ...current,
        [normalizedFamily]: {
          loading: true,
          error: "",
          candidates: current[normalizedFamily]?.candidates ?? [],
        },
      }));

      try {
        const result = await Call.ByName(
          "dreamcreator/internal/presentation/wails.SystemHandler.SearchRemoteFontFamily",
          {
            family: normalizedFamily,
            sources: enabledSources,
          },
        );
        const candidates = Array.isArray(result)
          ? (result as RemoteFontSearchCandidate[])
          : [];
        setRemoteFontSearchState((current) => ({
          ...current,
          [normalizedFamily]: {
            loading: false,
            error: "",
            candidates,
          },
        }));
      } catch (error) {
        setRemoteFontSearchState((current) => ({
          ...current,
          [normalizedFamily]: {
            loading: false,
            error: error instanceof Error ? error.message : String(error ?? ""),
            candidates: [],
          },
        }));
      }
    },
    [subtitleStyleFontSources],
  );

  const syncFontCatalogQueries = React.useCallback(async () => {
    await Promise.all([
      queryClient.invalidateQueries({
        queryKey: FONT_FAMILIES_QUERY_KEY,
        refetchType: "all",
      }),
      queryClient.invalidateQueries({
        queryKey: FONT_CATALOG_QUERY_KEY,
        refetchType: "all",
      }),
    ]);
  }, [queryClient]);

  const handleRefreshFontList = React.useCallback(async () => {
    setRefreshingFontList(true);
    try {
      const result = (await Call.ByName(
        "dreamcreator/internal/presentation/wails.SystemHandler.RefreshFontCatalog",
      )) as RefreshFontCatalogResult;
      await syncFontCatalogQueries();
      messageBus.publishToast({
        intent: "success",
        title: t("library.config.subtitleStyles.refreshFontListSuccessTitle"),
        description: formatTemplate(
          t("library.config.subtitleStyles.refreshFontListSuccessDescription"),
          {
            count: typeof result?.familyCount === "number" ? result.familyCount : 0,
          },
        ),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.config.subtitleStyles.refreshFontListFailedTitle"),
        description:
          error instanceof Error ? error.message : String(error ?? ""),
      });
    } finally {
      setRefreshingFontList(false);
    }
  }, [syncFontCatalogQueries, t]);

  const handleRepairSubtitleStyleFont = React.useCallback(
    async (family: string, target: "user" | "machine", sourceId?: string) => {
      const normalizedFamily = family.trim();
      if (!normalizedFamily) {
        return;
      }
      const actionKey = `${target}:${normalizedFamily}`;
      setRepairingFontFamilies((current) => ({
        ...current,
        [actionKey]: true,
      }));
      try {
        const result = (await Call.ByName(
          "dreamcreator/internal/presentation/wails.SystemHandler.InstallRemoteFontFamily",
          {
            family: normalizedFamily,
            target,
            sourceId: sourceId ?? "",
            sources: subtitleStyleFontSources.filter(
              (source) => source.enabled === true,
            ),
          },
        )) as { target?: string };
        const installedTarget =
          result?.target === "machine" ? "machine" : "user";
        messageBus.publishToast({
          intent: "success",
          title:
            installedTarget === "machine"
              ? t("library.config.subtitleStyles.installMachineFontSuccessTitle")
              : t("library.config.subtitleStyles.installUserFontSuccessTitle"),
          description:
            installedTarget === "machine"
              ? t("library.config.subtitleStyles.installMachineFontSuccessDescription")
              : t("library.config.subtitleStyles.installUserFontSuccessDescription"),
        });
        try {
          await syncFontCatalogQueries();
        } catch (refreshError) {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.config.subtitleStyles.refreshFontListFailedTitle"),
            description:
              refreshError instanceof Error
                ? refreshError.message
                : String(refreshError ?? ""),
          });
        }
      } catch (error) {
        messageBus.publishToast({
          intent: "danger",
          title:
            target === "machine"
              ? t("library.config.subtitleStyles.installMachineFontFailedTitle")
              : t("library.config.subtitleStyles.installUserFontFailedTitle"),
          description:
            error instanceof Error ? error.message : String(error ?? ""),
        });
      } finally {
        setRepairingFontFamilies((current) => {
          const next = { ...current };
          delete next[actionKey];
          return next;
        });
      }
    },
    [subtitleStyleFontSources, syncFontCatalogQueries, t],
  );

  const handleSyncSubtitleStyleFontSource = React.useCallback(
    async (source: LibrarySubtitleStyleSourceDTO) => {
      const sourceId = source.id?.trim();
      if (!sourceId) {
        return;
      }

      setSyncingFontSources((current) => ({
        ...current,
        [sourceId]: true,
      }));

      try {
        const result = (await Call.ByName(
          "dreamcreator/internal/presentation/wails.SystemHandler.SyncRemoteFontSource",
          {
            source,
          },
        )) as SyncRemoteFontSourceResult;

        updateSubtitleStyleSources(
          subtitleStyleSources.map((candidate) =>
            candidate.id === sourceId
              ? {
                  ...candidate,
                  name:
                    result?.remoteFontManifest?.sourceInfo?.name?.trim() ||
                    candidate.name,
                  remoteFontManifest:
                    result?.remoteFontManifest ?? candidate.remoteFontManifest,
                  enabled: true,
                  fontCount:
                    typeof result?.remoteFontManifest?.sourceInfo
                      ?.totalFonts === "number"
                      ? result.remoteFontManifest.sourceInfo.totalFonts
                      : typeof result?.fontCount === "number"
                        ? result.fontCount
                        : candidate.fontCount,
                  syncStatus: result?.syncStatus?.trim() || "synced",
                  lastSyncedAt:
                    result?.lastSyncedAt?.trim() ||
                    candidate.lastSyncedAt ||
                    "",
                  lastError: result?.lastError?.trim() || "",
                }
              : candidate,
          ),
        );
        window.setTimeout(() => onRequestPersist?.(), 0);

        if (result?.syncStatus === "error") {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.config.subtitleStyles.syncFontSourceFailedTitle"),
            description:
              result.lastError?.trim() ||
              t("library.config.subtitleStyles.syncFontSourceFailedFallback"),
          });
          return;
        }

        messageBus.publishToast({
          intent: "success",
          title: t("library.config.subtitleStyles.syncFontSourceSuccessTitle"),
          description: t("library.config.subtitleStyles.syncFontSourceSuccessDescription"),
        });
      } catch (error) {
        const description =
          error instanceof Error ? error.message : String(error ?? "");
        updateSubtitleStyleSources(
          subtitleStyleSources.map((candidate) =>
            candidate.id === sourceId
              ? {
                  ...candidate,
                  enabled: true,
                  syncStatus: "error",
                  lastError: description,
                }
              : candidate,
          ),
        );
        window.setTimeout(() => onRequestPersist?.(), 0);
        messageBus.publishToast({
          intent: "danger",
          title: t("library.config.subtitleStyles.syncFontSourceFailedTitle"),
          description,
        });
      } finally {
        setSyncingFontSources((current) => {
          const next = { ...current };
          delete next[sourceId];
          return next;
        });
      }
    },
    [onRequestPersist, subtitleStyleSources, t, updateSubtitleStyleSources],
  );

  const selectedDocument = selectedSubtitleStyleDocument;
  const selectedDocumentReadOnly = selectedDocument?.source === "builtin";
  const selectedDocumentCardID = selectedDocument
    ? buildEditableCardID("subtitle-style-document", selectedDocument.id)
    : "";
  const selectedDocumentEditing = selectedDocument
    ? !selectedDocumentReadOnly && isCardEditing(selectedDocumentCardID)
    : false;
  const selectedProfile = selectedSubtitleExportPreset;
  const selectedProfileReadOnly =
    isBuiltInSubtitleExportPreset(selectedProfile);
  const selectedProfileCardID = selectedProfile
    ? buildEditableCardID("subtitle-export-preset", selectedProfile.id)
    : "";
  const selectedProfileEditing = selectedProfile
    ? !selectedProfileReadOnly && isCardEditing(selectedProfileCardID)
    : false;
  const selectedVideoPreset = selectedVideoExportPreset;
  const selectedVideoPresetReadOnly =
    videoExportPresetDraftMode !== "create" &&
    (selectedVideoPreset?.isBuiltin ?? false);
  const selectedVideoPresetEditing =
    videoExportPresetDraftMode !== null &&
    (videoExportPresetDraftMode === "create" ||
      (videoExportPresetDraft?.id ?? "") === (selectedVideoPreset?.id ?? ""));
  const effectiveVideoPreset =
    videoExportPresetDraftMode === "create"
      ? videoExportPresetDraft
      : selectedVideoPresetEditing
        ? videoExportPresetDraft
        : selectedVideoPreset;
  const activeTaskRuntimeCardID = buildEditableCardID(
    "task-runtime",
    activeTaskRuntimeTask,
  );
  const activeTaskRuntimeEditing = isCardEditing(activeTaskRuntimeCardID);

  const toolbarState = React.useMemo<LibraryConfigToolbarState | null>(() => {
    switch (activePage) {
      case "task-runtime":
        return {
          actions: (
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="gap-2"
              onClick={() =>
                activeTaskRuntimeEditing
                  ? closeCardEditor(activeTaskRuntimeCardID)
                  : openCardEditor(activeTaskRuntimeCardID)
              }
            >
              {activeTaskRuntimeEditing ? (
                <Check className="h-3.5 w-3.5" />
              ) : (
                <PencilLine className="h-3.5 w-3.5" />
              )}
              {activeTaskRuntimeEditing ? t("common.done") : t("common.edit")}
            </Button>
          ),
        };
      case "languages":
        return {
          actions: (
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="gap-2"
              onClick={handleAddLanguage}
            >
              <Plus className="h-3.5 w-3.5" />
              {t("library.config.translateLanguages.add")}
            </Button>
          ),
        };
      case "glossary":
        return {
          actions: (
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="gap-2"
              onClick={handleAddGlossaryProfile}
            >
              <Plus className="h-3.5 w-3.5" />
              {t("library.config.languageAssets.addGlossary")}
            </Button>
          ),
        };
      case "prompts":
        return {
          actions: (
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="gap-2"
              onClick={handleAddPromptProfile}
            >
              <Plus className="h-3.5 w-3.5" />
              {t("library.config.languageAssets.addPrompt")}
            </Button>
          ),
        };
      case "subtitle-styles":
        return {
          actions: subtitleStyleToolbarActions ?? undefined,
        };
      case "font-management":
        return {
          actions: (
            <Button
              type="button"
              variant="outline"
              size="compact"
              className="gap-2"
              onClick={() => void handleRefreshFontList()}
              disabled={refreshingFontList}
            >
              {refreshingFontList ? (
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
              ) : (
                <Type className="h-3.5 w-3.5" />
              )}
              {refreshingFontList
                ? t("library.config.subtitleStyles.refreshingFontList")
                : t("library.config.subtitleStyles.refreshFontList")}
            </Button>
          ),
        };
      case "subtitle-export-presets":
        return {
          actions: (
            <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="compact"
                    className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                    disabled={selectedProfileEditing}
                  >
                    <Plus className="h-3.5 w-3.5" />
                    {t("library.config.subtitleStyles.addExportProfile")}
                    <ChevronDown className="h-3.5 w-3.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="start">
                  {SUBTITLE_EXPORT_PROFILE_FORMAT_OPTIONS.map((option) => (
                    <DropdownMenuItem
                      key={option.value}
                      onClick={() =>
                        handleAddSubtitleExportPreset(option.value)
                      }
                    >
                      <Captions className="h-3.5 w-3.5" />
                      {option.label}
                    </DropdownMenuItem>
                  ))}
                </DropdownMenuContent>
              </DropdownMenu>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button
                    type="button"
                    variant="ghost"
                    size="compact"
                    className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
                    disabled={!selectedProfile || selectedProfileEditing}
                  >
                    {t("library.config.subtitleStyles.setDefault")}
                    <ChevronDown className="h-3.5 w-3.5" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end" className="w-56">
                  <DropdownMenuItem
                    disabled={
                      !selectedProfile ||
                      subtitleStyleDefaults.subtitleExportPresetId ===
                        selectedProfile?.id
                    }
                    onClick={() => {
                      if (!selectedProfile) {
                        return;
                      }
                      handleUpdateSubtitleStyleDefault(
                        "subtitleExportPresetId",
                        selectedProfile.id,
                      );
                      onRequestPersist?.();
                    }}
                  >
                    {t("library.config.subtitleStyles.setDefaultExportProfile")}
                  </DropdownMenuItem>
                </DropdownMenuContent>
              </DropdownMenu>
            </div>
          ),
        };
      case "video-export-presets":
        return {
          actions: (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  size="compact"
                  className="gap-2"
                  disabled={selectedVideoPresetEditing}
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t("library.config.videoExportPresets.addVideo")}
                  <ChevronDown className="h-3.5 w-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem
                  onClick={() => handleCreateVideoExportPreset("video")}
                >
                  <Clapperboard className="h-3.5 w-3.5" />
                  {t("library.config.videoExportPresets.addVideoOption")}
                </DropdownMenuItem>
                <DropdownMenuItem
                  onClick={() => handleCreateVideoExportPreset("audio")}
                >
                  <AudioLines className="h-3.5 w-3.5" />
                  {t("library.config.videoExportPresets.addAudioOption")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ),
        };
      case "remote-sources":
        return {
          actions: (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button
                  type="button"
                  variant="outline"
                  size="compact"
                  className="gap-2"
                >
                  <Plus className="h-3.5 w-3.5" />
                  {t("library.config.subtitleStyles.addRemoteSource")}
                  <ChevronDown className="h-3.5 w-3.5" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuItem disabled>
                  <Captions className="h-3.5 w-3.5" />
                  {t("library.config.subtitleStyles.addStyleSource")}
                </DropdownMenuItem>
                <DropdownMenuItem disabled>
                  <Type className="h-3.5 w-3.5" />
                  {t("library.config.subtitleStyles.addFontSource")}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          ),
        };
      default:
        return null;
    }
  }, [
    activePage,
    activeTaskRuntimeCardID,
    activeTaskRuntimeEditing,
    closeCardEditor,
    handleAddSubtitleStyleFontSources,
    handleAddGlossaryProfile,
    handleAddLanguage,
    handleAddPromptProfile,
    handleAddSubtitleExportPreset,
    handleAddSubtitleStyleSource,
    handleCreateVideoExportPreset,
    handleRefreshFontList,
    handleUpdateSubtitleStyleDefault,
    openCardEditor,
    onRequestPersist,
    refreshingFontList,
    selectedProfile,
    selectedProfileEditing,
    selectedVideoPresetEditing,
    subtitleStyleDefaults.subtitleExportPresetId,
    subtitleStyleToolbarActions,
    t,
  ]);

  React.useEffect(() => {
    onToolbarStateChange?.(toolbarState);
    return () => {
      onToolbarStateChange?.(null);
    };
  }, [onToolbarStateChange, toolbarState]);

  React.useEffect(() => {
    if (activePage !== "font-management") {
      return;
    }
    if (!subtitleStyleFontSources.some((source) => source.enabled === true)) {
      return;
    }

    const missingFamilies = subtitleReferencedFonts
      .filter((entry) => entry.status === "missing")
      .map((entry) => entry.family.trim())
      .filter(Boolean);

    for (const family of missingFamilies) {
      const state = remoteFontSearchState[family];
      if (state?.loading || state) {
        continue;
      }
      void handleSearchRemoteFontCandidates(family);
    }
  }, [
    activePage,
    handleSearchRemoteFontCandidates,
    remoteFontSearchState,
    subtitleReferencedFonts,
    subtitleStyleFontSources,
  ]);

  const describeLanguageNavigationItem = React.useCallback(
    (item: LanguageConfigItem) => {
      const code = item.language.code?.trim().toUpperCase() ?? "";
      const summary = summarizeLanguageConfigItem(item.language, item.kind);
      return [code, summary].filter(Boolean).join(" · ");
    },
    [],
  );
  const describeGlossaryNavigationItem = React.useCallback(
    (profile: LibraryGlossaryProfileDTO) =>
      summarizeGlossaryProfile(profile, languageOptions) ||
      resolveGlossaryCategoryLabel(profile.category),
    [language, languageOptions],
  );
  const describePromptNavigationItem = React.useCallback(
    (profile: LibraryPromptProfileDTO) =>
      summarizePromptProfile(profile) ||
      resolvePromptCategoryLabel(profile.category),
    [language],
  );

  const renderOverviewPage = () => {
    const monoStyleCount = subtitleMonoStyles.length;
    const bilingualStyleCount = subtitleBilingualStyles.length;
    const enabledFontSourceCount = subtitleStyleFontSources.filter(
      (source) => source.enabled === true,
    ).length;
    const defaultMonoStyle = resolveDefaultMonoStyle(value);
    const defaultBilingualStyle = resolveDefaultBilingualStyle(value);
    const defaultMonoLabel =
      defaultMonoStyle?.name?.trim() ||
      t("library.config.taskDefaults.none");
    const defaultBilingualLabel =
      defaultBilingualStyle?.name?.trim() ||
      t("library.config.taskDefaults.none");
    const defaultExportProfile =
      subtitleExportPresets.find(
        (profile) =>
          profile.id === subtitleStyleDefaults.subtitleExportPresetId,
      ) ?? null;
    const defaultExportProfileLabel =
      defaultExportProfile?.name?.trim() ||
      t("library.config.taskDefaults.none");

    return (
      <div className="min-w-0 space-y-4 overflow-y-auto pr-1">
        <ConfigSectionCard
          title={t("library.config.workspace.title")}
          description={t("library.config.pages.overviewWorkspaceDescription")}
          icon={Settings2}
          contentClassName="space-y-4"
        >
          <div
            className={cn(
              "grid gap-3 px-3 py-2.5 md:grid-cols-[minmax(0,1fr)_auto] md:items-center",
              DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
            )}
          >
            <div className="min-w-0">
              <div className="text-xs font-medium text-foreground">
                {t("library.config.workspace.fastReadLatestState")}
              </div>
              <div className="mt-1 text-xs leading-5 text-muted-foreground">
                {t("library.config.workspace.fastReadLatestStateDescription")}
              </div>
            </div>
            <Badge
              variant="outline"
              className="h-6 shrink-0 px-2 text-[10px] tracking-[0.08em]"
            >
              {value.workspace.fastReadLatestState
                ? t("common.enabled")
                : t("common.disabled")}
            </Badge>
          </div>
        </ConfigSectionCard>

        <ConfigSectionCard
          title={t("library.config.subtitleStyles.overviewTitle")}
          description={t("library.config.subtitleStyles.overviewDescription")}
          icon={Captions}
          contentClassName="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
        >
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.monoStyles")}
            value={String(monoStyleCount)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.bilingualStyles")}
            value={String(bilingualStyleCount)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.totalStyles")}
            value={String(subtitlePresetCount)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.requiredFonts")}
            value={String(subtitleFontCoverage.required.length)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.fontSources")}
            value={String(enabledFontSourceCount)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.missingFonts")}
            value={String(subtitleFontCoverage.missing.length)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.managedFonts")}
            value={String(subtitleFontCoverage.managed.length)}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.subtitleExportPresets")}
            value={String(subtitleExportPresets.length || 0)}
          />
        </ConfigSectionCard>

        <ConfigSectionCard
          title={t("library.config.subtitleStyles.defaultsTitle")}
          description={t("library.config.subtitleStyles.defaultsDescription")}
          icon={MonitorPlay}
          contentClassName="grid gap-3 md:grid-cols-2"
        >
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.defaultMono")}
            value={defaultMonoLabel}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.defaultBilingual")}
            value={defaultBilingualLabel}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.defaultExportProfile")}
            value={defaultExportProfileLabel}
          />
        </ConfigSectionCard>

        <ConfigSectionCard
          title={t("library.config.subtitleStyles.deliveryReadinessTitle")}
          description={t("library.config.subtitleStyles.deliveryReadinessDescription")}
          icon={Database}
          contentClassName="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
        >
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.systemFontCatalog")}
            value={
              isSystemFontsLoading
                ? t("common.loading")
                : String(systemFontFamilies?.length ?? 0)
            }
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.resolvedFonts")}
            value={`${subtitleFontCoverage.resolved.length}/${subtitleFontCoverage.required.length || 0}`}
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.missingFonts")}
            value={
              isSystemFontsLoading
                ? t("common.loading")
                : formatFontList(
                    subtitleFontCoverage.missing,
                    t("library.config.subtitleStyles.fontCoverageComplete"),
                  )
            }
          />
          <ReadOnlyInfoField
            label={t("library.config.subtitleStyles.remoteStatus")}
            value={`${enabledFontSourceCount} ${t("library.config.subtitleStyles.fontSources")}`}
          />
        </ConfigSectionCard>
      </div>
    );
  };

  const renderLanguagesTab = () => {
    const selectedLanguageEditing = selectedLanguageCardID
      ? isCardEditing(selectedLanguageCardID)
      : false;
    const selectedLanguageIsCustom = selectedLanguageItem?.kind === "custom";

    return (
      <ConfigMasterDetailLayout
        sidebar={
          <ConfigNavigationSidebar
            showSearch={languageConfigItems.length > 0}
            searchValue={languageSearch}
            onSearchChange={setLanguageSearch}
            searchPlaceholder={t("library.config.translateLanguages.searchPlaceholder")}
            count={visibleLanguageConfigItems.length}
            emptyState={
              <EmptyConfigState
                title={
                  languageConfigItems.length === 0
                    ? t("library.config.translateLanguages.empty")
                    : t("library.config.translateLanguages.searchEmpty")
                }
              />
            }
          >
            {groupedVisibleLanguageConfigItems.map((group) => (
              <ConfigNavigationGroup
                key={group.id}
                title={group.title}
                count={group.items.length}
              >
                {group.items.map((item) => {
                  const isSelected = item.id === selectedLanguageItem?.id;
                  return (
                    <ConfigNavigationItem
                      key={item.id}
                      title={
                        item.language.label ||
                        item.language.code ||
                        t("library.config.translateLanguages.untitled")
                      }
                      description={
                        describeLanguageNavigationItem(item) || undefined
                      }
                      selected={isSelected}
                      disabled={selectedLanguageEditing && !isSelected}
                      compact
                      onClick={() => {
                        if (selectedLanguageEditing && !isSelected) {
                          return;
                        }
                        setSelectedAssetItemIds((current) => ({
                          ...current,
                          languages: item.id,
                        }));
                      }}
                    />
                  );
                })}
              </ConfigNavigationGroup>
            ))}
          </ConfigNavigationSidebar>
        }
        detail={
          <div className="h-full min-w-0 min-h-0">
            {selectedLanguage ? (
              <ConfigDetailPanel
                title={
                  selectedLanguageIsCustom && selectedLanguageEditing ? (
                    <Input
                      value={selectedLanguage.label ?? ""}
                      onChange={(event) => {
                        if (selectedCustomLanguageIndex >= 0) {
                          handleChangeLanguage(selectedCustomLanguageIndex, {
                            label: event.target.value,
                          });
                        }
                      }}
                      placeholder={t("library.config.translateLanguages.untitled")}
                      className="h-8 max-w-full border-border/70 bg-background/80 text-sm font-semibold"
                    />
                  ) : (
                    selectedLanguage.label ||
                    selectedLanguage.code ||
                    t("library.config.translateLanguages.untitled")
                  )
                }
                actions={
                  selectedLanguageIsCustom ? (
                    <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                      <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
                        <Button
                          type="button"
                          variant="ghost"
                          size="compact"
                          className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                          onClick={() =>
                            selectedLanguageEditing
                              ? closeCardEditor(selectedLanguageCardID)
                              : openCardEditor(selectedLanguageCardID)
                          }
                        >
                          {selectedLanguageEditing ? (
                            <Check className="h-3.5 w-3.5" />
                          ) : (
                            <PencilLine className="h-3.5 w-3.5" />
                          )}
                          {selectedLanguageEditing
                            ? t("common.done")
                            : t("common.edit")}
                        </Button>
                      </div>
                    </div>
                  ) : null
                }
                badges={
                  <>
                    {selectedLanguage.code ? (
                      <Badge
                        variant="outline"
                        className="text-[10px] tracking-[0.08em]"
                      >
                        {selectedLanguage.code.toUpperCase()}
                      </Badge>
                    ) : null}
                    <Badge
                      variant="outline"
                      className="text-[10px] tracking-[0.08em]"
                    >
                      {selectedLanguageItem?.kind === "builtin"
                        ? t("library.config.translateLanguages.builtinBadge")
                        : t("library.config.translateLanguages.customBadge")}
                    </Badge>
                  </>
                }
                footer={
                  selectedLanguageIsCustom &&
                  selectedCustomLanguageIndex >= 0 ? (
                    <div className="flex justify-center pt-2">
                      <Button
                        type="button"
                        variant="destructive"
                        size="compact"
                        className="min-w-[160px]"
                        onClick={() =>
                          handleDeleteLanguage(selectedCustomLanguageIndex)
                        }
                      >
                        {t("common.delete")}
                      </Button>
                    </div>
                  ) : undefined
                }
              >
                <ConfigInputField
                  label={t("library.config.translateLanguages.code")}
                  description={t("library.config.translateLanguages.description")}
                  inline
                  inlineDescriptionBelow
                  value={selectedLanguage.code ?? ""}
                  disabled={!selectedLanguageEditing || !selectedLanguageIsCustom}
                  placeholder={t("library.config.translateLanguages.code")}
                  onChange={(nextValue) => {
                    if (selectedCustomLanguageIndex >= 0) {
                      handleChangeLanguage(selectedCustomLanguageIndex, {
                        code: nextValue,
                      });
                    }
                  }}
                />
                <ConfigInputField
                  label={t("library.config.translateLanguages.aliases")}
                  description={t("library.config.translateLanguages.aliasesDescription")}
                  inline
                  inlineDescriptionBelow
                  value={(selectedLanguage.aliases ?? []).join(", ")}
                  disabled={!selectedLanguageEditing || !selectedLanguageIsCustom}
                  placeholder={t("library.config.translateLanguages.aliases")}
                  onChange={(nextValue) => {
                    if (selectedCustomLanguageIndex >= 0) {
                      handleChangeLanguage(selectedCustomLanguageIndex, {
                        aliases: splitAliases(nextValue),
                      });
                    }
                  }}
                />
              </ConfigDetailPanel>
            ) : (
              <div
                className={cn(
                  "flex h-full min-h-0 flex-col overflow-hidden p-4",
                  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                )}
              >
                <EmptyConfigState
                  title={t("library.config.translateLanguages.empty")}
                />
              </div>
            )}
          </div>
        }
      />
    );
  };

  const renderLanguageAssetsSection = (page: LanguageAssetTabId) => {
    const selectedGlossaryCardID = selectedGlossaryProfile
      ? buildEditableCardID("glossary", selectedGlossaryProfile.id)
      : "";
    const selectedPromptCardID = selectedPromptProfile
      ? buildEditableCardID("prompt", selectedPromptProfile.id)
      : "";
    const selectedGlossaryEditing = selectedGlossaryCardID
      ? isCardEditing(selectedGlossaryCardID)
      : false;
    const selectedPromptEditing = selectedPromptCardID
      ? isCardEditing(selectedPromptCardID)
      : false;

    return (
      <div className="flex h-full min-h-0 w-full min-w-0 flex-1 flex-col gap-4">
        <div className="flex min-h-0 min-w-0 w-full flex-1 flex-col">
          <ConditionalPanel
            active={page === "languages"}
            className="h-full min-h-0 w-full min-w-0 flex-1"
          >
            {renderLanguagesTab()}
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "glossary"}
            className="h-full min-h-0 w-full min-w-0 flex-1"
          >
            <ConfigMasterDetailLayout
              sidebar={
                <ConfigNavigationSidebar
                  showSearch={glossaryProfiles.length > 0}
                  searchValue={glossarySearch}
                  onSearchChange={setGlossarySearch}
                  searchPlaceholder={t("library.config.languageAssets.glossarySearchPlaceholder")}
                  count={visibleGlossaryProfiles.length}
                  emptyState={
                    glossaryProfiles.length === 0 ? (
                      <ConfigStandardEmptyState
                        icon={BookOpen}
                        title={t("library.config.languageAssets.glossaryEmpty")}
                        description={t("library.config.languageAssets.glossaryEmptyDescription")}
                      />
                    ) : (
                      <EmptyConfigState
                        title={t("library.config.languageAssets.glossarySearchEmpty")}
                      />
                    )
                  }
                >
                  {groupedVisibleGlossaryProfiles.map((group) => (
                    <ConfigNavigationGroup
                      key={group.id}
                      title={group.title}
                      count={group.profiles.length}
                    >
                      {group.profiles.map((profile) => {
                        const isSelected =
                          profile.id === selectedGlossaryProfile?.id;
                        return (
                          <ConfigNavigationItem
                            key={profile.id}
                            title={
                              profile.name ||
                              t("library.config.languageAssets.untitledGlossary")
                            }
                            description={
                              describeGlossaryNavigationItem(profile) ||
                              undefined
                            }
                            selected={isSelected}
                            disabled={selectedGlossaryEditing && !isSelected}
                            compact
                            onClick={() => {
                              if (selectedGlossaryEditing && !isSelected) {
                                return;
                              }
                              setSelectedAssetItemIds((current) => ({
                                ...current,
                                glossary: profile.id,
                              }));
                            }}
                          />
                        );
                      })}
                    </ConfigNavigationGroup>
                  ))}
                </ConfigNavigationSidebar>
              }
              detail={
                <div className="h-full min-w-0 min-h-0">
                {selectedGlossaryProfile ? (
                  <ConfigDetailPanel
                    title={
                      selectedGlossaryEditing ? (
                        <Input
                          value={selectedGlossaryProfile.name ?? ""}
                          onChange={(event) =>
                            handleUpdateGlossaryProfile(
                              selectedGlossaryProfile.id,
                              { name: event.target.value },
                            )
                          }
                          placeholder={t("library.config.languageAssets.untitledGlossary")}
                          className="h-8 max-w-full border-border/70 bg-background/80 text-sm font-semibold"
                        />
                      ) : (
                        selectedGlossaryProfile.name ||
                        t("library.config.languageAssets.untitledGlossary")
                      )
                    }
                    actions={
                      <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                        <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
                          <Button
                            type="button"
                            variant="ghost"
                            size="compact"
                            className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                            onClick={() =>
                              selectedGlossaryEditing
                                ? closeCardEditor(selectedGlossaryCardID)
                                : openCardEditor(selectedGlossaryCardID)
                            }
                          >
                            {selectedGlossaryEditing ? (
                              <Check className="h-3.5 w-3.5" />
                            ) : (
                              <PencilLine className="h-3.5 w-3.5" />
                            )}
                            {selectedGlossaryEditing
                              ? t("common.done")
                              : t("common.edit")}
                          </Button>
                        </div>
                      </div>
                    }
                    badges={
                      <Badge
                        variant="outline"
                        className="text-[10px] tracking-[0.08em]"
                      >
                        {resolveGlossaryCategoryLabel(
                          selectedGlossaryProfile.category,
                        )}
                      </Badge>
                    }
                    footer={
                      <div className="flex justify-center pt-2">
                        <Button
                          type="button"
                          variant="destructive"
                          size="compact"
                          className="min-w-[160px]"
                          onClick={() =>
                            handleDeleteGlossaryProfile(
                              selectedGlossaryProfile.id,
                            )
                          }
                        >
                          {t("common.delete")}
                        </Button>
                      </div>
                    }
                  >
                    <div
                      className={cn(
                        "grid gap-3 px-3 py-2.5 md:grid-cols-[120px_minmax(0,1fr)_minmax(0,1fr)]",
                        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                      )}
                    >
                      <div className="space-y-2">
                        <div className="text-xs font-medium text-foreground">
                          {t("library.config.languageAssets.promptCategory")}
                        </div>
                        <Select
                          value={normalizeGlossaryCategory(
                            selectedGlossaryProfile.category,
                          )}
                          onChange={(event) =>
                            handleUpdateGlossaryProfile(
                              selectedGlossaryProfile.id,
                              { category: event.target.value },
                            )
                          }
                          disabled={!selectedGlossaryEditing}
                          className="h-8 w-full min-w-0 border-border/70 bg-background/80"
                        >
                          <option value="all">
                            {t("library.config.languageAssets.promptAll")}
                          </option>
                          <option value="translate">
                            {t("library.config.languageAssets.promptTranslate")}
                          </option>
                          <option value="proofread">
                            {t("library.config.languageAssets.promptProofread")}
                          </option>
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <div className="text-xs font-medium text-foreground">
                          {t("library.config.languageAssets.sourceLanguage")}
                        </div>
                        <Select
                          value={normalizeScopedLanguageValue(
                            selectedGlossaryProfile.sourceLanguage,
                          )}
                          onChange={(event) =>
                            handleUpdateGlossaryProfile(
                              selectedGlossaryProfile.id,
                              { sourceLanguage: event.target.value },
                            )
                          }
                          disabled={!selectedGlossaryEditing}
                          className="h-8 w-full min-w-0 border-border/70 bg-background/80"
                        >
                          {glossaryLanguageScopeOptions.map((option) => (
                            <option
                              key={`glossary-source-${option.value}`}
                              value={option.value}
                            >
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                      <div className="space-y-2">
                        <div className="text-xs font-medium text-foreground">
                          {t("library.config.languageAssets.targetLanguage")}
                        </div>
                        <Select
                          value={normalizeScopedLanguageValue(
                            selectedGlossaryProfile.targetLanguage,
                          )}
                          onChange={(event) =>
                            handleUpdateGlossaryProfile(
                              selectedGlossaryProfile.id,
                              { targetLanguage: event.target.value },
                            )
                          }
                          disabled={!selectedGlossaryEditing}
                          className="h-8 w-full min-w-0 border-border/70 bg-background/80"
                        >
                          {glossaryLanguageScopeOptions.map((option) => (
                            <option
                              key={`glossary-target-${option.value}`}
                              value={option.value}
                            >
                              {option.label}
                            </option>
                          ))}
                        </Select>
                      </div>
                    </div>
                    <ConfigTextareaField
                      label={t("library.config.languageAssets.glossaryDescriptionLabel")}
                      value={selectedGlossaryProfile.description ?? ""}
                      disabled={!selectedGlossaryEditing}
                      placeholder={t("library.config.languageAssets.glossaryDescriptionLabel")}
                      rows={2}
                      onChange={(nextValue) =>
                        handleUpdateGlossaryProfile(
                          selectedGlossaryProfile.id,
                          { description: nextValue },
                        )
                      }
                    />
                    <div className="space-y-2">
                      <div className="flex items-center justify-between gap-3">
                        <div className="text-xs font-medium text-foreground">
                          {t("library.config.languageAssets.terms")}
                        </div>
                        <Button
                          type="button"
                          variant="outline"
                          size="compact"
                          className="gap-2"
                          disabled={!selectedGlossaryEditing}
                          onClick={() =>
                            handleAddGlossaryTerm(selectedGlossaryProfile.id)
                          }
                        >
                          <Plus className="h-3.5 w-3.5" />
                          {t("library.config.languageAssets.addTerm")}
                        </Button>
                      </div>
                      <div className="space-y-2">
                        {(selectedGlossaryProfile.terms ?? []).map(
                          (term, termIndex) => (
                            <div
                              key={`${selectedGlossaryProfile.id}-term-${termIndex}`}
                              className={cn(
                                "grid gap-2 p-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)_minmax(0,1.2fr)_auto]",
                                DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                              )}
                            >
                              <Input
                                value={term.source ?? ""}
                                disabled={!selectedGlossaryEditing}
                                onChange={(event) =>
                                  handleUpdateGlossaryTerm(
                                    selectedGlossaryProfile.id,
                                    termIndex,
                                    { source: event.target.value },
                                  )
                                }
                                placeholder={t("library.config.languageAssets.termSource")}
                                className="h-8"
                              />
                              <Input
                                value={term.target ?? ""}
                                disabled={!selectedGlossaryEditing}
                                onChange={(event) =>
                                  handleUpdateGlossaryTerm(
                                    selectedGlossaryProfile.id,
                                    termIndex,
                                    { target: event.target.value },
                                  )
                                }
                                placeholder={t("library.config.languageAssets.termTarget")}
                                className="h-8"
                              />
                              <Input
                                value={term.note ?? ""}
                                disabled={!selectedGlossaryEditing}
                                onChange={(event) =>
                                  handleUpdateGlossaryTerm(
                                    selectedGlossaryProfile.id,
                                    termIndex,
                                    { note: event.target.value },
                                  )
                                }
                                placeholder={t("library.config.languageAssets.termNote")}
                                className="h-8"
                              />
                              <Button
                                type="button"
                                variant="ghost"
                                size="compactIcon"
                                className="h-8 w-8"
                                disabled={!selectedGlossaryEditing}
                                onClick={() =>
                                  handleDeleteGlossaryTerm(
                                    selectedGlossaryProfile.id,
                                    termIndex,
                                  )
                                }
                                aria-label={t("library.config.languageAssets.removeTerm")}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </div>
                          ),
                        )}
                      </div>
                    </div>
                  </ConfigDetailPanel>
                ) : (
                  <div
                    className={cn(
                      "flex h-full min-h-0 flex-col overflow-hidden p-4",
                      DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                    )}
                  >
                    <ConfigStandardEmptyState
                      icon={BookOpen}
                      title={t("library.config.languageAssets.glossaryEmpty")}
                      description={t("library.config.languageAssets.glossaryEmptyDescription")}
                    />
                  </div>
                )}
                </div>
              }
            />
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "prompts"}
            className="h-full min-h-0 w-full min-w-0 flex-1"
          >
            <ConfigMasterDetailLayout
              sidebar={
                <ConfigNavigationSidebar
                  showSearch={promptProfiles.length > 0}
                  searchValue={promptSearch}
                  onSearchChange={setPromptSearch}
                  searchPlaceholder={t("library.config.languageAssets.promptSearchPlaceholder")}
                  count={visiblePromptProfiles.length}
                  emptyState={
                    promptProfiles.length === 0 ? (
                      <ConfigStandardEmptyState
                        icon={Sparkles}
                        title={t("library.config.languageAssets.promptEmpty")}
                        description={t("library.config.languageAssets.promptEmptyDescription")}
                      />
                    ) : (
                      <EmptyConfigState
                        title={t("library.config.languageAssets.promptSearchEmpty")}
                      />
                    )
                  }
                >
                  {groupedVisiblePromptProfiles.map((group) => (
                    <ConfigNavigationGroup
                      key={group.id}
                      title={group.title}
                      count={group.profiles.length}
                    >
                      {group.profiles.map((profile) => {
                        const isSelected =
                          profile.id === selectedPromptProfile?.id;
                        return (
                          <ConfigNavigationItem
                            key={profile.id}
                            title={
                              profile.name ||
                              t("library.config.languageAssets.untitledPrompt")
                            }
                            description={
                              describePromptNavigationItem(profile) ||
                              undefined
                            }
                            selected={isSelected}
                            disabled={selectedPromptEditing && !isSelected}
                            compact
                            onClick={() => {
                              if (selectedPromptEditing && !isSelected) {
                                return;
                              }
                              setSelectedAssetItemIds((current) => ({
                                ...current,
                                prompts: profile.id,
                              }));
                            }}
                          />
                        );
                      })}
                    </ConfigNavigationGroup>
                  ))}
                </ConfigNavigationSidebar>
              }
              detail={
                <div className="h-full min-w-0 min-h-0">
                {selectedPromptProfile ? (
                  <ConfigDetailPanel
                    title={
                      selectedPromptEditing ? (
                        <Input
                          value={selectedPromptProfile.name ?? ""}
                          onChange={(event) =>
                            handleUpdatePromptProfile(
                              selectedPromptProfile.id,
                              { name: event.target.value },
                            )
                          }
                          placeholder={t("library.config.languageAssets.untitledPrompt")}
                          className="h-8 max-w-full border-border/70 bg-background/80 text-sm font-semibold"
                        />
                      ) : (
                        selectedPromptProfile.name ||
                        t("library.config.languageAssets.untitledPrompt")
                      )
                    }
                    actions={
                      <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                        <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
                          <Button
                            type="button"
                            variant="ghost"
                            size="compact"
                            className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                            onClick={() =>
                              selectedPromptEditing
                                ? closeCardEditor(selectedPromptCardID)
                                : openCardEditor(selectedPromptCardID)
                            }
                          >
                            {selectedPromptEditing ? (
                              <Check className="h-3.5 w-3.5" />
                            ) : (
                              <PencilLine className="h-3.5 w-3.5" />
                            )}
                            {selectedPromptEditing
                              ? t("common.done")
                              : t("common.edit")}
                          </Button>
                        </div>
                      </div>
                    }
                    badges={
                      <Badge
                        variant="outline"
                        className="text-[10px] tracking-[0.08em]"
                      >
                        {resolvePromptCategoryLabel(
                          selectedPromptProfile.category,
                        )}
                      </Badge>
                    }
                    footer={
                      <div className="flex justify-center pt-2">
                        <Button
                          type="button"
                          variant="destructive"
                          size="compact"
                          className="min-w-[160px]"
                          onClick={() =>
                            handleDeletePromptProfile(selectedPromptProfile.id)
                          }
                        >
                          {t("common.delete")}
                        </Button>
                      </div>
                    }
                  >
                    <div
                      className={cn(
                        "flex items-center justify-between gap-3 px-3 py-2.5",
                        DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                      )}
                    >
                      <div className="text-xs font-medium text-foreground">
                        {t("library.config.languageAssets.promptCategory")}
                      </div>
                      <Select
                        value={selectedPromptProfile.category ?? "all"}
                        onChange={(event) =>
                          handleUpdatePromptProfile(selectedPromptProfile.id, {
                            category: event.target.value,
                          })
                        }
                        disabled={!selectedPromptEditing}
                        className="h-8 w-full max-w-[220px] min-w-0 border-border/70 bg-background/80"
                      >
                        <option value="all">
                          {t("library.config.languageAssets.promptAll")}
                        </option>
                        <option value="translate">
                          {t("library.config.languageAssets.promptTranslate")}
                        </option>
                        <option value="proofread">
                          {t("library.config.languageAssets.promptProofread")}
                        </option>
                        <option value="glossary">
                          {t("library.config.languageAssets.promptGlossary")}
                        </option>
                      </Select>
                    </div>
                    <ConfigTextareaField
                      label={t("library.config.languageAssets.promptDescriptionLabel")}
                      value={selectedPromptProfile.description ?? ""}
                      disabled={!selectedPromptEditing}
                      placeholder={t("library.config.languageAssets.promptDescriptionLabel")}
                      rows={2}
                      onChange={(nextValue) =>
                        handleUpdatePromptProfile(selectedPromptProfile.id, {
                          description: nextValue,
                        })
                      }
                    />
                    <ConfigTextareaField
                      label={t("library.config.languageAssets.promptBodyLabel")}
                      value={selectedPromptProfile.prompt ?? ""}
                      disabled={!selectedPromptEditing}
                      placeholder={t("library.config.languageAssets.promptBody")}
                      rows={8}
                      onChange={(nextValue) =>
                        handleUpdatePromptProfile(selectedPromptProfile.id, {
                          prompt: nextValue,
                        })
                      }
                    />
                  </ConfigDetailPanel>
                ) : (
                  <div
                    className={cn(
                      "flex h-full min-h-0 flex-col overflow-hidden p-4",
                      DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                    )}
                  >
                    <ConfigStandardEmptyState
                      icon={Sparkles}
                      title={t("library.config.languageAssets.promptEmpty")}
                      description={t("library.config.languageAssets.promptEmptyDescription")}
                    />
                  </div>
                )}
                </div>
              }
            />
          </ConditionalPanel>
        </div>
      </div>
    );
  };

  const renderSubtitleStylesSection = (
    page: "overview" | "documents" | "profiles" | "fonts" | "sources",
  ) => {
    const enabledStyleSourceCount = subtitleStyleDocumentSources.filter(
      (source) => source.enabled === true,
    ).length;
    const enabledFontSourceCount = subtitleStyleFontSources.filter(
      (source) => source.enabled === true,
    ).length;
    const selectedDocumentDraft =
      selectedDocument && subtitleStyleDocumentDraft?.id === selectedDocument.id
        ? subtitleStyleDocumentDraft
        : null;
    const effectiveSelectedDocument = selectedDocumentDraft ?? selectedDocument;
    const selectedDocumentSummary = effectiveSelectedDocument
      ? resolveAssDocumentSummary(effectiveSelectedDocument)
      : null;
    const selectedDocumentFontCoverage =
      resolveSubtitleStyleDocumentFontCoverage(
        selectedDocumentSummary?.fonts ?? [],
        normalizedSystemFonts,
        value.subtitleStyles.fonts ?? [],
      );
    const selectedDocumentAvailableFonts =
      selectedDocumentFontCoverage.resolved;
    const selectedDocumentMissingFonts = selectedDocumentFontCoverage.missing;
    const selectedDocumentSourceLabel = effectiveSelectedDocument
      ? effectiveSelectedDocument.source === "builtin"
        ? t("library.config.subtitleStyles.sourceBuiltin")
        : effectiveSelectedDocument.source === "remote"
          ? t("library.config.subtitleStyles.sourceRemote")
          : t("library.config.subtitleStyles.sourceLibrary")
      : "";
    const selectedDocumentResolutionLabel =
      selectedDocumentSummary?.playResX && selectedDocumentSummary.playResY
        ? `${selectedDocumentSummary.playResX}x${selectedDocumentSummary.playResY}`
        : "-";
    const selectedDocumentMatchedFontsLabel =
      selectedDocumentSummary && selectedDocumentSummary.fonts.length > 0
        ? `${t("library.config.subtitleStyles.resolvedFonts")} ${selectedDocumentAvailableFonts.length}/${selectedDocumentSummary.fonts.length}`
        : "";
    const selectedDocumentMissingFontsLabel =
      selectedDocumentMissingFonts.length > 0
        ? `${t("library.config.subtitleStyles.missingFonts")} ${selectedDocumentMissingFonts.join(", ")}`
        : "";
    const defaultExportProfile =
      subtitleExportPresets.find(
        (profile) =>
          profile.id === subtitleStyleDefaults.subtitleExportPresetId,
      ) ?? null;
    const defaultExportProfileLabel =
      defaultExportProfile?.name?.trim() ||
      t("library.config.taskDefaults.none");
    const selectedProfileTargetFormat = selectedProfile
      ? resolveSubtitleExportPresetFormat(selectedProfile)
      : "srt";
    const selectedProfileMediaStrategy = normalizeSubtitleExportMediaStrategy(
      selectedProfile?.mediaStrategy ?? "",
    );
    const selectedProfileConfig: NonNullable<
      LibrarySubtitleExportPresetDTO["config"]
    > = selectedProfile?.config ?? {};
    const hasEnabledFontSource = enabledFontSourceCount > 0;

    return (
      <div
        className={cn(
          "min-w-0",
          page === "documents" || page === "profiles"
            ? "flex h-full min-h-0 w-full flex-1 flex-col"
            : "space-y-4",
        )}
      >
        <div
          className={cn(
            "min-w-0",
            page === "documents" || page === "profiles"
              ? "flex min-h-0 w-full flex-1 flex-col"
              : "",
          )}
        >
          <ConditionalPanel
            active={page === "overview"}
            className="space-y-4"
          >
            <ConfigSectionCard
              title={t("library.config.subtitleStyles.overviewTitle")}
              description={t("library.config.subtitleStyles.overviewDescription")}
              icon={Captions}
              contentClassName="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.monoStyles")}
                value={String(subtitleMonoStyles.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.bilingualStyles")}
                value={String(subtitleBilingualStyles.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.totalStyles")}
                value={String(subtitlePresetCount)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.subtitleExportPresets")}
                value={String(subtitleExportPresets.length || 0)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.requiredFonts")}
                value={String(subtitleFontCoverage.required.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.resolvedFonts")}
                value={String(subtitleFontCoverage.resolved.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.missingFonts")}
                value={String(subtitleFontCoverage.missing.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.fontSources")}
                value={String(enabledFontSourceCount)}
              />
            </ConfigSectionCard>

            <ConfigSectionCard
              title={t("library.config.subtitleStyles.defaultsTitle")}
              description={t("library.config.subtitleStyles.defaultsDescription")}
              icon={MonitorPlay}
              contentClassName="grid gap-3 md:grid-cols-1"
            >
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.defaultExportProfile")}
                value={defaultExportProfileLabel}
              />
            </ConfigSectionCard>

            <ConfigSectionCard
              title={t("library.config.subtitleStyles.deliveryReadinessTitle")}
              description={t("library.config.subtitleStyles.deliveryReadinessDescription")}
              icon={Database}
              contentClassName="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.systemFontCatalog")}
                value={
                  isSystemFontsLoading
                    ? t("common.loading")
                    : String(systemFontFamilies?.length ?? 0)
                }
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.resolvedFonts")}
                value={`${subtitleFontCoverage.resolved.length}/${subtitleFontCoverage.required.length || 0}`}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.missingFonts")}
                value={
                  isSystemFontsLoading
                    ? t("common.loading")
                    : formatFontList(
                        subtitleFontCoverage.missing,
                        t("library.config.subtitleStyles.fontCoverageComplete"),
                      )
                }
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.remoteStatus")}
                value={`${enabledStyleSourceCount} ${t("library.config.subtitleStyles.styleSources")} · ${enabledFontSourceCount} ${t("library.config.subtitleStyles.fontSources")}`}
              />
            </ConfigSectionCard>
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "documents"}
            className="min-h-0 w-full flex-1"
          >
            <div className="grid h-full min-h-0 w-full gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
              <div
                className={cn(
                  "flex h-full min-h-0 flex-col overflow-hidden p-4",
                  DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                )}
              >
                {subtitleStyleDocuments.length === 0 ? (
                  <EmptyConfigState
                    title={t("library.config.subtitleStyles.emptyDocuments")}
                  />
                ) : (
                  <div className="min-h-0 flex-1 space-y-2 overflow-y-auto pr-1">
                    {subtitleStyleDocuments.map((document) => {
                      const summary = resolveAssDocumentSummary(document);
                      const missingFonts =
                        resolveSubtitleStyleDocumentFontCoverage(
                          summary.fonts,
                          normalizedSystemFonts,
                          [],
                        ).missing;
                      const isSelected = document.id === selectedDocument?.id;

                      return (
                        <button
                          key={document.id}
                          type="button"
                          onClick={() => {
                            if (
                              selectedDocumentEditing &&
                              document.id !== selectedDocument?.id
                            ) {
                              return;
                            }
                            setSelectedSubtitleStyleDocumentId(document.id);
                          }}
                          className={cn(
                            "w-full rounded-xl border px-3 py-3 text-left transition-colors",
                            isSelected
                              ? "border-border/70 bg-background/90 shadow-sm"
                              : "border-transparent bg-background/45 hover:border-border/60 hover:bg-background/70",
                            selectedDocumentEditing &&
                              document.id !== selectedDocument?.id
                              ? "cursor-not-allowed opacity-70"
                              : "",
                          )}
                        >
                          <div className="flex items-start justify-between gap-3">
                            <div className="min-w-0 flex-1">
                              <div className="truncate text-sm font-medium text-foreground">
                                {document.name ||
                                  t("library.config.subtitleStyles.untitledDocument")}
                              </div>
                            </div>
                            <Badge
                              variant="outline"
                              className={cn(
                                "text-[10px] tracking-[0.08em]",
                                document.enabled === false
                                  ? "border-muted-foreground/30 text-muted-foreground"
                                  : "border-emerald-300/50 text-emerald-700",
                              )}
                            >
                              {document.enabled === false
                                ? t("library.config.subtitleStyles.disabled")
                                : t("library.config.subtitleStyles.enabled")}
                            </Badge>
                          </div>

                          <div className="mt-2 flex flex-wrap gap-1.5">
                            <Badge
                              variant="outline"
                              className="text-[10px] tracking-[0.08em]"
                            >
                              {t("library.config.subtitleStyles.stylesCount")}{" "}
                              {summary.styleCount}
                            </Badge>
                            {missingFonts.length > 0 ? (
                              <Badge
                                variant="outline"
                                className="border-amber-300/60 text-[10px] tracking-[0.08em] text-amber-800"
                              >
                                {t("library.config.subtitleStyles.missingFonts")}{" "}
                                {missingFonts.length}
                              </Badge>
                            ) : null}
                          </div>
                        </button>
                      );
                    })}
                  </div>
                )}
              </div>

              <div className="h-full min-w-0 min-h-0">
                {selectedDocument ? (
                  <div
                    className={cn(
                      "flex h-full min-h-0 flex-col gap-4 overflow-hidden p-4",
                      DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                    )}
                  >
                    <div className="flex flex-col gap-3">
                      <div className="flex flex-wrap items-start justify-between gap-3">
                        <div className="min-w-0 flex-1">
                          {selectedDocumentEditing ? (
                            <Input
                              value={selectedDocumentDraft?.name ?? ""}
                              onChange={(event) =>
                                setSubtitleStyleDocumentDraft((current) =>
                                  current && current.id === selectedDocument.id
                                    ? { ...current, name: event.target.value }
                                    : current,
                                )
                              }
                              placeholder={t("library.config.subtitleStyles.documentName")}
                              className="h-8 w-full max-w-xl"
                            />
                          ) : (
                            <div className="truncate text-sm font-semibold text-foreground">
                              {effectiveSelectedDocument?.name ||
                                t("library.config.subtitleStyles.untitledDocument")}
                            </div>
                          )}
                        </div>

                        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                          <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
                            <Button
                              type="button"
                              variant="ghost"
                              size="compact"
                              className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                              onClick={() => {
                                const duplicateSource =
                                  selectedDocumentEditing &&
                                  selectedDocumentDraft?.id ===
                                    selectedDocument.id
                                    ? selectedDocumentDraft
                                    : effectiveSelectedDocument;
                                if (!duplicateSource) {
                                  return;
                                }
                                handleCreateDuplicateSubtitleStyleDocument(
                                  duplicateSource,
                                );
                              }}
                            >
                              <Copy className="h-3.5 w-3.5" />
                              {t("library.config.subtitleStyles.duplicate")}
                            </Button>

                            {selectedDocumentReadOnly ? null : selectedDocumentEditing ? (
                              <>
                                <Button
                                  type="button"
                                  variant="ghost"
                                  size="compact"
                                  className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
                                  onClick={() =>
                                    handleSaveSubtitleStyleDocumentEdit(
                                      selectedDocument.id,
                                    )
                                  }
                                >
                                  <Check className="h-3.5 w-3.5" />
                                  {t("common.done")}
                                </Button>
                                <Button
                                  type="button"
                                  variant="ghost"
                                  size="compact"
                                  className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
                                  onClick={() =>
                                    handleCancelSubtitleStyleDocumentEdit(
                                      selectedDocument.id,
                                    )
                                  }
                                >
                                  <X className="h-3.5 w-3.5" />
                                  {t("common.cancel")}
                                </Button>
                              </>
                            ) : (
                              <Button
                                type="button"
                                variant="ghost"
                                size="compact"
                                className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
                                onClick={() =>
                                  handleStartSubtitleStyleDocumentEdit(
                                    selectedDocument,
                                  )
                                }
                              >
                                <PencilLine className="h-3.5 w-3.5" />
                                {t("common.edit")}
                              </Button>
                            )}
                          </div>
                          <Switch
                            checked={
                              effectiveSelectedDocument?.enabled !== false
                            }
                            disabled={selectedDocumentReadOnly}
                            aria-label={t("library.config.subtitleStyles.enabled")}
                            title={t("library.config.subtitleStyles.enabled")}
                            onCheckedChange={(checked) => {
                              handleUpdateSubtitleStyleDocument(
                                selectedDocument.id,
                                { enabled: checked },
                              );
                              setSubtitleStyleDocumentDraft((current) =>
                                current && current.id === selectedDocument.id
                                  ? { ...current, enabled: checked }
                                  : current,
                              );
                              onRequestPersist?.();
                            }}
                          />
                        </div>
                      </div>

                      <div className="flex flex-wrap gap-1.5">
                        {[
                          {
                            key: "source",
                            label: selectedDocumentSourceLabel,
                            className: "",
                          },
                          {
                            key: "resolution",
                            label: `${t("library.config.subtitleStyles.playRes")} ${selectedDocumentResolutionLabel}`,
                            className: "",
                          },
                          {
                            key: "styles",
                            label: `${t("library.config.subtitleStyles.stylesCount")} ${selectedDocumentSummary?.styleCount ?? 0}`,
                            className: "",
                          },
                          selectedDocumentMatchedFontsLabel
                            ? {
                                key: "matched-fonts",
                                label: selectedDocumentMatchedFontsLabel,
                                className:
                                  "border-emerald-300/60 text-emerald-800",
                              }
                            : null,
                          selectedDocumentMissingFontsLabel
                            ? {
                                key: "missing-fonts",
                                label: selectedDocumentMissingFontsLabel,
                                className: "border-amber-300/60 text-amber-800",
                              }
                            : null,
                        ]
                          .filter(
                            (
                              badge,
                            ): badge is {
                              key: string;
                              label: string;
                              className: string;
                            } => Boolean(badge),
                          )
                          .map((badge) => (
                            <Badge
                              key={badge.key}
                              variant="outline"
                              title={badge.label}
                              className={cn(
                                "max-w-[260px] truncate text-[11px]",
                                badge.className,
                              )}
                            >
                              {badge.label}
                            </Badge>
                          ))}
                      </div>
                    </div>

                    <div className="mt-4 min-h-0 flex-1 space-y-4 overflow-y-auto pr-1">
                      <div className="grid gap-3 md:grid-cols-3">
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.detectedFormat")}
                          value={
                            selectedDocumentSummary?.detectedFormat
                              ? selectedDocumentSummary.detectedFormat.toUpperCase()
                              : "ASS"
                          }
                        />
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.playRes")}
                          value={
                            selectedDocumentSummary?.playResX &&
                            selectedDocumentSummary.playResY
                              ? `${selectedDocumentSummary.playResX} x ${selectedDocumentSummary.playResY}`
                              : "-"
                          }
                        />
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.stylesCount")}
                          value={String(
                            selectedDocumentSummary?.styleCount ?? 0,
                          )}
                        />
                      </div>

                      <div className="grid gap-3 md:grid-cols-3">
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.scriptType")}
                          value={selectedDocumentSummary?.scriptType || "-"}
                        />
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.dialogueCount")}
                          value={String(
                            selectedDocumentSummary?.dialogueCount ?? 0,
                          )}
                        />
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.commentCount")}
                          value={String(
                            selectedDocumentSummary?.commentCount ?? 0,
                          )}
                        />
                      </div>

                      <div className="grid gap-3 md:grid-cols-2">
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.styleNames")}
                          value={
                            selectedDocumentSummary?.styleNames.length
                              ? selectedDocumentSummary.styleNames.join(", ")
                              : t("library.config.taskDefaults.none")
                          }
                        />
                        <ReadOnlyInfoField
                          label={t("library.config.subtitleStyles.fontCoverage")}
                          value={
                            isSystemFontsLoading
                              ? t("common.loading")
                              : selectedDocumentMissingFonts.length > 0
                                ? `${selectedDocumentAvailableFonts.length}/${selectedDocumentSummary?.fonts.length ?? 0} ${t("library.config.subtitleStyles.defaultDocumentReady")} · ${t("library.config.subtitleStyles.missingFonts")} ${selectedDocumentMissingFonts.join(", ")}`
                                : (selectedDocumentSummary?.fonts.length ?? 0) >
                                    0
                                  ? `${selectedDocumentSummary?.fonts.length ?? 0}/${selectedDocumentSummary?.fonts.length ?? 0} ${t("library.config.subtitleStyles.defaultDocumentReady")}`
                                  : t("library.config.subtitleStyles.noFontsReferenced")
                          }
                        />
                      </div>

                      <ReadOnlyInfoField
                        label={t("library.config.subtitleStyles.requiredFonts")}
                        value={
                          selectedDocumentSummary?.fonts.length
                            ? selectedDocumentSummary.fonts.join(", ")
                            : t("library.config.taskDefaults.none")
                        }
                      />

                      <ReadOnlyInfoField
                        label={t("library.config.subtitleStyles.featureFlags")}
                        value={
                          selectedDocumentSummary?.featureFlags.length
                            ? selectedDocumentSummary.featureFlags
                                .map((flag) =>
                                  formatSubtitleStyleDocumentFeatureFlag(flag),
                                )
                                .join(", ")
                            : t("library.config.taskDefaults.none")
                        }
                      />

                      {selectedDocumentSummary?.validationIssues.length ? (
                        <div
                          className={cn(
                            "space-y-2 px-3 py-3",
                            DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                            "border-amber-300/70 bg-amber-50/80",
                          )}
                        >
                          <div className="text-xs font-medium text-amber-900">
                            {t("library.config.subtitleStyles.validationIssues")}
                          </div>
                          <div className="space-y-1 text-xs leading-5 text-amber-900/80">
                            {selectedDocumentSummary.validationIssues.map(
                              (issue) => (
                                <div key={`${selectedDocument.id}-${issue}`}>
                                  {issue}
                                </div>
                              ),
                            )}
                          </div>
                        </div>
                      ) : null}

                      <div
                        className={cn(
                          "space-y-3 px-3 py-3",
                          DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                        )}
                      >
                        <div className="text-xs font-medium text-foreground">
                          {t("library.config.subtitleStyles.raw")}
                        </div>
                        {selectedDocumentReadOnly ||
                        !selectedDocumentEditing ? (
                          <pre className="min-h-[320px] whitespace-pre-wrap break-words rounded-lg border border-border/70 bg-background/80 px-3 py-3 font-mono text-[11px] leading-5 text-foreground">
                            {effectiveSelectedDocument?.content ?? ""}
                          </pre>
                        ) : (
                          <ConfigTextarea
                            value={effectiveSelectedDocument?.content ?? ""}
                            onChange={(event) =>
                              setSubtitleStyleDocumentDraft((current) =>
                                current && current.id === selectedDocument.id
                                  ? { ...current, content: event.target.value }
                                  : current,
                              )
                            }
                            placeholder={t("library.config.subtitleStyles.assContent")}
                            rows={30}
                            className="min-h-[320px] resize-y font-mono text-[11px] leading-5"
                          />
                        )}
                      </div>
                    </div>

                    {!selectedDocumentReadOnly ? (
                      <div className="flex justify-center pt-2">
                        <Button
                          type="button"
                          variant="destructive"
                          size="compact"
                          className="min-w-[160px]"
                          onClick={() =>
                            handleDeleteSubtitleStyleDocument(
                              selectedDocument.id,
                            )
                          }
                        >
                          {t("common.delete")}
                        </Button>
                      </div>
                    ) : null}
                  </div>
                ) : (
                  <EmptyConfigState
                    title={t("library.config.subtitleStyles.emptyDocuments")}
                  />
                )}
              </div>
            </div>
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "profiles"}
            className="min-h-0 w-full flex-1"
          >
            <ConfigMasterDetailLayout
              sidebar={
                <div
                  className={cn(
                    "flex h-full min-h-0 flex-col overflow-hidden p-4",
                    DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
                  )}
                >
                  {subtitleExportPresets.length > 0 ? (
                    <div className="mb-3 flex items-center gap-2 rounded-lg border border-border/60 bg-background/75 px-2.5 py-2">
                      <Search className="h-3.5 w-3.5 text-muted-foreground" />
                      <Input
                        value={subtitleExportPresetSearch}
                        onChange={(event) =>
                          setSubtitleExportPresetSearch(event.target.value)
                        }
                        placeholder={t("library.config.videoExportPresets.searchPlaceholder")}
                        className="h-7 border-none bg-transparent px-0 text-xs shadow-none focus-visible:ring-0"
                      />
                      <Badge
                        variant="outline"
                        className="h-5 shrink-0 px-1.5 text-[10px]"
                      >
                        {visibleSubtitleExportPresets.length}
                      </Badge>
                    </div>
                  ) : null}
                  {subtitleExportPresets.length === 0 ? (
                    <EmptyConfigState
                      title={t("library.config.subtitleStyles.emptySubtitleExportPresets")}
                    />
                  ) : visibleSubtitleExportPresets.length === 0 ? (
                    <EmptyConfigState
                      title={t("library.config.videoExportPresets.searchEmpty")}
                    />
                  ) : (
                    <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                      {groupedSubtitleExportPresets.map((group) => (
                        <div key={group.format} className="space-y-2">
                          <div className="flex items-center justify-between px-1">
                            <div className="text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
                              {group.format.toUpperCase()}
                            </div>
                            <div className="text-[10px] text-muted-foreground">
                              {group.profiles.length}
                            </div>
                          </div>
                          {group.profiles.map((profile) => {
                            const isSelected =
                              profile.id === selectedProfile?.id;
                            const mediaStrategy =
                              normalizeSubtitleExportMediaStrategy(
                                profile.mediaStrategy ?? "",
                              );
                            const mediaStrategyLabel =
                              resolveSubtitleExportMediaStrategyLabel(
                                mediaStrategy,
                              );
                            const isDefaultProfile =
                              subtitleStyleDefaults.subtitleExportPresetId ===
                              profile.id;
                            const profileReadOnly =
                              isBuiltInSubtitleExportPreset(profile);
                            const profileOriginLabel = profileReadOnly
                              ? t("library.config.subtitleStyles.builtinBadge")
                              : t("library.config.subtitleStyles.customBadge");
                            const profileMetaParts = [
                              mediaStrategyLabel,
                              profileOriginLabel,
                            ];
                            if (isDefaultProfile) {
                              profileMetaParts.push(
                                t("library.config.subtitleStyles.defaultBadge"),
                              );
                            }
                            return (
                              <ConfigNavigationItem
                                key={profile.id}
                                title={
                                  profile.name ||
                                  t("library.config.subtitleStyles.exportProfileFallback")
                                }
                                description={profileMetaParts.join(" · ")}
                                selected={isSelected}
                                disabled={selectedProfileEditing && !isSelected}
                                compact
                                onClick={() => {
                                  if (selectedProfileEditing && !isSelected) {
                                    return;
                                  }
                                  setSelectedSubtitleExportPresetId(profile.id);
                                }}
                              />
                            );
                          })}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              }
              detail={
                <div className="h-full min-w-0 min-h-0">
                  {selectedProfile ? (
                    <ConfigDetailPanel
                      title={
                        selectedProfileEditing ? (
                          <Input
                            value={selectedProfile.name ?? ""}
                            onChange={(event) =>
                              handleUpdateSubtitleExportPreset(
                                selectedProfile.id,
                                { name: event.target.value },
                              )
                            }
                            placeholder={t("library.config.subtitleStyles.exportProfileFallback")}
                            className="h-8 max-w-full border-border/70 bg-background/80 text-sm font-semibold"
                          />
                        ) : (
                          selectedProfile.name ||
                          t("library.config.subtitleStyles.exportProfileFallback")
                        )
                      }
                      actions={
                        <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                          <div className="inline-flex overflow-hidden rounded-lg border border-border/70 bg-background/80 shadow-sm">
                            <Button
                              type="button"
                              variant="ghost"
                              size="compact"
                              className="gap-2 rounded-none border-0 px-3 hover:bg-background"
                              onClick={() =>
                                handleDuplicateSubtitleExportPreset(
                                  selectedProfile,
                                )
                              }
                            >
                              <Copy className="h-3.5 w-3.5" />
                              {t("library.config.subtitleStyles.duplicate")}
                            </Button>
                            {!selectedProfileReadOnly ? (
                              <Button
                                type="button"
                                variant="ghost"
                                size="compact"
                                className="gap-2 rounded-none border-0 border-l border-border/60 px-3 hover:bg-background"
                                onClick={() =>
                                  selectedProfileEditing
                                    ? closeCardEditor(selectedProfileCardID)
                                    : openCardEditor(selectedProfileCardID)
                                }
                              >
                                {selectedProfileEditing ? (
                                  <Check className="h-3.5 w-3.5" />
                                ) : (
                                  <PencilLine className="h-3.5 w-3.5" />
                                )}
                                {selectedProfileEditing
                                  ? t("common.done")
                                  : t("common.edit")}
                              </Button>
                            ) : null}
                          </div>
                        </div>
                      }
                      badges={
                        <>
                          <Badge
                            variant="outline"
                            className="text-[10px] tracking-[0.08em]"
                          >
                            {selectedProfileTargetFormat.toUpperCase()}
                          </Badge>
                          <Badge
                            variant="outline"
                            className="text-[10px] tracking-[0.08em]"
                          >
                            {resolveSubtitleExportMediaStrategyLabel(
                              selectedProfileMediaStrategy,
                            )}
                          </Badge>
                          {subtitleStyleDefaults.subtitleExportPresetId ===
                          selectedProfile.id ? (
                            <Badge
                              variant="outline"
                              className="text-[10px] tracking-[0.08em]"
                            >
                              {t("library.config.subtitleStyles.defaultBadge")}
                            </Badge>
                          ) : null}
                          <Badge
                            variant="outline"
                            className="text-[10px] tracking-[0.08em]"
                          >
                            {selectedProfileReadOnly
                              ? t("library.config.subtitleStyles.builtinBadge")
                              : t("library.config.subtitleStyles.customBadge")}
                          </Badge>
                        </>
                      }
                      footer={
                        !selectedProfileReadOnly ? (
                          <div className="flex justify-center pt-2">
                            <Button
                              type="button"
                              variant="destructive"
                              size="compact"
                              className="min-w-[160px]"
                              onClick={() =>
                                handleDeleteSubtitleExportPreset(
                                  selectedProfile.id,
                                )
                              }
                            >
                              {t("common.delete")}
                            </Button>
                          </div>
                        ) : undefined
                      }
                      contentClassName="space-y-2"
                    >
                      <fieldset
                        disabled={!selectedProfileEditing}
                        className="m-0 space-y-2 border-0 p-0"
                      >
                        <ConfigSelectField
                          label={t("library.config.subtitleStyles.exportProfileMediaStrategy")}
                          inline
                          value={selectedProfileMediaStrategy}
                          onChange={(nextValue) =>
                            handleUpdateSubtitleExportPreset(
                              selectedProfile.id,
                              {
                                mediaStrategy:
                                  normalizeSubtitleExportMediaStrategy(
                                    nextValue,
                                  ),
                              },
                            )
                          }
                          options={subtitleExportMediaStrategyOptions}
                        />

                        {selectedProfileTargetFormat === "srt" ? (
                          <ConfigSelectField
                            label={t("library.workspace.dialogs.exportSubtitle.encoding")}
                            inline
                            value={
                              selectedProfileConfig.srt?.encoding || "utf-8"
                            }
                            onChange={(nextValue) =>
                              handleUpdateSubtitleExportPreset(
                                selectedProfile.id,
                                {
                                  config: {
                                    ...selectedProfileConfig,
                                    srt: {
                                      ...(selectedProfileConfig.srt ?? {}),
                                      encoding: nextValue,
                                    },
                                  },
                                },
                              )
                            }
                            options={[
                              { value: "utf-8", label: "UTF-8" },
                              { value: "gbk", label: "GBK" },
                              { value: "big5", label: "Big5" },
                            ]}
                          />
                        ) : null}

                        {selectedProfileTargetFormat === "vtt" ? (
                          <>
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.kind")}
                              inline
                              value={
                                selectedProfileConfig.vtt?.kind || "subtitles"
                              }
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      vtt: {
                                        ...(selectedProfileConfig.vtt ?? {}),
                                        kind: nextValue,
                                      },
                                    },
                                  },
                                )
                              }
                              options={subtitleExportVttKindOptions}
                            />
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.language")}
                              inline
                              value={
                                (
                                  selectedProfileConfig.vtt?.language ?? ""
                                ).trim() || "en-US"
                              }
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      vtt: {
                                        ...(selectedProfileConfig.vtt ?? {}),
                                        language: nextValue,
                                      },
                                    },
                                  },
                                )
                              }
                              options={subtitleExportLanguageOptions.concat(
                                subtitleExportLanguageOptions.some(
                                  (option) =>
                                    option.value ===
                                    ((
                                      selectedProfileConfig.vtt?.language ?? ""
                                    ).trim() || "en-US"),
                                )
                                  ? []
                                  : [
                                      {
                                        value:
                                          (
                                            selectedProfileConfig.vtt
                                              ?.language ?? ""
                                          ).trim() || "en-US",
                                        label:
                                          (
                                            selectedProfileConfig.vtt
                                              ?.language ?? ""
                                          ).trim() || "en-US",
                                      },
                                    ],
                              )}
                            />
                          </>
                        ) : null}

                        {selectedProfileTargetFormat === "ass" ? (
                          <>
                            <div
                              className={cn(
                                "grid gap-3 px-3 py-2.5 md:grid-cols-2",
                                DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                              )}
                            >
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.config.subtitleStyles.playResX")}
                                </div>
                                <Input
                                  type="number"
                                  min={1}
                                  value={String(
                                    selectedProfileConfig.ass?.playResX ?? 1920,
                                  )}
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          ass: {
                                            ...(selectedProfileConfig.ass ??
                                              {}),
                                            playResX: parsePositiveInt(
                                              event.target.value,
                                              selectedProfileConfig.ass
                                                ?.playResX ?? 1920,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 border-border/70 bg-background/80"
                                />
                              </div>
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.config.subtitleStyles.playResY")}
                                </div>
                                <Input
                                  type="number"
                                  min={1}
                                  value={String(
                                    selectedProfileConfig.ass?.playResY ?? 1080,
                                  )}
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          ass: {
                                            ...(selectedProfileConfig.ass ??
                                              {}),
                                            playResY: parsePositiveInt(
                                              event.target.value,
                                              selectedProfileConfig.ass
                                                ?.playResY ?? 1080,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 border-border/70 bg-background/80"
                                />
                              </div>
                            </div>
                          </>
                        ) : null}

                        {selectedProfileTargetFormat === "itt" ? (
                          <>
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.frameRate")}
                              inline
                              value={resolveITTFrameRatePresetValue(
                                selectedProfileConfig.itt?.frameRate,
                                selectedProfileConfig.itt?.frameRateMultiplier,
                              )}
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      itt: {
                                        ...(selectedProfileConfig.itt ?? {}),
                                        ...resolveITTFrameTimingFromPresetValue(
                                          nextValue,
                                        ),
                                      },
                                    },
                                  },
                                )
                              }
                              options={ITT_FRAME_RATE_PRESETS.map(
                                (preset) => ({
                                  value: preset.value,
                                  label: preset.label,
                                }),
                              )}
                            />
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.language")}
                              inline
                              value={
                                (
                                  selectedProfileConfig.itt?.language ?? ""
                                ).trim() || "en-US"
                              }
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      itt: {
                                        ...(selectedProfileConfig.itt ?? {}),
                                        language: nextValue,
                                      },
                                    },
                                  },
                                )
                              }
                              options={subtitleExportLanguageOptions}
                            />
                          </>
                        ) : null}

                        {selectedProfileTargetFormat === "fcpxml" ? (
                          <>
                            <div
                              className={cn(
                                "grid gap-3 px-3 py-2.5 md:grid-cols-2",
                                DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                              )}
                            >
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.workspace.dialogs.exportSubtitle.width")}
                                </div>
                                <Input
                                  type="number"
                                  min={1}
                                  value={String(
                                    selectedProfileConfig.fcpxml?.width ?? 1920,
                                  )}
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          fcpxml: {
                                            ...(selectedProfileConfig.fcpxml ??
                                              {}),
                                            width: parsePositiveInt(
                                              event.target.value,
                                              selectedProfileConfig.fcpxml
                                                ?.width ?? 1920,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 border-border/70 bg-background/80"
                                />
                              </div>
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.workspace.dialogs.exportSubtitle.height")}
                                </div>
                                <Input
                                  type="number"
                                  min={1}
                                  value={String(
                                    selectedProfileConfig.fcpxml?.height ??
                                      1080,
                                  )}
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          fcpxml: {
                                            ...(selectedProfileConfig.fcpxml ??
                                              {}),
                                            height: parsePositiveInt(
                                              event.target.value,
                                              selectedProfileConfig.fcpxml
                                                ?.height ?? 1080,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 border-border/70 bg-background/80"
                                />
                              </div>
                            </div>
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.frameDuration")}
                              inline
                              value={normalizeFCPXMLFrameDuration(
                                selectedProfileConfig.fcpxml?.frameDuration,
                              )}
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      fcpxml: {
                                        ...(selectedProfileConfig.fcpxml ?? {}),
                                        frameDuration:
                                          normalizeFCPXMLFrameDuration(
                                            nextValue,
                                          ),
                                      },
                                    },
                                  },
                                )
                              }
                              options={FCPXML_FRAME_DURATION_PRESETS.map(
                                (preset) => ({
                                  value: preset.value,
                                  label: `${preset.label} (${preset.value})`,
                                }),
                              )}
                            />
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.colorSpace")}
                              inline
                              value={
                                selectedProfileConfig.fcpxml?.colorSpace ||
                                "1-1-1 (Rec. 709)"
                              }
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      fcpxml: {
                                        ...(selectedProfileConfig.fcpxml ?? {}),
                                        colorSpace: nextValue,
                                      },
                                    },
                                  },
                                )
                              }
                              options={[
                                {
                                  value: "1-1-1 (Rec. 709)",
                                  label: "Rec. 709",
                                },
                                { value: "Rec. 2020", label: "Rec. 2020" },
                                { value: "P3-D65", label: "P3-D65" },
                              ]}
                            />
                            <div
                              className={cn(
                                "grid gap-3 px-3 py-2.5 md:grid-cols-2",
                                DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                              )}
                            >
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.workspace.dialogs.exportSubtitle.version")}
                                </div>
                                <Select
                                  value={
                                    normalizeFCPXMLVersion(
                                      selectedProfileConfig.fcpxml?.version,
                                    )
                                  }
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          fcpxml: {
                                            ...(selectedProfileConfig.fcpxml ??
                                              {}),
                                            version: normalizeFCPXMLVersion(
                                              event.target.value,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 w-full border-border/70 bg-background/80"
                                >
                                  {FCPXML_VERSION_OPTIONS.map((option) => (
                                    <option
                                      key={option.value}
                                      value={option.value}
                                    >
                                      {option.label}
                                    </option>
                                  ))}
                                </Select>
                              </div>
                              <div className="space-y-1">
                                <div className="text-xs font-medium text-foreground">
                                  {t("library.workspace.dialogs.exportSubtitle.defaultLane")}
                                </div>
                                <Input
                                  type="number"
                                  value={String(
                                    selectedProfileConfig.fcpxml?.defaultLane ??
                                      1,
                                  )}
                                  onChange={(event) =>
                                    handleUpdateSubtitleExportPreset(
                                      selectedProfile.id,
                                      {
                                        config: {
                                          ...selectedProfileConfig,
                                          fcpxml: {
                                            ...(selectedProfileConfig.fcpxml ??
                                              {}),
                                            defaultLane: parseNonNegativeInt(
                                              event.target.value,
                                              selectedProfileConfig.fcpxml
                                                ?.defaultLane ?? 1,
                                            ),
                                          },
                                        },
                                      },
                                    )
                                  }
                                  className="h-8 border-border/70 bg-background/80"
                                />
                              </div>
                            </div>
                            <ConfigSelectField
                              label={t("library.workspace.dialogs.exportSubtitle.startTimecodeSeconds")}
                              inline
                              value={String(
                                normalizeFCPXMLStartTimecodeSeconds(
                                  selectedProfileConfig.fcpxml
                                    ?.startTimecodeSeconds,
                                ),
                              )}
                              onChange={(nextValue) =>
                                handleUpdateSubtitleExportPreset(
                                  selectedProfile.id,
                                  {
                                    config: {
                                      ...selectedProfileConfig,
                                      fcpxml: {
                                        ...(selectedProfileConfig.fcpxml ?? {}),
                                        startTimecodeSeconds:
                                          normalizeFCPXMLStartTimecodeSeconds(
                                            Number.parseInt(nextValue, 10),
                                          ),
                                      },
                                    },
                                  },
                                )
                              }
                              options={FCPXML_START_TIMECODE_PRESETS.map(
                                (preset) => ({
                                  value: String(preset.value),
                                  label: preset.label,
                                }),
                              )}
                            />
                          </>
                        ) : null}

                      </fieldset>
                    </ConfigDetailPanel>
                  ) : (
                    <EmptyConfigState
                      title={t("library.config.subtitleStyles.emptySubtitleExportPresets")}
                    />
                  )}
                </div>
              }
            />
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "fonts"}
            className="space-y-4"
          >
            <ConfigSectionCard
              title={t("library.config.subtitleStyles.fontManagementTitle")}
              description={t("library.config.subtitleStyles.fontManagementDescription")}
              icon={Database}
              contentClassName="grid gap-3 md:grid-cols-2 xl:grid-cols-4"
            >
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.requiredFonts")}
                value={String(subtitleFontCoverage.required.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.resolvedFonts")}
                value={String(subtitleFontCoverage.resolved.length)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.fontSources")}
                value={String(enabledFontSourceCount)}
              />
              <ReadOnlyInfoField
                label={t("library.config.subtitleStyles.missingFonts")}
                value={String(subtitleFontCoverage.missing.length)}
              />
            </ConfigSectionCard>

            <ConfigSectionCard
              title={t("library.config.subtitleStyles.referencedFontsTitle")}
              description={t("library.config.subtitleStyles.referencedFontsDescription")}
              icon={Type}
              contentClassName="space-y-2"
            >
              {subtitleReferencedFonts.length === 0 ? (
                <EmptyConfigState
                  title={t("library.config.subtitleStyles.noFontsReferenced")}
                />
              ) : (
                <div className="max-h-[640px] space-y-2 overflow-y-auto pr-1">
                  {subtitleReferencedFonts.map((entry) => {
                    const isMissing = entry.status === "missing";
                    const searchState =
                      remoteFontSearchState[entry.family.trim()];
                    const installableCandidates = (
                      searchState?.candidates ?? []
                    ).filter((candidate) => candidate.installable);
                    const unavailableCandidates = (
                      searchState?.candidates ?? []
                    ).filter((candidate) => !candidate.installable);
                    const installableMatchCount = installableCandidates.length;
                    const unavailableMatchCount = unavailableCandidates.length;
                    const primaryInstallableCandidate =
                      installableCandidates[0] ?? null;
                    const isSearchLoading = Boolean(searchState?.loading);
                    const isRepairingUser = Boolean(
                      repairingFontFamilies[`user:${entry.family.trim()}`],
                    );
                    const unavailableSummary =
                      formatRemoteFontUnavailableCandidates(
                        unavailableCandidates,
                        language,
                      );
                    return (
                      <div
                        key={entry.family}
                        className={cn(
                          "grid gap-3 px-3 py-3 md:grid-cols-[minmax(0,1fr)_auto]",
                          DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                        )}
                      >
                        <div className="min-w-0">
                          <div className="flex flex-wrap items-center gap-2">
                            <div className="truncate text-sm font-medium text-foreground">
                              {entry.family}
                            </div>
                            <Badge
                              variant="outline"
                              className={cn(
                                "text-[10px] tracking-[0.08em]",
                                isMissing
                                  ? "border-amber-300/60 text-amber-800"
                                  : "border-emerald-300/60 text-emerald-800",
                              )}
                            >
                              {isMissing
                                ? t("library.config.subtitleStyles.fontStatusMissing")
                                : t("library.config.subtitleStyles.fontStatusInstalled")}
                            </Badge>
                            {entry.systemFamily && !isMissing ? (
                              <Badge
                                variant="outline"
                                className="text-[10px] tracking-[0.08em]"
                              >
                                {t("library.config.subtitleStyles.systemFontFamily")}{" "}
                                {entry.systemFamily}
                              </Badge>
                            ) : null}
                          </div>
                          <div className="mt-1 text-xs leading-5 text-muted-foreground">
                            {t("library.config.subtitleStyles.usedByStyles")}{" "}
                            {entry.documents.join(", ")}
                          </div>
                          {isMissing && !hasEnabledFontSource ? (
                            <div className="mt-1 text-xs leading-5 text-muted-foreground">
                              {t("library.config.subtitleStyles.noFontSourceEnabled")}
                            </div>
                          ) : null}
                          {isMissing && hasEnabledFontSource ? (
                            <div className="mt-1 text-xs leading-5 text-muted-foreground">
                              {isSearchLoading
                                ? t("library.config.subtitleStyles.searchingRemoteFonts")
                                : searchState?.error
                                  ? searchState.error
                                  : installableMatchCount > 0
                                    ? `${t("library.config.subtitleStyles.remoteMatches")} ${installableMatchCount} · ${formatRemoteFontSourceNames(installableCandidates)}${
                                        unavailableMatchCount > 0
                                          ? ` · ${t("library.config.subtitleStyles.remoteUnavailableMatches")} ${unavailableMatchCount}`
                                          : ""
                                      }`
                                    : unavailableMatchCount > 0
                                      ? `${t("library.config.subtitleStyles.remoteMatchUnavailable")} ${formatRemoteFontSourceNames(unavailableCandidates)}`
                                      : t("library.config.subtitleStyles.noRemoteFontMatch")}
                            </div>
                          ) : null}
                        </div>
                        {isMissing ? (
                          <div className="flex items-center justify-end gap-2">
                            {primaryInstallableCandidate ? (
                              <Button
                                type="button"
                                variant="outline"
                                size="compact"
                                className="gap-2"
                                disabled={
                                  !hasEnabledFontSource ||
                                  isSearchLoading ||
                                  isRepairingUser
                                }
                                onClick={() =>
                                  handleRepairSubtitleStyleFont(
                                    entry.family,
                                    "user",
                                    primaryInstallableCandidate.sourceId,
                                  )
                                }
                              >
                                {isRepairingUser ? (
                                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                ) : (
                                  <Download className="h-3.5 w-3.5" />
                                )}
                                {isRepairingUser
                                  ? t("library.config.subtitleStyles.repairingFont")
                                  : t("library.config.subtitleStyles.installToUserFont")}
                              </Button>
                            ) : unavailableMatchCount > 0 &&
                              !isSearchLoading &&
                              !searchState?.error ? (
                              <div className="max-w-[260px] text-right text-xs leading-5 text-muted-foreground">
                                {unavailableSummary}
                              </div>
                            ) : null}
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              )}
            </ConfigSectionCard>
          </ConditionalPanel>

          <ConditionalPanel
            active={page === "sources"}
            className="space-y-4"
          >
            <div className="space-y-4">
              <ConfigSectionCard
                title={t("library.config.subtitleStyles.styleSourcesTitle")}
                description={t("library.config.subtitleStyles.styleSourcesDescription")}
                icon={FileText}
                contentClassName="space-y-3"
              >
                {subtitleStyleDocumentSources.length === 0 ? (
                  <EmptyConfigState
                    title={t("library.config.subtitleStyles.emptyStyleSources")}
                  />
                ) : (
                  subtitleStyleDocumentSources.map((source) => {
                    const browserState = remoteStyleBrowserState[source.id];
                    return (
                      <div
                        key={source.id}
                        className={cn(
                          "space-y-3 p-3",
                          DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                        )}
                      >
                        <div className="flex items-start justify-between gap-3">
                          <div className="min-w-0 flex-1 space-y-1">
                            <Input
                              value={source.name ?? ""}
                              onChange={(event) =>
                                handleUpdateSubtitleStyleSource(source.id, {
                                  name: event.target.value,
                                })
                              }
                              onBlur={() => onRequestPersist?.()}
                              placeholder={t("library.config.subtitleStyles.sourceName")}
                              className="h-8"
                            />
                            <div className="flex flex-wrap gap-1.5">
                              <Badge
                                variant="outline"
                                className="text-[10px] tracking-[0.08em]"
                              >
                                {t("library.config.subtitleStyles.sourceKindStyle")}
                              </Badge>
                              <Badge
                                variant="outline"
                                className={cn(
                                  "text-[10px] tracking-[0.08em]",
                                  source.enabled === true
                                    ? "border-emerald-300/60 text-emerald-800"
                                    : "text-muted-foreground",
                                )}
                              >
                                {source.enabled === true
                                  ? t("library.config.subtitleStyles.enabled")
                                  : t("library.config.subtitleStyles.disabled")}
                              </Badge>
                            </div>
                          </div>
                          <div className="flex items-center gap-2">
                            <Button
                              type="button"
                              variant="outline"
                              size="compact"
                              className="gap-2"
                              onClick={() =>
                                handleBrowseSubtitleStyleSource(source)
                              }
                              disabled={browserState?.loading}
                            >
                              {browserState?.loading ? (
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              ) : (
                                <Search className="h-3.5 w-3.5" />
                              )}
                              {t("library.config.subtitleStyles.browseSource")}
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="compactIcon"
                              className="h-8 w-8"
                              onClick={() =>
                                handleDeleteSubtitleStyleSource(source.id)
                              }
                              aria-label={t("common.delete")}
                            >
                              <Trash2 className="h-4 w-4" />
                            </Button>
                          </div>
                        </div>

                        <div className="grid gap-3 md:grid-cols-[minmax(0,1fr)_160px]">
                          <div className="space-y-1">
                            <div className="text-xs font-medium text-foreground">
                              {t("library.config.subtitleStyles.provider")}
                            </div>
                            <Select
                              value={source.provider ?? "github"}
                              onChange={(event) =>
                                handleUpdateSubtitleStyleSource(source.id, {
                                  provider: event.target.value,
                                })
                              }
                              onBlur={() => onRequestPersist?.()}
                              className="h-8 w-full min-w-0 border-border/70 bg-background/80"
                            >
                              <option value="github">GitHub</option>
                            </Select>
                          </div>
                          <div className="space-y-1">
                            <div className="text-xs font-medium text-foreground">
                              {t("library.config.subtitleStyles.sourceEnabled")}
                            </div>
                            <div className="flex h-8 items-center">
                              <Switch
                                checked={source.enabled === true}
                                onCheckedChange={(checked) => {
                                  handleUpdateSubtitleStyleSource(source.id, {
                                    enabled: checked,
                                  });
                                  onRequestPersist?.();
                                }}
                              />
                            </div>
                          </div>
                        </div>

                        <div className="grid gap-3 lg:grid-cols-[minmax(0,0.8fr)_minmax(0,0.8fr)_160px_180px]">
                          <Input
                            value={source.owner ?? ""}
                            onChange={(event) =>
                              handleUpdateSubtitleStyleSource(source.id, {
                                owner: event.target.value,
                              })
                            }
                            onBlur={() => onRequestPersist?.()}
                            placeholder={t("library.config.subtitleStyles.owner")}
                            className="h-8"
                          />
                          <Input
                            value={source.repo ?? ""}
                            onChange={(event) =>
                              handleUpdateSubtitleStyleSource(source.id, {
                                repo: event.target.value,
                              })
                            }
                            onBlur={() => onRequestPersist?.()}
                            placeholder={t("library.config.subtitleStyles.repo")}
                            className="h-8"
                          />
                          <Input
                            value={source.ref ?? ""}
                            onChange={(event) =>
                              handleUpdateSubtitleStyleSource(source.id, {
                                ref: event.target.value,
                              })
                            }
                            onBlur={() => onRequestPersist?.()}
                            placeholder={t("library.config.subtitleStyles.ref")}
                            className="h-8"
                          />
                          <Input
                            value={source.manifestPath ?? ""}
                            onChange={(event) =>
                              handleUpdateSubtitleStyleSource(source.id, {
                                manifestPath: event.target.value,
                              })
                            }
                            onBlur={() => onRequestPersist?.()}
                            placeholder={t("library.config.subtitleStyles.manifestPath")}
                            className="h-8"
                          />
                        </div>

                        <div className="grid gap-3 md:grid-cols-2">
                          <ReadOnlyInfoField
                            label={t("library.config.subtitleStyles.syncStatus")}
                            value={
                              source.syncStatus
                                ? resolveSubtitleStyleSyncStatusLabel(
                                    source.syncStatus,
                                  )
                                : t("library.config.subtitleStyles.syncIdle")
                            }
                          />
                          <ReadOnlyInfoField
                            label={t("library.config.subtitleStyles.lastError")}
                            value={
                              source.lastError ||
                              t("library.config.taskDefaults.none")
                            }
                          />
                        </div>

                        {browserState?.error ? (
                          <div className="rounded-lg border border-amber-300/60 bg-amber-50/80 px-3 py-2 text-xs leading-5 text-amber-900">
                            {browserState.error}
                          </div>
                        ) : null}

                        {browserState ? (
                          browserState.items.length > 0 ? (
                            <div className="max-h-[360px] space-y-2 overflow-y-auto pr-1">
                              {browserState.items.map((item) => {
                                const actionKey = `${source.id}:${item.id}`;
                                const importedDocument =
                                  subtitleStyleDocuments.find(
                                    (document) =>
                                      document.source === "remote" &&
                                      document.sourceRef ===
                                        buildRemoteStyleDocumentSourceRef(
                                          source,
                                          item.id,
                                        ),
                                  );
                                return (
                                  <div
                                    key={actionKey}
                                    className={cn(
                                      "grid gap-3 px-3 py-3 md:grid-cols-[minmax(0,1fr)_auto]",
                                      DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                                    )}
                                  >
                                    <div className="min-w-0">
                                      <div className="flex flex-wrap items-center gap-2">
                                        <div className="truncate text-sm font-medium text-foreground">
                                          {item.name || item.id}
                                        </div>
                                        {item.version ? (
                                          <Badge
                                            variant="outline"
                                            className="text-[10px] tracking-[0.08em]"
                                          >
                                            {t("library.config.subtitleStyles.version")}{" "}
                                            {item.version}
                                          </Badge>
                                        ) : null}
                                        {importedDocument ? (
                                          <Badge
                                            variant="outline"
                                            className="border-emerald-300/60 text-[10px] tracking-[0.08em] text-emerald-800"
                                          >
                                            {t("library.config.subtitleStyles.imported")}
                                          </Badge>
                                        ) : null}
                                      </div>
                                      <div className="mt-1 text-xs leading-5 text-muted-foreground">
                                        {item.description?.trim() ||
                                          item.filePath ||
                                          item.id}
                                      </div>
                                    </div>
                                    <div className="flex items-center justify-end">
                                      <Button
                                        type="button"
                                        variant="outline"
                                        size="compact"
                                        className="gap-2"
                                        disabled={Boolean(
                                          importingRemoteStyleItems[actionKey],
                                        )}
                                        onClick={() =>
                                          handleImportRemoteSubtitleStyleItem(
                                            source,
                                            item,
                                          )
                                        }
                                      >
                                        {importingRemoteStyleItems[
                                          actionKey
                                        ] ? (
                                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                                        ) : (
                                          <Download className="h-3.5 w-3.5" />
                                        )}
                                        {importedDocument
                                          ? t("library.config.subtitleStyles.updateImport")
                                          : t("library.config.subtitleStyles.importToLibrary")}
                                      </Button>
                                    </div>
                                  </div>
                                );
                              })}
                            </div>
                          ) : (
                            <EmptyConfigState
                              title={t("library.config.subtitleStyles.emptyRemoteStyleItems")}
                            />
                          )
                        ) : null}
                      </div>
                    );
                  })
                )}
              </ConfigSectionCard>

              <ConfigSectionCard
                title={t("library.config.subtitleStyles.fontSourcesTitle")}
                description={t("library.config.subtitleStyles.fontSourcesDescription")}
                icon={Type}
                contentClassName="space-y-2"
              >
                {subtitleStyleFontSources.length === 0 ? (
                  <EmptyConfigState
                    title={t("library.config.subtitleStyles.emptyFontSources")}
                  />
                ) : (
                  subtitleStyleFontSources.map((source) => {
                    const syncing = Boolean(syncingFontSources[source.id]);
                    return (
                      <div
                        key={source.id}
                        className={cn(
                          "flex items-center justify-between gap-4 p-4",
                          DASHBOARD_DIALOG_FIELD_SURFACE_CLASS,
                        )}
                      >
                        <div className="min-w-0 flex-1 space-y-1">
                          <div className="truncate text-sm font-semibold text-foreground">
                            {resolveFontSourceDisplayName(source)}
                          </div>
                          <div
                            className={cn(
                              "text-xs leading-5",
                              source.syncStatus === "error" && source.lastError
                                ? "text-destructive"
                                : "text-muted-foreground",
                            )}
                          >
                            {resolveFontSourceSummary(
                              source,
                              syncing,
                              language,
                            )}
                          </div>
                        </div>
                        <div className="flex shrink-0 items-center">
                          <Button
                            type="button"
                            variant="outline"
                            size="compact"
                            className="gap-2"
                            onClick={() =>
                              handleSyncSubtitleStyleFontSource(source)
                            }
                            disabled={syncing}
                          >
                            {syncing ? (
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <RefreshCw className="h-3.5 w-3.5" />
                            )}
                            {syncing
                              ? t("library.config.subtitleStyles.syncingFontSource")
                              : t("library.config.subtitleStyles.syncFontSource")}
                          </Button>
                        </div>
                      </div>
                    );
                  })
                )}
              </ConfigSectionCard>
            </div>
          </ConditionalPanel>
        </div>
      </div>
    );
  };

  const renderVideoExportPresetsSection = () => {
    const normalizedSearch = videoExportPresetSearch.trim().toLowerCase();
    const visibleVideoExportPresets = videoExportPresets.filter((preset) => {
      if (!normalizedSearch) {
        return true;
      }
      const searchable = [
        resolvePresetName(preset, t),
        buildPresetSummary(preset, t),
        preset.container,
        preset.videoCodec ?? "",
        preset.audioCodec ?? "",
        preset.outputType,
      ]
        .join(" ")
        .toLowerCase();
      return searchable.includes(normalizedSearch);
    });
    const presetGroups = [
      {
        id: "video",
        label: t("library.workspace.transcode.tabVideo"),
        items: visibleVideoExportPresets.filter(
          (preset) => preset.outputType !== "audio",
        ),
      },
      {
        id: "audio",
        label: t("library.workspace.transcode.tabAudio"),
        items: visibleVideoExportPresets.filter(
          (preset) => preset.outputType === "audio",
        ),
      },
    ]
      .filter((group) => group.items.length > 0)
      .map((group) => {
        const containerMap = new Map<string, TranscodePreset[]>();
        for (const preset of group.items) {
          const items = containerMap.get(preset.container);
          if (items) {
            items.push(preset);
          } else {
            containerMap.set(preset.container, [preset]);
          }
        }
        return {
          ...group,
          containers: Array.from(containerMap.entries()).map(
            ([container, items]) => ({
              container,
              items,
            }),
          ),
        };
      });
    const draft = effectiveVideoPreset;
    const draftOutputType = draft?.outputType ?? "video";
    const draftContainer =
      draft?.container ?? (draftOutputType === "audio" ? "mp3" : "mp4");
    const containerOptions =
      draftOutputType === "audio"
        ? AUDIO_PRESET_CONTAINER_OPTIONS
        : VIDEO_PRESET_CONTAINER_OPTIONS;
    const supportedVideoCodecs =
      draftOutputType === "video"
        ? getSupportedVideoCodecs(draftContainer)
        : [];
    const supportedAudioCodecs = getSupportedAudioCodecs(draftContainer);
    const resolveCodecOptions = <T extends { value: string; label: string }>(
      baseOptions: readonly T[],
      supported: string[],
      currentValue?: string,
    ) => {
      const filtered = baseOptions.filter((option) =>
        supported.includes(option.value),
      );
      if (
        currentValue &&
        !filtered.some((option) => option.value === currentValue)
      ) {
        return [
          { value: currentValue, label: currentValue.toUpperCase() },
          ...filtered,
        ];
      }
      return filtered;
    };
    const videoCodecOptions = resolveCodecOptions(
      VIDEO_CODEC_OPTIONS,
      supportedVideoCodecs,
      draft?.videoCodec,
    );
    const baseAudioCodecOptions: ReadonlyArray<{
      value: string;
      label: string;
    }> =
      draftOutputType === "audio"
        ? AUDIO_CODEC_OPTIONS
        : VIDEO_AUDIO_CODEC_OPTIONS;
    const audioCodecOptions = resolveCodecOptions(
      baseAudioCodecOptions,
      supportedAudioCodecs,
      draft?.audioCodec,
    );
    const canSaveVideoPreset =
      Boolean(videoExportPresetDraft?.name?.trim()) &&
      !saveTranscodePreset.isPending;
    const disablePresetSelection = selectedVideoPresetEditing;

    return (
      <ConfigMasterDetailLayout
        sidebar={
          <div
            className={cn(
              "flex h-full min-h-0 flex-col overflow-hidden p-4",
              DASHBOARD_DIALOG_SOFT_SURFACE_CLASS,
            )}
          >
            {videoExportPresets.length > 0 ? (
              <div className="mb-3 flex items-center gap-2 rounded-lg border border-border/60 bg-background/75 px-2.5 py-2">
                <Search className="h-3.5 w-3.5 text-muted-foreground" />
                <Input
                  value={videoExportPresetSearch}
                  onChange={(event) =>
                    setVideoExportPresetSearch(event.target.value)
                  }
                  placeholder={t("library.config.videoExportPresets.searchPlaceholder")}
                  className="h-7 border-none bg-transparent px-0 text-xs shadow-none focus-visible:ring-0"
                />
                <Badge
                  variant="outline"
                  className="h-5 shrink-0 px-1.5 text-[10px]"
                >
                  {visibleVideoExportPresets.length}
                </Badge>
              </div>
            ) : null}
            {videoExportPresets.length === 0 ? (
              <EmptyConfigState
                title={t("library.config.videoExportPresets.empty")}
              />
            ) : visibleVideoExportPresets.length === 0 ? (
              <EmptyConfigState
                title={t("library.config.videoExportPresets.searchEmpty")}
              />
            ) : (
              <div className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
                {presetGroups.map((group) => (
                  <div key={group.id} className="space-y-2">
                    <div className="flex items-center justify-between px-1">
                      <div className="text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
                        {group.label}
                      </div>
                      <div className="text-[10px] text-muted-foreground">
                        {group.items.length}
                      </div>
                    </div>
                    {group.containers.map((containerGroup) => (
                      <div
                        key={`${group.id}-${containerGroup.container}`}
                        className="space-y-1.5"
                      >
                        <div className="px-1 text-[10px] font-medium uppercase tracking-[0.12em] text-muted-foreground/80">
                          {containerGroup.container.toUpperCase()}
                        </div>
                        {containerGroup.items.map((preset) => {
                          const isSelected =
                            videoExportPresetDraftMode === "create"
                              ? false
                              : preset.id === selectedVideoPreset?.id;
                          const presetOriginLabel = preset.isBuiltin
                            ? t("library.config.translateLanguages.builtinBadge")
                            : t("library.config.translateLanguages.customBadge");
                          return (
                            <ConfigNavigationItem
                              key={preset.id}
                              title={resolvePresetName(preset, t)}
                              description={`${resolveCompactPresetMeta(preset, t)} · ${presetOriginLabel}`}
                              selected={isSelected}
                              disabled={disablePresetSelection && !isSelected}
                              compact
                              onClick={() => {
                                if (disablePresetSelection && !isSelected) {
                                  return;
                                }
                                setSelectedVideoExportPresetId(preset.id);
                              }}
                            />
                          );
                        })}
                      </div>
                    ))}
                  </div>
                ))}
              </div>
            )}
          </div>
        }
        detail={
          draft ? (
            <ConfigDetailPanel
              title={
                selectedVideoPresetEditing ? (
                  <Input
                    value={videoExportPresetDraft?.name ?? ""}
                    onChange={(event) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? { ...current, name: event.target.value }
                          : current,
                      )
                    }
                    placeholder={t("library.config.videoExportPresets.name")}
                    className="h-8 max-w-full border-border/70 bg-background/80 text-sm font-semibold"
                  />
                ) : (
                  resolvePresetName(draft as TranscodePreset, t)
                )
              }
              actions={
                <div className="ml-auto flex flex-wrap items-center justify-end gap-2">
                  {selectedVideoPreset &&
                  videoExportPresetDraftMode !== "create" ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="compact"
                      className="gap-2"
                      onClick={() =>
                        handleDuplicateVideoExportPreset(selectedVideoPreset)
                      }
                    >
                      <Copy className="h-3.5 w-3.5" />
                      {t("library.config.subtitleStyles.duplicate")}
                    </Button>
                  ) : null}
                  {selectedVideoPresetEditing ? (
                    <>
                      <Button
                        type="button"
                        variant="outline"
                        size="compact"
                        className="gap-2"
                        onClick={() => void handleSaveVideoExportPreset()}
                        disabled={!canSaveVideoPreset}
                      >
                        {saveTranscodePreset.isPending ? (
                          <Loader2 className="h-3.5 w-3.5 animate-spin" />
                        ) : (
                          <Check className="h-3.5 w-3.5" />
                        )}
                        {t("common.save")}
                      </Button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="compact"
                        className="gap-2"
                        onClick={handleCancelVideoExportPresetEdit}
                      >
                        <X className="h-3.5 w-3.5" />
                        {t("common.cancel")}
                      </Button>
                    </>
                  ) : selectedVideoPreset && !selectedVideoPresetReadOnly ? (
                    <Button
                      type="button"
                      variant="outline"
                      size="compact"
                      className="gap-2"
                      onClick={() =>
                        handleEditVideoExportPreset(selectedVideoPreset)
                      }
                    >
                      <PencilLine className="h-3.5 w-3.5" />
                      {t("common.edit")}
                    </Button>
                  ) : null}
                </div>
              }
              badges={
                <>
                  <Badge
                    variant="outline"
                    className="text-[10px] tracking-[0.08em]"
                  >
                    {resolveTranscodeOutputTypeLabel(draftOutputType, t)}
                  </Badge>
                  <Badge
                    variant="outline"
                    className="text-[10px] tracking-[0.08em]"
                  >
                    {draftContainer.toUpperCase()}
                  </Badge>
                  <Badge
                    variant="outline"
                    className="text-[10px] tracking-[0.08em]"
                  >
                    {selectedVideoPresetReadOnly
                      ? t("library.config.translateLanguages.builtinBadge")
                      : t("library.config.translateLanguages.customBadge")}
                  </Badge>
                </>
              }
              footer={
                selectedVideoPreset &&
                !selectedVideoPresetReadOnly &&
                videoExportPresetDraftMode !== "create" ? (
                  <div className="flex justify-center pt-2">
                    <Button
                      type="button"
                      variant="destructive"
                      size="compact"
                      className="min-w-[180px]"
                      onClick={() =>
                        void handleDeleteVideoExportPreset(selectedVideoPreset)
                      }
                      disabled={deleteTranscodePreset.isPending}
                    >
                      {t("common.delete")}
                    </Button>
                  </div>
                ) : undefined
              }
              contentClassName="space-y-3"
            >
              <ConfigSelectField
                label={t("library.config.videoExportPresets.outputType")}
                inline
                value={draftOutputType}
                disabled={!selectedVideoPresetEditing}
                onChange={(nextValue) =>
                  setVideoExportPresetDraft((current) => {
                    if (!current) {
                      return current;
                    }
                    const nextOutputType =
                      nextValue as TranscodePreset["outputType"];
                    return nextOutputType === "audio"
                      ? {
                          ...DEFAULT_AUDIO_EXPORT_PRESET_DRAFT,
                          id: current.id,
                          name: current.name,
                        }
                      : {
                          ...DEFAULT_VIDEO_EXPORT_PRESET_DRAFT,
                          id: current.id,
                          name: current.name,
                        };
                  })
                }
                options={[
                  {
                    value: "video",
                    label: t("library.workspace.transcode.outputVideo"),
                  },
                  {
                    value: "audio",
                    label: t("library.workspace.transcode.outputAudio"),
                  },
                ]}
              />

              <ConfigSelectField
                label={t("library.config.videoExportPresets.container")}
                inline
                value={draftContainer}
                disabled={!selectedVideoPresetEditing}
                onChange={(nextValue) =>
                  setVideoExportPresetDraft((current) => {
                    if (!current) {
                      return current;
                    }
                    if (current.outputType === "audio") {
                      const nextAudioCodec =
                        resolveDefaultAudioCodecForContainer(nextValue);
                      return {
                        ...current,
                        container: nextValue,
                        audioCodec: nextAudioCodec,
                        audioBitrateKbps:
                          resolveRecommendedAudioBitrateKbps(nextAudioCodec),
                      };
                    }
                    const nextVideoCodec =
                      getSupportedVideoCodecs(nextValue)[0] ?? "h264";
                    const nextAudioCodec =
                      getSupportedAudioCodecs(nextValue)[0] ?? "aac";
                    const resolvedAudioCodec =
                      getSupportedAudioCodecs(nextValue).includes(
                        current.audioCodec ?? "",
                      )
                        ? current.audioCodec
                        : nextAudioCodec;
                    return {
                      ...current,
                      container: nextValue,
                      videoCodec: getSupportedVideoCodecs(nextValue).includes(
                        current.videoCodec ?? "",
                      )
                        ? current.videoCodec
                        : nextVideoCodec,
                      audioCodec: resolvedAudioCodec,
                      audioBitrateKbps:
                        resolveRecommendedAudioBitrateKbps(resolvedAudioCodec),
                    };
                  })
                }
                options={containerOptions.map((option) => ({
                  value: option.value,
                  label: option.label,
                }))}
              />

              {draftOutputType === "video" ? (
                <>
                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.videoCodec")}
                    inline
                    value={draft.videoCodec ?? "h264"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              videoCodec: nextValue,
                              crf: resolveDefaultVideoCRF(nextValue),
                            }
                          : current,
                      )
                    }
                    options={videoCodecOptions.map((option) => ({
                      value: option.value,
                      label: option.label,
                    }))}
                  />

                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.audioCodec")}
                    inline
                    value={draft.audioCodec ?? "aac"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              audioCodec: nextValue,
                              audioBitrateKbps:
                                resolveRecommendedAudioBitrateKbps(nextValue),
                            }
                          : current,
                      )
                    }
                    options={audioCodecOptions.map((option) => ({
                      value: option.value,
                      label: option.label,
                    }))}
                  />

                  {draft.audioCodec !== "copy" ? (
                    <ConfigNumberField
                      label={t("library.config.videoExportPresets.audioBitrate")}
                      inline
                      description={t("library.config.videoExportPresets.audioBitrateDescription")}
                      value={
                        draft.audioBitrateKbps ??
                        resolveRecommendedAudioBitrateKbps(draft.audioCodec) ??
                        256
                      }
                      min={1}
                      disabled={!selectedVideoPresetEditing}
                      onChange={(nextValue) =>
                        setVideoExportPresetDraft((current) =>
                          current
                            ? { ...current, audioBitrateKbps: nextValue }
                            : current,
                        )
                      }
                    />
                  ) : null}

                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.qualityMode")}
                    inline
                    value={draft.qualityMode ?? "crf"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              qualityMode:
                                nextValue as TranscodePreset["qualityMode"],
                            }
                          : current,
                      )
                    }
                    options={[
                      { value: "crf", label: "CRF" },
                      {
                        value: "bitrate",
                        label: t("library.workspace.transcode.bitrate"),
                      },
                    ]}
                  />

                  <ConfigNumberField
                    label={
                      draft.qualityMode === "bitrate"
                        ? t("library.config.videoExportPresets.videoBitrate")
                        : t("library.config.videoExportPresets.crf")
                    }
                    description={
                      draft.qualityMode === "bitrate"
                        ? t("library.config.videoExportPresets.videoBitrateDescription")
                        : t("library.config.videoExportPresets.crfDescription")
                    }
                    inline
                    value={
                      draft.qualityMode === "bitrate"
                        ? (draft.bitrateKbps ?? 6000)
                        : (draft.crf ?? resolveDefaultVideoCRF(draft.videoCodec))
                    }
                    min={1}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? current.qualityMode === "bitrate"
                            ? { ...current, bitrateKbps: nextValue }
                            : { ...current, crf: nextValue }
                          : current,
                      )
                    }
                  />

                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.scale")}
                    inline
                    value={draft.scale ?? "original"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              scale: nextValue as TranscodePreset["scale"],
                            }
                          : current,
                      )
                    }
                    options={VIDEO_SCALE_OPTIONS.map((option) => ({
                      value: option.value,
                      label: option.label,
                    }))}
                  />

                  {draft.scale === "custom" ? (
                    <div className="grid gap-3 md:grid-cols-2">
                      <ConfigNumberField
                        label={t("library.workspace.fields.width")}
                        description={t("library.config.videoExportPresets.customWidthDescription")}
                        value={draft.width ?? 1920}
                        min={1}
                        disabled={!selectedVideoPresetEditing}
                        onChange={(nextValue) =>
                          setVideoExportPresetDraft((current) =>
                            current
                              ? { ...current, width: nextValue }
                              : current,
                          )
                        }
                      />
                      <ConfigNumberField
                        label={t("library.workspace.fields.height")}
                        description={t("library.config.videoExportPresets.customHeightDescription")}
                        value={draft.height ?? 1080}
                        min={1}
                        disabled={!selectedVideoPresetEditing}
                        onChange={(nextValue) =>
                          setVideoExportPresetDraft((current) =>
                            current
                              ? { ...current, height: nextValue }
                              : current,
                          )
                        }
                      />
                    </div>
                  ) : null}

                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.ffmpegPreset")}
                    inline
                    value={draft.ffmpegPreset ?? "slow"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              ffmpegPreset:
                                nextValue as TranscodePreset["ffmpegPreset"],
                            }
                          : current,
                      )
                    }
                    options={FFMPEG_SPEED_PRESET_OPTIONS.map((option) => ({
                      value: option.value,
                      label: option.label,
                    }))}
                  />

                  <ConfigSwitchField
                    label={t("library.config.videoExportPresets.allowUpscale")}
                    description={t("library.config.videoExportPresets.allowUpscaleDescription")}
                    checked={Boolean(draft.allowUpscale)}
                    disabled={!selectedVideoPresetEditing}
                    onCheckedChange={(checked) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? { ...current, allowUpscale: checked }
                          : current,
                      )
                    }
                  />
                </>
              ) : (
                <>
                  <ConfigSelectField
                    label={t("library.config.videoExportPresets.audioCodec")}
                    inline
                    value={draft.audioCodec ?? "mp3"}
                    disabled={!selectedVideoPresetEditing}
                    onChange={(nextValue) =>
                      setVideoExportPresetDraft((current) =>
                        current
                          ? {
                              ...current,
                              audioCodec: nextValue,
                              audioBitrateKbps:
                                resolveRecommendedAudioBitrateKbps(nextValue),
                            }
                          : current,
                      )
                    }
                    options={audioCodecOptions.map((option) => ({
                      value: option.value,
                      label: option.label,
                    }))}
                  />

                  {resolveRecommendedAudioBitrateKbps(draft.audioCodec) ? (
                    <ConfigNumberField
                      label={t("library.config.videoExportPresets.audioBitrate")}
                      inline
                      description={t("library.config.videoExportPresets.audioBitrateDescription")}
                      value={
                        draft.audioBitrateKbps ??
                        resolveRecommendedAudioBitrateKbps(draft.audioCodec) ??
                        320
                      }
                      min={1}
                      disabled={!selectedVideoPresetEditing}
                      onChange={(nextValue) =>
                        setVideoExportPresetDraft((current) =>
                          current
                            ? { ...current, audioBitrateKbps: nextValue }
                            : current,
                        )
                      }
                    />
                  ) : null}
                </>
              )}

            </ConfigDetailPanel>
          ) : (
            <ConfigStandardEmptyState
              icon={FileText}
              title={t("library.config.videoExportPresets.emptyDetailTitle")}
              description={t("library.config.videoExportPresets.emptyDetailDescription")}
            />
          )
        }
      />
    );
  };

  const renderTaskRuntimeContent = () => {
    const translateTaskRuntimeCardID = buildEditableCardID(
      "task-runtime",
      "translate",
    );
    const proofreadTaskRuntimeCardID = buildEditableCardID(
      "task-runtime",
      "proofread",
    );
    const translateTaskRuntimeEditing = isCardEditing(
      translateTaskRuntimeCardID,
    );
    const proofreadTaskRuntimeEditing = isCardEditing(
      proofreadTaskRuntimeCardID,
    );

    return (
      <div className="h-full min-h-0">
        <ConfigMasterDetailLayout
          sidebar={
            <ConfigNavigationSidebar
              showSearch
              searchValue={taskRuntimeSearch}
              onSearchChange={setTaskRuntimeSearch}
              searchPlaceholder={t("library.config.taskRuntime.searchPlaceholder")}
              count={visibleTaskRuntimeItems.length}
              emptyState={
                <EmptyConfigState
                  title={t("library.config.taskRuntime.searchEmpty")}
                />
              }
              bodyClassName="space-y-2"
            >
              {visibleTaskRuntimeItems.map((item) => (
                <ConfigNavigationItem
                  key={item.id}
                  title={item.title}
                  description={item.description}
                  selected={activeTaskRuntimeTask === item.id}
                  disabled={
                    item.id === "translate"
                      ? proofreadTaskRuntimeEditing
                      : translateTaskRuntimeEditing
                  }
                  compact
                  onClick={() => setActiveTaskRuntimeTask(item.id)}
                />
              ))}
            </ConfigNavigationSidebar>
          }
          detail={
            activeTaskRuntimeTask === "translate" ? (
              <ConfigDetailPanel title={t("library.config.taskRuntime.translateTitle")}>
                <TaskRuntimeFields
                  value={value.taskRuntime.translate}
                  disabled={!translateTaskRuntimeEditing}
                  onChange={(nextValue) =>
                    updateTaskRuntime({
                      translate: nextValue,
                    })
                  }
                />
              </ConfigDetailPanel>
            ) : (
              <ConfigDetailPanel title={t("library.config.taskRuntime.proofreadTitle")}>
                <TaskRuntimeFields
                  value={value.taskRuntime.proofread}
                  disabled={!proofreadTaskRuntimeEditing}
                  onChange={(nextValue) =>
                    updateTaskRuntime({
                      proofread: nextValue,
                    })
                  }
                />
              </ConfigDetailPanel>
            )
          }
        />
      </div>
    );
  };

  const renderSection = () => {
    switch (activePage) {
      case "overview":
        return renderOverviewPage();
      case "task-runtime":
        return renderTaskRuntimeContent();
      case "languages":
        return renderLanguageAssetsSection("languages");
      case "glossary":
        return renderLanguageAssetsSection("glossary");
      case "prompts":
        return renderLanguageAssetsSection("prompts");
      case "subtitle-styles":
        return (
          <SubtitleStylePresetManager
            value={value}
            onChange={onChange}
            onRequestPersist={onRequestPersist}
            onToolbarActionsChange={setSubtitleStyleToolbarActions}
          />
        );
      case "font-management":
        return renderSubtitleStylesSection("fonts");
      case "subtitle-export-presets":
        return renderSubtitleStylesSection("profiles");
      case "video-export-presets":
        return renderVideoExportPresetsSection();
      case "remote-sources":
        return renderSubtitleStylesSection("sources");
      default:
        return null;
    }
  };
  const isContainedSection =
    activePage === "task-runtime" ||
    activePage === "languages" ||
    activePage === "glossary" ||
    activePage === "prompts" ||
    activePage === "subtitle-styles" ||
    activePage === "subtitle-export-presets" ||
    activePage === "video-export-presets";

  return (
    <div className="grid h-full min-h-0 gap-4 overflow-hidden xl:grid-cols-[220px_minmax(0,1fr)]">
      <aside
        className={cn(
          DASHBOARD_PANEL_CARD_CLASS,
          "min-h-0 overflow-hidden p-2 shadow-sm",
        )}
      >
        <div className="min-h-0 h-full overflow-auto">
          <div className="space-y-1">
            {configPages.map((page) => {
              const Icon = page.icon;
              const isActive = page.id === activePage;
              return (
                <button
                  key={page.id}
                  type="button"
                  onClick={() => setActivePage(page.id)}
                  className={cn(
                    "flex w-full items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-colors",
                    isActive
                      ? "border-border/70 bg-background/85 text-foreground shadow-sm"
                      : "border-transparent text-muted-foreground hover:border-border/60 hover:bg-background/60 hover:text-foreground",
                  )}
                >
                  <div
                    className={cn(
                      "rounded-md p-1.5",
                      isActive
                        ? "bg-card text-foreground"
                        : "bg-muted/70 text-muted-foreground",
                    )}
                  >
                    <Icon className="h-4 w-4" />
                  </div>
                  <div className="min-w-0 flex flex-1 items-center gap-2">
                    <span className="truncate text-sm font-medium">
                      {page.label}
                    </span>
                    {page.badge ? (
                      <Badge
                        variant="outline"
                        className="ml-auto h-5 px-1.5 text-xs"
                      >
                        {page.badge}
                      </Badge>
                    ) : null}
                  </div>
                </button>
              );
            })}
          </div>
        </div>
      </aside>

      <section
        className={cn(
          "h-full min-h-0 min-w-0 overflow-x-hidden",
          isContainedSection
            ? "flex flex-col overflow-y-hidden"
            : "overflow-y-auto pr-1",
        )}
        onBlurCapture={(event) => {
          if (
            event.relatedTarget instanceof Node &&
            event.currentTarget.contains(event.relatedTarget)
          ) {
            return;
          }
          onRequestPersist?.();
        }}
      >
        {renderSection()}
      </section>
    </div>
  );
}
