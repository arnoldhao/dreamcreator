// 与项目统一：仅使用 Wails v3 生成的绑定 API（bindings/dreamcreator/backend/api/llmapi）。
// 注意：不要在日志中输出 api_key。

import {
  ListProviders,
  ListEnabledProviders,
  CreateProvider,
  UpdateProvider,
  DeleteProvider,
  TestProvider,
  RefreshModels,
  ListGlobalProfiles,
  CreateGlobalProfile,
  UpdateGlobalProfile,
  DeleteGlobalProfile,
  ResetLLMData,
  ListAddableProviders,
} from 'bindings/dreamcreator/backend/api/llmapi'

function toJS(resp) {
  if (!resp || typeof resp !== 'object' || !('success' in resp)) { return resp }
  if (!resp.success) throw new Error(resp.msg || 'request failed')
  let data = resp.data
  if (typeof data === 'string') { try { data = JSON.parse(data) } catch {} }
  return data
}

// Providers
export const listProviders = async () => toJS(await ListProviders())
export const listEnabledProviders = async () => toJS(await ListEnabledProviders())
export const listAddableProviders = async () => toJS(await ListAddableProviders())
export const createProvider = async (p) => toJS(await CreateProvider(p))
export const getProvider = async (id, reveal=false) => {
  const list = toJS(await ListProviders())
  return (list || []).find(x => x?.id === id)
}
export const updateProvider = async (id, p) => toJS(await UpdateProvider(id, p))
export const deleteProvider = async (id) => toJS(await DeleteProvider(id))
export const testProvider = async (id) => toJS(await TestProvider(id))
export const refreshModels = async (id) => toJS(await RefreshModels(id))

// LLM Profiles
// legacy LLMProfile API removed

// Global Profiles (model-agnostic)
export const listGlobalProfiles = async () => toJS(await ListGlobalProfiles())
export const createGlobalProfile = async (p) => toJS(await CreateGlobalProfile(p))
export const updateGlobalProfile = async (id, p) => toJS(await UpdateGlobalProfile(id, p))
export const deleteGlobalProfile = async (id) => toJS(await DeleteGlobalProfile(id))

export const resetLLMData = async () => toJS(await ResetLLMData())

// Policy helpers (front-end convenience)
export const canRename = (p) => String(p?.policy || '').toLowerCase() === 'custom'
export const canDelete = (p) => {
  const pol = String(p?.policy || '').toLowerCase()
  if (pol === 'preset_show') return false
  return true // preset_hidden/custom 均可触发 delete（后端会按策略处理）
}
