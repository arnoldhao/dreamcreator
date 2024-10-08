<template>
  <div class="page-container">
    <!-- Model area-->
    <CurrentModel />

    <!-- Current Task area-->
    <div v-if="!isBlank">
      <CurrentTask :is-blank="isBlank" :tab-id="tabId" :tab-title="tabTitle" :language="language" :is-stream="isStream"
        :is-canceled="isCanceled" :translation-progress="safeTranslationProgress" :is-translating="isTranslating"
        :has-error="hasError" :is-completed="isCompleted" :action-description="actionDescription"
        :is-pending="isPending" :allow-action="allowAction" @switch-tab="switchToTab"
        @show-edit-title-dialog="showEditTitleDialog" @cancel-confirm-dialog="cancelConfirmDialog" />
    </div>

    <!-- AI Translate area-->
    <div v-if="!isBlank">
      <AITranslate :originalSubtitleId="tabId" :originalSubtitleTitle="tabTitle" :originalSubtitleLang="language"
        :allowAction="allowAction" />
    </div>


    <!-- Tasks area-->
    <n-list :bordered="true" clickable embedded>
      <!-- title-->
      <template #header>
        <n-button text>
          <template #icon>
            <n-icon class="icon-size">
              <TasksIcon />
            </n-icon>
          </template>
          <n-text strong class="page-title">{{ $t('task.tasks') }}</n-text>
        </n-button>
      </template>
      <!-- tasks -->
      <n-list-item v-for="tab in streamingTabs" :key="tab.id" @click="switchToTab(tab.id)">
        <template #suffix>
          <!-- action -->
          <n-tooltip trigger="hover">
            <template #trigger>
              <n-button @click.stop="cancelConfirmDialog(tab.id)"
                :disabled="tab.getTranslationState().translationStatus !== 'running'" quaternary circle type="warning">
                <template #icon>
                  <CancelIcon />
                </template>
              </n-button>
            </template>
            <n-text>{{ $t('task.cancel_task') }}</n-text>
          </n-tooltip>
        </template>
        <!-- progress -->
        <n-thing :title="tab.title" class="file-name">
          <n-progress type="line" :percentage="Number(tab.getTranslationState().translationProgress)"
            :processing="tab.getTranslationState().translationStatus === 'running'" :status="getProgressStatus(tab)">
          </n-progress>
        </n-thing>

        <!-- action description -->
        <n-text v-if="tab.getTranslationState().actionDescription" :type="getTextType(tab)">
          {{ tab.getTranslationState().actionDescription }}
        </n-text>
      </n-list-item>

      <!-- history -->
      <template #footer>
        <n-space justify="center">
          <n-button @click="switchToNav('history')">
            <n-icon>
              <HistoryIcon />
            </n-icon>
            {{ $t('task.open_history') }}</n-button>
        </n-space>
      </template>
    </n-list>

    <!-- edit title modal -->
    <EditTitleModal v-model:show="showEditModal" :id="currentEditId" :title="currentTitle" />
  </div>


</template>

<script setup>
import { computed, ref, onMounted } from 'vue'
import TasksIcon from '@/components/icons/Tasks.vue'
import HistoryIcon from '@/components/icons/History.vue';
import CancelIcon from '@/components/icons/Cancel.vue'
import useSuperTabStore from "stores/supertab.js";
import CurrentModel from '@/components/content_value/CurrentModel.vue'
import CurrentTask from '@/components/content_value/CurrentTask.vue'
import AITranslate from '@/components/content_value/AITranslate.vue'
import EditTitleModal from '@/components/modal/EditTitleModal.vue'
import { useI18n } from 'vue-i18n'
import { GetLanguage } from 'wailsjs/go/languages/Service'
import { CancelTranslation } from 'wailsjs/go/trans/WorkQueue'
import { storeToRefs } from 'pinia'
import { useDialog, NButton } from 'naive-ui'

