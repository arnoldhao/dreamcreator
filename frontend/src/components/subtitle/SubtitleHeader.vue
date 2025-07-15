<template>
  <div class="bg-base-200/50 border-b border-base-300">
    <!-- 主标题栏 -->
    <div class="px-4 py-3 group">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-3">
          <v-icon class="w-6 h-6 text-base-content/50" name="md-subtitles-outlined"></v-icon>
          <h1 class="text-lg font-medium text-base-content">{{ $t('subtitle.title') }}</h1>

          <!-- 项目名称 - 可编辑 -->
          <div v-if="currentProject" class="flex items-center space-x-2">
            <span class="text-base-content/40">|</span>
            <div v-if="!editingName" @click="startEditName"
              class="text-sm text-base-content/80 hover:text-base-content cursor-pointer px-2 py-1 rounded hover:bg-base-200 transition-colors"
              :class="{ 'text-base-content/40 italic': !currentProject.project_name }">
              {{ currentProject.project_name }}
            </div>
            <input v-else ref="nameInput" type="text" v-model="tempName" @blur="saveProjectName"
              @keyup.enter="$refs.nameInput.blur()"
              class="text-sm bg-transparent border border-primary rounded px-2 py-1 focus:outline-none focus:border-primary focus:ring-1 focus:ring-primary/20" />
          </div>

          <!-- 项目信息指示器 -->
          <div v-if="currentProject" class="relative">
            <div class="w-2 h-2 bg-primary/60 rounded-full opacity-0 group-hover:opacity-100 transition-opacity"></div>

            <!-- 悬停显示的项目详情 -->
            <div
              class="absolute top-6 left-0 bg-base-100 border border-base-300 rounded-lg shadow-lg p-3 min-w-64 opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200 z-50">
              <div class="text-xs text-base-content/70 space-y-1">
                <div>{{ $t('subtitle.common.lang') }}: {{ getLanguageDisplay(currentProject.language_metadata) }}</div>
                <div>{{ $t('subtitle.common.cues') }}: {{ currentProject.segments ? currentProject.segments.length : 0
                  }}</div>
                <div>{{ $t('subtitle.common.original_format') }}: {{ currentProject.metadata?.source_info?.file_ext ||
                  '未知' }}</div>
                <div>{{ $t('subtitle.common.created_at') }}: {{ formatDate(currentProject.created_at) }}</div>
                <div>{{ $t('subtitle.common.updated_at') }}: {{ formatDate(currentProject.updated_at) }}</div>
              </div>
            </div>
          </div>
        </div>

        <div class="flex items-center space-x-2">
          <!-- 自动保存状态 -->
          <div v-if="autoSaveStatus" class="flex items-center space-x-2 px-3 py-1 rounded-full bg-base-200">
            <div class="w-2 h-2 rounded-full" :class="{
              'bg-orange-400 animate-pulse': autoSaveStatus === 'saving',
              'bg-green-500': autoSaveStatus === 'saved',
              'bg-red-500': autoSaveStatus === 'error'
            }"></div>
            <span class="text-xs text-base-content/70">{{ getAutoSaveStatusText() }}</span>
          </div>

          <!-- 刷新按钮 -->
          <button v-if="currentProject" @click="$emit('refresh-projects')" :disabled="subtitleStore.isLoading" class="btn-macos"
            :class="{ 'opacity-50 cursor-not-allowed': subtitleStore.isLoading }">
            <svg class="w-4 h-4 mr-2" :class="{ 'animate-spin': subtitleStore.isLoading }" fill="none"
              stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15">
              </path>
            </svg>
            {{ subtitleStore.isLoading ? $t('common.refreshing') : $t('common.refresh') }}
          </button>

          <button @click="$emit('open-file')" class="btn-macos">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-5l-2-2H5a2 2 0 00-2 2z"></path>
            </svg>
            {{ $t('subtitle.common.open_file') }}
          </button>

          <button @click="$emit('show-history')" class="btn-macos">
            <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"></path>
            </svg>
            {{ $t('subtitle.common.history') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { useSubtitleStore } from '@/stores/subtitle'
import { subtitleService } from '@/services/subtitleService.js'
import { useI18n } from 'vue-i18n'

export default {
  name: 'SubtitleHeader',
  props: {
    currentProject: {
      type: Object,
      default: null
    },
    autoSaveStatus: {
      type: String,
      default: null
    }
  },
  emits: ['refresh-projects', 'open-file', 'show-history', 'update:projectData'],
  setup() {
    const { t } = useI18n()
    const subtitleStore = useSubtitleStore()
    return { t, subtitleStore }
  },
  data() {
    return {
      editingName: false,
      tempName: '',
      showProjectDetails: false
    }
  },
  methods: {
    getAutoSaveStatusText() {
      switch (this.autoSaveStatus) {
        case 'saving': return this.t('subtitle.header.auto_save_saving')
        case 'saved': return this.t('subtitle.header.auto_save_saved')
        case 'error': return this.t('subtitle.header.auto_save_error')
        default: return ''
      }
    },

    getLanguageDisplay(languageMetadata) {
      if (!languageMetadata) {
        return this.t('subtitle.header.not_set')
      }

      // 如果是对象，提取所有语言的 language_name
      if (typeof languageMetadata === 'object' && !Array.isArray(languageMetadata)) {
        const languageNames = Object.values(languageMetadata)
          .map(lang => lang.language_name)
          .filter(name => name) // 过滤掉空值

        if (languageNames.length > 0) {
          return languageNames.join(', ')
        }
      }

      // 如果是数组（兼容之前的逻辑）
      if (Array.isArray(languageMetadata)) {
        const names = languageMetadata
          .map(lang => lang.language_name)
          .filter(name => name)
        return names.length > 0 ? names.join(', ') : this.t('subtitle.header.not_set')
      }

      return this.t('subtitle.header.not_set')
    },

    startEditName() {
      this.editingName = true
      this.tempName = this.currentProject.project_name || ''
      this.$nextTick(() => {
        this.$refs.nameInput?.focus()
        this.$refs.nameInput?.select()
      })
    },

    async saveProjectName() {
      if (this.tempName.trim() === (this.currentProject.project_name || '').trim()) {
        this.editingName = false;
        return;
      }

      try {
        // 先保存到后端，等待结果
        const result = await subtitleService.saveProjectName(this.tempName.trim());

        if (result.success) {
          // 保存成功，更新前端状态
          const projectData = JSON.parse(result.data);
          this.$emit('update:projectData', projectData);
          this.editingName = false;
        } else {
          throw new Error(result.msg);
        }
      } catch (error) {
        // 保存失败，恢复原值，保持编辑状态
        this.tempName = this.currentProject.project_name || '';
        // 显示错误提示
        $message.error(error.message);
        // 保持编辑状态，让用户可以重试
        this.editingName = false; // 不要关闭编辑状态
      }
    },

    formatDate(timestamp) {
      if (!timestamp) return 'N/A'
      const date = new Date(timestamp * 1000)
      return date.toLocaleString('zh-CN')
    }
  }
}
</script>

<style scoped></style>