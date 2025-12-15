import {
  UpdateProjectName,
  ExportSubtitleToFile,
  UpdateProjectMetadata,
  UpdateSubtitleSegment,
  UpdateLanguageContent,
  UpdateLanguageMetadata,
  RemoveProjectLanguage,
  // 新增中文转换相关 API
  GetSupportedConverters,
  ZHConvertSubtitle,
  // LLM 翻译 + 术语表
  TranslateSubtitleLLM,
  TranslateSubtitleLLMWithOptions,
  TranslateSubtitleLLMRetryFailedWithOptions,
  TranslateSubtitleLLMWithGlobalProfile,
  TranslateSubtitleLLMWithGlobalProfileOptions,
  TranslateSubtitleLLMRetryFailedWithGlobalProfileOptions,
  ListGlossary,
  ListGlossaryBySet,
  UpsertGlossaryEntry,
  DeleteGlossaryEntry,
  ListGlossarySets,
  UpsertGlossarySet,
  DeleteGlossarySet,
  // Target languages (AI translation)
  ListTargetLanguages,
  UpsertTargetLanguage,
  DeleteTargetLanguage,
  ResetTargetLanguagesToDefault
} from 'bindings/dreamcreator/backend/api/subtitlesapi';
import { createAutoSaveManager } from '@/utils/autoSave.js';
import { useDtStore } from '@/stores/downloadTasks'
import { useSubtitleStore } from '@/stores/subtitle'
import { i18nGlobal } from '@/utils/i18n.js'

/**
 * 字幕服务类
 * 处理字幕相关的所有操作和自动保存
 */
export class SubtitleService {
  constructor() {
    this.autoSaveManager = null;
    this.currentProject = null;
    this.saveStatus = 'idle';
    this.statusCallbacks = new Set();
    this.projectUpdateCallbacks = new Set();
    // 新增中文转换相关状态
    this.supportedConverters = [];
    // 目标语言缓存（用于 code -> name 映射）
    this.targetLanguages = [];
    this.conversionCallbacks = new Set();
    this.dtStore = null;
    // 记录 WebSocket 回调绑定与订阅状态，避免重复注册造成多次提示
    this._subtitleProgressHandler = null;
    this._wsSubscribed = false;
  }

  /**
   * 初始化服务
   * @param {Object} project - 当前项目
   */
  initialize(project) {
    this.currentProject = project;

    // 创建自动保存管理器
    this.autoSaveManager = createAutoSaveManager({
      debounceTime: 1000,
      maxWaitTime: 5000,
      onSave: this.handleSave.bind(this),
      onError: this.handleSaveError.bind(this),
      onStatusChange: this.handleStatusChange.bind(this)
    });

    // 初始化中文转换功能
    this.initializeZHConvert();

    // 初始化 WebSocket 监听
    this.initializeWebSocketListeners();
  }

  /**
   * 处理保存操作
   * @param {Object} data - 保存数据
   * @param {string} type - 保存类型
   * @returns {Promise<Object>} 保存结果
   */
  async handleSave(data, type) {
    if (!this.currentProject) {
      throw new Error('No project loaded');
    }

    const projectId = this.currentProject.id;
    let result;

    try {
      switch (type) {
        case 'project_name':
          result = await UpdateProjectName(projectId, data);
          break;

        case 'metadata':
          result = await UpdateProjectMetadata(projectId, data);
          break;

        case 'segment':
          result = await UpdateSubtitleSegment(projectId, data.segmentId, data.segment);
          break;

        case 'language_content':
          result = await UpdateLanguageContent(projectId, data.segmentId, data.languageCode, data.content);
          break;

        case 'language_metadata':
          result = await UpdateLanguageMetadata(projectId, data.languageCode, data.metadata);
          break;

        default:
          throw new Error(`Unknown save type: ${type}`);
      }

      // 检查后端返回的结果
      if (!result.success) {
        throw new Error(result.msg || 'Save operation failed');
      }

      // 如果有数据返回，可以更新本地状态
      if (result.data) {
        try {
          const updatedProject = typeof result.data === 'string' ? JSON.parse(result.data) : result.data;
          this.handleProjectUpdate(updatedProject);
        } catch (parseError) {
          console.warn('Failed to parse updated project data:', parseError);
        }
      }

      return result;
    } catch (error) {
      // 不在这里处理UI提示，只抛出错误
      throw error;
    }
  }

