import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { subtitleService } from '@/services/subtitleService.js'

// Default target language list (kept in sync with backend/core/subtitles/service.go)
const DEFAULT_TARGET_LANGUAGES = [
  { code: 'en', name: 'English' },
  { code: 'zh-CN', name: '简体中文' },
  { code: 'zh-TW', name: '繁體中文' },
  { code: 'ja', name: '日本語' },
  { code: 'ko', name: '한국어' },
  { code: 'fr', name: 'Français' },
  { code: 'de', name: 'Deutsch' },
  { code: 'es', name: 'Español' },
  { code: 'ru', name: 'Русский' },
  { code: 'vi', name: 'Tiếng Việt' },
  { code: 'pt-BR', name: 'Português (Brasil)' },
  { code: 'pt-PT', name: 'Português (Portugal)' },
  { code: 'id', name: 'Bahasa Indonesia' },
  { code: 'hi', name: 'हिन्दी' },
  { code: 'ar', name: 'العربية' },
  { code: 'it', name: 'Italiano' },
  { code: 'tr', name: 'Türkçe' },
  { code: 'th', name: 'ไทย' },
  { code: 'nl', name: 'Nederlands' },
  { code: 'pl', name: 'Polski' },
]

export const useTargetLanguagesStore = defineStore('targetLanguages', () => {
  const list = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  const error = ref(null)

  const nameMap = computed(() => {
    const map = {}
    for (const l of list.value || []) {
      if (!l || !l.code) continue
      map[l.code] = l.name || l.code
    }
    return map
  })

  const codes = computed(() => (list.value || []).map(l => l && l.code).filter(Boolean))

  async function load(force = false) {
    if (loading.value) return list.value
    if (loaded.value && !force) return list.value
    loading.value = true
    error.value = null
    try {
      const res = await subtitleService.listTargetLanguages()
      const arr = Array.isArray(res) ? res : []
      list.value = arr
      subtitleService.targetLanguages = arr
      loaded.value = true
    } catch (e) {
      console.error('Load target languages failed:', e)
      error.value = e
      // Fallback to a sensible default set so UI still works when backend is unavailable
      const fallback = DEFAULT_TARGET_LANGUAGES.map(it => ({ ...it }))
      list.value = fallback
      subtitleService.targetLanguages = fallback
      loaded.value = true
    } finally {
      loading.value = false
    }
    return list.value
  }

  async function ensureLoaded() {
    return load(false)
  }

  async function upsert(lang) {
    const payload = { ...(lang || {}) }
    payload.code = (payload.code || '').trim()
    payload.name = (payload.name || '').trim()
    if (!payload.code) throw new Error('Language code is required')
    const saved = await subtitleService.upsertTargetLanguage(payload)
    if (!saved || !saved.code) return saved
    const idx = (list.value || []).findIndex(x => x && x.code === saved.code)
    if (idx >= 0) list.value[idx] = saved
    else list.value.push(saved)
    return saved
  }

  async function remove(code) {
    const c = (code || '').trim()
    if (!c) return
    await subtitleService.deleteTargetLanguage(c)
    list.value = (list.value || []).filter(x => !x || x.code !== c)
  }

  async function resetToDefaults() {
    try {
      await subtitleService.resetTargetLanguages()
    } catch (e) {
      console.error('Reset target languages to default failed:', e)
      error.value = e
    }
    loaded.value = false
    return load(true)
  }

  function getName(code) {
    const c = code || ''
    return nameMap.value[c] || c
  }

  return {
    list,
    loading,
    loaded,
    error,
    nameMap,
    codes,
    load,
    ensureLoaded,
    upsert,
    remove,
    resetToDefaults,
    getName,
  }
})
