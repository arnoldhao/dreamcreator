/**
 * 自动保存管理器
 * 提供防抖保存和立即保存功能
 */
export class AutoSaveManager {
  constructor(options = {}) {
    this.debounceTime = options.debounceTime || 1000; // 默认1秒防抖
    this.maxWaitTime = options.maxWaitTime || 5000; // 最大等待时间5秒
    this.onSave = options.onSave || (() => {});
    this.onError = options.onError || console.error;
    this.onStatusChange = options.onStatusChange || (() => {});
    
    this.debounceTimer = null;
    this.maxWaitTimer = null;
    this.pendingData = null;
    this.isSaving = false;
    this.lastSaveTime = 0;
  }

  /**
   * 触发保存（防抖）
   * @param {Object} data - 要保存的数据
   * @param {string} type - 保存类型
   */
  save(data, type) {
    this.pendingData = { data, type };
    
    // 清除之前的防抖定时器
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
    }
    
    // 设置防抖定时器
    this.debounceTimer = setTimeout(() => {
      this.executeSave();
    }, this.debounceTime);
    
    // 设置最大等待定时器（防止一直防抖不保存）
    if (!this.maxWaitTimer) {
      this.maxWaitTimer = setTimeout(() => {
        this.executeSave();
      }, this.maxWaitTime);
    }
    
    this.updateStatus('pending');
  }

  /**
   * 立即保存
   * @param {Object} data - 要保存的数据
   * @param {string} type - 保存类型
   * @returns {Promise<Object>} 保存结果
   */
  async saveNow(data, type) {
    // 清除所有定时器
    this.clearTimers();
    
    this.pendingData = { data, type };
    return await this.executeSave();
  }

  /**
   * 执行保存操作
   * @returns {Promise<Object>} 保存结果
   */
  async executeSave() {
    if (this.isSaving || !this.pendingData) {
      return;
    }
  
    this.isSaving = true;
    this.updateStatus('saving');
    this.clearTimers();
  
    try {
      const { data, type } = this.pendingData;
      const result = await this.onSave(data, type);
      
      this.lastSaveTime = Date.now();
      this.pendingData = null;
      this.updateStatus('saved');
      
      // 3秒后清除保存状态
      setTimeout(() => {
        if (!this.isSaving && !this.pendingData) {
          this.updateStatus('idle');
        }
      }, 3000);
      
      return result; // 返回结果给调用方
    } catch (error) {
      this.updateStatus('error');
      
      // 调用错误回调（用于日志记录等）
      this.onError(error);
      
      // 5秒后清除错误状态
      setTimeout(() => {
        if (!this.isSaving) {
          this.updateStatus('idle');
        }
      }, 5000);
      
      throw error; // 重新抛出错误，让上层处理
    } finally {
      this.isSaving = false;
    }
  }

  /**
   * 清除所有定时器
   */
  clearTimers() {
    if (this.debounceTimer) {
      clearTimeout(this.debounceTimer);
      this.debounceTimer = null;
    }
    if (this.maxWaitTimer) {
      clearTimeout(this.maxWaitTimer);
      this.maxWaitTimer = null;
    }
  }

  /**
   * 更新保存状态
   * @param {string} status - 状态：idle, pending, saving, saved, error
   */
  updateStatus(status) {
    this.onStatusChange(status);
  }

  /**
   * 销毁管理器
   */
  destroy() {
    this.clearTimers();
    this.pendingData = null;
    this.isSaving = false;
  }

  /**
   * 获取上次保存时间
   */
  getLastSaveTime() {
    return this.lastSaveTime;
  }

  /**
   * 检查是否有待保存的数据
   */
  hasPendingChanges() {
    return this.pendingData !== null || this.debounceTimer !== null;
  }

  /**
   * 获取当前状态
   */
  getStatus() {
    if (this.isSaving) return 'saving';
    if (this.pendingData) return 'pending';
    return 'idle';
  }
}

/**
 * 创建自动保存管理器的工厂函数
 * @param {Object} options - 配置选项
 * @returns {AutoSaveManager}
 */
export function createAutoSaveManager(options) {
  return new AutoSaveManager(options);
}