  /**
   * 处理项目更新
   * @param {Object} updatedProject - 更新后的项目数据
   */
  handleProjectUpdate(updatedProject) {
    this.currentProject = updatedProject;
    // 同步到 Pinia store，确保页面计算属性（语言数量等）即时更新
    try {
      const store = useSubtitleStore()
      // 更新列表中的项目与当前项目引用
      if (typeof store.updateProject === 'function') {
        store.updateProject(updatedProject)
      } else if (typeof store.setCurrentProject === 'function') {
        store.setCurrentProject(updatedProject)
      } else {
        // 直接赋值兜底（不建议，但保证不丢刷新）
        store.currentProject = updatedProject
      }
    } catch (e) {
      // 安全兜底：不影响后续回调
      console.warn('sync store currentProject failed:', e)
    }
    // 触发项目更新回调
    this.projectUpdateCallbacks.forEach(callback => {
      try {
        callback(updatedProject);
      } catch (error) {
        console.error('Project update callback error:', error);
      }
    });
  }

  /**
   * 订阅项目更新
   * @param {Function} callback - 回调函数
   * @returns {Function} 取消订阅函数
   */
  onProjectUpdate(callback) {
    if (!this.projectUpdateCallbacks) {
      this.projectUpdateCallbacks = new Set();
    }
    this.projectUpdateCallbacks.add(callback);
    return () => this.projectUpdateCallbacks.delete(callback);
  }

  /**
   * 处理保存错误（仅用于日志记录）
   * @param {Error} error - 错误对象
   */
  handleSaveError(error) {
    console.error('Save error:', error);
    // 可以在这里添加错误日志记录逻辑，但不处理UI提示
  }

  /**
   * 处理状态变化
   * @param {string} status - 新状态
   */
  handleStatusChange(status) {
    this.saveStatus = status;
    this.statusCallbacks.forEach(callback => {
      try {
        callback(status);
      } catch (error) {
        console.error('Status change callback error:', error);
      }
    });
  }

  // ==================== 同步保存方法（返回Promise） ====================

  /**
   * 保存项目名称
   * @param {string} name - 项目名称
   * @returns {Promise<Object>} 保存结果
   */
  async saveProjectName(name) {
    if (!name || typeof name !== 'string') {
      throw new Error('Project name is required and must be a string');
    }
    return await this.autoSaveManager.saveNow(name.trim(), 'project_name');
  }

  /**
   * 保存项目元数据
   * @param {Object} metadata - 元数据
   * @returns {Promise<Object>} 保存结果
   */
  async saveProjectMetadata(metadata) {
    if (!metadata || typeof metadata !== 'object') {
      throw new Error('Metadata is required and must be an object');
    }
    return await this.autoSaveManager.saveNow(metadata, 'metadata');
  }

  /**
   * 保存字幕片段
   * @param {string} segmentId - 片段ID
   * @param {Object} segment - 片段数据
   * @returns {Promise<Object>} 保存结果
   */
  async saveSubtitleSegment(segmentId, segment) {
    if (!segmentId || !segment) {
      throw new Error('Segment ID and segment data are required');
    }
    return await this.autoSaveManager.saveNow({ segmentId, segment }, 'segment');
  }

  /**
   * 保存语言内容
   * @param {string} segmentId - 片段ID
   * @param {string} languageCode - 语言代码
   * @param {Object} content - 内容数据
   * @returns {Promise<Object>} 保存结果
   */
  async saveLanguageContent(segmentId, languageCode, content) {
    if (!segmentId || !languageCode || !content) {
      throw new Error('Segment ID, language code and content are required');
    }
    return await this.autoSaveManager.saveNow({ segmentId, languageCode, content }, 'language_content');
  }

  /**
   * 保存语言元数据
   * @param {string} languageCode - 语言代码
   * @param {Object} metadata - 元数据
   * @returns {Promise<Object>} 保存结果
   */
  async saveLanguageMetadata(languageCode, metadata) {
    if (!languageCode || !metadata) {
      throw new Error('Language code and metadata are required');
    }
    return await this.autoSaveManager.saveNow({ languageCode, metadata }, 'language_metadata');
  }

  /**
   * 通用立即保存方法
   * @param {Object} data - 数据
   * @param {string} type - 类型
   * @returns {Promise<Object>} 保存结果
   */
  async saveNow(data, type) {
    if (!data || !type) {
      throw new Error('Data and type are required');
    }
    return await this.autoSaveManager.saveNow(data, type);
  }

  // ==================== 异步保存方法（用于自动保存等场景） ====================

