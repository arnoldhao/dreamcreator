import * as React from "react"
import { Call, Dialogs } from "@wailsio/runtime"
import type { ColumnDef, RowSelectionState, Updater, VisibilityState } from "@tanstack/react-table"
import {
  Activity,
  AlertTriangle,
  ArrowRight,
  Captions,
  CheckCircle2,
  ChevronDown,
  Clapperboard,
  Database,
  Download,
  FilePlus2,
  FolderOpen,
  Image as ImageIcon,
  LayoutGrid,
  List,
  ListChecks,
  Loader2,
  PencilLine,
  RefreshCcw,
  Search,
  Settings2,
  SlidersHorizontal,
  Sparkles,
  Trash2,
  Video,
  Zap,
} from "lucide-react"

import { setPendingSettingsSection } from "@/app/settings/sectionStorage"
import { Progress } from "@/shared/ui/progress"
import defaultThumbnail from "@/shared/assets/default-thumbnail.png"
import { useI18n } from "@/shared/i18n"
import { messageBus } from "@/shared/message"
import {
  useCreateSubtitleImport,
  useCreateSubtitleTranslateJob,
  useCreateTranscodeJob,
  useCreateVideoImport,
  useDeleteFile,
  useDeleteFiles,
  useCreateYtdlpJob,
  useDeleteLibrary,
  useDeleteOperation,
  useDeleteOperations,
  useGetLibraryModuleConfig,
  useGetLibrary,
  useGetWorkspaceProject,
  useListLibraries,
  useListOperations,
  useOpenLibraryPath,
  useParseYtdlpDownload,
  usePrepareYtdlpDownload,
  useRenameLibrary,
  useResolveDomainIcon,
  useTranscodePresets,
  useTranscodePresetsForDownload,
  useUpdateLibraryModuleConfig,
} from "@/shared/query/library"
import {
  useExternalToolInstallState,
  useExternalTools,
  useInstallExternalTool,
} from "@/shared/query/externalTools"
import { useShowSettingsWindow } from "@/shared/query/settings"
import type { ExternalTool } from "@/shared/store/externalTools"
import type {
  CreateSubtitleImportRequest,
  CreateVideoImportRequest,
  LibraryDTO,
  LibraryFileDTO,
  LibraryHistoryRecordDTO,
  LibraryModuleConfigDTO,
  OperationListItemDTO,
  TranscodePreset,
  WorkspaceProjectDTO,
  YtdlpFormatOption,
  YtdlpSubtitleOption,
} from "@/shared/contracts/library"
import { openTaskDialog } from "@/shared/store/taskDialog"
import { useLibraryRealtimeStore, toOperationListItem } from "@/shared/store/libraryRealtime"
import { Badge } from "@/shared/ui/badge"
import { Button } from "@/shared/ui/button"
import { Card } from "@/shared/ui/card"
import {
  DASHBOARD_CONTROL_GROUP_CLASS,
  DASHBOARD_PANEL_SOLID_SURFACE_CLASS,
  PanelCard,
} from "@/shared/ui/dashboard"
import {
  DashboardDialogBody,
  DashboardDialogContent,
  DashboardDialogFooter,
  DashboardDialogHeader,
  DashboardDialogSection,
} from "@/shared/ui/dashboard-dialog"
import {
  Dialog,
  DialogDescription,
  DialogTitle,
} from "@/shared/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuCheckboxItem,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/shared/ui/dropdown-menu"
import { Input } from "@/shared/ui/input"
import { Separator } from "@/shared/ui/separator"
import { Select } from "@/shared/ui/select"
import { Switch } from "@/shared/ui/switch"
import { Tabs, TabsList, TabsTrigger } from "@/shared/ui/tabs"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/shared/ui/tooltip"
import { cn } from "@/lib/utils"
import { resolveToolDependencyIssues } from "@/features/setup/readiness"

import { LibraryOverviewPage } from "./components/LibraryOverviewPage"
import {
  LibraryConfigPage,
  type LibraryConfigPageId,
  type LibraryConfigToolbarState,
} from "./components/LibraryConfigPage"
import { getFileColumns } from "./components/LibraryTableFileColumns"
import { getTaskColumns } from "./components/LibraryTableTaskColumns"
import { LibraryTable } from "./components/LibraryTable"
import { LibraryTableSelectionCheckbox } from "./components/LibraryTableSelectionCheckbox"
import { LibraryWorkspacePage, type LibraryWorkspaceToolbarState } from "./components/LibraryWorkspacePage"
import {
  WorkspaceDialogFormRow,
  WorkspaceDialogHeaderCard,
  WorkspaceDialogItemsCard,
  WorkspaceDialogMetricsCard,
  WorkspaceDialogSectionBadge,
  WorkspaceDialogSectionCard,
  WorkspaceDialogSummaryRow,
} from "./components/workspace/WorkspaceDashboardDialog"
import { useLibraryViewStore } from "./model/viewStore"
import { openLibraryWorkspace, useLibraryWorkspaceStore } from "./model/workspaceStore"
import type { LibraryFileRow, LibraryProgress, LibraryTaskOutput, LibraryTaskRow, LibraryWorkspaceTarget } from "./model/types"
import { formatBytes } from "./utils/format"
import { formatTemplate } from "./utils/i18n"
import { formatDuration, formatRelativeTime } from "./utils/time"
import { resolvePresetName } from "./utils/transcodePresets"

type LibraryNewAction = "download" | "importVideo" | "importSubtitle"
type LibraryPageTab = "overview" | "tasks" | "resources" | "workspace" | "config"
type ResourceViewMode = "library" | "file"
type ResourceFileTypeFilter = "all" | "video" | "subtitle"
type ResourceFileStatusFilter = "active" | "deleted" | "all"
type TaskStatusFilter = "queued" | "running" | "succeeded" | "failed" | "canceled"
type TaskTableFilters = {
  statuses: TaskStatusFilter[]
  taskTypes: string[]
}
type ImportTargetMode = "new" | "existing"
type DependencyIssue = { name: string; status: "missing" | "invalid" }
type LibraryImagePreview = { id: string; name: string; path: string }
type Translator = (key: string) => string

type LibraryLabelMaps = {
  typeLabels: Record<string, string>
  jobTypeLabels: Record<string, string>
  originLabels: Record<string, string>
}

const VIDEO_IMPORT_EXTENSIONS = [
  "mp4",
  "mkv",
  "mov",
  "webm",
  "avi",
  "m4v",
  "mpg",
  "mpeg",
  "ts",
  "m2ts",
  "mts",
  "wmv",
  "flv",
  "3gp",
  "ogv",
]

const SUBTITLE_IMPORT_EXTENSIONS = ["srt", "vtt", "webvtt", "ass", "ssa", "itt", "ttml", "xml", "dfxp", "fcpxml"]
const RESOURCE_LIBRARY_GRID_BATCH_SIZE = 12
const RESOURCE_RECORD_BATCH_SIZE = 10

