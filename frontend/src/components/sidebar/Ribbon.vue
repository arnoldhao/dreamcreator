<template>
  <div class="menu h-full w-[48px] flex flex-col justify-between py-3" 
    :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'"
    :style="{
      width: '48px',
      minWidth: '48px',
    }">
    <!-- Main menu items -->
    <div class="menu-items flex flex-col items-center pt-1 space-y-3">
      <div v-for="(m, i) in menuOptions" :key="i" class="tooltip tooltip-right w-full" :data-tip="$t(m.label)">
        <div
          class="w-full h-9 flex items-center justify-center relative hover:bg-primary/10 rounded-full cursor-pointer"
          :class="[
            { 'active-item': props.value === m.key },
            prefStore.isDark ? 'hover:bg-neutral' : 'hover:bg-base-200'
          ]"
          @click="emit('update:value', m.key)">
          <v-icon :name="m.icon" class="w-5 h-5 transition-transform hover:scale-110"
            :class="{ 'text-primary': props.value === m.key }" scale="1" />
        </div>
      </div>
    </div>

    <!-- Bottom menu items -->
    <div class="menu-items flex flex-col items-center pb-1 space-y-3">
      <div v-for="(m, i) in bottomMenuOptions" :key="i" class="tooltip tooltip-right w-full" :data-tip="$t(m.label)">
        <div
          class="w-full h-9 flex items-center justify-center relative hover:bg-primary/10 rounded-full cursor-pointer"
          :class="[
            { 'active-item': props.value === m.key },
            prefStore.isDark ? 'hover:bg-neutral' : 'hover:bg-base-200'
          ]"
          @click="m.key === 'theme' ? handleThemeClick() : emit('update:value', m.key)">
          <v-icon :name="m.icon" class="w-5 h-5 transition-transform hover:scale-110"
            :class="{ 'text-primary': props.value === m.key }" scale="1" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import usePreferencesStore from 'stores/preferences.js'

const prefStore = usePreferencesStore()

const handleThemeClick = () => {
  // 切换主题
  const currentTheme = prefStore.general.theme
  const newTheme = currentTheme === 'dark' ? 'light' : 'dark'
  prefStore.general.theme = newTheme
  prefStore.savePreferences()
}

const props = defineProps({
  value: {
    type: String,
    default: 'optimize',
  },
  width: {
    type: Number,
    default: 60,
  },
})

const emit = defineEmits(['update:value'])

const menuOptions = computed(() => {
  return [
    {
      label: 'ribbon.download',
      key: 'download',
      icon: 'ri-download-cloud-line',
    },
    {
      label: 'ribbon.history',
      key: 'history',
      icon: 'ri-history-line',
    },
    {
      label: 'ribbon.optimize',
      key: 'optimize',
      icon: 'oi-rocket',
    },
    // {
    //   label: 'menu.subtitle',
    //   key: 'subtitle',
    //   icon: 'md-subtitles',
    // },
    // {
    //   label: 'menu.ai',
    //   key: 'ai',
    //   icon: 'la-robot-solid',
    // },
  ]
})

const themeIcon = computed(() => {
  return prefStore.isDark === true ? 'ri-moon-line' : 'ri-sun-line'
})

const bottomMenuOptions = computed(() => {
  return [
    {
      label: 'bottom.theme',
      key: 'theme',
      icon: themeIcon.value,
    },
    {
      label: 'bottom.settings',
      key: 'settings',
      icon: 'ri-settings-3-line',
    },
  ]
})
</script>

<style lang="scss" scoped>
.menu {
  height: 100%;
}

.active-item {
  @apply text-primary bg-primary/20;

  :global(.dark) & {
    @apply bg-primary/10;
  }
}
</style>
