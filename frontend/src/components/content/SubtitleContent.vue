<template>
  <div v-if="isBlank" class="blank-page">
    <n-result status="404" title="Welcome" description="Can Me? I Can!">
      <template #default>
        <file-open-input v-model="filePath" placeholder="Please select a file" />
      </template>
      <template #footer>
        <n-flex justify="center">
          <n-button @click="handleConvert()">{{ $t('subtitle.convert') }}</n-button>
        </n-flex>
      </template>
    </n-result>
  </div>
  <div v-else style="position: relative">
    <content-editor 
    :tabId="currentTab.id"
    :content="currentTabCaption" 
    :readonly="String(!allowAction)" 
    :border="true"
    class="editor-border" 
    style="height: 100%" 
    @input="onInput" 
    @reset="onReset" 
    @save="onSave" />
  </div>
</template>

<script setup>
import { computed, ref } from 'vue';
import { storeToRefs } from 'pinia';
import useSuperTabStore from 'stores/supertab.js';
import ContentEditor from '@/components/content_value/ContentEditor.vue'

const filePath = ref('');
async function handleConvert() {
  tabStore.convertSubtitle(filePath.value);
}

const tabStore = useSuperTabStore()
const { currentTab } = storeToRefs(tabStore)

const currentTabCaption = computed(() => {
  return currentTab.value?.captions
})
const isBlank = computed(() => currentTab.value?.blank ?? false)
const isStream = computed(() => currentTab.value?.stream || false)
const translationState = computed(() => currentTab.value?.getTranslationState() || {})
const isCompleted = computed(() => translationState.value.isCompleted)
const allowAction = computed(() => !isStream.value || isCompleted.value)

function onInput(value) {
}

function onSave(value) {
}

function onReset(value) {
}

</script>

<style lang="scss" scoped>
.editor-inst {
  position: absolute;
  top: 2px;
  bottom: 2px;
  left: 2px;
  right: 2px;
}

:deep(.line-numbers) {
  white-space: nowrap;
}

.blank-page {
  display: flex;
  flex-direction: column;
  justify-content: center;
  align-items: center;
  height: 100%;
}
</style>