  /**
   * 异步保存字幕片段（用于自动保存等场景）
   * @param {string} segmentId - 片段ID
   * @param {Object} segment - 片段数据
   */
  saveSubtitleSegmentAsync(segmentId, segment) {
    if (!segmentId || !segment) {
      console.warn('Segment ID and segment data are required for async save');
      return;
    }
    this.autoSaveManager.save({ segmentId, segment }, 'segment');
  }

  /**
   * 异步保存项目名称（用于自动保存等场景）
   * @param {string} name - 项目名称
   */
  saveProjectNameAsync(name) {
    if (!name || typeof name !== 'string') {
      console.warn('Project name is required and must be a string for async save');
      return;
    }
    this.autoSaveManager.save(name.trim(), 'project_name');
  }

  /**
   * 异步保存项目元数据
   * @param {Object} metadata - 元数据
   */
  saveProjectMetadataAsync(metadata) {
    if (!metadata || typeof metadata !== 'object') {
      console.warn('Metadata is required and must be an object for async save');
      return;
    }
    this.autoSaveManager.save(metadata, 'metadata');
  }

  /**
   * 异步保存语言内容
   * @param {string} segmentId - 片段ID
   * @param {string} languageCode - 语言代码
   * @param {Object} content - 内容数据
   */
  saveLanguageContentAsync(segmentId, languageCode, content) {
    if (!segmentId || !languageCode || !content) {
      console.warn('Segment ID, language code and content are required for async save');
      return;
    }
    this.autoSaveManager.save({ segmentId, languageCode, content }, 'language_content');
  }

  /**
   * 异步保存语言元数据
   * @param {string} languageCode - 语言代码
   * @param {Object} metadata - 元数据
   */
  saveLanguageMetadataAsync(languageCode, metadata) {
    if (!languageCode || !metadata) {
      console.warn('Language code and metadata are required for async save');
      return;
    }
    this.autoSaveManager.save({ languageCode, metadata }, 'language_metadata');
  }

  // ==================== 中文转换功能 ====================

  /**
   * 初始化中文转换功能
   */
  async initializeZHConvert() {
    try {
      await this.loadSupportedConverters();
    } catch (error) {
      console.error('Failed to initialize ZH convert:', error);
    }
  }

  // ==================== LLM 翻译（AI） ====================

