<template>
  <div v-if="show" class="fixed inset-0 bg-black/50 backdrop-blur-sm flex items-center justify-center z-50 p-4">
    <div
      class="bg-base-100 rounded-xl shadow-2xl max-w-5xl w-full max-h-[90vh] overflow-hidden border border-base-300 font-system">
      <!-- 模态框头部 -->
      <div class="flex items-center justify-between px-6 py-4 bg-base-200/30 border-b border-base-300">
        <div class="flex items-center space-x-4">
          <div class="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
            <v-icon name="ri-leaf-line" class="w-5 h-5 text-primary"></v-icon>
          </div>
          <h3 class="text-lg font-medium text-base-content">{{ $t('cookies.title') }}</h3>
          <span class="text-sm text-base-content/60">({{ browsers.length }} {{ $t('cookies.browsers') }})</span>
        </div>
        <div class="flex items-center space-x-2">
          <button @click="closeModal"
            class="p-2 hover:bg-base-200 rounded-lg transition-colors text-base-content/60 hover:text-base-content">
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
            </svg>
          </button>
        </div>
      </div>

      <!-- 搜索和过滤栏 -->
      <div class="px-6 py-4 bg-base-200/10 border-b border-base-300">
        <div class="flex items-center space-x-4">
          <!-- 搜索框 -->
          <div class="flex-1 relative">
            <svg
              class="w-4 h-4 absolute left-3 top-1/2 transform -translate-y-1/2 text-base-content/40 pointer-events-none"
              fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
                d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path>
            </svg>
            <input v-model="searchQuery" type="text" :placeholder="$t('cookies.search_placeholder')"
              class="input-macos w-full text-sm pl-10 pr-10 py-2">
            <!-- 清除搜索按钮 -->
            <button v-if="searchQuery" @click="searchQuery = ''"
              class="absolute right-3 top-1/2 transform -translate-y-1/2 text-base-content/40 hover:text-base-content transition-colors">
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path>
              </svg>
            </button>
          </div>

          <!-- 浏览器过滤 -->
          <div class="flex-shrink-0">
            <select v-model="selectedBrowser" class="select-macos text-sm min-w-[140px]">
              <option value="">{{ $t('cookies.all_browsers') }}</option>
              <option v-for="browser in browsers" :key="browser" :value="browser">
                {{ browser }}
              </option>
            </select>
          </div>
        </div>
      </div>

      <!-- 主要内容区域 -->
      <div class="flex-1 overflow-hidden">
        <!-- 加载状态 -->
        <div v-if="isLoading" class="flex flex-col items-center justify-center h-64">
          <div class="w-8 h-8 border-2 border-primary border-t-transparent rounded-full animate-spin mb-4"></div>
          <p class="text-sm text-base-content/60">{{ $t('cookies.loading_cookies') }}</p>
        </div>

        <!-- 空状态 -->
        <div v-else-if="!filteredBrowsers.length" class="flex flex-col items-center justify-center h-64">
          <div class="w-16 h-16 mb-4 bg-base-200 rounded-full flex items-center justify-center">
            <svg class="w-8 h-8 text-base-content/40" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
                d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"></path>
            </svg>
          </div>
          <h4 class="text-base font-medium text-base-content mb-2">
            {{ searchQuery ? $t('cookies.no_matching_cookies') : $t('cookies.no_cookies_found') }}
          </h4>
          <p class="text-sm text-base-content/60">
            {{ searchQuery ? $t('cookies.try_different_search') : $t('cookies.try_sync') }}
          </p>
        </div>

        <!-- Cookies 展示 -->
        <div v-else class="p-6">
          <div class="space-y-4 max-h-96 overflow-y-auto cookies-scroll">
            <div v-for="browser in filteredBrowsers" :key="browser"
              class="bg-white/80 backdrop-blur-sm border border-gray-200/60 rounded-xl overflow-hidden shadow-sm hover:shadow-md transition-all duration-300 ease-out">

              <!-- 浏览器头部 - 重新设计 -->
              <div class="px-5 py-4 bg-gradient-to-r from-gray-50/50 to-white/30 border-b border-gray-100/80">
                <div class="flex items-center justify-between">
                  <!-- 左侧：浏览器信息 -->
                  <div class="flex items-center space-x-4 flex-1 min-w-0">
                    <!-- 浏览器图标 -->
                    <div
                      class="w-10 h-10 bg-gradient-to-br from-blue-50 to-blue-100 rounded-lg flex items-center justify-center shadow-sm hover:shadow-md transition-all duration-200">
                      <v-icon :name="getBrowserIcon(browser)" class="w-6 h-6 text-blue-600"></v-icon>
                    </div>

                    <!-- 浏览器详情 -->
                    <div class="flex-1 min-w-0">
                      <!-- 第一行：浏览器名称 + 状态 + 展开按钮 -->
                      <div class="flex items-center space-x-3 mb-1">
                        <h4 class="text-base font-semibold text-gray-900 tracking-tight">{{ browser }}</h4>
                        <!-- 状态指示器 -->
                        <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                          :class="getStatusClass(browser)">
                          <span class="w-1.5 h-1.5 rounded-full mr-1.5" :class="getStatusDotClass(browser)"></span>
                          {{ getStatusText(browser) }}
                        </span>
                        <!-- 展开按钮移到这里 -->
                        <button @click="toggleBrowser(browser)"
                          class="flex items-center justify-center w-6 h-6 rounded-full bg-gray-100 hover:bg-gray-200 transition-colors duration-200 ml-auto">
                          <v-icon name="ri-arrow-down-s-line"
                            class="w-4 h-4 text-gray-600 transition-transform duration-200"
                            :class="{ 'rotate-180': expandedBrowsers.includes(browser) }"></v-icon>
                        </button>
                      </div>

                      <!-- 第二行：统计信息 + 同步时间 -->
                      <div class="flex items-center space-x-4 text-sm text-gray-600 mb-1">
                        <span class="flex items-center space-x-1">
                          <v-icon name="ri-database-2-line" class="w-3.5 h-3.5"></v-icon>
                          <span>{{ getBrowserCookieCount(browser) }} {{ $t('cookies.cookies') }}</span>
                        </span>
                        <span class="flex items-center space-x-1">
                          <v-icon name="ri-global-line" class="w-3.5 h-3.5"></v-icon>
                          <span>{{ getDomainCount(browser) }} {{ $t('cookies.domains') }}</span>
                        </span>
                        <!-- 同步时间信息 -->
                        <span v-if="getLastSyncTime(browser)" class="flex items-center space-x-1 text-gray-500">
                          <v-icon name="ri-time-line" class="w-3.5 h-3.5"></v-icon>
                          <span class="text-xs">{{ formatSyncTime(getLastSyncTime(browser)) }}</span>
                        </span>
                      </div>

                      <!-- 第三行：同步状态信息（如果有错误或重要状态） -->
                      <div v-if="getSyncStatusText(browser)" class="flex items-start space-x-2">
                        <span class="w-1 h-1 rounded-full mt-1.5 flex-shrink-0"
                          :class="getSyncStatusClass(browser)"></span>
                        <p class="text-xs select-text cursor-text leading-relaxed flex-1"
                          :class="getSyncStatusClass(browser)" :title="getSyncStatusText(browser)">
                          {{ getSyncStatusText(browser) }}
                        </p>
                      </div>
                    </div>
                  </div>

                  <!-- 右侧：同步按钮组 -->
                  <div class="flex items-center space-x-2 ml-4">
                    <div v-for="syncType in getSyncFromOptions(browser)" :key="syncType">
                      <button @click="syncCookies(syncType, [browser])" :disabled="syncingBrowsers.has(browser)"
                        class="inline-flex items-center px-3 py-1.5 text-xs font-medium rounded-lg transition-all duration-200 focus:outline-none focus:ring-2 focus:ring-offset-1"
                        :class="{
                          'opacity-50 cursor-not-allowed': syncingBrowsers.has(browser),
                          'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 shadow-sm': syncType === 'yt-dlp',
                          'bg-gray-100 text-gray-700 hover:bg-gray-200 focus:ring-gray-400': syncType === 'canme'
                        }">
                        <v-icon name="ri-refresh-line" class="w-3 h-3 mr-1.5"
                          :class="{ 'animate-spin': syncingBrowsers.has(browser) }"></v-icon>
                        {{ $t('cookies.sync_with', { type: syncType }) }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>

              <!-- Cookies 表格 -->
              <div v-show="expandedBrowsers.includes(browser)" class="transition-all duration-200">
                <div class="overflow-x-auto">
                  <table class="w-full text-sm">
                    <thead class="bg-base-200/30">
                      <tr>
                        <th
                          class="px-4 py-3 text-left text-xs font-medium text-base-content/70 uppercase tracking-wider">
                          {{ $t('cookies.domain') }}
                        </th>
                        <th
                          class="px-4 py-3 text-left text-xs font-medium text-base-content/70 uppercase tracking-wider">
                          {{ $t('cookies.name') }}
                        </th>
                        <th
                          class="px-4 py-3 text-left text-xs font-medium text-base-content/70 uppercase tracking-wider">
                          {{ $t('cookies.value') }}
                        </th>
                      </tr>
                    </thead>
                    <tbody class="divide-y divide-base-300">
                      <tr v-for="(cookie, index) in getFilteredCookies(browser)" :key="index"
                        class="hover:bg-base-200/20 transition-colors">
                        <td class="px-4 py-3 font-mono text-xs text-base-content/80 select-text cursor-text">
                          {{ cookie.Domain }}
                        </td>
                        <td class="px-4 py-3 font-mono text-xs text-base-content select-text cursor-text">
                          {{ cookie.Name }}
                        </td>
                        <td class="px-4 py-3 font-mono text-xs text-base-content/60 max-w-xs select-text cursor-text">
                          <div class="truncate select-text" :title="cookie.Value">
                            {{ cookie.Value }}
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 模态框底部 -->
      <div class="flex items-center justify-between px-6 py-4 bg-base-200/20 border-t border-base-300">
        <div class="text-sm text-base-content/60">
          {{ $t('cookies.total_cookies', { count: totalCookiesCount }) }}
        </div>
        <button @click="closeModal" class="btn-macos-secondary">
          {{ $t('common.close') }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, watch, onMounted, computed, onUnmounted } from 'vue';
import { useI18n } from 'vue-i18n';
import { ListAllCookies, SyncCookies } from 'wailsjs/go/api/CookiesAPI';
import { useDtStore } from '@/handlers/downtasks';

const props = defineProps({
  show: Boolean,
});

const emit = defineEmits(['update:show']);

const { t } = useI18n();
const dtStore = useDtStore();

const isLoading = ref(false);
const browsers = ref([]);
const cookiesByBrowser = ref({});
const searchQuery = ref('');
const selectedBrowser = ref('');
const expandedBrowsers = ref([]);
const syncingBrowsers = ref(new Set()); // 跟踪正在同步的浏览器

// 计算属性
const filteredBrowsers = computed(() => {
  let filtered = browsers.value;

  if (selectedBrowser.value) {
    filtered = filtered.filter(browser => browser === selectedBrowser.value);
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    filtered = filtered.filter(browser => {
      const browserMatch = browser.toLowerCase().includes(query);
      const cookiesMatch = getFilteredCookies(browser).length > 0;
      return browserMatch || cookiesMatch;
    });
  }

  return filtered;
});

const totalCookiesCount = computed(() => {
  if (!browsers.value || browsers.value.length === 0) {
    return 0;
  }

  const total = browsers.value.reduce((total, browser) => {
    const count = getBrowserCookieCount(browser);
    return total + count;
  }, 0);

  return total;
});

// 方法
const closeModal = () => {
  emit('update:show', false);
};

const toggleBrowser = (browser) => {
  const index = expandedBrowsers.value.indexOf(browser);
  if (index > -1) {
    expandedBrowsers.value.splice(index, 1);
  } else {
    expandedBrowsers.value.push(browser);
  }
};

const getBrowserCookieCount = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData || !browserData.domain_cookies) {
    return 0;
  }
  return Object.values(browserData.domain_cookies).reduce((total, domain) => {
    return total + (domain && Array.isArray(domain.cookies) ? domain.cookies.length : 0);
  }, 0);
};

