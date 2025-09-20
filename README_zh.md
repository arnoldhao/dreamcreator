<div align="center">
  <a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="140" alt="CanMe 标志" /></a>
</div>

<h1 align="center">CanMe</h1>

<p align="center">
  <a href="/README.md"><strong>English</strong></a> |
  <strong>简体中文</strong>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/canme?label=version" alt="最新版本" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="许可证" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="平台" />
  <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20Vue3-green" alt="技术栈" />
</div>

<br />

CanMe 是一个开源的视频下载与字幕处理桌面工具。后端使用 Go + Wails，结合 BoltDB 持久化与 WebSocket 消息；前端基于 Vue 3 + Tailwind。项目重点围绕可重复的下载流程与可审计的字幕处理，后续会在辅助能力上继续迭代。

<div align="center">
  <img src="images/ui_en.png" width="85%" alt="CanMe 界面" />
</div>

## 目录
- [项目亮点](#项目亮点)
- [功能概览](#功能概览)
- [快速开始](#快速开始)
- [从源码构建](#从源码构建)
- [使用说明](#使用说明)
- [路线图](#路线图)
- [许可证](#许可证)
- [致谢](#致谢)

## 项目亮点
- 提供 Windows 10+/macOS 10.15+ 桌面发行版，内置 yt-dlp 与 FFmpeg
- 依赖管理器负责版本固定、校验、镜像回落与自动修复
- 使用 BoltDB 存储下载任务、Cookie 快照与字幕工程，确保数据可追踪
- Vue 3 + Pinia UI 通过 WebSocket 实时呈现状态，内置中英文双语
- Apache License 2.0 授权，正在拆分 AI 翻译与 Cookie 模块以便扩展

## 功能概览

### 下载管线
- **多平台下载：** 基于 yt-dlp 的封装，支持逐任务格式选择与阶段化进度（探测 → 下载 → 合并 → 完成）
- **代理感知：** 支持全局 HTTP/SOCKS 代理、PAC、Windows 提权等场景
- **依赖监控：** 背景校验 yt-dlp/FFmpeg 是否可用，损坏时自动重新下载并验证哈希

### 字幕能力
- **导入格式：** 通过专用解析器支持 `.srt`、`.vtt/.webvtt`、`.ass/.ssa`、`.itt`（参见 `backend/core/subtitles/format.go`）
- **文本规范化：** 可配置标点清理、空白处理、繁简转换（`pkg/zhconvert`）
- **质量评估：** `quality_assessor.go` 提供时序间隔、片段长度、字符密度等指标
- **工程建模：** 以 `types.SubtitleProject` 持久化字幕，包含语言索引与导出配置
- **导出格式：** 支持 SRT、VTT、ASS/SSA、ITT、Final Cut Pro XML，并自动补全帧率、分辨率、轨道元数据
- **翻译阶段：** 下载任务生命周期预留 `translate` 阶段（`service.go:675+`），可挂接外部 MT/LLM 适配器；当前默认返回占位结果，避免误触发外部调用
- **嵌入接口：** 转码管线可在 FFmpeg 阶段烧录或附加字幕轨

### Cookie 管理
- 覆盖 Windows/macOS 上的 Chrome、Chromium、Edge、Firefox、Safari、Brave、Opera、Vivaldi，并为每个浏览器记录状态与同步历史
- 导出 Netscape Cookie 供 yt-dlp 或高级调试使用，可针对域名筛选并清理中间文件
- 支持手动维护集合，可粘贴 Netscape 文本、浏览器 DevTools 导出的 JSON 数组，或直接粘贴 `Cookie:` 头串，在无需浏览器同步的情况下进行合并或覆盖
- 通过事件总线推送同步进度，前端实时展示状态与错误信息

### 媒体转换
- 基于 FFmpeg 的常用转封装、音频提取、字幕附加预设，并缓存依赖元数据
- 任务分类（`video`/`subtitle`/`other`）方便在下载面板中筛选附属文件

## 快速开始

### 前提条件
- Windows 10 或 macOS 10.15 及以上
- 足够的磁盘空间以满足视频与 FFmpeg 临时文件需求
- 首次启动自动下载并校验 yt-dlp 与 FFmpeg，无需手动安装

### 安装
1. 在 [GitHub Releases](https://github.com/arnoldhao/canme/releases) 下载最新版本
2. 解压到可写目录
3. 启动应用

#### macOS Gatekeeper
未签名版本需要手动确认：
- 右键应用选择 **打开**，或
- 使用终端移除隔离属性：
  ```bash
  sudo xattr -rd com.apple.quarantine /path/to/CanMe.app
  ```

#### Windows SmartScreen
- 首次运行时点击 **更多信息 → 仍要运行**

### 依赖管理
- yt-dlp、FFmpeg 保存在应用的托管缓存目录，并执行 SHA 校验
- Windows 同步浏览器 Cookie 时可能需要管理员权限，界面会提示是否需要提升权限

## 从源码构建
> 普通用户可直接使用发行版。以下步骤面向开发者与贡献者。

### 环境要求
- Go 1.24+
- Node.js 18+（配合 npm 或 pnpm）
- Wails CLI（`go install github.com/wailsapp/wails/v2/cmd/wails@latest`）

### 构建流程
```bash
# 后端模块
go mod tidy

# 前端资源
cd frontend
npm install
npm run build
cd ..

# 构建桌面应用
wails build
```

如需热更新，可在仓库根目录运行 `wails dev`，并在 `frontend` 中执行 `npm run dev` 以启用 Vite 开发服务器。

## 使用说明
- 先在 Chrome / Edge 登录目标站点，再在 **Cookies → 同步** 中刷新认证；若暂时无法同步，也可以在手动集合中粘贴 Netscape/JSON/Header 数据
- 调度器并行获取元数据，但会串行执行重型合并和转码，避免磁盘竞争
- 导入的字幕工程在字幕工作台展示，可先检查质量评分，再接入翻译适配器执行 `translate` 阶段
- 若下载异常或受地区限制，可在 **偏好设置 → 网络** 中配置代理

## 路线图
进度可在 GitHub Issues/Projects 中查看。近期重点：
- 为翻译阶段接入外部 AI 服务，并补充重试与审计日志
- 扩展 Cookie 工具：手动导入导出、冲突解决、沙箱化抓取
- 提供 FFmpeg 预设目录（面向平台/社交媒体）和批量转换脚本

## 许可证

CanMe 依据 Apache License 2.0 发布，详见 `LICENSE` 文件。

## 致谢
- [yt-dlp](https://github.com/yt-dlp/yt-dlp)
- [FFmpeg](https://ffmpeg.org/)
- [Wails](https://wails.io/)
- [Vue](https://vuejs.org/)
- [TailwindCSS](https://tailwindcss.com/)
