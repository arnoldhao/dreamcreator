<template>
  <n-modal v-model:show="showLocal" preset="dialog" title="编辑标题">
    <template #default>
      <p>{{ $t('dialogue.edit_title.please_input_new_title') }}</p>
      <n-input v-model:value="titleLocal" />
    </template>
    <template #action>
      <n-button @click="handleCancel">{{ $t('common.cancel') }}</n-button>
      <n-button type="primary" @click="handleConfirm">{{ $t('common.confirm') }}</n-button>
    </template>
  </n-modal>
</template>

<script setup>
import { ref, watch } from 'vue'
import { NModal, NInput, NButton } from 'naive-ui'
import { UpdateTitle } from 'wailsjs/go/subtitles/Service'
import useSuperTabStore from "stores/supertab.js";
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const tabStore = useSuperTabStore()

const props = defineProps({
  show: Boolean,
  id: [String, Number],
  title: String
})

const emit = defineEmits(['update:show'])

const showLocal = ref(props.show)
const titleLocal = ref(props.title)

watch(() => props.show, (newVal) => {
  showLocal.value = newVal
  if (newVal) {
    titleLocal.value = props.title
  }
})

watch(showLocal, (newVal) => {
  emit('update:show', newVal)
})

async function handleConfirm() {
  if (titleLocal.value && titleLocal.value !== props.title) {
    const { success, msg } = await UpdateTitle(props.id, titleLocal.value)
    if (success) {
      $message.success(t('task.title_updated'))
      tabStore.updateTitle(props.id, titleLocal.value)
      showLocal.value = false
    } else {
      $message.error(t('task.title_update_failed') + msg)
    }
  } else {
    $message.info(t('task.title_not_changed'))
  }
}

function handleCancel() {
  $message.info(t('task.operation_cancelled'))
  showLocal.value = false
}
</script>