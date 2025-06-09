<div align="center">
<a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="150"/></a>
</div>

<h1 align="center">CanMe</h1>

<p align="center">
  <a href="/README.md"><strong>English</strong></a> |
  <strong>简体中文</strong>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/canme?label=版本" alt="版本" />
  <img src="https://img.shields.io/badge/平台-Windows%20%7C%20macOS-lightgrey" alt="平台" />
  <img src="https://img.shields.io/badge/技术-Go%20%7C%20Vue3-green" alt="技术" />
  <img src="https://img.shields.io/badge/字幕-ITT%20%7C%20SRT%20%7C%20FCPXML-blue" alt="字幕" />
</div>

<p align="center">
  <strong>CanMe 是一个功能全面的多语言视频下载管理器，具备先进的字幕处理能力和流畅的用户体验。</strong>
</p>

<p align="center">
  <strong>基于 <a href="https://github.com/yt-dlp/yt-dlp">yt-dlp</a> 构建，支持多个视频平台，具有实时下载进度、多语言界面和专业字幕工作流程。</strong>
</p>

<div align="center">
  <img src="images/ui_chs.png" width="80%" alt="CanMe UI" />
</div>

<br/>

## ✨ 核心功能

### 🎬 视频下载引擎
- **多平台支持** - 通过 yt-dlp 集成从各种视频平台下载
- **实时进度** - 带有详细进度指示器的实时下载状态
- **格式选择** - 从可用的视频/音频质量选项中选择
- **批量处理** - 智能管理多个下载队列

### 📝 高级字幕系统
- **📥 导入支持** - ITT 和 SRT 字幕格式导入
- **📤 导出格式** - 导出为 SRT 和 FCPXML 格式，用于专业编辑
- **🔄 自动提取** - 在可用时自动下载视频字幕
- **🎯 精确时间** - 保持准确的字幕同步

### 🌐 用户体验
- **多语言界面** - 完整的英文和中文语言支持
- **跨平台** - 原生支持 Windows 和 macOS
- **现代化界面** - 使用 Vue3 + TailwindCSS + DaisyUI 构建的简洁设计
- **MCP 集成** - 支持 LLM 工作流程的模型上下文协议

### 🔧 技术能力
- **视频转码** - 在不同视频/音频格式之间转换
- **代理支持** - 网络代理配置，支持全球访问
- **本地存储** - 使用 BBolt 的高效本地数据管理
- **WebSocket 通信** - 实时更新和通知

## 🚀 快速开始

### 前置要求
- **FFmpeg** - 视频处理和格式转换所需
- **稳定网络** - 初始设置需要下载必要的 yt-dlp 组件
- **系统要求** - Windows 10+ 或 macOS 10.15+

### 安装
1. 下载适合您平台的最新版本
2. 在系统上安装 FFmpeg
3. 启动 CanMe 并按照设置向导操作

## 📋 版本信息

### 🆕 最新更新
- ✨ **新字幕导出系统** - 专业的 ITT/SRT 导入和 SRT/FCPXML 导出
- 🔄 增强的下载引擎，采用 yt-dlp 核心集成
- 🎨 重新设计的界面，改善用户体验
- 🧹 精简的代码库，优化性能
- 🔧 高级视频转码功能

### ⚠️ 系统要求
- 🔧 **依赖项**: 需要安装 FFmpeg
- 🌐 **网络**: 初始组件下载需要互联网连接
- 💾 **存储**: 下载和处理需要足够的磁盘空间

### 📌 已知限制
- YouTube 字幕下载可能不显示进度更新（下载会成功完成）
- 下载暂停/恢复功能计划在未来版本中推出
- 某些平台可能需要额外的身份验证

## 🔮 开发路线图

### 🎯 短期目标
- **增强字幕流水线**
  - 🤖 AI 驱动的字幕翻译
  - 📺 视频中直接嵌入字幕
  - 🔄 批量字幕处理
  - 🎨 字幕样式和格式选项

### 🚀 长期愿景
- **AI 增强工作流程**
  - 💬 智能内容助手
  - 📝 教育工具（语言学习、作文评审）
  - 📊 内容分析和推荐
  - 🧠 智能内容分类

## 🛠️ 技术栈

- **后端**: Go 与 Wails 框架
- **前端**: Vue3 + TailwindCSS + DaisyUI
- **视频处理**: yt-dlp + FFmpeg
- **存储**: BBolt 嵌入式数据库
- **通信**: WebSocket 实时更新

## 📖 项目理念

> CanMe 代表了现代应用程序开发的探索之旅，将强大的后端工程与优雅的前端设计相结合。这个项目既是一个实用工具，也是一个学习平台，探索视频处理、用户体验设计和跨平台开发的交汇点。

## 🤝 贡献

作为个人学习项目，CanMe 欢迎反馈和建议。虽然代码库持续发展，但感谢您对持续改进的理解和耐心。

---

<p align="center">© 2025 <a href="https://github.com/arnoldhao">Arnold Hao</a>. 保留所有权利。</p>