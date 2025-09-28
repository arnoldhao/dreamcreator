<div align="center">
  <a href="https://github.com/arnoldhao/dreamcreator/"><img src="frontend/src/assets/images/icon.png" width="140" alt="dreamcreator 图标" /></a>
</div>

<h1 align="center">追创作（dreamcreator）</h1>

<p align="center">
  <strong>简体中文</strong> |
  <a href="./README_en.md"><strong>English</strong></a>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/dreamcreator?label=version" alt="最新版本" />
  <img src="https://img.shields.io/badge/license-Apache--2.0-blue" alt="许可证" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="支持平台" />
  <img src="https://img.shields.io/badge/stack-Go%20%E2%80%A2%20Wails%20%E2%80%A2%20Vue3-green" alt="技术栈" />
</div>

> 追创作（dreamcreator）是一款面向视频创作者的开源桌面工作站，聚焦“素材获取 → 字幕编辑 → 全球发行”的完整链路。依托 Wails + Vue 技术栈，集成 yt-dlp、FFmpeg 与繁化姬字幕引擎，并可扩展自定义 AI 适配器。

---

## 目录
- [愿景与定位](#愿景与定位)
- [能力地图](#能力地图)
  - [素材获取 Download](#素材获取-download)
  - [字幕编辑 Subtitle](#字幕编辑-subtitle)
  - [全球发行 Transcode（规划中）](#全球发行-transcode规划中)
  - [依赖自愈与可观测性](#依赖自愈与可观测性)
- [快速上手](#快速上手)
- [工作流速览](#工作流速览)
  - [下载任务面板](#下载任务面板)
  - [字幕工作台](#字幕工作台)
- [配置与依赖管理](#配置与依赖管理)
- [路线图](#路线图)
- [文档与支持](#文档与支持)
- [贡献指南](#贡献指南)
- [许可证](#许可证)

---

## 愿景与定位
追创作的目标是成为视频创作者的“趁手工具”，而非简单的下载器。我们坚持：
- **创作者优先**：界面保持低干扰，功能全部围绕“获取素材、打磨字幕、准备跨语言交付”设计。
- **可依赖的基础设施**：代理、Cookies、依赖校验、日志、窗口记忆等能力由内置服务托管，保障生产力。
- **开放路线图**：所有能力在 GitHub 上开源，欢迎社区一起完善面向创作者的工作流。

更多背景可阅读官网文章：[dreamcreator.dreamapp.cc](https://dreamapp.cc/zh-CN/products/dreamcreator/)。

## 能力地图

### 素材获取 Download
- **Cookies 管理**：支持浏览器同步（Chrome/Edge/Firefox/Brave/Vivaldi 等）与手动导入 Netscape / JSON / Header String；下载前自动检测可用 Cookies。
- **代理 & 网络策略**：全局 HTTP / SOCKS 代理配置、PAC 支持，解决地域限制与速率限制问题。
- **多格式拉流**：调用 yt-dlp 解析可下载的视频/音频/字幕，提供自定义选择与快速下载模式。
- **任务可视化**：并行跟踪“探测 → 获取 → 合并 → 收尾”阶段，实时显示速度、剩余时间与输出文件。

详细说明：[素材获取文档](https://dreamapp.cc/zh-CN/docs/dreamcreator/download)。

### 字幕编辑 Subtitle
- **多格式导入**：支持从下载任务或本地文件导入 SRT、VTT、ASS/SSA、ITT、FCPXML 等格式。
- **指导标准**：内置 Netflix / BBC / ADE 三套指标，实时显示 Duration、CPS、WPM、CPL “交通灯”反馈。
- **中文本地化**：集成繁化姬，提供大陆 / 香港 / 台湾等地区化转换；AI 翻译适配器正在接入。
- **多语言并行**：右下角语言切换器可在多语稿之间往返编辑，导出时保留帧率与分辨率信息。

详细说明：[字幕编辑文档](https://dreamapp.cc/zh-CN/docs/dreamcreator/subtitles)。

### 全球发行 Transcode（规划中）
- 当前版本由 yt-dlp 内部调用 FFmpeg 完成基础转码，后续将引入自研管线，支持可视化进度、GPU 转码与多种发行预设。
- 规划能力：音频转字幕、跨语言翻译、唇形同步等创作链路增强。

路线图详情见 [全球发行文档](https://dreamapp.cc/zh-CN/docs/dreamcreator/transcode)。

### 依赖自愈与可观测性
- **依赖管理**：后台监控 yt-dlp / FFmpeg 版本，支持快速校验、深度验证与镜像更新，自动校验 SHA 后切换。
- **日志系统**：可配置等级、文件大小与归档策略，日志默认存放在 `~/.dreamcreator/logs`。
- **服务状态**：设置页可查看 WebSocket / MCP 监听端口，便于调试扩展。
- **数据存储**：BoltDB 持久化任务、字幕与 Cookies，路径可自定义。

配置说明：[软件配置文档](https://dreamapp.cc/zh-CN/docs/dreamcreator/settings)。

## 快速上手
1. **下载发行包**：访问 [GitHub Releases](https://github.com/arnoldhao/dreamcreator/releases)。macOS 选择对应架构的 `.dmg`，Windows 选择 `.exe` 安装包或 `.zip` 便携版。
2. **通过系统安全提示**：
   - macOS：右键 → “打开”，或执行
     ```bash
     sudo xattr -rd com.apple.quarantine /Applications/DreamCreator.app
     ```
   - Windows：首次运行点击 “更多信息 → 仍要运行”。
3. **首次启动**：应用会自动释放 yt-dlp 与 FFmpeg；如需代理，请先在 **偏好设置 → 网络** 配置。
4. **准备 Cookies**：在浏览器登录目标站点后，前往 **下载 → 浏览器 Cookies** 同步，或在自定义集合粘贴导出的 Netscape/JSON/Header 数据。

源码编译与更多细节请参考 [安装与更新指南](https://dreamapp.cc/zh-CN/docs/dreamcreator/setup)。

## 工作流速览

### 下载任务面板
1. 点击“新建任务”解析 URL。
2. 系统检测 Cookies，可随时切换集合。
3. 选择自定义下载或快速下载：
   - 自定义模式可手选音视频轨与字幕。
   - 快速模式自动选取最高质量轨道。
4. 任务卡片提供实时进度、文件列表与快捷操作。

> 当前依赖 yt-dlp，尚未支持暂停 / GPU 转码；我们正在迭代专属转码管线。

### 字幕工作台
1. 从任务详情点击“编辑”或在字幕页面导入文件。
2. 按需切换 Netflix / BBC / ADE 指导标准，利用交通灯提示快速修正时长、CPS、WPM、CPL。
3. 通过“添加语言”调用繁化姬进行地道地区化转换；AI 翻译即将上线。
4. 导出为 SRT、VTT、ASS/SSA、ITT 或 Final Cut XML，帧率与分辨率可一键调整。

## 配置与依赖管理
- **通用设置**：外观、语言、代理、下载目录、数据目录等全局配置。
- **依赖面板**：一键执行“快速校验 / 验证 / 检查更新”，并提供“修复”“更新”按钮。
- **日志设置**：调整日志等级与存储策略，审计下载与字幕流程。
- **监听信息**：查看 WebSocket 与 MCP 端口，用于集成外部脚本或 IDE 插件。

更多细节：[软件配置文档](https://dreamapp.cc/zh-CN/docs/dreamcreator/settings)。

## 路线图
- AI 音频转字幕（Audio to Subtitle）
- AI 音频翻译（Audio Translate）
- AI 视频唇形同步（Lip Sync）
- 原生转码与发行管线（GPU / 批量 / 预设模板）

路线图保持更新，欢迎在 [GitHub Issues](https://github.com/arnoldhao/dreamcreator/issues) 参与讨论。

## 文档与支持
- 产品概览：[dreamapp.cc/products/dreamcreator](https://dreamapp.cc/products/dreamcreator)
- 中文文档导航：[dreamapp.cc/zh-CN/docs/dreamcreator](https://dreamapp.cc/zh-CN/docs/dreamcreator)
- 英文文档导航：[dreamapp.cc/docs/dreamcreator](https://dreamapp.cc/docs/dreamcreator)
- 邮件支持：xunruhao@gmail.com

## 许可证
本项目以 [Apache License 2.0](LICENSE) 开源
