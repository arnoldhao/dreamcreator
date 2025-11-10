<template>
  <div v-if="show" class="macos-modal">
    <div class="modal-card card-frosted card-translucent" @keydown.esc.stop.prevent="emit('close')" tabindex="-1">
      <!-- Header: traffic lights left, title on right -->
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title-area">
          <div v-if="filePath" class="title-chips">
            <div class="chip-frosted chip-sm chip-translucent file-chip" :title="filePath">
              <Icon name="file-text" class="w-3 h-3" />
              <span class="text mono">{{ filePath }}</span>
              <button class="chip-action" @click.stop="emit('reselect')" :aria-label="$t('common.edit')">
                <Icon name="edit" class="w-3 h-3" />
              </button>
            </div>
          </div>
        </div>
      </div>

      <!-- Body -->
      <div class="modal-body">
        <!-- 已选文件移至标题右上角的胶囊展示 -->

        <!-- 验证标准选择 - 最高优先级 -->
        <div class="priority-section">
          <h4 class="section-title">{{ $t('subtitle.import.guideline_standard') }}</h4>
          <div class="standard-options">
            <label class="standard-option netflix" :class="{ 'active': options.guideline_standard === 'netflix' }">
              <input type="radio" v-model="options.guideline_standard" value="netflix" class="standard-radio">
              <div class="standard-content">
                <div class="standard-name">Netflix</div>
                <div class="standard-badge netflix">{{ $t('subtitle.import.guideline_recommendation') }}</div>
              </div>
            </label>

            <label class="standard-option bbc" :class="{ 'active': options.guideline_standard === 'bbc' }">
              <input type="radio" v-model="options.guideline_standard" value="bbc" class="standard-radio">
              <div class="standard-content">
                <div class="standard-name">BBC</div>
                <div class="standard-badge bbc">{{ $t('subtitle.import.guideline_professional') }}</div>
              </div>
            </label>

            <label class="standard-option ade" :class="{ 'active': options.guideline_standard === 'ade' }">
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
        <!-- Inline centered actions below text processing -->
        <div class="actions-center">
          <button @click="emit('close')" class="btn-chip">
            <Icon name="close" class="w-4 h-4 mr-1" />
            {{ $t('common.cancel') }}
          </button>
          <button @click="handleImport" class="btn-chip btn-primary" :disabled="importing">
            <template v-if="importing">
              <div class="loading-spinner"></div>
            </template>
            <template v-else>
              <Icon name="file-plus" class="w-4 h-4 mr-1" />
            </template>
            {{ importing ? $t('subtitle.import.importing') : $t('subtitle.import.start_import') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, defineProps, defineEmits } from 'vue'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'

const props = defineProps({
  show: Boolean,
  filePath: String
})

const emit = defineEmits(['close', 'import', 'reselect'])

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
/* Use shared .macos-modal convention for overlay */
.macos-modal { animation: fadeIn 0.2s ease-out; }

.modal-card {
  background: var(--macos-surface);
  backdrop-filter: var(--macos-surface-blur);
  border-radius: 12px;
  box-shadow: var(--macos-shadow-2);
  max-width: 640px;
  width: 100%;
  max-height: 85vh;
  overflow: hidden;
  border: 1px solid rgba(255,255,255,0.22);
  animation: slideInUp 0.3s ease-out;
}

.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: var(--macos-surface);
  backdrop-filter: var(--macos-surface-blur);
  border-bottom: 1px solid var(--macos-separator);
}

/* no traffic lights for sheet-like modal */

.title-area {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  min-width: 0;
}

.title-chips { display: flex; align-items: center; gap: 6px; min-width: 0; }