  async translateSubtitleLLM(sourceLang, targetLang, providerID, model) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    if (!sourceLang || !targetLang || !providerID || !model) throw new Error('Missing params')
    const res = await TranslateSubtitleLLM(this.currentProject.id, sourceLang, targetLang, providerID, model)
    if (!res?.success) throw new Error(res?.msg || 'Start translate failed')
    return true
  }

  async translateSubtitleLLMWithGlossary(sourceLang, targetLang, providerID, model, setIDs = [], taskTerms = [], strictGlossary = false) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    const res = await TranslateSubtitleLLMWithOptions(this.currentProject.id, sourceLang, targetLang, providerID, model, setIDs, taskTerms, strictGlossary)
    if (!res?.success) throw new Error(res?.msg || 'Start translate failed')
    return true
  }

  async retryFailedTranslations(sourceLang, targetLang, providerID, model, setIDs = [], taskTerms = [], strictGlossary = false) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    const res = await TranslateSubtitleLLMRetryFailedWithOptions(this.currentProject.id, sourceLang, targetLang, providerID, model, setIDs, taskTerms, strictGlossary)
    if (!res?.success) throw new Error(res?.msg || 'Start retry failed')
    return true
  }

  async translateSubtitleLLMWithGlobalProfile(sourceLang, targetLang, providerID, model, profileID) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    if (!sourceLang || !targetLang || !providerID || !model || !profileID) throw new Error('Missing params')
    const res = await TranslateSubtitleLLMWithGlobalProfile(this.currentProject.id, sourceLang, targetLang, providerID, model, profileID)
    if (!res?.success) throw new Error(res?.msg || 'Start translate failed')
    return true
  }

  async translateSubtitleLLMWithGlobalProfileAndGlossary(sourceLang, targetLang, providerID, model, profileID, setIDs = [], taskTerms = [], strictGlossary = false) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    const res = await TranslateSubtitleLLMWithGlobalProfileOptions(this.currentProject.id, sourceLang, targetLang, providerID, model, profileID, setIDs, taskTerms, strictGlossary)
    if (!res?.success) throw new Error(res?.msg || 'Start translate failed')
    return true
  }

  async retryFailedTranslationsWithGlobalProfile(sourceLang, targetLang, providerID, model, profileID, setIDs = [], taskTerms = [], strictGlossary = false) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    const res = await TranslateSubtitleLLMRetryFailedWithGlobalProfileOptions(this.currentProject.id, sourceLang, targetLang, providerID, model, profileID, setIDs, taskTerms, strictGlossary)
    if (!res?.success) throw new Error(res?.msg || 'Start retry failed')
    return true
  }

  // ==================== 术语表（Glossary） ====================

  async listGlossary() {
    const res = await ListGlossary()
    if (!res?.success) throw new Error(res?.msg || 'List glossary failed')
    const raw = res.data
    if (Array.isArray(raw)) return raw
    try { return JSON.parse(raw || '[]') } catch { return [] }
  }
  async listGlossaryBySet(setID) {
    const res = await ListGlossaryBySet(setID)
    if (!res?.success) throw new Error(res?.msg || 'List glossary by set failed')
    const raw = res.data
    if (Array.isArray(raw)) return raw
    try { return JSON.parse(raw || '[]') } catch { return [] }
  }

  async upsertGlossaryEntry(entry) {
    const res = await UpsertGlossaryEntry(entry)
    if (!res?.success) throw new Error(res?.msg || 'Upsert glossary failed')
    const raw = res.data
    if (raw && typeof raw === 'object') return raw
    try { return JSON.parse(raw || '{}') } catch { return null }
  }

  async deleteGlossaryEntry(id) {
    const res = await DeleteGlossaryEntry(id)
    if (!res?.success) throw new Error(res?.msg || 'Delete glossary failed')
    return true
  }

  // Glossary sets (for Settings management and modal selection)
  async listGlossarySets() {
    const res = await ListGlossarySets()
    if (!res?.success) throw new Error(res?.msg || 'List glossary sets failed')
    const raw = res.data
    if (Array.isArray(raw)) return raw
    try { return JSON.parse(raw || '[]') } catch { return [] }
  }
  async upsertGlossarySet(gs) {
    const res = await UpsertGlossarySet(gs)
    if (!res?.success) throw new Error(res?.msg || 'Upsert glossary set failed')
    const raw = res.data
    if (raw && typeof raw === 'object') return raw
    try { return JSON.parse(raw || '{}') } catch { return null }
  }
  async deleteGlossarySet(id) {
    const res = await DeleteGlossarySet(id)
    if (!res?.success) throw new Error(res?.msg || 'Delete glossary set failed')
    return true
  }

  // ==================== 目标语言（AI 翻译） ====================

  async listTargetLanguages() {
    const res = await ListTargetLanguages()
    if (!res?.success) throw new Error(res?.msg || 'List target languages failed')
    const raw = res.data
    const list = Array.isArray(raw) ? raw : (function () {
      try { return JSON.parse(raw || '[]') } catch { return [] }
    }())
    // 兜底：确保每一项都有 name（至少等于 code）
    const normalized = (list || []).map(it => {
      if (!it) return it
      if (!it.name && it.code) it.name = it.code
      return it
    })
    this.targetLanguages = normalized
    return normalized
  }

  async upsertTargetLanguage(lang) {
    const res = await UpsertTargetLanguage(lang)
    if (!res?.success) throw new Error(res?.msg || 'Upsert target language failed')
    const raw = res.data
    let out = null
    if (raw && typeof raw === 'object') out = raw
    else {
      try { out = JSON.parse(raw || '{}') } catch { out = null }
    }
    if (out && out.code) {
      if (!out.name) out.name = out.code
      // 更新本地缓存
      const idx = Array.isArray(this.targetLanguages)
        ? this.targetLanguages.findIndex(l => l && l.code === out.code)
        : -1
      if (idx >= 0) this.targetLanguages[idx] = out
      else {
        if (!Array.isArray(this.targetLanguages)) this.targetLanguages = []
        this.targetLanguages.push(out)
      }
    }
    return out
  }

  async deleteTargetLanguage(code) {
    const res = await DeleteTargetLanguage(code)
    if (!res?.success) throw new Error(res?.msg || 'Delete target language failed')
    try {
      if (Array.isArray(this.targetLanguages)) {
        this.targetLanguages = this.targetLanguages.filter(l => !l || l.code !== code)
      }
    } catch {}
    return true
  }

  async resetTargetLanguages() {
    const res = await ResetTargetLanguagesToDefault()
    if (!res?.success) throw new Error(res?.msg || 'Reset target languages failed')
    try { await this.listTargetLanguages() } catch {}
    return true
  }

  /**
   * 加载支持的转换器列表
   * @returns {Promise<string[]>} 转换器列表
   */
  async loadSupportedConverters() {
    try {
      const result = await GetSupportedConverters();
      if (result.success) {
        const raw = result.data
        this.supportedConverters = Array.isArray(raw) ? raw : (JSON.parse(raw || '[]'));
        return this.supportedConverters;
      } else {
        throw new Error(result.msg || 'Failed to get supported converters');
      }
    } catch (error) {
      console.error('Error loading supported converters:', error);
      throw error;
    }
  }

  /**
   * 获取支持的转换器列表
   * @returns {string[]} 转换器列表
   */
  getSupportedConverters() {
    return this.supportedConverters;
  }

  /**
   * 执行中文转换
   * @param {string} origin - 源语言代码
   * @param {string} converter - 转换器类型
   * @returns {Promise<Object>} 转换结果
   */
  async convertSubtitle(origin, converter) {
    if (!this.currentProject) {
      throw new Error('No project loaded');
    }

    if (!origin || !converter) {
      throw new Error('Origin and converter are required');
    }

    if (!this.supportedConverters.includes(converter)) {
      throw new Error(`Unsupported converter: ${converter}`);
    }

    try {
      const result = await ZHConvertSubtitle(this.currentProject.id, origin, converter);

      if (!result.success) {
        throw new Error(result.msg || 'Conversion failed');
      }

      // 触发转换开始事件
      this.handleConversionStart(origin, converter);

      return result;
    } catch (error) {
      console.error('Conversion error:', error);
      throw error;
    }
  }

  /**
   * 处理转换开始
   * @param {string} origin - 源语言
   * @param {string} converter - 转换器
   */
  handleConversionStart(origin, converter) {
    const event = {
      type: 'conversion_started',
      origin,
      converter,
      timestamp: Date.now()
    };

    this.conversionCallbacks.forEach(callback => {
      try {
        callback(event);
      } catch (error) {
        console.error('Conversion callback error:', error);
      }
    });
  }

  /**
   * 订阅转换事件
   * @param {Function} callback - 回调函数
   * @returns {Function} 取消订阅函数
   */
  onConversionEvent(callback) {
    this.conversionCallbacks.add(callback);
    return () => this.conversionCallbacks.delete(callback);
  }

  // ==================== 其他功能方法 ====================

  /**
   * 导出字幕文件
   * @param {string} projectId - 项目ID
   * @param {string} languageCode - 语言代码
   * @param {string} format - 导出格式
   * @returns {Promise<Object>} 导出结果
   */
  async exportSubtitles(projectId, languageCode, format, formatConfig) {
    if (!projectId || !languageCode || !format) {
      throw new Error('Project ID, language code and format are required');
    }

    try {
      // If caller provided a format-specific config, persist it first
      if (formatConfig && this.currentProject && this.currentProject.metadata) {
        const metadata = JSON.parse(JSON.stringify(this.currentProject.metadata));
        metadata.export_configs = metadata.export_configs || {};
        const prev = metadata.export_configs[format] || {};
        metadata.export_configs[format] = { ...prev, ...formatConfig };
        const saveRes = await UpdateProjectMetadata(projectId, metadata);
        if (!saveRes?.success) {
          throw new Error(saveRes?.msg || 'Failed to save export config');
        }
        // keep local state in sync if backend returned updated project
        try {
          if (saveRes.data) {
            const updatedProject = typeof saveRes.data === 'string' ? JSON.parse(saveRes.data) : saveRes.data;
            this.handleProjectUpdate(updatedProject);
          }
        } catch {}
      }

      // 调用后端导出API
      const result = await ExportSubtitleToFile(projectId, languageCode, format);
      const cancelled = !!result?.data && result.data.cancelled === true;

      if (cancelled) {
        return { success: false, cancelled: true };
      }

      if (!result.success) {
        throw new Error(result.msg);
      }

      return {
        success: true,
        cancelled: false,
        filePath: result.data?.filePath,
        fileName: result.data?.fileName
      };
    } catch (error) {
      console.error(error);
      throw error;
    }
  }

  /**
   * 删除当前项目的一种翻译语言（不可删除原始语言）
   */
  async deleteLanguage(languageCode) {
    if (!this.currentProject?.id) throw new Error('No project loaded')
    if (!languageCode) throw new Error('Language code required')
    const res = await RemoveProjectLanguage(this.currentProject.id, languageCode)
    if (!res?.success) throw new Error(res?.msg || 'Delete language failed')
    // 更新当前项目
    try {
      const updatedProject = typeof res.data === 'string' ? JSON.parse(res.data) : res.data
      this.handleProjectUpdate(updatedProject)
    } catch {}
    return true
  }

  // ==================== 状态管理方法 ====================

  /**
   * 订阅状态变化
   * @param {Function} callback - 回调函数
   * @returns {Function} 取消订阅函数
   */
  onStatusChange(callback) {
    this.statusCallbacks.add(callback);
    return () => this.statusCallbacks.delete(callback);
  }

  /**
   * 获取当前保存状态
   * @returns {string} 当前状态
   */
  getSaveStatus() {
    return this.saveStatus;
  }

  /**
   * 检查是否有待保存的更改
   * @returns {boolean} 是否有待保存的更改
   */
  hasPendingChanges() {
    return this.autoSaveManager?.hasPendingChanges() || false;
  }

  /**
   * 获取上次保存时间
   * @returns {number} 上次保存时间戳
   */
  getLastSaveTime() {
    return this.autoSaveManager?.getLastSaveTime() || 0;
  }

  /**
   * 获取当前项目
   * @returns {Object|null} 当前项目数据
   */
  getCurrentProject() {
    return this.currentProject;
  }

  /**
   * 强制刷新项目数据
   * @param {Object} project - 新的项目数据
   */
  updateCurrentProject(project) {
    if (project) {
      this.handleProjectUpdate(project);
    }
  }

  // ==================== WebSocket 事件处理 ====================

  /**
   * 初始化 WebSocket 监听器
   */
  initializeWebSocketListeners() {
    // 若已订阅则直接返回，避免重复注册
    if (this._wsSubscribed && this._subtitleProgressHandler) return

    this.dtStore = useDtStore()
    // 固定绑定引用，便于后续正确注销
    this._subtitleProgressHandler = this.handleSubtitleProgress.bind(this)
    // 注册字幕进度回调
    this.dtStore.registerSubtitleProgressCallback(this._subtitleProgressHandler)
    this._wsSubscribed = true
  }

  /**
   * 处理字幕进度更新
   * @param {Object} data - 进度数据
   */
  // 在现有的 handleSubtitleProgress 方法中
  handleSubtitleProgress(data) {
    // 检查是否为终态
    const statusRaw = String(data?.status || '').toLowerCase()
    const terminalStatuses = ['completed', 'failed', 'cancelled']
    const isTerminalStatus = terminalStatuses.includes(statusRaw)
    const failedCount = Number(data?.failed_segments ?? data?.failedSegments ?? 0)
    const isPartial = failedCount > 0 && statusRaw === 'completed'
    const hasError = statusRaw === 'failed' || statusRaw === 'cancelled' || isPartial

    // 触发转换进度事件
    const event = {
      type: 'conversion_progress',
      data,
      isTerminal: isTerminalStatus,
      timestamp: Date.now()
    }

    if (isTerminalStatus) {
      const base = isPartial
        ? (i18nGlobal.t('subtitle.add_language.conversion_partial', { count: failedCount }) || 'Conversion finished with partial failures')
        : i18nGlobal.t('subtitle.add_language.conversion_finished', { status: data.status })
      if (hasError) {
        $message.warning(base)
      } else {
        $message.info(base)
      }
    }

    this.conversionCallbacks.forEach(callback => {
      try {
        callback(event)
      } catch (error) {
        console.error('Progress callback error:', error)
      }
    })
  }

  // ==================== 生命周期方法 ====================

  /**
 * 销毁服务
 */
  destroy() {
    if (this.autoSaveManager) {
      this.autoSaveManager.destroy();
      this.autoSaveManager = null;
    }

    // 清理 WebSocket 监听器
    if (this.dtStore && this._subtitleProgressHandler && this._wsSubscribed) {
      try { this.dtStore.unregisterSubtitleProgressCallback(this._subtitleProgressHandler) } catch {}
    }
    this._subtitleProgressHandler = null
    this._wsSubscribed = false
    this.dtStore = null

    // 清理回调
    this.statusCallbacks.clear();
    this.projectUpdateCallbacks.clear();
    this.conversionCallbacks.clear();

    // 重置状态
    this.currentProject = null;
    this.saveStatus = 'idle';
    this.supportedConverters = [];
  }

  /**
   * 重新初始化服务
   * @param {Object} project - 项目数据
   */
  reinitialize(project) {
    this.destroy();
    this.initialize(project);
  }
}

// 创建全局实例
export const subtitleService = new SubtitleService();