export function LibraryPage() {
  const { t, language } = useI18n()
  const librariesQuery = useListLibraries()
  const selectedLibraryRealtimeFiles = useLibraryRealtimeStore((state) => state.files)
  const selectedLibraryRealtimeHistory = useLibraryRealtimeStore((state) => state.histories)
  const selectedLibraryRealtimeOperations = useLibraryRealtimeStore((state) => state.operations)
  const workspaceOpenRevision = useLibraryWorkspaceStore((state) => state.openRevision)
  const workspaceTargetLibraryId = useLibraryWorkspaceStore((state) => state.libraryId)

  const externalTools = useExternalTools()
  const showSettingsWindow = useShowSettingsWindow()
  const createYtdlp = useCreateYtdlpJob()
  const prepareYtdlp = usePrepareYtdlpDownload()
  const parseYtdlp = useParseYtdlpDownload()
  const resolveDomainIcon = useResolveDomainIcon()
  const createSubtitleImport = useCreateSubtitleImport()
  const createVideoImport = useCreateVideoImport()
  const createTranscode = useCreateTranscodeJob()
  const createSubtitleTranslate = useCreateSubtitleTranslateJob()
  const deleteFile = useDeleteFile()
  const deleteFiles = useDeleteFiles()
  const deleteOperation = useDeleteOperation()
  const deleteOperations = useDeleteOperations()
  const openLibraryPath = useOpenLibraryPath()
  const presetsQuery = useTranscodePresets()
  const moduleConfigQuery = useGetLibraryModuleConfig()
  const updateModuleConfig = useUpdateLibraryModuleConfig()

  const rowsPerPage = useLibraryViewStore((state) => state.rowsPerPage)
  const columnVisibility = useLibraryViewStore((state) => state.columnVisibility)
  const setRowsPerPage = useLibraryViewStore((state) => state.setRowsPerPage)
  const setColumnVisibility = useLibraryViewStore((state) => state.setColumnVisibility)

  const [pageTab, setPageTab] = React.useState<LibraryPageTab>("overview")
  const [resourceViewMode, setResourceViewMode] = React.useState<ResourceViewMode>("library")
  const [resourceFileTypeFilter, setResourceFileTypeFilter] = React.useState<ResourceFileTypeFilter>("all")
  const [resourceFileStatusFilter, setResourceFileStatusFilter] = React.useState<ResourceFileStatusFilter>("active")
  const [resourceFocusedLibraryId, setResourceFocusedLibraryId] = React.useState("")
  const [taskStatusFilters, setTaskStatusFilters] = React.useState<TaskStatusFilter[]>([])
  const [taskTypeFilters, setTaskTypeFilters] = React.useState<string[]>([])
  const [searchQuery, setSearchQuery] = React.useState("")
  const [taskSelectionMode, setTaskSelectionMode] = React.useState(false)
  const [taskRowSelection, setTaskRowSelection] = React.useState<RowSelectionState>({})
  const [fileSelectionMode, setFileSelectionMode] = React.useState(false)
  const [fileRowSelection, setFileRowSelection] = React.useState<RowSelectionState>({})
  const [selectedLibraryId, setSelectedLibraryId] = React.useState("")
  const [chartGranularity, setChartGranularity] = React.useState("7d")
  const translateLanguage = "en"
  const [activeNewDialog, setActiveNewDialog] = React.useState<LibraryNewAction | null>(null)
  const [downloadStep, setDownloadStep] = React.useState<"dependency" | "input" | "config">("dependency")
  const [downloadTab, setDownloadTab] = React.useState<"quick" | "custom">("quick")
  const [downloadUrl, setDownloadUrl] = React.useState("")
  const [downloadPrepared, setDownloadPrepared] = React.useState<ReturnType<typeof usePrepareYtdlpDownload>["data"] | null>(null)
  const [downloadUseConnector, setDownloadUseConnector] = React.useState(false)
  const [downloadDependencyIssues, setDownloadDependencyIssues] = React.useState<DependencyIssue[]>([])
  const [downloadPrepareError, setDownloadPrepareError] = React.useState("")
  const [downloadSubmitError, setDownloadSubmitError] = React.useState("")
  const [quickQuality, setQuickQuality] = React.useState<"best" | "audio">("best")
  const [quickSubtitle, setQuickSubtitle] = React.useState(true)
  const [quickPresetId, setQuickPresetId] = React.useState("")
  const [quickDeleteSourceAfterTranscode, setQuickDeleteSourceAfterTranscode] = React.useState(true)
  const [customParseResult, setCustomParseResult] = React.useState<ReturnType<typeof useParseYtdlpDownload>["data"] | null>(null)
  const [customParseError, setCustomParseError] = React.useState("")
  const [customFormatId, setCustomFormatId] = React.useState("")
  const [customSubtitleId, setCustomSubtitleId] = React.useState("")
  const [customPresetId, setCustomPresetId] = React.useState("")
  const [customDeleteSourceAfterTranscode, setCustomDeleteSourceAfterTranscode] = React.useState(true)
  const [libraryConfigRequestedPage, setLibraryConfigRequestedPage] = React.useState<LibraryConfigPageId | null>(null)
  const [subtitlePath, setSubtitlePath] = React.useState("")
  const [subtitleTitle, setSubtitleTitle] = React.useState("")
  const [importVideoPath, setImportVideoPath] = React.useState("")
  const [importVideoTitle, setImportVideoTitle] = React.useState("")
  const [importTargetMode, setImportTargetMode] = React.useState<ImportTargetMode>("new")
  const [importTargetLibraryID, setImportTargetLibraryID] = React.useState("")
  const [imagePreview, setImagePreview] = React.useState<LibraryImagePreview | null>(null)
  const [httpBaseURL, setHttpBaseURL] = React.useState("")
  const [moduleConfigDraft, setModuleConfigDraft] = React.useState<LibraryModuleConfigDTO | null>(null)
  const [workspaceToolbarState, setWorkspaceToolbarState] = React.useState<LibraryWorkspaceToolbarState | null>(null)
  const [configToolbarState, setConfigToolbarState] = React.useState<LibraryConfigToolbarState | null>(null)
  const moduleConfigDraftRef = React.useRef<LibraryModuleConfigDTO | null>(null)
  const lastAttemptedModuleConfigSignatureRef = React.useRef("")
  const queuedModuleConfigPersistRef = React.useRef(false)

  const selectedLibraryQuery = useGetLibrary(
    selectedLibraryId,
    pageTab === "workspace" || activeNewDialog === "importVideo" || activeNewDialog === "importSubtitle",
  )
  const operationsQuery = useListOperations({})

  const quickMediaType = quickQuality === "audio" ? "audio" : "video"
  const quickPresetsQuery = useTranscodePresetsForDownload({ mediaType: quickMediaType })
  const quickPresets = React.useMemo(() => {
    const presets = quickPresetsQuery.data ?? []
    if (quickMediaType === "audio") {
      return presets.filter((preset) => preset.outputType === "audio")
    }
    return presets.filter((preset) => preset.outputType !== "audio")
  }, [quickMediaType, quickPresetsQuery.data])

  const customFormats = customParseResult?.formats ?? []
  const customSubtitles = customParseResult?.subtitles ?? []
  const customVideoFormats = React.useMemo(() => customFormats.filter((format) => format.hasVideo), [customFormats])
  const customAudioFormats = React.useMemo(
    () => customFormats.filter((format) => format.hasAudio && !format.hasVideo),
    [customFormats],
  )
  const customSelectedFormat = React.useMemo(
    () => customFormats.find((format) => format.id === customFormatId) ?? null,
    [customFormatId, customFormats],
  )
  const customSelectedSubtitle = React.useMemo(
    () => customSubtitles.find((subtitle) => subtitle.id === customSubtitleId) ?? null,
    [customSubtitleId, customSubtitles],
  )
  const customMediaType = customSelectedFormat ? (customSelectedFormat.hasVideo ? "video" : "audio") : null
  const customPresetsQuery = useTranscodePresetsForDownload(
    customMediaType ? { mediaType: customMediaType } : null,
  )
  const customPresets = React.useMemo(() => {
    const presets = customPresetsQuery.data ?? []
    if (!customMediaType) {
      return []
    }
    if (customMediaType === "audio") {
      return presets.filter((preset) => preset.outputType === "audio")
    }
    return presets.filter((preset) => preset.outputType !== "audio")
  }, [customMediaType, customPresetsQuery.data])

  const openVideoExportPresetConfig = React.useCallback(() => {
    setLibraryConfigRequestedPage("video-export-presets")
    setPageTab("config")
  }, [])

  const currentViewMode = pageTab === "tasks" ? "task" : "file"
  const defaultVisibility = React.useMemo<VisibilityState>(() => {
    const base: VisibilityState = {
      library: false,
      source: false,
      domain: false,
      action: false,
      outputCount: false,
      outputSize: false,
      status: false,
      progress: false,
      duration: false,
      startedAt: false,
      platform: false,
      uploader: false,
      publishTime: false,
      createdAt: false,
      size: false,
      task: false,
      fileFormat: false,
    }
    if (currentViewMode === "task") {
      return {
        ...base,
        library: false,
        action: true,
        status: true,
        progress: true,
        outputCount: true,
        outputSize: true,
        createdAt: true,
        domain: false,
        duration: false,
        startedAt: false,
        source: false,
        platform: false,
        uploader: false,
        publishTime: false,
      }
    }
    return {
      ...base,
      source: true,
      status: true,
      size: true,
      task: true,
      fileFormat: true,
      createdAt: true,
    }
  }, [currentViewMode])
  const currentColumnVisibility = { ...defaultVisibility, ...(columnVisibility[currentViewMode] ?? {}) }

  const labels = React.useMemo<LibraryLabelMaps>(() => {
    const typeLabels = {
      video: t("library.type.video"),
      audio: t("library.type.audio"),
      subtitle: t("library.type.subtitle"),
      thumbnail: t("library.type.thumbnail"),
      transcode: t("library.type.transcode"),
      import: t("library.type.import"),
      manual: t("library.type.manual"),
      other: t("library.type.other"),
    }
    return {
      typeLabels,
      jobTypeLabels: {
        download: t("library.jobType.ytdlp"),
        transcode: t("library.jobType.transcode"),
        subtitle_translate: t("library.jobType.subtitleTranslate"),
        subtitle_proofread: t("library.jobType.subtitleProofread"),
        subtitle_qa_review: t("library.jobType.subtitleQaReview"),
      },
      originLabels: {
        download: typeLabels.manual,
        transcode: typeLabels.transcode,
        subtitle_translate: typeLabels.subtitle,
        subtitle_proofread: typeLabels.subtitle,
        subtitle_qa_review: typeLabels.subtitle,
        import_video: typeLabels.import,
        import_subtitle: typeLabels.import,
      },
    }
  }, [t])

  const taskStatusOptions = React.useMemo<Array<{ value: TaskStatusFilter; label: string }>>(
    () => [
      { value: "queued", label: t("library.status.queued") },
      { value: "running", label: t("library.status.running") },
      { value: "succeeded", label: t("library.status.succeeded") },
      { value: "failed", label: t("library.status.failed") },
      { value: "canceled", label: t("library.status.canceled") },
    ],
    [t],
  )

  React.useEffect(() => {
    let active = true
    Call.ByName("dreamcreator/internal/presentation/wails.RealtimeHandler.HTTPBaseURL")
      .then((url) => {
        if (!active) {
          return
        }
        const resolved = typeof url === "string" ? url : String(url ?? "")
        setHttpBaseURL(resolved)
      })
      .catch(() => {
        if (active) {
          setHttpBaseURL("")
        }
      })
    return () => {
      active = false
    }
  }, [])

  React.useEffect(() => {
    const libraries = librariesQuery.data ?? []
    if (selectedLibraryId && !libraries.some((item) => item.id === selectedLibraryId)) {
      setSelectedLibraryId("")
    }
  }, [librariesQuery.data, selectedLibraryId])

  React.useEffect(() => {
    const trimmedLibraryId = workspaceTargetLibraryId.trim()
    if (!workspaceOpenRevision || !trimmedLibraryId) {
      return
    }
    if (selectedLibraryId !== trimmedLibraryId) {
      setSelectedLibraryId(trimmedLibraryId)
    }
    setPageTab("workspace")
  }, [selectedLibraryId, workspaceOpenRevision, workspaceTargetLibraryId])

  React.useEffect(() => {
    const libraries = librariesQuery.data ?? []
    if (importTargetMode === "existing" && libraries.length === 0) {
      setImportTargetMode("new")
      setImportTargetLibraryID("")
      return
    }
    if (importTargetMode !== "existing") {
      return
    }
    if (!selectedLibraryId || !libraries.some((item) => item.id === selectedLibraryId)) {
      setImportTargetMode("new")
      setImportTargetLibraryID("")
      return
    }
    if (importTargetLibraryID !== selectedLibraryId) {
      setImportTargetLibraryID(selectedLibraryId)
    }
  }, [importTargetLibraryID, importTargetMode, librariesQuery.data, selectedLibraryId])

  React.useEffect(() => {
    if (!quickPresetId) {
      return
    }
    if (!quickPresets.some((preset) => preset.id === quickPresetId)) {
      setQuickPresetId("")
    }
  }, [quickPresetId, quickPresets])

  React.useEffect(() => {
    if (!customPresetId) {
      return
    }
    if (!customPresets.some((preset) => preset.id === customPresetId)) {
      setCustomPresetId("")
    }
  }, [customPresetId, customPresets])

  React.useEffect(() => {
    if (!customParseResult) {
      setCustomFormatId("")
      setCustomSubtitleId("")
      return
    }
    if (!customFormatId) {
      const defaultFormat = pickDefaultFormat(customParseResult.formats)
      setCustomFormatId(defaultFormat?.id ?? "")
    }
  }, [customFormatId, customParseResult])

  const lastServerConfigJSONRef = React.useRef("")
  React.useEffect(() => {
    if (!moduleConfigQuery.data) {
      return
    }
    const nextServerJSON = JSON.stringify(moduleConfigQuery.data)
    const currentDraftJSON = moduleConfigDraft ? JSON.stringify(moduleConfigDraft) : ""
    if (moduleConfigDraft === null || currentDraftJSON === lastServerConfigJSONRef.current) {
      moduleConfigDraftRef.current = moduleConfigQuery.data
      setModuleConfigDraft(moduleConfigQuery.data)
    }
    lastServerConfigJSONRef.current = nextServerJSON
  }, [moduleConfigDraft, moduleConfigQuery.data])

  React.useEffect(() => {
    moduleConfigDraftRef.current = moduleConfigDraft
  }, [moduleConfigDraft])

  const moduleConfigServerSignature = React.useMemo(
    () => (moduleConfigQuery.data ? JSON.stringify(moduleConfigQuery.data) : ""),
    [moduleConfigQuery.data],
  )
  const moduleConfigDraftSignature = React.useMemo(
    () => (moduleConfigDraft ? JSON.stringify(moduleConfigDraft) : ""),
    [moduleConfigDraft],
  )
  const moduleConfigDirty = Boolean(moduleConfigDraft) && moduleConfigDraftSignature !== moduleConfigServerSignature

  const persistModuleConfig = React.useCallback(
    async (requestConfigOverride?: LibraryModuleConfigDTO | null, requestSignatureOverride?: string) => {
      const requestConfig = requestConfigOverride ?? moduleConfigDraftRef.current ?? moduleConfigDraft
      if (!requestConfig) {
        return
      }
      const requestSignature = requestSignatureOverride ?? JSON.stringify(requestConfig)
      lastAttemptedModuleConfigSignatureRef.current = requestSignature
      try {
        const savedConfig = await updateModuleConfig.mutateAsync({ config: requestConfig })
        const latestDraft = moduleConfigDraftRef.current
        const latestDraftSignature = latestDraft ? JSON.stringify(latestDraft) : ""
        if (latestDraftSignature === requestSignature) {
          moduleConfigDraftRef.current = savedConfig
          setModuleConfigDraft(savedConfig)
        }
      } catch (error) {
        lastAttemptedModuleConfigSignatureRef.current = ""
        messageBus.publishToast({
          intent: "warning",
          title: t("library.config.saveFailedTitle"),
          description: error instanceof Error ? error.message : String(error ?? ""),
        })
      }
    },
    [moduleConfigDraft, t, updateModuleConfig],
  )

  const requestPersistModuleConfig = React.useCallback(() => {
    const latestDraft = moduleConfigDraftRef.current ?? moduleConfigDraft
    if (!latestDraft) {
      return
    }
    const persistableDraft = createPersistableModuleConfig(latestDraft)
    const latestDraftSignature = JSON.stringify(persistableDraft)
    if (updateModuleConfig.isPending) {
      queuedModuleConfigPersistRef.current = true
      return
    }
    if (latestDraftSignature === moduleConfigServerSignature) {
      lastAttemptedModuleConfigSignatureRef.current = latestDraftSignature
      queuedModuleConfigPersistRef.current = false
      return
    }
    if (lastAttemptedModuleConfigSignatureRef.current === latestDraftSignature) {
      queuedModuleConfigPersistRef.current = false
      return
    }
    queuedModuleConfigPersistRef.current = false
    void persistModuleConfig(persistableDraft, latestDraftSignature)
  }, [moduleConfigDraft, moduleConfigServerSignature, persistModuleConfig, updateModuleConfig.isPending])

  React.useEffect(() => {
    if (!queuedModuleConfigPersistRef.current || updateModuleConfig.isPending) {
      return
    }
    const latestDraft = moduleConfigDraftRef.current ?? moduleConfigDraft
    if (!latestDraft) {
      queuedModuleConfigPersistRef.current = false
      return
    }
    const persistableDraft = createPersistableModuleConfig(latestDraft)
    const latestDraftSignature = JSON.stringify(persistableDraft)
    if (latestDraftSignature === moduleConfigServerSignature) {
      lastAttemptedModuleConfigSignatureRef.current = latestDraftSignature
      queuedModuleConfigPersistRef.current = false
      return
    }
    if (lastAttemptedModuleConfigSignatureRef.current === latestDraftSignature) {
      queuedModuleConfigPersistRef.current = false
      return
    }
    queuedModuleConfigPersistRef.current = false
    void persistModuleConfig(persistableDraft, latestDraftSignature)
  }, [
    moduleConfigDraft,
    moduleConfigServerSignature,
    persistModuleConfig,
    updateModuleConfig.isPending,
  ])

  React.useEffect(() => {
    if (!moduleConfigDirty && moduleConfigDraftSignature === moduleConfigServerSignature) {
      lastAttemptedModuleConfigSignatureRef.current = moduleConfigDraftSignature
      queuedModuleConfigPersistRef.current = false
    }
  }, [moduleConfigDirty, moduleConfigDraftSignature, moduleConfigServerSignature])

  const libraryOptions = librariesQuery.data ?? []
  const moduleConfigValue = moduleConfigDraft ?? moduleConfigQuery.data ?? null
  const handleModuleConfigChange = React.useCallback((next: LibraryModuleConfigDTO) => {
    moduleConfigDraftRef.current = next
    setModuleConfigDraft(next)
  }, [])

  const previousPageTabRef = React.useRef<LibraryPageTab>(pageTab)
  React.useEffect(() => {
    if (previousPageTabRef.current === "config" && pageTab !== "config" && moduleConfigDirty) {
      requestPersistModuleConfig()
    }
    if (pageTab !== "config") {
      setConfigToolbarState(null)
    }
    previousPageTabRef.current = pageTab
  }, [moduleConfigDirty, pageTab, requestPersistModuleConfig])

  const selectedLibrary = React.useMemo(
    () =>
      selectedLibraryQuery.data ??
      libraryOptions.find((item) => item.id === selectedLibraryId),
    [libraryOptions, selectedLibraryId, selectedLibraryQuery.data],
  )

  const workspaceLiveFiles = React.useMemo(
    () => selectedLibraryRealtimeFiles.filter((item) => item.libraryId === selectedLibraryId),
    [selectedLibraryId, selectedLibraryRealtimeFiles],
  )

  const workspaceFiles = React.useMemo(
    () => mergeFiles(selectedLibrary?.files ?? [], workspaceLiveFiles),
    [selectedLibrary?.files, workspaceLiveFiles],
  )
  const activeWorkspaceFiles = React.useMemo(
    () => workspaceFiles.filter((file) => !file.state.deleted),
    [workspaceFiles],
  )

  const selectedLibraryDisplay = React.useMemo(() => {
    if (!selectedLibrary) {
      return undefined
    }
    return {
      ...selectedLibrary,
      name: resolveEffectiveLibraryName(selectedLibrary, activeWorkspaceFiles),
    }
  }, [activeWorkspaceFiles, selectedLibrary])

  const resourceLibraries = React.useMemo(() => {
    const nextLibraries = libraryOptions.map((library) => {
      const liveFiles = selectedLibraryRealtimeFiles.filter((item) => item.libraryId === library.id)
      const liveHistory = selectedLibraryRealtimeHistory.filter((item) => item.libraryId === library.id)
      const files = mergeFiles(library.files ?? [], liveFiles)
      return {
        ...library,
        name: resolveEffectiveLibraryName(library, files),
        files,
        records: {
          ...library.records,
          history: mergeHistory(library.records.history ?? [], liveHistory),
        },
      }
    })
    return sortByUpdatedAtDesc(nextLibraries)
  }, [libraryOptions, selectedLibraryRealtimeFiles, selectedLibraryRealtimeHistory])

  const libraryNameByID = React.useMemo(() => {
    const map = new Map<string, string>()
    resourceLibraries.forEach((library) => {
      map.set(library.id, library.name)
    })
    return map
  }, [resourceLibraries])

  const displayFiles = React.useMemo(
    () => mergeFiles(libraryOptions.flatMap((library) => library.files ?? []), selectedLibraryRealtimeFiles),
    [libraryOptions, selectedLibraryRealtimeFiles],
  )
  const displayHistory = React.useMemo(
    () => mergeHistory(libraryOptions.flatMap((library) => library.records.history ?? []), selectedLibraryRealtimeHistory),
    [libraryOptions, selectedLibraryRealtimeHistory],
  )
  const displayOperations = React.useMemo(
    () => mergeOperations(operationsQuery.data ?? [], selectedLibraryRealtimeOperations.map(toOperationListItem)),
    [operationsQuery.data, selectedLibraryRealtimeOperations],
  )

  const filesById = React.useMemo(() => {
    const map = new Map<string, LibraryFileDTO>()
    displayFiles.forEach((file) => {
      map.set(file.id, file)
    })
    return map
  }, [displayFiles])
  const operationsById = React.useMemo(() => {
    const map = new Map<string, OperationListItemDTO>()
    displayOperations.forEach((operation) => {
      map.set(operation.operationId, operation)
    })
    return map
  }, [displayOperations])
  const liveOperationById = React.useMemo(() => {
    const map = new Map<string, (typeof selectedLibraryRealtimeOperations)[number]>()
    selectedLibraryRealtimeOperations.forEach((operation) => {
      map.set(operation.id, operation)
    })
    return map
  }, [selectedLibraryRealtimeOperations])

  const allTaskRows = React.useMemo(() => {
    const rows = displayOperations.map((operation) =>
      toTaskRowFromOperation(
        operation,
        filesById,
        liveOperationById.get(operation.operationId),
        labels,
        libraryNameByID,
      ),
    )
    return sortByCreatedAtDesc(rows)
  }, [displayOperations, filesById, labels, libraryNameByID, liveOperationById])

  const taskTypeOptions = React.useMemo(() => {
    const map = new Map<string, string>()
    allTaskRows.forEach((task) => {
      const taskType = task.taskType.trim().toLowerCase()
      if (!taskType || map.has(taskType)) {
        return
      }
      map.set(taskType, task.taskTypeLabel?.trim() || task.taskType)
    })
    return Array.from(map.entries())
      .map(([value, label]) => ({ value, label: label || value }))
      .sort((left, right) => left.label.localeCompare(right.label))
  }, [allTaskRows])

  React.useEffect(() => {
    setTaskTypeFilters((current) => {
      const next = current.filter((value) => taskTypeOptions.some((option) => option.value === value))
      return next.length === current.length ? current : next
    })
  }, [taskTypeOptions])

  const taskRows = React.useMemo(
    () =>
      filterTasksForTable(
        allTaskRows,
        {
          statuses: taskStatusFilters,
          taskTypes: taskTypeFilters,
        },
        searchQuery,
      ),
    [allTaskRows, searchQuery, taskStatusFilters, taskTypeFilters],
  )

  const taskFilterCount = taskStatusFilters.length + taskTypeFilters.length + (searchQuery.trim() ? 1 : 0)
  const resourceFileFilterCount =
    (searchQuery.trim() ? 1 : 0) +
    (resourceFileTypeFilter !== "all" ? 1 : 0) +
    (resourceFileStatusFilter !== "active" ? 1 : 0)

  React.useEffect(() => {
    if (pageTab !== "tasks" || !taskSelectionMode) {
      setTaskRowSelection((current) => (Object.keys(current).length > 0 ? {} : current))
      return
    }
    const visibleTaskIDs = new Set(taskRows.map((task) => task.id))
    setTaskRowSelection((current) => {
      let changed = false
      const next: RowSelectionState = {}
      Object.entries(current).forEach(([taskID, selected]) => {
        if (!selected) {
          changed = true
          return
        }
        if (!visibleTaskIDs.has(taskID)) {
          changed = true
          return
        }
        next[taskID] = true
      })
      if (!changed && Object.keys(next).length === Object.keys(current).length) {
        return current
      }
      return next
    })
  }, [pageTab, taskRows, taskSelectionMode])

  const selectedTaskIDs = React.useMemo(
    () => taskRows.filter((task) => taskRowSelection[task.id]).map((task) => task.id),
    [taskRowSelection, taskRows],
  )
  const selectedTaskCount = selectedTaskIDs.length

  const fileRows = React.useMemo(() => {
    const rows = displayFiles.map((file) => toFileRowFromDTO(file, operationsById, labels))
    return filterFilesForTable(sortByCreatedAtDesc(rows), searchQuery)
  }, [displayFiles, labels, operationsById, searchQuery])

  const resourceRows = React.useMemo(
    () => filterFilesByResourceTypeAndStatus(fileRows, resourceFileTypeFilter, resourceFileStatusFilter),
    [fileRows, resourceFileStatusFilter, resourceFileTypeFilter],
  )

  React.useEffect(() => {
    if (pageTab !== "resources" || resourceViewMode !== "file" || !fileSelectionMode) {
      setFileRowSelection((current) => (Object.keys(current).length > 0 ? {} : current))
      return
    }
    const visibleFileIDs = new Set(resourceRows.map((file) => file.id))
    setFileRowSelection((current) => {
      let changed = false
      const next: RowSelectionState = {}
      Object.entries(current).forEach(([fileID, selected]) => {
        if (!selected) {
          changed = true
          return
        }
        if (!visibleFileIDs.has(fileID)) {
          changed = true
          return
        }
        next[fileID] = true
      })
      if (!changed && Object.keys(next).length === Object.keys(current).length) {
        return current
      }
      return next
    })
  }, [fileSelectionMode, pageTab, resourceRows, resourceViewMode])

  const selectedFileIDs = React.useMemo(
    () => resourceRows.filter((file) => fileRowSelection[file.id]).map((file) => file.id),
    [fileRowSelection, resourceRows],
  )
  const selectedFileCount = selectedFileIDs.length

  const filteredResourceLibraries = React.useMemo(
    () => filterLibrariesForResourceView(resourceLibraries, searchQuery, t),
    [resourceLibraries, searchQuery, t],
  )

  const focusedResourceLibrary = React.useMemo(
    () => resourceLibraries.find((item) => item.id === resourceFocusedLibraryId),
    [resourceFocusedLibraryId, resourceLibraries],
  )

  const overviewCards = React.useMemo(
    () => buildOverviewCards(taskRows, fileRows, libraryOptions.length, t),
    [fileRows, libraryOptions.length, t, taskRows],
  )
  const overviewTrendData = React.useMemo(() => buildLibraryTrendData(taskRows, chartGranularity), [chartGranularity, taskRows])

  const baseTaskColumns = React.useMemo<ColumnDef<LibraryTaskRow>[]>(
    () =>
      getTaskColumns({
        onDeleteTask: async (id, deleteFiles) => {
          try {
            await deleteOperation.mutateAsync({ operationId: id, cascadeFiles: deleteFiles })
            messageBus.publishToast({
              intent: "success",
              title: t("library.task.deleteSuccessTitle"),
              description: deleteFiles
                ? t("library.task.deleteSuccessWithFiles")
                : t("library.task.deleteSuccess"),
            })
          } catch (error) {
            messageBus.publishToast({
              intent: "danger",
              title: t("library.task.deleteFailedTitle"),
              description: resolveErrorMessage(error, t("library.errors.unknown")),
            })
            throw error
          }
        },
        onOpenTaskDialog: openTaskDialog,
        onOpenLibrary: (libraryId) => {
          const nextLibraryID = libraryId.trim()
          if (!nextLibraryID) {
            return
          }
          setSelectedLibraryId(nextLibraryID)
          setPageTab("resources")
        },
        language,
        t,
      }),
    [deleteOperation, language, t],
  )

  const taskSelectionColumn = React.useMemo<ColumnDef<LibraryTaskRow>>(
    () => ({
      id: "select",
      enableHiding: false,
      header: ({ table }) => (
        <div className="flex justify-center">
          <LibraryTableSelectionCheckbox
            aria-label={t("library.task.selectAllPage")}
            checked={table.getIsAllPageRowsSelected()}
            indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
            onChange={table.getToggleAllPageRowsSelectedHandler()}
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className="flex justify-center">
          <LibraryTableSelectionCheckbox
            aria-label={formatTemplate(t("library.task.selectRow"), {
              name: row.original.name || t("library.rowMenu.renameFallback"),
            })}
            checked={row.getIsSelected()}
            disabled={!row.getCanSelect()}
            onChange={row.getToggleSelectedHandler()}
          />
        </div>
      ),
    }),
    [t],
  )

  const taskColumns = React.useMemo<ColumnDef<LibraryTaskRow>[]>(
    () => (taskSelectionMode ? [taskSelectionColumn, ...baseTaskColumns] : baseTaskColumns),
    [baseTaskColumns, taskSelectionColumn, taskSelectionMode],
  )

  const baseFileColumns = React.useMemo<ColumnDef<LibraryFileRow>[]>(
    () =>
      getFileColumns({
        onOpenWorkspace: (target) => {
          openLibraryWorkspace(target)
        },
        onPreviewImage: (file) => {
          const path = file.path?.trim() ?? ""
          if (!path) {
            return
          }
          setImagePreview({ id: file.id, name: file.name, path })
        },
        onOpenPath: (path) => {
          if (!path) {
            return
          }
          openLibraryPath.mutate({ path })
        },
        onDeleteFile: async (id, deleteFiles) => {
          try {
            await deleteFile.mutateAsync({ fileId: id, deleteFiles })
            messageBus.publishToast({
              intent: "success",
              title: t("library.file.deleteSuccessTitle"),
              description: deleteFiles
                ? t("library.file.deleteSuccessWithLocal")
                : t("library.file.deleteSuccess"),
            })
          } catch (error) {
            messageBus.publishToast({
              intent: "danger",
              title: t("library.file.deleteFailedTitle"),
              description: resolveErrorMessage(error, t("library.errors.unknown")),
            })
            throw error
          }
        },
        onCreateTranscode: async (file) => {
          const preset = pickDefaultTranscodePreset(file, presetsQuery.data ?? [])
          if (!preset) {
            openVideoExportPresetConfig()
            messageBus.publishToast({
              intent: "warning",
              title: t("library.config.pages.videoExportPresets"),
              description: t("library.transcode.missingPreset"),
            })
            return
          }
          await createTranscode.mutateAsync({
            fileId: file.id,
            libraryId: file.libraryId,
            rootFileId: file.id,
            presetId: preset.id,
            title: `${file.name} (${resolvePresetName(preset, t)})`,
            source: "manual",
          })
          messageBus.publishToast({
            intent: "success",
            title: t("library.actions.newTranscode"),
            description: t("library.transcode.created"),
          })
        },
        onCreateSubtitleTranslate: async (file) => {
          const source = filesById.get(file.id)
          const documentId = source?.storage.documentId?.trim() ?? ""
          if (!documentId) {
            messageBus.publishToast({
              intent: "warning",
              title: t("library.actions.newSubtitleTranslate"),
              description: t("library.subtitle.documentMissing"),
            })
            return
          }
          await createSubtitleTranslate.mutateAsync({
            fileId: source?.id,
            documentId,
            libraryId: source?.libraryId,
            rootFileId: source?.lineage.rootFileId || source?.id,
            targetLanguage: translateLanguage.trim() || "en",
            source: "manual",
          })
          messageBus.publishToast({
            intent: "success",
            title: t("library.actions.newSubtitleTranslate"),
            description: t("library.subtitle.translateCreated"),
          })
        },
        onOpenTaskDialog: openTaskDialog,
        language,
        t,
      }),
    [
      createSubtitleTranslate,
      createTranscode,
      deleteFile,
      filesById,
      language,
      openLibraryPath,
      presetsQuery.data,
      t,
      translateLanguage,
    ],
  )
  const fileSelectionColumn = React.useMemo<ColumnDef<LibraryFileRow>>(
    () => ({
      id: "select",
      enableHiding: false,
      header: ({ table }) => (
        <div className="flex justify-center">
          <LibraryTableSelectionCheckbox
            aria-label={t("library.file.selectAllPage")}
            checked={table.getIsAllPageRowsSelected()}
            indeterminate={table.getIsSomePageRowsSelected() && !table.getIsAllPageRowsSelected()}
            onChange={table.getToggleAllPageRowsSelectedHandler()}
          />
        </div>
      ),
      cell: ({ row }) => (
        <div className="flex justify-center">
          <LibraryTableSelectionCheckbox
            aria-label={formatTemplate(t("library.file.selectRow"), {
              name: row.original.name || t("library.rowMenu.renameFallback"),
            })}
            checked={row.getIsSelected()}
            disabled={!row.getCanSelect()}
            onChange={row.getToggleSelectedHandler()}
          />
        </div>
      ),
    }),
    [t],
  )
  const fileColumns = React.useMemo<ColumnDef<LibraryFileRow>[]>(
    () => (fileSelectionMode ? [fileSelectionColumn, ...baseFileColumns] : baseFileColumns),
    [baseFileColumns, fileSelectionColumn, fileSelectionMode],
  )

  const taskColumnOptions = React.useMemo(() => columnsToOptions(baseTaskColumns), [baseTaskColumns])
  const fileColumnOptions = React.useMemo(() => columnsToOptions(baseFileColumns), [baseFileColumns])
  const handleColumnVisibilityChange = React.useCallback(
    (next: Updater<VisibilityState>) => {
      const resolved = typeof next === "function" ? next(currentColumnVisibility) : next
      setColumnVisibility(currentViewMode, resolved)
    },
    [currentColumnVisibility, currentViewMode, setColumnVisibility],
  )
  const isTaskColumnVisibilityDefault = React.useMemo(
    () =>
      taskColumnOptions.every(
        (column) => (currentColumnVisibility[column.id] ?? true) === (defaultVisibility[column.id] ?? true),
      ),
    [currentColumnVisibility, defaultVisibility, taskColumnOptions],
  )
  const handleResetTaskColumnVisibility = React.useCallback(() => {
    handleColumnVisibilityChange(defaultVisibility)
  }, [defaultVisibility, handleColumnVisibilityChange])
  const isFileColumnVisibilityDefault = React.useMemo(
    () =>
      fileColumnOptions.every(
        (column) => (currentColumnVisibility[column.id] ?? true) === (defaultVisibility[column.id] ?? true),
      ),
    [currentColumnVisibility, defaultVisibility, fileColumnOptions],
  )
  const handleResetFileColumnVisibility = React.useCallback(() => {
    handleColumnVisibilityChange(defaultVisibility)
  }, [defaultVisibility, handleColumnVisibilityChange])

  const handleEnterTaskSelectionMode = React.useCallback(() => {
    setTaskSelectionMode(true)
  }, [])

  const handleExitTaskSelectionMode = React.useCallback(() => {
    setTaskSelectionMode(false)
    setTaskRowSelection({})
  }, [])
  const handleEnterFileSelectionMode = React.useCallback(() => {
    setFileSelectionMode(true)
  }, [])

  const handleExitFileSelectionMode = React.useCallback(() => {
    setFileSelectionMode(false)
    setFileRowSelection({})
  }, [])

  const handleBatchDeleteTasks = React.useCallback(() => {
    if (selectedTaskIDs.length === 0) {
      return
    }
    const deletingTaskIDs = [...selectedTaskIDs]
    const deletingCount = deletingTaskIDs.length
    messageBus.publishDialog({
      intent: "danger",
      destructive: true,
      title: t("library.task.bulkDeleteTitle"),
      description: formatTemplate(
        t("library.task.bulkDeleteDescription"),
        { count: deletingCount },
      ),
      confirmLabel: formatTemplate(t("library.task.bulkDeleteConfirm"), {
        count: deletingCount,
      }),
      cancelLabel: t("library.rowMenu.deleteCancel"),
      onConfirm: async () => {
        try {
          await deleteOperations.mutateAsync({ operationIds: deletingTaskIDs, cascadeFiles: false })
          setTaskRowSelection({})
          messageBus.publishToast({
            intent: "success",
            title: t("library.task.bulkDeleteSuccessTitle"),
            description: formatTemplate(
              t("library.task.bulkDeleteSuccess"),
              { count: deletingCount },
            ),
          })
        } catch (error) {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.task.bulkDeleteFailedTitle"),
            description: resolveErrorMessage(error, t("library.errors.unknown")),
          })
        }
      },
    })
  }, [deleteOperations, selectedTaskIDs, t])
  const handleBatchDeleteFiles = React.useCallback(() => {
    if (selectedFileIDs.length === 0) {
      return
    }
    const deletingFileIDs = [...selectedFileIDs]
    const deletingCount = deletingFileIDs.length
    messageBus.publishDialog({
      intent: "danger",
      destructive: true,
      title: t("library.file.bulkDeleteTitle"),
      description: formatTemplate(
        t("library.file.bulkDeleteDescription"),
        { count: deletingCount },
      ),
      confirmLabel: formatTemplate(t("library.file.bulkDeleteConfirm"), {
        count: deletingCount,
      }),
      cancelLabel: t("library.rowMenu.deleteCancel"),
      onConfirm: async () => {
        try {
          await deleteFiles.mutateAsync({ fileIds: deletingFileIDs, deleteFiles: false })
          setFileRowSelection({})
          messageBus.publishToast({
            intent: "success",
            title: t("library.file.bulkDeleteSuccessTitle"),
            description: formatTemplate(
              t("library.file.bulkDeleteSuccess"),
              { count: deletingCount },
            ),
          })
        } catch (error) {
          messageBus.publishToast({
            intent: "danger",
            title: t("library.file.bulkDeleteFailedTitle"),
            description: resolveErrorMessage(error, t("library.errors.unknown")),
          })
        }
      },
    })
  }, [deleteFiles, selectedFileIDs, t])

  const requiredYtdlpTools = React.useMemo(() => ["yt-dlp", "ffmpeg", "bun"], [])
  const toolsByName = React.useMemo(() => {
    const map = new Map<string, ExternalTool>()
    ;(externalTools.data ?? []).forEach((tool) => {
      map.set(tool.name, tool)
    })
    return map
  }, [externalTools.data])

  const refreshExternalToolsNow = React.useCallback(async () => {
    try {
      await externalTools.refetch()
    } catch {
      // ignore refresh failures in the dialog
    }
  }, [externalTools])

  const resolveDependencyIssues = React.useCallback(
    (required: string[], sourceTools?: ExternalTool[]) =>
      resolveToolDependencyIssues(required, sourceTools ?? (externalTools.data ?? [])),
    [externalTools.data],
  )

  const handleOpenExternalTools = React.useCallback(() => {
    setPendingSettingsSection("external-tools")
    showSettingsWindow.mutate()
  }, [showSettingsWindow])

  const resetDownloadState = React.useCallback(() => {
    setDownloadStep("dependency")
    setDownloadTab("quick")
    setDownloadUrl("")
    setDownloadPrepared(null)
    setDownloadUseConnector(false)
    setDownloadDependencyIssues([])
    setDownloadPrepareError("")
    setDownloadSubmitError("")
    setQuickQuality("best")
    setQuickSubtitle(true)
    setQuickPresetId("")
    setQuickDeleteSourceAfterTranscode(true)
    setCustomParseResult(null)
    setCustomParseError("")
    setCustomFormatId("")
    setCustomSubtitleId("")
    setCustomPresetId("")
    setCustomDeleteSourceAfterTranscode(true)
  }, [])

  const previousDialogRef = React.useRef<LibraryNewAction | null>(null)
  React.useEffect(() => {
    let active = true
    if (activeNewDialog === "download" && previousDialogRef.current !== "download") {
      resetDownloadState()
      void (async () => {
        let nextTools = externalTools.data ?? []
        try {
          const refreshed = await externalTools.refetch()
          if (refreshed.data) {
            nextTools = refreshed.data
          }
        } catch {
          // ignore refresh failure here
        }
        if (!active) {
          return
        }
        const issues = resolveDependencyIssues(requiredYtdlpTools, nextTools)
        setDownloadDependencyIssues(issues)
        setDownloadStep(issues.length > 0 ? "dependency" : "input")
      })()
    }
    previousDialogRef.current = activeNewDialog
    return () => {
      active = false
    }
  }, [activeNewDialog, externalTools, requiredYtdlpTools, resetDownloadState, resolveDependencyIssues])

  React.useEffect(() => {
    if (activeNewDialog !== "download" || downloadStep !== "dependency") {
      return
    }
    const issues = resolveDependencyIssues(requiredYtdlpTools)
    setDownloadDependencyIssues((current) => (sameDependencyIssues(current, issues) ? current : issues))
    if (issues.length === 0) {
      setDownloadStep("input")
    }
  }, [activeNewDialog, downloadStep, requiredYtdlpTools, resolveDependencyIssues])

  const handlePrepareDownload = React.useCallback(async () => {
    const url = downloadUrl.trim()
    if (!url) {
      return
    }
    setDownloadPrepareError("")
    setDownloadSubmitError("")
    try {
      const response = await prepareYtdlp.mutateAsync({ url })
      setDownloadPrepared(response)
      setDownloadUseConnector(Boolean(response.connectorAvailable))
      setDownloadStep("config")
      setDownloadTab("quick")
      if (!response.icon && response.domain) {
        resolveDomainIcon.mutate(
          { domain: response.domain },
          {
            onSuccess: (iconResponse) => {
              if (!iconResponse?.icon) {
                return
              }
              setDownloadPrepared((current) => {
                if (!current || current.domain !== response.domain) {
                  return current
                }
                return { ...current, icon: iconResponse.icon }
              })
            },
          },
        )
      }
    } catch (error) {
      setDownloadPrepareError(resolveErrorMessage(error, t("library.errors.unknown")))
    }
  }, [downloadUrl, prepareYtdlp, resolveDomainIcon])

  const handleParseDownload = React.useCallback(async () => {
    if (!downloadPrepared) {
      return
    }
    setCustomParseError("")
    try {
      const response = await parseYtdlp.mutateAsync({
        url: downloadPrepared.url,
        connectorId: downloadPrepared.connectorId,
        useConnector: downloadUseConnector,
      })
      setCustomParseResult(response)
      const defaultFormat = pickDefaultFormat(response.formats)
      setCustomFormatId(defaultFormat?.id ?? "")
      setCustomSubtitleId("")
      setCustomPresetId("")
    } catch (error) {
      setCustomParseError(resolveErrorMessage(error, t("library.errors.unknown")))
    }
  }, [downloadPrepared, downloadUseConnector, parseYtdlp])

  const handleStartQuickDownload = React.useCallback(async () => {
    if (!downloadPrepared) {
      return
    }
    setDownloadSubmitError("")
    try {
      await createYtdlp.mutateAsync({
        url: downloadPrepared.url,
        source: "manual",
        mode: "quick",
        quality: quickQuality,
        writeThumbnail: true,
        subtitleAll: quickSubtitle,
        subtitleAuto: quickSubtitle,
        transcodePresetId: quickPresetId || undefined,
        deleteSourceFileAfterTranscode: quickPresetId ? quickDeleteSourceAfterTranscode : undefined,
        connectorId: downloadPrepared.connectorId,
        useConnector: downloadUseConnector && downloadPrepared.connectorAvailable,
      })
      resetDownloadState()
      setActiveNewDialog(null)
      messageBus.publishToast({
        intent: "success",
        title: t("library.download.start"),
        description: t("library.download.created"),
      })
    } catch (error) {
      setDownloadSubmitError(resolveErrorMessage(error, t("library.errors.unknown")))
    }
  }, [
    createYtdlp,
    downloadPrepared,
    downloadUseConnector,
    quickPresetId,
    quickQuality,
    quickSubtitle,
    quickDeleteSourceAfterTranscode,
    resetDownloadState,
    t,
  ])

  const handleStartCustomDownload = React.useCallback(async () => {
    if (!downloadPrepared || !customSelectedFormat) {
      return
    }
    setDownloadSubmitError("")
    const isAudioOnly = !customSelectedFormat.hasVideo
    const selectedSubtitle = customSelectedSubtitle
    const subtitleLang = selectedSubtitle?.language?.trim()
    const audioFormatId = selectAudioFormatId(customFormats)
    const needsAudioJoin = customSelectedFormat.hasVideo && !customSelectedFormat.hasAudio
    try {
      await createYtdlp.mutateAsync({
        url: downloadPrepared.url,
        source: "manual",
        mode: "custom",
        title: customParseResult?.title ?? "",
        extractor: customParseResult?.extractor ?? undefined,
        author: customParseResult?.author ?? undefined,
        thumbnailUrl: customParseResult?.thumbnailUrl ?? undefined,
        writeThumbnail: true,
        quality: isAudioOnly ? "audio" : "best",
        formatId: customSelectedFormat.id,
        audioFormatId: needsAudioJoin ? audioFormatId || "bestaudio" : undefined,
        subtitleLangs: subtitleLang ? [subtitleLang] : undefined,
        subtitleAuto: Boolean(selectedSubtitle?.isAuto),
        subtitleFormat: selectedSubtitle?.ext ?? undefined,
        transcodePresetId: customPresetId || undefined,
        deleteSourceFileAfterTranscode: customPresetId ? customDeleteSourceAfterTranscode : undefined,
        connectorId: downloadPrepared.connectorId,
        useConnector: downloadUseConnector && downloadPrepared.connectorAvailable,
      })
      resetDownloadState()
      setActiveNewDialog(null)
      messageBus.publishToast({
        intent: "success",
        title: t("library.download.start"),
        description: t("library.download.created"),
      })
    } catch (error) {
      setDownloadSubmitError(resolveErrorMessage(error, t("library.errors.unknown")))
    }
  }, [
    createYtdlp,
    customFormats,
    customParseResult,
    customPresetId,
    customSelectedFormat,
    customSelectedSubtitle,
    customDeleteSourceAfterTranscode,
    downloadPrepared,
    downloadUseConnector,
    resetDownloadState,
    t,
  ])

  const handleCreateImport = React.useCallback(
    async (request: CreateSubtitleImportRequest | CreateVideoImportRequest, kind: "subtitle" | "video") => {
      if (kind === "subtitle") {
        await createSubtitleImport.mutateAsync(request as CreateSubtitleImportRequest)
      } else {
        await createVideoImport.mutateAsync(request as CreateVideoImportRequest)
      }
      setActiveNewDialog(null)
      messageBus.publishToast({
        intent: "success",
        title: kind === "subtitle" ? t("library.actions.importSubtitle") : t("library.actions.importVideo"),
        description: t("library.import.completed"),
      })
    },
    [createSubtitleImport, createVideoImport, t],
  )

  const handleSubtitleImport = React.useCallback(async () => {
    if (!subtitlePath.trim()) {
      return
    }
    await handleCreateImport(
      {
        path: subtitlePath.trim(),
        libraryId: importTargetMode === "existing" ? importTargetLibraryID || undefined : undefined,
        title: subtitleTitle.trim(),
        source: "import",
      },
      "subtitle",
    )
    setSubtitlePath("")
    setSubtitleTitle("")
  }, [handleCreateImport, importTargetLibraryID, importTargetMode, subtitlePath, subtitleTitle])

  const handleVideoImport = React.useCallback(async () => {
    if (!importVideoPath.trim()) {
      return
    }
    await handleCreateImport(
      {
        path: importVideoPath.trim(),
        libraryId: importTargetMode === "existing" ? importTargetLibraryID || undefined : undefined,
        title: importVideoTitle.trim(),
        source: "import",
      },
      "video",
    )
    setImportVideoPath("")
    setImportVideoTitle("")
  }, [handleCreateImport, importTargetLibraryID, importTargetMode, importVideoPath, importVideoTitle])

  const handleSelectImportVideo = React.useCallback(async () => {
    try {
      const selection = await Dialogs.OpenFile({
        Title: t("library.import.videoPickerTitle"),
        AllowsOtherFiletypes: false,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [
          {
            DisplayName: t("library.import.videoFilter"),
            Pattern: VIDEO_IMPORT_EXTENSIONS.map((ext) => `*.${ext}`).join(";"),
          },
        ],
      })
      const selectedPath = resolveDialogPath(selection)
      if (!selectedPath) {
        return
      }
      const extension = extractExtensionFromPath(selectedPath)
      if (!VIDEO_IMPORT_EXTENSIONS.includes(extension)) {
        messageBus.publishToast({
          intent: "warning",
          title: t("library.import.unsupportedTitle"),
          description: t("library.import.videoUnsupportedHint"),
        })
        return
      }
      return selectedPath
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.import.openPickerFailed"),
        description: resolveErrorMessage(error, t("library.errors.unknown")),
      })
    }
    return ""
  }, [t])

  const handleSelectImportSubtitle = React.useCallback(async () => {
    try {
      const selection = await Dialogs.OpenFile({
        Title: t("library.import.subtitlePickerTitle"),
        AllowsOtherFiletypes: false,
        CanChooseFiles: true,
        CanChooseDirectories: false,
        Filters: [
          {
            DisplayName: t("library.import.subtitleFilter"),
            Pattern: SUBTITLE_IMPORT_EXTENSIONS.map((ext) => `*.${ext}`).join(";"),
          },
        ],
      })
      const selectedPath = resolveDialogPath(selection)
      if (!selectedPath) {
        return
      }
      const extension = extractExtensionFromPath(selectedPath)
      if (!SUBTITLE_IMPORT_EXTENSIONS.includes(extension)) {
        messageBus.publishToast({
          intent: "warning",
          title: t("library.import.unsupportedTitle"),
          description: t("library.import.subtitleUnsupportedHint"),
        })
        return
      }
      return selectedPath
    } catch (error) {
      messageBus.publishToast({
        intent: "warning",
        title: t("library.import.openPickerFailed"),
        description: resolveErrorMessage(error, t("library.errors.unknown")),
      })
    }
    return ""
  }, [t])

  const openImportVideoDialog = React.useCallback(async () => {
    setImportVideoPath("")
    setImportVideoTitle("")
    setImportTargetMode(selectedLibraryId ? "existing" : "new")
    setImportTargetLibraryID(selectedLibraryId)
    const selectedPath = (await handleSelectImportVideo())?.trim() ?? ""
    if (!selectedPath) {
      return
    }
    setImportVideoPath(selectedPath)
    setImportVideoTitle(stripPathExtension(getPathBaseName(selectedPath)))
    setActiveNewDialog("importVideo")
  }, [handleSelectImportVideo, selectedLibraryId])

  const openImportSubtitleDialog = React.useCallback(async () => {
    setSubtitlePath("")
    setSubtitleTitle("")
    setImportTargetMode(selectedLibraryId ? "existing" : "new")
    setImportTargetLibraryID(selectedLibraryId)
    const selectedPath = (await handleSelectImportSubtitle())?.trim() ?? ""
    if (!selectedPath) {
      return
    }
    setSubtitlePath(selectedPath)
    setSubtitleTitle(stripPathExtension(getPathBaseName(selectedPath)))
    setActiveNewDialog("importSubtitle")
  }, [handleSelectImportSubtitle, selectedLibraryId])

  const handleReselectImportVideo = React.useCallback(async () => {
    const selectedPath = (await handleSelectImportVideo())?.trim() ?? ""
    if (!selectedPath) {
      return
    }
    setImportVideoPath(selectedPath)
    setImportVideoTitle(stripPathExtension(getPathBaseName(selectedPath)))
  }, [handleSelectImportVideo])

  const handleReselectImportSubtitle = React.useCallback(async () => {
    const selectedPath = (await handleSelectImportSubtitle())?.trim() ?? ""
    if (!selectedPath) {
      return
    }
    setSubtitlePath(selectedPath)
    setSubtitleTitle(stripPathExtension(getPathBaseName(selectedPath)))
  }, [handleSelectImportSubtitle])

  const canSubmitImportTarget = importTargetMode === "new" || Boolean(importTargetLibraryID.trim())
  const imagePreviewURL = React.useMemo(
    () => buildAssetPreviewURL(httpBaseURL, imagePreview?.path ?? ""),
    [httpBaseURL, imagePreview?.path],
  )
  const filteredHistory = React.useMemo(
    () => filterHistory(displayHistory, searchQuery, t),
    [displayHistory, searchQuery, t],
  )
  const resourceSidebarFiles = React.useMemo(
    () =>
      focusedResourceLibrary
        ? filterLibraryFilesForSidebar(sortByCreatedAtDesc(focusedResourceLibrary.files ?? []), searchQuery)
        : [],
    [focusedResourceLibrary, searchQuery],
  )
  const resourceSidebarRecords = React.useMemo(
    () =>
      focusedResourceLibrary
        ? filterHistory(focusedResourceLibrary.records.history ?? [], searchQuery, t)
        : filteredHistory,
    [filteredHistory, focusedResourceLibrary, searchQuery, t],
  )

  React.useEffect(() => {
    if (!resourceFocusedLibraryId) {
      return
    }
    if (!filteredResourceLibraries.some((item) => item.id === resourceFocusedLibraryId)) {
      setResourceFocusedLibraryId("")
    }
  }, [filteredResourceLibraries, resourceFocusedLibraryId])

  const isRefreshingLibraryView =
    librariesQuery.isFetching ||
    operationsQuery.isFetching ||
    moduleConfigQuery.isFetching ||
    (selectedLibraryId ? selectedLibraryQuery.isFetching : false)

  const handleRefreshLibraryView = React.useCallback(async () => {
    const jobs: Array<Promise<unknown>> = [librariesQuery.refetch(), operationsQuery.refetch(), moduleConfigQuery.refetch()]
    if (selectedLibraryId) {
      jobs.push(selectedLibraryQuery.refetch())
    }
    await Promise.all(jobs)
  }, [librariesQuery, moduleConfigQuery, operationsQuery, selectedLibraryId, selectedLibraryQuery])

  const showResourcesEmptyState = pageTab === "resources" && !librariesQuery.isLoading && libraryOptions.length === 0
  const showLibraryLoading =
    librariesQuery.isLoading ||
    operationsQuery.isLoading ||
    (pageTab === "workspace" && Boolean(selectedLibraryId) && selectedLibraryQuery.isLoading) ||
    (pageTab === "config" && moduleConfigQuery.isLoading)

  return (
    <TooltipProvider>
      <div className="flex h-full min-h-0 flex-col gap-4 overflow-hidden">
        <div className="grid shrink-0 grid-cols-[auto_minmax(0,1fr)] items-center gap-3 pt-1">
          <Tabs
            value={pageTab}
            onValueChange={(value) => setPageTab(value as LibraryPageTab)}
            className="min-w-0 w-auto"
          >
            <TabsList>
              <TabsTrigger value="overview">
                <Sparkles className="h-3.5 w-3.5" />
                {t("library.tabs.overview")}
              </TabsTrigger>
              <TabsTrigger value="tasks">
                <ListChecks className="h-3.5 w-3.5" />
                {t("library.tabs.tasks")}
              </TabsTrigger>
              <TabsTrigger value="resources">
                <Database className="h-3.5 w-3.5" />
                {t("library.tabs.resources")}
              </TabsTrigger>
              <TabsTrigger value="workspace">
                <Clapperboard className="h-3.5 w-3.5" />
                {t("library.tabs.workspace")}
              </TabsTrigger>
              <TabsTrigger value="config">
                <Settings2 className="h-3.5 w-3.5" />
                {t("library.tabs.config")}
              </TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="flex min-w-0 flex-nowrap items-center justify-end gap-2 overflow-x-auto pb-1 -mb-1">
            {pageTab === "overview" ? (
              <>
                <Button variant="outline" size="compact" className="gap-2" onClick={() => setActiveNewDialog("download")}>
                  <Download className="h-4 w-4" />
                  {t("library.actions.newDownload")}
                </Button>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline" size="compactIcon" onClick={() => void handleRefreshLibraryView()}>
                      <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("library.actions.refresh")}</TooltipContent>
                </Tooltip>
              </>
            ) : null}

            {pageTab === "tasks" ? (
              <>
                <TaskFilterCombobox
                  searchQuery={searchQuery}
                  onSearchQueryChange={setSearchQuery}
                  statusOptions={taskStatusOptions}
                  selectedStatuses={taskStatusFilters}
                  onToggleStatus={(value, checked) =>
                    setTaskStatusFilters((current) => toggleMultiFilterValue(current, value, checked))
                  }
                  taskTypeOptions={taskTypeOptions}
                  selectedTaskTypes={taskTypeFilters}
                  onToggleTaskType={(value, checked) =>
                    setTaskTypeFilters((current) => toggleMultiFilterValue(current, value, checked))
                  }
                  onClearAll={() => {
                    setSearchQuery("")
                    setTaskStatusFilters([])
                    setTaskTypeFilters([])
                  }}
                  filterCount={taskFilterCount}
                  t={t}
                />
                {taskSelectionMode ? (
                  <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                    <div className="inline-flex items-center gap-2 px-3 text-xs text-muted-foreground">
                      <ListChecks className="h-3.5 w-3.5" />
                      <span>
                        {formatTemplate(t("library.actions.selectedTaskCount"), {
                          count: selectedTaskCount,
                        })}
                      </span>
                    </div>
                    <Button
                      variant="ghost"
                      size="compact"
                      className="gap-1.5 rounded-none border-0 border-l border-border/70 text-destructive hover:text-destructive"
                      disabled={selectedTaskCount === 0 || deleteOperations.isPending}
                      onClick={handleBatchDeleteTasks}
                    >
                      <Trash2 className="h-4 w-4" />
                      {t("library.task.deleteConfirm")}
                    </Button>
                    <Button
                      variant="ghost"
                      size="compact"
                      className="gap-1.5 rounded-none border-0 border-l border-border/70"
                      onClick={handleExitTaskSelectionMode}
                    >
                      {t("library.actions.cancelSelection")}
                    </Button>
                  </div>
                ) : (
                  <Button variant="outline" size="compact" className="gap-2" onClick={handleEnterTaskSelectionMode}>
                    <ListChecks className="h-4 w-4" />
                    {t("library.actions.selectTasks")}
                  </Button>
                )}
                <Button variant="outline" size="compact" className="gap-2" onClick={() => setActiveNewDialog("download")}>
                  <Download className="h-4 w-4" />
                  {t("library.actions.newDownload")}
                </Button>
                <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                  <DropdownMenu>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <DropdownMenuTrigger asChild>
                          <Button
                            variant="ghost"
                            size="compactIcon"
                            className="rounded-none border-0"
                            aria-label={t("library.actions.customizeColumns")}
                          >
                            <SlidersHorizontal className="h-4 w-4" />
                          </Button>
                        </DropdownMenuTrigger>
                      </TooltipTrigger>
                      <TooltipContent>{t("library.actions.customizeColumns")}</TooltipContent>
                    </Tooltip>
                    <DropdownMenuContent align="end">
                      <DropdownMenuLabel>{t("library.actions.visibleColumns")}</DropdownMenuLabel>
                      <DropdownMenuSeparator />
                      {taskColumnOptions.length === 0 ? (
                        <DropdownMenuItem disabled>
                          {t("library.actions.noConfigurableColumns")}
                        </DropdownMenuItem>
                      ) : (
                        <>
                          {taskColumnOptions.map((column) => (
                            <DropdownMenuCheckboxItem
                              key={column.id}
                              checked={currentColumnVisibility[column.id] ?? true}
                              onCheckedChange={() =>
                                handleColumnVisibilityChange({
                                  ...currentColumnVisibility,
                                  [column.id]: !(currentColumnVisibility[column.id] ?? true),
                                })
                              }
                            >
                              {column.label}
                            </DropdownMenuCheckboxItem>
                          ))}
                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            disabled={isTaskColumnVisibilityDefault}
                            onClick={handleResetTaskColumnVisibility}
                          >
                            {t("library.actions.resetVisibleColumns")}
                          </DropdownMenuItem>
                        </>
                      )}
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="compactIcon"
                        className="rounded-none border-0 border-l border-border/70"
                        onClick={() => void handleRefreshLibraryView()}
                      >
                        <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("library.actions.refresh")}</TooltipContent>
                  </Tooltip>
                </div>
              </>
            ) : null}

            {pageTab === "resources" ? (
              <>
                {resourceViewMode === "file" ? (
                  <ResourceFileFilterCombobox
                    searchQuery={searchQuery}
                    onSearchQueryChange={setSearchQuery}
                    fileTypeFilter={resourceFileTypeFilter}
                    onFileTypeFilterChange={setResourceFileTypeFilter}
                    fileStatusFilter={resourceFileStatusFilter}
                    onFileStatusFilterChange={setResourceFileStatusFilter}
                    onClearAll={() => {
                      setSearchQuery("")
                      setResourceFileTypeFilter("all")
                      setResourceFileStatusFilter("active")
                    }}
                    filterCount={resourceFileFilterCount}
                    t={t}
                  />
                ) : (
                  <LibraryPageSearchInput
                    value={searchQuery}
                    onChange={(event) => setSearchQuery(event.target.value)}
                    placeholder={t("library.filter.search")}
                    wrapperClassName="min-w-[144px] w-[172px] max-w-[196px] flex-[0_1_172px]"
                  />
                )}
                {resourceViewMode === "file" ? (
                  <>
                    {fileSelectionMode ? (
                      <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                        <div className="inline-flex items-center gap-2 px-3 text-xs text-muted-foreground">
                          <ListChecks className="h-3.5 w-3.5" />
                          <span>
                            {formatTemplate(t("library.actions.selectedTaskCount"), {
                              count: selectedFileCount,
                            })}
                          </span>
                        </div>
                        <Button
                          variant="ghost"
                          size="compact"
                          className="gap-1.5 rounded-none border-0 border-l border-border/70 text-destructive hover:text-destructive"
                          disabled={selectedFileCount === 0 || deleteFiles.isPending}
                          onClick={handleBatchDeleteFiles}
                        >
                          <Trash2 className="h-4 w-4" />
                          {t("library.file.deleteConfirm")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="compact"
                          className="gap-1.5 rounded-none border-0 border-l border-border/70"
                          onClick={handleExitFileSelectionMode}
                        >
                          {t("library.actions.cancelSelection")}
                        </Button>
                      </div>
                    ) : (
                      <Button variant="outline" size="compact" className="gap-2" onClick={handleEnterFileSelectionMode}>
                        <ListChecks className="h-4 w-4" />
                        {t("library.actions.selectFiles")}
                      </Button>
                    )}
                  </>
                ) : null}
                <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant={resourceViewMode === "library" ? "secondary" : "ghost"}
                        size="compactIcon"
                        className="rounded-none border-0"
                        onClick={() => setResourceViewMode("library")}
                        aria-label={t("library.resources.viewLibrary")}
                      >
                        <LayoutGrid className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("library.resources.viewLibrary")}</TooltipContent>
                  </Tooltip>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant={resourceViewMode === "file" ? "secondary" : "ghost"}
                        size="compactIcon"
                        className="rounded-none border-l border-border/70"
                        onClick={() => setResourceViewMode("file")}
                        aria-label={t("library.resources.viewFile")}
                      >
                        <List className="h-4 w-4" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("library.resources.viewFile")}</TooltipContent>
                  </Tooltip>
                </div>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="outline" size="compact" className="gap-2">
                      <FilePlus2 className="h-4 w-4" />
                      {t("library.tools.import")}
                      <ChevronDown className="h-3.5 w-3.5" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuItem onClick={openImportVideoDialog}>
                      <Video className="h-4 w-4" />
                      <span>{t("library.actions.importVideo")}</span>
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={openImportSubtitleDialog}>
                      <ImageIcon className="h-4 w-4" />
                      <span>{t("library.actions.importSubtitle")}</span>
                    </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                {resourceViewMode === "file" ? (
                  <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                    <DropdownMenu>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <DropdownMenuTrigger asChild>
                            <Button
                              variant="ghost"
                              size="compactIcon"
                              className="rounded-none border-0"
                              aria-label={t("library.actions.customizeColumns")}
                            >
                              <SlidersHorizontal className="h-4 w-4" />
                            </Button>
                          </DropdownMenuTrigger>
                        </TooltipTrigger>
                        <TooltipContent>{t("library.actions.customizeColumns")}</TooltipContent>
                      </Tooltip>
                      <DropdownMenuContent align="end">
                        <DropdownMenuLabel>{t("library.actions.visibleColumns")}</DropdownMenuLabel>
                        <DropdownMenuSeparator />
                        {fileColumnOptions.length === 0 ? (
                          <DropdownMenuItem disabled>
                            {t("library.actions.noConfigurableColumns")}
                          </DropdownMenuItem>
                        ) : (
                          <>
                            {fileColumnOptions.map((column) => (
                              <DropdownMenuCheckboxItem
                                key={column.id}
                                checked={currentColumnVisibility[column.id] ?? true}
                                onCheckedChange={() =>
                                  handleColumnVisibilityChange({
                                    ...currentColumnVisibility,
                                    [column.id]: !(currentColumnVisibility[column.id] ?? true),
                                  })
                                }
                              >
                                {column.label}
                              </DropdownMenuCheckboxItem>
                            ))}
                            <DropdownMenuSeparator />
                            <DropdownMenuItem
                              disabled={isFileColumnVisibilityDefault}
                              onClick={handleResetFileColumnVisibility}
                            >
                              {t("library.actions.resetVisibleColumns")}
                            </DropdownMenuItem>
                          </>
                        )}
                      </DropdownMenuContent>
                    </DropdownMenu>
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="compactIcon"
                          className="rounded-none border-0 border-l border-border/70"
                          onClick={() => void handleRefreshLibraryView()}
                        >
                          <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("library.actions.refresh")}</TooltipContent>
                    </Tooltip>
                  </div>
                ) : (
                  <Button variant="outline" size="compactIcon" onClick={() => void handleRefreshLibraryView()}>
                    <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                  </Button>
                )}
              </>
            ) : null}

            {pageTab === "workspace" ? (
              <>
                <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                  <DropdownMenu>
                    <DropdownMenuTrigger asChild>
                      <Button
                        variant="ghost"
                        size="compact"
                        className="gap-2 rounded-none border-0 bg-transparent focus-visible:ring-1 focus-visible:ring-offset-1"
                      >
                        <FilePlus2 className="h-4 w-4" />
                        {t("library.tools.import")}
                        <ChevronDown className="h-3.5 w-3.5" />
                      </Button>
                    </DropdownMenuTrigger>
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem onClick={openImportVideoDialog}>
                        <Video className="h-4 w-4" />
                        <span>{t("library.actions.importVideo")}</span>
                      </DropdownMenuItem>
                      <DropdownMenuItem onClick={openImportSubtitleDialog}>
                        <ImageIcon className="h-4 w-4" />
                        <span>{t("library.actions.importSubtitle")}</span>
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                  <DropdownMenu>
                    {(workspaceToolbarState?.exportDisabledReason ?? "").trim() ? (
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="inline-flex">
                            <DropdownMenuTrigger asChild>
                              <Button
                                variant="ghost"
                                size="compact"
                                className="gap-2 rounded-none border-0 border-l border-border/70 bg-transparent focus-visible:ring-1 focus-visible:ring-offset-1"
                                disabled={
                                  !workspaceToolbarState ||
                                  (!workspaceToolbarState.canExportVideo && !workspaceToolbarState.canExportSubtitle)
                                }
                              >
                                <Download className="h-4 w-4" />
                                {t("library.workspace.actions.export")}
                                <ChevronDown className="h-3.5 w-3.5" />
                              </Button>
                            </DropdownMenuTrigger>
                          </span>
                        </TooltipTrigger>
                        <TooltipContent className="max-w-[18rem] text-xs leading-5">
                          {workspaceToolbarState?.exportDisabledReason}
                        </TooltipContent>
                      </Tooltip>
                    ) : (
                      <DropdownMenuTrigger asChild>
                        <Button
                          variant="ghost"
                          size="compact"
                          className="gap-2 rounded-none border-0 border-l border-border/70 bg-transparent focus-visible:ring-1 focus-visible:ring-offset-1"
                          disabled={
                            !workspaceToolbarState ||
                            (!workspaceToolbarState.canExportVideo && !workspaceToolbarState.canExportSubtitle)
                          }
                        >
                          <Download className="h-4 w-4" />
                          {t("library.workspace.actions.export")}
                          <ChevronDown className="h-3.5 w-3.5" />
                        </Button>
                      </DropdownMenuTrigger>
                    )}
                    <DropdownMenuContent align="end">
                      <DropdownMenuItem
                        disabled={!workspaceToolbarState?.canExportVideo}
                        onClick={() => workspaceToolbarState?.onExportVideo()}
                      >
                        <Video className="h-4 w-4" />
                        <span>{t("library.workspace.exportVideo")}</span>
                      </DropdownMenuItem>
                      <DropdownMenuItem
                        disabled={!workspaceToolbarState?.canExportSubtitle}
                        onClick={() => workspaceToolbarState?.onExportSubtitle()}
                      >
                        <Captions className="h-4 w-4" />
                        <span>{t("library.workspace.exportSubtitle")}</span>
                      </DropdownMenuItem>
                    </DropdownMenuContent>
                  </DropdownMenu>
                </div>
                <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                  {workspaceToolbarState?.activeEditor === "video" ? (
                    <Tooltip>
                      <TooltipTrigger asChild>
                        <Button
                          variant="ghost"
                          size="compactIcon"
                          className="rounded-none border-0"
                          onClick={() => workspaceToolbarState.onOpenCurrentFile()}
                          disabled={!workspaceToolbarState.canOpenCurrentFile}
                        >
                          <FolderOpen className="h-4 w-4" />
                        </Button>
                      </TooltipTrigger>
                      <TooltipContent>{t("library.tooltips.openFolder")}</TooltipContent>
                    </Tooltip>
                  ) : null}
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        variant="ghost"
                        size="compactIcon"
                        className={cn(
                          "rounded-none border-0",
                          workspaceToolbarState?.activeEditor === "video" ? "border-l border-border/70" : "",
                        )}
                        onClick={() => void handleRefreshLibraryView()}
                      >
                        <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>{t("library.actions.refresh")}</TooltipContent>
                  </Tooltip>
                </div>
              </>
            ) : null}

            {pageTab === "config" ? (
              <>
                {configToolbarState?.actions}
                <Tooltip>
                  <TooltipTrigger asChild>
                    <Button variant="outline" size="compactIcon" onClick={() => void handleRefreshLibraryView()}>
                      <RefreshCcw className={cn("h-4 w-4", isRefreshingLibraryView ? "animate-spin" : "")} />
                    </Button>
                  </TooltipTrigger>
                  <TooltipContent>{t("library.actions.refresh")}</TooltipContent>
                </Tooltip>
              </>
            ) : null}
          </div>
        </div>

        <div className="min-h-0 flex-1 overflow-hidden">
          {showResourcesEmptyState ? (
            <EmptyState
              title={t("library.resources.libraryEmpty")}
              description={t("library.empty.description")}
            />
          ) : (
            <div className="flex h-full min-h-0 flex-col gap-4 overflow-hidden">
              {showLibraryLoading ? <LoadingCard t={t} /> : null}

              {!showLibraryLoading && pageTab === "overview" ? (
                <LibraryOverviewPage
                  cards={overviewCards}
                  chartData={overviewTrendData}
                  chartTitle={t("library.overview.chart.title")}
                  chartSuccessLabel={t("library.overview.chart.success")}
                  chartFailedLabel={t("library.overview.chart.failed")}
                  chartGranularity={chartGranularity}
                  chartGranularityOptions={[
                    { value: "1d", label: t("library.overview.granularity.1d") },
                    { value: "7d", label: t("library.overview.granularity.7d") },
                    { value: "30d", label: t("library.overview.granularity.30d") },
                  ]}
                  onChartGranularityChange={setChartGranularity}
                  recentTitle={t("library.overview.recent")}
                  recentContent={
                    displayHistory.length > 0 ? (
                      <ResourceRecordTimeline
                        records={displayHistory}
                        onOpenTaskDialog={openTaskDialog}
                        t={t}
                        language={language}
                      />
                    ) : undefined
                  }
                  emptyRecentText={t("library.overview.recentEmpty")}
                />
              ) : null}

              {!showLibraryLoading && pageTab === "tasks" ? (
                <LibraryTable
                  viewMode="task"
                  data={taskRows}
                  columns={taskColumns}
                  columnVisibility={currentColumnVisibility}
                  onColumnVisibilityChange={handleColumnVisibilityChange}
                  rowsPerPage={rowsPerPage}
                  onRowsPerPageChange={setRowsPerPage}
                  getRowId={(row) => row.id}
                  rowSelection={taskSelectionMode ? taskRowSelection : undefined}
                  onRowSelectionChange={taskSelectionMode ? setTaskRowSelection : undefined}
                  enableRowSelection={taskSelectionMode}
                />
              ) : null}

              {!showLibraryLoading && pageTab === "resources" ? (
                resourceViewMode === "library" ? (
                  <LibraryResourcesPanel
                    libraries={filteredResourceLibraries}
                    focusedLibrary={focusedResourceLibrary}
                    sidebarFiles={resourceSidebarFiles}
                    sidebarRecords={resourceSidebarRecords}
                    httpBaseURL={httpBaseURL}
                    language={language}
                    t={t}
                    onSelectLibrary={(libraryId) => {
                      setResourceFocusedLibraryId((current) => (current === libraryId ? "" : libraryId))
                    }}
                    onClearSelection={() => setResourceFocusedLibraryId("")}
                    onLibraryDeleted={(libraryId) => {
                      setResourceFocusedLibraryId("")
                      setSelectedLibraryId((current) => (current === libraryId ? "" : current))
                    }}
                    onPreviewImage={(preview) => {
                      if (!preview.path.trim()) {
                        return
                      }
                      setImagePreview(preview)
                    }}
                    onOpenLibraryFile={(file) => {
                      const target = buildWorkspaceTargetFromLibraryFile(file, focusedResourceLibrary?.files ?? [])
                      if (!target) {
                        return
                      }
                      setSelectedLibraryId(file.libraryId)
                      openLibraryWorkspace(target)
                    }}
                    onOpenTaskDialog={openTaskDialog}
                  />
                ) : (
                  <LibraryTable
                    viewMode="file"
                    data={resourceRows}
                    columns={fileColumns}
                    columnVisibility={currentColumnVisibility}
                    onColumnVisibilityChange={handleColumnVisibilityChange}
                    rowsPerPage={rowsPerPage}
                    onRowsPerPageChange={setRowsPerPage}
                    getRowId={(row) => row.id}
                    rowSelection={fileSelectionMode ? fileRowSelection : undefined}
                    onRowSelectionChange={fileSelectionMode ? setFileRowSelection : undefined}
                    enableRowSelection={
                      fileSelectionMode
                        ? (row) => row.original.status !== "deleted"
                        : false
                    }
                  />
                )
              ) : null}

              {!showLibraryLoading && pageTab === "workspace" ? (
                <LibraryWorkspacePage
                  library={selectedLibraryDisplay}
                  moduleConfig={moduleConfigValue ?? undefined}
                  onModuleConfigChange={setModuleConfigDraft}
                  files={activeWorkspaceFiles}
                  httpBaseURL={httpBaseURL}
                  onRequestImportVideo={openImportVideoDialog}
                  onRequestImportSubtitle={openImportSubtitleDialog}
                  onToolbarStateChange={setWorkspaceToolbarState}
                />
              ) : null}

              {!showLibraryLoading && pageTab === "config" && moduleConfigValue ? (
                <div className="relative min-h-0 flex-1 overflow-hidden">
                  <LibraryConfigPage
                    value={moduleConfigValue}
                    onChange={handleModuleConfigChange}
                    onRequestPersist={requestPersistModuleConfig}
                    onToolbarStateChange={setConfigToolbarState}
                    requestedPage={libraryConfigRequestedPage}
                  />
                  {updateModuleConfig.isPending ? (
                    <div className="pointer-events-none absolute bottom-5 right-5 z-30 flex items-center gap-2 rounded-full border border-border/70 bg-background/92 px-3 py-1.5 text-xs text-muted-foreground shadow-lg backdrop-blur">
                      <Loader2 className="h-3.5 w-3.5 animate-spin" />
                      <span>{t("library.config.saving")}</span>
                    </div>
                  ) : null}
                </div>
              ) : null}

              {!showLibraryLoading &&
              pageTab !== "overview" &&
              pageTab !== "tasks" &&
              pageTab !== "resources" &&
              pageTab !== "workspace" &&
              pageTab !== "config" ? (
                <div className="min-h-0 flex-1">
                  <div className="flex h-full items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center text-sm text-muted-foreground">
                    {t("library.select.description")}
                  </div>
                </div>
              ) : null}
            </div>
          )}
        </div>

        <Dialog
          open={activeNewDialog !== null}
          onOpenChange={(open) => {
            if (!open && activeNewDialog === "download") {
              resetDownloadState()
            }
            setActiveNewDialog(open ? activeNewDialog : null)
          }}
        >
          <DashboardDialogContent
            size={
              activeNewDialog === "download"
                ? "compact"
                : activeNewDialog === "importSubtitle" || activeNewDialog === "importVideo"
                  ? "standard"
                  : "compact"
            }
            className={cn(
              (activeNewDialog === "importSubtitle" || activeNewDialog === "importVideo") &&
                "flex max-h-[80vh] min-h-0 flex-col gap-4 text-xs",
            )}
          >
            {activeNewDialog === "download" ? (
              <>
	                <DashboardDialogHeader>
	                  <DialogTitle>{t("library.download.title")}</DialogTitle>
	                  {downloadStep === "dependency" ? (
	                    <DialogDescription>
	                      {t("library.download.dependencyDescription")}
	                    </DialogDescription>
	                  ) : null}
                </DashboardDialogHeader>

                {downloadStep === "dependency" ? (
                  <>
                    <DashboardDialogSection className="space-y-2">
                      {downloadDependencyIssues.map((item) => (
                        <DependencyToolRow
                          key={item.name}
                          item={item}
                          installed={isToolReady(toolsByName.get(item.name))}
                          t={t}
                          onRequestRefresh={refreshExternalToolsNow}
                        />
                      ))}
                    </DashboardDialogSection>
                    <DashboardDialogFooter>
                      <Button variant="ghost" size="compact" onClick={() => setActiveNewDialog(null)}>
                        {t("common.close")}
                      </Button>
                      <Button size="compact" onClick={handleOpenExternalTools}>
                        {t("library.tools.dependencyOpenSettings")}
                      </Button>
                    </DashboardDialogFooter>
                  </>
                ) : null}

                {downloadStep === "input" ? (
                  <DashboardDialogSection tone="field" className="space-y-3">
                    <div className="space-y-2">
                      <div className="text-xs font-medium text-muted-foreground">
                        {t("library.download.inputTitle")}
                      </div>
                      <div className="flex items-center gap-2">
                        <Input
                          value={downloadUrl}
                          onChange={(event) => setDownloadUrl(event.target.value)}
                          placeholder={t("library.download.inputPlaceholder")}
                          onKeyDown={(event) => {
                            if (event.key === "Enter") {
                              event.preventDefault()
                              void handlePrepareDownload()
                            }
                          }}
                        />
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button size="compactIcon" onClick={() => void handlePrepareDownload()} disabled={!downloadUrl.trim() || prepareYtdlp.isPending}>
                              {prepareYtdlp.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : <ArrowRight className="h-4 w-4" />}
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>{t("library.download.request")}</TooltipContent>
                        </Tooltip>
                      </div>
                    </div>
                    {downloadPrepareError ? <div className="text-xs text-destructive">{downloadPrepareError}</div> : null}
                  </DashboardDialogSection>
                ) : null}

                {downloadStep === "config" ? (
                  <DashboardDialogBody>
                    <DashboardDialogSection className="space-y-2">
                      <div className="flex items-center gap-2 text-xs font-medium text-muted-foreground">
                        <span>{formatDomainLabel(downloadPrepared?.domain, downloadPrepared?.url)}</span>
                        {downloadPrepared?.reachable === false ? (
                          <Badge variant="outline" className="border-amber-300 text-amber-700">
                            {t("library.download.reachabilityWarning")}
                          </Badge>
                        ) : null}
                      </div>
                      <div className="flex items-center gap-2">
                        <div className="relative flex-1">
                          <Input value={downloadPrepared?.url ?? downloadUrl} readOnly className="bg-muted pr-9" />
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                size="compactIcon"
                                variant="ghost"
                                className="absolute right-1 top-1/2 h-6 w-6 -translate-y-1/2"
                                onClick={() => {
                                  if (downloadPrepared?.url) {
                                    setDownloadUrl(downloadPrepared.url)
                                  }
                                  setDownloadPrepared(null)
                                  setDownloadStep("input")
                                  setCustomParseResult(null)
                                  setCustomParseError("")
                                  setCustomFormatId("")
                                  setCustomSubtitleId("")
                                  setCustomPresetId("")
                                }}
                              >
                                <PencilLine className="h-3.5 w-3.5" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent>{t("library.download.modifyLink")}</TooltipContent>
                          </Tooltip>
                        </div>
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <div>
                              <Switch
                                checked={downloadPrepared?.connectorAvailable ? downloadUseConnector : false}
                                onCheckedChange={(checked) => {
                                  if (downloadPrepared?.connectorAvailable) {
                                    setDownloadUseConnector(checked)
                                  }
                                }}
                                disabled={!downloadPrepared?.connectorAvailable}
                              />
                            </div>
                          </TooltipTrigger>
                          <TooltipContent>
                            {downloadPrepared?.connectorAvailable
                              ? t("library.download.connectorHint")
                              : t("library.download.connectorUnsupportedHint")}
                          </TooltipContent>
                        </Tooltip>
                      </div>
                    </DashboardDialogSection>

                    <Tabs
                      value={downloadTab}
                      onValueChange={(value) => setDownloadTab(value as "quick" | "custom")}
                    >
                      <div className="flex justify-center">
                        <TabsList>
                          <TabsTrigger value="quick">
                            <Zap className="h-3.5 w-3.5" />
                            {t("library.download.tabs.quick")}
                          </TabsTrigger>
                          <TabsTrigger value="custom">
                            <SlidersHorizontal className="h-3.5 w-3.5" />
                            {t("library.download.tabs.custom")}
                          </TabsTrigger>
                        </TabsList>
                      </div>
                    </Tabs>

                    {downloadTab === "quick" ? (
                      <>
                        <Card className="border-border/70 bg-card">
                          <div className="divide-y">
                            <div className="flex items-center justify-between gap-4 p-3 text-sm">
                              <span className="text-muted-foreground">{t("library.download.quality")}</span>
                              <div className="flex items-center gap-2">
                                <Button variant={quickQuality === "best" ? "default" : "outline"} size="compact" onClick={() => setQuickQuality("best")}>
                                  {t("library.download.quality.best")}
                                </Button>
                                <Button variant={quickQuality === "audio" ? "default" : "outline"} size="compact" onClick={() => setQuickQuality("audio")}>
                                  {t("library.download.quality.audio")}
                                </Button>
                              </div>
                            </div>
                            <div className="flex items-center justify-between gap-4 p-3 text-sm">
                              <span className="text-muted-foreground">{t("library.download.subtitle")}</span>
                              <Switch checked={quickSubtitle} onCheckedChange={setQuickSubtitle} />
                            </div>
                            <div className="flex items-center justify-between gap-4 p-3 text-sm">
                              <span className="text-muted-foreground">{t("library.download.transcode")}</span>
                              <Select className="w-[160px]" value={quickPresetId} onChange={(event) => setQuickPresetId(event.target.value)}>
                                <option value="">{t("library.download.transcode.none")}</option>
                                {quickPresets.map((preset) => (
                                  <option key={preset.id} value={preset.id}>
                                    {resolvePresetName(preset, t)}
                                  </option>
                                ))}
                              </Select>
                            </div>
                            {quickPresetId ? (
                              <div className="flex items-center justify-between gap-4 p-3 text-sm">
                                <span className="text-muted-foreground">
                                  {t("library.download.transcode.deleteSourceAfterTranscode")}
                                </span>
                                <Switch checked={quickDeleteSourceAfterTranscode} onCheckedChange={setQuickDeleteSourceAfterTranscode} />
                              </div>
                            ) : null}
                          </div>
                        </Card>
                        <DashboardDialogFooter>
                          <Button variant="ghost" size="compact" onClick={() => setActiveNewDialog(null)}>
                            {t("common.cancel")}
                          </Button>
                          <Button size="compact" onClick={() => void handleStartQuickDownload()} disabled={createYtdlp.isPending}>
                            {createYtdlp.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                            {t("library.download.start")}
                          </Button>
                        </DashboardDialogFooter>
                      </>
                    ) : null}

                    {downloadTab === "custom" ? (
                      <>
                        {customParseResult ? (
                          <>
                            <Card className="border-border/70 bg-card">
                              <div className="divide-y">
                                <div className="flex items-center justify-between gap-4 p-3 text-sm">
                                  <span className="text-muted-foreground">{t("library.download.quality")}</span>
                                  <Select className="w-[220px]" value={customFormatId} onChange={(event) => setCustomFormatId(event.target.value)}>
                                    <option value="">{t("library.download.quality.select")}</option>
                                    {customVideoFormats.length > 0 ? (
                                      <optgroup label={t("library.download.quality.groupAv")}>
                                        {customVideoFormats.map((format) => (
                                          <option key={format.id} value={format.id}>{format.label}</option>
                                        ))}
                                      </optgroup>
                                    ) : null}
                                    {customAudioFormats.length > 0 ? (
                                      <optgroup label={t("library.download.quality.groupAudio")}>
                                        {customAudioFormats.map((format) => (
                                          <option key={format.id} value={format.id}>{format.label}</option>
                                        ))}
                                      </optgroup>
                                    ) : null}
                                  </Select>
                                </div>
                                <div className="flex items-center justify-between gap-4 p-3 text-sm">
                                  <span className="text-muted-foreground">{t("library.download.subtitle")}</span>
                                  <Select className="w-[220px]" value={customSubtitleId} onChange={(event) => setCustomSubtitleId(event.target.value)}>
                                    <option value="">{t("library.download.subtitle.none")}</option>
                                    {customSubtitles.map((subtitle) => (
                                      <option key={subtitle.id} value={subtitle.id}>{formatSubtitleLabel(subtitle, t)}</option>
                                    ))}
                                  </Select>
                                </div>
                                <div className="flex items-center justify-between gap-4 p-3 text-sm">
                                  <span className="text-muted-foreground">{t("library.download.transcode")}</span>
                                  <Select className="w-[220px]" value={customPresetId} onChange={(event) => setCustomPresetId(event.target.value)}>
                                    <option value="">{t("library.download.transcode.none")}</option>
                                    {customPresets.map((preset) => (
                                      <option key={preset.id} value={preset.id}>{resolvePresetName(preset, t)}</option>
                                    ))}
                                  </Select>
                                </div>
                                {customPresetId ? (
                                  <div className="flex items-center justify-between gap-4 p-3 text-sm">
                                    <span className="text-muted-foreground">
                                      {t("library.download.transcode.deleteSourceAfterTranscode")}
                                    </span>
                                    <Switch checked={customDeleteSourceAfterTranscode} onCheckedChange={setCustomDeleteSourceAfterTranscode} />
                                  </div>
                                ) : null}
                              </div>
                            </Card>
                            <DashboardDialogFooter>
                              <Button variant="ghost" size="compact" onClick={() => setActiveNewDialog(null)}>
                                {t("common.cancel")}
                              </Button>
                              <Button size="compact" onClick={() => void handleStartCustomDownload()} disabled={createYtdlp.isPending || !customFormatId}>
                                {createYtdlp.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                                {t("library.download.start")}
                              </Button>
                            </DashboardDialogFooter>
                          </>
                        ) : (
                          <div className="flex justify-center">
                            <Button size="compact" onClick={() => void handleParseDownload()} disabled={parseYtdlp.isPending}>
                              {parseYtdlp.isPending ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : null}
                              {t("library.download.parse")}
                            </Button>
                          </div>
                        )}
                        {customParseError ? <div className="text-xs text-destructive">{customParseError}</div> : null}
                      </>
                    ) : null}

                    {downloadSubmitError ? <div className="text-xs text-destructive">{downloadSubmitError}</div> : null}
                  </DashboardDialogBody>
                ) : null}
              </>
            ) : null}

            {activeNewDialog === "importSubtitle" ? (
              <ImportAssetDialog
                kind="subtitle"
                filePath={subtitlePath}
                importTargetMode={importTargetMode}
                currentLibrary={selectedLibraryDisplay}
                titleValue={subtitleTitle}
                onTitleChange={setSubtitleTitle}
                onModeChange={setImportTargetMode}
                onClose={() => setActiveNewDialog(null)}
                onSelectFile={() => {
                  void handleReselectImportSubtitle()
                }}
                onSubmit={() => {
                  void handleSubtitleImport()
                }}
                submitting={createSubtitleImport.isPending}
                canSubmit={Boolean(subtitlePath.trim()) && canSubmitImportTarget}
                t={t}
              />
            ) : null}

            {activeNewDialog === "importVideo" ? (
              <ImportAssetDialog
                kind="video"
                filePath={importVideoPath}
                importTargetMode={importTargetMode}
                currentLibrary={selectedLibraryDisplay}
                titleValue={importVideoTitle}
                onTitleChange={setImportVideoTitle}
                onModeChange={setImportTargetMode}
                onClose={() => setActiveNewDialog(null)}
                onSelectFile={() => {
                  void handleReselectImportVideo()
                }}
                onSubmit={() => {
                  void handleVideoImport()
                }}
                submitting={createVideoImport.isPending}
                canSubmit={Boolean(importVideoPath.trim()) && canSubmitImportTarget}
                t={t}
              />
            ) : null}
          </DashboardDialogContent>
        </Dialog>

        <Dialog open={imagePreview !== null} onOpenChange={(open) => (!open ? setImagePreview(null) : undefined)}>
          <DashboardDialogContent size="workspace" className="flex max-h-[88vh] w-full flex-col overflow-hidden">
            <DashboardDialogHeader>
              <DialogTitle>{imagePreview?.name || t("library.preview.imageTitle")}</DialogTitle>
              <DialogDescription>{imagePreview?.path || "-"}</DialogDescription>
            </DashboardDialogHeader>
            <DashboardDialogSection tone="inset" className="min-h-0 flex-1 overflow-auto p-2">
              {imagePreviewURL ? (
                <img src={imagePreviewURL} alt={imagePreview?.name ?? ""} className="mx-auto block h-auto max-h-[72vh] w-auto max-w-full object-contain" />
              ) : (
                <div className="flex h-full min-h-[240px] items-center justify-center text-xs text-muted-foreground">
                  <ImageIcon className="mr-2 h-4 w-4" />
                  {t("library.preview.imageUnavailable")}
                </div>
              )}
            </DashboardDialogSection>
          </DashboardDialogContent>
        </Dialog>
      </div>
    </TooltipProvider>
  )
}

