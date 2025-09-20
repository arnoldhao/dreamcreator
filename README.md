<div align="center">
  <a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="140" alt="CanMe logo" /></a>
</div>

<h1 align="center">CanMe</h1>

<p align="center">
  <strong>English</strong> |
  <a href="/README_zh.md"><strong>简体中文</strong></a>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/canme?label=version" alt="Latest tag" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="Supported platforms" />
  <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20Vue3-green" alt="Tech stack" />
</div>

<br />

CanMe is an open-source desktop toolkit for downloading online media and running a reproducible subtitle workflow. The backend is implemented in Go (Wails) with BoltDB persistence and WebSocket messaging; the frontend is a Vue 3 / Tailwind interface. The project focuses on reliable downloads and transparent subtitle processing, with room to expand into auxiliary tooling as it matures.

<div align="center">
  <img src="images/ui_en.png" width="85%" alt="CanMe user interface" />
</div>

## Table of Contents
- [Highlights](#highlights)
- [Feature Overview](#feature-overview)
- [Getting Started](#getting-started)
- [Build from Source](#build-from-source)
- [Usage Notes](#usage-notes)
- [Roadmap](#roadmap)
- [License](#license)
- [Acknowledgements](#acknowledgements)

## Highlights
- Desktop binaries for Windows 10+ and macOS 10.15+, packaged with yt-dlp and FFmpeg
- Explicit dependency manager: version pinning, checksum verification, mirrors, and auto-heal
- BoltDB-backed metadata store for downloads, cookie snapshots, and subtitle projects
- Real-time status streaming over WebSocket to a Vue 3 + Pinia UI with bilingual i18n
- Apache License 2.0, active refactoring toward modular AI translation hooks and cookie tooling

## Feature Overview

### Acquisition Pipeline
- **Multi-source downloads:** yt-dlp wrapper with per-task format selection, staged progress events (probe → fetch → merge → finalize)
- **Proxy-aware execution:** per-app HTTP/SOCKS proxy including PAC support and Windows elevation helpers
- **Dependency watchdog:** background checks for yt-dlp/FFmpeg availability, mirror fallback, and hash validation

### Subtitle Capabilities
- **Import formats:** `.srt`, `.vtt/.webvtt`, `.ass/.ssa`, `.itt` via dedicated parsers (`backend/core/subtitles/format.go`)
- **Normalization:** configurable text processing (punctuation cleanup, trimming, zh Hans⇄Hant conversion through `pkg/zhconvert`)
- **Quality analysis:** heuristics evaluating timing gaps, segment length, and character density (`quality_assessor.go`)
- **Project store:** subtitle data is persisted as `types.SubtitleProject` with language indexes for instant lookups and diffing
- **Export formats:** SRT, VTT, ASS/SSA, ITT, Final Cut Pro XML; export configs auto-fill frame rate, resolutions, and track metadata
- **Translation staging:** backend task lifecycle reserves a `translate` phase (`service.go:675+`, `translateSubtitles`) ready for pluggable MT/LLM adapters; current builds emit placeholders without calling external APIs
- **Embedding hooks:** conversion routines feed FFmpeg mux steps to burn or attach tracks during post-processing

### Cookie Management
- Scoped browser scanners for Chrome, Chromium, Edge, Firefox, Safari, Brave, Opera, and Vivaldi on Windows/macOS; results stored by browser with status and sync history
- Netscape-format export for yt-dlp and manual inspection, optional per-domain filtering, and cleanup of temporary exports
- Manual cookie collections that accept Netscape text, JSON arrays exported from browser devtools, or raw `Cookie:` header strings—merge or replace data without running a browser sync
- WebSocket notifications for sync progress so the UI can surface granular status and errors

### Media Conversion
- FFmpeg runners with presets for remuxing, audio extraction, and subtitle attachment; dependency metadata cached for repeated runs
- Task classification (`video`, `subtitle`, `other`) to keep ancillary files grouped in the UI’s download inspector

## Getting Started

### Prerequisites
- Windows 10 or later, or macOS 10.15 or later
- Sufficient storage for target downloads; FFmpeg temp files may require additional space
- No manual dependency installation is required; CanMe downloads and validates yt-dlp/FFmpeg on first launch

### Install Packages
1. Download the latest release from [GitHub Releases](https://github.com/arnoldhao/canme/releases)
2. Extract the archive to a writable directory
3. Launch the application

#### macOS Gatekeeper
Unsigned builds require manual confirmation:
- Control-click the app → **Open**, or
- Remove the quarantine flag:
  ```bash
  sudo xattr -rd com.apple.quarantine /path/to/CanMe.app
  ```

#### Windows SmartScreen
- Choose **More info → Run anyway** the first time you launch the unsigned binary

### Dependency Management
- yt-dlp and FFmpeg live under the app’s managed cache; damaged binaries are redownloaded with SHA validation
- Chrome/Edge cookie sync may need elevated privileges on Windows; the UI surfaces when elevation is required

## Build from Source

> Use the release artifacts for day-to-day work. Build steps below target contributors.

### Requirements
- Go 1.24+
- Node.js 18+ (with npm or pnpm)
- Wails CLI (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

### Steps
```bash
# Backend modules
go mod tidy

# Frontend assets
cd frontend
npm install
npm run build
cd ..

# Desktop bundle
wails build
```

For hot reloading, run `wails dev` in the repository root and `npm run dev` inside `frontend` to attach Vite’s dev server.

## Usage Notes
- Sign in to streaming services in Chrome/Edge before running **Cookies → Sync** to capture fresh authentication, or build a manual collection by pasting Netscape/JSON/header data when the browser flow is not possible
- The scheduler parallelizes metadata fetches but serializes heavy merge/transcode steps to avoid I/O contention
- Subtitle imports appear in the Subtitle workspace; review segments, run quality checks, then trigger translation once adapters are configured
- Configure proxy settings in **Preferences → Network** if downloads return geo-errors or throttled responses

## Roadmap
Upcoming work is tracked via GitHub issues/projects. Current focus areas:
- Wire the translation stage to external AI services with sensible retry policies and audit logging
- Expand cookie tooling with manual import/export, conflict resolution, and sandboxed scraping helpers
- Provide FFmpeg preset catalogues (platform/social-specific) and batch conversion scripts

## License

CanMe is distributed under the Apache License 2.0. See `LICENSE` for the full text.

## Acknowledgements
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)
- [Wails](https://wails.io/)
- [Vue](https://vuejs.org/)
- [TailwindCSS](https://tailwindcss.com/)
