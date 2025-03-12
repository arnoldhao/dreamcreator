<template>
  <div class="menu h-full w-[48px] flex flex-col justify-between py-3" 
    :class="prefStore.isDark ? 'bg-neutral-focus' : 'bg-base-100'"
    :style="{
      width: '48px',
      minWidth: '48px',
    }">
    <!-- Main menu items -->
    <div class="menu-items flex flex-col items-center pt-1 space-y-3">
      <div v-for="(m, i) in navStore.menuOptions" :key="i" class="tooltip tooltip-right w-full" :data-tip="$t(m.label)">
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
      <div v-for="(m, i) in navStore.bottomMenuOptions" :key="i" class="tooltip tooltip-right w-full" :data-tip="$t(m.label)">
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
import { computed, watch } from 'vue'
import usePreferencesStore from 'stores/preferences.js'
import useNavStore from 'stores/nav.js'

const prefStore = usePreferencesStore()
const navStore = useNavStore()

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
    default: 'download',
  },
  width: {
    type: Number,
    default: 60,
  },
})

const emit = defineEmits(['update:value'])

// 监听主题变化，更新主题图标
watch(() => prefStore.isDark, (isDark) => {
  navStore.updateThemeIcon(isDark)
}, { immediate: true })
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
