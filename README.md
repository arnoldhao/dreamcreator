<div align="center">
  <img src="./frontend/public/appicon.png" width="112" alt="追创作 / Dream Creator 图标" />
  <h1>追创作 / Dream Creator</h1>
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

追创作是一款面向内容创作者的人工智能助手，把原本分散在素材下载、字幕处理、资料检索、桌面操作与移动沟通中的工作，收束到同一个入口。它不只是工具的组合，更是一位能够理解上下文、持续推进任务的创作助手：从素材采集、字幕处理、转码导出，到资料研究、灵感整理与跨设备跟进，创作者日常反复切换的一整套流程，如今可以在同一套系统里连续完成。

## 覆盖每一个创作现场

- 🎬 创作准备更集中：素材、字幕、资料与灵感被收束到同一工作台，创作前期最分散的准备工作，可以更快沉淀为可复用的资源与更清晰的方向。
- 🖥️ 桌面执行更完整：在用户授权下，助手不止能够理解需求，还能访问网络与本地电脑，调用工具、技能与记忆系统，把对话、操作与执行合并成同一条工作流。
- 📱 离开桌面也不中断：强大的多渠道接入能力，让同一位助手不被桌面限制。当前支持 Telegram 等渠道接入（飞书 / 微信开发中），离开电脑后，任务依然可以继续推进。

## 核心能力

- 📥 视频下载：千余个视频网站素材一键下载，支持身份认证，无论是 YouTube 的 BGM 还是 Bilibili 的 4K 视频，都可以沉淀为你的资源库。
- 📝 字幕校对与翻译：已有字幕不必推倒重来，校对、翻译、复核一套衔接完成，让跨语言发布更稳，也让内容复用更高效。
- 🎞️ 视频转码与字幕烧录：下载、翻译、转码、字幕内嵌可串成一条流水线，原本分散的发布前处理，现在可以一键交付。
- 💡 对话式资料检索与灵感激发：资料查找、信息整理、观点延展都能在对话里完成，从灵感萌发到结构成形不必跳出当前工作流。
- 🤖 可执行的人工智能助手：不止于回答问题，在授权范围内还可访问网页、操作电脑、调用工具，把想法直接推进为结果。
- 🧩 多助手切换机制：不同场景可定义不同助手，分别承载各自的角色、记忆与能力边界，让研究、处理与发布互不干扰。
- ⚙️ 人工智能原生自动化：重复性的下载、整理、处理与定时任务可以持续运行，把有限精力留给判断、审美与创作本身。
- 📲 移动端渠道接入：同一套能力不被桌面绑定，当前支持 Telegram 等渠道接入（飞书 / 微信开发中），手机上也能持续跟进任务。

## 产品界面

![追创作中文界面预览](./images/ui_chs.png)

## 快速开始

下载应用并完成一次基础配置，即可开始搭建自己的创作工作流。

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

### 基础配置

- 应用以本地方式运行，首次使用前需配置可用的模型服务商 API 密钥。
- 视频、字幕与自动化能力依赖 `yt-dlp`、`FFmpeg`、`bun`、`playwright` 等外部工具；首次打开时，应用会引导完成安装。

### 首次体验

1. 配置模型服务商并完成外部工具安装。
2. 粘贴一个视频链接，或直接发起一次字幕校对、翻译或转码任务。
3. 在资源库或对话中继续衔接后续处理，完成一次完整工作流。

## 感谢

追创作建立在一系列优秀的开源项目与生态能力之上。项目的桌面体验、助手执行、媒体处理、本地存储、浏览器自动化与渠道接入，都离不开这些依赖的支持。

| 分类 | 项目主页 |
| --- | --- |
| 桌面框架 | <a href="https://go.dev/" target="_blank" rel="noreferrer">Go</a> / <a href="https://v3alpha.wails.io/" target="_blank" rel="noreferrer">Wails 3</a> / <a href="https://react.dev/" target="_blank" rel="noreferrer">React</a> |
| 本地存储 | <a href="https://www.sqlite.org/" target="_blank" rel="noreferrer">SQLite</a> / <a href="https://bun.uptrace.dev/" target="_blank" rel="noreferrer">bun</a> / <a href="https://github.com/asg017/sqlite-vec" target="_blank" rel="noreferrer">sqlite-vec</a> |
| 媒体处理 | <a href="https://github.com/yt-dlp/yt-dlp" target="_blank" rel="noreferrer">yt-dlp</a> / <a href="https://ffmpeg.org/" target="_blank" rel="noreferrer">FFmpeg</a> |
| 浏览器自动化 | <a href="https://playwright.dev/" target="_blank" rel="noreferrer">Playwright</a> |
| 渠道接入 | <a href="https://telegram.org/" target="_blank" rel="noreferrer">Telegram</a> / <a href="https://github.com/mymmrac/telego" target="_blank" rel="noreferrer">telego</a> |

正是这些项目与它们背后的维护者，让追创作能够在桌面、媒体处理、自动化与渠道接入之间建立起一条完整而持续演进的工作链路。

## 协作

- 项目正在持续演进，界面体验、工作流设计与渠道能力会围绕真实使用场景继续完善。
- 项目由作者持续维护。
- 当前暂不接受 PR，主要通过 [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues) 或邮件反馈问题、分享建议与真实使用场景，持续推进迭代。
- 仓库采用 `Apache-2.0` 许可证，详见 [LICENSE](./LICENSE)。

## 联系

- 官网：<https://dreamcreator.dreamapp.cc>
- 邮箱：<xunruhao@gmail.com>