const dialog = useDialog()
const { t } = useI18n()
const tabStore = useSuperTabStore()
const { tabs, currentTab } = storeToRefs(tabStore)
const isBlank = computed(() => currentTab.value?.blank ?? false)
const tabId = computed(() => currentTab.value?.id || '')
const tabTitle = computed(() => currentTab.value?.title || '')
const language = computed(() => currentTab.value?.language || 'Original')
const isStream = computed(() => currentTab.value?.stream || false)
const translationState = computed(() => currentTab.value?.getTranslationState() || {})
const streamStatus = computed(() => translationState.value.streamStatus)
const translationStatus = computed(() => translationState.value.translationStatus)
const translationProgress = computed(() => translationState.value.translationProgress)
const actionDescription = computed(() => translationState.value.actionDescription)

const isPending = computed(() =>
  streamStatus.value === 'streaming' && translationStatus.value === 'pending'
)
const isTranslating = computed(() =>
  streamStatus.value === 'streaming' && translationStatus.value === 'running'
)
const isCompleted = computed(() =>
  streamStatus.value === 'translation_done' && translationStatus.value === 'completed'
)
const isCanceled = computed(() =>
  streamStatus.value === 'canceled' || translationStatus.value === 'canceled'
)
const hasError = computed(() =>
  streamStatus.value === 'error' || translationStatus.value === 'error'
)

const streamingTabs = computed(() =>
  tabs.value.filter(tab => tab.stream === true)
)

const allowAction = computed(() => !isStream.value || isCompleted.value || isCanceled.value)

const safeTranslationProgress = computed(() => {
  const progress = translationProgress.value;
  if (typeof progress === 'number' && !isNaN(progress)) {
    return progress;
  }
  if (typeof progress === 'string') {
    const parsed = parseFloat(progress);
    return isNaN(parsed) ? 0 : parsed;
  }
  return 0;
});


function switchToTab(id) {
  tabStore.switchTab(id)
}

function showEditTitleDialog(id, title) {
  currentEditId.value = id
  currentTitle.value = title
  showEditModal.value = true
}

function cancelConfirmDialog(id) {
  dialog.warning({
    title: t('language.confirm_operation'),
    content: t('language.confirm_cancel_translation'),
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      processCancelTranslation(id)
    },
    onNegativeClick: () => {
      $message.info(t('language.operation_cancelled'))
    }
  })
}

function getProgressStatus(tab) {
  const state = tab.getTranslationState()
  if (state.streamStatus === 'error' || state.translationStatus === 'error') return 'error'
  if (state.streamStatus === 'translation_done' && state.translationStatus === 'completed') return 'success'
  return 'default'
}

function getTextType(tab) {
  const state = tab.getTranslationState()
  return (state.streamStatus === 'error' || state.translationStatus === 'error') ? 'error' : 'default'
}


async function processCancelTranslation(id) {
  try {
    const { success, msg } = await CancelTranslation(id)
    if (success) {
      $message.success(t('language.cancel_translation_success'))
      tabStore.updateTabTranslationState(id, {
        streamStatus: 'canceled',
        translationStatus: 'canceled',
        actionDescription: 'cancel process...'
      })
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('language.cancel_translation_error') + error.message);
  }
}

const TransLangs = ref([])

async function getTransLangs() {
  const { data, success, msg } = await GetLanguage()
  if (success) {
    try {
      const parsedData = JSON.parse(data)
      if (Array.isArray(parsedData.langs)) {
        TransLangs.value = parsedData.langs
      } else {
        throw new Error(t('language.parse_trans_langs_error') + parsedData)
      }
    } catch (error) {
      throw error
    }
  } else {
    $message.error(t('language.get_trans_langs_error') + msg)
  }
}

onMounted(async () => {
  await getTransLangs()
})


function switchToNav(nav) {
  tabStore.nav = nav
}

// edit title
const showEditModal = ref(false)
const currentTitle = ref('')
const currentEditId = ref(null)
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow-y: auto;
}

.page-title {
  font-weight: bold;
  font-size: 1.1em;
  font-style: italic;
}

.current-model {
  font-weight: bold;
  font-size: 1em;
  font-style: italic;
  display: flex;
  align-items: center;
}

.file-name :deep(.n-thing-header__title) {
  font-size: 14px !important;
  font-style: italic;
  font-weight: bold !important;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 200px;
}

@media screen and (max-width: 768px) {
  .custom-title :deep(.n-thing-header__title) {
    font-size: 12px;
  }
}
</style>