<template>
  <div v-if="blankLLM" class="empty-card">
    <!-- blank page -->
    <n-empty size="huge" description="LLM Configuration">
    <template #icon>
      <n-icon>
        <AIIcon />
      </n-icon>
    </template>
    <template #extra>
      <n-text>
        {{ $t('ai.llm_description') }}
      </n-text>
      </template>
    </n-empty>
  </div>

  <!-- ollama page -->
  <div v-else-if="isOllama">
    <OllamaPage />
  </div>

  <!-- openai-like page -->
  <div v-else embedded style="flex: 1; overflow: auto;">
    <OpenaiLikePage />
  </div>
</template>

<script setup>
import { computed, watch, } from "vue";
import AIIcon from '@/components/icons/AI.vue';
import useLLMTabStore from "stores/llmtab.js";
import OllamaPage from '@/components/content/OllamaPage.vue';
import OpenaiLikePage from '@/components/content/OpenaiLikePage.vue';
const llmtabStore = useLLMTabStore()
const configContent = computed(() => llmtabStore.configContent)
const blankLLM = computed(() => !configContent.value?.name && !configContent.value.isNew)
const isOllama = computed(() => configContent.value.name === 'ollama')
const isNew = computed(() => !!configContent.value.isNew)

</script>

<style scoped>
.empty-card {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
}
</style>