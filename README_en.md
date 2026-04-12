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

Dream Creator is an AI assistant for content creators that brings together work usually scattered across media downloads, subtitle workflows, research, desktop operations, and mobile channels. It is not just a collection of tools, but a creative assistant that understands context and keeps work moving: from sourcing media, refining subtitles, and preparing exports to researching references, shaping ideas, and following through across devices, the workflows creators switch between every day can now continue inside one system.

## Built for Every Creative Context

- 🎬 Creative preparation in one place: source material, subtitles, research, and idea development come together in one workspace, so the most fragmented part of creation turns into reusable assets and clearer direction faster.
- 🖥️ Complete desktop execution: with user authorization, the assistant goes beyond understanding requests to accessing the web and the local computer, using tools, skills, and memory to unify conversation, operation, and execution in one workflow.
- 📱 Work continues beyond the desktop: strong multi-channel access keeps the same assistant available outside the desktop. Telegram is supported today (Feishu / WeChat in development), so work does not stop when you step away from the computer.

## Core Capabilities

- 📥 Video download: one-click downloads from more than a thousand video sites, with authenticated access supported. From a YouTube BGM reference to a 4K Bilibili clip, everything can become part of your content library.
- 📝 Subtitle proofreading and translation: existing subtitles do not need to be rebuilt from scratch. Proofreading, translation, and review connect in one flow, making multilingual publishing more reliable and content reuse more efficient.
- 🎞️ Video transcoding and subtitle burn-in: downloading, translation, transcoding, and subtitle embedding can be chained into one pipeline, turning fragmented release prep into a one-click delivery flow.
- 💡 Conversational research and ideation: reference gathering, information organization, and idea expansion all happen inside the conversation, so creative momentum is not broken between inspiration and structure.
- 🤖 Executable AI assistant: it goes beyond answering questions, with the ability to access webpages, operate the computer, and call tools within the granted scope to move ideas directly into results.
- 🧩 Multi-assistant switching: different assistants can be defined for different contexts, each with its own role, memory, and capability boundary, so research, processing, and publishing do not interfere with each other.
- ⚙️ AI-native automation: recurring downloading, organization, processing, and scheduled tasks can keep running on their own, leaving more attention for judgment, taste, and the creative work itself.
- 📲 Mobile channel access: the same capability set is not confined to the desktop. Telegram is supported today, with Feishu and WeChat in development, so work can continue from your phone.

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