// 获取域名数量
const getDomainCount = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData || !browserData.domain_cookies) {
    return 0;
  }
  return Object.keys(browserData.domain_cookies).length;
};

// 获取浏览器状态
const getStatusText = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData) return t('cookies.status.unknown');

  switch (browserData.status) {
    case 'synced': return t('cookies.status.synced');
    case 'never': return t('cookies.status.never');
    case 'syncing': return t('cookies.status.syncing');
    case 'error': return t('cookies.status.error');
    default: return t('cookies.status.unknown');
  }
};

// 获取状态样式类
const getStatusClass = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData) return 'bg-gray-100 text-gray-600';

  switch (browserData.status) {
    case 'synced': return 'bg-green-100 text-green-700';
    case 'never': return 'bg-gray-100 text-gray-600';
    case 'syncing': return 'bg-blue-100 text-blue-700';
    case 'error': return 'bg-red-100 text-red-700';
    default: return 'bg-gray-100 text-gray-600';
  }
};

// 获取状态指示点的样式类
const getStatusDotClass = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData) return 'bg-gray-400';

  switch (browserData.status) {
    case 'synced': return 'bg-green-500';
    case 'never': return 'bg-gray-400';
    case 'syncing': return 'bg-blue-500';
    case 'error': return 'bg-red-500';
    default: return 'bg-gray-400';
  }
};

