<template>
  <div v-if="show" class="modal-overlay">
    <div class="modal-container">
      <!-- Modal头部 -->
      <div class="modal-header">
        <div class="header-content">
          <div class="header-icon">
            <v-icon name="fa-file-import"></v-icon>
          </div>
          <h3 class="modal-title">{{ $t('subtitle.import.title') }}</h3>
        </div>
        <button @click="$emit('close')" class="close-button">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <!-- Modal内容 -->
      <div class="modal-content">
        <!-- 文件信息卡片 -->
        <div class="file-info-card">
          <div class="card-header">
            <div class="file-icon">
              <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                  d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
              </svg>
            </div>
            <span class="card-title">{{ $t('subtitle.import.selected_file') }}</span>
          </div>
          <div class="file-path">{{ filePath }}</div>
        </div>

        <!-- 验证标准选择 - 最高优先级 -->
        <div class="priority-section">
          <h4 class="section-title">{{ $t('subtitle.import.guideline_standard') }}</h4>
          <div class="standard-options">
            <label class="standard-option" :class="{ 'active': options.guideline_standard === 'netflix' }">
              <input type="radio" v-model="options.guideline_standard" value="netflix" class="standard-radio">
              <div class="standard-content">
                <div class="standard-name">Netflix</div>
                <div class="standard-badge netflix">{{ $t('subtitle.import.guideline_recommendation') }}</div>
              </div>
            </label>
            
            <label class="standard-option" :class="{ 'active': options.guideline_standard === 'bbc' }">
              <input type="radio" v-model="options.guideline_standard" value="bbc" class="standard-radio">
              <div class="standard-content">
                <div class="standard-name">BBC</div>
                <div class="standard-badge bbc">{{ $t('subtitle.import.guideline_professional') }}</div>
              </div>
            </label>
            
            <label class="standard-option" :class="{ 'active': options.guideline_standard === 'ade' }">
              <input type="radio" v-model="options.guideline_standard" value="ade" class="standard-radio">
              <div class="standard-content">
                <div class="standard-name">ADE</div>
                <div class="standard-badge ade">{{ $t('subtitle.import.guideline_general') }}</div>
              </div>
            </label>
          </div>
        </div>

        <!-- 文本处理选项 -->
        <div class="options-section">
          <h4 class="section-title">{{ $t('subtitle.import.text_processing') }}</h4>
          <div class="compact-options">
            <label class="compact-option">
              <input type="checkbox" v-model="options.remove_empty_lines" class="option-checkbox">
              <span class="option-label">{{ $t('subtitle.import.remove_empty_lines') }}</span>
            </label>
            
            <label class="compact-option">
              <input type="checkbox" v-model="options.trim_whitespace" class="option-checkbox">
              <span class="option-label">{{ $t('subtitle.import.trim_whitespace') }}</span>
            </label>
            
            <label class="compact-option">
              <input type="checkbox" v-model="options.normalize_line_breaks" class="option-checkbox">
              <span class="option-label">{{ $t('subtitle.import.normalize_line_breaks') }}</span>
            </label>
            
            <label class="compact-option">
              <input type="checkbox" v-model="options.fix_encoding" class="option-checkbox">
              <span class="option-label">{{ $t('subtitle.import.fix_encoding') }}</span>
            </label>

            <label class="compact-option">
              <input type="checkbox" v-model="options.fix_common_errors" class="option-checkbox">
              <span class="option-label">{{ $t('subtitle.import.fix_common_errors') }}</span>
            </label>
          </div>
        </div>
      </div>

      <!-- Modal底部 -->
      <div class="modal-footer">
        <button @click="$emit('close')" class="btn-macos-secondary btn-macos-sm">
          {{ $t('common.cancel') }}
        </button>
        <button @click="handleImport" class="btn-macos-primary btn-macos-sm" :disabled="importing">
          <div v-if="importing" class="loading-spinner"></div>
          {{ importing ? $t('subtitle.import.importing') : $t('subtitle.import.start_import') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue'

const props = defineProps({
  show: Boolean,
  filePath: String
})

const emit = defineEmits(['close', 'import'])

const importing = ref(false)

// 导入选项
const options = ref({
  validate_guidelines: true, // 默认验证
  is_kids_content: false, // 默认成人
  guideline_standard: 'netflix', // 默认选择Netflix标准
  remove_empty_lines: true,
  trim_whitespace: true,
  normalize_line_breaks: true,
  fix_encoding: true,
  fix_common_errors: true
})

const handleImport = () => {
  emit('import', {
    filePath: props.filePath,
    options: options.value
  })
}
</script>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(8px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 50;
  padding: 1rem;
  animation: fadeIn 0.2s ease-out;
}

.modal-container {
  background: var(--macos-background);
  border-radius: 12px;
  box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1), 0 10px 10px -5px rgba(0, 0, 0, 0.04);
  max-width: 600px;
  width: 100%;
  max-height: 85vh;
  overflow: hidden;
  border: 1px solid var(--macos-separator);
  animation: slideInUp 0.3s ease-out;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
}

.header-content {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: var(--macos-gray-hover);
  border-radius: 8px;
  color: white;
}

.modal-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0;
}

