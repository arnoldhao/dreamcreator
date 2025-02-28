<template>
    <div class="optimize-container rounded-tl-lg">
        <!-- 左侧菜单 -->
        <div class="optimize-menu h-full bg-base-200 rounded-tl-lg border-r border-base-300">
            <div class="h-full p-2">
                <ul class="menu p-1 bg-base-100 h-full">
                    <li v-for="item in menuItems" :key="item.key">
                        <a :class="[
                            'menu-item',
                            currentPage === item.key ? 'menu-item-active' : ''
                        ]" @click="handleCurrentPage(item.key)">
                            <v-icon :name="item.icon" class="w-5 h-5 shrink-0"
                                :class="{ 'text-primary': currentPage === item.key, 'text-base-content': currentPage !== item.key }" />
                            <span class="flex-1">{{ item.label }}</span>
                            <span v-if="currentPage === item.key" class="badge badge-primary badge-xs"></span>
                        </a>
                    </li>
                </ul>
            </div>
        </div>

        <!-- 右侧内容区 -->
        <div class="h-full bg-base-200 optimize-content">
            <div v-if="currentPage === 'subtitle'">
                <resizeable-wrapper v-model:size="prefStore.behavior.asideWidth" :min-size="300" :offset="50"
                    :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'" @update:size="handleResize">
                    <subtitle-command class="h-full" />
                </resizeable-wrapper>
                <subtitle-content v-for="t in tabStore.tabs" v-show="tabStore.currentTabId === t.id" :key="t.id"
                    :name="t.id" :title="t.title" class="flex-1" />
            </div>
            <div v-else-if="currentPage === 'ai'">
                <resizeable-wrapper v-model:size="prefStore.behavior.asideWidth" :min-size="300" :offset="50"
                    class="bg-base-200" @update:size="handleResize">
                    <ai-configuration-sidebar class="h-full" />
                </resizeable-wrapper>
                <LLMConfiguration class="flex-1" />
            </div>
            <div v-else class="h-full p-2">
                <component :is="currentComponent" />
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { debounce } from 'lodash'
import SubtitleCommand from "@/components/sidebar/SubtitleCommand.vue";
import SubtitleContent from '@/components/content/SubtitleContent.vue';
import AiConfigurationSidebar from "@/components/sidebar/AIConfiguration.vue";
import LLMConfiguration from "@/components/content/LLMConfiguration.vue";
import { useI18n } from 'vue-i18n'
import usePreferencesStore from '@/stores/preferences.js'
import useSuperTabStore from '@/stores/supertab.js'
import ResizeableWrapper from '@/components/common/ResizeableWrapper.vue';

const { t } = useI18n()
const prefStore = usePreferencesStore()
const tabStore = useSuperTabStore()
const menuItems = computed(() => [
    { key: 'subtitle', icon: 'ri-history-line', label: t('optimize.subtitle') },
    { key: 'ai', icon: 'ri-history-line', label: t('optimize.ai') },
])

// 当前选中的页面
const currentPage = ref('subtitle')
const handleCurrentPage = (key) => {
    currentPage.value = key
}

// 组件映射表
const componentMap = {
    "subtitle": null,
    "ai": null,
}

// 计算当前应该显示的组件
const currentComponent = computed(() => componentMap[currentPage.value])

const saveSidebarWidth = debounce(prefStore.savePreferences, 1000, { trailing: true })
const handleResize = () => {
    saveSidebarWidth()
}
</script>

<style scoped>
.optimize-container {
    display: flex;
    height: 100%;
}

.optimize-menu {
    height: 100%;
    width: 220px;
    min-width: 220px;
    flex-shrink: 0;
    overflow: hidden;
}

.optimize-content {
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