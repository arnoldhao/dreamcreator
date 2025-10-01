<template>
  <div v-if="show" class="macos-modal" @click.self="emit('close')">
    <div class="modal-card" @keydown.esc.stop.prevent="emit('close')" tabindex="-1">
      <!-- Header: macOS traffic lights + segmented mode switch -->
      <div class="modal-header">
        <ModalTrafficLights @close="emit('close')" />
        <div class="title-area">
          <div class="mode-toggle" role="tablist" aria-label="Mode">
            <button role="tab" :aria-selected="activeTab==='zhconvert'" :class="['seg-item', { active: activeTab==='zhconvert' }]" @click="activeTab='zhconvert'">{{ $t('subtitle.add_language.zhconvert') }}</button>
            <button role="tab" :aria-selected="activeTab==='llm'" :class="['seg-item', { active: activeTab==='llm', disabled: true }]" @click="activeTab='llm'" disabled>{{ $t('subtitle.add_language.llm') }}</button>
          </div>
        </div>
      </div>

      <!-- 标签页导航改为顶部模式切换，移除旧导航 -->

      <!-- Modal内容 -->
      <div class="modal-body">
        <!-- ZHConvert 标签页 -->
        <div v-if="activeTab === 'zhconvert'" class="tab-content">
          <!-- 源语言选择 -->
          <div class="form-section">
            <h4 class="section-title">{{ $t('subtitle.add_language.source_language') }}</h4>
            <select v-model="selectedSourceLanguage" class="select-macos select-fixed">
              <option value="">{{ $t('subtitle.add_language.select_source_language') }}</option>
              <option v-for="lang in availableSourceLanguages" :key="lang.code" :value="lang.code">
                {{ lang.name }}
              </option>
            </select>
          </div>

          <!-- 转换器选择 -->
          <div class="form-section">
            <h4 class="section-title">{{ $t('subtitle.add_language.converter_type') }}</h4>
            <div v-if="loading" class="loading-state">
              <div class="loading-spinner"></div>
              <span>{{ $t('subtitle.add_language.loading_converters') }}</span>
            </div>
            <div v-else-if="converters.length === 0" class="empty-state">
              <svg class="w-8 h-8 text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                  d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
              </svg>
              <p class="text-sm text-gray-500">{{ $t('subtitle.add_language.no_converters_available') }}</p>
            </div>
            <div v-else>
              <select v-model="selectedConverter" class="select-macos select-fixed">
                <option value="">{{ $t('subtitle.add_language.select_converter') }}</option>
                <option v-for="converter in converters" :key="converter" :value="converter">
                  {{ getConverterDisplayName(converter) }}
                </option>
              </select>

              <!-- 显示转换器描述 -->
              <div v-if="selectedConverter" class="converter-description-box">
                <div class="description-icon">
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path>
                  </svg>
                </div>
                <span class="description-text">{{ getConverterDescription(selectedConverter) }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- LLM 标签页 -->
        <div v-else-if="activeTab === 'llm'" class="tab-content">
          <div class="coming-soon-content">
            <div class="coming-soon-icon">
              <svg class="w-16 h-16 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                  d="M9.663 17h4.673M12 3v1m6.364 1.636l-.707.707M21 12h-1M4 12H3m3.343-5.657l-.707-.707m2.828 9.9a5 5 0 117.072 0l-.548.547A3.374 3.374 0 0014 18.469V19a2 2 0 11-4 0v-.531c0-.895-.356-1.754-.988-2.386l-.548-.547z">
                </path>
              </svg>
            </div>
            <h4 class="coming-soon-title">{{ $t('subtitle.add_language.llm_translation') }}</h4>
            <p class="coming-soon-description">{{ $t('subtitle.add_language.llm_description') }}</p>
          </div>
        </div>
        <!-- Inline centered actions below content -->
        <div class="actions-center">
          <button @click="emit('close')" class="btn-glass">{{ $t('common.cancel') }}</button>
          <button @click="handleConvert" class="btn-glass btn-primary" :disabled="!canConvert">
            <div v-if="converting" class="loading-spinner"></div>
            {{ converting ? $t('subtitle.add_language.converting') : $t('subtitle.add_language.start_convert') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, defineProps, defineEmits, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'

const props = defineProps({
  show: Boolean,
  availableLanguages: {
    type: Array,
    default: () => []
  },
  subtitleService: {
    type: Object,
    required: true
  }
})

const emit = defineEmits(['close', 'convert-started'])

const { t } = useI18n()

// 响应式数据
const activeTab = ref('zhconvert')
const selectedSourceLanguage = ref('')
const selectedConverter = ref('')
const converters = ref([])
const loading = ref(false)
const converting = ref(false)

// 计算属性
const availableSourceLanguages = computed(() => {
  return props.availableLanguages.map(lang => ({
    code: lang,
    name: getLanguageDisplayName(lang)
  }))
})

const canConvert = computed(() => {
  return activeTab.value === 'zhconvert' &&
    selectedSourceLanguage.value &&
    selectedConverter.value &&
    !converting.value
})

// 方法
const loadConverters = async () => {
  try {
    loading.value = true
    const supportedConverters = await props.subtitleService.loadSupportedConverters()
    converters.value = supportedConverters || []
  } catch (error) {
    $message.error(t('subtitle.add_language.load_converters_failed'))
    converters.value = []
  } finally {
    loading.value = false
  }
}

const handleConvert = async () => {
  if (!canConvert.value) return

  try {
    converting.value = true

    // 调用转换服务
    await props.subtitleService.convertSubtitle(
      selectedSourceLanguage.value,
      selectedConverter.value
    )

    // 发出转换开始事件
    emit('convert-started', {
      sourceLanguage: selectedSourceLanguage.value,
      converter: selectedConverter.value
    })

    // 显示成功消息
    $message.success(t('subtitle.add_language.conversion_started'))

    // 关闭模态框
    emit('close')

  } catch (error) {
    console.error('Conversion failed:', error)
    $message.error(error.message || t('subtitle.add_language.conversion_failed'))
  } finally {
    converting.value = false
  }
}

const getLanguageDisplayName = (langCode) => {
  const languageNames = {
    'zh-CN': '简体中文',
    'zh-TW': '繁體中文',
    'zh-HK': '繁體中文（香港）',
    'en': 'English',
    'ja': '日本語',
    'ko': '한국어'
  }
  return languageNames[langCode] || langCode
}

const getConverterDisplayName = (converter) => {
  const converterNames = {
    'Simplified': '简体中文',
    'Traditional': '繁体中文',
    'China': '中国大陆简体',
    'Hongkong': '香港繁体',
    'Taiwan': '台湾繁体',
    'Pinyin': '拼音',
    'Bopomofo': '注音符号',
    'Mars': '火星文',
    'WikiSimplified': '维基简体',
    'WikiTraditional': '维基繁体'
  }
  return converterNames[converter] || converter
}

const getConverterDescription = (converter) => {
  const descriptions = {
    'Simplified': '转换为简体中文',
    'Traditional': '转换为繁体中文',
    'China': '转换为中国大陆简体中文',
    'Hongkong': '转换为香港繁体中文',
    'Taiwan': '转换为台湾繁体中文',
    'Pinyin': '转换为拼音',
    'Bopomofo': '转换为注音符号',
    'Mars': '转换为火星文',
    'WikiSimplified': '转换为维基百科简体中文',
    'WikiTraditional': '转换为维基百科繁体中文'
  }
  return descriptions[converter] || ''
}

// 生命周期
onMounted(() => {
  if (props.show) {
    loadConverters()
  }
})

// 监听 show 属性变化
watch(() => props.show, (newValue) => {
  if (newValue) {
    // 重置表单
    selectedSourceLanguage.value = ''
    selectedConverter.value = ''
    activeTab.value = 'zhconvert'

    // 加载转换器
    loadConverters()
  }
})
 </script>

<style scoped>
/* use global .macos-modal */

.modal-card { background: var(--macos-background); border-radius: 12px; box-shadow: 0 20px 25px -5px rgba(0,0,0,0.1), 0 10px 10px -5px rgba(0,0,0,0.04); max-width: 640px; width: 100%; max-height: 85vh; overflow: hidden; border: 1px solid var(--macos-separator); animation: slideInUp 0.3s ease-out; }
.modal-header { display:flex; align-items:center; justify-content: space-between; height: 36px; padding: 0 12px; background: var(--macos-background-secondary); border-bottom: 1px solid var(--macos-separator); }
.title-area { flex:1; min-width: 0; display:flex; align-items:center; justify-content:flex-end; }
.modal-body { padding: 16px; padding-bottom: 12px; max-height: calc(85vh - 120px); overflow-y: auto; }
.actions-center { display:flex; align-items:center; justify-content:center; gap:10px; margin-top: 16px; }

/* Mode toggle (header center) */
.mode-toggle { display:inline-flex; align-items:center; height: 26px; background: var(--macos-background); border: 1px solid var(--macos-separator); border-radius: 8px; padding: 2px; gap: 2px; }
.mode-toggle .seg-item { min-width: 80px; height: 24px; padding: 0 10px; border-radius: 6px; font-size: var(--fs-sub); color: var(--macos-text-secondary); background: transparent; border: none; cursor: pointer; display: inline-flex; align-items: center; justify-content: center; line-height: 1; }
.mode-toggle .seg-item:hover { background: color-mix(in oklab, var(--macos-blue) 16%, transparent); color: #fff; }
.mode-toggle .seg-item.active { background: var(--macos-blue); color: #fff; }
.mode-toggle .seg-item.disabled { opacity: .6; cursor: not-allowed; }

/* Compact selects */
/* Fixed select width to align visuals with Export inspector */
.select-fixed { width: 250px; }
/* Ensure fixed width wins over local .select-macos { width:100% } */
.select-macos.select-fixed { width: 250; max-width: 250px; }
/* Boxed sections for consistent macOS-like grouping */
.form-section { border: 1px solid var(--macos-separator); border-radius: 10px; padding: 12px; background: var(--macos-background); }
.form-section + .form-section { margin-top: 10px; }

.tab-navigation {
  display: flex;
  background: var(--macos-background-secondary);
  border-bottom: 1px solid var(--macos-separator);
}

.tab-button {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 12px 16px;
  border: none;
  background: transparent;
  color: var(--macos-text-secondary);
  font-size: var(--fs-title);
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  position: relative;
}

.tab-button:hover:not(:disabled) {
  background: var(--macos-gray-hover);
  color: var(--macos-text-primary);
}

.tab-button.active {
  color: var(--macos-blue);
  background: var(--macos-background);
}

.tab-button.active::after {
  content: '';
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--macos-blue);
}

.tab-button:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.tab-icon {
  display: flex;
  align-items: center;
  justify-content: center;
}

.coming-soon-badge {
  font-size: var(--fs-micro);
  padding: 2px 6px;
  background: var(--macos-orange);
  color: white;
  border-radius: 10px;
  margin-left: 4px;
}

.modal-content {
  padding: 20px;
  max-height: 400px;
  overflow-y: auto;
}

.tab-content {
    min-height: 200px;
  display: block;
  visibility: visible;
}

.form-section {
  margin-bottom: 24px;
}

.section-title {
  font-size: var(--fs-title);
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 12px 0;
}

.select-macos {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid var(--macos-separator);
  border-radius: 6px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: var(--fs-title);
  transition: border-color 0.2s ease;
}

.select-macos:focus {
  outline: none;
  border-color: var(--macos-blue);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--macos-blue) 15%, transparent);
}

.loading-state {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px;
  text-align: center;
  color: var(--macos-text-secondary);
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 40px 20px;
  text-align: center;
}

.converter-options {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.converter-option {
  display: flex;
  align-items: center;
  padding: 12px;
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.converter-option:hover {
  background: var(--macos-gray-hover);
  border-color: var(--macos-blue);
}

.converter-option.active {
  background: color-mix(in oklab, var(--macos-blue) 10%, var(--macos-background));
  border-color: var(--macos-blue);
}

.converter-radio {
  margin-right: 12px;
}

.converter-content {
  flex: 1;
}

.converter-name {
  font-weight: 500;
  color: var(--macos-text-primary);
  margin-bottom: 4px;
}

.converter-description {
  font-size: var(--fs-sub);
  color: var(--macos-text-secondary);
}


.converter-description-box {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  padding: 12px;
  background: color-mix(in oklab, var(--macos-blue) 10%, var(--macos-background));
  border: 1px solid color-mix(in oklab, var(--macos-blue) 20%, transparent);
  border-radius: 6px;
  font-size: var(--fs-base);
  color: color-mix(in oklab, var(--macos-blue) 80%, var(--macos-text-secondary));
}

.description-icon {
  display: flex;
  align-items: center;
  color: var(--macos-blue);
}

.description-text {
  flex: 1;
}

/* 删除了固定蓝色回退，统一使用变量以支持自定义主题色 */

.coming-soon-content {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  text-align: center;
}

.coming-soon-icon {
  margin-bottom: 16px;
}

.coming-soon-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--macos-text-primary);
  margin: 0 0 8px 0;
}

.coming-soon-description {
  font-size: var(--fs-title);
  color: var(--macos-text-secondary);
  margin: 0;
}

/* remove legacy footer visuals */

.loading-spinner {
  width: 16px;
  height: 16px;
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
    transform: translateY(20px);
  }

  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
