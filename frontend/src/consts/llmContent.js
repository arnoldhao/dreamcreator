import { i18nGlobal } from "@/utils/i18n.js";
import OpenAIcon from '@/components/icons/OpenAI.vue';
import OllamaIcon from '@/components/icons/Ollama.vue';

export const LLM_CONTENT_RULES = {
  isNew: { required: true, message: i18nGlobal.t('ai.is_new_required'), trigger: 'change' },
  name: { required: true, message: i18nGlobal.t('ai.name_required'), trigger: 'blur' },
  region: { required: true, message: i18nGlobal.t('ai.region_required'), trigger: 'change' },
  baseURL: { required: true, message: i18nGlobal.t('ai.base_url_required'), trigger: 'blur' },
  APIKey: { required: true, message: i18nGlobal.t('ai.api_key_required'), trigger: 'blur' },
  icon: { required: true, message: i18nGlobal.t('ai.icon_required'), trigger: 'change' },
};

export const REGION_OPTIONS = [
  { label: 'Local', value: 'Local' },
  { label: 'China', value: 'China' },
  { label: 'ExceptChina', value: 'ExceptChina' }
];

export const ICON_OPTIONS = [
  { label: 'OpenAI-like', value: 'open-like', icon: OpenAIcon },
  { label: 'Ollama', value: 'ollama', icon: OllamaIcon },
];