<div align="center">
<a href="https://github.com/arnoldhao/canme/"><img src="build/appicon.png" width="120"/></a>
</div>
<h1 align="center">CanMe</h1>
<h4 align="center"><strong><a href="/">English</a></strong> | 简体中文</h4>
<div align="center">

<strong>一款基于AI的字幕转换工具，支持直接下载Youtube视频与字幕，支持将剪映字幕导出为SubRip (SRT) 格式，并可将字幕翻译为任意语言的客户端，支持Mac、Windows</strong>
</div>

## 学习项目
本项目目的为学习前后端开发，软件本身可能会有很多奇奇怪怪的问题，为个人能力水平受限，敬请谅解。

## 当前功能
### Version: 0.0.9
当前版本多项功能异常，请不要使用
- 支持导出剪映JSON字幕文件导出
- 支持直接导入SRT文件
- 支持Ollama模型下载管理
- 支持本地Ollama模型以及远程类OpenAI API调用
- 支持历史管理
- 支持同时多任务AI翻译
- 支持配置代理
- 支持Youtube视频与字幕下载
- 支持Bilibili视频下载(暂不支持登陆)

## 路线图
### version: 0.0.10+
- 重构model provider页面至配置
- 重构model config页面至配置
- 重构字幕页面
- 重构下载页面
- 支持Bilibili登陆
- 支持导入BCut

### Version: 0.1.X
- AI翻译后的字幕编辑
- 字幕格式校准
- Ollama模型下载速度展示、重试机制

### Version: 0.2.X
- AI Chat
- ChatGPT API代理

### Version: 0.3.X
- ~~直接导入Youtube字幕(已支持下载Youtbe视频与字幕)~~
- 直接导入Bilibili字幕

### Version： Future
- 流水线（Youtube -> Bilibili）
- AI底座（如雅思作文批改）