type DependencyToolRowProps = {
  item: DependencyIssue
  installed: boolean
  t: Translator
  onRequestRefresh: () => Promise<unknown> | void
}

function DependencyToolRow({ item, installed, t, onRequestRefresh }: DependencyToolRowProps) {
  const installTool = useInstallExternalTool()
  const [running, setRunning] = React.useState(false)
  const [installError, setInstallError] = React.useState("")
  const installState = useExternalToolInstallState(item.name, running || installTool.isPending)
  const stage = (installState.data?.stage ?? "").trim().toLowerCase()
  const progress = clampProgress(installState.data?.progress)
  const stageMessage = installState.data?.message?.trim() ?? ""
  const isRepair = item.status === "invalid"
  const isRunningStage = stage === "downloading" || stage === "extracting" || stage === "verifying"
  const isInstalling = running || installTool.isPending || isRunningStage
  const effectiveStage = stage || (isInstalling ? "downloading" : "idle")
  const actionLabel = isRepair
    ? t("settings.externalTools.actions.repair")
    : t("settings.externalTools.actions.install")
  const stageLabel = t(`settings.externalTools.installDialog.stage.${effectiveStage}`)

  React.useEffect(() => {
    if (!isInstalling) {
      return
    }
    if (stage === "done") {
      setRunning(false)
      setInstallError("")
      void onRequestRefresh()
      return
    }
    if (stage === "error") {
      setRunning(false)
      setInstallError(stageMessage || t("settings.externalTools.installDialog.error"))
      void onRequestRefresh()
    }
  }, [isInstalling, onRequestRefresh, stage, stageMessage, t])

  React.useEffect(() => {
    if (!installed) {
      return
    }
    setInstallError("")
    setRunning(false)
  }, [installed])

  const handleInstall = async () => {
    setInstallError("")
    setRunning(true)
    try {
      await installTool.mutateAsync({ name: item.name })
      await installState.refetch()
      await onRequestRefresh()
    } catch (error) {
      setRunning(false)
      setInstallError(resolveErrorMessage(error, t("library.errors.unknown")))
      await onRequestRefresh()
    }
  }

  return (
    <div className="space-y-2 rounded-md border px-3 py-2">
      <div className="flex items-center justify-between gap-3">
        <div className="min-w-0">
          <div className="truncate text-xs font-semibold uppercase tracking-[0.2em]">{item.name.toUpperCase()}</div>
          <div className="text-xs text-muted-foreground">
            {isRepair ? t("settings.externalTools.status.repair") : t("settings.externalTools.status.install")}
          </div>
        </div>

        {installed ? (
          <span className="inline-flex items-center gap-1 text-xs font-semibold text-emerald-600">
            <CheckCircle2 className="h-3.5 w-3.5" />
            {t("settings.externalTools.status.latest")}
          </span>
        ) : (
          <Button size="compact" className="h-7 text-xs" onClick={() => void handleInstall()} disabled={isInstalling}>
            {isInstalling ? <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" /> : <Download className="mr-1.5 h-3.5 w-3.5" />}
            {isInstalling ? stageLabel : actionLabel}
          </Button>
        )}
      </div>

      {isInstalling ? (
        <div className="space-y-1">
          <Progress value={progress} className="h-1.5" />
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>{stageLabel}</span>
            <span>{progress}%</span>
          </div>
        </div>
      ) : null}

      {installError ? (
        <div className="flex items-center gap-1 text-xs text-destructive">
          <AlertTriangle className="h-3.5 w-3.5 shrink-0" />
          <span className="line-clamp-2">{installError}</span>
        </div>
      ) : null}
    </div>
  )
}

function LoadingCard(props: { t: Translator }) {
  return (
    <div className={cn(DASHBOARD_PANEL_SOLID_SURFACE_CLASS, "px-4 py-8 text-sm text-muted-foreground")}>
      <div className="flex items-center gap-2">
        <Loader2 className="h-4 w-4 animate-spin" />
        {props.t("library.loading.data")}
      </div>
    </div>
  )
}

function TaskFilterCombobox(props: {
  searchQuery: string
  onSearchQueryChange: (value: string) => void
  statusOptions: Array<{ value: TaskStatusFilter; label: string }>
  selectedStatuses: TaskStatusFilter[]
  onToggleStatus: (value: TaskStatusFilter, checked: boolean) => void
  taskTypeOptions: Array<{ value: string; label: string }>
  selectedTaskTypes: string[]
  onToggleTaskType: (value: string, checked: boolean) => void
  onClearAll: () => void
  filterCount: number
  t: Translator
}) {
  const hasFilters = props.filterCount > 0
  const hasSearchQuery = props.searchQuery.trim().length > 0
  const triggerLabel = hasSearchQuery
    ? props.searchQuery
    : props.t("library.filter.searchAndFilterTasks")

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
          title={triggerLabel}
        >
          <span className="flex min-w-0 items-center gap-2">
            <Search className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className={cn("min-w-0 truncate text-xs", hasSearchQuery ? "text-foreground" : "text-muted-foreground")}>
              {triggerLabel}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            {hasFilters ? (
              <Badge variant="subtle" className="h-5 px-1.5 text-[10px] font-medium">
                {props.filterCount}
              </Badge>
            ) : null}
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("library.filter.searchTasksPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>
            {props.t("library.filter.taskStatus")}
          </DropdownMenuLabel>
          {props.statusOptions.map((option) => (
            <DropdownMenuCheckboxItem
              key={option.value}
              checked={props.selectedStatuses.includes(option.value)}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => props.onToggleStatus(option.value, Boolean(checked))}
            >
              {option.label}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>
            {props.t("library.filter.taskType")}
          </DropdownMenuLabel>
          {props.taskTypeOptions.length === 0 ? (
            <DropdownMenuItem disabled>
              {props.t("library.filter.noTaskTypes")}
            </DropdownMenuItem>
          ) : (
            props.taskTypeOptions.map((option) => (
              <DropdownMenuCheckboxItem
                key={option.value}
                checked={props.selectedTaskTypes.includes(option.value)}
                onSelect={(event) => event.preventDefault()}
                onCheckedChange={(checked) => props.onToggleTaskType(option.value, Boolean(checked))}
              >
                {option.label}
              </DropdownMenuCheckboxItem>
            ))
          )}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("library.filter.clearAllTaskFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function ResourceFileFilterCombobox(props: {
  searchQuery: string
  onSearchQueryChange: (value: string) => void
  fileTypeFilter: ResourceFileTypeFilter
  onFileTypeFilterChange: (value: ResourceFileTypeFilter) => void
  fileStatusFilter: ResourceFileStatusFilter
  onFileStatusFilterChange: (value: ResourceFileStatusFilter) => void
  onClearAll: () => void
  filterCount: number
  t: Translator
}) {
  const hasFilters = props.filterCount > 0
  const hasSearchQuery = props.searchQuery.trim().length > 0
  const triggerLabel = hasSearchQuery
    ? props.searchQuery
    : props.t("library.filter.searchAndFilterFiles")

  const fileTypeOptions: Array<{ value: ResourceFileTypeFilter; label: string }> = [
    { value: "all", label: props.t("library.tabs.all") },
    { value: "video", label: props.t("library.tabs.video") },
    { value: "subtitle", label: props.t("library.tabs.subtitle") },
  ]
  const fileStatusOptions: Array<{ value: ResourceFileStatusFilter; label: string }> = [
    { value: "active", label: props.t("library.filter.activeFilesOption") },
    { value: "deleted", label: props.t("library.filter.recycleBinOption") },
    { value: "all", label: props.t("library.filter.allFilesOption") },
  ]

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="outline"
          size="compact"
          className="w-fit min-w-[156px] max-w-[220px] justify-between gap-2 px-2.5"
          title={triggerLabel}
        >
          <span className="flex min-w-0 items-center gap-2">
            <Search className="h-3.5 w-3.5 text-muted-foreground/70" />
            <span className={cn("min-w-0 truncate text-xs", hasSearchQuery ? "text-foreground" : "text-muted-foreground")}>
              {triggerLabel}
            </span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            {hasFilters ? (
              <Badge variant="subtle" className="h-5 px-1.5 text-[10px] font-medium">
                {props.filterCount}
              </Badge>
            ) : null}
            <ChevronDown className="h-3.5 w-3.5 text-muted-foreground/70" />
          </span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent
        align="end"
        className="min-w-[var(--radix-dropdown-menu-trigger-width)] max-w-[280px] p-0"
      >
        <div className="space-y-2 p-2">
          <Input
            size="compact"
            autoFocus
            value={props.searchQuery}
            onChange={(event) => props.onSearchQueryChange(event.target.value)}
            placeholder={props.t("library.filter.searchFilesPlaceholder")}
            className="w-full text-xs placeholder:text-xs"
            onKeyDown={(event) => event.stopPropagation()}
          />
        </div>
        <DropdownMenuSeparator />
        <div className="max-h-[320px] overflow-y-auto p-1">
          <DropdownMenuLabel>
            {props.t("library.filter.fileTypeLabel")}
          </DropdownMenuLabel>
          {fileTypeOptions.map((option) => (
            <DropdownMenuCheckboxItem
              key={option.value}
              checked={props.fileTypeFilter === option.value}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onFileTypeFilterChange(option.value)
                }
              }}
            >
              {option.label}
            </DropdownMenuCheckboxItem>
          ))}

          <DropdownMenuSeparator />
          <DropdownMenuLabel>
            {props.t("library.filter.fileStatusLabel")}
          </DropdownMenuLabel>
          {fileStatusOptions.map((option) => (
            <DropdownMenuCheckboxItem
              key={option.value}
              checked={props.fileStatusFilter === option.value}
              onSelect={(event) => event.preventDefault()}
              onCheckedChange={(checked) => {
                if (checked) {
                  props.onFileStatusFilterChange(option.value)
                }
              }}
            >
              {option.label}
            </DropdownMenuCheckboxItem>
          ))}
        </div>
        <DropdownMenuSeparator />
        <div className="p-1">
          <DropdownMenuItem disabled={!hasFilters} onClick={props.onClearAll}>
            {props.t("library.filter.clearAllFileFilters")}
          </DropdownMenuItem>
        </div>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function LibraryPageSearchInput(props: {
  value: string
  placeholder: string
  onChange: React.ChangeEventHandler<HTMLInputElement>
  wrapperClassName?: string
  inputClassName?: string
}) {
  return (
    <div className={cn("relative w-[240px] py-1", props.wrapperClassName)}>
      <Search className="pointer-events-none absolute left-2.5 top-1/2 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground/70" />
      <Input
        size="compact"
        value={props.value}
        onChange={props.onChange}
        placeholder={props.placeholder}
        className={cn(
          "min-w-0 w-full !pl-9 !text-xs placeholder:text-xs focus-visible:border-ring focus-visible:ring-1 focus-visible:ring-ring/60 focus-visible:ring-offset-0",
          props.inputClassName,
        )}
      />
    </div>
  )
}

function EmptyState(props: { title: string; description: string; compact?: boolean }) {
  return (
    <div
      className={cn(
        "flex min-h-[240px] items-center justify-center rounded-xl border border-dashed border-border/70 bg-card/40 px-6 text-center",
        props.compact ? "min-h-[160px]" : "",
      )}
    >
      <div className="max-w-md space-y-2">
        <div className="text-sm font-semibold text-foreground">{props.title}</div>
        <div className="text-sm text-muted-foreground">{props.description}</div>
      </div>
    </div>
  )
}

type LibraryResourcesPanelProps = {
  libraries: LibraryDTO[]
  focusedLibrary?: LibraryDTO
  sidebarFiles: LibraryFileDTO[]
  sidebarRecords: LibraryHistoryRecordDTO[]
  httpBaseURL: string
  language: string
  t: Translator
  onSelectLibrary: (libraryId: string) => void
  onClearSelection: () => void
  onLibraryDeleted: (libraryId: string) => void
  onPreviewImage: (preview: LibraryImagePreview) => void
  onOpenLibraryFile: (file: LibraryFileDTO) => void
  onOpenTaskDialog: (operationId: string) => void
}

function LibraryResourcesPanel(props: LibraryResourcesPanelProps) {
  const {
    libraries,
    focusedLibrary,
    sidebarFiles,
    sidebarRecords,
    httpBaseURL,
    language,
    t,
    onSelectLibrary,
    onClearSelection,
    onLibraryDeleted,
    onPreviewImage,
    onOpenLibraryFile,
    onOpenTaskDialog,
  } = props
  const renameLibrary = useRenameLibrary()
  const deleteLibrary = useDeleteLibrary()
  const workspaceProjectQuery = useGetWorkspaceProject(
    focusedLibrary?.id ?? "",
    Boolean(focusedLibrary?.id),
  )
  const [isEditingLibraryName, setIsEditingLibraryName] = React.useState(false)
  const [libraryNameDraft, setLibraryNameDraft] = React.useState("")
  const [deleteDialogOpen, setDeleteDialogOpen] = React.useState(false)
  const [visibleLibraryCount, setVisibleLibraryCount] = React.useState(RESOURCE_LIBRARY_GRID_BATCH_SIZE)
  const libraryNameInputRef = React.useRef<HTMLInputElement | null>(null)
  const workspaceTrackLabelByFileId = React.useMemo(
    () => buildWorkspaceTrackLabelByFileIdMap(workspaceProjectQuery.data),
    [workspaceProjectQuery.data],
  )
  const librarySummaryMap = React.useMemo(
    () => new Map(libraries.map((library) => [library.id, summarizeLibraryFiles(library.files ?? [])])),
    [libraries],
  )
  const visibleLibraries = React.useMemo(
    () => libraries.slice(0, visibleLibraryCount),
    [libraries, visibleLibraryCount],
  )
  const canLoadMoreLibraries = visibleLibraryCount < libraries.length
  const focusedLibraryName = focusedLibrary?.name?.trim() || focusedLibrary?.id || ""
  const focusedSummary =
    (focusedLibrary ? librarySummaryMap.get(focusedLibrary.id) : undefined) ?? {
      videos: 0,
      subtitles: 0,
      thumbnails: 0,
      deletedCount: 0,
      totalSizeBytes: 0,
    }
  const focusedFileCount = React.useMemo(
    () => focusedLibrary?.files?.length ?? 0,
    [focusedLibrary?.files],
  )
  const focusedTaskCount = React.useMemo(
    () => (focusedLibrary?.records.history ?? []).filter((record) => record.category === "operation").length,
    [focusedLibrary?.records.history],
  )
  const focusedRecordCount = React.useMemo(
    () =>
      (focusedLibrary?.records.history?.length ?? 0) +
      (focusedLibrary?.records.fileEvents?.length ?? 0) +
      (focusedLibrary?.records.workspaceStates?.length ?? 0),
    [focusedLibrary?.records.fileEvents?.length, focusedLibrary?.records.history?.length, focusedLibrary?.records.workspaceStates?.length],
  )
  const focusedCoverFile = React.useMemo(
    () => resolveLibraryCoverFile(focusedLibrary),
    [focusedLibrary],
  )
  const focusedCoverURL = React.useMemo(
    () => buildAssetPreviewURL(httpBaseURL, focusedCoverFile?.storage.localPath?.trim() ?? ""),
    [focusedCoverFile, httpBaseURL],
  )
  const focusedCoverPreview = React.useMemo<LibraryImagePreview | null>(() => {
    const path = focusedCoverFile?.storage.localPath?.trim() ?? ""
    if (!path) {
      return null
    }
    return {
      id: focusedCoverFile?.id ?? `library-cover:${focusedLibrary?.id ?? ""}`,
      name: focusedCoverFile?.displayLabel?.trim() || focusedCoverFile?.name?.trim() || focusedLibraryName,
      path,
    }
  }, [focusedCoverFile, focusedLibrary?.id, focusedLibraryName])
  const focusedSummaryText = React.useMemo(
    () =>
      formatTemplate(
        t("library.workspace.libraryMeta"),
        {
          videos: focusedSummary.videos,
          subtitles: focusedSummary.subtitles,
          thumbnails: focusedSummary.thumbnails,
        },
      ),
    [focusedSummary.subtitles, focusedSummary.thumbnails, focusedSummary.videos, t],
  )

  React.useEffect(() => {
    setIsEditingLibraryName(false)
    setLibraryNameDraft(focusedLibraryName)
    setDeleteDialogOpen(false)
  }, [focusedLibrary?.id, focusedLibraryName])

  React.useEffect(() => {
    setVisibleLibraryCount(RESOURCE_LIBRARY_GRID_BATCH_SIZE)
  }, [libraries.length, libraries[0]?.id])

  React.useEffect(() => {
    if (!isEditingLibraryName) {
      return
    }
    libraryNameInputRef.current?.focus()
    libraryNameInputRef.current?.select()
  }, [isEditingLibraryName])

  const handleLibraryNameCommit = React.useCallback(async () => {
    if (!focusedLibrary || renameLibrary.isPending) {
      return
    }
    const currentName = focusedLibrary.name?.trim() || focusedLibrary.id
    const trimmed = libraryNameDraft.trim()
    if (!trimmed || trimmed === currentName) {
      setLibraryNameDraft(currentName)
      setIsEditingLibraryName(false)
      return
    }
    try {
      await renameLibrary.mutateAsync({ libraryId: focusedLibrary.id, name: trimmed })
      setLibraryNameDraft(trimmed)
      setIsEditingLibraryName(false)
    } catch (error) {
      setLibraryNameDraft(currentName)
      setIsEditingLibraryName(false)
      messageBus.publishToast({
        intent: "danger",
        title: t("library.rowMenu.renameTitle"),
        description: resolveErrorMessage(error, t("library.errors.unknown")),
      })
    }
  }, [focusedLibrary, libraryNameDraft, renameLibrary, t])

  const handleDeleteLibrary = React.useCallback(async () => {
    if (!focusedLibrary || deleteLibrary.isPending) {
      return
    }
    try {
      await deleteLibrary.mutateAsync({ libraryId: focusedLibrary.id })
      setDeleteDialogOpen(false)
      onLibraryDeleted(focusedLibrary.id)
      messageBus.publishToast({
        intent: "success",
        title: t("library.resources.deleteLibraryTitle"),
        description: t("library.resources.deleteLibrarySuccess"),
      })
    } catch (error) {
      messageBus.publishToast({
        intent: "danger",
        title: t("library.resources.deleteLibraryTitle"),
        description: resolveErrorMessage(error, t("library.errors.unknown")),
      })
    }
  }, [deleteLibrary, focusedLibrary, onLibraryDeleted, t])

  if (!libraries.length) {
    return (
      <PanelCard tone="solid" className="min-h-0 flex-1 p-4">
        <EmptyState
          title={t("library.resources.filteredEmpty")}
          description={t("library.resources.filteredEmptyDescription")}
          compact
        />
      </PanelCard>
    )
  }

  return (
    <PanelCard tone="solid" className="min-h-0 flex-1 overflow-hidden p-0">
      <div className="grid h-full min-h-0 xl:grid-cols-[minmax(0,1fr)_392px]">
        <div className="min-h-0 overflow-auto p-3" onClick={onClearSelection}>
          <div className="grid grid-cols-[repeat(auto-fill,minmax(176px,1fr))] gap-3">
              {visibleLibraries.map((library) => {
                const title = library.name?.trim() || library.id
                const coverURL = buildAssetPreviewURL(httpBaseURL, resolveLibraryCoverPath(library))
                const summary = librarySummaryMap.get(library.id) ?? {
                  videos: 0,
                  subtitles: 0,
                  thumbnails: 0,
                  deletedCount: 0,
                  totalSizeBytes: 0,
                }
                const isSelected = focusedLibrary?.id === library.id
                return (
                  <button
                    key={library.id}
                    type="button"
                    className={cn(
                      "group flex h-[182px] min-w-0 flex-col overflow-hidden rounded-2xl border border-border/60 bg-background/[0.56] text-left shadow-sm transition-colors motion-safe:duration-150",
                      isSelected
                        ? "border-primary/80 bg-accent/40 ring-1 ring-inset ring-primary/35"
                        : "hover:border-border/80 hover:bg-accent/32",
                    )}
                    onClick={(event) => {
                      event.stopPropagation()
                      onSelectLibrary(library.id)
                    }}
                    title={title}
                  >
                    <div className="h-[102px] w-full shrink-0 overflow-hidden bg-muted/30">
                      <img
                        src={coverURL || defaultThumbnail}
                        alt={title}
                        className="h-full w-full object-cover transition-transform duration-200 group-hover:scale-[1.02]"
                        loading="lazy"
                        onError={(event) => {
                          event.currentTarget.src = defaultThumbnail
                        }}
                      />
                    </div>

                    <div className="flex min-h-0 flex-1 flex-col border-t border-border/50 bg-muted/40 px-3 py-2">
                      <div className="min-w-0 truncate text-[13px] font-medium leading-5 text-foreground">{title}</div>
                      <div className="mt-1.5 flex items-center gap-2 overflow-hidden text-xs leading-4 text-muted-foreground">
                        <LibrarySummaryInlineMetric
                          icon={Video}
                          value={summary.videos}
                          tooltip={`${summary.videos} ${t("library.tabs.video")}`}
                        />
                        <LibrarySummaryInlineMetric
                          icon={Captions}
                          value={summary.subtitles}
                          tooltip={`${summary.subtitles} ${t("library.tabs.subtitle")}`}
                        />
                        <LibrarySummaryInlineMetric
                          icon={ImageIcon}
                          value={summary.thumbnails}
                          tooltip={`${summary.thumbnails} ${t("library.tabs.thumbnail")}`}
                        />
                        {summary.deletedCount > 0 ? (
                          <LibrarySummaryInlineMetric
                            icon={Trash2}
                            value={summary.deletedCount}
                            tooltip={formatTemplate(t("library.resources.deletedFileCount"), {
                              count: summary.deletedCount,
                            })}
                          />
                        ) : null}
                      </div>
                      <div className="mt-auto truncate text-xs leading-4 text-muted-foreground">
                        {t("library.resources.updatedAt")} · {formatRelativeTime(library.updatedAt, language)}
                      </div>
                    </div>
                  </button>
                )
              })}
          </div>
          {canLoadMoreLibraries ? (
            <div className="flex justify-center pt-3">
              <Button
                type="button"
                variant="outline"
                size="compact"
                onClick={(event) => {
                  event.stopPropagation()
                  setVisibleLibraryCount((current) => current + RESOURCE_LIBRARY_GRID_BATCH_SIZE)
                }}
              >
                {t("library.resources.loadMore")}
              </Button>
            </div>
          ) : null}
        </div>

        <div className="min-h-0 border-t border-border/70 bg-background/10 xl:border-l xl:border-t-0">
          <div className="flex h-full min-h-0 flex-col p-3">
            <div className="min-w-0 space-y-3">
              {focusedLibrary ? (
                <>
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0 flex-1 space-y-1.5">
                      {isEditingLibraryName ? (
                        <Input
                          ref={libraryNameInputRef}
                          size="compact"
                          value={libraryNameDraft}
                          onChange={(event) => setLibraryNameDraft(event.target.value)}
                          onBlur={() => void handleLibraryNameCommit()}
                          onKeyDown={(event) => {
                            if (event.key === "Enter") {
                              event.preventDefault()
                              void handleLibraryNameCommit()
                            }
                            if (event.key === "Escape") {
                              event.preventDefault()
                              setLibraryNameDraft(focusedLibraryName)
                              setIsEditingLibraryName(false)
                            }
                          }}
                          className="h-8 text-sm"
                          disabled={renameLibrary.isPending}
                          placeholder={t("library.rowMenu.renamePlaceholder")}
                        />
                      ) : (
                        <div className="truncate text-base font-semibold text-foreground">{focusedLibraryName}</div>
                      )}
                      <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                        <span>{focusedSummaryText}</span>
                        {focusedSummary.deletedCount > 0 ? (
                          <span className="inline-flex items-center rounded-md border border-border/70 bg-muted/20 px-2 py-0.5">
                            {formatTemplate(t("library.resources.deletedFileCount"), {
                              count: focusedSummary.deletedCount,
                            })}
                          </span>
                        ) : null}
                        <span className="inline-flex items-center rounded-md border border-border/70 bg-muted/20 px-2 py-0.5">
                          {t("library.columns.size")} · {formatBytes(focusedSummary.totalSizeBytes)}
                        </span>
                      </div>
                    </div>
                    <div className={DASHBOARD_CONTROL_GROUP_CLASS}>
                      {!isEditingLibraryName ? (
                        <Tooltip>
                          <TooltipTrigger asChild>
                            <Button
                              type="button"
                              variant="ghost"
                              size="compactIcon"
                              className="h-7 w-7 rounded-none border-0 text-muted-foreground hover:text-foreground"
                              onClick={() => setIsEditingLibraryName(true)}
                              disabled={renameLibrary.isPending}
                              aria-label={t("library.rowMenu.renameTitle")}
                            >
                              {renameLibrary.isPending ? (
                                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                              ) : (
                                <PencilLine className="h-3.5 w-3.5" />
                              )}
                            </Button>
                          </TooltipTrigger>
                          <TooltipContent>{t("library.rowMenu.renameTitle")}</TooltipContent>
                        </Tooltip>
                      ) : null}
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <Button
                            type="button"
                            variant="ghost"
                            size="compactIcon"
                            className={cn(
                              "h-7 w-7 rounded-none border-0 text-muted-foreground hover:text-destructive",
                              !isEditingLibraryName ? "border-l border-border/70" : "",
                            )}
                            onClick={() => setDeleteDialogOpen(true)}
                            disabled={deleteLibrary.isPending}
                            aria-label={t("library.resources.deleteLibraryTitle")}
                          >
                            {deleteLibrary.isPending ? (
                              <Loader2 className="h-3.5 w-3.5 animate-spin" />
                            ) : (
                              <Trash2 className="h-3.5 w-3.5" />
                            )}
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>{t("library.resources.deleteLibraryTitle")}</TooltipContent>
                      </Tooltip>
                    </div>
                  </div>
                </>
              ) : (
                <>
                  <div className="truncate text-base font-semibold text-foreground">
                    {t("library.resources.currentRecordsTitle")}
                  </div>
                </>
              )}
            </div>
            <Separator className="mt-3 bg-border/70" />
            <div className="min-h-0 flex-1 overflow-auto pr-1 pt-3">
                {focusedLibrary ? (
                  <div className="space-y-4">
                    <div className="overflow-hidden rounded-xl border border-border/70 bg-muted/25">
                      <div className="relative aspect-[16/8] bg-muted/30">
                        {focusedCoverPreview ? (
                          <button
                            type="button"
                            className="h-full w-full cursor-zoom-in"
                            onClick={() => onPreviewImage(focusedCoverPreview)}
                            aria-label={t("library.preview.imageTitle")}
                          >
                            <img
                              src={focusedCoverURL || defaultThumbnail}
                              alt={focusedLibrary.name?.trim() || focusedLibrary.id}
                              className="h-full w-full object-cover transition-transform duration-200 hover:scale-[1.01]"
                              onError={(event) => {
                                event.currentTarget.src = defaultThumbnail
                              }}
                            />
                          </button>
                        ) : (
                          <img
                            src={focusedCoverURL || defaultThumbnail}
                            alt={focusedLibrary.name?.trim() || focusedLibrary.id}
                            className="h-full w-full object-cover"
                            onError={(event) => {
                              event.currentTarget.src = defaultThumbnail
                            }}
                          />
                        )}
                      </div>
                    </div>

                    <div className="grid gap-2 sm:grid-cols-2">
                      <LibraryResourceMetricCard label={t("library.tabs.video")} value={String(focusedSummary.videos)} />
                      <LibraryResourceMetricCard label={t("library.tabs.subtitle")} value={String(focusedSummary.subtitles)} />
                      <LibraryResourceMetricCard label={t("library.tabs.thumbnail")} value={String(focusedSummary.thumbnails)} />
                      <LibraryResourceMetricCard
                        label={t("library.records.title")}
                        value={String(focusedLibrary.records.history?.length ?? 0)}
                      />
                    </div>

                    <Separator className="bg-border/70" />

                    <LibraryResourceSidebarSection
                      title={t("library.resources.fileInfoTitle")}
                      badge={
                        <Badge variant="outline" className="h-5 text-[10px]">
                          {sidebarFiles.length}
                        </Badge>
                      }
                    >
                      {sidebarFiles.length > 0 ? (
                        <div className="space-y-2">
                          {sidebarFiles.map((file) => {
                            const title = resolveWorkspaceAwareLibraryFileLabel(
                              file,
                              workspaceTrackLabelByFileId,
                            )
                            const metaLine = [
                              resolveLibraryKindLabel(file.kind, t),
                              resolveLibraryFileFormat(file),
                              formatBytes(file.media?.sizeBytes),
                              formatRelativeTime(file.updatedAt || file.createdAt, language),
                            ]
                              .filter(Boolean)
                              .join(" · ")
                            return (
                              <Tooltip key={file.id}>
                                <TooltipTrigger asChild>
                                  <button
                                    type="button"
                                    className="flex w-full items-start justify-between gap-3 rounded-lg border border-border/70 bg-muted/20 px-3 py-3 text-left transition-colors hover:bg-accent/40"
                                    onClick={() => onOpenLibraryFile(file)}
                                  >
                                    <div className="min-w-0">
                                      <div className="truncate text-sm font-medium text-foreground">{title}</div>
                                      <div className="mt-1 text-xs text-muted-foreground">{metaLine || "-"}</div>
                                    </div>
                                    <ArrowRight className="mt-0.5 h-4 w-4 shrink-0 text-muted-foreground" />
                                  </button>
                                </TooltipTrigger>
                                <TooltipContent>
                                  {t("library.resources.fileInfoTooltip")}
                                </TooltipContent>
                              </Tooltip>
                            )
                          })}
                        </div>
                      ) : (
                        <div className="rounded-lg border border-dashed border-border/70 px-3 py-4 text-xs text-muted-foreground">
                          {t("library.resources.fileInfoEmpty")}
                        </div>
                      )}
                    </LibraryResourceSidebarSection>

                    <Separator className="bg-border/70" />

                    <LibraryResourceSidebarSection
                      title={t("library.resources.recordInfoTitle")}
                      badge={
                        <Badge variant="outline" className="h-5 text-[10px]">
                          {sidebarRecords.length}
                        </Badge>
                      }
                    >
                      {sidebarRecords.length > 0 ? (
                        <ResourceRecordTimeline records={sidebarRecords} onOpenTaskDialog={onOpenTaskDialog} t={t} language={language} />
                      ) : (
                        <EmptyState
                          title={t("library.records.empty")}
                          description={t("library.records.emptyDescription")}
                          compact
                        />
                      )}
                    </LibraryResourceSidebarSection>

                    <Separator className="bg-border/70" />

                    <LibraryResourceSidebarSection title={t("library.resources.libraryInfoTitle")}>
                      <div className="space-y-2">
                        <LibraryResourceInfoRow label={t("library.columns.id")} value={focusedLibrary.id} />
                        <LibraryResourceInfoRow
                          label={t("library.resources.updatedAt")}
                          value={formatRelativeTime(focusedLibrary.updatedAt, language)}
                          title={focusedLibrary.updatedAt}
                        />
                        <LibraryResourceInfoRow
                          label={t("library.columns.createTime")}
                          value={formatRelativeTime(focusedLibrary.createdAt, language)}
                          title={focusedLibrary.createdAt}
                        />
                      </div>
                    </LibraryResourceSidebarSection>
                  </div>
                ) : sidebarRecords.length > 0 ? (
                  <ResourceRecordTimeline records={sidebarRecords} onOpenTaskDialog={onOpenTaskDialog} t={t} language={language} />
                ) : (
                  <EmptyState
                    title={t("library.records.empty")}
                    description={t("library.records.emptyDescription")}
                    compact
                  />
                )}
            </div>
          </div>
        </div>
      </div>
      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DashboardDialogContent size="compact">
          <DashboardDialogHeader>
            <DialogTitle>{t("library.resources.deleteLibraryConfirmTitle")}</DialogTitle>
            <DialogDescription>
              {t("library.resources.deleteLibraryConfirmDescription")}
            </DialogDescription>
          </DashboardDialogHeader>
          {focusedLibrary ? (
            <DashboardDialogBody className="space-y-3">
              <DashboardDialogSection tone="field" className="text-center text-sm text-foreground">
                {focusedLibraryName}
              </DashboardDialogSection>
              <DashboardDialogSection className="grid gap-2 sm:grid-cols-3">
                <LibraryResourceMetricCard label={t("library.resources.deleteLibraryFiles")} value={String(focusedFileCount)} />
                <LibraryResourceMetricCard label={t("library.resources.deleteLibraryTasks")} value={String(focusedTaskCount)} />
                <LibraryResourceMetricCard label={t("library.resources.deleteLibraryRecords")} value={String(focusedRecordCount)} />
              </DashboardDialogSection>
            </DashboardDialogBody>
          ) : null}
          <DashboardDialogFooter>
            <Button
              type="button"
              variant="ghost"
              size="compact"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleteLibrary.isPending}
            >
              {t("library.rowMenu.renameCancel")}
            </Button>
            <Button
              type="button"
              variant="destructive"
              size="compact"
              onClick={() => void handleDeleteLibrary()}
              disabled={deleteLibrary.isPending}
            >
              {deleteLibrary.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
              {t("library.resources.deleteLibraryConfirmAction")}
            </Button>
          </DashboardDialogFooter>
        </DashboardDialogContent>
      </Dialog>
    </PanelCard>
  )
}

function LibraryResourceSidebarSection(props: {
  title: string
  description?: string
  badge?: React.ReactNode
  className?: string
  bodyClassName?: string
  children: React.ReactNode
}) {
  return (
    <section className={cn("space-y-3", props.className)}>
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-muted-foreground">{props.title}</div>
          {props.description ? <div className="text-xs text-muted-foreground">{props.description}</div> : null}
        </div>
        {props.badge}
      </div>
      <div className={props.bodyClassName}>{props.children}</div>
    </section>
  )
}

function LibraryResourceInfoRow(props: { label: string; value: string; title?: string }) {
  return (
    <div className="grid grid-cols-[92px_minmax(0,1fr)] items-start gap-3 text-xs">
      <span className="text-muted-foreground">{props.label}</span>
      <span className="min-w-0 break-words text-right text-foreground" title={props.title ?? props.value}>
        {props.value}
      </span>
    </div>
  )
}

function LibraryResourceMetricCard(props: { label: string; value: string }) {
  return (
    <div className="rounded-lg border border-border/70 bg-muted/20 px-3 py-2.5">
      <div className="text-[11px] font-medium uppercase tracking-[0.14em] text-muted-foreground">{props.label}</div>
      <div className="mt-1 text-sm font-semibold tracking-tight text-foreground">{props.value}</div>
    </div>
  )
}

function LibrarySummaryInlineMetric(props: {
  icon: typeof Video
  value: number
  tooltip: string
}) {
  const Icon = props.icon
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className="inline-flex shrink-0 items-center gap-1 whitespace-nowrap text-xs text-muted-foreground">
          <Icon className="h-3.5 w-3.5" />
          <span className="font-medium tabular-nums text-foreground">{props.value}</span>
        </span>
      </TooltipTrigger>
      <TooltipContent>{props.tooltip}</TooltipContent>
    </Tooltip>
  )
}

function ResourceRecordTimeline(props: {
  records: LibraryHistoryRecordDTO[]
  onOpenTaskDialog: (operationId: string) => void
  t: Translator
  language: string
}) {
  const { records, onOpenTaskDialog, t, language } = props
  const [visibleCount, setVisibleCount] = React.useState(RESOURCE_RECORD_BATCH_SIZE)
  const visibleRecords = React.useMemo(
    () => records.slice(0, visibleCount),
    [records, visibleCount],
  )
  const canLoadMore = visibleCount < records.length

  React.useEffect(() => {
    setVisibleCount(RESOURCE_RECORD_BATCH_SIZE)
  }, [records.length, records[0]?.recordId])

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        {visibleRecords.map((record) => (
          <ResourceRecordTimelineItem
            key={record.recordId}
            record={record}
            isLast={false}
            onOpenTaskDialog={onOpenTaskDialog}
            t={t}
            language={language}
          />
        ))}
      </div>
      {canLoadMore ? (
        <div className="flex justify-center border-t border-border/60 pt-2">
          <Button
            type="button"
            variant="outline"
            size="compact"
            onClick={() => setVisibleCount((current) => current + RESOURCE_RECORD_BATCH_SIZE)}
          >
            {t("library.resources.loadMore")}
          </Button>
        </div>
      ) : null}
    </div>
  )
}

