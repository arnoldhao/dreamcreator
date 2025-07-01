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
          <button @click="syncCookies" :disabled="isLoading" class="btn-macos">
            <v-icon name="ri-refresh-line" class="w-4 h-4 mr-2" :class="{ 'animate-spin': isLoading }"></v-icon>
            {{ isLoading ? $t('common.loading') : $t('common.sync') }}
          </button>
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
          <!-- 搜索框 - 修复padding问题 -->
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
          <div class="space-y-3 max-h-96 overflow-y-auto cookies-scroll">
            <div v-for="browser in filteredBrowsers" :key="browser"
              class="bg-base-100 border border-base-300 rounded-lg overflow-hidden hover:shadow-sm transition-all duration-200">
              <!-- 浏览器头部 -->
              <div
                class="flex items-center justify-between p-4 bg-base-200/20 border-b border-base-300 cursor-pointer hover:bg-base-200/40 transition-colors"
                @click="toggleBrowser(browser)">
                <div class="flex items-center space-x-3">
                  <div class="w-8 h-8 bg-primary/10 rounded-lg flex items-center justify-center">
                    <v-icon :name="getBrowserIcon(browser)" class="w-5 h-5 text-primary"></v-icon>
                  </div>
                  <div>
                    <h4 class="text-sm font-medium text-base-content">{{ browser }}</h4>
                    <p class="text-xs text-base-content/60">
                      {{ getBrowserCookieCount(browser) }} {{ $t('cookies.cookies') }}
                    </p>
                    <!-- 新增：显示上次同步时间 -->
                    <p class="text-xs text-base-content/40" v-if="getLastSyncTime(browser)">
                      {{ $t('cookies.last_sync') }}: {{ formatSyncTime(getLastSyncTime(browser)) }}
                    </p>
                  </div>
                </div>
                <svg class="w-5 h-5 text-base-content/40 transition-transform duration-200"
                  :class="{ 'rotate-180': expandedBrowsers.includes(browser) }" fill="none" stroke="currentColor"
                  viewBox="0 0 24 24">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"></path>
                </svg>
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
import { ref, watch, onMounted, computed } from 'vue';
import { useI18n } from 'vue-i18n';
import { ListAllCookies, SyncCookies } from 'wailsjs/go/api/CookiesAPI';

const props = defineProps({
  show: Boolean,
});

const emit = defineEmits(['update:show']);

const { t } = useI18n();

const isLoading = ref(false);
const browsers = ref([]);
const cookiesByBrowser = ref({});
const searchQuery = ref('');
const selectedBrowser = ref('');
const expandedBrowsers = ref([]);

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
  // 1. 检查 browserData 和正确的字段名 domain_cookies 是否存在
  if (!browserData || !browserData.domain_cookies) {
    return 0;
  } else {
    // 2. 遍历所有域名，并累加每个域名下的 Cookie 数量
    return Object.values(browserData.domain_cookies).reduce((total, domain) => {
      // 3. 确保 domain 对象和其下的 Cookies 数组存在
      return total + (domain && Array.isArray(domain.cookies) ? domain.cookies.length : 0);
    }, 0);
  }
};

// 同样修复getFilteredCookies方法
const getFilteredCookies = (browser) => {
  const browserData = cookiesByBrowser.value[browser];
  if (!browserData) return [];

  let allCookies = [];

  if (browserData.domain_cookies) {
    // domain_cookies -> domain -> cookies
    Object.values(browserData.domain_cookies).forEach(domain => {
      if (domain.cookies && Array.isArray(domain.cookies)) {
        allCookies = allCookies.concat(domain.cookies);
      }
    });
  } else if (Array.isArray(browserData)) {
    // 直接是cookies数组
    allCookies = browserData;
  } else if (typeof browserData === 'object') {
    // 尝试从对象中提取cookies
    Object.values(browserData).forEach(domain => {
      if (domain && domain.cookies && Array.isArray(domain.cookies)) {
        allCookies = allCookies.concat(domain.cookies);
      } else if (Array.isArray(domain)) {
        allCookies = allCookies.concat(domain);
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
      // 解析JSON数据，得到 map[string]*BrowserCookies 结构
      const cookiesData = JSON.parse(res.data || '{}') || {};
      cookiesByBrowser.value = cookiesData;
      // 从cookiesData的键中提取浏览器名称
      browsers.value = Object.keys(cookiesData);
      // 默认展开第一个浏览器
      if (browsers.value.length > 0 && expandedBrowsers.value.length === 0) {
        expandedBrowsers.value.push(browsers.value[0]);
      }
    } else {
      throw new Error(res.msg);
    }
  } catch (error) {
    console.error('Fetch cookies error:', error);
  } finally {
    isLoading.value = false;
  }
};

const syncCookies = async () => {
  isLoading.value = true;
  try {
    const res = await SyncCookies();
    if (res.success) {
      // 解析JSON数据，得到 map[string]*BrowserCookies 结构
      const cookiesData = JSON.parse(res.data || '{}') || {};
      cookiesByBrowser.value = cookiesData;
      // 从cookiesData的键中提取浏览器名称
      browsers.value = Object.keys(cookiesData);
    } else {
      throw new Error(res.msg);
    }
  } catch (error) {
    console.error('Sync cookies error:', error);
  } finally {
    isLoading.value = false;
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

watch(() => props.show, (newVal) => {
  if (newVal) {
    fetchCookies();
    // 重置搜索和过滤状态
    searchQuery.value = '';
    selectedBrowser.value = '';
    expandedBrowsers.value = [];
  }
});

onMounted(() => {
  if (props.show) {
    fetchCookies();
  }
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

/* macOS 风格输入框 */
.input-macos {
  @apply bg-base-100 border border-base-300 rounded-lg focus:outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 transition-all duration-200;
}

/* 确保搜索框内的元素不重叠 */
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