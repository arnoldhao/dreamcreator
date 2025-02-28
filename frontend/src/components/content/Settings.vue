<template>
  <div class="settings-container rounded-tl-lg">
    <!-- 左侧菜单 -->
    <div class="settings-menu h-full bg-base-200 rounded-tl-lg border-r border-base-300">
        <div class="h-full p-2">
            <ul class="menu p-1 bg-base-100 h-full">
                <li v-for="item in menuItems" :key="item.key">
                    <a :class="[
                        'menu-item',
                        currentPage === item.key ? 'menu-item-active' : ''
                    ]" @click="handleCurrentPage(item.key)">
                        <v-icon :name="item.icon" class="w-5 h-5 shrink-0" :class="{'text-primary': currentPage === item.key, 'text-base-content': currentPage !== item.key}" />
                        <span class="flex-1">{{ item.label }}</span>
                        <span v-if="currentPage === item.key" class="badge badge-primary badge-xs"></span>
                    </a>
                </li>
            </ul>
        </div>
    </div>
    
    <!-- 右侧内容区 -->
    <div class="h-full bg-base-200 settings-content">
      <component class="h-full p-2" :is="currentComponent" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import ModelProvider from '@/components/content/ModelProvider.vue'
import ModelConfig from '@/components/content/ModelConfig.vue'
import General from '@/components/content/General.vue'
import Editor from '@/components/content/Editor.vue'
import About from '@/components/content/About.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const prefStore = usePreferencesStore()

const menuItems = computed(() => [
    // { key: 'model-provider', icon: 'ri-cloud-line', label: t('settings.model_provider') },
    // { key: 'model-config', icon: 'hi-cube-transparent', label: t('settings.model_config') },
    { key: 'general', icon: 'ri-settings-3-line', label: t('settings.general.name') },
    { key: 'editor', icon: 'md-editnote', label: t('settings.editor.name') },
    { key: 'about', icon: 'ri-information-line', label: t('settings.about') }
])

// 当前选中的页面
const currentPage = ref('general')
const handleCurrentPage = (key) => {
    currentPage.value = key
}

// 组件映射表
const componentMap = {
  // 'model-provider': ModelProvider,
  // 'model-config': ModelConfig,
  'general': General,
  'editor': Editor,
  'about': About
}

// 计算当前应该显示的组件
const currentComponent = computed(() => componentMap[currentPage.value])
</script>

<style scoped>
.settings-container {
  display: flex;
  height: 100%;
}

.settings-menu {
    height: 100%;
    width: 220px;
    min-width: 220px;
    flex-shrink: 0;
    overflow: hidden;
}

.settings-content {
  height: 100%;
  width: 500px;
  overflow: hidden;
  flex: 1;
  min-width: 500px;
}

.menu {
    @apply bg-transparent;
}

.menu-item {
    @apply text-sm rounded-btn py-2 px-2 mx-1 my-0.5;
}

.menu-item-active {
    @apply text-primary bg-primary/20 !important;

    :global(.dark) & {
        @apply bg-primary/10;
    }
}

.menu li:first-child a {
    @apply rounded-t-lg;
}

.menu li:last-child a {
    @apply rounded-b-lg;
}
</style>