function ResourceRecordTimelineItem(props: {
  record: LibraryHistoryRecordDTO
  isLast: boolean
  onOpenTaskDialog: (operationId: string) => void
  t: Translator
  language: string
}) {
  const { record, onOpenTaskDialog, t, language } = props
  const actionLabel = resolveHistoryActionLabel(record.action, t)
  const categoryLabel = resolveHistoryCategoryLabel(record.category, t)
  const fileCount = record.files?.length ?? record.metrics.fileCount ?? 0
  const metaLine = [
    categoryLabel,
    actionLabel,
    fileCount > 0 ? formatTemplate(t("library.records.fileCount"), { count: fileCount }) : "",
    formatBytes(record.metrics.totalSizeBytes ?? undefined),
    formatRelativeTime(record.occurredAt, language),
  ]
    .filter(Boolean)
    .join(" · ")
  const clickable = Boolean(record.refs.operationId)
  const statusClassName = resolveResourceRecordStatusClassName(record.status)
  const itemContent = (
    <div
      className={cn(
        "flex items-start gap-3 rounded-lg border border-border/70 bg-muted/20 px-3 py-3",
        clickable ? "transition-colors group-hover:bg-accent/40" : "",
      )}
    >
      <span className={cn("mt-1.5 h-2 w-2 shrink-0 rounded-full", statusClassName)} />
      <div className="min-w-0 flex-1">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="truncate text-sm font-medium text-foreground">{record.displayName}</div>
            <div className="mt-1 text-xs text-muted-foreground">{metaLine || "-"}</div>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            {clickable ? (
              <Badge variant="outline" className="h-6 rounded-md px-1.5 text-xs">
                <ArrowRight className="h-3.5 w-3.5 text-muted-foreground" />
              </Badge>
            ) : null}
          </div>
        </div>
        {record.operationMeta?.errorMessage ? (
          <div
            className="mt-1 truncate text-xs text-destructive"
            title={record.operationMeta.errorMessage}
          >
            {record.operationMeta.errorMessage}
          </div>
        ) : null}
      </div>
    </div>
  )

  if (!clickable) {
    return itemContent
  }

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <button type="button" className="group block w-full text-left" onClick={() => onOpenTaskDialog(record.refs.operationId ?? "")}>
          {itemContent}
        </button>
      </TooltipTrigger>
      <TooltipContent>{t("library.resources.recordInfoTooltip")}</TooltipContent>
    </Tooltip>
  )
}

