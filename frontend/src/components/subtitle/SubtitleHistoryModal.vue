<template>
  <div v-if="show" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
    <div class="bg-base-100 rounded-xl shadow-2xl max-w-4xl w-full max-h-[90vh] overflow-hidden border border-base-300 font-system">
      <!-- 模态框头部 -->
      <div class="flex items-center justify-between px-6 py-4 bg-base-200/30 border-b border-base-300">
        <div class="flex items-center space-x-4">
          <div class="header-icon">
            <v-icon name="co-history"></v-icon>
          </div>
          <h3 class="text-lg font-medium text-base-content">{{ $t('subtitle.history.title') }}</h3>
          <span class="text-sm text-base-content/60">({{ filteredProjects.length }} {{ $t('subtitle.history.projects') }})</span>
        </div>
        <button 
          @click="$emit('close')"
          class="p-2 hover:bg-base-200 rounded-lg transition-colors text-base-content/60 hover:text-base-content"
        >
          <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
          </svg>
        </button>
      </div>

      <!-- 搜索和过滤栏 -->
      <div class="px-6 py-4 bg-base-200/10 border-b border-base-300">
        <div class="flex items-center space-x-4">
          <!-- 搜索框 -->
          <div class="flex-1 relative">
            <svg class="w-4 h-4 absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/40 pointer-events-none" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
            <input 
              v-model="searchQuery"
              type="text" 
              :placeholder="$t('subtitle.history.search_project_placeholder')"
              class="input-macos w-full text-sm pl-10 pr-4 py-2"
            >
            <!-- 清除搜索按钮 -->
            <button 
              v-if="searchQuery"
              @click="searchQuery = ''"
              class="absolute right-3 top-1/2 transform -translate-y-1/2 text-base-content/40 hover:text-base-content transition-colors"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>
          
          <!-- 排序选择 -->
          <div class="flex-shrink-0">
            <select v-model="sortBy" class="select-macos text-sm min-w-[120px]">
              <option value="updated_at">{{ $t('subtitle.common.updated_at') }}</option>
              <option value="created_at">{{ $t('subtitle.common.created_at') }}</option>
              <option value="project_name">{{ $t('subtitle.common.project_name') }}</option>
              <option value="segments_count">{{ $t('subtitle.common.cues') }}</option>
            </select>
          </div>
          
          <!-- 视图切换 -->
          <div class="flex bg-base-200 rounded-lg p-1 flex-shrink-0">
            <button 
              @click="viewMode = 'list'"
              :class="['px-3 py-1 rounded text-xs transition-colors', viewMode === 'list' ? 'bg-base-100 text-base-content shadow-sm' : 'text-base-content/60 hover:text-base-content']"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 6h16M4 12h16M4 18h16"></path>
              </svg>
            </button>
            <button 
              @click="viewMode = 'grid'"
              :class="['px-3 py-1 rounded text-xs transition-colors', viewMode === 'grid' ? 'bg-base-100 text-base-content shadow-sm' : 'text-base-content/60 hover:text-base-content']"
            >
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1V5zM14 5a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1V5zM4 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1H5a1 1 0 01-1-1v-4zM14 15a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"></path>
              </svg>
            </button>
          </div>
        </div>
      </div>

      <!-- 历史记录列表 -->
      <div class="flex-1 overflow-hidden">
        <!-- 空状态 -->
        <div v-if="!filteredProjects.length" class="flex flex-col items-center justify-center h-64">
          <div class="w-16 h-16 mb-4 bg-base-200 rounded-full flex items-center justify-center">
            <svg class="w-8 h-8 text-base-content/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" 
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
          </div>
          <h4 class="text-base font-medium text-base-content mb-2">
            {{ searchQuery ? $t('subtitle.history.no_matching_item_found') : $t('subtitle.history.no_historical_records') }}
          </h4>
          <p class="text-sm text-base-content/60">
            {{ searchQuery ? $t('subtitle.history.try_diff_words') : $t('subtitle.history.no_imported_sub_found') }}
          </p>
        </div>

        <!-- 项目列表 - 列表视图 -->
        <div v-else-if="viewMode === 'list'" class="p-6">
          <div class="space-y-2 max-h-96 overflow-y-auto history-scroll">
            <div 
              v-for="(project, index) in paginatedProjects" 
              :key="project.id || index"
              class="group flex items-center justify-between p-3 bg-base-100 hover:bg-base-200/50 border border-base-300 rounded-lg transition-all duration-200 hover:shadow-sm"
            >
              <div class="flex items-center space-x-3 flex-1 min-w-0">
                <!-- 文件图标 -->
                <div class="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center flex-shrink-0">
                  <svg class="w-4 h-4 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                  </svg>
                </div>
                
                <!-- 项目信息 -->
                <div class="min-w-0 flex-1">
                  <h4 class="text-sm font-medium text-base-content truncate group-hover:text-primary transition-colors">
                    {{ project.project_name}}
                  </h4>
                  <div class="flex items-center space-x-3 mt-1 text-xs text-base-content/60">
                    <span>{{ project.segments?.length || 0 }} {{ $t('subtitle.common.cues') }}</span>
                    <span>{{ formatDate(project.updated_at) }}</span>
                    <span v-if="project.metadata?.source_info?.file_ext" class="px-2 py-0.5 bg-base-200 rounded text-xs">
                      {{ project.metadata.source_info.file_ext.toUpperCase() }}
                    </span>
                  </div>
                </div>
              </div>
              
              <!-- 操作按钮 -->
              <div class="flex items-center space-x-2 opacity-0 group-hover:opacity-100 transition-opacity">
                <button 
                  @click="$emit('load-recent-file', project)"
                  class="btn-macos-primary btn-macos-sm"
                >
                {{ $t('subtitle.common.open') }}
                </button>
                <button 
                  @click="$emit('remove-from-history', project.id || index)"
                  class="btn-macos-danger btn-macos-sm"
                  :title="$t('subtitle.history.remove_from_history')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
                  </svg>
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 项目列表 - 网格视图 -->
        <div v-else class="p-6">
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4 max-h-96 overflow-y-auto history-scroll">
            <div 
              v-for="(project, index) in paginatedProjects" 
              :key="project.id || index"
              class="group bg-base-100 hover:bg-base-200/50 border border-base-300 rounded-lg p-4 transition-all duration-200 hover:shadow-md cursor-pointer"
              @click="$emit('load-recent-file', project)"
            >
              <div class="flex items-start justify-between mb-3">
                <div class="w-10 h-10 bg-primary/10 rounded-lg flex items-center justify-center flex-shrink-0">
                  <svg class="w-5 h-5 text-primary" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" 
                      d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path>
                  </svg>
                </div>
                <button 
                  @click.stop="$emit('remove-from-history', project.id || index)"
                  class="opacity-0 group-hover:opacity-100 p-1 hover:bg-error/10 rounded transition-all text-error"
                  :title="$t('subtitle.history.remove_from_history')"
                >
                  <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
                  </svg>
                </button>
              </div>
              
              <h4 class="text-sm font-medium text-base-content mb-2 line-clamp-2 group-hover:text-primary transition-colors">
                {{ project.project_name}}
              </h4>
              
              <div class="space-y-2 text-xs text-base-content/60">
                <div class="flex items-center justify-between">
                  <span>{{ $t('subtitle.common.cues') }}</span>
                  <span class="font-medium">{{ project.segments?.length || 0 }}</span>
                </div>
                <div class="flex items-center justify-between">
                  <span>{{ $t('subtitle.common.updated_at') }}</span>
                  <span>{{ formatDate(project.updated_at) }}</span>
                </div>
                <div v-if="project.metadata?.source_info?.file_ext" class="flex items-center justify-between">
                  <span>{{ $t('subtitle.common.format') }}</span>
                  <span class="px-2 py-0.5 bg-base-200 rounded font-medium">
                    {{ project.metadata.source_info.file_ext.toUpperCase() }}
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 分页控制 -->
        <div v-if="totalPages > 1" class="flex items-center justify-center px-6 py-4 border-t border-base-300">
          <div class="flex items-center space-x-2">
            <button 
              @click="currentPage = Math.max(1, currentPage - 1)"
              :disabled="currentPage === 1"
              class="btn-macos-secondary btn-macos-sm"
              :class="{ 'opacity-50 cursor-not-allowed': currentPage === 1 }"
            >
            {{ $t('subtitle.common.previous') }}
            </button>
            
            <div class="flex items-center space-x-1">
              <button 
                v-for="page in visiblePages" 
                :key="page"
                @click="currentPage = page"
                :class="['px-3 py-1 rounded text-sm transition-colors', page === currentPage ? 'bg-primary text-white' : 'text-base-content/60 hover:text-base-content hover:bg-base-200']"
              >
                {{ page }}
              </button>
            </div>
            
            <button 
              @click="currentPage = Math.min(totalPages, currentPage + 1)"
              :disabled="currentPage === totalPages"
              class="btn-macos-secondary btn-macos-sm"
              :class="{ 'opacity-50 cursor-not-allowed': currentPage === totalPages }"
            >
            {{ $t('subtitle.common.next') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 模态框底部 -->
      <div class="flex items-center justify-between px-6 py-4 bg-base-200/20 border-t border-base-300">
        <button 
          v-if="subtitleProjects?.length > 0"
          @click="$emit('clear-history')"
          class="btn-macos-danger btn-macos-sm"
        >
          <svg class="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path>
          </svg>
          {{ $t('subtitle.history.remove_all_history') }}
        </button>
        <div class="flex space-x-3 ml-auto">
          <button 
            @click="$emit('close')"
            class="btn-macos-secondary btn-macos-sm"
          >
          {{ $t('subtitle.common.cancel') }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SubtitleHistoryModal',
  props: {
    show: {
      type: Boolean,
      default: false
    },
    subtitleProjects: {
      type: Array,
      default: () => []
    }
  },
  emits: ['close', 'load-recent-file', 'remove-from-history', 'clear-history'],
  data() {
    return {
      searchQuery: '',
      sortBy: 'updated_at',
      viewMode: 'list', // 'list' or 'grid'
      currentPage: 1,
      itemsPerPage: 10
    }
  },
  computed: {
    filteredProjects() {
      let projects = [...(this.subtitleProjects || [])]
      
      // 搜索过滤
      if (this.searchQuery) {
        const query = this.searchQuery.toLowerCase()
        projects = projects.filter(project => 
          (project.project_name || '').toLowerCase().includes(query)
        )
      }
      
      // 排序
      projects.sort((a, b) => {
        switch (this.sortBy) {
          case 'project_name':
            return (a.project_name || '').localeCompare(b.project_name || '')
          case 'segments_count':
            return (b.segments?.length || 0) - (a.segments?.length || 0)
          case 'created_at':
            return (b.created_at || 0) - (a.created_at || 0)
          case 'updated_at':
          default:
            return (b.updated_at || 0) - (a.updated_at || 0)
        }
      })
      
      return projects
    },
    
    totalPages() {
      return Math.ceil(this.filteredProjects.length / this.itemsPerPage)
    },
    
    paginatedProjects() {
      const start = (this.currentPage - 1) * this.itemsPerPage
      const end = start + this.itemsPerPage
      return this.filteredProjects.slice(start, end)
    },
    
    visiblePages() {
      const pages = []
      const total = this.totalPages
      const current = this.currentPage
      
      if (total <= 7) {
        for (let i = 1; i <= total; i++) {
          pages.push(i)
        }
      } else {
        if (current <= 4) {
          for (let i = 1; i <= 5; i++) {
            pages.push(i)
          }
          pages.push('...', total)
        } else if (current >= total - 3) {
          pages.push(1, '...')
          for (let i = total - 4; i <= total; i++) {
            pages.push(i)
          }
        } else {
          pages.push(1, '...')
          for (let i = current - 1; i <= current + 1; i++) {
            pages.push(i)
          }
          pages.push('...', total)
        }
      }
      
      return pages.filter(page => page !== '...' || pages.indexOf(page) === pages.lastIndexOf(page))
    }
  },
  watch: {
    searchQuery() {
      this.currentPage = 1
    },
    sortBy() {
      this.currentPage = 1
    }
  },
  methods: {
    formatDate(timestamp) {
      if (!timestamp) return 'N/A'
      const date = new Date(timestamp * 1000)
      const now = new Date()
      const diff = now - date
      
      if (diff < 24 * 60 * 60 * 1000) {
        return date.toLocaleString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit'
        })
      } else if (diff < 7 * 24 * 60 * 60 * 1000) {
        return date.toLocaleString('zh-CN', {
          weekday: 'short',
          hour: '2-digit',
          minute: '2-digit'
        })
      } else {
        return date.toLocaleString('zh-CN', {
          month: 'short',
          day: 'numeric'
        })
      }
    }
  }
};
</script>

