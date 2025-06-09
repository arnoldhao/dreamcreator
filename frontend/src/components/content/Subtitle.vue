<template>
  <div class="subtitle-editor h-full bg-base-100 font-system flex flex-col">
    <!-- 标题栏组件 -->
    <SubtitleHeader :current-project="currentProject" :auto-save-status="autoSaveStatus" @open-file="openFile"
      @show-history="showHistory = true" @update:projectData="updateCurrentProject" class="flex-shrink-0" />

    <!-- 主内容区域 -->
    <div v-if="currentProject" class="flex flex-1 min-h-0">
      <!-- 左主编辑区域 -->
      <div class="flex-1 p-4 min-h-0">
        <!-- 字幕列表组件 -->
        <SubtitleList :subtitles="currentLanguageSegments" :current-language="currentLanguage"
          :available-languages="availableLanguages" :subtitle-counts="subtitleCounts" @add-language="addLanguage"
          @update:currentLanguage="setCurrentLanguage" @update:projectData="updateCurrentProject" />
      </div>

      <!-- 右侧边栏 -->
      <div class="w-64 bg-base-200/30 border-l border-base-300 flex flex-col">
        <div class="flex-1 overflow-y-auto sidebar-scroll p-4">
          <!-- 导出配置组件 -->
          <SubtitleExportConfig :project-data="currentProject" :current-language="currentLanguage"
            @save-config="saveConfig" @export-subtitles="exportSubtitles" />
        </div>
      </div>
    </div>

    <!-- 欢迎页面 - 使用 flex: 1 占据剩余空间 -->
    <div v-else class="flex flex-1 min-h-0">
      <SubtitleWelcome @open-file="openFile" @show-history="showHistory = true" />
    </div>

    <!-- 导入配置Modal -->
    <SubtitleImportModal :show="showImportModal" :file-path="selectedFilePath" @close="showImportModal = false"
      @import="handleImportWithOptions" />

    <!-- 历史记录模态框 -->
    <SubtitleHistoryModal :show="showHistory" :subtitle-projects="subtitleProjects" @close="showHistory = false"
      @load-recent-file="loadRecentFile" @remove-from-history="removeFromHistory" @clear-history="clearAllHistory" />
  </div>
</template>

<script>
import { ref, computed, watch } from 'vue'
import { SelectFile } from 'wailsjs/go/systems/Service'
import { OpenFileWithOptions, GetSubtitle, ListSubtitles, DeleteSubtitle, DeleteAllSubtitle } from 'wailsjs/go/api/SubtitlesAPI'
import { subtitleService } from '@/services/subtitleService.js'
import SubtitleHeader from '@/components/subtitle/SubtitleHeader.vue'
import SubtitleExportConfig from '@/components/subtitle/SubtitleExportConfig.vue'
import SubtitleList from '@/components/subtitle/SubtitleList.vue'
import SubtitleImportModal from '@/components/subtitle/SubtitleImportModal.vue'
import SubtitleHistoryModal from '@/components/subtitle/SubtitleHistoryModal.vue'
import SubtitleWelcome from '@/components/subtitle/SubtitleWelcome.vue'
import { useI18n } from 'vue-i18n'

