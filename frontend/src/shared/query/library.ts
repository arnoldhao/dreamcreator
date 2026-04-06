import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query"

import * as LibraryBindings from "../../../bindings/dreamcreator/internal/application/library/dto/models"
import * as LibraryHandler from "../../../bindings/dreamcreator/internal/presentation/wails/libraryhandler"
import {
  parseApplySubtitleReviewSessionPayload,
  parseCheckYtdlpOperationFailurePayload,
  parseDiscardSubtitleReviewSessionPayload,
  parseFileEventPayload,
  parseGenerateSubtitleStylePreviewPayload,
  parseGenerateWorkspacePreviewPayload,
  parseGetYtdlpOperationLogPayload,
  parseLibraryFilePayload,
  parseLibraryHistoryPayload,
  parseLibraryListPayload,
  parseLibraryModuleConfigPayload,
  parseLibraryOperationPayload,
  parseLibraryPayload,
  parseOperationListPayload,
  parseParseYtdlpDownloadPayload,
  parsePrepareYtdlpDownloadPayload,
  parseResolveDomainIconPayload,
  parseRestoreSubtitleOriginalPayload,
  parseSubtitleConvertPayload,
  parseSubtitleExportPayload,
  parseSubtitleFixTyposPayload,
  parseSubtitleParsePayload,
  parseSubtitleReviewSessionPayload,
  parseSubtitleSavePayload,
  parseSubtitleValidatePayload,
  parseTranscodePresetListPayload,
  parseTranscodePresetPayload,
  parseWorkspaceProjectPayload,
  parseWorkspaceStatePayload,
} from "./library.contract"
import type {
  CancelOperationRequest,
  CheckYtdlpOperationFailureRequest,
  CheckYtdlpOperationFailureResponse,
  CreateSubtitleImportRequest,
  ApplySubtitleReviewSessionRequest,
  ApplySubtitleReviewSessionResult,
  CreateTranscodeJobRequest,
  DiscardSubtitleReviewSessionRequest,
  DiscardSubtitleReviewSessionResult,
  CreateVideoImportRequest,
  CreateYtdlpJobRequest,
  DeleteFileRequest,
  DeleteFilesRequest,
  DeleteLibraryRequest,
  DeleteOperationRequest,
  DeleteOperationsRequest,
  DeleteTranscodePresetRequest,
  FileEventRecordDTO,
  GetLibraryRequest,
  GetOperationRequest,
  GetSubtitleReviewSessionRequest,
  GetWorkspaceProjectRequest,
  GetYtdlpOperationLogRequest,
  GetYtdlpOperationLogResponse,
  GetWorkspaceStateRequest,
  LibraryDTO,
  LibraryFileDTO,
  LibraryHistoryRecordDTO,
  LibraryOperationDTO,
  ListFileEventsRequest,
  ListLibraryHistoryRequest,
  ListOperationsRequest,
  ListTranscodePresetsForDownloadRequest,
  OpenFileLocationRequest,
  OpenPathRequest,
  OperationListItemDTO,
  GenerateWorkspacePreviewVTTRequest,
  GenerateWorkspacePreviewVTTResult,
  GenerateSubtitleStylePreviewASSRequest,
  GenerateSubtitleStylePreviewASSResult,
  ParseYtdlpDownloadRequest,
  ParseYtdlpDownloadResponse,
  PrepareYtdlpDownloadRequest,
  PrepareYtdlpDownloadResponse,
  RenameLibraryRequest,
  ResolveDomainIconRequest,
  ResolveDomainIconResponse,
  ResumeOperationRequest,
  RestoreSubtitleOriginalRequest,
  RestoreSubtitleOriginalResult,
  RetryYtdlpOperationRequest,
  SaveWorkspaceStateRequest,
  SubtitleConvertRequest,
  SubtitleConvertResult,
  SubtitleExportRequest,
  SubtitleExportResult,
  SubtitleFixTyposRequest,
  SubtitleFixTyposResult,
  LibraryModuleConfigDTO,
  SubtitleProofreadRequest,
  SubtitleParseRequest,
  SubtitleParseResult,
  SubtitleQAReviewRequest,
  SubtitleSaveRequest,
  SubtitleSaveResult,
  SubtitleReviewSessionDetailDTO,
  SubtitleTranslateRequest,
  SubtitleValidateRequest,
  SubtitleValidateResult,
  TranscodePreset,
  UpdateLibraryModuleConfigRequest,
  WorkspaceProjectDTO,
  WorkspaceStateRecordDTO,
} from "@/shared/contracts/library"
import { useLibraryRealtimeStore } from "@/shared/store/libraryRealtime"

