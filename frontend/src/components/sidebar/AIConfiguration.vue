<template>
  <div class="ai-configuration-container">
    <!-- current model area-->
    <CurrentModel />

    <n-list hoverable clickable show-divider>
      <template #header>
        <n-row>
          <n-col :span="12">
            <n-statistic :label="$t('ai.total_llms')">
              <template #prefix>
                <n-icon>
                  <OpenAIcon />
                </n-icon>
              </template>
              {{ totalLLMs }}
            </n-statistic>
          </n-col>
          <n-col :span="12">
            <n-statistic :label="$t('ai.total_models')">
              <template #prefix>
                <n-icon>
                  <ModelIcon />
                </n-icon>
              </template>
              {{ totalModels }}
            </n-statistic>
          </n-col>
        </n-row>
      </template>

      <template #default  v-if="llms.length > 0">
        <n-list-item v-for="(llm) in llms" :key="llm.num">
          <n-thing content-style="margin-top: 10px;" @click="editLLM(llm.name)">
            <template #header>
              <n-space align="center">
                <component :is="getIconComponent(llm.icon)" class="llm-icon" />
                <n-text strong>{{ capitalizeFirstLetter(llm.name) }}</n-text>
              </n-space>
            </template>
            <template #description>
              <n-space size="small" style="margin-top: 4px" v-if="llm.name !== 'ollama'">
                <n-tag :bordered="true" :type="llm.available ? 'success' : 'warning'" size="small">
                  {{ $t('ai.models') }}: {{ llm.models.length }}
                  <template #icon v-if="llm.available">
                    <n-icon :component="CheckmarkCircle" />
                  </template>
                </n-tag>
                <n-tag :bordered="true" type="info" size="small">
                  {{ llm.region }}
                </n-tag>
              </n-space>
            </template>
          </n-thing>
          <template #suffix v-if="llm.name !== 'ollama'">
            <n-button @click="deleteLLMConfirm(llm.name)" type="warning">{{ $t('ai.delete_llm_confirm') }}</n-button>
          </template>
        </n-list-item>
      </template>
      <template #footer>
        <n-space justify="center">
          <n-button @click="restoreAIsConfirm">
            <n-icon>
              <AIIcon />
            </n-icon>
            {{ $t('ai.restore_llm') }}</n-button>
          <n-button @click="addLLM">
            <n-icon>
              <AIIcon />
            </n-icon>
            {{ $t('ai.add_llm') }}</n-button>
        </n-space>

      </template>
    </n-list>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import AIIcon from '@/components/icons/AI.vue'
import OpenAIcon from '@/components/icons/OpenAI.vue';
import ModelIcon from '@/components/icons/Model.vue';
import { useI18n } from 'vue-i18n'
import { CheckmarkCircle } from "@vicons/ionicons5";
import { useDialog } from 'naive-ui';
import OllamaIcon from '@/components/icons/Ollama.vue';
import CurrentModel from '@/components/content_value/CurrentModel.vue'
import useLLMTabStore from "stores/llmtab.js";

const { t } = useI18n()
const dialog = useDialog()
const llmtabStore = useLLMTabStore()
const llms = computed(() => llmtabStore.llms)
const totalLLMs = computed(() => llms.value?.length ?? 0)
const totalModels = computed(() => llms.value?.reduce((total, llm) => total + (llm.models?.length ?? 0), 0) ?? 0)

const capitalizeFirstLetter = (string) => {
  // if string is null, return string
  if (!string) return string
  return string.charAt(0).toUpperCase() + string.slice(1);
}

function addLLM() {
  llmtabStore.newLLm()
}

async function editLLM(llmName) {
  llmtabStore.editLLm(llmName)
}

function deleteLLMConfirm(llmName) {
  dialog.warning({
    title: t('ai.delete_llm_title'),
    content: t('ai.delete_llm_content'),
    positiveText: t('common.delete'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => {
      deleteLLM(llmName)
    }
  })
}

async function deleteLLM(llmName) {
  llmtabStore.deleteLLM(llmName)
}

const getIconComponent = (iconName) => {
  switch (iconName) {
    case 'open-like':
      return OpenAIcon;
    case 'ollama':
      return OllamaIcon;
    default:
      return null;
  }
};

async function restoreAIsConfirm() {
  dialog.warning({
    title: t('ai.restore_title'),
    content: t('ai.restore_content'),
    positiveText: t('ai.restore_confirm'),
    negativeText: t('ai.restore_cancel'),
    onPositiveClick: () => {
      restoreAIs()
    }
  })
}

async function restoreAIs() {
  await llmtabStore.restoreAIs()
}
</script>

<style scoped>
.ai-configuration-container {
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

.llm-icon {
  width: 24px;
  height: 24px;
}

.icon-size {
  font-size: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
}
</style>