<style scoped>
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

/* macOS 风格的滚动条 */
.history-scroll {
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent;
}

.history-scroll::-webkit-scrollbar {
  width: 6px;
}

.history-scroll::-webkit-scrollbar-track {
  background: transparent;
  margin: 4px 0;
}

.history-scroll::-webkit-scrollbar-thumb {
  background: rgba(0, 0, 0, 0.15);
  border-radius: 3px;
  transition: all 0.2s ease;
  opacity: 0;
}

.history-scroll::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.25);
}

.history-scroll:hover::-webkit-scrollbar-thumb {
  opacity: 1;
}

/* 输入框和选择框样式 */
.input-macos {
  padding: 8px 12px;
  border: 1px solid var(--macos-border);
  border-radius: 6px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: 13px;
  transition: all 0.15s ease;
  min-height: 36px;
}

.input-macos:focus {
  outline: none;
  border-color: var(--macos-blue);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.3);
}

.select-macos {
  padding: 8px 12px;
  border: 1px solid var(--macos-border);
  border-radius: 6px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: 13px;
  cursor: pointer;
  min-height: 36px;
}

.select-macos:focus {
  outline: none;
  border-color: var(--macos-blue);
  box-shadow: 0 0 0 3px rgba(0, 122, 255, 0.3);
}

/* 确保搜索框内的元素不重叠 */
.input-macos[class*="pl-10"] {
  padding-left: 40px;
}

.input-macos[class*="pr-4"] {
  padding-right: 40px;
}

/* 文本截断 */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

/* 模态框动画 */
.fixed {
  animation: modalFadeIn 0.2s ease-out;
}

.bg-base-100 {
  animation: modalSlideIn 0.3s ease-out;
}

@keyframes modalFadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes modalSlideIn {
  from {
    opacity: 0;
    transform: scale(0.95) translateY(-10px);
  }
  to {
    opacity: 1;
    transform: scale(1) translateY(0);
  }
}
</style>