export const LIBRARY_LIST_QUERY_KEY = ["library", "libraries"] as const
export const LIBRARY_DETAIL_QUERY_KEY = ["library", "detail"] as const
export const LIBRARY_MODULE_CONFIG_QUERY_KEY = ["library", "module-config"] as const
export const LIBRARY_DEFAULT_MODULE_CONFIG_QUERY_KEY = ["library", "module-config-default"] as const
export const LIBRARY_OPERATIONS_QUERY_KEY = ["library", "operations"] as const
export const LIBRARY_HISTORY_QUERY_KEY = ["library", "history"] as const
export const LIBRARY_FILE_EVENTS_QUERY_KEY = ["library", "file-events"] as const
export const LIBRARY_WORKSPACE_QUERY_KEY = ["library", "workspace"] as const
export const LIBRARY_WORKSPACE_PROJECT_QUERY_KEY = ["library", "workspace-project"] as const
export const LIBRARY_TRANSCODE_PRESETS_QUERY_KEY = ["library", "transcode-presets"] as const
export const LIBRARY_TRANSCODE_PRESETS_FOR_DOWNLOAD_QUERY_KEY = ["library", "transcode-presets-download"] as const

function invalidateLibraryQueries(queryClient: ReturnType<typeof useQueryClient>, libraryId?: string) {
  queryClient.invalidateQueries({ queryKey: LIBRARY_LIST_QUERY_KEY })
  queryClient.invalidateQueries({ queryKey: LIBRARY_OPERATIONS_QUERY_KEY })
  queryClient.invalidateQueries({ queryKey: LIBRARY_HISTORY_QUERY_KEY })
  queryClient.invalidateQueries({ queryKey: LIBRARY_FILE_EVENTS_QUERY_KEY })
  if (libraryId) {
    queryClient.invalidateQueries({ queryKey: [...LIBRARY_DETAIL_QUERY_KEY, libraryId] })
    queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_QUERY_KEY, libraryId] })
    queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, libraryId] })
  } else {
    queryClient.invalidateQueries({ queryKey: LIBRARY_DETAIL_QUERY_KEY })
    queryClient.invalidateQueries({ queryKey: LIBRARY_WORKSPACE_QUERY_KEY })
    queryClient.invalidateQueries({ queryKey: LIBRARY_WORKSPACE_PROJECT_QUERY_KEY })
  }
}

function normalizeGeneratedValue(value: unknown): unknown {
  if (value === null) {
    return undefined
  }
  if (Array.isArray(value)) {
    return value.map((item) => normalizeGeneratedValue(item))
  }
  if (value && typeof value === "object") {
    return Object.fromEntries(
      Object.entries(value as Record<string, unknown>)
        .map(([key, item]) => [key, normalizeGeneratedValue(item)] as const)
        .filter(([, item]) => item !== undefined),
    )
  }
  return value
}

function parseGeneratedPayload<T>(value: unknown, parser: (input: unknown) => T): T {
  return parser(normalizeGeneratedValue(value))
}

export function useListLibraries() {
  return useQuery({
    queryKey: LIBRARY_LIST_QUERY_KEY,
    queryFn: async (): Promise<LibraryDTO[]> => {
      return parseGeneratedPayload((await LibraryHandler.ListLibraries()) ?? [], parseLibraryListPayload)
    },
    staleTime: 5_000,
  })
}

export function useGetLibrary(libraryId: string, enabled = true) {
  return useQuery({
    queryKey: [...LIBRARY_DETAIL_QUERY_KEY, libraryId],
    enabled: enabled && libraryId.trim().length > 0,
    queryFn: async (): Promise<LibraryDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetLibrary(
          LibraryBindings.GetLibraryRequest.createFrom({ libraryId } satisfies GetLibraryRequest),
        ),
        parseLibraryPayload,
      )
    },
    staleTime: 5_000,
  })
}

