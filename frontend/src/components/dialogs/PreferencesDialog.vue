<script setup>
import { ref, watchEffect, watch, h, computed } from 'vue'
import useDialog from 'stores/dialog'
import usePreferencesStore from 'stores/preferences.js'
import Help from '@/components/icons/Help.vue'
import { NButton, NIcon, NTooltip, NCard, NTag, NDataTable, NSwitch } from 'naive-ui'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { Project } from "@/consts/global.js";
import emitter from '@/utils/eventBus'
import { EMITTER_EVENTS } from '@/consts/emitter'
import useCommonStore from '@/stores/common'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { OpenDirectoryDialog } from 'wailsjs/go/systems/Service'

const { t } = useI18n()

const prefStore = usePreferencesStore()
// common store
const commonStore = useCommonStore()
const { testProxySites } = storeToRefs(commonStore)
watch(testProxySites, (newVal) => {
  console.log('testProxySites', newVal)
})

const prevPreferences = ref({})
const tab = ref('general')
const dialogStore = useDialog()
const loading = ref(false)

const initPreferences = async () => {
  try {
    loading.value = true
    tab.value = dialogStore.preferencesTag || 'general'
    await prefStore.loadPreferences()
    prevPreferences.value = {
      general: prefStore.general,
      editor: prefStore.editor,
      proxy: prefStore.proxy,
    }
  } finally {
    loading.value = false
  }
}

watchEffect(() => {
  if (dialogStore.preferencesDialogVisible) {
    initPreferences()
  }
})

const onOpenPrivacy = () => {
  let helpUrl = ''
  switch (prefStore.currentLanguage) {
    case 'zh':
      helpUrl = Project.HelpURLZh
      break
    default:
      helpUrl = Project.HelpURLEn
      break
  }
  BrowserOpenURL(helpUrl)
}

const onSavePreferences = async () => {
  const success = await prefStore.savePreferences()
  if (success) {
    // $message.success(i18n.t('dialogue.handle_succ'))
    dialogStore.closePreferencesDialog()
  }
}

const onClose = () => {
  // restore to old preferences
  prefStore.resetToLastPreferences()
  dialogStore.closePreferencesDialog()
}

const testProxyLoading = ref(false)
const onTestProxy = async () => {
  testProxyLoading.value = true
  const success = await prefStore.setProxy()
  if (!success) {
    return
  }

  // empty testProxySites
  commonStore.emptyTestProxySites()

  // emit
  emitter.emit(EMITTER_EVENTS.TEST_PROXY, {
    id: 'test_proxy_request_id', // any id will do
  })
}

// Add table column definitions
const proxyTestColumns = [
  {
    title: t('preferences.proxy.status'),
    key: 'status',
    width: 80,
    render: (row) => {
      return h(NTag, {
        type: row.success ? 'success' : 'error',
        size: 'small'
      }, {
        default: () => row.success ? '✓' : '✗'
      })
    }
  },
  {
    title: t('preferences.proxy.website'),
    key: 'url',
    ellipsis: true
  },
  {
    title: t('preferences.proxy.latency'),
    key: 'latency',
    width: 100,
    render: (row) => {
      return row.latency + 'ms'
    }
  }
]

// Optional: Custom switch style
const railStyle = ({ focused, checked }) => {
  const style = {}
  if (checked) {
    style.background = '#18a058'
  }
  return style
}


// Validate IP address format
const validateHost = (value) => {
  const regex = /^([0-9]{1,3}\.){3}[0-9]{1,3}$/
  return regex.test(value)
}

const isValidProxy = computed(() => {
  return validateHost(prefStore.proxy.addr) && prefStore.proxy.port > 0 && prefStore.proxy.port <= 65535
})

const changeDownloadDir = async() => {
  const { success, data, msg } = await OpenDirectoryDialog(t('preferences.download.change_download_dir'))
  if (success) {
    prefStore.updateDownloadDir(data.path)
  } else {
    $message.error(msg)
  }
}
</script>

