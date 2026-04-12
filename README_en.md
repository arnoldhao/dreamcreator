<div align="center">
  <img src="./frontend/public/appicon.png" width="112" alt="Dream Creator icon" />
  <h1>Dream Creator</h1>
  <p><strong>An AI assistant for content creators.</strong></p>
  <p>
    <a href="./README.md">简体中文</a> ·
    <strong>English</strong>
  </p>
  <p>
    <img src="https://img.shields.io/github/v/tag/arnoldhao/dreamcreator?label=version" alt="Latest version" />
    <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="License" />
    <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="Supported platforms" />
    <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20React-green" alt="Tech stack" />
  </p>
</div>

## Overview

Dream Creator is an AI assistant for content creators that brings media download, subtitle work, research, desktop execution, and mobile follow-up into one workflow. It is not a loose bundle of tools, but a context-aware creative assistant that keeps work moving from one step to the next.

## Built for Every Creative Context

From preparation and desktop execution to follow-up away from the desk, work no longer has to break across different tools and devices.

- 🎬 Creative preparation: source material, subtitles, references, and idea fragments can be gathered in one place before moving into the next step.
- 🖥️ Desktop execution: within the granted scope, the assistant can access webpages, call tools, and operate the computer so conversation can turn directly into action.
- 📱 Mobile follow-up: the same assistant can stay available through channels such as Telegram, so tasks keep moving even away from the desktop.

## Core Capabilities

- 📥 Video download: from public footage to authenticated sources, everything can be collected into the same library.
- 📝 Subtitle proofreading and translation: existing subtitles do not need to be rebuilt; proofreading, translation, and review can run as one chain.
- 🎞️ Video transcoding and subtitle burn-in: downloading, translation, transcoding, and subtitle embedding can be completed as a continuous flow.
- 💡 Conversational research and ideation: reference lookup, information organization, and idea expansion can stay inside the same conversation.
- 🤖 Executable AI assistant: it does more than answer questions; within the granted scope it can access webpages, call tools, and operate the computer.
- 🧩 Multi-assistant switching: different creative contexts can use different assistants, each with its own role, memory, and capability boundary.
- ⚙️ AI-native automation: recurring downloading, organization, processing, and scheduled tasks can keep running over time.
- 📲 Mobile channel access: the same capability set is not tied to the desktop, so work can keep moving from your phone.

## Product Preview

![Dream Creator English UI preview](./images/ui_en.png)

## Quick Start

Download the app, complete a basic setup, and your creative workflow is ready to begin.

### Download and install

Download the latest build directly below. Older releases are available on [GitHub Releases](https://github.com/arnoldhao/dreamcreator/releases).

| Platform | Architecture | Package | Download |
| --- | --- | --- | --- |
| macOS | Apple Silicon | Archive | [Download](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-macos-arm64-latest.zip) |
| macOS | Intel | Archive | [Download](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-macos-x64-latest.zip) |
| Windows | x64 | Installer | [Download](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-windows-x64-latest-installer.exe) |
| Windows | x64 | Portable | [Download](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-windows-x64-latest.zip) |

### First launch

1. `macOS`: unzip the package and move `DreamCreator.app` to the Applications folder. If macOS says the app cannot be opened or is damaged, run `sudo xattr -rd com.apple.quarantine /Applications/DreamCreator.app`.
2. `Windows`: run the `.exe` installer directly, or unzip the portable package and launch it. If SmartScreen appears on first launch, choose `More info -> Run anyway`.

### Basic setup

- The app runs locally, so you need to configure an available model provider API key before first use.
- Video, subtitle, and automation workflows depend on external tools such as `yt-dlp`, `FFmpeg`, `bun`, and `playwright`; the app will guide you through installation when first opened.

### First experience

1. Configure your model provider and finish installing the required external tools.
2. Paste a video link, or start a subtitle proofreading, translation, or transcoding task directly.
3. Continue the follow-up work in the library or in chat to complete one full workflow.

## Acknowledgements

Dream Creator is built on top of a number of excellent open-source projects and supporting ecosystems. The desktop experience, assistant execution, media pipeline, local storage, browser automation, and channel integrations all depend on these foundations.

| Category | Homepage |
| --- | --- |
| Desktop Framework | <a href="https://go.dev/" target="_blank" rel="noreferrer">Go</a> / <a href="https://v3alpha.wails.io/" target="_blank" rel="noreferrer">Wails 3</a> / <a href="https://react.dev/" target="_blank" rel="noreferrer">React</a> |
| Local Storage | <a href="https://www.sqlite.org/" target="_blank" rel="noreferrer">SQLite</a> / <a href="https://bun.uptrace.dev/" target="_blank" rel="noreferrer">bun</a> / <a href="https://github.com/asg017/sqlite-vec" target="_blank" rel="noreferrer">sqlite-vec</a> |
| Media Processing | <a href="https://github.com/yt-dlp/yt-dlp" target="_blank" rel="noreferrer">yt-dlp</a> / <a href="https://ffmpeg.org/" target="_blank" rel="noreferrer">FFmpeg</a> |
| Browser Automation | <a href="https://playwright.dev/" target="_blank" rel="noreferrer">Playwright</a> |
| Channel Access | <a href="https://telegram.org/" target="_blank" rel="noreferrer">Telegram</a> / <a href="https://github.com/mymmrac/telego" target="_blank" rel="noreferrer">telego</a> |

These projects, and the maintainers behind them, make it possible for Dream Creator to connect desktop workflows, media processing, automation, and channel access into one evolving system.

## Collaboration

- The project is under active development, and the interface, workflow design, and channel capabilities will continue to evolve around real-world usage.
- The project is actively maintained by its author.
- Pull requests are not being accepted for now. Iteration currently moves forward through [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues) and email, including bug reports, feedback, and real usage scenarios.
- This repository is licensed under `Apache-2.0`. See [LICENSE](./LICENSE).

## Contact

- Website: <https://dreamcreator.dreamapp.cc>
- Email: <xunruhao@gmail.com>