export function useRenameLibrary() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: RenameLibraryRequest): Promise<LibraryDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.RenameLibrary(LibraryBindings.RenameLibraryRequest.createFrom(request)),
        parseLibraryPayload,
      )
    },
    onMutate: async (request) => {
      await queryClient.cancelQueries({ queryKey: LIBRARY_LIST_QUERY_KEY })
      await queryClient.cancelQueries({ queryKey: [...LIBRARY_DETAIL_QUERY_KEY, request.libraryId] })
      const previousLibraries = queryClient.getQueryData<LibraryDTO[]>(LIBRARY_LIST_QUERY_KEY)
      const previousLibrary = queryClient.getQueryData<LibraryDTO>([...LIBRARY_DETAIL_QUERY_KEY, request.libraryId])
      const nextName = request.name.trim()
      const nextUpdatedAt = new Date().toISOString()
      queryClient.setQueryData(LIBRARY_LIST_QUERY_KEY, (current: LibraryDTO[] | undefined) =>
        (current ?? []).map((item) =>
          item.id === request.libraryId ? { ...item, name: nextName || item.name, updatedAt: nextUpdatedAt } : item,
        ),
      )
      queryClient.setQueryData([...LIBRARY_DETAIL_QUERY_KEY, request.libraryId], (current: LibraryDTO | undefined) =>
        current
          ? {
              ...current,
              name: nextName || current.name,
              updatedAt: nextUpdatedAt,
            }
          : current,
      )
      return { previousLibraries, previousLibrary }
    },
    onError: (_error, request, context) => {
      if (context?.previousLibraries) {
        queryClient.setQueryData(LIBRARY_LIST_QUERY_KEY, context.previousLibraries)
      }
      if (context?.previousLibrary) {
        queryClient.setQueryData([...LIBRARY_DETAIL_QUERY_KEY, request.libraryId], context.previousLibrary)
      }
    },
    onSuccess: (library) => {
      queryClient.setQueryData(LIBRARY_LIST_QUERY_KEY, (current: LibraryDTO[] | undefined) =>
        (current ?? []).map((item) => (item.id === library.id ? { ...item, name: library.name } : item)),
      )
      queryClient.setQueryData([...LIBRARY_DETAIL_QUERY_KEY, library.id], library)
      invalidateLibraryQueries(queryClient, library.id)
    },
    onSettled: (_library, _error, request) => {
      queryClient.invalidateQueries({ queryKey: LIBRARY_LIST_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: [...LIBRARY_DETAIL_QUERY_KEY, request.libraryId] })
    },
  })
}

export function useGetLibraryModuleConfig() {
  return useQuery({
    queryKey: LIBRARY_MODULE_CONFIG_QUERY_KEY,
    queryFn: async (): Promise<LibraryModuleConfigDTO> => {
      return parseGeneratedPayload(await LibraryHandler.GetModuleConfig(), parseLibraryModuleConfigPayload)
    },
    staleTime: 5_000,
  })
}

export function useGetDefaultLibraryModuleConfig() {
  return useQuery({
    queryKey: LIBRARY_DEFAULT_MODULE_CONFIG_QUERY_KEY,
    queryFn: async (): Promise<LibraryModuleConfigDTO> => {
      return parseGeneratedPayload(await LibraryHandler.GetDefaultModuleConfig(), parseLibraryModuleConfigPayload)
    },
    staleTime: Infinity,
  })
}

export function useUpdateLibraryModuleConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: UpdateLibraryModuleConfigRequest): Promise<LibraryModuleConfigDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.UpdateModuleConfig(
          LibraryBindings.UpdateLibraryModuleConfigRequest.createFrom(request),
        ),
        parseLibraryModuleConfigPayload,
      )
    },
    onSuccess: (config) => {
      queryClient.setQueryData(LIBRARY_MODULE_CONFIG_QUERY_KEY, config)
    },
  })
}

export function useListOperations(request: ListOperationsRequest) {
  return useQuery({
    queryKey: [...LIBRARY_OPERATIONS_QUERY_KEY, request],
    queryFn: async (): Promise<OperationListItemDTO[]> => {
      return parseGeneratedPayload(
        (await LibraryHandler.ListOperations(
          LibraryBindings.ListOperationsRequest.createFrom(request),
        )) ?? [],
        parseOperationListPayload,
      )
    },
    staleTime: 3_000,
  })
}