<template>
  <n-modal v-model:show="dialogStore.preferencesDialogVisible" :auto-focus="false" :closable="false"
    :mask-closable="false" :show-icon="false" :title="$t('preferences.name')" close-on-esc preset="dialog"
    style="width: 640px" transform-origin="center" @esc="onClose">
    <!-- FIXME: set loading will slow down appear animation of dialog in linux -->
    <!-- <n-spin :show="loading"> -->
    <n-tabs v-model:value="tab" animated pane-style="min-height: 300px" placement="left"
      tab-style="justify-content: right; font-weight: 420;" type="line">
      <!-- general pane -->
      <n-tab-pane :tab="$t('preferences.general.name')" display-directive="show" name="general">
        <n-form :disabled="loading" :model="prefStore.general" :show-require-mark="false" label-placement="top">
          <n-grid :x-gap="10">
            <n-form-item-gi :label="$t('preferences.general.theme')" :span="24" required>
              <n-radio-group v-model:value="prefStore.general.theme" name="theme" size="medium">
                <n-radio-button v-for="opt in prefStore.themeOption" :key="opt.value" :value="opt.value">
                  {{ $t(opt.label) }}
                </n-radio-button>
              </n-radio-group>
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.general.language')" :span="24" required>
              <n-select v-model:value="prefStore.general.language" :options="prefStore.langOption"
                :render-label="({ label, value }) => (value === 'auto' ? $t(label) : label)" filterable />
            </n-form-item-gi>
            <n-form-item-gi :span="24" required>
              <template #label>
                {{ $t('preferences.general.font') }}
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <n-icon :component="Help" />
                  </template>
                  <div class="text-block">
                    {{ $t('preferences.font_tip') }}
                  </div>
                </n-tooltip>
              </template>
              <n-select v-model:value="prefStore.general.fontFamily" :options="prefStore.fontOption"
                :placeholder="$t('preferences.general.font_tip')"
                :render-label="({ label, value }) => (value === '' ? $t(label) : label)" filterable multiple tag />
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.general.font_size')" :span="24">
              <n-input-number v-model:value="prefStore.general.fontSize" :max="65535" :min="1" />
            </n-form-item-gi>
            <n-form-item-gi :span="12">
              <template #label>
                {{ $t('preferences.general.scan_size') }}
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <n-icon :component="Help" />
                  </template>
                  <div class="text-block">
                    {{ $t('preferences.general.scan_size_tip') }}
                  </div>
                </n-tooltip>
              </template>
              <n-input-number v-model:value="prefStore.general.scanSize" :min="1" :show-button="false"
                style="width: 100%" />
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.general.update')" :span="24">
              <n-checkbox v-model:checked="prefStore.general.checkUpdate">
                {{ $t('preferences.general.auto_check_update') }}
              </n-checkbox>
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.general.privacy')" :span="24">
              <n-checkbox v-model:checked="prefStore.general.allowTrack">
                {{ $t('preferences.general.allow_track') }}
                <n-button style="text-decoration: underline" text type="primary" @click="onOpenPrivacy">
                  {{ $t('preferences.general.privacy') }}
                </n-button>
              </n-checkbox>
            </n-form-item-gi>
          </n-grid>
        </n-form>
      </n-tab-pane>

      <!-- editor pane -->
      <n-tab-pane :tab="$t('preferences.editor.name')" display-directive="show" name="editor">
        <n-form :disabled="loading" :model="prefStore.editor" :show-require-mark="false" label-placement="top">
          <n-grid :x-gap="10">
            <n-form-item-gi :span="24" required>
              <template #label>
                {{ $t('preferences.general.font') }}
                <n-tooltip trigger="hover">
                  <template #trigger>
                    <n-icon :component="Help" />
                  </template>
                  <div class="text-block">
                    {{ $t('preferences.font_tip') }}
                  </div>
                </n-tooltip>
              </template>
              <n-select v-model:value="prefStore.editor.fontFamily" :options="prefStore.fontOption"
                :placeholder="$t('preferences.general.font_tip')"
                :render-label="({ label, value }) => value || $t(label)" filterable multiple tag />
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.general.font_size')" :span="24">
              <n-input-number v-model:value="prefStore.editor.fontSize" :max="65535" :min="1" />
            </n-form-item-gi>
            <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
              <n-checkbox v-model:checked="prefStore.editor.showLineNum">
                {{ $t('preferences.editor.show_linenum') }}
              </n-checkbox>
            </n-form-item-gi>
            <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
              <n-checkbox v-model:checked="prefStore.editor.showFolding">
                {{ $t('preferences.editor.show_folding') }}
              </n-checkbox>
            </n-form-item-gi>
            <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
              <n-checkbox v-model:checked="prefStore.editor.dropText">
                {{ $t('preferences.editor.drop_text') }}
              </n-checkbox>
            </n-form-item-gi>
            <n-form-item-gi :show-feedback="false" :show-label="false" :span="24">
              <n-checkbox v-model:checked="prefStore.editor.links">
                {{ $t('preferences.editor.links') }}
              </n-checkbox>
            </n-form-item-gi>
          </n-grid>
        </n-form>
      </n-tab-pane>

      <!-- proxy pane -->
      <n-tab-pane :tab="$t('preferences.proxy.name')" display-directive="show" name="proxy">
        <n-form :disabled="loading" :model="prefStore.proxy" :show-require-mark="false" label-placement="top">
          <n-grid :x-gap="10">
            <n-form-item-gi :label="$t('preferences.proxy.enabled_tip')" :span="24" required>
              <n-switch v-model:value="prefStore.proxy.enabled" :rail-style="railStyle" />
            </n-form-item-gi>
            <n-form-item-gi :label="$t('preferences.proxy.protocal')" :span="24" required>
              <n-radio-group v-model:value="prefStore.proxy.protocal" name="protocal" size="medium">
                <n-radio-button v-for="opt in prefStore.protocalOption" :key="opt.value" :value="opt.value">
                  {{ $t(opt.label) }}
                </n-radio-button>
              </n-radio-group>
            </n-form-item-gi>

            <n-form-item-gi :label="$t('preferences.proxy.addr')" :span="24" required>
              <n-input-group>
                <n-input v-model:value="prefStore.proxy.addr" :placeholder="'127.0.0.1'" style="width: 140px"
                  @input="validateHost" />
                <n-input-group-label>:</n-input-group-label>
                <n-input-number v-model:value="prefStore.proxy.port" :placeholder="'7890'" :min="1" :max="65535"
                  :show-button="false" style="width: 100px" />

                <n-button @click="onTestProxy" :disabled="!isValidProxy">
                  {{ $t('preferences.proxy.test') }}
                </n-button>
              </n-input-group>
            </n-form-item-gi>

            <n-form-item-gi :span="24" v-if="testProxyLoading">
              <n-card size="small" class="proxy-test-results">
                <template #header>
                  {{ $t('preferences.proxy.test_results') }}
                </template>
                <n-data-table :columns="proxyTestColumns" :data="testProxySites" :bordered="false" size="small" />
              </n-card>
            </n-form-item-gi>

          </n-grid>
        </n-form>
      </n-tab-pane>

      <!-- download pane -->
      <n-tab-pane :tab="$t('preferences.download.name')" display-directive="show" name="download">
        <n-form :disabled="loading" :model="prefStore.download" :show-require-mark="false" label-placement="top">
          <n-grid :x-gap="12">
            <n-form-item-gi :label="$t('preferences.download.download_dir')" :span="24">
              <n-input-group>
                <n-input v-model:value="prefStore.download.dir" :placeholder="$t('preferences.download.download_dir')" />
                <n-button @click="changeDownloadDir">
                  {{ $t('preferences.download.change_download_dir') }}
                </n-button>
              </n-input-group>
            </n-form-item-gi>
          </n-grid>
        </n-form>
      </n-tab-pane>
    </n-tabs>
    <!-- </n-spin> -->

    <template #action>
      <div class="flex-item-expand">
        <n-button :disabled="loading" @click="prefStore.restorePreferences">
          {{ $t('preferences.restore_defaults') }}
        </n-button>
      </div>
      <div class="flex-item n-dialog__action">
        <n-button :disabled="loading" @click="onClose">{{ $t('common.cancel') }}</n-button>
        <n-button :disabled="loading" type="primary" @click="onSavePreferences">
          {{ $t('common.save') }}
        </n-button>
      </div>
    </template>
  </n-modal>
</template>

<style lang="scss" scoped>
.inline-form-item {
  padding-right: 10px;
}

.proxy-test-results {
  margin-top: 8px;
}

:deep(.n-input-group-label) {
  padding: 0 8px;
}
</style>
