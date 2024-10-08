<script setup>
import PreferencesDialog from './components/dialogs/PreferencesDialog.vue'
import AboutDialog from '@/components/dialogs/AboutDialog.vue'
import AppContent from './AppContent.vue'
import { h, onMounted, ref, watch } from 'vue'
import usePreferencesStore from './stores/preferences.js'
import { useI18n } from 'vue-i18n'
import { darkTheme, NButton, NSpace } from 'naive-ui'
import { Environment, WindowSetDarkTheme, WindowSetLightTheme } from 'wailsjs/runtime/runtime.js'
import { darkThemeOverrides, themeOverrides } from '@/utils/theme.js'
import hljs from "highlight.js/lib/core"
import { useTranslationEventHandlers } from '@/handlers/translation.js'
import { useOllamaEventHandlers } from '@/handlers/ollama.js'
import useLLMTabStore from '@/stores/llmtab.js'
const { initEventHandlers } = useTranslationEventHandlers()
const { initOllamaEventHandlers } = useOllamaEventHandlers()
const prefStore = usePreferencesStore()
const i18n = useI18n()
const initializing = ref(true)
const llmStore = useLLMTabStore()
onMounted(async () => {
  try {
    initializing.value = true
    await prefStore.loadFontList()
    if (prefStore.autoCheckUpdate) {
      prefStore.checkForUpdate()
    }
    const env = await Environment()

    // show greetings and user behavior tracking statements
    // if (!!!prefStore.behavior.welcomed) {
    //   const n = $notification.show({
    //     title: () => i18n.t('dialogue.welcome.title'),
    //     content: () => i18n.t('dialogue.welcome.content'),
    //     // duration: 5000,
    //     keepAliveOnHover: true,
    //     closable: false,
    //     meta: ' ',
    //     action: () =>
    //       h(
    //         NSpace,
    //         {},
    //         {
    //           default: () => [
    //             h(
    //               NButton,
    //               {
    //                 secondary: true,
    //                 type: 'tertiary',
    //                 onClick: () => {
    //                   prefStore.setAsWelcomed(false)
    //                   n.destroy()
    //                 },
    //               },
    //               {
    //                 default: () => i18n.t('dialogue.welcome.reject'),
    //               },
    //             ),
    //             h(
    //               NButton,
    //               {
    //                 secondary: true,
    //                 type: 'primary',
    //                 onClick: () => {
    //                   prefStore.setAsWelcomed(true)
    //                   n.destroy()
    //                 },
    //               },
    //               {
    //                 default: () => i18n.t('dialogue.welcome.accept'),
    //               },
    //             ),
    //           ],
    //         },
    //       ),
    //   })
    // }
    // initialize subtitle event handlers
    initEventHandlers()
    // initialize ollama event handlers
    initOllamaEventHandlers()
    // initialize LLM
    llmStore.initialize(true)
  } finally {
    initializing.value = false
  }
})

// watch theme and dynamically switch
watch(
  () => prefStore.isDark,
  (isDark) => (isDark ? WindowSetDarkTheme() : WindowSetLightTheme()),
)

// watch language and dynamically switch
watch(
  () => prefStore.general.language,
  (lang) => (i18n.locale.value = prefStore.currentLanguage),
)
</script>

<template>
  <n-config-provider :inline-theme-disabled="true" :locale="prefStore.themeLocale"
    :theme="prefStore.isDark ? darkTheme : undefined"
    :theme-overrides="prefStore.isDark ? darkThemeOverrides : themeOverrides" :hljs="hljs" class="fill-height">
    <n-dialog-provider>
      <n-modal-provider>
        <app-content :loading="initializing" />
     

      <!-- top modal dialogs -->
      <preferences-dialog />
      <about-dialog />
    </n-modal-provider>
    </n-dialog-provider>
  </n-config-provider>
</template>

<style lang="scss"></style>
