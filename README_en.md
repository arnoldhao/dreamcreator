<div align="center">
  <img src="./frontend/public/appicon.png" width="112" alt="DreamCreator icon" />
  <h1>DreamCreator</h1>
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

DreamCreator is an AI-native desktop app that combines video download, subtitle workflows, research, desktop execution, and mobile follow-up.

Whether you work in the interface, through conversation, or from mobile, DreamCreator keeps content preparation orderly and the product experience consistent.

## Built for Creative Workflows

- 🎬 **Creative preparation**: Organize source material, subtitles, references, and idea fragments in one place.
- 🖥️ **Desktop execution**: Access webpages, call tools, and keep tasks moving forward.
- 📱 **Mobile follow-up**: Keep checking results and follow-up progress through Telegram.

## Core Capabilities

- 📥 **Video download**: Download public videos and videos that require sign-in into the local library.
- 📝 **Subtitle proofreading and translation**: Continue from existing subtitles for proofreading, translation, QA, and export.
- 🎞️ **Video transcoding and subtitle burn-in**: Complete transcoding, subtitle export, embedded subtitle tracks, and burn-in.
- 💡 **Conversation, research, and idea organization**: Organize references and continue into the next action in one thread.
- 🤖 **AI assistant**: Access webpages, call tools, and operate the computer within the granted scope.
- 🧩 **Multi-assistant setup**: Split assistants by role, model, and permission boundary.
- ⚙️ **Scheduled tasks**: Hand recurring download, organization, inspection, and delivery work to the system.
- 📲 **Mobile channel access**: Keep checking results and following up through channels such as Telegram.

## Product Preview

![DreamCreator English UI preview](./images/ui_en.png)

## Quick Start

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
3. Full instructions for installation, initialization, usage, and updates are available in [Install, First Launch & Updates](https://dreamapp.cc/docs/dreamcreator/install-and-update/).

## Documentation

- [Install, First Launch & Updates](https://dreamapp.cc/docs/dreamcreator/install-and-update/)
- [DreamCreator Overview](https://dreamapp.cc/docs/dreamcreator/overview/)
- [Video Download](https://dreamapp.cc/docs/dreamcreator/video-download/)
- [Subtitle Proofreading & Translation](https://dreamapp.cc/docs/dreamcreator/subtitle-proofreading-and-translation/)
- [Video Transcoding & Subtitle Burn-In](https://dreamapp.cc/docs/dreamcreator/transcoding-and-subtitle-burn-in/)
- [Conversation, Research & Idea Organization](https://dreamapp.cc/docs/dreamcreator/conversation-and-research/)
- [Scheduled Tasks](https://dreamapp.cc/docs/dreamcreator/scheduled-tasks/)
- [Mobile Channel Access](https://dreamapp.cc/docs/dreamcreator/mobile-channel-access/)

## Acknowledgements

DreamCreator is built on top of a number of excellent open-source projects and supporting ecosystems. The desktop experience, assistant execution, media pipeline, local storage, browser automation, and channel integrations all depend on these foundations.

| Category | Homepage |
| --- | --- |
| Desktop Framework | <a href="https://go.dev/" target="_blank" rel="noreferrer">Go</a> / <a href="https://v3alpha.wails.io/" target="_blank" rel="noreferrer">Wails 3</a> / <a href="https://react.dev/" target="_blank" rel="noreferrer">React</a> |
| Local Storage | <a href="https://www.sqlite.org/" target="_blank" rel="noreferrer">SQLite</a> / <a href="https://bun.uptrace.dev/" target="_blank" rel="noreferrer">bun</a> / <a href="https://github.com/asg017/sqlite-vec" target="_blank" rel="noreferrer">sqlite-vec</a> |
| Media Processing | <a href="https://github.com/yt-dlp/yt-dlp" target="_blank" rel="noreferrer">yt-dlp</a> / <a href="https://ffmpeg.org/" target="_blank" rel="noreferrer">FFmpeg</a> |
| Browser Automation | <a href="https://playwright.dev/" target="_blank" rel="noreferrer">Playwright</a> |
| Channel Access | <a href="https://telegram.org/" target="_blank" rel="noreferrer">Telegram</a> / <a href="https://github.com/mymmrac/telego" target="_blank" rel="noreferrer">telego</a> |

These projects, and the maintainers behind them, make it possible for DreamCreator to connect desktop workflows, media processing, automation, and channel access into one evolving system.

## Collaboration

- The project is under active development, and the interface, workflow design, and channel capabilities will continue to evolve around real-world usage.
- The project is actively maintained by its author.
- Pull requests are not being accepted for now. Iteration currently moves forward through [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues) and email, including bug reports, feedback, and real usage scenarios.
- This repository is licensed under `Apache-2.0`. See [LICENSE](./LICENSE).

## Contact

- Website: <https://dreamapp.cc>
- Email: <xunruhao@gmail.com>
