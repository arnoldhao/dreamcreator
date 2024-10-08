<template>
  <dialog :open="showModal" class="modal">
    <div class="modal-box w-11/12 max-w-md">
      <h3 class="font-bold text-lg">{{ $t('ai.change_model') }}</h3>
      <div class="py-4">
        <form class="flex flex-col gap-4">
          <div class="form-control w-full">
            <label class="label">
              <span class="label-text">{{ $t('ai.llm_name') }}</span>
            </label>
            <select v-model="newModel.llmName" class="select select-bordered" @change="updateModelOptions">
              <option disabled selected>{{ $t('ai.llm_placeholder') }}</option>
              <option v-for="option in llmOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
          <div class="form-control w-full">
            <label class="label">
              <span class="label-text">{{ $t('ai.llm_model') }}</span>
            </label>
            <select v-model="newModel.modelName" class="select select-bordered">
              <option disabled selected>{{ $t('ai.model_placeholder') }}</option>
              <option v-for="option in modelOptions" :key="option.value" :value="option.value">
                {{ option.label }}
              </option>
            </select>
          </div>
        </form>
      </div>
      <div class="modal-action">
        <button class="btn" @click="closeModal">{{ $t('common.cancel') }}</button>
        <button class="btn btn-primary" @click="confirmSelection">{{ $t('common.confirm') }}</button>
      </div>
    </div>
  </dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import { storeToRefs } from 'pinia'
import useLLMTabStore from "stores/llmtab.js";
import { useI18n } from 'vue-i18n'
import { List } from 'wailsjs/go/ollama/Service'

const { t } = useI18n()
const props = defineProps({
  show: Boolean,
})

const emit = defineEmits(['update:show'])

const llmtabStore = useLLMTabStore()
const { currentModel } = storeToRefs(llmtabStore)
const llms = computed(() => llmtabStore.llms)

const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const llmOptions = computed(() => {
  return llms.value.map(llm => ({
    label: llm.name,
    value: llm.name
  }))
})

const newModel = ref({})
const modelOptions = ref([]);
const updateModelOptions = async (event) => {
  const selectedLLMName = event.target.value || newModel.value.llmName
  const selectedLLM = llms.value.find(llm => llm.name === selectedLLMName);
  if (selectedLLM) {
    if (selectedLLM.name === 'ollama') {
      const { data, success } = await List()
      if (success) {
        let parsedData;
        try {
          parsedData = typeof data === 'string' ? JSON.parse(data) : data;
        } catch (error) {
          parsedData = [];
        }
        
        if (Array.isArray(parsedData)) {
          modelOptions.value = parsedData.map(model => ({
            label: model.name,
            value: model.model || model.name
          }))
        } else {
          modelOptions.value = []
        }
      } else {
        modelOptions.value = [];
      }
    } else {
      modelOptions.value = selectedLLM.models.map(model => ({
        label: model.name,
      value: model.name
      }));
    }
    // refresh currentModel
    newModel.value.llmName = selectedLLMName
    newModel.value.modelName = modelOptions.value[0].value
  } else {
    modelOptions.value = [];
  }
};

watch(() => props.show, (newValue) => {
  showModal.value = newValue
});

watch(currentModel, (newValue) => {
  newModel.value.llmName = newValue.llmName
  newModel.value.modelName = newValue.modelName
});

const closeModal = () => {
  showModal.value = false
};

watch(() => showModal.value, (newValue) => {
  if (newValue) {
    // when modal is open, update newModel with currentModel
    newModel.value = {
      llmName: currentModel.value.llmName,
      modelName: currentModel.value.modelName
    }
    // update model options
    updateModelOptions({ target: { value: currentModel.value.llmName } })
  }
});

const confirmSelection = async () => {
  if (newModel.value.llmName && newModel.value.modelName) {
    await llmtabStore.updateCurrentModel(newModel.value.llmName, newModel.value.modelName)
    closeModal()
  } else {
    $message.error(t('ai.please_select_model'))
  }
};
</script>
