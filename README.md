<div align="center">
  <img src="./frontend/public/appicon.png" width="112" alt="追创作 / DreamCreator 图标" />
  <h1>追创作 / DreamCreator</h1>
  <p><strong>一款面向内容创作者的人工智能助手</strong></p>
  <p>
    <strong>简体中文</strong> ·
    <a href="./README_en.md">English</a>
  </p>
  <p>
    <img src="https://img.shields.io/github/v/tag/arnoldhao/dreamcreator?label=version" alt="最新版本" />
    <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="许可证" />
    <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="支持平台" />
    <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20React-green" alt="技术栈" />
  </p>
</div>

## 项目简介

追创作是一款AI Native桌面应用，整合视频下载、字幕处理、资料检索、桌面执行与移动跟进能力。无论你是通过界面、对话或者是移动端，都可以有序地完成创作准备，获得一致的产品体验。

## 覆盖每一个创作现场

- 🎬 **创作准备**：集中整理素材、字幕、资料与灵感线索。
- 🖥️ **桌面执行**：访问网页、调用工具并继续推进任务。
- 📱 **移动跟进**：通过 Telegram 持续查看结果与后续进展。

## 核心能力

- 📥 **视频下载**：将公开视频和需登录访问的视频下载到本地资源库。
- 📝 **字幕校对与翻译**：围绕现有字幕完成校对、翻译、QA 与导出。
- 🎞️ **视频转码与字幕烧录**：完成转码、字幕导出、内嵌字幕轨与烧录字幕。
- 💡 **对话、检索与灵感整理**：在同一条线程中整理资料与推进后续动作。
- 🤖 **人工智能助手**：在授权范围内访问网页、调用工具并操作电脑。
- 🧩 **多助手机制**：按任务拆分不同助手的角色、模型与权限边界。
- ⚙️ **定时任务**：把重复性的下载、整理、巡检与投递交给系统运行。
- 📲 **移动端渠道接入**：通过 Telegram 等渠道继续查看结果与跟进任务。

## 产品界面

![追创作中文界面预览](./images/ui_chs.png)

## 快速开始

### 下载安装

可直接下载最新安装包；历史版本见 [GitHub 发布页](https://github.com/arnoldhao/dreamcreator/releases)。

| 平台 | 架构 | 形式 | 下载 |
| --- | --- | --- | --- |
| macOS | Apple 芯片 | 压缩包 | [点击下载](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-macos-arm64-latest.zip) |
| macOS | Intel | 压缩包 | [点击下载](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-macos-x64-latest.zip) |
| Windows | x64 | 安装版 | [点击下载](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-windows-x64-latest-installer.exe) |
| Windows | x64 | 便携版 | [点击下载](https://updates.dreamapp.cc/dreamcreator/downloads/dreamcreator-windows-x64-latest.zip) |

### 首次打开

1. `macOS`：解压后将 `DreamCreator.app` 移动到“应用程序”目录。若系统提示“无法打开”或“已损坏”，请在终端执行 `sudo xattr -rd com.apple.quarantine /Applications/DreamCreator.app`。
2. `Windows`：安装版直接运行 `.exe`；便携版解压后直接启动。若首次启动出现 SmartScreen，选择“更多信息 -> 仍要运行”。
3. 完整的安装、初始化、使用与更新说明见 [安装、首次启动与更新](https://dreamapp.cc/zh-CN/docs/dreamcreator/install-and-update/)。

## 文档

- [安装、首次启动与更新](https://dreamapp.cc/zh-CN/docs/dreamcreator/install-and-update/)
- [追创作概览](https://dreamapp.cc/zh-CN/docs/dreamcreator/overview/)
- [视频下载](https://dreamapp.cc/zh-CN/docs/dreamcreator/video-download/)
- [字幕校对与翻译](https://dreamapp.cc/zh-CN/docs/dreamcreator/subtitle-proofreading-and-translation/)
- [视频转码与字幕烧录](https://dreamapp.cc/zh-CN/docs/dreamcreator/transcoding-and-subtitle-burn-in/)
- [对话、检索与灵感整理](https://dreamapp.cc/zh-CN/docs/dreamcreator/conversation-and-research/)
- [定时任务](https://dreamapp.cc/zh-CN/docs/dreamcreator/scheduled-tasks/)
- [移动端渠道接入](https://dreamapp.cc/zh-CN/docs/dreamcreator/mobile-channel-access/)

## 感谢

追创作建立在一系列优秀的开源项目与生态能力之上。项目的桌面体验、助手执行、媒体处理、本地存储、浏览器自动化与渠道接入，都离不开这些依赖的支持。

| 分类 | 项目主页 |
| --- | --- |
| 桌面框架 | <a href="https://go.dev/" target="_blank" rel="noreferrer">Go</a> / <a href="https://v3alpha.wails.io/" target="_blank" rel="noreferrer">Wails 3</a> / <a href="https://react.dev/" target="_blank" rel="noreferrer">React</a> |
| 本地存储 | <a href="https://www.sqlite.org/" target="_blank" rel="noreferrer">SQLite</a> / <a href="https://bun.uptrace.dev/" target="_blank" rel="noreferrer">bun</a> / <a href="https://github.com/asg017/sqlite-vec" target="_blank" rel="noreferrer">sqlite-vec</a> |
| 媒体处理 | <a href="https://github.com/yt-dlp/yt-dlp" target="_blank" rel="noreferrer">yt-dlp</a> / <a href="https://ffmpeg.org/" target="_blank" rel="noreferrer">FFmpeg</a> |
| 浏览器自动化 | <a href="https://chromedevtools.github.io/devtools-protocol/" target="_blank" rel="noreferrer">Chrome DevTools Protocol</a> / <a href="https://github.com/chromedp/chromedp" target="_blank" rel="noreferrer">chromedp</a> |
| 渠道接入 | <a href="https://telegram.org/" target="_blank" rel="noreferrer">Telegram</a> / <a href="https://github.com/mymmrac/telego" target="_blank" rel="noreferrer">telego</a> |

正是这些项目与它们背后的维护者，让追创作能够在桌面、媒体处理、自动化与渠道接入之间建立起一条持续演进的工作链路。

## 协作

- 项目正在持续演进，界面体验、工作流设计与渠道能力会围绕真实使用场景继续完善。
- 项目由作者持续维护。
- 当前暂不接受 PR，主要通过 [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues) 或邮件反馈问题、分享建议与真实使用场景，持续推进迭代。
- 仓库采用 `Apache-2.0` 许可证，详见 [LICENSE](./LICENSE)。

## 联系

- 官网：<https://dreamapp.cc>
- 邮箱：<xunruhao@gmail.com>
