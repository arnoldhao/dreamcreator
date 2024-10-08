<template>
  <div class="container mx-auto p-6 relative">
    <div class="flex justify-center items-center mb-8 max-w-2xl mx-auto mt-8">
      <!-- search input -->
      <div class="form-control w-full">
        <div class="input-group">
          <input type="text" v-model="searchQuery" :placeholder="$t('ollama.search_model')" class="input input-bordered w-full"
            @input="handleSearch" />
        </div>
      </div>
    </div>

    <!-- data table -->
    <div class="overflow-x-auto h-[300px] md:h-[400px] lg:h-[500px] overflow-y-auto">
      <table class="table w-full">
        <!-- table header -->
        <thead>
          <tr>
            <th class="text-center">{{ $t('ollama.model') }}</th>
            <th class="text-center">{{ $t('ollama.size') }}</th>
            <th class="text-center">{{ $t('ollama.parameter_size') }}</th>
            <th class="text-center">{{ $t('ollama.status') }}</th>
            <th class="text-center">{{ $t('ollama.action') }}</th>
          </tr>
        </thead>
        <tbody>
          <!-- table content -->
          <tr v-for="model in filteredModels" :key="model.name">
            <td class="text-center"><strong>{{ model.name }}</strong></td>
            <td class="text-center">{{ formatSize(model.size) }}</td>
            <td class="text-center">
              <span class="badge badge-info">{{ model.details.parameter_size || 'Unknown' }}</span>
            </td>
            <td class="text-center">
              <!-- status column -->
              <progress v-if="isDownloading(model)" class="progress progress-primary w-56"
                :value="model.downloadProgress" max="100"></progress>
              <span v-else-if="isModelRunning(model)" class="badge badge-success">Running</span>
              <span v-else class="badge badge-ghost">Not Running</span>
            </td>
            <td class="text-center">
              <!-- action dropdown -->
              <div class="dropdown">
                <label tabindex="0" class="btn btn-sm m-1">{{ $t('ollama.action') }}</label>
                <ul tabindex="0"
                  class="dropdown-content menu p-2 shadow bg-base-100 rounded-box z-50">
                  <!-- // add chat button -->
                  <!-- <li><a @click="handleRun(model)">{{ $t('ollama.run') }}</a></li> -->
                  <li><a @click="handleDelete(model)" class="whitespace-nowrap">{{ $t('ollama.delete') }}</a></li>
                  <li><a @click="showDetail(model)" class="whitespace-nowrap">{{ $t('ollama.detail') }}</a></li>
                </ul>
              </div>
            </td>
          </tr>
        </tbody>
      </table>

      <!-- use wider card component -->
      <div v-if="searchQuery && !filteredModels.some(model => model.name === searchQuery)" class="mt-4 px-4">
        <div class="card bg-base-200 shadow-xl w-full">
          <div class="card-body items-center text-center py-6">
            <h2 class="card-title text-xl mb-2">{{ $t('ollama.model_not_found') }}</h2>
            <p class="text-base mb-4">{{ $t('ollama.download_model') }} {{ searchQuery }} {{ $t('ollama.from_ollama') }}</p>
            <div class="card-actions">
              <button class="btn btn-primary" @click="downloadModel">{{ $t('ollama.download_model') }}</button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- modal -->
    <div class="modal" :class="{ 'modal-open': showDetailModal }">
      <div class="modal-box">
        <h3 class="font-bold text-lg">{{ $t('ollama.model_details') }}</h3>
        <div class="py-4 space-y-4">
          <!-- model details content -->
          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.model_name') }}</span>
            <span class="badge badge-primary">{{ selectedModelDetails.name }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.size') }}</span>
            <span>{{ formatSize(selectedModelDetails.size) }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.parameter_size') }}</span>
            <span class="badge badge-info">{{ selectedModelDetails.parameter_size || 'Unknown' }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.quantization_level') }}</span>
            <span>{{ selectedModelDetails.quantization_level || 'Unknown' }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.model_family') }}</span>
            <span>{{ selectedModelDetails.family || 'Unknown' }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.modified_time') }}</span>
            <span>{{ formatDate(selectedModelDetails.modified_at) }}</span>
          </div>

          <div class="flex items-center">
            <span class="w-1/3 font-semibold">{{ $t('ollama.model_status') }}</span>
            <span>{{ isModelRunning(selectedModelDetails) ? 'Running' : 'Not Running' }}</span>
          </div>
        </div>
        <div class="modal-action">
          <button class="btn" @click="showDetailModal = false">{{ $t('ollama.close') }}</button>
        </div>
      </div>
    </div>

    <!-- delete confirm modal -->
    <div class="modal" :class="{ 'modal-open': showDeleteConfirm }">
      <div class="modal-box">
        <h3 class="font-bold text-lg">{{ $t('ollama.confirm_delete') }}</h3>
        <p class="py-4">{{ $t('ollama.confirm_delete_model', { name: modelToDelete.name }) }}</p>
        <div class="modal-action">
          <button class="btn" @click="cancelDelete">{{ $t('ollama.cancel') }}</button>
          <button class="btn btn-error" @click="confirmDelete">{{ $t('ollama.delete') }}</button>
        </div>
      </div>
    </div>
    <!-- config button and connection status -->
    <div class="fixed bottom-4 right-4 flex items-center space-x-2">
      <span class="text-sm font-medium" :class="ollamaConnected ? 'text-success' : 'text-error'">
        {{ ollamaConnected ? 'Connected' : 'Disconnected' }}
      </span>
      <button class="btn btn-circle btn-ghost" @click="refreshOllamaStatus">
        <RefreshIcon class="w-6 h-6" />
      </button>
      <button class="btn btn-circle btn-ghost" @click="showConfigModal = true">
        <ConfigIcon class="w-6 h-6" />
      </button>
    </div>

    <!-- config modal -->
    <div class="modal" :class="{ 'modal-open': showConfigModal }">
      <div class="modal-box">
        <h3 class="font-bold text-lg">{{ $t('ollama.ollama_config') }}</h3>
        <div class="py-4">
          <div class="form-control">
            <label class="label">
              <span class="label-text">{{ $t('ollama.ollama_server_address') }}</span>
            </label>
            <input type="text" v-model="ollamaConfig" :placeholder=ollamaConfig class="input input-bordered w-full" />
          </div>
        </div>
        <div class="modal-action">
          <button class="btn btn-primary" @click="saveConfig">{{ $t('ollama.save') }}</button>
          <button class="btn" @click="showConfigModal = false">{{ $t('ollama.close') }}</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick, watch } from 'vue';
import ConfigIcon from '@/components/icons/Config.vue'
import RefreshIcon from '@/components/icons/Refresh.vue'
import { List, ListRunning, Delete, Heartbeat, GetHost, SetHost } from 'wailsjs/go/ollama/Service'
import emitter from '@/utils/eventBus'
import { EMITTER_EVENTS } from '@/consts/emitter'
import { useOllamaStore } from '@/stores/ollama'
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const ollamaStore = useOllamaStore()
const { downloads } = storeToRefs(ollamaStore)

function handleDownloadStatus(downloadStatuses) {
  // update download status to ollamaStore
  if (downloadStatuses.length > 0) {
    for (const status of downloadStatuses) {
      updateModel(status)
    }
  }
}

const models = ref([])
async function listModels() {
  try {
    const { data, success, msg } = await List()
    if (success) {
      models.value = JSON.parse(data)
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.info(t('ollama.error_fetching_models', error.message))
  }
}

const runningModels = ref([])
async function listRunningModels() {
  try {
    const { data, success, msg } = await ListRunning()
    if (success) {
      runningModels.value = JSON.parse(data)
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.info(t('ollama.error_fetching_running_models', error.message))
  }
}

const downloadCompleted = ref([])
async function updateModel(data) {
  const modelName = data.id.split('_').at(-1);
  
  // if model completed, return directly
  if (downloadCompleted.value.includes(modelName)) {
    return
  }

  const existingModelIndex = models.value.findIndex(model => model.name === modelName);
  if (existingModelIndex === -1) {
    // if model not exsited, add to model.value
    const newModel = {
      name: modelName,
      size: parseInt(data.total) || 0,
      details: {
        parameter_size: 'Unknown',
        quantization_level: 'Unknown',
        family: 'Unknown'
      },
      modified_at: new Date().toISOString(),
      downloadStatus: data.status,
      downloadProgress: data.progress
    };
    models.value.push(newModel);
  } else {
    // if model exsited, add new data to model.value
    const model = models.value[existingModelIndex];
    model.size = parseInt(data.total) || model.size;
    model.downloadStatus = data.status;
    model.downloadProgress = data.progress;
    model.modified_at = new Date().toISOString();
  }

  // if model download complated, refresh models list
  if (data.status === 'success') {
    await listModels()
      // if list models contains current model, it means model is downloaded,no need add new data to model.value
      nextTick(() => {
        const existingModelIndex = models.value.findIndex(model => model.name === modelName);
        if (existingModelIndex !== -1) {
          downloadCompleted.value.push(modelName)
        }
      })
  }
}

watch(downloads, (newVal) => {
  handleDownloadStatus(newVal)
}, { deep: true })

// format date
function formatDate(dateString) {
  const date = new Date(dateString);
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  });
}

// format file size
function formatSize(bytes) {
  if (bytes === 0) return '0 Bytes';
  const k = 1024;
  const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
}

const searchQuery = ref('')
const trimedSearchQuery = computed(() => searchQuery.value.trim())
const filteredModels = computed(() => {
  if (!trimedSearchQuery.value) return models.value;
  return models.value.filter(model =>
    model.name.toLowerCase().includes(trimedSearchQuery.value.toLowerCase())
  );
});

function handleSearch() {
  // search logic already implemented by computed property filteredModels
}

function downloadModel() {
  if (trimedSearchQuery.value) {
    const id = EMITTER_EVENTS.OLLAMA_PULL_MODEL + '_' + trimedSearchQuery.value
    emitter.emit(EMITTER_EVENTS.OLLAMA_PULL_MODEL, {
      id: id,
      model: trimedSearchQuery.value,
    })
    $message.success(t('ollama.start_download_model', trimedSearchQuery.value))
  }
}

// check if model is running
const isModelRunning = (model) => {
  return runningModels.value.some(rm => rm.name === model.name)
}

// check if model is downloading
const isDownloading = (model) => {
  return model.downloadProgress !== '' && model.downloadProgress !== undefined
}


const showDetailModal = ref(false)
const selectedModelDetails = ref({})
const showDeleteConfirm = ref(false)
const modelToDelete = ref('')

const handleRun = (model) => {
  console.log('run model:', model)
}

// delete model
const handleDelete = (model) => {
  modelToDelete.value = model;
  showDeleteConfirm.value = true;
}

const showDetail = (model) => {
  selectedModelDetails.value = {
    ...model,
    ...model.details,
    parameter_size: model.details.parameter_size,
    quantization_level: model.details.quantization_level,
    family: model.details.family
  }
  showDetailModal.value = true
}

const cancelDelete = () => {
  showDeleteConfirm.value = false;
  modelToDelete.value = '';
}

const confirmDelete = async () => {
  try {
    const { success, msg } = await Delete(modelToDelete.value.name);
    if (success) {
      $message.success(t('ollama.delete_success', modelToDelete.value.name));
    } else {
      throw new Error(msg)
    }
    // refresh model list
    await listModels();
  } catch (error) {
    $message.error(t('ollama.delete_failed', error.message));
  } finally {
    showDeleteConfirm.value = false;
    modelToDelete.value = '';
  }
}


// ollama config
const showConfigModal = ref(false);
const ollamaConfig = ref();
const ollamaConnected = ref(false);

async function handleOllamaStatus() {
  const { success } = await Heartbeat();
  if (success) {
    ollamaConnected.value = true;
  } else {
    ollamaConnected.value = false;
  }
}

async function refreshOllamaStatus() {
  await handleOllamaStatus();
  await listModels();
  await listRunningModels();
}

async function getOllamaHost() {
  try {
    const { data, success, msg } = await GetHost();
    if (success) {
      ollamaConfig.value = data;
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('ollama.get_ollama_host_failed', error))
  }
}

const saveConfig = async () => {
  try {
    const { data, success, msg } = await SetHost(ollamaConfig.value);
    if (success) {
      $message.success(t('ollama.save_config_success'));
      ollamaConfig.value = data;
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('ollama.save_config_failed', error.message));
  }
  showConfigModal.value = false;
};

onMounted(async () => {
  await listModels()
  await listRunningModels()
  await handleOllamaStatus();
  await getOllamaHost();
})
</script>