function ImportAssetDialog(props: {
  kind: "video" | "subtitle"
  filePath: string
  importTargetMode: ImportTargetMode
  currentLibrary?: LibraryDTO
  titleValue: string
  onTitleChange: (value: string) => void
  onModeChange: (value: ImportTargetMode) => void
  onClose: () => void
  onSelectFile: () => void
  onSubmit: () => void
  submitting: boolean
  canSubmit: boolean
  t: Translator
}) {
  const Icon = props.kind === "video" ? Video : Captions
  const hasFile = Boolean(props.filePath.trim())
  const fileName = getPathBaseName(props.filePath) || "-"
  const fileFormat = extractExtensionFromPath(props.filePath).toUpperCase() || "-"
  const libraryLabel = resolveLibraryDialogLabel(props.currentLibrary)
  const titlePreview = props.titleValue.trim() || stripPathExtension(getPathBaseName(props.filePath))
  const targetPreview =
    props.importTargetMode === "existing"
      ? libraryLabel.primary || props.t("library.import.targetCurrentEmpty")
      : props.t("library.import.targetNew")

  return (
    <>
      <DashboardDialogHeader>
        <DialogTitle>
          {props.kind === "video"
            ? props.t("library.actions.importVideo")
            : props.t("library.actions.importSubtitle")}
        </DialogTitle>
        <DialogDescription>
          {props.kind === "video"
            ? props.t("library.import.videoDialogDescription")
            : props.t("library.import.subtitleDialogDescription")}
        </DialogDescription>
      </DashboardDialogHeader>

      <DashboardDialogBody className="min-h-0 flex-1 space-y-3 overflow-y-auto pr-1">
        {!hasFile ? (
          <div className="flex min-h-[280px] items-center justify-center">
            <Button size="compact" onClick={props.onSelectFile}>
              <FilePlus2 className="h-4 w-4" />
              {props.t("library.import.selectFile")}
            </Button>
          </div>
        ) : (
          <>
            <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(220px,248px)]">
              <WorkspaceDialogHeaderCard
                title={props.t("library.task.summary")}
                badge={
                  <WorkspaceDialogSectionBadge>
                    <Icon className="h-3.5 w-3.5" />
                    {props.kind === "video"
                      ? props.t("library.actions.importVideo")
                      : props.t("library.actions.importSubtitle")}
                  </WorkspaceDialogSectionBadge>
                }
              >
                <div className="space-y-2">
                  <WorkspaceDialogSummaryRow label={props.t("library.import.fileName")} value={fileName} />
                  <WorkspaceDialogSummaryRow label={props.t("library.import.fileFormat")} value={fileFormat} />
                  <WorkspaceDialogSummaryRow label={props.t("library.import.targetMode")} value={targetPreview} />
                </div>
              </WorkspaceDialogHeaderCard>

              <WorkspaceDialogHeaderCard
                title={props.t("library.task.overview")}
                badge={
                  <WorkspaceDialogSectionBadge>
                    {props.t("library.import.dialog.ready")}
                  </WorkspaceDialogSectionBadge>
                }
              >
                <WorkspaceDialogMetricsCard
                  columns={2}
                  items={[
                    {
                      label: props.t("library.import.dialog.metrics.file"),
                      value: props.t("library.import.dialog.selected"),
                    },
                    {
                      label: props.t("library.import.dialog.metrics.target"),
                      value:
                        props.importTargetMode === "existing"
                          ? props.t("library.import.targetCurrentLabel")
                          : props.t("library.import.targetNew"),
                    },
                    {
                      label: props.t("library.import.dialog.metrics.title"),
                      value: titlePreview || "-",
                    },
                    {
                      label: props.t("library.import.dialog.metrics.naming"),
                      value: props.titleValue.trim()
                        ? props.t("library.import.dialog.customTitle")
                        : props.t("library.import.dialog.autoTitle"),
                    },
                  ]}
                />
              </WorkspaceDialogHeaderCard>
            </div>

            <ImportTargetCard
              importTargetMode={props.importTargetMode}
              currentLibrary={props.currentLibrary}
              onModeChange={props.onModeChange}
              t={props.t}
            />

            <ImportFileCard
              kind={props.kind}
              path={props.filePath}
              t={props.t}
            />

            <WorkspaceDialogSectionCard
              title={props.t("library.import.dialog.namingTitle")}
              description={props.t("library.import.dialog.namingDescription")}
            >
              <WorkspaceDialogFormRow
                label={props.t("library.tools.optionalTitle")}
                description={props.t("library.import.dialog.namingHint")}
                control={
                  <Input
                    value={props.titleValue}
                    onChange={(event) => props.onTitleChange(event.target.value)}
                    placeholder={props.t("library.tools.optionalTitle")}
                    className="h-8 text-xs"
                  />
                }
              />
            </WorkspaceDialogSectionCard>

          </>
        )}
      </DashboardDialogBody>

      <DashboardDialogFooter className="sm:justify-end">
        <div className="flex flex-col-reverse gap-2 sm:flex-row sm:items-center">
          <Button variant="outline" size="compact" onClick={props.onClose}>
            {props.t("common.close")}
          </Button>
          {hasFile ? (
            <Button variant="outline" size="compact" onClick={props.onSelectFile}>
              <FilePlus2 className="h-4 w-4" />
              {props.t("library.import.reselect")}
            </Button>
          ) : null}
          <Button size="compact" onClick={props.onSubmit} disabled={props.submitting || !props.canSubmit}>
            {props.submitting ? <Loader2 className="mr-2 h-4 w-4 animate-spin" /> : <Icon className="h-4 w-4" />}
            {props.t("library.tools.import")}
          </Button>
        </div>
      </DashboardDialogFooter>
    </>
  )
}

