export type LibraryTab = "all" | "video" | "subtitle" | "thumbnail"
export type LibraryViewMode = "task" | "file"

export type LibraryProgress = {
  label: string
  percent?: number
  detail?: string
  speed?: string
  updatedAt?: string
}

export type LibraryTaskOutput = {
  id: string
  label: string
  path: string
  fileType?: string
  format?: string
  sourceType?: string
  sourceLabel?: string
  isOriginal?: boolean
  isPrimary?: boolean
  language?: string
  cueCount?: number
  deleted?: boolean
}

export type LibraryWorkspaceTarget = {
  id?: string
  recordId?: string
  libraryId?: string
  openMode?: "video" | "subtitle"
  fileId: string
  name: string
  fileType: string
  path?: string
  taskId?: string
  videoAssetId?: string
  subtitleAssetId?: string
  videoPath?: string
  subtitlePath?: string
  videoName?: string
  subtitleName?: string
}

export type LibraryTaskRow = {
  id: string
  libraryId?: string
  libraryName?: string
  name: string
  taskType: string
  taskTypeLabel?: string
  typeLabel: string
  platform?: string
  uploader?: string
  status: string
  progress?: LibraryProgress | null
  outputs?: {
    count: number
    sizeBytes?: number | null
    totalCount?: number
    deletedCount?: number
    totalSizeBytes?: number | null
    deletedSizeBytes?: number | null
  } | null
  outputTypes?: string[] | null
  duration?: string
  publishedAt?: string
  startedAt?: string
  createdAt?: string
  outputFiles?: LibraryTaskOutput[]
  libraryFiles?: LibraryTaskOutput[]
  sourceDomain?: string
  sourceIcon?: string
}

export type LibraryFileRow = {
  id: string
  libraryId?: string
  name: string
  displayLabel?: string
  fileType: string
  format?: string
  sourceType?: string
  isOriginal?: boolean
  language?: string
  cueCount?: number
  typeLabel: string
  status: string
  progress?: LibraryProgress | null
  sizeBytes?: number | null
  taskId?: string | null
  taskName?: string | null
  createdAt?: string
  path?: string
  requestUrl?: string
  thumbnailUrl?: string
  lastCheckTime?: string
}