.close-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  color: var(--macos-text-secondary);
  cursor: pointer;
  transition: all 0.15s ease;
}

.close-button:hover {
  background: var(--macos-gray-hover);
  color: var(--macos-text-primary);
}

.modal-content {
  padding: 20px;
  max-height: calc(85vh - 140px);
  overflow-y: auto;
}

.file-info-card {
  background: var(--macos-background-secondary);
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  margin-bottom: 24px;
  overflow: hidden;
}

.card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 16px;
  background: var(--macos-background-tertiary);
  border-bottom: 1px solid var(--macos-separator);
}

.file-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.card-title {
  font-size: 13px;
  font-weight: 500;
  color: var(--macos-text-primary);
}

.file-path {
  padding: 12px 16px;
  font-size: 12px;
  color: var(--macos-text-secondary);
  font-family: 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace;
  word-break: break-all;
  background: var(--macos-background);
}

/* 优先级最高的验证标准选择 */
.priority-section {
  margin-bottom: 24px;
  padding: 16px;
  background: linear-gradient(135deg, rgba(var(--macos-blue-rgb), 0.05) 0%, rgba(var(--macos-blue-rgb), 0.02) 100%);
  border: 1px solid rgba(var(--macos-blue-rgb), 0.1);
  border-radius: 10px;
}

.section-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 16px 0;
}

.standard-options {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}

.standard-option {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 16px 12px;
  background: var(--macos-background);
  border: 2px solid var(--macos-separator);
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.standard-option:hover {
  border-color: var(--macos-blue);
  background: var(--macos-background-secondary);
}

.standard-option.active {
  border-color: var(--macos-blue);
  background: rgba(var(--macos-blue-rgb), 0.08);
  box-shadow: 0 0 0 1px rgba(var(--macos-blue-rgb), 0.2);
}

.standard-radio {
  position: absolute;
  opacity: 0;
  pointer-events: none;
}

.standard-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}

.standard-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--macos-text-primary);
}

.standard-badge {
  font-size: 11px;
  font-weight: 500;
  padding: 3px 8px;
  border-radius: 12px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.standard-badge.netflix {
  background: rgba(229, 9, 20, 0.1);
  color: #e50914;
}

.standard-badge.bbc {
  background: rgba(0, 51, 102, 0.1);
  color: #003366;
}

.standard-badge.ade {
  background: rgba(34, 197, 94, 0.1);
  color: #22c55e;
}

/* 紧凑的选项布局 */
.options-section {
  margin-bottom: 20px;
}

.compact-options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 8px;
}

.compact-option {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: var(--macos-background-secondary);
  border: 1px solid var(--macos-separator);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
}

.compact-option:hover {
  background: var(--macos-background-tertiary);
  border-color: var(--macos-blue);
}

.option-checkbox {
  width: 16px;
  height: 16px;
  accent-color: var(--macos-blue);
  margin: 0;
}

.option-label {
  font-size: 13px;
  font-weight: 400;
  color: var(--macos-text-primary);
  line-height: 1.3;
}

.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  background: var(--macos-background-secondary);
  border-top: 1px solid var(--macos-separator);
}

.btn-macos-sm {
  padding: 6px 16px;
  font-size: 13px;
  min-height: 32px;
}

.loading-spinner {
  width: 14px;
  height: 14px;
  border: 2px solid transparent;
  border-top: 2px solid currentColor;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-right: 8px;
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(20px) scale(0.95);
  }
  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

/* 滚动条样式 */
.modal-content::-webkit-scrollbar {
  width: 6px;
}

.modal-content::-webkit-scrollbar-track {
  background: transparent;
}

.modal-content::-webkit-scrollbar-thumb {
  background: var(--macos-scrollbar-thumb);
  border-radius: 3px;
}

.modal-content::-webkit-scrollbar-thumb:hover {
  background: var(--macos-scrollbar-thumb-hover);
}

/* 响应式设计 */
@media (max-width: 640px) {
  .standard-options {
    grid-template-columns: 1fr;
  }
  
  .compact-options {
    grid-template-columns: 1fr;
  }
}
</style>