export function useGetOperation(operationId: string) {
  return useQuery({
    queryKey: [...LIBRARY_OPERATIONS_QUERY_KEY, "detail", operationId],
    enabled: operationId.trim().length > 0,
    queryFn: async (): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetOperation(
          LibraryBindings.GetOperationRequest.createFrom({ operationId } satisfies GetOperationRequest),
        ),
        parseLibraryOperationPayload,
      )
    },
    staleTime: 3_000,
  })
}

export function useCancelOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: CancelOperationRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CancelOperation(LibraryBindings.CancelOperationRequest.createFrom(request)),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useResumeOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: ResumeOperationRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.ResumeOperation(LibraryBindings.ResumeOperationRequest.createFrom(request)),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useDeleteOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteOperationRequest): Promise<void> => {
      await LibraryHandler.DeleteOperation(LibraryBindings.DeleteOperationRequest.createFrom(request))
    },
    onSuccess: () => {
      invalidateLibraryQueries(queryClient)
    },
  })
}

export function useDeleteOperations() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteOperationsRequest): Promise<void> => {
      await LibraryHandler.DeleteOperations(LibraryBindings.DeleteOperationsRequest.createFrom(request))
    },
    onSuccess: () => {
      invalidateLibraryQueries(queryClient)
    },
  })
}

export function useDeleteLibrary() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteLibraryRequest): Promise<void> => {
      await LibraryHandler.DeleteLibrary(LibraryBindings.DeleteLibraryRequest.createFrom(request))
    },
    onSuccess: (_value, request) => {
      queryClient.setQueryData(LIBRARY_LIST_QUERY_KEY, (current: LibraryDTO[] | undefined) =>
        (current ?? []).filter((item) => item.id !== request.libraryId),
      )
      queryClient.removeQueries({ queryKey: [...LIBRARY_DETAIL_QUERY_KEY, request.libraryId] })
      queryClient.removeQueries({ queryKey: [...LIBRARY_WORKSPACE_QUERY_KEY, request.libraryId] })
      useLibraryRealtimeStore.getState().removeLibrary(request.libraryId)
      invalidateLibraryQueries(queryClient)
    },
  })
}

export function useDeleteFile() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteFileRequest): Promise<void> => {
      await LibraryHandler.DeleteFile(LibraryBindings.DeleteFileRequest.createFrom(request))
    },
    onSuccess: () => {
      invalidateLibraryQueries(queryClient)
    },
  })
}

export function useDeleteFiles() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteFilesRequest): Promise<void> => {
      await LibraryHandler.DeleteFiles(LibraryBindings.DeleteFilesRequest.createFrom(request))
    },
    onSuccess: () => {
      invalidateLibraryQueries(queryClient)
    },
  })
}

export function useListLibraryHistory(request: ListLibraryHistoryRequest) {
  return useQuery({
    queryKey: [...LIBRARY_HISTORY_QUERY_KEY, request],
    enabled: request.libraryId.trim().length > 0,
    queryFn: async (): Promise<LibraryHistoryRecordDTO[]> => {
      return parseGeneratedPayload(
        (await LibraryHandler.ListLibraryHistory(
          LibraryBindings.ListLibraryHistoryRequest.createFrom(request),
        )) ?? [],
        parseLibraryHistoryPayload,
      )
    },
    staleTime: 3_000,
  })
}

export function useListFileEvents(request: ListFileEventsRequest) {
  return useQuery({
    queryKey: [...LIBRARY_FILE_EVENTS_QUERY_KEY, request],
    enabled: request.libraryId.trim().length > 0,
    queryFn: async (): Promise<FileEventRecordDTO[]> => {
      return parseGeneratedPayload(
        (await LibraryHandler.ListFileEvents(
          LibraryBindings.ListFileEventsRequest.createFrom(request),
        )) ?? [],
        parseFileEventPayload,
      )
    },
    staleTime: 3_000,
  })
}

export function useSaveWorkspaceState() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: SaveWorkspaceStateRequest): Promise<WorkspaceStateRecordDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.SaveWorkspaceState(LibraryBindings.SaveWorkspaceStateRequest.createFrom(request)),
        parseWorkspaceStatePayload,
      )
    },
    onSuccess: (workspace) => {
      invalidateLibraryQueries(queryClient, workspace.libraryId)
    },
  })
}

