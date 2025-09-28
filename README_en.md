<div align="center">
  <a href="https://github.com/arnoldhao/dreamcreator/"><img src="frontend/src/assets/images/icon.png" width="140" alt="dreamcreator icon" /></a>
</div>

<h1 align="center">dreamcreator</h1>

<p align="center">
  <a href="./README.md"><strong>简体中文</strong></a> |
  <strong>English</strong>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/dreamcreator?label=version" alt="Latest tag" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="Supported platforms" />
  <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20Vue3-green" alt="Tech stack" />
</div>

> dreamcreator is an open-source desktop workstation for video creators, built to cover the full workflow from **acquiring footage** through **subtitle refinement** to **global release prep**. Powered by Wails + Vue, it bundles yt-dlp, FFmpeg, Fanhuaji localisation, and extensible AI adapters.

---

## Table of Contents
- [Vision & Positioning](#vision--positioning)
- [Capability Map](#capability-map)
  - [Download](#download)
  - [Subtitle](#subtitle)
  - [Transcode (planned)](#transcode-planned)
  - [Self-Healing & Observability](#self-healing--observability)
- [Quick Start](#quick-start)
- [Workflow Highlights](#workflow-highlights)
  - [Download Board](#download-board)
  - [Subtitle Workbench](#subtitle-workbench)
- [Configuration & Dependencies](#configuration--dependencies)
- [Roadmap](#roadmap)
- [Docs & Support](#docs--support)
- [Contributing](#contributing)
- [License](#license)

---

## Vision & Positioning
We build dreamcreator as a **creator-first** toolkit—not just another downloader.
- **Focus on creators**: a low-distraction UI where every feature serves the “capture → polish → deliver” loop.
- **Reliable infrastructure**: proxy management, cookies, dependency checks, logging, and window state are handled by services inside the app.
- **Open roadmap**: the entire stack is open source on GitHub; we welcome contributions that strengthen creator workflows.

Read more on the product site: [dreamapp.cc/products/dreamcreator](https://dreamapp.cc/products/dreamcreator).

## Capability Map

### Download
- **Cookie management**: sync from local browsers (Chrome, Edge, Firefox, Brave, Vivaldi, …) or import Netscape / JSON / header strings. dreamcreator auto-detects usable cookies before each download.
- **Proxy & network strategy**: global HTTP/SOCKS/PAC settings help bypass geo restrictions and throttling.
- **Format selection**: leverages yt-dlp to surface video/audio/subtitle tracks. Choose manually with “Custom Download” or run a one-click “Quick Download”.
- **Task observability**: monitor probe → fetch → merge → finalize stages with live speed, remaining time, and generated files.

Detailed guide: [Download docs](https://dreamapp.cc/docs/dreamcreator/download).

### Subtitle
- **Flexible import**: launch from a download task or load local SRT, VTT, ASS/SSA, ITT, or FCPXML files.
- **Guideline presets**: Netflix / BBC / ADE presets with traffic-light indicators for duration, CPS, WPM, and CPL on every cue.
- **Chinese localisation**: Fanhuaji conversions deliver region-specific wording (Mainland, Hong Kong, Taiwan). AI translation adapters are in progress.
- **Multilingual editing**: quickly switch between language tracks; export keeps frame rate & resolution metadata in sync.

Detailed guide: [Subtitle docs](https://dreamapp.cc/docs/dreamcreator/subtitles).

### Transcode (planned)
- Current builds rely on yt-dlp’s FFmpeg call for basic mux/remux. We are designing a native pipeline with progress visualisation, GPU support, and publishing presets.
- Upcoming enhancements include audio-to-subtitle, audio translation, lip-sync, and richer release workflows.

Roadmap notes: [Transcode overview](https://dreamapp.cc/docs/dreamcreator/transcode).

### Self-Healing & Observability
- **Dependency manager**: background checks for yt-dlp / FFmpeg. Run quick checks, deep validation, or mirror-based updates; SHA verification guards every switch.
- **Logging**: configurable level/rotation with logs stored under `~/.dreamcreator/logs` by default.
- **Service status**: settings pages display WebSocket and MCP ports for integration debugging.
- **Persistence**: BoltDB stores tasks, subtitles, and cookies with user-selectable directories.

Configuration details: [Settings docs](https://dreamapp.cc/docs/dreamcreator/settings).

## Quick Start
1. **Download a release** from [GitHub Releases](https://github.com/arnoldhao/dreamcreator/releases). Use the `.dmg` matching your Mac architecture or the Windows `.exe` installer / `.zip` portable build.
2. **Approve first launch**:
   - macOS: Control-click → “Open” or run
     ```bash
     sudo xattr -rd com.apple.quarantine /Applications/DreamCreator.app
     ```
   - Windows: choose “More info → Run anyway”.
3. **Initial boot**: dreamcreator unpacks yt-dlp and FFmpeg automatically. Configure proxies under **Preferences → Network** if required.
4. **Prepare cookies**: sign in via your browser, then open **Downloads → Browser Cookies** to sync—or paste exported Netscape/JSON/header data into a custom collection.

More installation notes: [Setup guide](https://dreamapp.cc/docs/dreamcreator/setup).

## Workflow Highlights

### Download Board
1. Click “New task” and paste the URL.
2. dreamcreator inspects available cookies; switch collections anytime.
3. Choose between Custom (select tracks manually) or Quick (auto-pick best quality).
4. Watch progress across video, merge, and clean-up stages, and open files directly from the card.

> Because downloads rely on yt-dlp today, pause/resume and GPU-based transcode are not yet available. A dedicated pipeline is on the roadmap.

### Subtitle Workbench
1. Open a subtitle from a task or import a file.
2. Switch guideline presets and use the traffic lights to correct duration, CPS, WPM, and CPL.
3. Add languages via Fanhuaji conversions for region-specific Chinese output; AI translation is under construction.
4. Export SRT, VTT, ASS/SSA, ITT, or Final Cut XML with adjustable frame rate/resolution presets.

## Configuration & Dependencies
- **General settings**: theme, language, proxy, download directory, data directory.
- **Dependencies**: run quick check / verify / check updates, then use Repair or Update to recover binaries.
- **Logging**: configure verbosity and rotation to audit download/subtitle operations.
- **Listeners**: inspect WebSocket & MCP endpoints to connect external scripts or IDE tooling.

See [Settings docs](https://dreamapp.cc/docs/dreamcreator/settings) for full coverage.

## Roadmap
- AI audio-to-subtitle
- AI audio translation
- AI lip-sync
- Native transcode & release pipeline (GPU, batch, templated presets)

Track progress or open discussions via [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues).

## Docs & Support
- Product overview: [dreamapp.cc/products/dreamcreator](https://dreamapp.cc/products/dreamcreator)
- Chinese docs hub: [dreamapp.cc/zh-CN/docs/dreamcreator](https://dreamapp.cc/zh-CN/docs/dreamcreator)
- English docs hub: [dreamapp.cc/docs/dreamcreator](https://dreamapp.cc/docs/dreamcreator)
- Email: team@dreamapp.cc

## Contributing
1. Fork the repo and create a topic branch.
2. Run `npm run build` (frontend) and `go test ./...` (backend) before opening a PR.
3. Include context, test results, and UI captures (if applicable) with every PR.
4. Label suggestions: `feat`, `fix`, `chore`, `docs`—these feed the release drafter configuration.

## License
Licensed under the [Apache License 2.0](LICENSE). You’re welcome to adapt dreamcreator for your workflow—please keep attribution and share your improvements with the community.
