import * as React from "react";
import { Dialogs, Events } from "@wailsio/runtime";

import { FolderOpen, Loader2 } from "lucide-react";

import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from "@/shared/ui/empty";
import { useI18n } from "@/shared/i18n";
import { messageBus } from "@/shared/message";
import {
  useApplySubtitleReviewSession,
  useCreateTranscodeJob,
  useCreateSubtitleProofreadJob,
  useCreateSubtitleTranslateJob,
  useDiscardSubtitleReviewSession,
  useExportSubtitle,
  useGenerateSubtitleStylePreviewASS,
  useGenerateWorkspacePreviewVTT,
  useGetSubtitleReviewSession,
  useGetWorkspaceProject,
  useGetWorkspaceState,
  useOpenFileLocation,
  useParseSubtitle,
  useRestoreSubtitleOriginal,
  useSaveWorkspaceState,
  useSaveSubtitle,
  useTranscodePresetsForDownload,
  useUpdateLibraryModuleConfig,
} from "@/shared/query/library";
import { useAssistants } from "@/shared/query/assistant";
import {
  useEnabledProvidersWithModels,
  useProviderSecret,
} from "@/shared/query/providers";
import type {
  LibraryDTO,
  LibraryBilingualStyleDTO,
  LibraryFileDTO,
  LibraryMonoStyleDTO,
  LibraryModuleConfigDTO,
  SubtitleReviewCueDecisionDTO,
  SubtitleDocument,
  SubtitleExportConfig,
  WorkspaceSubtitleTrackDTO,
  WorkspaceVideoTrackDTO,
  TranscodePreset,
} from "@/shared/contracts/library";

import {
  useLibraryWorkspaceStore,
  type LibraryWorkspaceEditor,
} from "../model/workspaceStore";
import {
  resolveWorkspaceQaCheckDefinitions,
  type WorkspaceQaCheckId,
} from "../model/workspaceQa";
import {
  buildTextSubtitleContent,
  normalizeSubtitleExportFormat,
  resolveDefaultBilingualStyle,
  resolveDefaultMonoStyle,
  resolveSubtitleExportPresets,
  resolveSubtitleStyleDefaults,
  resolveVideoExportSoftSubtitleRoute,
} from "../utils/subtitleStyles";
import { createSubtitleStylePresetID } from "../utils/subtitleStylePresets";
import {
  buildDefaultSubtitleExportConfig,
  buildWorkspaceTaskProgressLabel,
  buildWorkspaceTranslateTaskLabel,
  createWorkspaceLingualStyleDraft,
  createWorkspaceMonoStyleDraft,
  createWorkspacePromptProfileId,
  derivePromptProfileName,
  mergeSubtitleExportConfig,
  parseAssistantModelRef,
  parseLibraryWorkspacePersistedState,
  resolveDirectoryName,
  resolveErrorMessage,
  resolveLibraryWorkspacePersistedState,
  resolveSubtitleExportMediaHint,
  resolveSubtitleExportPresetOverrideConfig,
  resolveSubtitleExportPresetFormat,
  resolveSubtitleExportPresetSelection,
  rowsEqual,
  stripFileExtension,
  toggleSelectedId,
  trackSupportsDualDisplay,
} from "./library-workspace-utils";
import { SubtitleEditorLayout } from "./workspace/SubtitleEditorLayout";
import { formatBytes } from "../utils/format";
import {
  resolveGlossaryConstraintOptions,
  resolveProofreadPromptConstraintOptions,
  resolveTranslateLanguageOptions,
  resolveTranslatePromptConstraintOptions,
} from "./workspace/languageTask";
import { WorkspaceActionDialogs } from "./workspace/WorkspaceActionDialogs";
import { WorkspaceExportSubtitleDialog } from "./workspace/WorkspaceExportSubtitleDialog";
import { WorkspaceExportVideoDialog } from "./workspace/WorkspaceExportVideoDialog";
import { VideoEditorLayout } from "./workspace/VideoEditorLayout";
import { WorkspaceHeader } from "./workspace/WorkspaceHeader";
import { WorkspaceSubtitleStyleCard } from "./workspace/WorkspaceSubtitleStyleCard";
import type {
  WorkspaceDensity,
  WorkspaceFilter,
  WorkspaceGuidelineProfileId,
  WorkspaceImportNormalizationOptions,
  WorkspaceLanguageTaskMode,
  WorkspaceProofreadOptions,
  WorkspaceQaFilter,
  WorkspaceReviewDecision,
  WorkspaceSelectOption,
  WorkspaceSubtitleRow,
} from "./workspace/types";
import { resolveGatewayCoreReadiness } from "@/features/setup/readiness";
import {
  buildAssetPreviewUrl,
  buildSubtitleDocument,
  buildSubtitleRows,
  clampMs,
  filterWorkspaceRows,
  formatFrameRate,
  formatMediaDuration,
  formatResolution,
  isWorkspaceVideoFile,
  normalizeFileKind,
  resolveActiveFile,
  resolveCurrentRow,
  resolveFileFormat,
  resolveLibraryMediaMimeType,
  resolveResourceName,
  resolveVideoVersionOptions,
  resolveSubtitleTrackOptions,
  resolveWorkspaceProjectSubtitleTrackOptions,
  resolveWorkspaceProjectVideoOptions,
  resolveWorkspaceGuidelineLabel,
  resolveWorkspaceGuidelineOptions,
  resolveWorkspaceQaSummary,
  resolveWorkspaceDurationMs,
  resolveWorkspaceWaveformGuardKind,
  resolveWorkspaceRows,
} from "./workspace/utils";

type WorkspacePageProps = {
  library?: LibraryDTO;
  moduleConfig?: LibraryModuleConfigDTO;
  onModuleConfigChange?: (next: LibraryModuleConfigDTO) => void;
  files: LibraryFileDTO[];
  httpBaseURL: string;
  onRequestImportVideo: () => void;
  onRequestImportSubtitle: () => void;
  onToolbarStateChange?: (state: LibraryWorkspaceToolbarState | null) => void;
};

export type LibraryWorkspaceToolbarState = {
  activeEditor: LibraryWorkspaceEditor;
  canExportSubtitle: boolean;
  canExportVideo: boolean;
  exportDisabledReason?: string;
  canOpenCurrentFile: boolean;
  onExportSubtitle: () => void;
  onExportVideo: () => void;
  onOpenCurrentFile: () => void;
};