// 获取同步状态文本
const getSyncStatusText = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData || !browserData.last_sync_status) return '';

  switch (browserData.last_sync_status) {
    case 'success': return t('cookies.sync_success');
    case 'failed': return t('cookies.sync_error', { msg: browserData.status_description });
    default: return '';
  }
};

// 获取同步状态样式类
const getSyncStatusClass = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData || !browserData.last_sync_status) return '';

  switch (browserData.last_sync_status) {
    case 'success': return 'text-green-600';
    case 'failed': return 'text-red-600';
    default: return '';
  }
};

// 获取同步选项
const getSyncFromOptions = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData || !browserData.sync_from) {
    return ['canme', 'yt-dlp']; // 默认选项
  }
  return browserData.sync_from;
};

const getFilteredCookies = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData) return [];

  let allCookies = [];

  if (browserData.domain_cookies) {
    Object.values(browserData.domain_cookies).forEach(domain => {
      if (domain.cookies && Array.isArray(domain.cookies)) {
        allCookies = allCookies.concat(domain.cookies);
      }
    });
  }

  if (searchQuery.value) {
    const query = searchQuery.value.toLowerCase();
    allCookies = allCookies.filter(cookie =>
      (cookie.Domain && cookie.Domain.toLowerCase().includes(query)) ||
      (cookie.Name && cookie.Name.toLowerCase().includes(query)) ||
      (cookie.Value && cookie.Value.toLowerCase().includes(query))
    );
  }

  return allCookies;
};