function ImportTargetCard(props: {
  importTargetMode: ImportTargetMode
  currentLibrary?: LibraryDTO
  onModeChange: (value: ImportTargetMode) => void
  t: Translator
}) {
  const libraryLabel = resolveLibraryDialogLabel(props.currentLibrary)

  return (
    <WorkspaceDialogSectionCard
      title={props.t("library.import.dialog.targetTitle")}
      description={props.t("library.import.dialog.targetDescription")}
      badge={<WorkspaceDialogSectionBadge>{props.t("library.workspace.dialogs.required")}</WorkspaceDialogSectionBadge>}
    >
      <div className="space-y-3">
        <WorkspaceDialogFormRow
          label={props.t("library.import.targetMode")}
          description={props.t("library.import.dialog.targetHint")}
          control={
            <Select
              value={props.importTargetMode}
              onChange={(event) => props.onModeChange(event.target.value === "existing" ? "existing" : "new")}
              className="h-8 w-full border-border/70 bg-background/80"
            >
              <option value="new">{props.t("library.import.targetNew")}</option>
              <option value="existing" disabled={!props.currentLibrary}>
                {props.t("library.import.targetCurrent")}
              </option>
            </Select>
          }
        />
        {props.importTargetMode === "existing" ? (
          <WorkspaceDialogItemsCard
            items={[
              {
                key: "current-library",
                label: props.t("library.import.targetCurrentLabel"),
                value: (
                  <span
                    className="block truncate text-xs text-muted-foreground"
                    title={libraryLabel.primary || props.t("library.import.targetCurrentEmpty")}
                  >
                    {libraryLabel.primary || props.t("library.import.targetCurrentEmpty")}
                  </span>
                ),
              },
            ]}
          />
        ) : null}
      </div>
    </WorkspaceDialogSectionCard>
  )
}