export default {
  name: 'Subtitle',
  components: {
    SubtitleHeader,
    SubtitleExportConfig,
    SubtitleList,
    SubtitleHistoryModal,
    SubtitleWelcome,
  },
  setup() {
    // 响应式数据
    const currentProject = ref(null)
    const currentLanguage = ref('English')
    const autoSaveStatus = ref(null)
    const isLoading = ref(false)
    const showHistory = ref(false)
    const subtitleProjects = ref([])
    const selectedFilePath = ref('')
    const showImportModal = ref(false)

    // i18
    const { t } = useI18n()

    // 公共函数
    // 可以提取的公共逻辑
    const setProjectData = (projectData) => {
      currentProject.value = projectData
      const availableLangs = Object.keys(projectData.language_metadata || {})
      if (availableLangs.length > 0) {
        currentLanguage.value = availableLangs[0]
      }
      subtitleService.initialize(projectData)
    }

    // 初始化时加载字幕项目列表
    const loadSubtitleProjects = async () => {
      try {
        isLoading.value = true
        const response = await ListSubtitles()
        if (response.success) {
          const projectsData = JSON.parse(response.data)
          subtitleProjects.value = projectsData
          // if no projects, set currentProject to null
          if (!projectsData || projectsData?.length === 0) {
            currentProject.value = null
            currentLanguage.value = null
          }
        } else {
          throw new Error(response.msg)
        }
      } catch (error) {
        $message.error(error.message)
      } finally {
        isLoading.value = false
      }
    }

    // 加载特定字幕项目
    const loadSubtitleProject = async (projectId) => {
      try {
        isLoading.value = true
        const result = await GetSubtitle(projectId)

        if (!result.success) {
          throw new Error(result.msg)
        }

        // 解析返回的数据
        let projectData
        try {
          projectData = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        } catch (parseError) {
          throw new Error(parseError.message)
        }

        // 设置当前项目
        setProjectData(projectData)
      } catch (error) {
        $message.error(error.message)
      } finally {
        isLoading.value = false
      }
    }

    // 计算属性
    const availableLanguages = computed(() => {
      if (!currentProject.value?.language_metadata) return {}
      return currentProject.value.language_metadata
    })

    const currentLanguageSegments = computed(() => {
      if (!currentProject.value?.segments) return []
      return currentProject.value.segments.filter(segment =>
        segment.languages && segment.languages[currentLanguage.value]
      )
    })

    const subtitleCounts = computed(() => {
      const counts = {}
      if (currentProject.value?.language_metadata) {
        Object.keys(currentProject.value.language_metadata).forEach(langCode => {
          counts[langCode] = getLanguageSegmentCount(langCode)
        })
      }
      return counts
    })

    // 工具函数
    const getLanguageSegmentCount = (langCode) => {
      if (!currentProject.value?.segments) return 0
      return currentProject.value.segments.filter(segment =>
        segment.languages && segment.languages[langCode]
      ).length
    }

    // 处理带选项的导入
    const handleImportWithOptions = async ({ filePath, options }) => {
      try {
        showImportModal.value = false
        isLoading.value = true
        const result = await OpenFileWithOptions(filePath, options)
        if (!result.success) {
          throw new Error(result.msg)
        }

        // 解析返回的数据
        let projectData
        try {
          projectData = typeof result.data === 'string' ? JSON.parse(result.data) : result.data
        } catch (parseError) {
          throw new Error(parseError.message)
        }

        setProjectData(projectData)

        // 重新加载项目列表
        await loadSubtitleProjects()
      } catch (error) {
        $message.error(error.message)
      } finally {
        isLoading.value = false
      }
    }
    // 主要功能函数
    const openFile = async () => {
      try {
        isLoading.value = true
        // 调用 Wails 的文件选择 API
        const fileResult = await SelectFile(t('subtitle.common.select_sub_file'), ['srt', 'itt']) //todo
        if (!fileResult.success) {
          return
        }

        const filePath = fileResult.data?.path
        if (!filePath) {
          return
        }
        // 停止loading，准备打开配置modal
        isLoading.value = false

        // 设置选中的文件路径并打开配置modal
        selectedFilePath.value = filePath
        showImportModal.value = true

      } catch (error) {
        isLoading.value = false
      }
    }

    const setCurrentLanguage = (langCode) => {
      currentLanguage.value = langCode
    }

    const loadRecentFile = async (fileItem) => {
      await loadSubtitleProject(fileItem.id)
      showHistory.value = false
    }

    const removeFromHistory = async (id) => {
      try {
        const response = await DeleteSubtitle(id)
        if (response.success) {
          await loadSubtitleProjects()
        } else {
          throw new Error(response.msg)
        }
      } catch (error) {
        $message.error(error.message)
      }
    }

    const clearAllHistory = async () => {
      try {
        const response = await DeleteAllSubtitle()
        if (response.success) {
          await loadSubtitleProjects()
        } else {
          throw new Error(response.msg)
        }
      } catch (error) {
        $message.error(error.message)
      }
    }

    const saveConfig = (config) => {
      // do nothing
    }

    const exportSubtitles = (config) => {
      // do nothing
    }

    const updateCurrentProject = (projectData) => {
      currentProject.value = projectData
    }

    const addLanguage = () => {
      $dialog.info({
        title: t('subtitle.add_language.title'),
        content: t('subtitle.add_language.coming_soon')
      })
    }

    // 生命周期
    watch(() => showHistory.value, async (newValue, oldValue) => {
      if (newValue === true && oldValue === false) {
        await loadSubtitleProjects()
      }
    })

    return {
      // i18n
      t,

      // 响应式数据
      currentProject,
      currentLanguage,
      autoSaveStatus,
      isLoading,
      showHistory,
      subtitleProjects,
      selectedFilePath,
      showImportModal,

      // 计算属性
      availableLanguages,
      currentLanguageSegments,
      subtitleCounts,

      // 方法
      setProjectData,
      loadSubtitleProjects,
      loadSubtitleProject,
      handleImportWithOptions,
      openFile,
      setCurrentLanguage,
      loadRecentFile,
      removeFromHistory,
      clearAllHistory,
      saveConfig,
      exportSubtitles,
      updateCurrentProject,
      addLanguage,
      getLanguageSegmentCount
    }
  }
}
</script>


<style scoped>
/* macOS原生应用风格样式 */
.font-system {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

/* 使用全局滚动条样式，删除重复定义 */
/* 保留侧边栏特殊滚动条样式 */
.sidebar-scroll {
  @extend .history-scroll;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar-scroll::-webkit-scrollbar-thumb {
  opacity: 0;
  transition: opacity 0.3s ease;
}

.sidebar-scroll:hover::-webkit-scrollbar-thumb {
  opacity: 1;
}

/* 动画效果使用全局定义 */
.subtitle-editor>* {
  animation: slideInUp 0.3s ease-out;
}
</style>