const fetchCookies = async () => {
  isLoading.value = true;
  try {
    const res = await ListAllCookies();
    if (res.success) {
      const cookiesData = JSON.parse(res.data || '{}') || {};
      cookiesByBrowser.value = cookiesData;
      browsers.value = Object.keys(cookiesData);
      // 默认不展开
      // if (browsers.value.length > 0 && expandedBrowsers.value.length === 0) {
      //   expandedBrowsers.value.push(browsers.value[0]);
      // }
    } else {
      throw new Error(res.msg);
    }
  } catch (error) {
    $message.error('Fetch cookies error:', error);
  } finally {
    isLoading.value = false;
  }
};

const syncCookies = async (syncFrom, browsers) => {
  if (!syncFrom || !browsers || browsers.length === 0) return;

  // 标记浏览器为同步中
  browsers.forEach(browser => {
    syncingBrowsers.value.add(browser);
  });

  try {
    // 注册 WebSocket 回调来监听同步结果
    const handleSyncResult = (data) => {
      switch (data.status) {
        case 'started':
          $message.info(t('cookies.sync_started', { type: data.sync_from }));
          break;
        case 'success':
          $message.success(t('cookies.sync_success', { type: data.sync_from }));
          break;
        case 'failed':
          $message.error(t('cookies.sync_error', { type: data.sync_from, error: data.error }));
          break;
        default:
          $message.warning('Unknown sync status:', data.status);
          break;
      }

      // 只有在完成时才移除回调
      if (data.done) {
        // 移除同步中标记
        browsers.forEach(browser => {
          syncingBrowsers.value.delete(browser);
        });
        fetchCookies(); // 刷新 cookie 列表
        dtStore.removeCookieSyncCallback(handleSyncResult);
      }
    };

    // 注册回调
    dtStore.registerCookieSyncCallback(handleSyncResult);

    // 调用同步 API（立即返回）
    const res = await SyncCookies(syncFrom, browsers);

    if (res.success) {
      $message.info(t('cookies.sync_started', { type: syncFrom }));
    } else {
      // 如果启动失败，移除同步中标记和回调
      browsers.forEach(browser => {
        syncingBrowsers.value.delete(browser);
      });
      dtStore.removeCookieSyncCallback(handleSyncResult);
      throw new Error(res.msg);
    }
  } catch (error) {
    // 移除同步中标记
    browsers.forEach(browser => {
      syncingBrowsers.value.delete(browser);
    });
    $message.error(t('cookies.sync_start_error', { error: error.message }));
  }
};

