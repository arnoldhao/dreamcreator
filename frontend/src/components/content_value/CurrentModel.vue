<template>
  <n-card :bordered="false" embedded>
    <template #header>
      <n-button text>
        <template #icon>
          <n-icon class="icon-size">
            <AIIcon />
          </n-icon>
        </template>
        <n-text strong class="title">{{ $t('ai.current_model') }}</n-text>
      </n-button>
    </template>

    <template #header-extra>
      <n-tooltip trigger="hover">
        <template #trigger>
          <n-button @click="openModelSelection" circle size="small" class="refresh-button">
            <n-icon>
          <ConfigIcon />
        </n-icon>
          </n-button>
        </template>
        {{ $t('ai.change_model') }}
      </n-tooltip>
    </template>

    <template v-if="currentModel.msg" #default>
      <n-tooltip trigger="hover">
        <template #trigger>
          <n-button text>
            <n-text strong class="current-model">{{ currentModel.msg }}</n-text>
          </n-button>
        </template>
        {{ $t('ai.click_to_change_model') }}
      </n-tooltip>
    </template>
    <template v-else #default>
      <n-button @click="openModelSelection" text>
        <n-space align="center">
          <n-icon class="icon-size">
            <ModelIcon />
          </n-icon>
          <n-text strong class="current-model">
            {{ capitalizeFirstLetter(currentModel.llmName) }} -> {{ currentModel.modelName }}
          </n-text>
        </n-space>
      </n-button>
      <ModelModal v-model:show="showModelModal" />
    </template>
  </n-card>
</template>

<script setup>
import { computed, ref } from 'vue'
import { NCard, NButton, NText, NSpace, NIcon } from 'naive-ui'
import AIIcon from '@/components/icons/AI.vue'
import ConfigIcon from '@/components/icons/Config.vue'
import ModelIcon from '@/components/icons/Model.vue';
import ModelModal from '@/components/modal/ModelModal.vue'
import useLLMTabStore from "stores/llmtab.js";
const llmtabStore = useLLMTabStore()
const currentModel = computed(() => llmtabStore.currentModel)
const showModelModal = ref(false)
const openModelSelection = () => {
  showModelModal.value = true
}
const capitalizeFirstLetter = (string) => {
  // if string is empty, return original string
  if (!string) return string
  return string.charAt(0).toUpperCase() + string.slice(1);
}

</script>

<style scoped>
.icon-size {
  font-size: 20px;
}

.title {
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
</style>