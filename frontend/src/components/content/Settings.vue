<template>
  <div class="settings-container rounded-tl-lg">
    <!-- Left menu -->
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
    
    <!-- Right Content -->
    <div class="h-full bg-base-200 settings-content">
      <component class="h-full p-2" :is="currentComponent" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import General from '@/components/content/General.vue'
import About from '@/components/content/About.vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const prefStore = usePreferencesStore()

const menuItems = computed(() => [
    { key: 'general', icon: 'ri-settings-3-line', label: t('settings.general.name') },
    { key: 'about', icon: 'ri-information-line', label: t('settings.about') }
])

// Current selected page
const currentPage = ref('general')
const handleCurrentPage = (key) => {
    currentPage.value = key
}

// Component mapping table
const componentMap = {
  'general': General,
  'about': About
}

// Calculate the component to display
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