const getBrowserIcon = (browserName) => {
  const name = browserName.toLowerCase();
  if (name.includes('chrome')) return 'ri-chrome-fill';
  if (name.includes('firefox')) return 'ri-firefox-fill';
  if (name.includes('safari')) return 'ri-safari-fill';
  if (name.includes('edge')) return 'ri-edge-fill';
  return 'ri-global-line';
};

// 获取浏览器的上次同步时间
const getLastSyncTime = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  return browserData?.last_sync_time || null;
};

// 格式化同步时间显示
const formatSyncTime = (syncTime) => {
  if (!syncTime) return '';

  const date = new Date(syncTime);
  const now = new Date();
  const diffMs = now - date;
  const diffMins = Math.floor(diffMs / (1000 * 60));
  const diffHours = Math.floor(diffMs / (1000 * 60 * 60));
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

  if (diffMins < 1) {
    return t('cookies.just_now');
  } else if (diffMins < 60) {
    return t('cookies.minutes_ago', { count: diffMins });
  } else if (diffHours < 24) {
    return t('cookies.hours_ago', { count: diffHours });
  } else if (diffDays < 7) {
    return t('cookies.days_ago', { count: diffDays });
  } else {
    return date.toLocaleDateString();
  }
};

// WebSocket 事件处理
const initWebSocketHandlers = () => {
  dtStore.init();
};

const cleanupWebSocketHandlers = () => {
  dtStore.cleanup();
};

watch(() => props.show, (newVal) => {
  if (newVal) {
    fetchCookies();
    initWebSocketHandlers();
    // 重置搜索和过滤状态
    searchQuery.value = '';
    selectedBrowser.value = '';
    expandedBrowsers.value = [];
    syncingBrowsers.value.clear();
  } else {
    cleanupWebSocketHandlers();
  }
});

watch(searchQuery, (newQuery) => {
  if (newQuery) {
    // 当有搜索查询时，自动展开所有有匹配 cookies 的浏览器
    const browsersWithMatches = filteredBrowsers.value.filter(browser => {
      return getFilteredCookies(browser).length > 0;
    });

    // 只展开有匹配结果的浏览器
    expandedBrowsers.value = [...browsersWithMatches];
  } else {
    // 清空搜索时，收起所有浏览器
    expandedBrowsers.value = [];
  }
});

onMounted(() => {
  if (props.show) {
    fetchCookies();
    initWebSocketHandlers();
  }
});

onUnmounted(() => {
  cleanupWebSocketHandlers();
});
</script>

<style scoped>
/* macOS 风格按钮 */
.btn-macos {
  @apply px-4 py-2 bg-base-200 hover:bg-base-300 text-base-content rounded-lg transition-all duration-200 text-sm font-medium border border-base-300 hover:border-gray-400;
}

.btn-macos-secondary {
  @apply px-4 py-2 bg-base-100 hover:bg-base-200 text-base-content rounded-lg transition-all duration-200 text-sm font-medium border border-base-300 hover:border-gray-400;
}

/* 同步按钮样式 */
.btn-sync {
  @apply px-3 py-1 text-xs font-medium rounded-md transition-all duration-200 border;
}

.btn-sync-primary {
  @apply bg-blue-50 text-blue-700 border-blue-200 hover:bg-blue-100 hover:border-blue-300;
}

.btn-sync-secondary {
  @apply bg-gray-50 text-gray-700 border-gray-200 hover:bg-gray-100 hover:border-gray-300;
}

/* macOS 风格输入框 */
.input-macos {
  @apply bg-base-100 border border-base-300 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 transition-all duration-200;
}

.input-macos[class*="pl-10"] {
  padding-left: 40px;
}

.input-macos[class*="pr-10"] {
  padding-right: 40px;
}

/* macOS 风格选择框 */
.select-macos {
  @apply bg-base-100 border border-base-300 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 transition-all duration-200 px-3 py-2;
}

/* 自定义滚动条 */
.cookies-scroll {
  scrollbar-width: thin;
  scrollbar-color: rgba(156, 163, 175, 0.3) transparent;
}

.cookies-scroll::-webkit-scrollbar {
  width: 6px;
}

.cookies-scroll::-webkit-scrollbar-track {
  background: transparent;
}

.cookies-scroll::-webkit-scrollbar-thumb {
  background-color: rgba(156, 163, 175, 0.3);
  border-radius: 3px;
}

.cookies-scroll::-webkit-scrollbar-thumb:hover {
  background-color: rgba(156, 163, 175, 0.5);
}
</style>