function ImportFileCard(props: { kind: "video" | "subtitle"; path: string; t: Translator }) {
  const formatLabel = extractExtensionFromPath(props.path).toUpperCase() || "-"
  const fileName = getPathBaseName(props.path) || "-"
  return (
    <WorkspaceDialogSectionCard
      title={
        props.kind === "video"
          ? props.t("library.actions.importVideo")
          : props.t("library.actions.importSubtitle")
      }
      description={props.t("library.import.dialog.fileDescription")}
    >
      <WorkspaceDialogItemsCard
        items={[
          {
            key: "file-name",
            label: props.t("library.import.fileName"),
            value: <span className="block truncate text-xs text-muted-foreground" title={fileName}>{fileName}</span>,
          },
          {
            key: "file-format",
            label: props.t("library.import.fileFormat"),
            value: <span className="text-xs text-muted-foreground">{formatLabel}</span>,
          },
          {
            key: "file-path",
            label: props.t("library.import.filePath"),
            value: <span className="block truncate text-xs text-muted-foreground" title={props.path || "-"}>{props.path || "-"}</span>,
          },
        ]}
      />
    </WorkspaceDialogSectionCard>
  )
}

function resolveLibraryDialogLabel(library?: LibraryDTO): { primary: string; secondary?: string } {
  const rawName = resolveEffectiveLibraryName(library, library?.files)?.trim() ?? ""
  if (!rawName) {
    return { primary: "" }
  }
  return { primary: rawName }
}

function buildWorkspaceTrackLabelByFileIdMap(workspaceProject?: WorkspaceProjectDTO) {
  const map = new Map<string, string>()
  for (const track of workspaceProject?.videoTracks ?? []) {
    const fileId = track.file?.id?.trim() ?? ""
    const label = track.display?.label?.trim() ?? ""
    if (fileId && label) {
      map.set(fileId, label)
    }
  }
  for (const track of workspaceProject?.subtitleTracks ?? []) {
    const fileId = track.file?.id?.trim() ?? ""
    const label = track.display?.label?.trim() ?? ""
    if (fileId && label) {
      map.set(fileId, label)
    }
  }
  return map
}

function resolveWorkspaceAwareLibraryFileLabel(
  file: Pick<LibraryFileDTO, "id" | "name" | "displayLabel">,
  workspaceTrackLabelByFileId: Map<string, string>,
) {
  const fileId = file.id?.trim() ?? ""
  if (fileId) {
    const trackLabel = workspaceTrackLabelByFileId.get(fileId)?.trim() ?? ""
    if (trackLabel) {
      return trackLabel
    }
  }
  const displayLabel = file.displayLabel?.trim() ?? ""
  if (displayLabel) {
    return displayLabel
  }
  const fileName = file.name?.trim() ?? ""
  if (fileName) {
    return fileName
  }
  return file.id
}

function filterLibrariesForResourceView(libraries: LibraryDTO[], query: string, t: Translator) {
  const keyword = query.trim().toLowerCase()
  if (!keyword) {
    return libraries
  }
  return libraries.filter((library) => {
    const fields = [
      library.name,
      library.id,
      ...(library.files ?? []).flatMap((file) => [file.name, file.displayLabel ?? "", file.kind, file.storage.localPath ?? ""]),
      ...(library.records.history ?? []).flatMap((record) => [
        record.displayName,
        record.action,
        resolveHistoryActionLabel(record.action, t),
        record.status,
        resolveHistoryStatusLabel(record.status, t),
      ]),
    ]
    return fields.some((field) => field.toLowerCase().includes(keyword))
  })
}

function filterLibraryFilesForSidebar(files: LibraryFileDTO[], query: string) {
  const workspaceFiles = files.filter((file) => canOpenLibraryWorkspaceFile(file))
  const keyword = query.trim().toLowerCase()
  if (!keyword) {
    return workspaceFiles
  }
  return workspaceFiles.filter((file) => {
    const fields = [file.name, file.displayLabel ?? "", file.kind, file.storage.localPath ?? ""]
    return fields.some((field) => field.toLowerCase().includes(keyword))
  })
}

function summarizeLibraryFiles(files: LibraryFileDTO[]) {
  return files.reduce(
    (summary, file) => {
      if (file.state.deleted) {
        summary.deletedCount += 1
        return summary
      }
      const kind = normalizeLibraryKind(file.kind)
      if (kind === "subtitle") {
        summary.subtitles += 1
      } else if (kind === "thumbnail") {
        summary.thumbnails += 1
      } else if (kind === "video" || kind === "audio" || kind === "transcode") {
        summary.videos += 1
      }
      summary.totalSizeBytes += file.media?.sizeBytes ?? 0
      return summary
    },
    { videos: 0, subtitles: 0, thumbnails: 0, deletedCount: 0, totalSizeBytes: 0 },
  )
}

function resolveLibraryCoverFile(library?: LibraryDTO) {
  if (!library) {
    return undefined
  }
  return sortByUpdatedAtDesc(
    (library.files ?? []).filter((file) => normalizeLibraryKind(file.kind) === "thumbnail" && !file.state.deleted),
  ).find((file) => Boolean(file.storage.localPath?.trim()))
}

function resolveLibraryCoverPath(library?: LibraryDTO) {
  return resolveLibraryCoverFile(library)?.storage.localPath?.trim() ?? ""
}

function canOpenLibraryWorkspaceFile(file?: LibraryFileDTO) {
  if (!file || file.state.deleted) {
    return false
  }
  const kind = normalizeLibraryKind(file?.kind)
  return kind === "video" || kind === "audio" || kind === "subtitle" || kind === "transcode"
}

function buildWorkspaceTargetFromLibraryFile(file: LibraryFileDTO, libraryFiles: LibraryFileDTO[]): LibraryWorkspaceTarget | null {
  if (!canOpenLibraryWorkspaceFile(file)) {
    return null
  }
  const rootFileId = file.lineage.rootFileId?.trim() || file.id
  const siblings = libraryFiles.filter((candidate) => {
    if (candidate.state.deleted) {
      return false
    }
    const candidateRootId = candidate.lineage.rootFileId?.trim() || candidate.id
    return candidateRootId === rootFileId
  })
  const linkedVideo = siblings.find((candidate) => {
    const kind = normalizeLibraryKind(candidate.kind)
    return kind === "video" || kind === "audio" || kind === "transcode"
  })
  const linkedSubtitle = siblings.find((candidate) => normalizeLibraryKind(candidate.kind) === "subtitle")
  const kind = normalizeLibraryKind(file.kind)
  return {
    libraryId: file.libraryId,
    fileId: file.id,
    name: file.name,
    fileType: file.kind,
    path: file.storage.localPath,
    openMode: kind === "subtitle" ? "subtitle" : "video",
    videoAssetId: kind === "subtitle" ? linkedVideo?.id ?? "" : undefined,
    videoPath: kind === "subtitle" ? linkedVideo?.storage.localPath ?? "" : undefined,
    videoName: kind === "subtitle" ? linkedVideo?.name ?? "" : undefined,
    subtitleAssetId: kind === "subtitle" ? undefined : linkedSubtitle?.id ?? "",
    subtitlePath: kind === "subtitle" ? undefined : linkedSubtitle?.storage.localPath ?? "",
    subtitleName: kind === "subtitle" ? undefined : linkedSubtitle?.name ?? "",
  }
}

function resolveLibraryKindLabel(kind: string, t: Translator) {
  switch (normalizeLibraryKind(kind)) {
    case "video":
      return t("library.type.video")
    case "audio":
      return t("library.type.audio")
    case "subtitle":
      return t("library.type.subtitle")
    case "thumbnail":
      return t("library.type.thumbnail")
    case "transcode":
      return t("library.type.transcode")
    default:
      return kind || t("library.type.other")
  }
}

