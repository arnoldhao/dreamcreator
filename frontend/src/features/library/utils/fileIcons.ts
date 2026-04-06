import {
  Captions,
  File,
  FileArchive,
  FileAudio,
  FileCode,
  FileImage,
  FileJson,
  FileText,
  FileVideo,
} from "lucide-react"

import { getBaseName } from "./format"

type FileCategory = "video" | "audio" | "subtitle" | "image" | "archive" | "code" | "json" | "text" | "other"

type FileIconInput = {
  fileType?: string
  path?: string
  name?: string
}

const CATEGORY_ICON: Record<FileCategory, typeof File> = {
  video: FileVideo,
  audio: FileAudio,
  subtitle: Captions,
  image: FileImage,
  archive: FileArchive,
  code: FileCode,
  json: FileJson,
  text: FileText,
  other: File,
}

const TYPE_CATEGORY: Record<string, FileCategory> = {
  video: "video",
  transcode: "video",
  subtitle: "subtitle",
  audio: "audio",
  thumbnail: "image",
  image: "image",
  archive: "archive",
}

const EXTENSION_CATEGORY: Record<string, FileCategory> = {
  mp4: "video",
  mkv: "video",
  mov: "video",
  webm: "video",
  avi: "video",
  mpg: "video",
  mpeg: "video",
  mp3: "audio",
  wav: "audio",
  flac: "audio",
  aac: "audio",
  m4a: "audio",
  ogg: "audio",
  srt: "subtitle",
  vtt: "subtitle",
  ass: "subtitle",
  ssa: "subtitle",
  sub: "subtitle",
  jpg: "image",
  jpeg: "image",
  png: "image",
  gif: "image",
  webp: "image",
  bmp: "image",
  tiff: "image",
  zip: "archive",
  rar: "archive",
  "7z": "archive",
  tar: "archive",
  gz: "archive",
  tgz: "archive",
  json: "json",
  yaml: "code",
  yml: "code",
  toml: "code",
  ini: "code",
  xml: "code",
  md: "text",
  txt: "text",
  csv: "text",
  tsv: "text",
  js: "code",
  jsx: "code",
  ts: "code",
  tsx: "code",
  go: "code",
  py: "code",
  rs: "code",
  java: "code",
  kt: "code",
  c: "code",
  cpp: "code",
  h: "code",
  hpp: "code",
}

export function resolveFileIcon({ fileType, path, name }: FileIconInput): typeof File {
  const category = resolveFileCategory({ fileType, path, name })
  return CATEGORY_ICON[category] ?? File
}

export function resolveFileCategory({ fileType, path, name }: FileIconInput): FileCategory {
  const normalizedType = fileType?.toLowerCase() ?? ""
  const typeCategory = TYPE_CATEGORY[normalizedType]
  if (typeCategory) {
    return typeCategory
  }
  const extension = getFileExtension(path, name)
  if (extension) {
    return EXTENSION_CATEGORY[extension] ?? "other"
  }
  return "other"
}

function getFileExtension(path?: string, name?: string): string {
  const base = getBaseName(path) || name || ""
  const index = base.lastIndexOf(".")
  if (index <= 0 || index >= base.length - 1) {
    return ""
  }
  return base.slice(index + 1).toLowerCase()
}
