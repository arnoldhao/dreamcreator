<template>
  <div class="card bg-base-100 shadow-xl flex-1 overflow-auto">
    <div class="card-body">
      <h2 class="card-title text-2xl italic">{{ $t('ai.llm') }}</h2>
      <p class="text-sm italic text-base-content/60 mb-6">{{ $t('ai.llm_description') }}</p>

      <form @submit.prevent="save">
        <div class="grid grid-cols-1 gap-4 mb-6">
          <div class="form-control">
            <label class="label" for="llm-name">
              <span class="label-text">{{ $t('ai.llm_name') }}</span>
            </label>
            <input id="llm-name" v-model="llmForm.name" type="text" class="input input-bordered" />
          </div>

          <div class="form-control">
            <label class="label" for="llm-base-url">
              <span class="label-text">{{ $t('ai.llm_base_url') }}</span>
            </label>
            <input id="llm-base-url" v-model="llmForm.baseURL" type="text" class="input input-bordered" />
          </div>

          <div class="form-control">
            <label class="label" for="llm-api-key">
              <span class="label-text">{{ $t('ai.llm_api_key') }}</span>
            </label>
            <input id="llm-api-key" v-model="llmForm.APIKey" type="password" class="input input-bordered" />
          </div>
        </div>

        <div class="flex flex-wrap gap-4 mb-6">
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ $t('ai.llm_region') }}</span>
            </label>
            <div class="btn-group">
              <input v-for="option in REGION_OPTIONS" :key="option.value" v-model="llmForm.region" 
                     type="radio" :value="option.value" :aria-label="option.label" 
                     class="btn btn-sm" />
            </div>
          </div>

          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ $t('ai.llm_icon') }}</span>
            </label>
            <div class="btn-group h-8">
              <label v-for="option in ICON_OPTIONS" :key="option.value" 
                     class="icon-btn">
                <input v-model="llmForm.icon" 
                       type="radio" :value="option.value" 
                       :aria-label="option.label" 
                       class="hidden" />
                <component :is="option.icon" class="w-4 h-4" />
              </label>
            </div>
          </div>
        </div>

        <div class="divider"></div>

        <h3 class="text-xl italic font-bold mb-2">{{ $t('ai.llm_models') }}</h3>
        <p class="text-sm italic text-base-content/60 mb-4">{{ $t('ai.model_name_description') }}</p>

        <div class="bg-base-200 rounded-box p-4 mb-4">
          <ul v-if="llmForm.models?.length > 0" class="space-y-2">
            <li v-for="(model, index) in llmForm.models" :key="index" 
                class="flex justify-between items-center bg-base-100 p-2 rounded-lg">
              <span class="text-base-content">{{ model.name }}</span>
              <button @click="removeModel(index)" 
                      class="btn btn-circle btn-xs btn-error" 
                      :disabled="llmForm.models.length <= 1">
                <DeleteIcon class="h-4 w-4" />
              </button>
            </li>
          </ul>

          <div v-if="showAddModelInput" class="mt-4">
            <div class="join w-full">
              <input v-model="newModelName" 
                     type="text" 
                     :placeholder="$t('ai.model_name_placeholder')" 
                     class="input input-bordered join-item flex-grow" />
              <button @click="confirmAddModel" 
                      class="btn btn-primary join-item" 
                      :disabled="!newModelName.trim()">
                {{ $t('ai.add_model') }}
              </button>
            </div>
          </div>
          <button v-else @click="showAddModelInput = true" 
                  type="button" 
                  class="btn btn-primary btn-block mt-4">
            {{ $t('ai.add_model') }}
          </button>
        </div>

        <div class="card-actions justify-between mt-6">
          <button @click="resetForm" type="button" class="btn">
            {{ $t('common.reset') }}
          </button>
          <div>
            <button @click="cancel" type="button" class="btn mr-2">{{ $t('common.cancel') }}</button>
            <button type="submit" class="btn btn-primary">{{ $t('common.save') }}</button>
          </div>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch } from "vue";
import { useI18n } from 'vue-i18n'
import DeleteIcon from '@/components/icons/Delete.vue';
import useLLMTabStore from "stores/llmtab.js";
import { LLM_CONTENT_RULES, REGION_OPTIONS, ICON_OPTIONS } from "@/consts/llmContent.js";

const llmtabStore = useLLMTabStore()
const configContent = computed(() => llmtabStore.configContent)
const isNew = computed(() => !!configContent.value.isNew)
const defaultForm = {
  name: '',
  region: '',
  baseURL: '',
  APIKey: '',
  icon: '',
  show: true,
  models: [{ name: '', available: true, description: '' }]
}

const llmForm = ref(isNew.value ? { ...configContent.value } : { ...defaultForm })

watch([isNew, configContent], ([newIsNew, newConfigContent]) => {
  if (!newIsNew) {
    Object.assign(llmForm.value, newConfigContent)
  } else {
    Object.assign(llmForm.value, defaultForm)
  }
}, { immediate: true })

const resetForm = () => {
  Object.assign(llmForm.value, isNew.value ? configContent.value : defaultForm)
}

const cancel = () => {
  llmtabStore.cancelLLM()
}

const showAddModelInput = ref(false)
const newModelName = ref('')

const confirmAddModel = () => {
  if (newModelName.value.trim()) {
    llmForm.value.models.push({ name: newModelName.value.trim(), available: true, description: '' })
    newModelName.value = ''
    showAddModelInput.value = false
  }
}

const removeModel = (index) => {
  if (llmForm.value.models.length > 1) {
    llmForm.value.models.splice(index, 1)
  }
}

const formRef = ref(null)

function save() {
  llmtabStore.saveLLM(isNew.value, llmForm.value)
}
</script>

<style lang="postcss">
.btn-group .btn-sm {
  height: 2rem;
  min-height: 2rem;
  font-size: 0.875rem;
}

.icon-btn {
  @apply btn btn-sm hover:bg-base-200;
}

.icon-btn:has(input:checked) {
  @apply bg-primary text-primary-content hover:bg-primary;
}
</style>