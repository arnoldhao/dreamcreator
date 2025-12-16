import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { 
  listProviders, createProvider, updateProvider, deleteProvider, testProvider, refreshModels
} from '@/services/llmProviderService.js'
import { useI18n } from 'vue-i18n'

export default function useLLMStore() {
  const { t } = useI18n()
  const providers = ref([])
  const profiles = ref([]) // legacy state removed; keep empty array to avoid UI break
  const loading = ref(false)

  const providerTypes = [
    { label: 'openai_compat', value: 'openai_compat' },
    { label: 'anthropic_compat', value: 'anthropic_compat' },
    { label: 'remote', value: 'remote' },
  ]

  async function fetchProviders() {
    loading.value = true
    try {
      providers.value = await listProviders()
    } catch (e) {
      window.$message?.error?.(t('common.refresh_failed') + (e?.message ? `: ${e.message}` : ''))
    } finally {
      loading.value = false
    }
  }

  async function addProvider(p) {
    try {
      const res = await createProvider(p)
      window.$message?.success?.(t('common.saved'))
      await fetchProviders()
      return res
    } catch (e) {
      window.$message?.error?.(t('common.save_failed'))
      throw e
    }
  }

  async function saveProvider(id, p) {
    try {
      await updateProvider(id, p)
      window.$message?.success?.(t('common.saved'))
      await fetchProviders()
    } catch (e) {
      window.$message?.error?.(t('common.save_failed'))
      throw e
    }
  }

  async function removeProvider(id) {
    try {
      await deleteProvider(id)
      window.$message?.success?.(t('common.deleted'))
      await fetchProviders()
    } catch (e) {
      window.$message?.error?.(t('common.delete_failed'))
      throw e
    }
  }

  async function testConn(id) {
    return await testProvider(id)
  }

  async function refresh(id) {
    return await refreshModels(id)
  }

  async function fetchProfiles() { profiles.value = [] }
  async function addProfile() { window.$message?.info?.('Profiles moved to Global Profiles'); }
  async function saveProfile() { window.$message?.info?.('Profiles moved to Global Profiles'); }
  async function removeProfile() { window.$message?.info?.('Profiles moved to Global Profiles'); }

  const providerMap = computed(() => {
    const list = Array.isArray(providers.value) ? providers.value.filter(Boolean) : []
    return Object.fromEntries(list.map(p => [p.id, p]))
  })

  return {
    providers, providerTypes, profiles, loading,
    fetchProviders, addProvider, saveProvider, removeProvider,
    testConn, refresh,
    fetchProfiles, addProfile, saveProfile, removeProfile,
    providerMap,
  }
}