.chip-frosted .text {
  max-width: 260px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.chip-frosted .chip-action {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 4px;
  border: none;
  background: transparent;
  color: rgba(255,255,255,0.8);
}

.chip-frosted .chip-action:hover { background: var(--macos-blue); color: #fff; }

.chip-frosted .mono {
  font-family: var(--font-mono);
}

.file-chip .w-3,
.file-chip .h-3 {
  display: block;
}

.modal-body {
  padding: 16px;
  padding-bottom: 12px;
  max-height: calc(85vh - 120px);
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
  font-size: var(--fs-base);
  font-weight: 500;
  color: var(--macos-text-primary);
}

.file-path {
  padding: 12px 16px;
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
  font-family: var(--font-mono);
  word-break: break-all;
  background: var(--macos-background);
}

/* 优先级最高的验证标准选择 */
.priority-section {
  margin-bottom: 12px;
  padding: 12px;
  background: var(--macos-surface);
  backdrop-filter: var(--macos-surface-blur);
  border: 1px solid rgba(255,255,255,0.22);
  border-radius: 10px;
}

.section-title {
  font-size: var(--fs-sub);
  font-weight: 600;
  color: var(--macos-text-secondary);
  margin: 0 0 8px 0;
}

.standard-options {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.standard-option {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 10px 8px;
  background: var(--macos-surface);
  backdrop-filter: var(--macos-surface-blur);
  border: 1px solid rgba(255,255,255,0.22);
  border-radius: 8px;
  cursor: pointer;
  transition: background .15s ease, border-color .15s ease;
}

.standard-option:hover {
  border-color: rgba(255,255,255,0.28);
}

.standard-option.active {
  border-color: rgba(255,255,255,0.3);
  box-shadow: 0 0 0 1.5px rgba(255,255,255,0.2);
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
  gap: 6px;
}

.standard-name {
  font-size: var(--fs-base);
  font-weight: 600;
  color: var(--macos-text-primary);
}

.standard-badge {
  font-size: var(--fs-micro);
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 999px;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  border: 1px solid rgba(255,255,255,0.22);
  background: rgba(30,30,30,0.40);
  -webkit-backdrop-filter: saturate(150%) blur(10px);
  backdrop-filter: saturate(150%) blur(10px);
  color: #fff;
  box-shadow: 0 1px 0 rgba(255,255,255,0.14) inset, 0 4px 12px rgba(0,0,0,0.24), 0 1px 3px rgba(0,0,0,0.18);
}

.standard-badge.netflix { border-color: var(--brand-netflix); }

.standard-badge.bbc { border-color: var(--brand-bbc); }

.standard-badge.ade { border-color: var(--brand-ade); }

/* Improve contrast by tinting option background subtly per standard */
.standard-option.netflix { background: color-mix(in oklab, #e50914 7%, var(--macos-surface)); border-color: color-mix(in oklab, #e50914 28%, rgba(255,255,255,0.22)); }
.standard-option.netflix:hover { background: color-mix(in oklab, #e50914 10%, var(--macos-surface)); }
.standard-option.netflix.active { box-shadow: 0 0 0 2px color-mix(in oklab, #e50914 36%, transparent); border-color: color-mix(in oklab, #e50914 42%, rgba(255,255,255,0.22)); }

.standard-option.bbc { background: color-mix(in oklab, #003366 7%, var(--macos-surface)); border-color: color-mix(in oklab, #003366 28%, rgba(255,255,255,0.22)); }
.standard-option.bbc:hover { background: color-mix(in oklab, #003366 10%, var(--macos-surface)); }
.standard-option.bbc.active { box-shadow: 0 0 0 2px color-mix(in oklab, #003366 36%, transparent); border-color: color-mix(in oklab, #003366 42%, rgba(255,255,255,0.22)); }

.standard-option.ade { background: color-mix(in oklab, #22c55e 7%, var(--macos-surface)); border-color: color-mix(in oklab, #22c55e 28%, rgba(255,255,255,0.22)); }
.standard-option.ade:hover { background: color-mix(in oklab, #22c55e 10%, var(--macos-surface)); }
.standard-option.ade.active { box-shadow: 0 0 0 2px color-mix(in oklab, #22c55e 36%, transparent); border-color: color-mix(in oklab, #22c55e 42%, rgba(255,255,255,0.22)); }

/* Tinted badges to reinforce identity while staying readable */
.standard-badge.netflix { background: color-mix(in oklab, var(--brand-netflix) 24%, rgba(30,30,30,0.35)); }
.standard-badge.bbc { background: color-mix(in oklab, var(--brand-bbc) 22%, rgba(30,30,30,0.35)); }
.standard-badge.ade { background: color-mix(in oklab, var(--brand-ade) 26%, rgba(30,30,30,0.35)); }

/* 紧凑的选项布局 */
.options-section {
  margin-bottom: 8px;
  padding: 12px;
  background: var(--macos-surface);
  backdrop-filter: var(--macos-surface-blur);
  border: 1px solid rgba(255,255,255,0.22);
  border-radius: 10px;
}

.actions-center {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  margin-top: 16px;
}

.compact-options {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 8px;
}

.compact-option {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  background: var(--macos-background-secondary);
  border: 1px solid var(--macos-separator);
  border-radius: 6px;
  cursor: pointer;
  transition: background .15s ease, border-color .15s ease;
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
  font-size: var(--fs-base);
  font-weight: 400;
  color: var(--macos-text-primary);
  line-height: 1.3;
}

.options-actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 10px 0 16px;
  border-top: 1px solid var(--macos-separator);
  background: var(--macos-background-secondary);
}

/* removed unused .btn-macos-sm */

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

/* 滚动条样式（作用于滚动的 body） */
.modal-body::-webkit-scrollbar {
  width: 6px;
}

.modal-body::-webkit-scrollbar-track {
  background: transparent;
}

.modal-body::-webkit-scrollbar-thumb {
  background: var(--macos-scrollbar-thumb);
  border-radius: 3px;
}

.modal-body::-webkit-scrollbar-thumb:hover {
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
