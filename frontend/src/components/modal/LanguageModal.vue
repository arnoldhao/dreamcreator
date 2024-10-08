<template>
  <input type="checkbox" id="language-modal" class="modal-toggle" v-model="showModal" />
  <div class="modal">
    <div class="modal-box w-11/12 max-w-5xl">
      <h3 class="font-bold text-xl mb-6">{{ $t('language.language_settings') }}</h3>
      <div class="space-y-8">
        <!-- common languages -->
        <div>
          <h4 class="font-semibold text-base mb-4">{{ $t('language.common') }}</h4>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div v-for="lang in commonLanguages" :key="lang.value" class="bg-base-200 rounded-lg p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between">
              <span class="text-base mb-2 sm:mb-0 break-words">{{ lang.label }}</span>
              <button @click="toggleLanguage(lang.value)" class="btn btn-sm btn-ghost text-error" :title="$t('language.remove_from_common')">
                <DownIcon class="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>

        <!-- extra languages -->
        <div>
          <h4 class="font-semibold text-base mb-4">{{ $t('language.extra') }}</h4>
          <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            <div v-for="lang in nonCommonLanguages" :key="lang.value" class="bg-base-100 border border-base-300 rounded-lg p-4 flex flex-col sm:flex-row items-start sm:items-center justify-between">
              <span class="text-base mb-2 sm:mb-0 break-words">{{ lang.label }}</span>
              <div class="flex flex-col sm:flex-row space-y-2 sm:space-y-0 sm:space-x-2">
                <button @click="toggleLanguage(lang.value)" class="btn btn-sm btn-primary" :title="$t('language.set_as_common')">
                  <UpIcon class="w-5 h-5" />
                </button>
                <button @click="deleteLanguage(lang.value)" class="btn btn-sm btn-error" :title="$t('common.delete')">
                  <DeleteIcon class="w-5 h-5" />
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="modal-action mt-8">
        <label for="language-modal" class="btn btn-ghost">{{ $t('common.close') }}</label>
        <label for="add-language-modal" class="btn btn-primary">
          <AddIcon class="w-5 h-5 mr-2" />
          {{ $t('language.add_language') }}
        </label>
      </div>
    </div>
  </div>

  <!-- add language modal -->
  <input type="checkbox" id="add-language-modal" class="modal-toggle" v-model="showAddLangModal" />
  <div class="modal">
    <div class="modal-box">
      <h3 class="font-bold text-lg">{{ $t('language.add_language') }}</h3>
      <form @submit.prevent="confirmAddLang" class="py-4">
        <div class="form-control">
          <label class="label">
            <span class="label-text">{{ $t('language.language_name') }}</span>
          </label>
          <input 
            v-model="newLang.label" 
            type="text" 
            :placeholder="t('language.please_enter_the_language_name')" 
            class="input input-bordered w-full" 
            required 
          />
        </div>
        <div class="form-control mt-4">
          <label class="label cursor-pointer">
            <span class="label-text">{{ $t('language.is_it_a_common_language') }}</span>
            <input v-model="newLang.isCommon" type="checkbox" class="toggle toggle-primary" />
          </label>
        </div>
        <div class="modal-action">
          <label for="add-language-modal" class="btn">{{ $t('common.cancel') }}</label>
          <button type="submit" class="btn btn-primary">{{ $t('language.add_language_confirm') }}</button>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch, toRef } from 'vue'
import { UpdateLanguage, AddLanguage, DeleteLanguage } from 'wailsjs/go/languages/Service'
import { useI18n } from 'vue-i18n'
import DeleteIcon from '@/components/icons/Delete.vue'
import UpIcon from '@/components/icons/Up.vue'
import DownIcon from '@/components/icons/Down.vue'
import AddIcon from '@/components/icons/Add.vue'

const { t } = useI18n()
const emit = defineEmits(['update:show', 'update:langs'])
const props = defineProps({
  show: {
    type: Boolean,
    required: true,
  },
  langs: {
    type: Array,
    required: true,
    default: () => [],
  },
  commonLangs: {
    type: Array,
    required: true,
    default: () => [],
  },
})

const commonLangsRef = toRef(props, 'commonLangs')
const showModal = computed({
  get: () => props.show,
  set: (value) => emit('update:show', value)
})

const selectedLangs = ref([])
const showAddLangModal = ref(false)
const newLang = ref({ label: '', isCommon: false })

onMounted(() => {
  selectedLangs.value = [...props.commonLangs]
})

watch(commonLangsRef, (newVal) => {
  selectedLangs.value = newVal
})

const commonLanguages = computed(() => {
  return props.langs.filter(lang => selectedLangs.value.includes(lang.value))
})

const nonCommonLanguages = computed(() => {
  return props.langs.filter(lang => !selectedLangs.value.includes(lang.value))
})

function isCommonLanguage(langValue) {
  return selectedLangs.value.includes(langValue)
}

function toggleLanguage(langValue) {
  if (isCommonLanguage(langValue)) {
    selectedLangs.value = selectedLangs.value.filter(l => l !== langValue)
  } else {
    selectedLangs.value = [...selectedLangs.value, langValue]
  }
  handleUpdateLanguage(selectedLangs.value)
}

async function handleUpdateLanguage(newSelectedLangs) {
  const oldSelectedSet = new Set(props.commonLangs)
  const newSelectedSet = new Set(newSelectedLangs)
  const addToCommon = newSelectedLangs
    .filter(lang => !oldSelectedSet.has(lang))
    .map(lang => updateLanguage('common', lang))

  const removeFromCommon = props.commonLangs
    .filter(lang => !newSelectedSet.has(lang))
    .map(lang => updateLanguage('extra', lang))

  const updatePromises = [...addToCommon, ...removeFromCommon]

  try {
    await Promise.all(updatePromises)
    emit('update:langs')
  } catch (error) {
    $message.error(t('language.update_error'), error)
  }
}

async function updateLanguage(group, lang) {
  try {
    const { success, msg } = await UpdateLanguage(group, lang)
    if (success) {
      $message.info(t('language.update_success') + ': ' + lang + ' -> ' + group)
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('language.update_error') + ': ' + error.message)
  }
}

async function confirmAddLang() {
  if (!newLang.value.label) {
    $message.error(t('language.please_fill_in_the_complete_language_information'))
    return
  }

  const group = newLang.value.isCommon ? 'common' : 'extra'
  const lang = newLang.value.label.charAt(0).toUpperCase() + newLang.value.label.slice(1)

  try {
    const { success, msg } = await AddLanguage(group, lang)
    if (success) {
      $message.success(t('language.add_new_language_success') + ': ' + lang + ' -> ' + group)
      emit('update:langs')
      showAddLangModal.value = false
      newLang.value = { label: '', isCommon: false }
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('language.add_new_language_error') + ': ' + error.message)
  }
}

async function deleteLanguage(lang) {
  try {
    // call backend API to delete language
    const { success, msg } = await DeleteLanguage('extra', lang)
    if (success) {
      $message.success(t('common.delete_success') + ': ' + lang)
      emit('update:langs')
    } else {
      throw new Error(msg)
    }
  } catch (error) {
    $message.error(t('common.delete_failed') + ': ' + error.message)
  }
}
</script>

