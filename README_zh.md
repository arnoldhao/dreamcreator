<div align="center">
<a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="150"/></a>
</div>

<h1 align="center">CanMe</h1>

<p align="center">
  <a href="/README.md"><strong>English</strong></a> |
  <strong>简体中文</strong>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/canme?label=version" alt="版本" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="平台" />
  <img src="https://img.shields.io/badge/tech-Go%20%7C%20Vue3-green" alt="技术" />
  <img src="https://img.shields.io/badge/subtitle-ITT%20%7C%20SRT%20%7C%20FCPXML-blue" alt="字幕" />
</div>

<p align="center">
  <strong>CanMe 是一款功能强大的多语言视频下载管理工具，具备高级字幕处理功能和流畅的用户体验。</strong>
</p>

<p align="center">
  <strong>基于 <a href="https://github.com/yt-dlp/yt-dlp">yt-dlp</a>，支持多个视频平台，提供实时下载进度、多语言界面和专业字幕工作流。</strong>
</p>

<div align="center">
  <img src="images/ui_en.png" width="80%" alt="CanMe 界面" />
</div>

<br/>

## ✨ 核心功能

### 🎬 视频下载引擎
- **多平台支持** - 集成 yt-dlp，支持多种视频平台下载
- **实时进度** - 实时显示下载状态和详细进度
- **格式选择** - 支持选择视频/音频质量
- **批量处理** - 支持多任务下载与智能管理

### 📝 高级字幕系统
- **📥 导入支持** - 支持 ITT 和 SRT 字幕格式
- **📤 导出格式** - 可导出为 SRT 和 FCPXML，适配专业编辑
- **🔄 自动提取** - 自动下载视频字幕（若可用）
- **🎯 精准同步** - 保持字幕时间轴精准对齐

### 🌐 用户体验
- **多语言界面** - 完整支持英文和中文
- **跨平台** - 原生支持 Windows 和 macOS
- **现代化界面** - 基于 Vue3 + TailwindCSS + DaisyUI 打造简洁设计
- **MCP 集成** - 支持模型上下文协议，适配大模型工作流

### 🔧 技术能力
- **视频转码** - 支持多种视频/音频格式转换
- **代理支持** - 提供网络代理配置，适配全球访问
- **本地存储** - 使用 BBolt 高效管理本地数据
- **WebSocket 通信** - 实时更新与通知

## 🚀 快速上手

### 前提条件
- **系统要求** - Windows 10+ 或 macOS 10.15+
- **依赖管理** - CanMe 自动管理所有依赖（yt-dlp、FFmpeg）

### 安装

#### 📦 下载与基础设置
1. 从 [GitHub Releases](https://github.com/arnoldhao/canme/releases) 下载对应平台的最新版本
2. 解压到任意位置

#### 🍎 macOS 安装

**⚠️ macOS 用户注意**

由于未使用 Apple 开发者证书，Intel 和 ARM64 版本需额外步骤：

##### 首次启动设置
1. **右键** CanMe 应用，选择 **"打开"**
2. 在弹出的安全提示中点击 **"打开"**
3. 若提示“无法打开，因为来自未识别的开发者”：
   - 打开 **系统设置** → **安全与隐私** → **通用**
   - 在 CanMe 警告旁点击 **"仍要打开"**
   - 输入管理员密码

##### 替代方法（终端）
若上述方法无效，可使用终端：
```bash
sudo xattr -rd com.apple.quarantine /path/to/CanMe.app
```

#### 🔧 内置依赖管理
- yt-dlp & FFmpeg：自动管理，无需手动安装
- Chrome Cookies：支持自动同步，提升平台访问能力
- 网络代理：内置代理配置，支持全球访问

#### 🪟 Windows 安装
1. 解压下载的压缩包
2. 直接运行 CanMe.exe，无需额外设置
3. Windows Defender 可能提示警告，点击“更多信息” → “仍要运行”

### 🚀 即可使用
安装完成后，CanMe 提供：
- ✅ 零配置 - 自动管理所有依赖
- ✅ Chrome Cookie 同步 - 无缝访问需认证内容（macOS）
- ✅ 多平台支持 - 支持多个视频平台下载
- ✅ 专业字幕工具 - 支持 ITT/SRT 导入，SRT/FCPXML 导出

### 🔍 macOS 问题排查
- “应用已损坏”：使用终端命令移除隔离属性
- 权限被拒绝：确保有管理员权限，尝试系统设置方法
- 应用无法启动：查看 Console.app 获取详细错误信息

### 通用问题
- 下载失败：检查网络连接和代理设置
- 功能缺失：确保使用最新版本
- 性能问题：关闭其他高资源占用应用

### ⚠️ 系统要求
- 💾 **存储**：需足够空间用于下载和处理

### 📌 已知限制
- YouTube 字幕下载可能无进度显示（下载可正常完成）
- 下载暂停/恢复功能将在未来版本支持
- 部分平台可能需额外认证

## 🔮 开发路线图

### 🎯 短期目标
- **增强字幕处理**
  - 🤖 AI 驱动的字幕翻译
  - 📺 字幕直接嵌入视频
  - 🔄 批量字幕处理
  - 🎨 字幕样式与格式化选项

### 🚀 长期愿景
- **AI 增强工作流**
  - 💬 智能内容助手
  - 📝 教育工具（语言学习、作文审阅）
  - 📊 内容分析与推荐
  - 🧠 智能内容分类

## 🛠️ 技术栈

- **后端**：Go + Wails 框架
- **前端**：Vue3 + TailwindCSS + DaisyUI
- **视频处理**：yt-dlp + FFmpeg
- **存储**：BBolt 嵌入式数据库
- **通信**：WebSocket 实现实时更新

## 📖 项目理念

> CanMe 是现代应用开发的探索之旅，融合了稳健的后端工程与优雅的前端设计。作为实用工具与学习平台，它致力于探索视频处理、用户体验设计与跨平台开发的交集。

## 🤝 贡献

CanMe 作为个人学习项目，欢迎反馈与建议。代码库持续完善，感谢您的理解与耐心。

---

<p align="center">© 2025 <a href="https://github.com/arnoldhao">Arnold Hao</a>. 保留所有权利。</p>