export function useGetWorkspaceState(libraryId: string, enabled = true) {
  return useQuery({
    queryKey: [...LIBRARY_WORKSPACE_QUERY_KEY, libraryId],
    enabled: enabled && libraryId.trim().length > 0,
    queryFn: async (): Promise<WorkspaceStateRecordDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetWorkspaceState(
          LibraryBindings.GetWorkspaceStateRequest.createFrom({ libraryId } satisfies GetWorkspaceStateRequest),
        ),
        parseWorkspaceStatePayload,
      )
    },
    staleTime: 3_000,
    retry: false,
  })
}

export function useGetWorkspaceProject(libraryId: string, enabled = true) {
  return useQuery({
    queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, libraryId],
    enabled: enabled && libraryId.trim().length > 0,
    queryFn: async (): Promise<WorkspaceProjectDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetWorkspaceProject(
          LibraryBindings.GetWorkspaceProjectRequest.createFrom({ libraryId } satisfies GetWorkspaceProjectRequest),
        ),
        parseWorkspaceProjectPayload,
      )
    },
    staleTime: 3_000,
  })
}

export function useGenerateWorkspacePreviewVTT() {
  return useMutation({
    mutationFn: async (
      request: GenerateWorkspacePreviewVTTRequest,
    ): Promise<GenerateWorkspacePreviewVTTResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.GenerateWorkspacePreviewVTT(
          LibraryBindings.GenerateWorkspacePreviewVTTRequest.createFrom(request),
        ),
        parseGenerateWorkspacePreviewPayload,
      )
    },
  })
}

export function useGenerateSubtitleStylePreviewASS() {
  return useMutation({
    mutationFn: async (
      request: GenerateSubtitleStylePreviewASSRequest,
    ): Promise<GenerateSubtitleStylePreviewASSResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.GenerateSubtitleStylePreviewASS(
          LibraryBindings.GenerateSubtitleStylePreviewASSRequest.createFrom(request),
        ),
        parseGenerateSubtitleStylePreviewPayload,
      )
    },
  })
}

export function useGetSubtitleReviewSession(sessionId: string, enabled = true) {
  return useQuery({
    queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, "review-session", sessionId],
    enabled: enabled && sessionId.trim().length > 0,
    queryFn: async (): Promise<SubtitleReviewSessionDetailDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetSubtitleReviewSession(
          LibraryBindings.GetSubtitleReviewSessionRequest.createFrom(
            { sessionId } satisfies GetSubtitleReviewSessionRequest,
          ),
        ),
        parseSubtitleReviewSessionPayload,
      )
    },
    staleTime: 1_000,
  })
}

export function useApplySubtitleReviewSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: ApplySubtitleReviewSessionRequest): Promise<ApplySubtitleReviewSessionResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.ApplySubtitleReviewSession(
          LibraryBindings.ApplySubtitleReviewSessionRequest.createFrom(request),
        ),
        parseApplySubtitleReviewSessionPayload,
      )
    },
    onSuccess: (_result, _request) => {
      invalidateLibraryQueries(queryClient)
      queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, "review-session"] })
    },
  })
}

export function useDiscardSubtitleReviewSession() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DiscardSubtitleReviewSessionRequest): Promise<DiscardSubtitleReviewSessionResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.DiscardSubtitleReviewSession(
          LibraryBindings.DiscardSubtitleReviewSessionRequest.createFrom(request),
        ),
        parseDiscardSubtitleReviewSessionPayload,
      )
    },
    onSuccess: () => {
      invalidateLibraryQueries(queryClient)
      queryClient.invalidateQueries({ queryKey: [...LIBRARY_WORKSPACE_PROJECT_QUERY_KEY, "review-session"] })
    },
  })
}

export function useOpenFileLocation() {
  return useMutation({
    mutationFn: async (request: OpenFileLocationRequest): Promise<void> => {
      await LibraryHandler.OpenFileLocation(LibraryBindings.OpenFileLocationRequest.createFrom(request))
    },
  })
}

export function useOpenLibraryPath() {
  return useMutation({
    mutationFn: async (request: OpenPathRequest): Promise<void> => {
      await LibraryHandler.OpenPath(LibraryBindings.OpenPathRequest.createFrom(request))
    },
  })
}