export function LibraryWorkspacePage(props: WorkspacePageProps) {
  const { t, language } = useI18n();
  const library = props.library;
  const fastReadLatestStateEnabled =
    props.moduleConfig?.workspace.fastReadLatestState !== false;
  const activeEditor = useLibraryWorkspaceStore((state) => state.activeEditor);
  const activeVideoFileId = useLibraryWorkspaceStore(
    (state) => state.activeVideoFileId,
  );
  const activeSubtitleFileId = useLibraryWorkspaceStore(
    (state) => state.activeSubtitleFileId,
  );
  const displayMode = useLibraryWorkspaceStore((state) => state.displayMode);
  const comparisonSubtitleFileId = useLibraryWorkspaceStore(
    (state) => state.comparisonSubtitleFileId,
  );
  const guidelineProfileId = useLibraryWorkspaceStore(
    (state) => state.guidelineProfileId,
  );
  const qaCheckSettings = useLibraryWorkspaceStore(
    (state) => state.qaCheckSettings,
  );
  const applyPersistedWorkspaceState = useLibraryWorkspaceStore(
    (state) => state.applyPersistedState,
  );
  const setActiveEditor = useLibraryWorkspaceStore(
    (state) => state.setActiveEditor,
  );
  const setActiveVideoFileId = useLibraryWorkspaceStore(
    (state) => state.setActiveVideoFileId,
  );
  const setActiveSubtitleFileId = useLibraryWorkspaceStore(
    (state) => state.setActiveSubtitleFileId,
  );
  const setDisplayMode = useLibraryWorkspaceStore(
    (state) => state.setDisplayMode,
  );
  const setComparisonSubtitleFileId = useLibraryWorkspaceStore(
    (state) => state.setComparisonSubtitleFileId,
  );
  const setGuidelineProfileId = useLibraryWorkspaceStore(
    (state) => state.setGuidelineProfileId,
  );
  const setQaCheckEnabled = useLibraryWorkspaceStore(
    (state) => state.setQaCheckEnabled,
  );

  const createTranscode = useCreateTranscodeJob();
  const parseSubtitle = useParseSubtitle();
  const exportSubtitle = useExportSubtitle();
  const generateSubtitleStylePreviewASS = useGenerateSubtitleStylePreviewASS();
  const generateWorkspacePreviewVTT = useGenerateWorkspacePreviewVTT();
  const workspaceProjectQuery = useGetWorkspaceProject(
    library?.id ?? "",
    Boolean(library?.id),
  );
  const workspaceStateQuery = useGetWorkspaceState(
    library?.id ?? "",
    Boolean(library?.id) && fastReadLatestStateEnabled,
  );
  const saveWorkspaceState = useSaveWorkspaceState();
  const saveSubtitle = useSaveSubtitle();
  const restoreSubtitleOriginal = useRestoreSubtitleOriginal();
  const openFileLocation = useOpenFileLocation();
  const createSubtitleProofread = useCreateSubtitleProofreadJob();
  const createSubtitleTranslate = useCreateSubtitleTranslateJob();
  const applySubtitleReview = useApplySubtitleReviewSession();
  const discardSubtitleReview = useDiscardSubtitleReviewSession();
  const updateModuleConfig = useUpdateLibraryModuleConfig();
  const assistantsQuery = useAssistants(true);
  const providersWithModelsQuery = useEnabledProvidersWithModels();
  const transcodePresetsQuery = useTranscodePresetsForDownload({
    mediaType: "video",
  });

  const workspaceProject = workspaceProjectQuery.data;
  const workspaceVideoTracks = React.useMemo<WorkspaceVideoTrackDTO[]>(
    () => workspaceProject?.videoTracks ?? [],
    [workspaceProject?.videoTracks],
  );
  const workspaceSubtitleTracks = React.useMemo<WorkspaceSubtitleTrackDTO[]>(
    () => workspaceProject?.subtitleTracks ?? [],
    [workspaceProject?.subtitleTracks],
  );
  const workspaceProjectMonoStyle = React.useMemo(
    () => createWorkspaceMonoStyleDraft(workspaceProject?.subtitleMonoStyle),
    [workspaceProject?.subtitleMonoStyle],
  );
  const workspaceProjectLingualStyle = React.useMemo(
    () =>
      createWorkspaceLingualStyleDraft(workspaceProject?.subtitleLingualStyle),
    [workspaceProject?.subtitleLingualStyle],
  );
  const videoFiles = React.useMemo(
    () =>
      workspaceVideoTracks.length > 0
        ? workspaceVideoTracks.map((track) => track.file)
        : props.files.filter(isWorkspaceVideoFile),
    [props.files, workspaceVideoTracks],
  );
  const subtitleFiles = React.useMemo(
    () =>
      workspaceSubtitleTracks.length > 0
        ? workspaceSubtitleTracks.map((track) => track.file)
        : props.files.filter((file) => normalizeFileKind(file.kind) == "subtitle"),
    [props.files, workspaceSubtitleTracks],
  );

  const activeVideoFile = React.useMemo(
    () => resolveActiveFile(videoFiles, activeVideoFileId),
    [activeVideoFileId, videoFiles],
  );
  const activeSubtitleFile = React.useMemo(
    () => resolveActiveFile(subtitleFiles, activeSubtitleFileId),
    [activeSubtitleFileId, subtitleFiles],
  );
  const activeSubtitleTrackState = React.useMemo(
    () =>
      workspaceSubtitleTracks.find(
        (track) => track.trackId === activeSubtitleFile?.id,
      ) ??
      workspaceSubtitleTracks[0] ??
      null,
    [activeSubtitleFile?.id, workspaceSubtitleTracks],
  );
  const pendingReview = activeSubtitleTrackState?.pendingReview ?? null;
  const reviewSessionQuery = useGetSubtitleReviewSession(
    pendingReview?.sessionId ?? "",
    Boolean(pendingReview?.sessionId),
  );
  const reviewSession = reviewSessionQuery.data ?? null;
  const actionablePendingReview = React.useMemo(
    () =>
      Boolean(pendingReview?.sessionId) &&
      ((pendingReview?.changedCueCount ?? 0) > 0 ||
        (reviewSession?.suggestions?.length ?? 0) > 0),
    [
      pendingReview?.changedCueCount,
      pendingReview?.sessionId,
      reviewSession?.suggestions?.length,
    ],
  );

  const [videoSearch, setVideoSearch] = React.useState("");
  const [videoFilter] = React.useState<WorkspaceFilter>("all");
  const [videoQaFilter, setVideoQaFilter] =
    React.useState<WorkspaceQaFilter>("all");
  const [videoDensity, setVideoDensity] =
    React.useState<WorkspaceDensity>("comfortable");
  const [autoFollow, setAutoFollow] = React.useState(true);
  const [subtitleStyleSidebarOpen, setSubtitleStyleSidebarOpen] =
    React.useState(false);
  const [workspaceSubtitleMonoStyle, setWorkspaceSubtitleMonoStyle] =
    React.useState<LibraryMonoStyleDTO | null>(null);
  const [workspaceSubtitleLingualStyle, setWorkspaceSubtitleLingualStyle] =
    React.useState<LibraryBilingualStyleDTO | null>(null);
  const [previewVttContent, setPreviewVttContent] = React.useState("");
  const [subtitleSearch, setSubtitleSearch] = React.useState("");
  const [subtitleReplaceValue, setSubtitleReplaceValue] = React.useState("");
  const [subtitleFilter, setSubtitleFilter] =
    React.useState<WorkspaceFilter>("all");
  const [subtitleQaFilter, setSubtitleQaFilter] =
    React.useState<WorkspaceQaFilter>("all");
  const [subtitleDensity, setSubtitleDensity] =
    React.useState<WorkspaceDensity>("compact");
  const [subtitleImportDialogOpen, setSubtitleImportDialogOpen] =
    React.useState(false);
  const [languageTaskDialogOpen, setLanguageTaskDialogOpen] =
    React.useState(false);
  const [exportSubtitleDialogOpen, setExportSubtitleDialogOpen] =
    React.useState(false);
  const [exportVideoDialogOpen, setExportVideoDialogOpen] =
    React.useState(false);
  const [languageTaskMode, setLanguageTaskMode] =
    React.useState<WorkspaceLanguageTaskMode>("translate");
  const [useCurrentGuidelineForImport, setUseCurrentGuidelineForImport] =
    React.useState(true);
  const [importGuidelineProfileId, setImportGuidelineProfileId] =
    React.useState<WorkspaceGuidelineProfileId>("netflix");
  const [normalizationOptions, setNormalizationOptions] =
    React.useState<WorkspaceImportNormalizationOptions>({
      normalizeLineBreaks: true,
      trimWhitespace: true,
      removeBlankLines: true,
      repairEncoding: true,
    });
  const [proofreadOptions, setProofreadOptions] =
    React.useState<WorkspaceProofreadOptions>({
      spelling: true,
      punctuation: true,
      terminology: false,
    });
  const [translateTargetLanguage, setTranslateTargetLanguage] =
    React.useState("");
  const [translateGlossaryProfileIds, setTranslateGlossaryProfileIds] =
    React.useState<string[]>([]);
  const [translateReferenceTrackId, setTranslateReferenceTrackId] =
    React.useState("");
  const [translatePromptProfileIds, setTranslatePromptProfileIds] =
    React.useState<string[]>([]);
  const [translateInlinePrompt, setTranslateInlinePrompt] = React.useState("");
  const [proofreadGlossaryProfileIds, setProofreadGlossaryProfileIds] =
    React.useState<string[]>([]);
  const [proofreadPromptProfileIds, setProofreadPromptProfileIds] =
    React.useState<string[]>([]);
  const [proofreadInlinePrompt, setProofreadInlinePrompt] = React.useState("");
  const [translatePromptProfileName, setTranslatePromptProfileName] =
    React.useState("");
  const [proofreadPromptProfileName, setProofreadPromptProfileName] =
    React.useState("");
  const [pendingTranslateOperationId, setPendingTranslateOperationId] =
    React.useState("");
  const [exportVideoPresetId, setExportVideoPresetId] = React.useState("");
  const [exportSubtitlePresetId, setExportSubtitlePresetId] =
    React.useState("");
  const [exportSubtitleFormat, setExportSubtitleFormat] = React.useState("srt");
  const [exportSubtitleConfig, setExportSubtitleConfig] =
    React.useState<SubtitleExportConfig>(() =>
      buildDefaultSubtitleExportConfig("", {
        width: 1920,
        height: 1080,
        frameRate: 30,
        source: "default",
      }),
    );
  const [exportVideoSubtitleHandling, setExportVideoSubtitleHandling] =
    React.useState<"none" | "embed" | "burnin">("embed");
  const [rows, setRows] = React.useState<WorkspaceSubtitleRow[]>([]);
  const [baseRows, setBaseRows] = React.useState<WorkspaceSubtitleRow[]>([]);
  const [comparisonRows, setComparisonRows] = React.useState<
    WorkspaceSubtitleRow[]
  >([]);
  const [subtitleFormat, setSubtitleFormat] = React.useState("srt");
  const [activeSubtitleDocument, setActiveSubtitleDocument] =
    React.useState<SubtitleDocument | null>(null);
  const [subtitleLoading, setSubtitleLoading] = React.useState(false);
  const [subtitleError, setSubtitleError] = React.useState("");
  const [subtitleSaveError, setSubtitleSaveError] = React.useState("");
  const [selectedRowId, setSelectedRowId] = React.useState("");
  const [editingRowId, setEditingRowId] = React.useState("");
  const [hoveredRowId, setHoveredRowId] = React.useState("");
  const [playheadMs, setPlayheadMs] = React.useState(0);
  const [isPlaying, setIsPlaying] = React.useState(false);
  const [previewRenderSize, setPreviewRenderSize] = React.useState({
    width: 0,
    height: 0,
  });
  const [reloadRevision, setReloadRevision] = React.useState(0);
  const [reviewDecisions, setReviewDecisions] = React.useState<
    Record<number, WorkspaceReviewDecision>
  >({});
  const [workspaceStateReady, setWorkspaceStateReady] = React.useState(
    () => !fastReadLatestStateEnabled,
  );

  const rowsRef = React.useRef(rows);
  const baseRowsRef = React.useRef(baseRows);
  const parseSubtitleAsyncRef = React.useRef(parseSubtitle.mutateAsync);
  const generateWorkspacePreviewVTTAsyncRef = React.useRef(
    generateWorkspacePreviewVTT.mutateAsync,
  );
  const lastLoadedSubtitleKeyRef = React.useRef("");
  const lastLoadedComparisonSubtitleKeyRef = React.useRef("");
  const lastSavedWorkspaceStateJSONRef = React.useRef("");
  const completedWorkspaceRestoreKeyRef = React.useRef("");
  const reviewReadyToastSessionIdRef = React.useRef("");
  const reviewDecisionSessionIdRef = React.useRef("");
  const reviewFocusSessionIdRef = React.useRef("");
  const previewVttRequestVersionRef = React.useRef(0);
  const lastPreviewVttRequestKeyRef = React.useRef("");
  const latestWorkspaceStateRequestRef = React.useRef<{
    libraryId: string;
    stateJson: string;
  } | null>(null);
  const handlePreviewRenderSizeChange = React.useCallback(
    (size: { width: number; height: number }) => {
      setPreviewRenderSize((current) =>
        current.width === size.width && current.height === size.height
          ? current
          : size,
      );
    },
    [],
  );

  React.useEffect(() => {
    rowsRef.current = rows;
  }, [rows]);

  React.useEffect(() => {
    baseRowsRef.current = baseRows;
  }, [baseRows]);

  React.useEffect(() => {
    parseSubtitleAsyncRef.current = parseSubtitle.mutateAsync;
  }, [parseSubtitle.mutateAsync]);

  React.useEffect(() => {
    generateWorkspacePreviewVTTAsyncRef.current =
      generateWorkspacePreviewVTT.mutateAsync;
  }, [generateWorkspacePreviewVTT.mutateAsync]);

  React.useEffect(() => {
    setWorkspaceSubtitleMonoStyle(null);
    setWorkspaceSubtitleLingualStyle(null);
    setSubtitleStyleSidebarOpen(false);
    setPreviewVttContent("");
    lastPreviewVttRequestKeyRef.current = "";
  }, [library?.id]);

  const workspaceEmptyTexts = React.useMemo(
    () => ({
      title: t("library.workspace.emptyTitle"),
      description: t("library.workspace.emptyDescription"),
    }),
    [t],
  );

  const subtitleLoadTexts = React.useMemo(
    () => ({
      empty: t("library.workspace.subtitle.noSelection"),
      documentMissing: t("library.workspace.subtitle.sourceUnavailable"),
      parseFailed: t("library.workspace.subtitle.parseFailed"),
    }),
    [t],
  );

  const activeSubtitleDocumentId =
    activeSubtitleFile?.storage.documentId?.trim() ?? "";
  const activeSubtitlePath =
    activeSubtitleFile?.storage.localPath?.trim() ?? "";
  const comparisonSubtitleFile = React.useMemo(
    () =>
      subtitleFiles.find(
        (file) =>
          file.id === comparisonSubtitleFileId &&
          file.id !== activeSubtitleFile?.id,
      ) ?? null,
    [activeSubtitleFile?.id, comparisonSubtitleFileId, subtitleFiles],
  );
  const comparisonSubtitleDocumentId =
    comparisonSubtitleFile?.storage.documentId?.trim() ?? "";
  const comparisonSubtitlePath =
    comparisonSubtitleFile?.storage.localPath?.trim() ?? "";
  const subtitleLoadKey = React.useMemo(
    () =>
      JSON.stringify({
        libraryId: library?.id ?? "",
        subtitleFileId: activeSubtitleFile?.id ?? "",
        subtitleDocumentId: activeSubtitleDocumentId,
        subtitlePath: activeSubtitlePath,
        reloadRevision,
        language,
      }),
    [
      activeSubtitleDocumentId,
      activeSubtitleFile?.id,
      activeSubtitlePath,
      language,
      library?.id,
      reloadRevision,
    ],
  );

  const setLoadedRows = React.useCallback(
    (nextRows: WorkspaceSubtitleRow[], nextFormat: string, nextError = "") => {
      rowsRef.current = nextRows;
      baseRowsRef.current = nextRows;
      setRows((current) => (rowsEqual(current, nextRows) ? current : nextRows));
      setBaseRows((current) =>
        rowsEqual(current, nextRows) ? current : nextRows,
      );
      setSubtitleFormat((current) =>
        current == nextFormat ? current : nextFormat,
      );
      setSubtitleError((current) =>
        current == nextError ? current : nextError,
      );
      setSubtitleSaveError("");
      setSelectedRowId((current) =>
        current == (nextRows[0]?.id ?? "") ? current : (nextRows[0]?.id ?? ""),
      );
      setEditingRowId("");
      setPlayheadMs((current) =>
        current == (nextRows[0]?.startMs ?? 0)
          ? current
          : (nextRows[0]?.startMs ?? 0),
      );
      setIsPlaying(false);
    },
    [],
  );

  React.useEffect(() => {
    if (lastLoadedSubtitleKeyRef.current == subtitleLoadKey) {
      return;
    }
    lastLoadedSubtitleKeyRef.current = subtitleLoadKey;

    if (!library) {
      setActiveSubtitleDocument(null);
      setLoadedRows([], "srt", "");
      setSubtitleLoading(false);
      return;
    }

    const file = activeSubtitleFile;
    if (!file) {
      setActiveSubtitleDocument(null);
      setLoadedRows([], "srt", subtitleLoadTexts.empty);
      setSubtitleLoading(false);
      return;
    }

    const request = {
      fileId: file.id,
      documentId: activeSubtitleDocumentId || undefined,
      path: activeSubtitlePath || undefined,
    };

    if (!request.documentId && !request.path) {
      setActiveSubtitleDocument(null);
      setLoadedRows(
        [],
        resolveFileFormat(file).toLowerCase() || "srt",
        subtitleLoadTexts.documentMissing,
      );
      setSubtitleLoading(false);
      return;
    }

    let cancelled = false;
    setSubtitleLoading(true);
    setActiveSubtitleDocument(null);

    void parseSubtitleAsyncRef
      .current(request)
      .then((result) => {
        if (cancelled) {
          return;
        }
        setActiveSubtitleDocument(result.document);
        setLoadedRows(
          buildSubtitleRows(result.document),
          (result.format || resolveFileFormat(file) || "srt").toLowerCase(),
        );
      })
      .catch((error) => {
        if (cancelled) {
          return;
        }
        setActiveSubtitleDocument(null);
        setLoadedRows(
          [],
          (resolveFileFormat(file) || "srt").toLowerCase(),
          resolveErrorMessage(error, subtitleLoadTexts.parseFailed),
        );
      })
      .finally(() => {
        if (!cancelled) {
          setSubtitleLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    activeSubtitleDocumentId,
    activeSubtitleFile,
    activeSubtitlePath,
    library?.id,
    setLoadedRows,
    subtitleLoadKey,
    subtitleLoadTexts,
  ]);

  const comparisonSubtitleLoadKey = React.useMemo(
    () =>
      JSON.stringify({
        primarySubtitleFileId: activeSubtitleFile?.id ?? "",
        comparisonSubtitleFileId: comparisonSubtitleFile?.id ?? "",
        comparisonSubtitleDocumentId,
        comparisonSubtitlePath,
        reloadRevision,
        reviewSessionId: pendingReview?.sessionId ?? "",
        reviewStatus: reviewSession?.status ?? "",
      }),
    [
      activeSubtitleFile?.id,
      comparisonSubtitleDocumentId,
      comparisonSubtitleFile?.id,
      comparisonSubtitlePath,
      pendingReview?.sessionId,
      reloadRevision,
      reviewSession?.status,
    ],
  );

  React.useEffect(() => {
    if (
      lastLoadedComparisonSubtitleKeyRef.current == comparisonSubtitleLoadKey
    ) {
      return;
    }
    lastLoadedComparisonSubtitleKeyRef.current = comparisonSubtitleLoadKey;

    const file = comparisonSubtitleFile;
    if (pendingReview) {
      if (reviewSession?.candidateDocument) {
        setComparisonRows(buildSubtitleRows(reviewSession.candidateDocument));
      } else {
        setComparisonRows([]);
      }
      return;
    }
    if (!file) {
      setComparisonRows([]);
      return;
    }

    const request = {
      fileId: file.id,
      documentId: comparisonSubtitleDocumentId || undefined,
      path: comparisonSubtitlePath || undefined,
    };

    if (!request.documentId && !request.path) {
      setComparisonRows([]);
      return;
    }

    let cancelled = false;
    setComparisonRows([]);

    void parseSubtitleAsyncRef
      .current(request)
      .then((result) => {
        if (cancelled) {
          return;
        }
        setComparisonRows(buildSubtitleRows(result.document));
      })
      .catch(() => {
        if (!cancelled) {
          setComparisonRows([]);
        }
      });

    return () => {
      cancelled = true;
    };
  }, [
    comparisonSubtitleDocumentId,
    comparisonSubtitleFile,
    comparisonSubtitleLoadKey,
    comparisonSubtitlePath,
    pendingReview,
    reviewSession?.candidateDocument,
  ]);

  const applyRows = React.useCallback(
    (updater: (current: WorkspaceSubtitleRow[]) => WorkspaceSubtitleRow[]) => {
      const current = rowsRef.current;
      const next = updater(current);
      if (rowsEqual(current, next)) {
        return;
      }
      rowsRef.current = next;
      setRows(next);
      setSubtitleSaveError("");
    },
    [],
  );

  const durationMs = React.useMemo(
    () => resolveWorkspaceDurationMs(activeVideoFile, rows),
    [activeVideoFile, rows],
  );
  const videoOptions = React.useMemo(
    () =>
      workspaceVideoTracks.length > 0
        ? resolveWorkspaceProjectVideoOptions(workspaceVideoTracks)
        : resolveVideoVersionOptions(videoFiles),
    [videoFiles, workspaceVideoTracks],
  );
  const blockedActions = React.useMemo(
    () =>
      actionablePendingReview
        ? new Set(pendingReview?.blockedActions ?? [])
        : new Set<string>(),
    [actionablePendingReview, pendingReview?.blockedActions],
  );
  const canUseDualDisplay =
    rows.length > 0 &&
    !blockedActions.has("proofread") &&
    !blockedActions.has("qa");
  const allSubtitleTrackOptions = React.useMemo(
    () =>
      workspaceSubtitleTracks.length > 0
        ? resolveWorkspaceProjectSubtitleTrackOptions(workspaceSubtitleTracks)
        : resolveSubtitleTrackOptions(subtitleFiles),
    [subtitleFiles, workspaceSubtitleTracks],
  );
  const dualEligibleTrackIDs = React.useMemo(
    () =>
      new Set(
        workspaceSubtitleTracks
          .filter((track) => trackSupportsDualDisplay(track))
          .map((track) => track.trackId),
      ),
    [workspaceSubtitleTracks],
  );
  const dualEligibleSubtitleTrackOptions = React.useMemo(
    () =>
      workspaceSubtitleTracks.length > 0
        ? allSubtitleTrackOptions.filter((option) =>
            dualEligibleTrackIDs.has(option.value),
          )
        : allSubtitleTrackOptions,
    [allSubtitleTrackOptions, dualEligibleTrackIDs, workspaceSubtitleTracks.length],
  );
  const effectiveDisplayMode = canUseDualDisplay ? displayMode : "mono";
  const subtitleTrackOptions = React.useMemo(
    () =>
      effectiveDisplayMode === "bilingual"
        ? dualEligibleSubtitleTrackOptions
        : allSubtitleTrackOptions,
    [
      allSubtitleTrackOptions,
      dualEligibleSubtitleTrackOptions,
      effectiveDisplayMode,
    ],
  );
  const monoStyleOptions = React.useMemo<WorkspaceSelectOption[]>(
    () =>
      (props.moduleConfig?.subtitleStyles?.monoStyles ?? []).map((style) => ({
        value: style.id,
        label: style.name || style.id,
      })),
    [props.moduleConfig?.subtitleStyles?.monoStyles],
  );
  const lingualStyleOptions = React.useMemo<WorkspaceSelectOption[]>(
    () =>
      (props.moduleConfig?.subtitleStyles?.bilingualStyles ?? []).map(
        (style) => ({
          value: style.id,
          label: style.name || style.id,
        }),
      ),
    [props.moduleConfig?.subtitleStyles?.bilingualStyles],
  );
  const subtitleStyleDefaults = React.useMemo(
    () => resolveSubtitleStyleDefaults(props.moduleConfig),
    [props.moduleConfig],
  );
  const subtitleExportPresets = React.useMemo(
    () => resolveSubtitleExportPresets(props.moduleConfig),
    [props.moduleConfig],
  );
  const videoPresetOptions = React.useMemo<TranscodePreset[]>(
    () =>
      (transcodePresetsQuery.data ?? []).filter(
        (preset) => preset.outputType !== "audio",
      ),
    [transcodePresetsQuery.data],
  );
  const selectedVideoExportPreset = React.useMemo(
    () =>
      videoPresetOptions.find((preset) => preset.id === exportVideoPresetId) ??
      videoPresetOptions[0] ??
      null,
    [exportVideoPresetId, videoPresetOptions],
  );
  const comparisonTrackOptions = React.useMemo(
    () => [
      {
        value: "",
        label: t("library.workspace.header.noPairedTrack"),
        language: "",
        hint: t("library.workspace.header.noPairedTrackHint"),
      },
      ...dualEligibleSubtitleTrackOptions.filter(
        (option) => option.value !== (activeSubtitleFile?.id ?? ""),
      ),
    ],
    [activeSubtitleFile?.id, dualEligibleSubtitleTrackOptions, t],
  );
  const activeTrack = React.useMemo(
    () =>
      allSubtitleTrackOptions.find(
        (option) => option.value == activeSubtitleFile?.id,
      ) ??
      allSubtitleTrackOptions[0] ??
      null,
    [activeSubtitleFile?.id, allSubtitleTrackOptions],
  );
  const comparisonTrack = React.useMemo(
    () =>
      allSubtitleTrackOptions.find(
        (option) => option.value == comparisonSubtitleFileId,
      ) ?? null,
    [comparisonSubtitleFileId, allSubtitleTrackOptions],
  );
  const currentTrackLanguage = React.useMemo(
    () =>
      activeTrack?.language?.trim() ||
      activeSubtitleFile?.media?.language?.trim() ||
      "",
    [activeSubtitleFile?.media?.language, activeTrack?.language],
  );
  const guidelineOptions = React.useMemo(
    () => resolveWorkspaceGuidelineOptions(),
    [],
  );
  const translateLanguageOptions = React.useMemo(
    () => resolveTranslateLanguageOptions(props.moduleConfig),
    [props.moduleConfig],
  );
  const subtitleExportLanguageOptions = React.useMemo(
    () =>
      translateLanguageOptions.map((option) => ({
        value: option.value,
        label: `${option.label} (${option.value})`,
      })),
    [translateLanguageOptions],
  );
  const translateGlossaryOptions = React.useMemo(
    () =>
      resolveGlossaryConstraintOptions(
        props.moduleConfig,
        "translate",
        currentTrackLanguage,
        translateTargetLanguage,
      ),
    [currentTrackLanguage, props.moduleConfig, translateTargetLanguage],
  );
  const proofreadGlossaryOptions = React.useMemo(
    () =>
      resolveGlossaryConstraintOptions(
        props.moduleConfig,
        "proofread",
        currentTrackLanguage,
        currentTrackLanguage,
      ),
    [currentTrackLanguage, props.moduleConfig],
  );
  const referenceTrackOptions = React.useMemo(
    () => [
      {
        value: "",
        label: t("library.workspace.header.noReferenceTrack"),
        hint: t("library.workspace.header.noReferenceTrackHint"),
      },
      ...allSubtitleTrackOptions
        .filter((option) => option.value !== (activeSubtitleFile?.id ?? ""))
        .map((option) => ({
          value: option.value,
          label: option.label,
          hint:
            option.hint ||
            t("library.workspace.header.existingSubtitleLane"),
        })),
    ],
    [activeSubtitleFile?.id, allSubtitleTrackOptions, t],
  );
  const translatePromptOptions = React.useMemo(
    () => resolveTranslatePromptConstraintOptions(props.moduleConfig),
    [props.moduleConfig],
  );
  const proofreadPromptOptions = React.useMemo(
    () => resolveProofreadPromptConstraintOptions(props.moduleConfig),
    [props.moduleConfig],
  );
  const defaultAssistant = React.useMemo(() => {
    const assistants = assistantsQuery.data ?? [];
    return assistants.find((item) => item.isDefault) ?? assistants[0] ?? null;
  }, [assistantsQuery.data]);
  const translateAssistantModelRef =
    defaultAssistant?.model?.agent?.primary?.trim() ?? "";
  const translateAssistantModel = React.useMemo(
    () => parseAssistantModelRef(translateAssistantModelRef),
    [translateAssistantModelRef],
  );
  const translateProviderId = translateAssistantModel.providerId || null;
  const translateProviderSecretQuery = useProviderSecret(translateProviderId);
  const translateReadiness = React.useMemo(() => {
    const providersWithModels = providersWithModelsQuery.data ?? [];
    const gatewayReadiness = resolveGatewayCoreReadiness({
      assistant: defaultAssistant,
      providersWithModels,
      hasProviderApiKey: translateProviderId
        ? Boolean(translateProviderSecretQuery.data?.apiKey?.trim())
        : null,
      checking:
        assistantsQuery.isLoading ||
        providersWithModelsQuery.isLoading ||
        (Boolean(translateProviderId) && translateProviderSecretQuery.isLoading),
      includeGatewayDisabled: false,
      requireAssistantEnabled: true,
    });
    if (gatewayReadiness.checking) {
      return {
        ready: false,
        checking: true,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.checkingTitle"),
        description: t("library.workspace.readiness.checkingDescription"),
        actionTarget: "gateway" as const,
      };
    }
    if (gatewayReadiness.issues.includes("assistant.selection")) {
      return {
        ready: false,
        checking: false,
        assistantId: "",
        title: t("library.workspace.readiness.noAssistantTitle"),
        description: t("library.workspace.readiness.noAssistantDescription"),
        actionTarget: "gateway" as const,
      };
    }
    if (gatewayReadiness.issues.includes("assistant.disabled")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.assistantDisabledTitle"),
        description: t("library.workspace.readiness.assistantDisabledDescription"),
        actionTarget: "gateway" as const,
      };
    }
    if (gatewayReadiness.issues.includes("model.agent.primary")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.assistantModelMissingTitle"),
        description: t("library.workspace.readiness.assistantModelMissingDescription"),
        actionTarget: "gateway" as const,
      };
    }
    if (gatewayReadiness.issues.includes("providers.models")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.noProviderModelTitle"),
        description: t("library.workspace.readiness.noProviderModelDescription"),
        actionTarget: "provider" as const,
      };
    }
    if (gatewayReadiness.issues.includes("model.reference.invalid")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.modelReferenceInvalidTitle"),
        description: t("library.workspace.readiness.modelReferenceInvalidDescription"),
        actionTarget: "gateway" as const,
      };
    }
    if (gatewayReadiness.issues.includes("provider.unavailable")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.providerUnavailableTitle"),
        description: t("library.workspace.readiness.providerUnavailableDescription"),
        actionTarget: "provider" as const,
      };
    }
    if (gatewayReadiness.issues.includes("model.unavailable")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.modelUnavailableTitle"),
        description: t("library.workspace.readiness.modelUnavailableDescription"),
        actionTarget: "provider" as const,
      };
    }
    if (gatewayReadiness.issues.includes("provider.apiKey.missing")) {
      return {
        ready: false,
        checking: false,
        assistantId: defaultAssistant?.id ?? "",
        title: t("library.workspace.readiness.apiKeyMissingTitle"),
        description: t("library.workspace.readiness.apiKeyMissingDescription"),
        actionTarget: "provider" as const,
      };
    }
    const providerLabel =
      gatewayReadiness.providerEntry?.provider.name?.trim() || gatewayReadiness.providerEntry?.provider.id || "";
    const modelLabel =
      gatewayReadiness.providerEntry?.models.find(
        (item) =>
          item.name.trim().toLowerCase() ===
          translateAssistantModel.modelName.trim().toLowerCase(),
      )?.displayName?.trim() || translateAssistantModel.modelName;
    const assistantLabel =
      defaultAssistant?.identity?.name?.trim() || defaultAssistant?.id || "";
    return {
      ready: true,
      checking: false,
      assistantId: defaultAssistant?.id ?? "",
      title: t("library.workspace.readiness.readyTitle"),
      description: t("library.workspace.readiness.readyDescription")
        .replace("{assistant}", assistantLabel)
        .replace("{provider}", providerLabel)
        .replace("{model}", modelLabel),
      actionTarget: "gateway" as const,
    };
  }, [
    assistantsQuery.data,
    assistantsQuery.isLoading,
    defaultAssistant,
    providersWithModelsQuery.data,
    providersWithModelsQuery.isLoading,
    t,
    translateAssistantModel.modelName,
    translateAssistantModel.providerId,
    translateProviderId,
    translateProviderSecretQuery.data?.apiKey,
    translateProviderSecretQuery.isLoading,
  ]);
  const nextComparisonTrackId = React.useMemo(() => {
    const activeTrackId =
      activeSubtitleFile?.id ?? dualEligibleSubtitleTrackOptions[0]?.value ?? "";
    if (
      comparisonSubtitleFileId &&
      comparisonSubtitleFileId != activeTrackId &&
      dualEligibleSubtitleTrackOptions.some(
        (option) => option.value == comparisonSubtitleFileId,
      )
    ) {
      return comparisonSubtitleFileId;
    }
    return (
      dualEligibleSubtitleTrackOptions.find(
        (option) => option.value != activeTrackId,
      )
        ?.value ?? ""
    );
  }, [
    activeSubtitleFile?.id,
    comparisonSubtitleFileId,
    dualEligibleSubtitleTrackOptions,
  ]);
  const resourceName = resolveResourceName(
    library,
    activeVideoFile,
    activeSubtitleFile,
  );
  const headerMediaVideoFile = React.useMemo(() => {
    const candidates = activeVideoFile
      ? [
          activeVideoFile,
          ...videoFiles.filter((file) => file.id !== activeVideoFile.id),
        ]
      : videoFiles;
    return (
      candidates.find((file) => {
        const kind = normalizeFileKind(file.kind);
        const hasVisualMeta =
          (file.media?.width ?? 0) > 0 && (file.media?.height ?? 0) > 0;
        return (kind === "video" || kind === "transcode") && hasVisualMeta;
      }) ??
      candidates.find((file) => {
        const kind = normalizeFileKind(file.kind);
        return kind === "video" || kind === "transcode";
      }) ??
      candidates.find(
        (file) =>
          (file.media?.durationMs ?? 0) > 0 || (file.media?.frameRate ?? 0) > 0,
      ) ??
      null
    );
  }, [activeVideoFile, videoFiles]);
  const workspaceHeaderMeta = React.useMemo(() => {
    if (activeEditor == "subtitle") {
      const subtitleCueCount =
        activeSubtitleFile?.media?.cueCount ?? rows.length;
      return [
        {
          label: t("library.workspace.meta.language"),
          value: currentTrackLanguage || "-",
        },
        {
          label: t("library.workspace.meta.format"),
          value:
            resolveFileFormat(activeSubtitleFile) ||
            activeSubtitleFile?.kind?.toUpperCase() ||
            "-",
        },
        {
          label: t("library.workspace.meta.cues"),
          value: subtitleCueCount > 0 ? String(subtitleCueCount) : "-",
        },
      ];
    }
    const headerDurationMs =
      activeVideoFile?.media?.durationMs ??
      headerMediaVideoFile?.media?.durationMs ??
      rows[rows.length - 1]?.endMs ??
      0;
    const headerResolutionFile =
      activeVideoFile?.media?.width && activeVideoFile?.media?.height
        ? activeVideoFile
        : headerMediaVideoFile;
    const headerFrameRate =
      activeVideoFile?.media?.frameRate && activeVideoFile.media.frameRate > 0
        ? activeVideoFile.media.frameRate
        : headerMediaVideoFile?.media?.frameRate;
    return [
      {
        label: t("library.workspace.meta.duration"),
        value: formatMediaDuration(headerDurationMs),
      },
      {
        label: t("library.workspace.meta.resolution"),
        value: formatResolution(headerResolutionFile),
      },
      {
        label: t("library.workspace.meta.frameRate"),
        value: formatFrameRate(headerFrameRate),
      },
    ];
  }, [
    activeEditor,
    activeSubtitleFile,
    activeVideoFile,
    currentTrackLanguage,
    headerMediaVideoFile,
    rows,
    t,
  ]);
  const resolvedRows = React.useMemo(
    () =>
      resolveWorkspaceRows(
        rows,
        baseRows,
        guidelineProfileId,
        qaCheckSettings,
        comparisonRows,
        reviewSession?.suggestions ?? [],
        pendingReview?.kind ?? "",
        reviewDecisions,
      ),
    [
      baseRows,
      comparisonRows,
      guidelineProfileId,
      pendingReview?.kind,
      qaCheckSettings,
      reviewDecisions,
      reviewSession?.suggestions,
      rows,
    ],
  );
  const activeWorkspaceMonoStyle = React.useMemo(
    () =>
      workspaceSubtitleMonoStyle ??
      workspaceProjectMonoStyle ??
      createWorkspaceMonoStyleDraft(
        resolveDefaultMonoStyle(props.moduleConfig) ?? undefined,
      ) ??
      null,
    [
      props.moduleConfig,
      workspaceProjectMonoStyle,
      workspaceSubtitleMonoStyle,
    ],
  );
  const activeWorkspaceLingualStyle = React.useMemo(
    () =>
      workspaceSubtitleLingualStyle ??
      workspaceProjectLingualStyle ??
      createWorkspaceLingualStyleDraft(
        resolveDefaultBilingualStyle(props.moduleConfig) ?? undefined,
      ) ??
      null,
    [
      props.moduleConfig,
      workspaceProjectLingualStyle,
      workspaceSubtitleLingualStyle,
    ],
  );
  const workspacePreviewRows = React.useMemo(
    () =>
      resolvedRows.map((row) => ({
        startMs: row.startMs,
        endMs: row.endMs,
        primaryText: row.sourceText,
        secondaryText: row.translationText,
      })),
    [resolvedRows],
  );
  const workspaceFontMappings = React.useMemo(
    () => props.moduleConfig?.subtitleStyles?.fonts ?? [],
    [props.moduleConfig?.subtitleStyles?.fonts],
  );
  const resolveWorkspaceStyleDocumentContent = React.useCallback(
    async (mode: "mono" | "bilingual") => {
      if (mode === "bilingual" && activeWorkspaceLingualStyle) {
        const result = await generateSubtitleStylePreviewASS.mutateAsync({
          type: "bilingual",
          bilingual: activeWorkspaceLingualStyle,
          fontMappings: workspaceFontMappings,
        });
        return result.assContent?.trim() ? `${result.assContent.trim()}\n` : "";
      }
      if (activeWorkspaceMonoStyle) {
        const result = await generateSubtitleStylePreviewASS.mutateAsync({
          type: "mono",
          mono: activeWorkspaceMonoStyle,
          fontMappings: workspaceFontMappings,
        });
        return result.assContent?.trim() ? `${result.assContent.trim()}\n` : "";
      }
      return "";
    },
    [
      activeWorkspaceLingualStyle,
      activeWorkspaceMonoStyle,
      generateSubtitleStylePreviewASS,
      workspaceFontMappings,
    ],
  );
  React.useEffect(() => {
    const monoStyle = activeWorkspaceMonoStyle;
    const lingualStyle = activeWorkspaceLingualStyle;
    const requestPayload = {
      displayMode: effectiveDisplayMode,
      mono: monoStyle ?? undefined,
      lingual: lingualStyle ?? undefined,
      rows: workspacePreviewRows,
      fontMappings: workspaceFontMappings,
      previewWidth:
        previewRenderSize.width > 0 ? previewRenderSize.width : undefined,
      previewHeight:
        previewRenderSize.height > 0 ? previewRenderSize.height : undefined,
    };
    const requestKey = JSON.stringify(requestPayload);
    if (lastPreviewVttRequestKeyRef.current === requestKey) {
      return;
    }
    lastPreviewVttRequestKeyRef.current = requestKey;
    const requestVersion = previewVttRequestVersionRef.current + 1;
    previewVttRequestVersionRef.current = requestVersion;
    const timer = window.setTimeout(() => {
      void generateWorkspacePreviewVTTAsyncRef
        .current(requestPayload)
        .then((result) => {
          if (previewVttRequestVersionRef.current !== requestVersion) {
            return;
          }
          const nextContent = result.vttContent?.trim() ? `${result.vttContent.trim()}\n` : "";
          setPreviewVttContent(nextContent);
        })
        .catch(() => {
          if (previewVttRequestVersionRef.current !== requestVersion) {
            return;
          }
          setPreviewVttContent("");
        });
    }, 90);
    return () => {
      window.clearTimeout(timer);
    };
  }, [
    activeWorkspaceLingualStyle,
    activeWorkspaceMonoStyle,
    effectiveDisplayMode,
    previewRenderSize,
    workspaceFontMappings,
    workspacePreviewRows,
  ]);
  const exportEmbeddedSubtitleRoute = React.useMemo(
    () =>
      resolveVideoExportSoftSubtitleRoute(
        selectedVideoExportPreset?.container ?? "mp4",
      ),
    [selectedVideoExportPreset?.container],
  );
  const currentRow = React.useMemo(
    () => resolveCurrentRow(resolvedRows, playheadMs),
    [playheadMs, resolvedRows],
  );
  const currentRowId = currentRow?.id ?? "";
  const qaSummary = React.useMemo(
    () => resolveWorkspaceQaSummary(resolvedRows),
    [resolvedRows],
  );
  const runningTrackTasks = activeSubtitleTrackState?.runningTasks;
  const subtitleReviewBlocked = blockedActions.size > 0;
  const translateTaskRunning = (runningTrackTasks?.translate?.length ?? 0) > 0;
  const proofreadTaskRunning = Boolean(runningTrackTasks?.proofread);
  const canTranslateAction = rows.length > 0 && !blockedActions.has("translate");
  const canProofreadAction = rows.length > 0 && !blockedActions.has("proofread");
  const canQaAction = rows.length > 0;
  const canRestoreAction = rows.length > 0;
  const subtitleActionLockDescription = React.useMemo(() => {
    if (!subtitleReviewBlocked) {
      return "";
    }
    return t("library.workspace.review.lockedTooltip");
  }, [subtitleReviewBlocked, t]);
  const exportBlocked = blockedActions.has("export");
  const reviewSuggestions = reviewSession?.suggestions ?? [];
  const reviewDecisionSummary = React.useMemo(() => {
    const summary = {
      totalCount:
        reviewSuggestions.length > 0
          ? reviewSuggestions.length
          : pendingReview?.changedCueCount ?? 0,
      acceptedCount: 0,
      rejectedCount: 0,
      unresolvedCount: 0,
    };
    for (const suggestion of reviewSuggestions) {
      const decision = reviewDecisions[suggestion.cueIndex] ?? "undecided";
      if (decision === "accept") {
        summary.acceptedCount += 1;
        continue;
      }
      if (decision === "reject") {
        summary.rejectedCount += 1;
        continue;
      }
      summary.unresolvedCount += 1;
    }
    if (reviewSuggestions.length === 0) {
      summary.unresolvedCount = summary.totalCount;
    }
    return summary;
  }, [pendingReview?.changedCueCount, reviewDecisions, reviewSuggestions]);
  const pendingReviewCount = reviewDecisionSummary.totalCount;
  const pendingReviewUnresolvedCount = reviewDecisionSummary.unresolvedCount;
  const reviewCompletionReady =
    Boolean(pendingReview) && reviewDecisionSummary.unresolvedCount === 0;
  const translateButtonLabel = React.useMemo(
    () => buildWorkspaceTranslateTaskLabel(runningTrackTasks?.translate, t),
    [runningTrackTasks?.translate, t],
  );
  const proofreadButtonLabel = React.useMemo(
    () =>
      proofreadTaskRunning
        ? buildWorkspaceTaskProgressLabel(
            runningTrackTasks?.proofread,
            t("library.workspace.header.proofreadRunning"),
          )
        : t("library.workspace.header.proofread"),
    [proofreadTaskRunning, runningTrackTasks?.proofread, t],
  );
  const qaButtonLabel = t("library.workspace.header.qa");
  const pendingReviewButtonLabel = React.useMemo(() => {
    if (!pendingReview) {
      return "";
    }
    if (reviewCompletionReady) {
      return t("library.workspace.review.complete");
    }
    const kindLabel =
      pendingReview.kind === "qa"
        ? t("library.workspace.review.pendingQaShort")
        : t("library.workspace.review.pendingProofreadShort");
    return t("library.workspace.review.pendingButtonWithMeta")
      .replace("{kind}", kindLabel)
      .replace("{count}", String(pendingReviewUnresolvedCount));
  }, [pendingReview, pendingReviewUnresolvedCount, reviewCompletionReady, t]);
  const pendingReviewMenuDescription = React.useMemo(() => {
    if (!pendingReview) {
      return "";
    }
    return t("library.workspace.review.pendingDescription").replace(
      "{count}",
      String(pendingReviewUnresolvedCount),
    );
  }, [pendingReview, pendingReviewUnresolvedCount, t]);
  const activeGuidelineLabel = React.useMemo(
    () => resolveWorkspaceGuidelineLabel(guidelineProfileId),
    [guidelineProfileId],
  );
  const qaCheckDefinitions = React.useMemo(
    () => resolveWorkspaceQaCheckDefinitions(),
    [language],
  );
  const persistedWorkspaceState = React.useMemo(
    () =>
      resolveLibraryWorkspacePersistedState(
        {
          activeEditor,
          activeVideoFileId: activeVideoFile?.id ?? activeVideoFileId,
          activeSubtitleFileId: activeSubtitleFile?.id ?? activeSubtitleFileId,
          displayMode,
          comparisonSubtitleFileId:
            comparisonSubtitleFile?.id ?? comparisonSubtitleFileId,
          guidelineProfileId,
          qaCheckSettings,
          subtitleMonoStyle: activeWorkspaceMonoStyle ?? undefined,
          subtitleLingualStyle: activeWorkspaceLingualStyle ?? undefined,
          subtitleStyleSidebarOpen,
        },
        {
          libraryId: library?.id ?? "",
          videoFiles,
          subtitleFiles,
          defaultSubtitleMonoStyle: activeWorkspaceMonoStyle ?? undefined,
          defaultSubtitleLingualStyle: activeWorkspaceLingualStyle ?? undefined,
        },
      ),
    [
      activeEditor,
      activeWorkspaceLingualStyle,
      activeWorkspaceMonoStyle,
      activeSubtitleFile?.id,
      activeSubtitleFileId,
      activeVideoFile?.id,
      activeVideoFileId,
      comparisonSubtitleFile?.id,
      comparisonSubtitleFileId,
      displayMode,
      guidelineProfileId,
      library?.id,
      qaCheckSettings,
      subtitleStyleSidebarOpen,
      subtitleFiles,
      videoFiles,
    ],
  );
  const persistedWorkspaceStateJSON = React.useMemo(
    () => JSON.stringify({ version: 1, ...persistedWorkspaceState }),
    [persistedWorkspaceState],
  );
  const workspaceRestoreKey = `${library?.id ?? ""}:${fastReadLatestStateEnabled ? "restore" : "skip"}`;

  React.useEffect(() => {
    lastSavedWorkspaceStateJSONRef.current = "";
    if (!library?.id || !fastReadLatestStateEnabled) {
      completedWorkspaceRestoreKeyRef.current = workspaceRestoreKey;
      setWorkspaceStateReady(true);
      return;
    }
    completedWorkspaceRestoreKeyRef.current = "";
    setWorkspaceStateReady(false);
  }, [fastReadLatestStateEnabled, library?.id, workspaceRestoreKey]);

  React.useEffect(() => {
    if (!library?.id || !fastReadLatestStateEnabled) {
      return;
    }
    if (completedWorkspaceRestoreKeyRef.current == workspaceRestoreKey) {
      return;
    }
    if (workspaceStateQuery.isLoading || workspaceStateQuery.isFetching) {
      return;
    }
    completedWorkspaceRestoreKeyRef.current = workspaceRestoreKey;
    if (workspaceStateQuery.status == "success") {
      const parsedState = parseLibraryWorkspacePersistedState(
        workspaceStateQuery.data.stateJson,
      );
      if (parsedState) {
        const restoredState = resolveLibraryWorkspacePersistedState(
          parsedState,
          {
            libraryId: library.id,
            videoFiles,
            subtitleFiles,
            defaultSubtitleMonoStyle: workspaceProjectMonoStyle ?? undefined,
            defaultSubtitleLingualStyle:
              workspaceProjectLingualStyle ?? undefined,
          },
        );
        applyPersistedWorkspaceState(restoredState);
        setWorkspaceSubtitleMonoStyle(
          createWorkspaceMonoStyleDraft(restoredState.subtitleMonoStyle) ??
            null,
        );
        setWorkspaceSubtitleLingualStyle(
          createWorkspaceLingualStyleDraft(
            restoredState.subtitleLingualStyle,
          ) ?? null,
        );
        setSubtitleStyleSidebarOpen(
          Boolean(restoredState.subtitleStyleSidebarOpen),
        );
        lastSavedWorkspaceStateJSONRef.current = JSON.stringify({
          version: 1,
          ...restoredState,
        });
      }
    }
    setWorkspaceStateReady(true);
  }, [
    applyPersistedWorkspaceState,
    fastReadLatestStateEnabled,
    library?.id,
    subtitleFiles,
    videoFiles,
    workspaceRestoreKey,
    workspaceStateQuery.data,
    workspaceStateQuery.isFetching,
    workspaceStateQuery.isLoading,
    workspaceStateQuery.status,
    workspaceProjectLingualStyle,
    workspaceProjectMonoStyle,
  ]);

  React.useEffect(() => {
    if (!library?.id || !workspaceStateReady) {
      return;
    }
    latestWorkspaceStateRequestRef.current = {
      libraryId: library.id,
      stateJson: persistedWorkspaceStateJSON,
    };
    if (
      saveWorkspaceState.isPending ||
      lastSavedWorkspaceStateJSONRef.current == persistedWorkspaceStateJSON
    ) {
      return;
    }
    const timer = window.setTimeout(() => {
      const nextStateJSON = persistedWorkspaceStateJSON;
      void saveWorkspaceState
        .mutateAsync({
          libraryId: library.id,
          stateJson: nextStateJSON,
        })
        .then(() => {
          lastSavedWorkspaceStateJSONRef.current = nextStateJSON;
        })
        .catch(() => {});
    }, 320);
    return () => {
      window.clearTimeout(timer);
    };
  }, [
    library?.id,
    persistedWorkspaceStateJSON,
    saveWorkspaceState.isPending,
    saveWorkspaceState.mutateAsync,
    workspaceStateReady,
  ]);

  React.useEffect(() => {
    return () => {
      const latestRequest = latestWorkspaceStateRequestRef.current;
      const currentLibraryID = library?.id ?? "";
      if (!latestRequest || latestRequest.libraryId !== currentLibraryID) {
        return;
      }
      if (latestRequest.stateJson === lastSavedWorkspaceStateJSONRef.current) {
        return;
      }
      void saveWorkspaceState
        .mutateAsync(latestRequest)
        .then(() => {
          lastSavedWorkspaceStateJSONRef.current = latestRequest.stateJson;
        })
        .catch(() => {});
    };
  }, [library?.id, saveWorkspaceState.mutateAsync]);

  React.useEffect(() => {
    if (!selectedRowId && resolvedRows[0]?.id) {
      setSelectedRowId(resolvedRows[0].id);
      return;
    }
    if (selectedRowId && !resolvedRows.some((row) => row.id == selectedRowId)) {
      setSelectedRowId(resolvedRows[0]?.id ?? "");
    }
  }, [resolvedRows, selectedRowId]);

  React.useEffect(() => {
    if (editingRowId && !resolvedRows.some((row) => row.id == editingRowId)) {
      setEditingRowId("");
    }
  }, [editingRowId, resolvedRows]);

  React.useEffect(() => {
    if (comparisonSubtitleFileId != nextComparisonTrackId) {
      setComparisonSubtitleFileId(nextComparisonTrackId);
    }
  }, [
    comparisonSubtitleFileId,
    nextComparisonTrackId,
    setComparisonSubtitleFileId,
  ]);

  React.useEffect(() => {
    const nextReferenceTrackId = referenceTrackOptions.some(
      (option) => option.value === translateReferenceTrackId,
    )
      ? translateReferenceTrackId
      : "";

    if (translateReferenceTrackId !== nextReferenceTrackId) {
      setTranslateReferenceTrackId(nextReferenceTrackId);
    }
  }, [referenceTrackOptions, translateReferenceTrackId]);

  React.useEffect(() => {
    if (workspaceSubtitleMonoStyle) {
      return;
    }
    const fallback =
      workspaceProjectMonoStyle ??
      createWorkspaceMonoStyleDraft(
        resolveDefaultMonoStyle(props.moduleConfig) ?? undefined,
      );
    if (fallback) {
      setWorkspaceSubtitleMonoStyle(fallback);
    }
  }, [
    props.moduleConfig,
    workspaceProjectMonoStyle,
    workspaceSubtitleMonoStyle,
  ]);

  React.useEffect(() => {
    if (workspaceSubtitleLingualStyle) {
      return;
    }
    const fallback =
      workspaceProjectLingualStyle ??
      createWorkspaceLingualStyleDraft(
        resolveDefaultBilingualStyle(props.moduleConfig) ?? undefined,
      );
    if (fallback) {
      setWorkspaceSubtitleLingualStyle(fallback);
    }
  }, [
    props.moduleConfig,
    workspaceProjectLingualStyle,
    workspaceSubtitleLingualStyle,
  ]);

  React.useEffect(() => {
    if (videoPresetOptions.length === 0) {
      if (exportVideoPresetId) {
        setExportVideoPresetId("");
      }
      return;
    }
    if (
      exportVideoPresetId &&
      videoPresetOptions.some((preset) => preset.id === exportVideoPresetId)
    ) {
      return;
    }
    setExportVideoPresetId(videoPresetOptions[0]?.id ?? "");
  }, [exportVideoPresetId, videoPresetOptions]);

  React.useEffect(() => {
    if (!pendingTranslateOperationId) {
      return;
    }
    const translatedFile = subtitleFiles.find(
      (file) => file.origin.operationId?.trim() == pendingTranslateOperationId,
    );
    if (!translatedFile) {
      return;
    }
    if (translatedFile.id != activeSubtitleFile?.id) {
      setComparisonSubtitleFileId(translatedFile.id);
      if (displayMode == "mono") {
        setDisplayMode("bilingual");
      }
    }
    setPendingTranslateOperationId("");
    messageBus.publishToast({
      intent: "success",
      title: t("library.workspace.notifications.translationReadyTitle"),
      description: t("library.workspace.notifications.translationReadyDescription").replace("{name}", translatedFile.name),
    });
  }, [
    activeSubtitleFile?.id,
    displayMode,
    pendingTranslateOperationId,
    setComparisonSubtitleFileId,
    setDisplayMode,
    subtitleFiles,
    t,
  ]);

  React.useEffect(() => {
    if (!pendingReview?.sessionId) {
      reviewReadyToastSessionIdRef.current = "";
      reviewDecisionSessionIdRef.current = "";
      reviewFocusSessionIdRef.current = "";
      setReviewDecisions({});
      return;
    }
    if (reviewDecisionSessionIdRef.current !== pendingReview.sessionId) {
      reviewDecisionSessionIdRef.current = pendingReview.sessionId;
      setReviewDecisions({});
    }
    if (reviewReadyToastSessionIdRef.current === pendingReview.sessionId) {
      return;
    }
    reviewReadyToastSessionIdRef.current = pendingReview.sessionId;
    messageBus.publishToast({
      intent: "success",
      title:
        pendingReview.kind === "qa"
          ? t("library.workspace.notifications.qaReviewReadyTitle")
          : t("library.workspace.notifications.proofreadReadyTitle"),
      description:
        pendingReview.kind === "qa"
          ? t("library.workspace.notifications.qaReviewReadyDescription").replace("{count}", String(pendingReviewCount))
          : t("library.workspace.notifications.proofreadReadyDescription").replace("{count}", String(pendingReviewCount)),
    });
  }, [pendingReview?.kind, pendingReview?.sessionId, pendingReviewCount, t]);

  React.useEffect(() => {
    if (!pendingReview?.sessionId) {
      reviewFocusSessionIdRef.current = "";
      return;
    }
    if (reviewFocusSessionIdRef.current === pendingReview.sessionId) {
      return;
    }
    if (reviewSuggestions.length === 0 || resolvedRows.length === 0) {
      return;
    }
    reviewFocusSessionIdRef.current = pendingReview.sessionId;
    setActiveEditor("subtitle");
    setSubtitleSearch("");
    setSubtitleFilter("needs-review");
    setSubtitleQaFilter("all");
    const targetCueIndex =
      reviewSuggestions.find(
        (suggestion) =>
          (reviewDecisions[suggestion.cueIndex] ?? "undecided") ===
          "undecided",
      )?.cueIndex ?? reviewSuggestions[0]?.cueIndex;
    if (typeof targetCueIndex !== "number") {
      return;
    }
    const targetRow = rowsRef.current.find((row) => row.index === targetCueIndex);
    if (!targetRow) {
      return;
    }
    setSelectedRowId(targetRow.id);
    setPlayheadMs(targetRow.startMs);
  }, [
    pendingReview?.sessionId,
    resolvedRows.length,
    reviewDecisions,
    reviewSuggestions,
    setActiveEditor,
  ]);

  React.useEffect(() => {
    if (useCurrentGuidelineForImport) {
      setImportGuidelineProfileId(guidelineProfileId);
    }
  }, [guidelineProfileId, useCurrentGuidelineForImport]);

  React.useEffect(() => {
    setTranslateTargetLanguage((current) => {
      if (
        current &&
        translateLanguageOptions.some((option) => option.value == current)
      ) {
        return current;
      }
      return translateLanguageOptions[0]?.value ?? "";
    });
  }, [translateLanguageOptions]);

  React.useEffect(() => {
    setPlayheadMs((current) => clampMs(current, durationMs));
  }, [durationMs]);

  const videoPreviewUrl = React.useMemo(() => {
    const path = activeVideoFile?.storage.localPath?.trim() ?? "";
    return path ? buildAssetPreviewUrl(props.httpBaseURL, path) : "";
  }, [activeVideoFile?.storage.localPath, props.httpBaseURL]);
  const videoPreviewMimeType = React.useMemo(
    () => resolveLibraryMediaMimeType(activeVideoFile),
    [activeVideoFile],
  );
  const waveformGuardKind = React.useMemo(
    () => resolveWorkspaceWaveformGuardKind(activeVideoFile),
    [activeVideoFile],
  );
  const waveformDisabledReason = React.useMemo(() => {
    if (waveformGuardKind === "size") {
      return t("library.workspace.waveform.disabledLargeMedia").replace("{size}", formatBytes(activeVideoFile?.media?.sizeBytes));
    }
    if (waveformGuardKind === "duration") {
      return t("library.workspace.waveform.disabledLongMedia").replace(
        "{duration}",
        formatMediaDuration(activeVideoFile?.media?.durationMs),
      );
    }
    return "";
  }, [
    activeVideoFile?.media?.durationMs,
    activeVideoFile?.media?.sizeBytes,
    t,
    waveformGuardKind,
  ]);

  React.useEffect(() => {
    if (activeEditor != "video" || isPlaying == false || videoPreviewUrl) {
      return;
    }
    const timer = window.setInterval(() => {
      setPlayheadMs((current) => clampMs(current + 120, durationMs));
    }, 120);
    return () => {
      window.clearInterval(timer);
    };
  }, [activeEditor, durationMs, isPlaying, videoPreviewUrl]);

  React.useEffect(() => {
    if (isPlaying && playheadMs >= durationMs) {
      setIsPlaying(false);
    }
  }, [durationMs, isPlaying, playheadMs]);

  React.useEffect(() => {
    if (displayMode !== "bilingual" || canUseDualDisplay) {
      return;
    }
    setDisplayMode("mono");
  }, [canUseDualDisplay, displayMode, setDisplayMode]);

  const filteredVideoRows = React.useMemo(
    () =>
      filterWorkspaceRows(
        resolvedRows,
        videoSearch,
        videoFilter,
        videoQaFilter,
        effectiveDisplayMode,
        currentRowId,
      ),
    [
      currentRowId,
      effectiveDisplayMode,
      resolvedRows,
      videoFilter,
      videoQaFilter,
      videoSearch,
    ],
  );
  const filteredSubtitleRows = React.useMemo(
    () =>
      filterWorkspaceRows(
        resolvedRows,
        subtitleSearch,
        subtitleFilter,
        subtitleQaFilter,
        effectiveDisplayMode,
        currentRowId,
      ),
    [
      currentRowId,
      effectiveDisplayMode,
      resolvedRows,
      subtitleFilter,
      subtitleQaFilter,
      subtitleSearch,
    ],
  );
  const rowsDirty = rowsEqual(rows, baseRows) == false;
  const subtitleDraftSaveState = React.useMemo<
    "saved" | "saving" | "dirty" | "editing" | "error"
  >(() => {
    if (saveSubtitle.isPending) {
      return "saving";
    }
    if (subtitleSaveError.trim()) {
      return "error";
    }
    if (rowsDirty && editingRowId) {
      return "editing";
    }
    if (rowsDirty) {
      return "dirty";
    }
    return "saved";
  }, [editingRowId, rowsDirty, saveSubtitle.isPending, subtitleSaveError]);

  const handleSelectRow = React.useCallback((rowId: string) => {
    setSelectedRowId(rowId);
    const row = rowsRef.current.find((item) => item.id == rowId);
    if (row) {
      setPlayheadMs(row.startMs);
    }
  }, []);

  const handleSeek = React.useCallback(
    (value: number) => {
      const next = Math.round(clampMs(value, durationMs));
      setPlayheadMs((current) => (current == next ? current : next));
    },
    [durationMs],
  );

  const handleEditSourceText = React.useCallback(
    (rowId: string, value: string) => {
      applyRows((current) =>
        current.map((row) =>
          row.id == rowId ? { ...row, sourceText: value } : row,
        ),
      );
    },
    [applyRows],
  );

  const persistSubtitleDraft = React.useCallback(
    async (silent = false) => {
      if (!activeSubtitleFile || !rowsDirty) {
        return true;
      }
      try {
        setSubtitleSaveError("");
        await saveSubtitle.mutateAsync({
          fileId: activeSubtitleFile.id,
          documentId:
            activeSubtitleFile.storage.documentId?.trim() || undefined,
          path: activeSubtitleFile.storage.localPath?.trim() || undefined,
          format: subtitleFormat,
          document: buildSubtitleDocument(
            rowsRef.current,
            subtitleFormat,
            activeSubtitleDocument,
          ),
        });
        setActiveSubtitleDocument(
          buildSubtitleDocument(
            rowsRef.current,
            subtitleFormat,
            activeSubtitleDocument,
          ),
        );
        const latestRows = rowsRef.current;
        baseRowsRef.current = latestRows;
        setBaseRows(latestRows);
        setSubtitleSaveError("");
        if (!silent) {
          messageBus.publishToast({
            intent: "success",
            title: t("library.workspace.notifications.subtitleSavedTitle"),
            description: t("library.workspace.notifications.subtitleSavedDescription"),
          });
        }
        return true;
      } catch (error) {
        setSubtitleSaveError(
          resolveErrorMessage(
            error,
            t("library.errors.unknown"),
          ),
        );
        messageBus.publishToast({
          intent: "danger",
          title: t("library.workspace.notifications.saveFailedTitle"),
          description: resolveErrorMessage(
            error,
            t("library.errors.unknown"),
          ),
        });
        return false;
      }
    },
    [
      activeSubtitleDocument,
      activeSubtitleFile,
      rowsDirty,
      saveSubtitle,
      subtitleFormat,
      t,
    ],
  );

  const handleEditingRowChange = React.useCallback(
    async (nextRowId: string) => {
      const previousEditingRowId = editingRowId;
      if (previousEditingRowId === nextRowId) {
        return;
      }
      if (!previousEditingRowId) {
        setEditingRowId(nextRowId);
        return;
      }

      setEditingRowId("");
      const persisted = await persistSubtitleDraft(true);
      if (!persisted) {
        setEditingRowId(previousEditingRowId);
        const previousRow = rowsRef.current.find(
          (row) => row.id === previousEditingRowId,
        );
        if (previousRow) {
          setSelectedRowId(previousRow.id);
          setPlayheadMs(previousRow.startMs);
        }
        return;
      }
      setEditingRowId(nextRowId);
    },
    [editingRowId, persistSubtitleDraft],
  );

  const handleEditorChange = React.useCallback(
    async (nextEditor: LibraryWorkspaceEditor) => {
      if (nextEditor === activeEditor) {
        return;
      }
      if (activeEditor === "subtitle" && editingRowId) {
        const previousEditingRowId = editingRowId;
        setEditingRowId("");
        const persisted = await persistSubtitleDraft(true);
        if (!persisted) {
          setEditingRowId(previousEditingRowId);
          return;
        }
      }
      setActiveEditor(nextEditor);
    },
    [activeEditor, editingRowId, persistSubtitleDraft, setActiveEditor],
  );

  const handleActiveSubtitleTrackChange = React.useCallback(
    async (nextTrackId: string) => {
      if (nextTrackId === (activeSubtitleFile?.id ?? "")) {
        return;
      }
      if (editingRowId) {
        const previousEditingRowId = editingRowId;
        setEditingRowId("");
        const persisted = await persistSubtitleDraft(true);
        if (!persisted) {
          setEditingRowId(previousEditingRowId);
          return;
        }
      }
      setActiveSubtitleFileId(nextTrackId);
    },
    [activeSubtitleFile?.id, editingRowId, persistSubtitleDraft, setActiveSubtitleFileId],
  );

  const handleApplyReplace = React.useCallback(() => {
    if (subtitleReviewBlocked) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.review.lockedTitle"),
        description: subtitleActionLockDescription,
      });
      return;
    }
    const search = subtitleSearch.trim();
    const replace = subtitleReplaceValue.trim();
    if (!search || !replace) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.subtitle.replace"),
        description: t("library.workspace.subtitle.validateHint"),
      });
      return;
    }
    const visibleIds = new Set(filteredSubtitleRows.map((row) => row.id));
    applyRows((current) =>
      current.map((row) => {
        if (!visibleIds.has(row.id)) {
          return row;
        }
        return {
          ...row,
          sourceText: row.sourceText.split(search).join(replace),
        };
      }),
    );
  }, [
    applyRows,
    filteredSubtitleRows,
    subtitleReplaceValue,
    subtitleSearch,
    subtitleActionLockDescription,
    subtitleReviewBlocked,
    t,
  ]);

  const notifyBlockedByPendingReview = React.useCallback(
    (actionLabel: string) => {
      messageBus.publishToast({
        intent: "warning",
        title: actionLabel,
        description: subtitleActionLockDescription,
      });
    },
    [subtitleActionLockDescription],
  );

  const handleReviewDecisionChange = React.useCallback(
    (cueIndex: number, decision: WorkspaceReviewDecision) => {
      setReviewDecisions((current) => {
        const next = { ...current };
        if (decision === "undecided") {
          delete next[cueIndex];
        } else {
          next[cueIndex] = decision;
        }
        return next;
      });
    },
    [],
  );

  const handleApplyPendingReview = React.useCallback(async () => {
    if (!pendingReview?.sessionId || !reviewSession) {
      return;
    }
    const decisions: SubtitleReviewCueDecisionDTO[] = (
      reviewSession.suggestions ?? []
    ).map((suggestion) => ({
      cueIndex: suggestion.cueIndex,
      action: reviewDecisions[suggestion.cueIndex] ?? "reject",
    }));
    try {
      const result = await applySubtitleReview.mutateAsync({
        sessionId: pendingReview.sessionId,
        decisions,
      });
      setReloadRevision((value) => value + 1);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.review.applySuccessTitle"),
        description: t("library.workspace.review.applySuccessDescription").replace(
          "{count}",
          String(result.changedCueCount),
        ),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.review.applyFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [applySubtitleReview, pendingReview?.sessionId, reviewDecisions, reviewSession, t]);

  const handleBatchAcceptPendingReview = React.useCallback(() => {
    if (reviewSuggestions.length === 0) {
      return;
    }
    setReviewDecisions(
      Object.fromEntries(
        reviewSuggestions.map((suggestion) => [suggestion.cueIndex, "accept"]),
      ),
    );
  }, [reviewSuggestions]);

  const handleBatchRejectPendingReview = React.useCallback(() => {
    if (reviewSuggestions.length === 0) {
      return;
    }
    setReviewDecisions(
      Object.fromEntries(
        reviewSuggestions.map((suggestion) => [suggestion.cueIndex, "reject"]),
      ),
    );
  }, [reviewSuggestions]);

  const handleDiscardPendingReview = React.useCallback(async () => {
    if (!pendingReview?.sessionId) {
      return;
    }
    try {
      await discardSubtitleReview.mutateAsync({
        sessionId: pendingReview.sessionId,
      });
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.review.discardSuccessTitle"),
        description: t("library.workspace.review.discardSuccessDescription"),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.review.discardFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [discardSubtitleReview, pendingReview?.sessionId, t]);

  const handleCompletePendingReview = React.useCallback(async () => {
    if (!pendingReview?.sessionId) {
      return;
    }
    if (reviewDecisionSummary.unresolvedCount > 0) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.review.completeBlockedTitle"),
        description: t("library.workspace.review.completeBlockedDescription"),
      });
      return;
    }
    if (reviewDecisionSummary.acceptedCount === 0) {
      await handleDiscardPendingReview();
      return;
    }
    await handleApplyPendingReview();
  }, [
    handleApplyPendingReview,
    handleDiscardPendingReview,
    pendingReview?.sessionId,
    reviewDecisionSummary.acceptedCount,
    reviewDecisionSummary.unresolvedCount,
    t,
  ]);

  const handleRestoreOriginal = React.useCallback(async () => {
    if (!activeSubtitleFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noSubtitleTrackTitle"),
        description: t("library.workspace.notifications.restoreTrackRequiredDescription"),
      });
      return;
    }
    try {
      await restoreSubtitleOriginal.mutateAsync({
        fileId: activeSubtitleFile.id,
        documentId: activeSubtitleFile.storage.documentId?.trim() || undefined,
        path: activeSubtitleFile.storage.localPath?.trim() || undefined,
      });
      setReloadRevision((value) => value + 1);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.originalRestoredTitle"),
        description: t("library.workspace.notifications.originalRestoredDescription"),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.restoreFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [activeSubtitleFile, restoreSubtitleOriginal, t]);

  const handleOpenCurrentFile = React.useCallback(async () => {
    const file =
      activeEditor == "video"
        ? (activeVideoFile ?? activeSubtitleFile)
        : (activeSubtitleFile ?? activeVideoFile);
    if (!file) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noFileSelectedTitle"),
        description: t("library.workspace.notifications.noFileSelectedDescription"),
      });
      return;
    }
    try {
      await openFileLocation.mutateAsync({ fileId: file.id });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.openFileFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [activeEditor, activeSubtitleFile, activeVideoFile, openFileLocation, t]);

  const handleExportSubtitle = React.useCallback(() => {
    if (exportBlocked) {
      notifyBlockedByPendingReview(
        t("library.workspace.exportSubtitle"),
      );
      return;
    }
    if (!activeSubtitleFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.exportSubtitle"),
        description: t("library.workspace.dialogs.exportSubtitle.trackRequired"),
      });
      return;
    }
    const exportMediaHint = resolveSubtitleExportMediaHint(
      activeVideoFile ?? headerMediaVideoFile,
      activeSubtitleFile,
      props.files,
    );
    const baseFormat = "srt";
    const baseConfig = buildDefaultSubtitleExportConfig(
      library?.name ?? "",
      exportMediaHint,
    );
    const selectedPreset = resolveSubtitleExportPresetSelection(
      subtitleExportPresets,
      subtitleStyleDefaults,
      baseFormat,
      exportMediaHint,
    );
    const nextFormat = baseFormat;
    const nextConfig = mergeSubtitleExportConfig(
      baseConfig,
      resolveSubtitleExportPresetOverrideConfig(selectedPreset?.preset),
    );
    setExportSubtitlePresetId(selectedPreset?.preset.id ?? "");
    setExportSubtitleFormat(nextFormat);
    setExportSubtitleConfig(nextConfig);
    setExportSubtitleDialogOpen(true);
  }, [
    activeSubtitleFile,
    activeVideoFile,
    headerMediaVideoFile,
    library?.name,
    props.files,
    subtitleExportPresets,
    subtitleStyleDefaults,
    exportBlocked,
    notifyBlockedByPendingReview,
    t,
  ]);

  const handleChangeExportSubtitleConfig = React.useCallback(
    (nextConfig: SubtitleExportConfig) => {
      setExportSubtitlePresetId("");
      setExportSubtitleConfig(nextConfig);
    },
    [],
  );

  const handleChangeExportSubtitlePreset = React.useCallback(
    (presetId: string) => {
      const normalizedID = presetId.trim();
      if (!normalizedID) {
        setExportSubtitlePresetId("");
        return;
      }
      const preset = subtitleExportPresets.find(
        (candidate) => candidate.id === normalizedID,
      );
      if (!preset) {
        setExportSubtitlePresetId("");
        return;
      }
      const exportMediaHint = resolveSubtitleExportMediaHint(
        activeVideoFile ?? headerMediaVideoFile,
        activeSubtitleFile,
        props.files,
      );
      const baseConfig = buildDefaultSubtitleExportConfig(
        library?.name ?? "",
        exportMediaHint,
      );
      setExportSubtitlePresetId(preset.id);
      setExportSubtitleFormat(resolveSubtitleExportPresetFormat(preset));
      setExportSubtitleConfig(
        mergeSubtitleExportConfig(
          baseConfig,
          resolveSubtitleExportPresetOverrideConfig(preset),
        ),
      );
    },
    [
      activeSubtitleFile,
      activeVideoFile,
      headerMediaVideoFile,
      library?.name,
      props.files,
      subtitleExportPresets,
    ],
  );

  const handleChangeExportSubtitleFormat = React.useCallback(
    (nextFormat: string) => {
      const normalizedFormat = normalizeSubtitleExportFormat(nextFormat);
      const exportMediaHint = resolveSubtitleExportMediaHint(
        activeVideoFile ?? headerMediaVideoFile,
        activeSubtitleFile,
        props.files,
      );
      const baseConfig = buildDefaultSubtitleExportConfig(
        library?.name ?? "",
        exportMediaHint,
      );
      const selectedPreset = resolveSubtitleExportPresetSelection(
        subtitleExportPresets,
        subtitleStyleDefaults,
        normalizedFormat,
        exportMediaHint,
      );
      setExportSubtitleFormat(normalizedFormat);
      setExportSubtitlePresetId(selectedPreset?.preset.id ?? "");
      setExportSubtitleConfig(
        mergeSubtitleExportConfig(
          baseConfig,
          resolveSubtitleExportPresetOverrideConfig(selectedPreset?.preset),
        ),
      );
    },
    [
      activeSubtitleFile,
      activeVideoFile,
      headerMediaVideoFile,
      library?.name,
      props.files,
      subtitleExportPresets,
      subtitleStyleDefaults,
    ],
  );

  const handleSubmitExportSubtitle = React.useCallback(async () => {
    if (exportBlocked) {
      notifyBlockedByPendingReview(
        t("library.workspace.exportSubtitle"),
      );
      return;
    }
    if (!activeSubtitleFile) {
      return;
    }
    const sourcePath = activeSubtitleFile.storage.localPath?.trim() ?? "";
    const fallbackName = resourceName || activeSubtitleFile.name || "subtitle";
    const filename = `${stripFileExtension(fallbackName) || "subtitle"}.${exportSubtitleFormat}`;

    let exportPath = "";
    try {
      exportPath =
        (
          await Dialogs.SaveFile({
            Title: t("library.workspace.exportSubtitle"),
            Message: t("library.workspace.dialogs.exportSubtitle.saveDescription"),
            ButtonText: t("library.workspace.exportSubtitle"),
            Filename: filename,
            Directory: resolveDirectoryName(sourcePath),
            CanChooseDirectories: false,
            CanChooseFiles: true,
            AllowsOtherFiletypes: false,
            Filters: [
              {
                DisplayName: t("library.workspace.dialogs.exportSubtitle.fileType"),
                Pattern: `*.${exportSubtitleFormat}`,
              },
            ],
          })
        )?.trim?.() ?? "";
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.exportSubtitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
      return;
    }
    if (!exportPath) {
      return;
    }

    try {
      const normalizedTargetFormat = normalizeSubtitleExportFormat(
        exportSubtitleFormat,
      );
      const styleDocumentContent =
        normalizedTargetFormat === "srt"
          ? ""
          : await resolveWorkspaceStyleDocumentContent(effectiveDisplayMode);
      const exportDocumentRows =
        effectiveDisplayMode === "bilingual"
          ? resolvedRows.map((row) => ({
              ...row,
              sourceText: [row.sourceText, row.translationText]
                .map((value) => value.trim())
                .filter(Boolean)
                .join("\n"),
            }))
          : resolvedRows;
      const exportDocument = buildSubtitleDocument(
        exportDocumentRows,
        subtitleFormat,
        activeSubtitleDocument,
      );
      if (effectiveDisplayMode === "bilingual") {
        exportDocument.metadata = {
          ...(exportDocument.metadata ?? {}),
          dreamcreatorExportDisplayMode: "bilingual",
          dreamcreatorExportPrimaryTexts: resolvedRows.map((row) => row.sourceText),
          dreamcreatorExportSecondaryTexts: resolvedRows.map((row) => row.translationText),
        };
      }
      const result = await exportSubtitle.mutateAsync({
        exportPath,
        fileId: activeSubtitleFile.id,
        documentId: activeSubtitleFile.storage.documentId?.trim() || undefined,
        format: subtitleFormat,
        targetFormat: exportSubtitleFormat,
        styleDocumentContent,
        exportConfig: exportSubtitleConfig,
        document: exportDocument,
      });
      setExportSubtitleDialogOpen(false);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.exportSubtitle"),
        description: `${t("library.workspace.subtitle.exported")} ${result.format.toUpperCase()}.`,
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.exportSubtitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    activeSubtitleDocument,
    activeSubtitleFile,
    exportBlocked,
    exportSubtitle,
    exportSubtitleConfig,
    exportSubtitleFormat,
    effectiveDisplayMode,
    notifyBlockedByPendingReview,
    resolveWorkspaceStyleDocumentContent,
    resolvedRows,
    resourceName,
    subtitleFormat,
    t,
  ]);

  const handleExportVideo = React.useCallback(() => {
    if (exportBlocked) {
      notifyBlockedByPendingReview(
        t("library.workspace.exportVideo"),
      );
      return;
    }
    if (!activeVideoFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noVideoSelectedTitle"),
        description: t("library.workspace.notifications.exportVideoTrackRequiredDescription"),
      });
      return;
    }
    setExportVideoSubtitleHandling(resolvedRows.length > 0 ? "embed" : "none");
    setExportVideoDialogOpen(true);
  }, [activeVideoFile, exportBlocked, notifyBlockedByPendingReview, resolvedRows.length, t]);

  const handleRequestOpenCurrentFile = React.useCallback(() => {
    void handleOpenCurrentFile();
  }, [handleOpenCurrentFile]);

  const toolbarState = React.useMemo<LibraryWorkspaceToolbarState>(
    () => ({
      activeEditor,
      canExportSubtitle: rows.length > 0 && !exportBlocked,
      canExportVideo: Boolean(activeVideoFile) && !exportBlocked,
      exportDisabledReason: exportBlocked ? subtitleActionLockDescription : "",
      canOpenCurrentFile: Boolean(
        activeEditor == "video"
          ? (activeVideoFile ?? activeSubtitleFile)
          : (activeSubtitleFile ?? activeVideoFile),
      ),
      onExportSubtitle: handleExportSubtitle,
      onExportVideo: handleExportVideo,
      onOpenCurrentFile: handleRequestOpenCurrentFile,
    }),
    [
      activeEditor,
      activeSubtitleFile,
      activeVideoFile,
      handleExportSubtitle,
      handleExportVideo,
      handleRequestOpenCurrentFile,
      rows.length,
      exportBlocked,
      subtitleActionLockDescription,
    ],
  );

  React.useEffect(() => {
    if (!props.onToolbarStateChange) {
      return;
    }
    props.onToolbarStateChange(library ? toolbarState : null);
  }, [library, props.onToolbarStateChange, toolbarState]);

  React.useEffect(() => {
    return () => {
      props.onToolbarStateChange?.(null);
    };
  }, [props.onToolbarStateChange]);

  const handleQueueExportVideo = React.useCallback(async () => {
    if (exportBlocked) {
      notifyBlockedByPendingReview(
        t("library.workspace.exportVideo"),
      );
      return;
    }
    if (!activeVideoFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noVideoSelectedTitle"),
        description: t("library.workspace.notifications.exportVideoTrackRequiredDescription"),
      });
      return;
    }
    if (!exportVideoPresetId) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noPresetSelectedTitle"),
        description: t("library.workspace.notifications.exportVideoPresetRequiredDescription"),
      });
      return;
    }
    const subtitlesEnabled = exportVideoSubtitleHandling !== "none";
    const generatedSubtitleFormat = !subtitlesEnabled
      ? ""
      : exportVideoSubtitleHandling === "burnin"
        ? "ass"
        : exportEmbeddedSubtitleRoute.format;
    let generatedSubtitleStyleDocumentContent: string | undefined;
    let generatedSubtitleDocument: SubtitleDocument | undefined;
    let generatedSubtitleContent = "";
    if (subtitlesEnabled) {
      if (
        exportVideoSubtitleHandling === "burnin" ||
        exportEmbeddedSubtitleRoute.format === "ass"
      ) {
        generatedSubtitleStyleDocumentContent =
          await resolveWorkspaceStyleDocumentContent(effectiveDisplayMode);
        generatedSubtitleDocument = buildSubtitleDocument(
          resolvedRows,
          subtitleFormat,
          activeSubtitleDocument,
        );
        if (effectiveDisplayMode === "bilingual") {
          generatedSubtitleDocument.metadata = {
            ...(generatedSubtitleDocument.metadata ?? {}),
            dreamcreatorExportDisplayMode: "bilingual",
            dreamcreatorExportPrimaryTexts: resolvedRows.map(
              (row) => row.sourceText,
            ),
            dreamcreatorExportSecondaryTexts: resolvedRows.map(
              (row) => row.translationText,
            ),
          };
        }
      } else {
        generatedSubtitleContent = buildTextSubtitleContent({
          rows: resolvedRows,
          displayMode: effectiveDisplayMode,
          format:
            exportEmbeddedSubtitleRoute.format === "vtt" ? "vtt" : "srt",
        });
      }
    }
    const hasStructuredASSPayload =
      generatedSubtitleFormat === "ass" &&
      generatedSubtitleDocument != null &&
      generatedSubtitleDocument.cues.length > 0;
    if (
      subtitlesEnabled &&
      !hasStructuredASSPayload &&
      !generatedSubtitleContent.trim()
    ) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.emptySubtitleOutputTitle"),
        description: t("library.workspace.notifications.emptySubtitleOutputDescription"),
      });
      return;
    }
    try {
      const operation = await createTranscode.mutateAsync({
        fileId: activeVideoFile.id,
        libraryId: activeVideoFile.libraryId,
        rootFileId: activeVideoFile.lineage.rootFileId || activeVideoFile.id,
        presetId: exportVideoPresetId,
        source: "workspace",
        subtitleHandling: exportVideoSubtitleHandling,
        subtitleFileId: activeSubtitleFile?.id || undefined,
        secondarySubtitleFileId: comparisonSubtitleFileId || undefined,
        displayMode: effectiveDisplayMode,
        subtitleDocumentId: undefined,
        generatedSubtitleFormat: subtitlesEnabled
          ? generatedSubtitleFormat
          : undefined,
        generatedSubtitleName: subtitlesEnabled
          ? `${resourceName || activeVideoFile.name} subtitles`
          : undefined,
        generatedSubtitleStyleDocumentContent: hasStructuredASSPayload
          ? generatedSubtitleStyleDocumentContent
          : undefined,
        generatedSubtitleDocument: hasStructuredASSPayload
          ? generatedSubtitleDocument
          : undefined,
        generatedSubtitleContent:
          subtitlesEnabled && generatedSubtitleContent.trim()
          ? generatedSubtitleContent
          : undefined,
      });
      setExportVideoDialogOpen(false);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.exportQueuedTitle"),
        description: t("library.workspace.notifications.exportQueuedDescription").replace("{name}", operation.displayName),
      });
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.exportFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    activeSubtitleDocument,
    activeSubtitleFile?.id,
    activeVideoFile,
    comparisonSubtitleFileId,
    createTranscode,
    exportEmbeddedSubtitleRoute.format,
    exportBlocked,
    exportVideoPresetId,
    exportVideoSubtitleHandling,
    effectiveDisplayMode,
    notifyBlockedByPendingReview,
    resolveWorkspaceStyleDocumentContent,
    resolvedRows,
    resourceName,
    subtitleFormat,
    t,
  ]);

  const handleNormalizationOptionChange = React.useCallback(
    (key: keyof WorkspaceImportNormalizationOptions, value: boolean) => {
      setNormalizationOptions((current) => ({ ...current, [key]: value }));
    },
    [],
  );

  const handleProofreadOptionChange = React.useCallback(
    (key: keyof WorkspaceProofreadOptions, value: boolean) => {
      setProofreadOptions((current) => ({ ...current, [key]: value }));
    },
    [],
  );
  const handleQaCheckToggle = React.useCallback(
    (id: WorkspaceQaCheckId, value: boolean) => {
      setQaCheckEnabled(id, value);
    },
    [setQaCheckEnabled],
  );

  const handleToggleTranslateGlossaryProfile = React.useCallback(
    (value: string, checked: boolean) => {
      setTranslateGlossaryProfileIds((current) =>
        toggleSelectedId(current, value, checked),
      );
    },
    [],
  );

  const handleToggleTranslatePromptProfile = React.useCallback(
    (value: string, checked: boolean) => {
      setTranslatePromptProfileIds((current) =>
        toggleSelectedId(current, value, checked),
      );
    },
    [],
  );

  const handleToggleProofreadGlossaryProfile = React.useCallback(
    (value: string, checked: boolean) => {
      setProofreadGlossaryProfileIds((current) =>
        toggleSelectedId(current, value, checked),
      );
    },
    [],
  );

  const handleToggleProofreadPromptProfile = React.useCallback(
    (value: string, checked: boolean) => {
      setProofreadPromptProfileIds((current) =>
        toggleSelectedId(current, value, checked),
      );
    },
    [],
  );

  const resetTranslateTaskDraft = React.useCallback(() => {
    setTranslateGlossaryProfileIds([]);
    setTranslateReferenceTrackId("");
    setTranslatePromptProfileIds([]);
    setTranslateInlinePrompt("");
    setTranslatePromptProfileName("");
  }, []);

  const resetProofreadTaskDraft = React.useCallback(() => {
    setProofreadGlossaryProfileIds([]);
    setProofreadPromptProfileIds([]);
    setProofreadInlinePrompt("");
    setProofreadPromptProfileName("");
  }, []);

  const handleOpenTranslateDialog = React.useCallback(() => {
    if (blockedActions.has("translate")) {
      notifyBlockedByPendingReview(
        t("library.workspace.actions.translate"),
      );
      return;
    }
    setLanguageTaskMode("translate");
    resetTranslateTaskDraft();
    setLanguageTaskDialogOpen(true);
  }, [blockedActions, notifyBlockedByPendingReview, resetTranslateTaskDraft, t]);

  const handleOpenProofreadDialog = React.useCallback(() => {
    if (blockedActions.has("proofread")) {
      notifyBlockedByPendingReview(
        t("library.workspace.header.proofread"),
      );
      return;
    }
    setLanguageTaskMode("proofread");
    resetProofreadTaskDraft();
    setLanguageTaskDialogOpen(true);
  }, [blockedActions, notifyBlockedByPendingReview, resetProofreadTaskDraft, t]);

  const handleOpenQaDialog = React.useCallback(() => {
    setLanguageTaskMode("qa");
    setLanguageTaskDialogOpen(true);
  }, []);

  const handleOpenRestoreDialog = React.useCallback(() => {
    setLanguageTaskMode("restore");
    setLanguageTaskDialogOpen(true);
  }, []);

  const handleOpenTranslateSettings = React.useCallback(() => {
    void Events.Emit("settings:navigate", translateReadiness.actionTarget);
  }, [translateReadiness.actionTarget]);

  const handleSaveTranslatePromptProfile = React.useCallback(async () => {
    const prompt = translateInlinePrompt.trim();
    if (!prompt) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.inlinePromptEmptyTitle"),
        description: t("library.workspace.notifications.translatePromptRequiredDescription"),
      });
      return;
    }
    if (!props.moduleConfig) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.moduleConfigUnavailableTitle"),
        description: t("library.workspace.notifications.moduleConfigUnavailableDescription"),
      });
      return;
    }

    const nextProfileId = createWorkspacePromptProfileId("prompt");
    const nextProfile = {
      id: nextProfileId,
      name: derivePromptProfileName(
        translatePromptProfileName,
        prompt,
        t("library.workspace.dialogs.languageTask.translatePromptFallback"),
      ),
      category: "all",
      description: "",
      prompt,
    };
    const nextConfig: LibraryModuleConfigDTO = {
      ...props.moduleConfig,
      languageAssets: {
        ...props.moduleConfig.languageAssets,
        promptProfiles: [
          ...(props.moduleConfig.languageAssets.promptProfiles ?? []),
          nextProfile,
        ],
      },
    };

    props.onModuleConfigChange?.(nextConfig);
    try {
      const savedConfig = await updateModuleConfig.mutateAsync({
        config: nextConfig,
      });
      props.onModuleConfigChange?.(savedConfig);
      setTranslatePromptProfileIds((current) =>
        toggleSelectedId(current, nextProfileId, true),
      );
      setTranslateInlinePrompt("");
      setTranslatePromptProfileName("");
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.promptProfileSavedTitle"),
        description: t("library.workspace.notifications.translatePromptProfileSavedDescription").replace("{name}", nextProfile.name),
      });
    } catch (error) {
      props.onModuleConfigChange?.(props.moduleConfig);
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.savePromptProfileFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    props,
    t,
    translateInlinePrompt,
    translatePromptProfileName,
    updateModuleConfig,
  ]);

  const handleSaveProofreadPromptProfile = React.useCallback(async () => {
    const prompt = proofreadInlinePrompt.trim();
    if (!prompt) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.inlinePromptEmptyTitle"),
        description: t("library.workspace.notifications.proofreadPromptRequiredDescription"),
      });
      return;
    }
    if (!props.moduleConfig) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.moduleConfigUnavailableTitle"),
        description: t("library.workspace.notifications.moduleConfigUnavailableDescription"),
      });
      return;
    }

    const nextProfileId = createWorkspacePromptProfileId("prompt");
    const nextProfile = {
      id: nextProfileId,
      name: derivePromptProfileName(
        proofreadPromptProfileName,
        prompt,
        t("library.workspace.dialogs.languageTask.proofreadPromptFallback"),
      ),
      category: "all",
      description: "",
      prompt,
    };
    const nextConfig: LibraryModuleConfigDTO = {
      ...props.moduleConfig,
      languageAssets: {
        ...props.moduleConfig.languageAssets,
        promptProfiles: [
          ...(props.moduleConfig.languageAssets.promptProfiles ?? []),
          nextProfile,
        ],
      },
    };

    props.onModuleConfigChange?.(nextConfig);
    try {
      const savedConfig = await updateModuleConfig.mutateAsync({
        config: nextConfig,
      });
      props.onModuleConfigChange?.(savedConfig);
      setProofreadPromptProfileIds((current) =>
        toggleSelectedId(current, nextProfileId, true),
      );
      setProofreadInlinePrompt("");
      setProofreadPromptProfileName("");
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.promptProfileSavedTitle"),
        description: t("library.workspace.notifications.proofreadPromptProfileSavedDescription").replace("{name}", nextProfile.name),
      });
    } catch (error) {
      props.onModuleConfigChange?.(props.moduleConfig);
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.savePromptProfileFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    proofreadInlinePrompt,
    proofreadPromptProfileName,
    props,
    t,
    updateModuleConfig,
  ]);

  const handleQueueTranslate = React.useCallback(async () => {
    if (!activeSubtitleFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noSubtitleTrackTitle"),
        description: t("library.workspace.notifications.translateTrackRequiredDescription"),
      });
      return;
    }
    if (blockedActions.has("translate")) {
      notifyBlockedByPendingReview(
        t("library.workspace.actions.translate"),
      );
      return;
    }
    if (!translateReadiness.ready) {
      messageBus.publishToast({
        intent: "warning",
        title: translateReadiness.title,
        description: translateReadiness.description,
      });
      return;
    }
    const persisted = await persistSubtitleDraft(true);
    if (!persisted) {
      return;
    }
    try {
      const operation = await createSubtitleTranslate.mutateAsync({
        fileId: activeSubtitleFile.id,
        documentId: activeSubtitleFile.storage.documentId?.trim() || undefined,
        path: activeSubtitleFile.storage.localPath?.trim() || undefined,
        libraryId: activeSubtitleFile.libraryId,
        rootFileId:
          activeSubtitleFile.lineage.rootFileId || activeSubtitleFile.id,
        assistantId: translateReadiness.assistantId || undefined,
        targetLanguage: translateTargetLanguage.trim() || "en",
        outputFormat: subtitleFormat,
        mode: "translate",
        source: "workspace",
        glossaryProfileIds: translateGlossaryProfileIds,
        referenceTrackFileIds: translateReferenceTrackId
          ? [translateReferenceTrackId]
          : [],
        promptProfileIds: translatePromptProfileIds,
        inlinePrompt: translateInlinePrompt.trim() || undefined,
      });
      setPendingTranslateOperationId(operation.id);
      const selectedConstraintCount =
        translateGlossaryProfileIds.length +
        translatePromptProfileIds.length +
        (translateReferenceTrackId ? 1 : 0) +
        (translateInlinePrompt.trim() ? 1 : 0);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.translationQueuedTitle"),
        description:
          selectedConstraintCount > 0
            ? t("library.workspace.notifications.translationQueuedWithConstraintsDescription")
                .replace("{language}", translateTargetLanguage.toUpperCase())
                .replace("{count}", String(selectedConstraintCount))
            : t("library.workspace.notifications.translationQueuedDescription").replace("{language}", translateTargetLanguage.toUpperCase()),
      });
      setLanguageTaskDialogOpen(false);
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.translationFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    activeSubtitleFile,
    blockedActions,
    createSubtitleTranslate,
    notifyBlockedByPendingReview,
    persistSubtitleDraft,
    subtitleFormat,
    t,
    translateReadiness.assistantId,
    translateReadiness.description,
    translateReadiness.ready,
    translateReadiness.title,
    translateGlossaryProfileIds,
    translateInlinePrompt,
    translatePromptProfileIds,
    translateReferenceTrackId,
    translateTargetLanguage,
  ]);

  const handleQueueProofread = React.useCallback(async () => {
    if (!activeSubtitleFile) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.notifications.noSubtitleTrackTitle"),
        description: t("library.workspace.notifications.proofreadTrackRequiredDescription"),
      });
      return;
    }
    if (blockedActions.has("proofread")) {
      notifyBlockedByPendingReview(
        t("library.workspace.header.proofread"),
      );
      return;
    }
    if (proofreadTaskRunning) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.workspace.header.proofread"),
        description: t("library.workspace.review.proofreadRunningDescription"),
      });
      return;
    }
    if (!translateReadiness.ready) {
      messageBus.publishToast({
        intent: "warning",
        title: translateReadiness.title,
        description: translateReadiness.description,
      });
      return;
    }
    const persisted = await persistSubtitleDraft(true);
    if (!persisted) {
      return;
    }
    try {
      await createSubtitleProofread.mutateAsync({
        fileId: activeSubtitleFile.id,
        documentId: activeSubtitleFile.storage.documentId?.trim() || undefined,
        path: activeSubtitleFile.storage.localPath?.trim() || undefined,
        libraryId: activeSubtitleFile.libraryId,
        rootFileId:
          activeSubtitleFile.lineage.rootFileId || activeSubtitleFile.id,
        assistantId: translateReadiness.assistantId || undefined,
        language: currentTrackLanguage || undefined,
        outputFormat: subtitleFormat,
        source: "workspace",
        spelling: proofreadOptions.spelling,
        punctuation: proofreadOptions.punctuation,
        terminology: proofreadOptions.terminology,
        glossaryProfileIds: proofreadGlossaryProfileIds,
        promptProfileIds: proofreadPromptProfileIds,
        inlinePrompt: proofreadInlinePrompt.trim() || undefined,
      });
      const enabledPasses = Object.entries(proofreadOptions)
        .filter(([, enabled]) => enabled)
        .map(([key]) =>
          key === "spelling"
            ? t("library.workspace.dialogs.languageTask.proofreadChecks.spelling")
            : key === "punctuation"
              ? t("library.workspace.dialogs.languageTask.proofreadChecks.punctuation")
              : t("library.workspace.dialogs.languageTask.proofreadChecks.terminology"),
        );
      const selectedPromptCount =
        proofreadGlossaryProfileIds.length +
        proofreadPromptProfileIds.length +
        (proofreadInlinePrompt.trim() ? 1 : 0);
      messageBus.publishToast({
        intent: "success",
        title: t("library.workspace.notifications.proofreadQueuedTitle"),
        description:
          enabledPasses.length > 0
            ? t("library.workspace.notifications.proofreadQueuedDescription")
                .replace("{checks}", enabledPasses.join(", "))
                .replace(
                  "{prompts}",
                  selectedPromptCount > 0
                    ? t("library.workspace.notifications.proofreadQueuedPromptSuffix").replace("{count}", String(selectedPromptCount))
                    : "",
                )
            : t("library.workspace.notifications.proofreadQueuedDefaultDescription"),
      });
      setLanguageTaskDialogOpen(false);
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.workspace.notifications.proofreadFailedTitle"),
        description: resolveErrorMessage(
          error,
          t("library.errors.unknown"),
        ),
      });
    }
  }, [
    activeSubtitleFile,
    blockedActions,
    createSubtitleProofread,
    currentTrackLanguage,
    notifyBlockedByPendingReview,
    persistSubtitleDraft,
    proofreadGlossaryProfileIds,
    proofreadInlinePrompt,
    proofreadOptions,
    proofreadPromptProfileIds,
    proofreadTaskRunning,
    subtitleFormat,
    t,
    translateReadiness.assistantId,
    translateReadiness.description,
    translateReadiness.ready,
    translateReadiness.title,
  ]);

  const handleConfirmImportSubtitle = React.useCallback(() => {
    const importGuideline = useCurrentGuidelineForImport
      ? guidelineProfileId
      : importGuidelineProfileId;
    const enabledNormalizations = Object.entries(normalizationOptions).filter(
      ([, enabled]) => enabled,
    ).length;

    messageBus.publishToast({
      intent: "info",
      title: t("library.workspace.notifications.importProfileReadyTitle"),
      description: t("library.workspace.notifications.importProfileReadyDescription")
        .replace("{guideline}", resolveWorkspaceGuidelineLabel(importGuideline))
        .replace("{count}", String(enabledNormalizations)),
    });
    setSubtitleImportDialogOpen(false);
    props.onRequestImportSubtitle();
  }, [
    guidelineProfileId,
    importGuidelineProfileId,
    normalizationOptions,
    props,
    t,
    useCurrentGuidelineForImport,
  ]);

  if (!library) {
    return (
      <div className="min-h-0 flex-1">
        <div className="flex h-full items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center">
          <Empty className="max-w-lg">
            <EmptyHeader>
              <EmptyMedia className="flex h-14 w-14 items-center justify-center rounded-full border border-border/70 bg-background/80 text-muted-foreground">
                <FolderOpen className="h-6 w-6" />
              </EmptyMedia>
              <EmptyTitle>{workspaceEmptyTexts.title}</EmptyTitle>
              <EmptyDescription>
                {workspaceEmptyTexts.description}
              </EmptyDescription>
            </EmptyHeader>
          </Empty>
        </div>
      </div>
    );
  }

  const workspaceSubtitleStyleSidebar = (
    <WorkspaceSubtitleStyleCard
      displayMode={effectiveDisplayMode}
      monoStyle={activeWorkspaceMonoStyle}
      lingualStyle={activeWorkspaceLingualStyle}
      fontMappings={workspaceFontMappings}
      monoStyles={props.moduleConfig?.subtitleStyles?.monoStyles ?? []}
      lingualStyles={props.moduleConfig?.subtitleStyles?.bilingualStyles ?? []}
      monoStyleOptions={monoStyleOptions}
      lingualStyleOptions={lingualStyleOptions}
      onMonoStyleChange={(next) =>
        setWorkspaceSubtitleMonoStyle(
          createWorkspaceMonoStyleDraft(next) ?? null,
        )
      }
      onLingualStyleChange={(next) =>
        setWorkspaceSubtitleLingualStyle(
          createWorkspaceLingualStyleDraft(next) ?? null,
        )
      }
      onApplyTemplate={(kind, styleId) => {
        if (kind === "mono") {
          const next = props.moduleConfig?.subtitleStyles?.monoStyles?.find(
            (item) => item.id === styleId,
          );
          if (next) {
            setWorkspaceSubtitleMonoStyle(
              createWorkspaceMonoStyleDraft(next) ?? null,
            );
          }
          return;
        }
        const next =
          props.moduleConfig?.subtitleStyles?.bilingualStyles?.find(
            (item) => item.id === styleId,
          );
        if (next) {
          setWorkspaceSubtitleLingualStyle(
            createWorkspaceLingualStyleDraft(next) ?? null,
          );
        }
      }}
      onSaveAs={(kind, name) => {
        const safeName = name.trim();
        if (!props.moduleConfig) {
          return;
        }
        if (!safeName) {
          return;
        }
        if (kind === "mono" && activeWorkspaceMonoStyle) {
          const current = props.moduleConfig.subtitleStyles.monoStyles ?? [];
          const nextStyle = {
            ...activeWorkspaceMonoStyle,
            id: createSubtitleStylePresetID("mono"),
            name: safeName,
          };
          const nextConfig = {
            ...props.moduleConfig,
            subtitleStyles: {
              ...props.moduleConfig.subtitleStyles,
              monoStyles: [...current, nextStyle],
            },
          };
          props.onModuleConfigChange?.(nextConfig);
          void updateModuleConfig.mutateAsync({ config: nextConfig }).then((saved) => {
            props.onModuleConfigChange?.(saved);
          });
          return;
        }
        if (kind === "lingual" && activeWorkspaceLingualStyle) {
          const current =
            props.moduleConfig.subtitleStyles.bilingualStyles ?? [];
          const nextStyle = {
            ...activeWorkspaceLingualStyle,
            id: createSubtitleStylePresetID("bilingual"),
            name: safeName,
          };
          const nextConfig = {
            ...props.moduleConfig,
            subtitleStyles: {
              ...props.moduleConfig.subtitleStyles,
              bilingualStyles: [...current, nextStyle],
            },
          };
          props.onModuleConfigChange?.(nextConfig);
          void updateModuleConfig.mutateAsync({ config: nextConfig }).then((saved) => {
            props.onModuleConfigChange?.(saved);
          });
        }
      }}
    />
  );

  return (
    <div className="relative flex h-full min-h-0 flex-1 flex-col overflow-hidden rounded-[20px] border border-border/70 bg-[linear-gradient(180deg,hsl(var(--background)),hsl(var(--background)))] shadow-[0_20px_65px_-42px_rgba(15,23,42,0.38)]">
      <WorkspaceHeader
        libraryName={
          library.name?.trim() ||
          t("library.workspace.page.workspaceName")
        }
        metaItems={workspaceHeaderMeta}
        hasPendingReview={Boolean(pendingReview)}
        pendingReviewButtonLabel={pendingReviewButtonLabel}
        pendingReviewMenuDescription={pendingReviewMenuDescription}
        reviewCompletionReady={reviewCompletionReady}
        reviewApplying={applySubtitleReview.isPending}
        reviewDiscarding={discardSubtitleReview.isPending}
        activeEditor={activeEditor}
        onEditorChange={(value) => {
          void handleEditorChange(value);
        }}
        activeVideoId={activeVideoFile?.id ?? ""}
        videoOptions={videoOptions}
        onVideoChange={setActiveVideoFileId}
        primarySubtitleTrackId={activeSubtitleFile?.id ?? ""}
        subtitleTrackOptions={subtitleTrackOptions}
        onPrimarySubtitleTrackChange={(value) => {
          void handleActiveSubtitleTrackChange(value);
        }}
        secondarySubtitleTrackId={comparisonSubtitleFileId}
        comparisonTrackOptions={comparisonTrackOptions}
        onSecondarySubtitleTrackChange={setComparisonSubtitleFileId}
        displayMode={displayMode}
        canUseDualDisplay={canUseDualDisplay}
        dualDisplayDisabledReason={
          blockedActions.has("proofread") || blockedActions.has("qa")
            ? subtitleActionLockDescription
            : ""
        }
        onDisplayModeChange={setDisplayMode}
        autoFollow={autoFollow}
        onAutoFollowChange={setAutoFollow}
        subtitleStyleSidebarOpen={subtitleStyleSidebarOpen}
        onSubtitleStyleSidebarOpenChange={setSubtitleStyleSidebarOpen}
        guidelineProfileId={guidelineProfileId}
        guidelineOptions={guidelineOptions}
        onGuidelineChange={setGuidelineProfileId}
        canSubtitleActions={rows.length > 0}
        canTranslateAction={canTranslateAction}
        canProofreadAction={canProofreadAction}
        canQaAction={canQaAction}
        canRestoreAction={canRestoreAction}
        translateDisabledReason={
          blockedActions.has("translate") ? subtitleActionLockDescription : ""
        }
        proofreadDisabledReason={
          blockedActions.has("proofread")
            ? subtitleActionLockDescription
            : ""
        }
        qaDisabledReason=""
        translateButtonLabel={translateButtonLabel}
        proofreadButtonLabel={proofreadButtonLabel}
        qaButtonLabel={qaButtonLabel}
        translateRunning={translateTaskRunning}
        proofreadRunning={proofreadTaskRunning}
        qaRunning={false}
        onRunQa={() => {
          handleOpenQaDialog();
        }}
        onTranslate={handleOpenTranslateDialog}
        onProofread={handleOpenProofreadDialog}
        onRestore={handleOpenRestoreDialog}
        onCompleteReview={() => {
          void handleCompletePendingReview();
        }}
        onBatchAcceptReview={handleBatchAcceptPendingReview}
        onBatchRejectReview={handleBatchRejectPendingReview}
      />

      <div className="min-h-0 flex-1 overflow-hidden p-3">
        {activeEditor == "video" ? (
          <VideoEditorLayout
            mediaUrl={videoPreviewUrl}
            mediaType={videoPreviewMimeType}
            waveformDisabledReason={waveformDisabledReason}
            rows={filteredVideoRows}
            selectedRowId={selectedRowId}
            currentRowId={currentRowId}
            hoveredRowId={hoveredRowId}
            displayMode={effectiveDisplayMode}
            density={videoDensity}
            searchValue={videoSearch}
            qaFilter={videoQaFilter}
            autoFollow={autoFollow}
            playheadMs={playheadMs}
            durationMs={durationMs}
            isPlaying={isPlaying}
            previewVttContent={previewVttContent}
            previewMonoStyle={activeWorkspaceMonoStyle}
            previewLingualStyle={activeWorkspaceLingualStyle}
            previewFontMappings={workspaceFontMappings}
            onPreviewRenderSizeChange={handlePreviewRenderSizeChange}
            showStyleSidebar={subtitleStyleSidebarOpen}
            styleSidebarContent={workspaceSubtitleStyleSidebar}
            isLoading={subtitleLoading}
            errorMessage={subtitleError}
            onDensityChange={setVideoDensity}
            onSearchChange={setVideoSearch}
            onQaFilterChange={setVideoQaFilter}
            onSelectRow={handleSelectRow}
            onHoverRow={setHoveredRowId}
            onSeek={handleSeek}
            onPlayingChange={setIsPlaying}
          />
        ) : (
          <SubtitleEditorLayout
            rows={filteredSubtitleRows}
            primaryTrackLabel={
              activeTrack?.label ??
              t("library.workspace.page.currentSubtitleTrack")
            }
            secondaryTrackLabel={comparisonTrack?.label || ""}
            selectedRowId={selectedRowId}
            editingRowId={editingRowId}
            reviewPending={actionablePendingReview}
            reviewApplying={applySubtitleReview.isPending}
            saveState={subtitleDraftSaveState}
            saveErrorMessage={subtitleSaveError}
            currentRowId={currentRowId}
            hoveredRowId={hoveredRowId}
            displayMode={effectiveDisplayMode}
            density={subtitleDensity}
            searchValue={subtitleSearch}
            replaceValue={subtitleReplaceValue}
            filterValue={subtitleFilter}
            qaFilter={subtitleQaFilter}
            isLoading={subtitleLoading}
            errorMessage={subtitleError}
            showStyleSidebar={subtitleStyleSidebarOpen}
            styleSidebarContent={workspaceSubtitleStyleSidebar}
            editingEnabled={!subtitleReviewBlocked}
            qaCheckSettings={qaCheckSettings}
            onSearchChange={setSubtitleSearch}
            onReplaceValueChange={setSubtitleReplaceValue}
            onApplyReplace={handleApplyReplace}
            onFilterChange={setSubtitleFilter}
            onQaFilterChange={setSubtitleQaFilter}
            onDensityChange={setSubtitleDensity}
            onSelectRow={handleSelectRow}
            onEditingRowChange={(rowId) => {
              void handleEditingRowChange(rowId);
            }}
            onHoverRow={setHoveredRowId}
            onEditSourceText={handleEditSourceText}
            onReviewDecisionChange={handleReviewDecisionChange}
          />
        )}
      </div>

      <WorkspaceActionDialogs
        cueCount={resolvedRows.length}
        importSubtitleOpen={subtitleImportDialogOpen}
        onImportSubtitleOpenChange={setSubtitleImportDialogOpen}
        useCurrentGuidelineForImport={useCurrentGuidelineForImport}
        onUseCurrentGuidelineForImportChange={setUseCurrentGuidelineForImport}
        importGuidelineProfileId={importGuidelineProfileId}
        onImportGuidelineProfileIdChange={setImportGuidelineProfileId}
        normalizationOptions={normalizationOptions}
        onNormalizationOptionChange={handleNormalizationOptionChange}
        guidelineOptions={guidelineOptions}
        guidelineProfileId={guidelineProfileId}
        onGuidelineChange={setGuidelineProfileId}
        activeGuidelineLabel={activeGuidelineLabel}
        onConfirmImportSubtitle={handleConfirmImportSubtitle}
        qaSummary={qaSummary}
        qaCheckDefinitions={qaCheckDefinitions}
        qaCheckSettings={qaCheckSettings}
        onQaCheckToggle={handleQaCheckToggle}
        languageTaskOpen={languageTaskDialogOpen}
        onLanguageTaskOpenChange={setLanguageTaskDialogOpen}
        languageTaskMode={languageTaskMode}
        onLanguageTaskModeChange={setLanguageTaskMode}
        translateLanguageOptions={translateLanguageOptions}
        translateTargetLanguage={translateTargetLanguage}
        onTranslateTargetLanguageChange={setTranslateTargetLanguage}
        translateGlossaryOptions={translateGlossaryOptions}
        translateGlossaryProfileIds={translateGlossaryProfileIds}
        onTranslateGlossaryProfileToggle={handleToggleTranslateGlossaryProfile}
        referenceTrackOptions={referenceTrackOptions}
        translateReferenceTrackId={translateReferenceTrackId}
        onTranslateReferenceTrackIdChange={setTranslateReferenceTrackId}
        translatePromptOptions={translatePromptOptions}
        translatePromptProfileIds={translatePromptProfileIds}
        onTranslatePromptProfileToggle={handleToggleTranslatePromptProfile}
        translateInlinePrompt={translateInlinePrompt}
        onTranslateInlinePromptChange={setTranslateInlinePrompt}
        translatePromptProfileName={translatePromptProfileName}
        onTranslatePromptProfileNameChange={setTranslatePromptProfileName}
        onSaveTranslatePromptProfile={() => {
          void handleSaveTranslatePromptProfile();
        }}
        translateReady={translateReadiness.ready}
        translateReadinessChecking={translateReadiness.checking}
        translateReadinessTitle={translateReadiness.title}
        translateReadinessDescription={translateReadiness.description}
        translateTaskRunning={translateTaskRunning}
        translateTaskLabel={translateButtonLabel}
        translateActionDisabled={blockedActions.has("translate")}
        translateDisabledReason={
          blockedActions.has("translate") ? subtitleActionLockDescription : ""
        }
        onOpenTranslateSettings={handleOpenTranslateSettings}
        proofreadReady={translateReadiness.ready}
        proofreadReadinessChecking={translateReadiness.checking}
        proofreadReadinessTitle={translateReadiness.title}
        proofreadReadinessDescription={translateReadiness.description}
        proofreadTaskRunning={proofreadTaskRunning}
        proofreadTaskLabel={proofreadButtonLabel}
        proofreadActionDisabled={blockedActions.has("proofread")}
        proofreadDisabledReason={
          blockedActions.has("proofread")
            ? subtitleActionLockDescription
            : ""
        }
        onOpenProofreadSettings={handleOpenTranslateSettings}
        proofreadOptions={proofreadOptions}
        onProofreadOptionChange={handleProofreadOptionChange}
        proofreadGlossaryOptions={proofreadGlossaryOptions}
        proofreadGlossaryProfileIds={proofreadGlossaryProfileIds}
        onProofreadGlossaryProfileToggle={handleToggleProofreadGlossaryProfile}
        proofreadPromptOptions={proofreadPromptOptions}
        proofreadPromptProfileIds={proofreadPromptProfileIds}
        onProofreadPromptProfileToggle={handleToggleProofreadPromptProfile}
        proofreadInlinePrompt={proofreadInlinePrompt}
        onProofreadInlinePromptChange={setProofreadInlinePrompt}
        proofreadPromptProfileName={proofreadPromptProfileName}
        onProofreadPromptProfileNameChange={setProofreadPromptProfileName}
        onSaveProofreadPromptProfile={() => {
          void handleSaveProofreadPromptProfile();
        }}
        primaryTrackOptions={subtitleTrackOptions}
        primaryTrackId={activeSubtitleFile?.id ?? ""}
        onPrimaryTrackChange={setActiveSubtitleFileId}
        primaryTrackLabel={
          activeTrack?.label ??
          t("library.workspace.page.currentSubtitleTrack")
        }
        onQueueTranslate={handleQueueTranslate}
        onQueueProofread={handleQueueProofread}
        restoreOriginalRunning={restoreSubtitleOriginal.isPending}
        onRestoreOriginal={handleRestoreOriginal}
      />

      <WorkspaceExportVideoDialog
        open={exportVideoDialogOpen}
        onOpenChange={setExportVideoDialogOpen}
        resourceName={resourceName}
        cueCount={resolvedRows.length}
        subtitleFormat={subtitleFormat}
        presetOptions={videoPresetOptions}
        presetId={exportVideoPresetId}
        onPresetIdChange={setExportVideoPresetId}
        subtitleHandling={exportVideoSubtitleHandling}
        onSubtitleHandlingChange={setExportVideoSubtitleHandling}
        displayMode={displayMode}
        canUseDualDisplay={canUseDualDisplay}
        dualDisplayDisabledReason={
          blockedActions.has("proofread") || blockedActions.has("qa")
            ? subtitleActionLockDescription
            : ""
        }
        onDisplayModeChange={setDisplayMode}
        primaryTrackOptions={subtitleTrackOptions}
        primaryTrackId={activeSubtitleFile?.id ?? ""}
        onPrimaryTrackIdChange={setActiveSubtitleFileId}
        secondaryTrackOptions={comparisonTrackOptions}
        secondaryTrackId={comparisonSubtitleFileId}
        onSecondaryTrackIdChange={setComparisonSubtitleFileId}
        embeddedSubtitleFormat={exportEmbeddedSubtitleRoute.format}
        embeddedSubtitleLabel={`${exportEmbeddedSubtitleRoute.label} / ${exportEmbeddedSubtitleRoute.codec}`}
        isSubmitting={createTranscode.isPending}
        disabled={exportBlocked}
        disabledReason={exportBlocked ? subtitleActionLockDescription : ""}
        onSubmit={() => {
          void handleQueueExportVideo();
        }}
      />

      <WorkspaceExportSubtitleDialog
        open={exportSubtitleDialogOpen}
        onOpenChange={setExportSubtitleDialogOpen}
        resourceName={resourceName}
        trackLabel={
          activeTrack?.label ??
          t("library.workspace.page.currentSubtitleTrack")
        }
        cueCount={resolvedRows.length}
        currentFormat={subtitleFormat}
        displayMode={effectiveDisplayMode}
        monoStyle={activeWorkspaceMonoStyle}
        lingualStyle={activeWorkspaceLingualStyle}
        targetFormat={exportSubtitleFormat}
        onTargetFormatChange={handleChangeExportSubtitleFormat}
        presetOptions={subtitleExportPresets}
        selectedPresetId={exportSubtitlePresetId}
        onSelectedPresetIdChange={handleChangeExportSubtitlePreset}
        exportConfig={exportSubtitleConfig}
        onExportConfigChange={handleChangeExportSubtitleConfig}
        ittLanguageOptions={subtitleExportLanguageOptions}
        hasUnsavedChanges={rowsDirty}
        isSubmitting={exportSubtitle.isPending}
        disabled={exportBlocked}
        disabledReason={exportBlocked ? subtitleActionLockDescription : ""}
        onSubmit={() => {
          void handleSubmitExportSubtitle();
        }}
      />

      {restoreSubtitleOriginal.isPending ||
      exportSubtitle.isPending ||
      createTranscode.isPending ||
      createSubtitleTranslate.isPending ||
      createSubtitleProofread.isPending ||
      applySubtitleReview.isPending ||
      discardSubtitleReview.isPending ||
      updateModuleConfig.isPending ? (
        <div className="pointer-events-none absolute bottom-5 right-5 z-30 flex items-center gap-2 rounded-full border border-border/70 bg-background/92 px-3 py-1.5 text-xs text-muted-foreground shadow-lg backdrop-blur">
          <Loader2 className="h-3.5 w-3.5 animate-spin" />
          <span>
            {t("library.workspace.page.actionInProgress")}
          </span>
        </div>
      ) : null}
    </div>
  );
}
