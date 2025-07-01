<div align="center">
<a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="150"/></a>
</div>

<h1 align="center">CanMe</h1>

<p align="center">
  <strong>English</strong> |
  <a href="/README_zh.md"><strong>简体中文</strong></a>
</p>

<div align="center">
  <img src="https://img.shields.io/github/v/tag/arnoldhao/canme?label=version" alt="Version" />
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS-lightgrey" alt="Platform" />
  <img src="https://img.shields.io/badge/tech-Go%20%7C%20Vue3-green" alt="Tech" />
  <img src="https://img.shields.io/badge/subtitle-ITT%20%7C%20SRT%20%7C%20FCPXML-blue" alt="Subtitle" />
</div>

<p align="center">
  <strong>CanMe is a comprehensive multilingual video download manager with advanced subtitle processing capabilities and a fluid user experience.</strong>
</p>

<p align="center">
  <strong>Built on <a href="https://github.com/yt-dlp/yt-dlp">yt-dlp</a>, supporting multiple video platforms with real-time download progress, multilingual interface, and professional subtitle workflow.</strong>
</p>

<div align="center">
  <img src="images/ui_en.png" width="80%" alt="CanMe UI" />
</div>

<br/>

## ✨ Core Features

### 🎬 Video Download Engine
- **Multi-platform Support** - Download from various video platforms with yt-dlp integration
- **Real-time Progress** - Live download status with detailed progress indicators
- **Format Selection** - Choose from available video/audio quality options
- **Batch Processing** - Queue multiple downloads with smart management

### 📝 Advanced Subtitle System
- **📥 Import Support** - ITT and SRT subtitle format import
- **📤 Export Formats** - Export to SRT and FCPXML for professional editing
- **🔄 Auto-extraction** - Automatically download video subtitles when available
- **🎯 Precision Timing** - Maintain accurate subtitle synchronization

### 🌐 User Experience
- **Multilingual Interface** - Complete English and Chinese language support
- **Cross-platform** - Native support for Windows and macOS
- **Modern UI** - Clean design built with Vue3 + TailwindCSS + DaisyUI
- **MCP Integration** - Model Context Protocol support for LLM workflows

### 🔧 Technical Capabilities
- **Video Recoding** - Convert between different video/audio formats
- **Proxy Support** - Network proxy configuration for global access
- **Local Storage** - Efficient local data management with BBolt
- **WebSocket Communication** - Real-time updates and notifications

## 🚀 Getting Started

### Prerequisites
- **System Requirements** - Windows 10+ or macOS 10.15+
- **Dependencies** - All required dependencies (yt-dlp, FFmpeg) are automatically managed by CanMe

### Installation

#### 📦 Download & Basic Setup
1. Download the latest release for your platform from [GitHub Releases](https://github.com/arnoldhao/canme/releases)
2. Extract the downloaded archive to your preferred location

#### 🍎 macOS Installation

**⚠️ Important for macOS Users**

Due to the lack of Apple Developer Certificate, both Intel and ARM64 versions require additional steps:

##### First Launch Setup
1. **Right-click** on the CanMe app and select **"Open"**
2. Click **"Open"** in the security dialog that appears
3. If you see "CanMe cannot be opened because it is from an unidentified developer":
   - Go to **System Preferences** → **Security & Privacy** → **General**
   - Click **"Open Anyway"** next to the CanMe warning message
   - Enter your admin password when prompted

##### Alternative Method (Terminal)
If the above doesn't work, you can use Terminal:
```bash
sudo xattr -rd com.apple.quarantine /path/to/CanMe.app
```

#### 🔧 Built-in Dependency Management
- yt-dlp & FFmpeg : Automatically managed - no manual installation required
- Chrome Cookies : Automatic synchronization support for enhanced platform access
- Network Proxy : Built-in proxy configuration for global access

#### 🪟 Windows Installation
1. Extract the downloaded archive
2. Run CanMe.exe directly - no additional setup required
3. Windows Defender may show a warning - click "More info" → "Run anyway"

### 🚀 Ready to Use
Once installed, CanMe is ready to use with:

- ✅ Zero Configuration - All dependencies automatically managed
- ✅ Chrome Cookie Sync - Seamless access to authenticated content (macOS)
- ✅ Multi-platform Support - Download from various video platforms
- ✅ Professional Subtitle Tools - ITT/SRT import, SRT/FCPXML export

### 🔍 Troubleshooting macOS Issues
- "App is damaged" : Use the Terminal command above to remove quarantine attributes
- Permission denied : Ensure you have admin privileges and try the System Preferences method
- App won't start : Check Console.app for detailed error messages General Issues
- Download failures : Check your internet connection and proxy settings
- Missing features : Ensure you downloaded the latest version
- Performance issues : Close other resource-intensive applications

### ⚠️ System Requirements
- 💾 **Storage**: Adequate disk space for downloads and processing

### 📌 Known Limitations
- YouTube subtitle downloads may not show progress updates (downloads complete successfully)
- Download pause/resume functionality planned for future releases
- Some platforms may require additional authentication

## 🔮 Development Roadmap

### 🎯 Short-term Goals
- **Enhanced Subtitle Pipeline**
  - 🤖 AI-powered subtitle translation
  - 📺 Direct subtitle embedding in videos
  - 🔄 Batch subtitle processing
  - 🎨 Subtitle styling and formatting options

### 🚀 Long-term Vision
- **AI-Enhanced Workflow**
  - 💬 Intelligent content assistant
  - 📝 Educational tools (language learning, essay review)
  - 📊 Content analysis and recommendations
  - 🧠 Smart content categorization

## 🛠️ Technical Stack

- **Backend**: Go with Wails framework
- **Frontend**: Vue3 + TailwindCSS + DaisyUI
- **Video Processing**: yt-dlp + FFmpeg
- **Storage**: BBolt embedded database
- **Communication**: WebSocket for real-time updates

## 📖 Project Philosophy

> CanMe represents a journey in modern application development, combining robust backend engineering with elegant frontend design. This project serves as both a practical tool and a learning platform, exploring the intersection of video processing, user experience design, and cross-platform development.

## 🤝 Contributing

As a personal learning project, CanMe welcomes feedback and suggestions. While the codebase continues to evolve, your understanding and patience with ongoing improvements are appreciated.

---

<p align="center">© 2025 <a href="https://github.com/arnoldhao">Arnold Hao</a>. All rights reserved.</p>