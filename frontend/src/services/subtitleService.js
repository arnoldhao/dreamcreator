import { UpdateProjectName, ExportSubtitleToFile, UpdateProjectMetadata, UpdateSubtitleSegment, UpdateLanguageContent, UpdateLanguageMetadata } from 'wailsjs/go/api/SubtitlesAPI';
import { createAutoSaveManager } from '@/utils/autoSave.js';

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
          const updatedProject = JSON.parse(result.data);
          // 可以触发项目更新事件
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

  // ==================== 其他功能方法 ====================

  /**
   * 导出字幕文件
   * @param {string} projectId - 项目ID
   * @param {string} languageCode - 语言代码
   * @param {string} format - 导出格式
   * @returns {Promise<Object>} 导出结果
   */
  async exportSubtitles(projectId, languageCode, format) {
    if (!projectId || !languageCode || !format) {
      throw new Error('Project ID, language code and format are required');
    }

    try {
      // 调用后端导出API
      const result = await ExportSubtitleToFile(projectId, languageCode, format);

      if (!result.success) {
        throw new Error(result.msg);
      }

      return {
        success: true,
        filePath: result.data?.filePath,
        fileName: result.data?.fileName
      };
    } catch (error) {
      console.error(error);
      throw error;
    }
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

  // ==================== 生命周期方法 ====================

  /**
   * 销毁服务
   */
  destroy() {
    if (this.autoSaveManager) {
      this.autoSaveManager.destroy();
      this.autoSaveManager = null;
    }
    this.statusCallbacks.clear();
    this.projectUpdateCallbacks.clear();
    this.currentProject = null;
    this.saveStatus = 'idle';
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