function resolveLibraryFileFormat(file: LibraryFileDTO) {
  const explicit = file.media?.format?.trim()
  if (explicit) {
    return explicit.toUpperCase()
  }
  const pathExtension = extractExtensionFromPath(file.storage.localPath ?? "")
  if (pathExtension) {
    return pathExtension.toUpperCase()
  }
  const fileExtension = extractExtensionFromPath(file.name)
  if (fileExtension) {
    return fileExtension.toUpperCase()
  }
  return ""
}

function normalizeLibraryKind(kind?: string) {
  return kind?.trim().toLowerCase() ?? ""
}

function resolveHistoryCategoryLabel(category: string, t: Translator) {
  const normalized = category.trim().toLowerCase()
  switch (normalized) {
    case "operation":
      return t("library.records.category.operation")
    case "import":
      return t("library.records.category.import")
    default:
      return category
  }
}

function resolveHistoryActionLabel(action: string, t: Translator) {
  const normalized = action.trim().toLowerCase()
  switch (normalized) {
    case "download":
      return t("library.task.download")
    case "transcode":
      return t("library.workspace.actions.transcode")
    case "subtitle_translate":
      return t("library.workspace.actions.translate")
    case "subtitle_proofread":
      return t("library.workspace.actions.proofread")
    case "subtitle_qa_review":
      return t("library.workspace.header.qa")
    case "import_video":
      return t("library.actions.importVideo")
    case "import_subtitle":
      return t("library.actions.importSubtitle")
    default:
      return action
  }
}

function resolveHistoryStatusLabel(status: string, t: Translator) {
  const normalized = status.trim().toLowerCase()
  switch (normalized) {
    case "queued":
      return t("library.status.queued")
    case "running":
      return t("library.status.running")
    case "succeeded":
      return t("library.status.succeeded")
    case "failed":
      return t("library.status.failed")
    case "canceled":
      return t("library.status.canceled")
    default:
      return status
  }
}

function resolveResourceRecordStatusClassName(status: string) {
  const normalized = status.trim().toLowerCase()
  switch (normalized) {
    case "succeeded":
      return "bg-emerald-500"
    case "failed":
      return "bg-destructive"
    case "running":
      return "bg-sky-500"
    case "queued":
      return "bg-amber-500"
    case "canceled":
      return "bg-muted-foreground"
    default:
      return "bg-border"
  }
}

function resolveEffectiveLibraryName(library?: LibraryDTO, files?: LibraryFileDTO[]) {
  const rawName = library?.name?.trim() ?? ""
  const libraryID = library?.id?.trim() ?? ""
  if (!rawName) {
    return ""
  }
  if (!libraryID || rawName !== libraryID) {
    return rawName
  }
  const firstFile = [...(files ?? [])]
    .sort((left, right) => {
      const leftTime = Date.parse(left.createdAt)
      const rightTime = Date.parse(right.createdAt)
      const leftValue = Number.isNaN(leftTime) ? 0 : leftTime
      const rightValue = Number.isNaN(rightTime) ? 0 : rightTime
      return leftValue - rightValue
    })
    .find((file) => Boolean(file.name.trim()))
  if (!firstFile) {
    return rawName
  }
  return stripPathExtension(firstFile.name.trim()) || rawName
}

function toTaskRowFromOperation(
  operation: OperationListItemDTO,
  filesById: Map<string, LibraryFileDTO>,
  liveOperation: { sourceIcon?: string } | undefined,
  labels: LibraryLabelMaps,
  libraryNameByID: Map<string, string>,
): LibraryTaskRow {
  const outputs = (operation.outputFiles ?? []).map((output) => {
    const file = filesById.get(output.fileId)
    return {
      id: output.fileId,
      label: file?.name || output.fileId,
      path: file?.storage.localPath ?? "",
      fileType: file?.kind || output.kind,
      format: output.format || file?.media?.format,
      sourceType: file?.origin.kind,
      sourceLabel: labels.originLabels[file?.origin.kind ?? ""] ?? file?.origin.kind,
      isPrimary: output.isPrimary,
      deleted: output.deleted,
    } satisfies LibraryTaskOutput
  })
  const totalSize = operation.outputFiles?.reduce((sum, item) => sum + (item.sizeBytes ?? 0), 0) ?? operation.metrics.totalSizeBytes ?? 0
  return {
    id: operation.operationId,
    libraryId: operation.libraryId,
    libraryName: operation.libraryName ?? libraryNameByID.get(operation.libraryId) ?? operation.libraryId,
    name: operation.name,
    taskType: operation.kind,
    taskTypeLabel: labels.jobTypeLabels[operation.kind] ?? operation.kind,
    typeLabel: labels.originLabels[operation.kind] ?? labels.typeLabels.manual,
    platform: operation.platform,
    uploader: operation.uploader,
    status: operation.status,
    progress: toLibraryProgress(operation.progress),
    outputs: {
      count: operation.outputFiles?.length ?? operation.metrics.fileCount,
      sizeBytes: totalSize || null,
      totalCount: operation.metrics.fileCount,
      deletedCount: operation.outputFiles?.filter((item) => item.deleted).length ?? 0,
      totalSizeBytes: operation.metrics.totalSizeBytes ?? null,
      deletedSizeBytes: null,
    },
    outputTypes: dedupeStrings((operation.outputFiles ?? []).map((item) => item.kind)),
    duration: formatDuration(operation.startedAt, operation.finishedAt),
    publishedAt: operation.publishTime,
    startedAt: operation.startedAt,
    createdAt: operation.createdAt,
    outputFiles: outputs,
    sourceDomain: operation.domain,
    sourceIcon: operation.sourceIcon ?? liveOperation?.sourceIcon,
  }
}

function toFileRowFromDTO(file: LibraryFileDTO, operationsById: Map<string, OperationListItemDTO>, labels: LibraryLabelMaps): LibraryFileRow {
  const operationId = file.latestOperationId || file.origin.operationId || ""
  const operation = operationId ? operationsById.get(operationId) : undefined
  let status = file.state.deleted ? "deleted" : "ready"
  if (!file.state.deleted && operation?.status === "running") {
    status = "running"
  } else if (!file.state.deleted && operation?.status === "failed") {
    status = "failed"
  }
  return {
    id: file.id,
    libraryId: file.libraryId,
    name: file.name,
    displayLabel: file.displayLabel,
    fileType: file.kind,
    format: file.media?.format,
    sourceType: file.origin.kind,
    language: file.media?.language,
    cueCount: file.media?.cueCount,
    typeLabel: labels.originLabels[file.origin.kind] ?? labels.typeLabels.import,
    status,
    progress: toLibraryProgress(operation?.progress),
    sizeBytes: file.media?.sizeBytes ?? null,
    taskId: operationId || null,
    taskName: operation?.name ?? null,
    createdAt: file.createdAt,
    path: file.storage.localPath,
  }
}

function toLibraryProgress(progress: OperationListItemDTO["progress"]): LibraryProgress | null {
  if (!progress) {
    return null
  }
  return {
    label: progress.stage || progress.message || "In progress",
    percent: progress.percent,
    speed: progress.speed,
    detail: progress.message,
    updatedAt: progress.updatedAt,
  }
}

function mergeFiles(queryFiles: LibraryFileDTO[], liveFiles: LibraryFileDTO[]) {
  const map = new Map<string, LibraryFileDTO>()
  queryFiles.forEach((item) => map.set(item.id, item))
  liveFiles.forEach((item) => map.set(item.id, item))
  return Array.from(map.values()).sort((left, right) => right.updatedAt.localeCompare(left.updatedAt))
}

function mergeHistory(queryHistory: LibraryHistoryRecordDTO[], liveHistory: LibraryHistoryRecordDTO[]) {
  const map = new Map<string, LibraryHistoryRecordDTO>()
  queryHistory.forEach((item) => map.set(item.recordId, item))
  liveHistory.forEach((item) => map.set(item.recordId, item))
  return Array.from(map.values()).sort((left, right) => right.occurredAt.localeCompare(left.occurredAt))
}

function mergeOperations(queryOperations: OperationListItemDTO[], liveOperations: OperationListItemDTO[]) {
  const map = new Map<string, OperationListItemDTO>()
  queryOperations.forEach((item) => map.set(item.operationId, item))
  liveOperations.forEach((item) => map.set(item.operationId, item))
  return Array.from(map.values()).sort((left, right) => right.createdAt.localeCompare(left.createdAt))
}

function createPersistableModuleConfig(config: LibraryModuleConfigDTO): LibraryModuleConfigDTO {
  return {
    ...config,
    translateLanguages: {
      ...config.translateLanguages,
      custom: (config.translateLanguages.custom ?? []).filter((language) => Boolean(language.code?.trim())),
    },
    languageAssets: {
      ...config.languageAssets,
      glossaryProfiles: (config.languageAssets.glossaryProfiles ?? []).map((profile) => ({
        ...profile,
        terms: (profile.terms ?? []).filter((term) => isPersistableGlossaryTerm(term)),
      })),
    },
  }
}

function isPersistableGlossaryTerm(
  term: NonNullable<NonNullable<LibraryModuleConfigDTO["languageAssets"]["glossaryProfiles"]>[number]["terms"]>[number],
) {
  return Boolean(term.source?.trim()) && Boolean(term.target?.trim())
}

function filterTasksForTable(tasks: LibraryTaskRow[], filters: TaskTableFilters, query: string) {
  const statusSet = new Set(filters.statuses.map((status) => status.trim().toLowerCase()))
  const taskTypeSet = new Set(filters.taskTypes.map((taskType) => taskType.trim().toLowerCase()))
  const normalizedQuery = query.trim().toLowerCase()
  return tasks.filter((task) => {
    if (statusSet.size > 0 && !statusSet.has(task.status.trim().toLowerCase())) {
      return false
    }
    if (taskTypeSet.size > 0 && !taskTypeSet.has(task.taskType.trim().toLowerCase())) {
      return false
    }
    if (!normalizedQuery) {
      return true
    }
    const fields = [
      task.name,
      task.taskType,
      task.taskTypeLabel ?? "",
      task.typeLabel,
      task.platform ?? "",
      task.uploader ?? "",
      task.sourceDomain ?? "",
    ]
    return fields.some((field) => field.toLowerCase().includes(normalizedQuery))
  })
}

function filterFilesForTable(files: LibraryFileRow[], query: string) {
  const normalizedQuery = query.trim().toLowerCase()
  if (!normalizedQuery) {
    return files
  }
  return files.filter((file) => {
    const fields = [file.displayLabel ?? "", file.name, file.fileType, file.typeLabel, file.taskName ?? "", file.path ?? ""]
    return fields.some((field) => field.toLowerCase().includes(normalizedQuery))
  })
}

function filterFilesByResourceTypeAndStatus(
  files: LibraryFileRow[],
  typeFilter: ResourceFileTypeFilter,
  statusFilter: ResourceFileStatusFilter,
) {
  return files.filter((file) => {
    if (typeFilter === "video" && !["video", "audio", "transcode"].includes(file.fileType)) {
      return false
    }
    if (typeFilter === "subtitle" && file.fileType !== "subtitle") {
      return false
    }
    if (typeFilter === "all" && file.fileType === "thumbnail") {
      return false
    }
    if (statusFilter === "active" && file.status === "deleted") {
      return false
    }
    if (statusFilter === "deleted" && file.status !== "deleted") {
      return false
    }
    return true
  })
}

function filterHistory(history: LibraryHistoryRecordDTO[], query: string, t: Translator) {
  const keyword = query.trim().toLowerCase()
  if (!keyword) {
    return history
  }
  return history.filter((item) => {
    const fields = [
      item.displayName,
      item.category,
      resolveHistoryCategoryLabel(item.category, t),
      item.action,
      resolveHistoryActionLabel(item.action, t),
      item.status,
      resolveHistoryStatusLabel(item.status, t),
      item.importMeta?.importPath ?? "",
    ]
    return fields.some((field) => field.toLowerCase().includes(keyword))
  })
}

function sortByCreatedAtDesc<T extends { createdAt?: string }>(items: T[]) {
  return [...items].sort((a, b) => {
    const aTime = Date.parse(a.createdAt ?? "")
    const bTime = Date.parse(b.createdAt ?? "")
    const aValue = Number.isNaN(aTime) ? 0 : aTime
    const bValue = Number.isNaN(bTime) ? 0 : bTime
    return bValue - aValue
  })
}

function sortByUpdatedAtDesc<T extends { updatedAt?: string }>(items: T[]) {
  return [...items].sort((a, b) => {
    const aTime = Date.parse(a.updatedAt ?? "")
    const bTime = Date.parse(b.updatedAt ?? "")
    const aValue = Number.isNaN(aTime) ? 0 : aTime
    const bValue = Number.isNaN(bTime) ? 0 : bTime
    return bValue - aValue
  })
}

function columnsToOptions<TData>(columns: Array<{ id?: string; accessorKey?: string; enableHiding?: boolean; meta?: unknown }>) {
  return columns
    .filter((column) => column.enableHiding !== false)
    .map((column) => {
      const id = String(column.id ?? column.accessorKey ?? "")
      const meta = column.meta as { label?: string } | undefined
      return {
        id,
        label: meta?.label ?? id,
      }
    })
    .filter((column) => column.id)
}

function buildOverviewCards(tasks: LibraryTaskRow[], files: LibraryFileRow[], libraryCount: number, t: Translator) {
  const succeeded = tasks.filter((task) => task.status === "succeeded").length
  const running = tasks.filter((task) => task.status === "running").length
  const queued = tasks.filter((task) => task.status === "queued").length
  const totalSize = files.reduce((sum, file) => sum + (file.sizeBytes ?? 0), 0)
  const successRate = tasks.length > 0 ? Math.round((succeeded / tasks.length) * 100) : 0
  const recentCount = countRecentTasks(tasks, 7)
  return [
    {
      id: "files",
      label: t("library.overview.card.files"),
      value: String(files.length),
      detail: formatTemplate(t("library.overview.card.filesDetail"), { count: libraryCount }),
      icon: Database,
    },
    {
      id: "operations",
      label: t("library.overview.card.operations"),
      value: String(tasks.length),
      detail: formatTemplate(t("library.overview.card.operationsDetail"), { count: running + queued }),
      icon: ListChecks,
    },
    {
      id: "success",
      label: t("library.overview.card.success"),
      value: `${successRate}%`,
      detail: formatTemplate(t("library.overview.card.successDetail"), { count: succeeded }),
      icon: Activity,
    },
    {
      id: "storage",
      label: t("library.overview.card.storage"),
      value: totalSize > 0 ? formatBytes(totalSize) : "-",
      detail: formatTemplate(t("library.overview.card.storageDetail"), { count: recentCount }),
      icon: Sparkles,
    },
  ]
}

function buildLibraryTrendData(tasks: LibraryTaskRow[], granularity: string) {
  const now = new Date()
  const normalized = granularity.trim().toLowerCase()
  const bucketSizeMs = normalized === "1d" ? 60 * 60 * 1000 : 24 * 60 * 60 * 1000
  const bucketCount = normalized === "1d" ? 24 : normalized === "7d" ? 7 : 30
  const rangeStart = now.getTime() - bucketSizeMs * (bucketCount - 1)
  const buckets = Array.from({ length: bucketCount }).map((_, index) => {
    const start = new Date(rangeStart + index * bucketSizeMs)
    return {
      ts: start.getTime(),
      label:
        normalized === "1d"
          ? `${String(start.getHours()).padStart(2, "0")}:00`
          : `${start.getMonth() + 1}/${start.getDate()}`,
      success: 0,
      failed: 0,
    }
  })

  tasks.forEach((task) => {
    const timestamp = resolveTaskTimestamp(task)
    if (!timestamp || timestamp < rangeStart) {
      return
    }
    const index = Math.floor((timestamp - rangeStart) / bucketSizeMs)
    if (index < 0 || index >= buckets.length) {
      return
    }
    if (task.status === "succeeded") {
      buckets[index].success += 1
    } else if (task.status === "failed") {
      buckets[index].failed += 1
    }
  })

  return buckets.map((bucket) => ({ label: bucket.label, success: bucket.success, failed: bucket.failed }))
}

function resolveTaskTimestamp(task: LibraryTaskRow) {
  const candidate = task.startedAt || task.createdAt || ""
  const parsed = Date.parse(candidate)
  if (!Number.isFinite(parsed)) {
    return 0
  }
  return parsed
}

function countRecentTasks(tasks: LibraryTaskRow[], days: number) {
  const threshold = Date.now() - days * 24 * 60 * 60 * 1000
  return tasks.filter((task) => resolveTaskTimestamp(task) >= threshold).length
}

function resolveErrorMessage(error: unknown, fallback = "Unknown error") {
  if (error instanceof Error) {
    return error.message
  }
  if (typeof error === "string") {
    return error
  }
  return error ? String(error) : fallback
}

function isToolReady(tool?: ExternalTool) {
  if (!tool) {
    return false
  }
  return tool.status?.trim().toLowerCase() === "installed" && Boolean(tool.execPath?.trim())
}

function sameDependencyIssues(left: DependencyIssue[], right: DependencyIssue[]) {
  if (left.length !== right.length) {
    return false
  }
  for (let index = 0; index < left.length; index += 1) {
    if (left[index]?.name !== right[index]?.name || left[index]?.status !== right[index]?.status) {
      return false
    }
  }
  return true
}

function clampProgress(value?: number) {
  if (!Number.isFinite(value)) {
    return 0
  }
  const number = Number(value)
  if (number < 0) {
    return 0
  }
  if (number > 100) {
    return 100
  }
  return Math.round(number)
}

function pickDefaultFormat(formats: YtdlpFormatOption[]) {
  if (!formats || formats.length === 0) {
    return null
  }
  const videoFormats = formats.filter((format) => format.hasVideo)
  if (videoFormats.length > 0) {
    return videoFormats.reduce((best, current) => {
      const bestHeight = best.height ?? 0
      const currentHeight = current.height ?? 0
      if (currentHeight !== bestHeight) {
        return currentHeight > bestHeight ? current : best
      }
      const bestSize = best.filesize ?? 0
      const currentSize = current.filesize ?? 0
      return currentSize > bestSize ? current : best
    })
  }
  const audioFormats = formats.filter((format) => format.hasAudio)
  if (audioFormats.length > 0) {
    return audioFormats.reduce((best, current) => {
      const bestSize = best.filesize ?? 0
      const currentSize = current.filesize ?? 0
      return currentSize > bestSize ? current : best
    })
  }
  return formats[0]
}

function selectAudioFormatId(formats: YtdlpFormatOption[]) {
  if (!formats || formats.length === 0) {
    return ""
  }
  const audioFormats = formats.filter((format) => format.hasAudio && !format.hasVideo)
  if (audioFormats.length === 0) {
    return ""
  }
  const best = audioFormats.reduce((currentBest, current) => {
    const bestSize = currentBest.filesize ?? 0
    const currentSize = current.filesize ?? 0
    return currentSize > bestSize ? current : currentBest
  })
  return best.id
}

function pickDefaultTranscodePreset(file: LibraryFileRow, presets: TranscodePreset[]) {
  const fileType = file.fileType.trim().toLowerCase()
  if (fileType === "audio") {
    return presets.find((preset) => preset.outputType === "audio") ?? null
  }
  return presets.find((preset) => preset.outputType !== "audio") ?? null
}

function formatDomainLabel(domain?: string, url?: string) {
  const raw = (domain ?? "").trim()
  let host = raw
  if (!host && url) {
    try {
      host = new URL(url).hostname
    } catch {
      host = ""
    }
  }
  host = host.replace(/^www\./i, "")
  if (!host) {
    return ""
  }
  const parts = host.split(".")
  const label = parts.length > 1 ? parts.slice(0, -1).join(".") : host
  return label.toUpperCase()
}

function formatSubtitleLabel(subtitle: YtdlpSubtitleOption, t: Translator) {
  const name = subtitle.name?.trim() || subtitle.language?.trim() || subtitle.id
  const parts = [name]
  if (subtitle.ext) {
    parts.push(subtitle.ext.toUpperCase())
  }
  if (subtitle.isAuto) {
    parts.push(t("library.download.subtitle.auto"))
  }
  return parts.filter(Boolean).join(" · ")
}

function resolveDialogPath(selection: unknown) {
  if (typeof selection === "string") {
    return selection.trim()
  }
  if (Array.isArray(selection) && typeof selection[0] === "string") {
    return selection[0].trim()
  }
  return ""
}

function getPathBaseName(path: string) {
  if (!path) {
    return ""
  }
  const normalized = path.replace(/\\/g, "/")
  return normalized.split("/").pop()?.trim() ?? ""
}

function extractExtensionFromPath(path: string) {
  const baseName = getPathBaseName(path)
  if (!baseName) {
    return ""
  }
  const dotIndex = baseName.lastIndexOf(".")
  if (dotIndex <= 0 || dotIndex >= baseName.length - 1) {
    return ""
  }
  return baseName.slice(dotIndex + 1).trim().toLowerCase()
}

function stripPathExtension(fileName: string) {
  if (!fileName) {
    return ""
  }
  const dotIndex = fileName.lastIndexOf(".")
  if (dotIndex <= 0) {
    return fileName
  }
  return fileName.slice(0, dotIndex)
}

function dedupeStrings(values: string[]) {
  const result: string[] = []
  const seen = new Set<string>()
  values.forEach((value) => {
    const trimmed = value.trim()
    if (!trimmed || seen.has(trimmed)) {
      return
    }
    seen.add(trimmed)
    result.push(trimmed)
  })
  return result
}

function toggleMultiFilterValue<T extends string>(current: T[], value: T, checked: boolean): T[] {
  if (checked) {
    if (current.includes(value)) {
      return current
    }
    return [...current, value]
  }
  return current.filter((item) => item !== value)
}

function buildAssetPreviewURL(baseURL: string, path: string) {
  if (!baseURL || !path) {
    return ""
  }
  const trimmed = baseURL.replace(/\/+$/, "")
  const previewName = path.replace(/\\/g, "/").split("/").pop()?.trim() || "asset"
  return `${trimmed}/api/library/asset/${encodeURIComponent(previewName)}?path=${encodeURIComponent(path)}`
}