export function usePrepareYtdlpDownload() {
  return useMutation({
    mutationFn: async (request: PrepareYtdlpDownloadRequest): Promise<PrepareYtdlpDownloadResponse> => {
      return parseGeneratedPayload(
        await LibraryHandler.PrepareYTDLPDownload(
          LibraryBindings.PrepareYTDLPDownloadRequest.createFrom(request),
        ),
        parsePrepareYtdlpDownloadPayload,
      )
    },
  })
}

export function useParseYtdlpDownload() {
  return useMutation({
    mutationFn: async (request: ParseYtdlpDownloadRequest): Promise<ParseYtdlpDownloadResponse> => {
      return parseGeneratedPayload(
        await LibraryHandler.ParseYTDLPDownload(
          LibraryBindings.ParseYTDLPDownloadRequest.createFrom(request),
        ),
        parseParseYtdlpDownloadPayload,
      )
    },
  })
}

export function useResolveDomainIcon() {
  return useMutation({
    mutationFn: async (request: ResolveDomainIconRequest): Promise<ResolveDomainIconResponse> => {
      return parseGeneratedPayload(
        await LibraryHandler.ResolveDomainIcon(
          LibraryBindings.ResolveDomainIconRequest.createFrom(request),
        ),
        parseResolveDomainIconPayload,
      )
    },
  })
}

export function useCreateYtdlpJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: CreateYtdlpJobRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateYTDLPJob(LibraryBindings.CreateYTDLPJobRequest.createFrom(request)),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useCheckYtdlpOperationFailure() {
  return useMutation({
    mutationFn: async (request: CheckYtdlpOperationFailureRequest): Promise<CheckYtdlpOperationFailureResponse> => {
      return parseGeneratedPayload(
        await LibraryHandler.CheckYTDLPOperationFailure(
          LibraryBindings.CheckYTDLPOperationFailureRequest.createFrom(request),
        ),
        parseCheckYtdlpOperationFailurePayload,
      )
    },
  })
}

export function useRetryYtdlpOperation() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: RetryYtdlpOperationRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.RetryYTDLPOperation(
          LibraryBindings.RetryYTDLPOperationRequest.createFrom(request),
        ),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useGetYtdlpOperationLog() {
  return useMutation({
    mutationFn: async (request: GetYtdlpOperationLogRequest): Promise<GetYtdlpOperationLogResponse> => {
      return parseGeneratedPayload(
        await LibraryHandler.GetYTDLPOperationLog(
          LibraryBindings.GetYTDLPOperationLogRequest.createFrom(request),
        ),
        parseGetYtdlpOperationLogPayload,
      )
    },
  })
}

export function useCreateSubtitleImport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: CreateSubtitleImportRequest): Promise<LibraryFileDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateSubtitleImport(
          LibraryBindings.CreateSubtitleImportRequest.createFrom(request),
        ),
        parseLibraryFilePayload,
      )
    },
    onSuccess: (file) => invalidateLibraryQueries(queryClient, file.libraryId),
  })
}

export function useCreateVideoImport() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: CreateVideoImportRequest): Promise<LibraryFileDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateVideoImport(LibraryBindings.CreateVideoImportRequest.createFrom(request)),
        parseLibraryFilePayload,
      )
    },
    onSuccess: (file) => invalidateLibraryQueries(queryClient, file.libraryId),
  })
}

export function useCreateTranscodeJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: CreateTranscodeJobRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateTranscodeJob(
          LibraryBindings.CreateTranscodeJobRequest.createFrom(request),
        ),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useCreateSubtitleTranslateJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: SubtitleTranslateRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateSubtitleTranslateJob(
          LibraryBindings.SubtitleTranslateRequest.createFrom(request),
        ),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useCreateSubtitleProofreadJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: SubtitleProofreadRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateSubtitleProofreadJob(
          LibraryBindings.SubtitleProofreadRequest.createFrom(request),
        ),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useTranscodePresets() {
  return useQuery({
    queryKey: LIBRARY_TRANSCODE_PRESETS_QUERY_KEY,
    queryFn: async (): Promise<TranscodePreset[]> => {
      return parseGeneratedPayload((await LibraryHandler.ListTranscodePresets()) ?? [], parseTranscodePresetListPayload)
    },
    staleTime: 30_000,
  })
}

