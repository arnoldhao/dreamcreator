import { create } from "zustand"
import { persist } from "zustand/middleware"

export type TranscodePresetOutputType = "video" | "audio"
export type TranscodeQualityMode = "crf" | "bitrate"
export type TranscodeScaleMode = "original" | "2160p" | "1080p" | "720p" | "480p" | "custom"
export type FFmpegSpeedPreset = "ultrafast" | "fast" | "medium" | "slow"

export type TranscodePreset = {
  id: string
  name: string
  outputType: TranscodePresetOutputType
  container: string
  videoCodec?: string
  audioCodec?: string
  qualityMode?: TranscodeQualityMode
  crf?: number
  bitrateKbps?: number
  audioBitrateKbps?: number
  scale?: TranscodeScaleMode
  width?: number
  height?: number
  ffmpegPreset?: FFmpegSpeedPreset
  allowUpscale?: boolean
  requiresVideo?: boolean
  requiresAudio?: boolean
  isBuiltin?: boolean
  description?: string
}

type TranscodePresetState = {
  customPresets: TranscodePreset[]
  addPreset: (preset: TranscodePreset) => void
  updatePreset: (preset: TranscodePreset) => void
  removePreset: (id: string) => void
  setCustomPresets: (presets: TranscodePreset[]) => void
}

export const useTranscodePresetStore = create<TranscodePresetState>()(
  persist(
    (set) => ({
      customPresets: [],
      addPreset: (preset) =>
        set((state) => ({
          customPresets: [...state.customPresets, preset],
        })),
      updatePreset: (preset) =>
        set((state) => ({
          customPresets: state.customPresets.map((item) => (item.id === preset.id ? preset : item)),
        })),
      removePreset: (id) =>
        set((state) => ({
          customPresets: state.customPresets.filter((item) => item.id !== id),
        })),
      setCustomPresets: (presets) => set({ customPresets: presets }),
    }),
    {
      name: "transcode-presets",
    }
  )
)