export function useTranscodePresetsForDownload(request: ListTranscodePresetsForDownloadRequest | null) {
  return useQuery({
    queryKey: [...LIBRARY_TRANSCODE_PRESETS_FOR_DOWNLOAD_QUERY_KEY, request],
    enabled: request !== null && request.mediaType.trim().length > 0,
    queryFn: async (): Promise<TranscodePreset[]> => {
      if (!request) {
        return []
      }
      return parseGeneratedPayload(
        (await LibraryHandler.ListTranscodePresetsForDownload(
          LibraryBindings.ListTranscodePresetsForDownloadRequest.createFrom(request),
        )) ?? [],
        parseTranscodePresetListPayload,
      )
    },
    staleTime: 30_000,
  })
}

export function useSaveTranscodePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (preset: TranscodePreset): Promise<TranscodePreset> => {
      return parseGeneratedPayload(
        await LibraryHandler.SaveTranscodePreset(LibraryBindings.TranscodePreset.createFrom(preset)),
        parseTranscodePresetPayload,
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LIBRARY_TRANSCODE_PRESETS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: LIBRARY_TRANSCODE_PRESETS_FOR_DOWNLOAD_QUERY_KEY })
    },
  })
}

export function useDeleteTranscodePreset() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: DeleteTranscodePresetRequest): Promise<void> => {
      await LibraryHandler.DeleteTranscodePreset(
        LibraryBindings.DeleteTranscodePresetRequest.createFrom(request),
      )
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: LIBRARY_TRANSCODE_PRESETS_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: LIBRARY_TRANSCODE_PRESETS_FOR_DOWNLOAD_QUERY_KEY })
    },
  })
}

export function useParseSubtitle() {
  return useMutation({
    mutationFn: async (request: SubtitleParseRequest): Promise<SubtitleParseResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.ParseSubtitle(LibraryBindings.SubtitleParseRequest.createFrom(request)),
        parseSubtitleParsePayload,
      )
    },
  })
}

export function useConvertSubtitle() {
  return useMutation({
    mutationFn: async (request: SubtitleConvertRequest): Promise<SubtitleConvertResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.ConvertSubtitle(LibraryBindings.SubtitleConvertRequest.createFrom(request)),
        parseSubtitleConvertPayload,
      )
    },
  })
}

export function useExportSubtitle() {
  return useMutation({
    mutationFn: async (request: SubtitleExportRequest): Promise<SubtitleExportResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.ExportSubtitle(LibraryBindings.SubtitleExportRequest.createFrom(request)),
        parseSubtitleExportPayload,
      )
    },
  })
}

export function useValidateSubtitle() {
  return useMutation({
    mutationFn: async (request: SubtitleValidateRequest): Promise<SubtitleValidateResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.ValidateSubtitle(LibraryBindings.SubtitleValidateRequest.createFrom(request)),
        parseSubtitleValidatePayload,
      )
    },
  })
}

export function useCreateSubtitleQAReviewJob() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (request: SubtitleQAReviewRequest): Promise<LibraryOperationDTO> => {
      return parseGeneratedPayload(
        await LibraryHandler.CreateSubtitleQAReviewJob(
          LibraryBindings.SubtitleQAReviewRequest.createFrom(request),
        ),
        parseLibraryOperationPayload,
      )
    },
    onSuccess: (operation) => invalidateLibraryQueries(queryClient, operation.libraryId),
  })
}

export function useFixSubtitleTypos() {
  return useMutation({
    mutationFn: async (request: SubtitleFixTyposRequest): Promise<SubtitleFixTyposResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.FixSubtitleTypos(LibraryBindings.SubtitleFixTyposRequest.createFrom(request)),
        parseSubtitleFixTyposPayload,
      )
    },
  })
}

export function useSaveSubtitle() {
  return useMutation({
    mutationFn: async (request: SubtitleSaveRequest): Promise<SubtitleSaveResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.SaveSubtitle(LibraryBindings.SubtitleSaveRequest.createFrom(request)),
        parseSubtitleSavePayload,
      )
    },
  })
}

export function useRestoreSubtitleOriginal() {
  return useMutation({
    mutationFn: async (request: RestoreSubtitleOriginalRequest): Promise<RestoreSubtitleOriginalResult> => {
      return parseGeneratedPayload(
        await LibraryHandler.RestoreSubtitleOriginal(
          LibraryBindings.RestoreSubtitleOriginalRequest.createFrom(request),
        ),
        parseRestoreSubtitleOriginalPayload,
      )
    },
  })
}
