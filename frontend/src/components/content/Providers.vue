<template>
  <div class="sr-root" :style="{ '--left-col': leftWidth }">
    <!-- 左列：Provider 列表（移除“提供商”标题行） -->
    <aside class="sr-left" :style="{ width: leftWidth, minWidth: leftWidth }">
      <div class="sr-left-scroll">
        <div class="source-group">
          <div v-for="it in providerListItems" :key="it.key"
               class="source-chip"
               :class="{ 'ribbon-active active': selectedProviderId===it.id }"
               @click="selectInstance(it.id)">
            <span class="icon-cell"><Icon name="database" class="source-row-icon" /></span>
            <span class="label-cell"><span class="source-row-label truncate">{{ it.name }}</span></span>
          </div>
        </div>
      </div>
      <div class="sr-left-actions">
        <button ref="addBtnRef" class="icon-chip-ghost" :data-tooltip="t('providers.add')" :aria-label="t('providers.add')" data-tip-pos="top" @click.stop="toggleAddMenu"><Icon name="plus" class="w-4 h-4"/></button>
        <button class="icon-chip-ghost" :data-tooltip="t('common.delete')" :aria-label="t('common.delete')" data-tip-pos="top" :disabled="!canDeleteCurrent" @click="onLeftDelete"><Icon name="minus" class="w-4 h-4"/></button>
        <button class="icon-chip-ghost danger" style="margin-left:auto" :title="t('providers.init_danger')" :data-tooltip="t('providers.init_danger')" @click="onInitBolt"><Icon name="status-warning" class="w-4 h-4"/></button>
      </div>
      
      <!-- Add menu popover (teleported to body to avoid clipping) -->
      <teleport to="body">
        <PopoverMenu v-if="showAddMenu"
          ref="addMenuEl"
          class="add-popover"
          :items="addMenuItems"
          :style="[addMenuStyle, { visibility: addMenuVisible ? 'visible' : 'hidden' }]"
          @select="onAddMenuSelect"
        />
      </teleport>
    </aside>

    <!-- 右列：标题 + 分隔 + 卡片内容 -->
    <section class="sr-right">
      <div class="sr-section-head">
        <div class="sr-section-title">{{ headerTitle }}</div>
      </div>
      <div class="sr-section-divider"></div>

      <div class="sr-card card-frosted card-translucent">
        <div class="sr-card-body">
          <!-- 详情：已选中 Provider（仅 Custom Provider 可编辑名称） -->
          <template v-if="mode==='provider' && currentProvider">
            <!-- Provider name field (custom providers only) -->
            <div class="field" v-if="isCustomProvider(currentProvider)">
              <div class="label label-strong">{{ t('providers.provider_name') }}</div>
              <div class="field-input">
                <input v-model="formProv.name" class="input-macos input-full" placeholder="Provider" @blur="onFieldBlur" />
              </div>
            </div>
            <!-- API Key field -->
            <div class="field">
              <div class="label label-strong">API Key</div>
              <div class="field-input">
                <input :type="showKey ? 'text' : 'password'"
                       :value="apiKeyDisplay"
                       class="input-macos input-full"
                       placeholder="sk-..."
                       @input="onKeyInput($event.target.value)"
                       @blur="onFieldBlur" />
                <button class="icon-input-end" @click="showKey = !showKey" :aria-label="showKey ? t('common.hide') : t('common.show')" :data-tooltip="showKey ? t('common.hide') : t('common.show')">
                  <Icon :name="showKey ? 'eye' : 'eye-off'" class="w-4 h-4" />
                </button>
              </div>
            </div>

            <!-- Base URL field -->
            <div class="field">
              <div class="label label-strong">{{ t('providers.api_base_url') }}</div>
              <div class="field-input">
                <input v-model="formProv.base_url"
                       class="input-macos input-full"
                       :placeholder="baseUrlPlaceholder"
                       @blur="onFieldBlur" />
              </div>
            </div>

            <div class="sr-section-divider subtle"></div>
            <div class="profiles-head">
              <span>{{ t('providers.profiles_title') }}</span>
              <button class="btn-chip-ghost btn-sm" @click="openNewProfile"
                      :disabled="!(currentProvider && (currentProvider.models || []).length)">
                <Icon name="plus" class="w-4 h-4 mr-1" />{{ t('providers.profiles_add') }}
              </button>
            </div>
            <div v-if="providerProfiles.length" class="profile-list">
              <div class="list-header">
                <div class="col">{{ t('providers.profile_model') }}</div>
                <div class="col">{{ t('providers.profile_temperature') }}</div>
                <div class="col">{{ t('providers.profile_top_p') }}</div>
                <div class="col">{{ t('providers.profile_json_mode') }}</div>
                <div class="col actions"></div>
              </div>
              <div class="list-row" v-for="prof in providerProfiles" :key="prof.id">
                <div class="col">{{ prof.model || '-' }}</div>
                <div class="col">{{ typeof prof.temperature === 'number' ? prof.temperature : '-' }}</div>
                <div class="col">{{ typeof prof.top_p === 'number' ? prof.top_p : '-' }}</div>
                <div class="col">
                  <span :class="['badge', prof.json_mode ? 'badge-primary' : 'badge-ghost']">
                    {{ prof.json_mode ? t('providers.profile_json_mode_on') : t('providers.profile_json_mode_off') }}
                  </span>
                </div>
                <div class="col actions">
                  <button class="icon-chip-ghost" :data-tooltip="t('common.edit')" data-tip-pos="top" @click="beginEditProfile(prof)">
                    <Icon name="edit" class="w-4 h-4" />
                  </button>
                  <button class="icon-chip-ghost danger" :data-tooltip="t('common.delete')" data-tip-pos="top" @click="onDeleteProfile(prof)">
                    <Icon name="trash" class="w-4 h-4" />
                  </button>
                </div>
              </div>
            </div>
            <div v-else class="profile-empty">
              {{ t('providers.profiles_empty') }}
            </div>

            <!-- 已去除显式保存按钮；失焦即自动保存 -->
          </template>

          <!-- 新建 Provider 表单（兼容风格） -->
          <template v-else-if="mode==='create'">
            <div class="detail-head">
              <div class="title">{{ t('providers.create_title') }}</div>
            </div>
            <div class="field">
              <div class="label label-strong">{{ t('providers.provider_name') }}</div>
              <div class="field-input">
                <input v-model="newProv.name" class="input-macos input-full" placeholder="Custom Provider 1" @blur="onCreateIfReady" />
              </div>
            </div>
            <div class="field">
              <div class="label label-strong">{{ t('providers.api_base_url') }}</div>
              <div class="field-input">
                <input v-model="newProv.base_url" class="input-macos input-full" :placeholder="createBaseUrlPlaceholder" @blur="onCreateIfReady" />
              </div>
              <div class="field-hint">{{ t('providers.base_url_hint') }}</div>
            </div>
            <div class="field">
              <div class="label label-strong">{{ t('providers.api_key') }}</div>
              <div class="field-input">
                <input :type="showKeyCreate ? 'text' : 'password'" v-model="newProv.api_key" class="input-macos input-full" placeholder="sk-xxxx" />
                <button class="icon-input-end" @click="showKeyCreate = !showKeyCreate" :aria-label="showKeyCreate ? t('common.hide') : t('common.show')" :data-tooltip="showKeyCreate ? t('common.hide') : t('common.show')">
                  <Icon :name="showKeyCreate ? 'eye' : 'eye-off'" class="w-4 h-4" />
                </button>
              </div>
            </div>
          </template>

          <!-- 空态 -->
          <template v-else>
            <div class="empty">
              <div class="hint">{{ t('providers.empty_hint') }}</div>
              </div>
          </template>
        </div>
      </div>
    </section>
  </div>
</template>

<script setup>
import { ref, onMounted, onBeforeUnmount, computed, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import useLLMStore from '@/stores/llm.js'
import { createProvider as apiCreateProvider, resetLLMData, canDelete, listAddableProviders } from '@/services/llmProviderService.js'
import useLayoutStore from '@/stores/layout.js'
import PopoverMenu from '@/components/common/PopoverMenu.vue'

const llm = useLLMStore()
const layout = useLayoutStore()
const { t } = useI18n()
const providersList = computed(() => {
  const list = Array.isArray(llm.providers?.value) ? llm.providers.value : (Array.isArray(llm.providers) ? llm.providers : [])
  return [...(list||[])].sort((a,b) => (a?.name||'').localeCompare(b?.name||'', undefined, {sensitivity:'base'}))
})
const profilesList = computed(() => Array.isArray(llm.profiles?.value) ? llm.profiles.value : (Array.isArray(llm.profiles) ? llm.profiles : []))
// 左侧列表：仅实例
const providerListItems = computed(() => (providersList.value || []).map(p => ({ key: 'inst:' + p.id, id: p.id, name: p.name })))

const leftWidth = computed(() => (layout.ribbonWidth || 160) + 'px')
const selectedProviderId = ref('')
const mode = ref('') // '', 'provider', 'create'
function selectInstance(id){ selectedProviderId.value = id || ''; mode.value = 'provider'; prepareFormForCurrent() }

// 根据 vendor 匹配预设
// 当前 Provider：按 ID 精确匹配
const currentProvider = computed(() => {
  if (mode.value !== 'provider') return null
  const list = providersList.value || []
  if (selectedProviderId.value) {
    return list.find(x => x?.id === selectedProviderId.value) || null
  }
  return null
})

watch(currentProvider, (cp) => {
  if (mode.value === 'provider' && cp) {
    prepareFormForCurrent()
  }
})

const formProv = ref({ name:'', base_url:'', enabled:true, api_key_input:'' })
const defaultRateLimit = () => ({ rps: 2, rpm: 120, burst: 4, concurrency: 4 })
const defaultNewProv = () => ({ name:'', base_url:'', api_key:'', enabled:true, rate_limit: defaultRateLimit() })
const newProv = ref(defaultNewProv())
const showKey = ref(false)
const showKeyCreate = ref(false)
const showAddMenu = ref(false)
const addBtnRef = ref(null)
const addMenuEl = ref(null)
const addMenuStyle = ref({ position: 'fixed', left: '8px', top: '8px', zIndex: 9999 })
const addMenuVisible = ref(false)
let addMenuRO = null
const addables = ref({ special: [], presets: [] })
const creationType = ref('openai_compat')
const creatingNew = ref(false)

function prepareFormForCurrent(){
  const cp = currentProvider.value
  if (cp) {
    formProv.value = {
      name: cp.name,
      base_url: cp.base_url,
      enabled: cp.enabled,
      api_key_input: cp.api_key || '',
    }
  }
}
// API key is fully visible by default in local client
const apiKeyDisplay = computed(() => formProv.value?.api_key_input || '')
function onKeyInput(v){ formProv.value.api_key_input = v }

// 保存当前选择（预设或自定义）；不存在则按预设数据创建
async function onSavePreset(){
  const existing = currentProvider.value
  if (!existing) return
  // 仅 custom 可改名
  const nextName = (isCustomProvider(existing) && (formProv.value?.name || '').trim()) ? (formProv.value?.name || '').trim() : existing.name
  const payload = { name: nextName, enabled: existing?.enabled ?? true }
  const base = (formProv.value.base_url || existing.base_url || '').trim(); if (base) payload.base_url = base
  const key = (formProv.value.api_key_input || '').trim(); if (key) payload.api_key = key
  await llm.saveProvider(existing.id, payload)
}

// 自定义判断：按 policy
const isCustomProvider = (p) => {
  if (!p) return false
  return String(p.policy || '').toLowerCase() === 'custom'
}
// 可删除：
// - 直接选中实例项 → 使用其 id
// - 若选中预设项，且当前预设在已配置列表中存在对应实例（按名称匹配）→ 允许删除该实例
const canDeleteCurrent = computed(() => currentProvider.value ? canDelete(currentProvider.value) : false)

async function onCreateProvider(){
  if (!newProv.value.name || !newProv.value.base_url) { window.$message?.error?.('名称/BaseURL 必填'); return }
  const res = await llm.addProvider({ ...newProv.value, type: 'openai_compat', policy: 'custom' })
  let selId = res?.id
  if (!selId) {
    const first = (providersList.value || [])[0]
    selId = first?.id || ''
  }
  if (selId) {
    selectedProviderId.value = selId
    mode.value = 'provider'
    prepareFormForCurrent()
  }
  newProv.value = defaultNewProv()
}

async function onLeftDelete(){
  const id = selectedProviderId.value
  if (!id || !currentProvider.value) { window?.$message?.warning?.('请选择可删除的 Provider'); return }
  // resolve provider name for confirmation text
  let provName = ''
  try {
    const list = providersList.value || []
    provName = (list.find(x => x?.id === selectedProviderId.value)?.name || '')
  } catch {}
  const msg = t('common.delete_confirm_detail', { title: provName || t('providers.provider') })
  const ask = () => new Promise(resolve => {
    if (window?.$dialog?.confirm) {
      window.$dialog.confirm(msg, {
        title: t('common.delete_confirm'),
        positiveText: t('common.delete'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
      })
    } else {
      resolve(confirm(msg))
    }
  })
  const ok = await ask()
  if (!ok) return
  try {
    await llm.removeProvider(id)
    await llm.fetchProfiles()
  } catch (e) {
    throw e
  }
  const next = (providersList.value || [])[0]
  if (next) {
    selectedProviderId.value = next.id
    mode.value = 'provider'
    prepareFormForCurrent()
  } else {
    selectedProviderId.value = ''
    mode.value = ''
  }
}

// Actions
async function onTest(p){ const r = await llm.testConn(p.id); if (r?.ok) window.$message?.success?.('连接成功，模型数：' + (r.models?.length||0)); else window.$message?.error?.('连接失败：' + (r?.error||'')) }
async function onRefreshModels(p){ const r = await llm.refresh(p.id); if (r?.ok) window.$message?.success?.('已刷新模型：' + (r.models?.length||0)); else window.$message?.error?.('刷新失败：' + (r?.error||'')) }

// Profiles filtered by current provider
const providerProfiles = computed(() => {
  if (!currentProvider.value) return []
  const list = profilesList.value || []
  return list.filter(x => x?.provider_id === currentProvider.value.id)
})
async function openNewProfile(){
  if (!currentProvider.value) return
  const models = Array.isArray(currentProvider.value.models) ? currentProvider.value.models.filter(Boolean) : []
  if (!models.length) {
    // 需要先探测/刷新模型列表，避免后端校验报错（provider_id/model required）
    window.$message?.warning?.('请先测试/刷新模型，再新增配置')
    return
  }
  const p = { provider_id: currentProvider.value.id, model: models[0], temperature: 0.2, top_p: 1, json_mode: true, max_tokens: 2048, cost_weight: 1.0 }
  await llm.addProfile(p)
}
function beginEditProfile(p){ window.$message?.info?.('请在后续版本使用详细编辑器'); }
async function onDeleteProfile(p){
  const ok = await new Promise(resolve => {
    if (window?.$dialog?.confirm) {
      window.$dialog.confirm(t('providers.profile_delete_confirm'), {
        title: t('common.delete_confirm'),
        positiveText: t('common.delete'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => resolve(true),
        onNegativeClick: () => resolve(false),
      })
    } else {
      resolve(confirm(t('providers.profile_delete_confirm')))
    }
  })
  if (!ok) return
  await llm.removeProfile(p.id)
}

async function loadAll(){
  await Promise.all([llm.fetchProviders(), llm.fetchProfiles()])
  const first = (providersList.value || [])[0]
  if (first && !selectedProviderId.value) {
    selectedProviderId.value = first.id
    mode.value = 'provider'
    prepareFormForCurrent()
  }
}
onMounted(loadAll)

async function toggleAddMenu(){
  if (showAddMenu.value) { closeAddMenu(); return }
  try {
    // open then position after next tick to get menu size
    showAddMenu.value = true
    await nextTick()
    // load addable items first (may affect menu size)
    try { addables.value = (await listAddableProviders()) || { special: [], presets: [] } } catch {}
    await nextTick()
    // observe size to reposition while content/layout settles
    try {
      if (window.ResizeObserver && addMenuEl.value) {
        addMenuRO = new ResizeObserver(() => positionAddMenu())
        const el = addMenuEl.value?.$el || addMenuEl.value
        if (el) addMenuRO.observe(el)
      }
    } catch {}
    // defer placement to next animation frame to ensure layout is ready
    requestAnimationFrame(() => {
      positionAddMenu()
      addMenuVisible.value = true
    })
  } catch {}
}
function closeAddMenu(){
  showAddMenu.value = false
  addMenuVisible.value = false
  try { if (addMenuRO) { addMenuRO.disconnect(); addMenuRO = null } } catch {}
}
function positionAddMenu(){
  try {
    const btn = addBtnRef.value
    const menu = addMenuEl.value?.$el || addMenuEl.value
    if (!btn || !menu) return
    const br = btn.getBoundingClientRect()
    const mr = menu.getBoundingClientRect()
    const gap = 8
    const mW = mr.width || menu.offsetWidth || 240
    const mH = mr.height || menu.offsetHeight || 120
    let left = br.left
    let top = br.top - mH - gap // prefer above
    if (top < 4) top = Math.min(br.bottom + gap, window.innerHeight - mH - 4) // flip below and clamp
    if (left + mW > window.innerWidth - 4) left = Math.max(4, window.innerWidth - mW - 4)
    addMenuStyle.value = { position: 'fixed', left: left + 'px', top: top + 'px', zIndex: 9999 }
  } catch {}
}
async function onAddPickCompat(kind = 'openai_compat'){
  try {
    // 直接新建一个自定义 Provider，并启用
    creationType.value = kind || 'openai_compat'
    const customIndex = ((providersList.value || []).filter(p => /^custom provider/i.test(p?.name || '')).length || 0) + 1
    const name = `Custom Provider ${customIndex}`
    const res = await apiCreateProvider({ type: creationType.value, policy: 'custom', name, base_url: '', api_key: '', enabled: true, rate_limit: { rps: 2, rpm: 120, burst: 4, concurrency: 4 } })
    await llm.fetchProviders()
    selectedProviderId.value = res?.id || ''
    mode.value = 'provider'
    prepareFormForCurrent()
  } catch (e) {
    window?.$message?.error?.(t('providers.create_failed'))
  } finally {
    closeAddMenu()
  }
}

async function onAddHiddenPreset(preset){
  try {
    // 后端已提供更新接口，直接启用该隐藏预设
    await llm.saveProvider(preset.id, { enabled: true })
    await llm.fetchProviders()
    selectedProviderId.value = preset.id
    mode.value = 'provider'
    prepareFormForCurrent()
  } catch (e) {
    window?.$message?.error?.(t('providers.enable_preset_failed'))
  } finally {
    closeAddMenu()
  }
}

// Build items for add menu (reuse PopoverMenu)
const addMenuItems = computed(() => {
  const items = []
  const sp = Array.isArray(addables.value?.special) ? addables.value.special : []
  sp.forEach(s => {
    const key = `sp:${s.type}`
    const labelKey = s.type === 'openai_compat' ? 'providers.add_openai' : 'providers.add_anthropic'
    items.push({ key, labelKey, icon: 'plus' })
  })
  const presets = Array.isArray(addables.value?.presets) ? addables.value.presets : []
  if (presets.length && sp.length) items.push({ key: 'div:presets', type: 'divider' })
  presets.forEach(p => {
    items.push({ key: `pre:${p.id}`, label: p.name, icon: 'database' })
  })
  return items
})

function onAddMenuSelect(it){
  if (!it || !it.key) return
  if (String(it.key).startsWith('sp:')) {
    const kind = String(it.key).slice(3)
    return onAddPickCompat(kind)
  }
  if (String(it.key).startsWith('pre:')) {
    const id = String(it.key).slice(4)
    const preset = (addables.value?.presets || []).find(p => String(p.id) === id)
    return onAddHiddenPreset(preset)
  }
}
// onAddPickPreset 已移除：统一采用 policy=custom 的新增

async function onCreateIfReady(){
  if (creatingNew.value) return
  const name = (newProv.value?.name || '').trim()
  const base = (newProv.value?.base_url || '').trim()
  if (!name || !base) return
  creatingNew.value = true
  try {
    const res = await apiCreateProvider({ type: creationType.value || 'openai_compat', policy: 'custom', name, base_url: base, api_key: (newProv.value.api_key||'').trim(), enabled: true, rate_limit: { rps: 2, rpm: 120, burst: 4, concurrency: 4 } })
    await llm.fetchProviders()
    selectedProviderId.value = res?.id || ''
    mode.value = 'provider'
    prepareFormForCurrent()
  } catch (e) {
    window?.$message?.error?.('创建失败：' + (e?.message || ''))
  } finally {
    creatingNew.value = false
  }
}
onMounted(() => { document.addEventListener('click', closeAddMenu); window.addEventListener('resize', positionAddMenu); window.addEventListener('scroll', positionAddMenu, true) })
onBeforeUnmount(() => {
  document.removeEventListener('click', closeAddMenu)
  window.removeEventListener('resize', positionAddMenu)
  window.removeEventListener('scroll', positionAddMenu, true)
  try { if (addMenuRO) { addMenuRO.disconnect(); addMenuRO = null } } catch {}
})

// Header title: 当前 Provider/预设名称
const headerTitle = computed(() => (currentProvider.value?.name || t('settings.model_provider')))

// 自动保存（失焦时）
async function onFieldBlur(){ try { await onSavePreset() } catch (e) {} }

// Placeholder for base URL in provider edit form
const baseUrlPlaceholder = computed(() => (currentProvider.value && (currentProvider.value.base_url || '')) || 'https://...')

// Placeholder for base URL in create form
const createBaseUrlPlaceholder = computed(() => (creationType.value === 'anthropic_compat' ? 'https://api.anthropic.com/v1' : 'https://api.openai.com/v1'))

// One-click initialize: clear LLM data and import presets into Bolt
async function onInitBolt(){
  const msg = t('providers.init_confirm')
  const ok = await new Promise(resolve => {
    if (window?.$dialog?.confirm) {
      window.$dialog.confirm(msg, { title: t('providers.init_title'), positiveText: t('providers.init_action'), negativeText: t('common.cancel'), onPositiveClick: () => resolve(true), onNegativeClick: () => resolve(false) })
    } else { resolve(confirm(msg)) }
  })
  if (!ok) return
  try {
    await resetLLMData()
    await Promise.all([llm.fetchProviders(), llm.fetchProfiles()])
    window?.$message?.success?.(t('providers.init_done'))
  } catch (e) {
    window?.$message?.error?.(t('providers.init_failed') + (e?.message ? `: ${e.message}` : ''))
  }
}
</script>

<style scoped>
/* 1:1 复刻 Settings 的两栏结构和风格 */
.sr-root { position: absolute; inset: 0; display: grid; grid-template-columns: var(--left-col, 160px) 1fr; overflow: hidden; }
.sr-left { position: relative; z-index: 1; padding: 6px; display: flex; flex-direction: column; gap: 6px; justify-content: flex-start; }
.sr-left { min-height: 0; --cmd-area-h: 44px; }
.sr-left-scroll { flex: 0 1 auto; height: calc(100% - var(--cmd-area-h)); min-height: 0; overflow-y: auto; overflow-x: hidden; padding-right: 10px; scrollbar-gutter: stable; }
.sr-right { position: relative; z-index: 1; background: var(--macos-background); padding: 12px; overflow: auto; font-size: var(--fs-base); height: 100%; }
.sr-item, .sr-item:hover, .sr-item.active, .sr-item-label { /* deprecated: replaced by .source-chip styles */ }
.sr-left-actions { position: absolute; bottom: 0; left: 6px; right: 6px; height: var(--cmd-area-h); display: flex; align-items: center; gap: 8px; padding: 8px 0; border-top: 1px solid var(--macos-divider-weak); background: transparent; }
.sr-group-title { font-size: var(--fs-sub); color: var(--macos-text-secondary); margin: 4px 4px 6px; }

.sr-section-head { display: flex; align-items: center; justify-content: space-between; padding: 2px 2px 6px 2px; }
.sr-section-title { font-size: var(--fs-base); color: var(--macos-text-secondary); letter-spacing: .3px; font-weight: 600; }
.sr-section-divider { height: 1px; background: var(--macos-divider-weak); margin: 0 -12px 10px -12px; }
.sr-section-divider.subtle { opacity: 0.6; margin-top: 12px; }

.sr-card { border-radius: 10px; padding: 12px; max-width: 100%; box-sizing: border-box; }
.sr-card-body { padding-top: 0; }
.sr-card-body .sr-row { border-bottom: 1px solid var(--macos-divider-weak); margin: 0 8px; }
.sr-card-body .sr-row:last-child { border-bottom: none; }
/* 创建表单网格布局（保留原样式以不影响“新建 Provider”） */
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(220px, 1fr)); gap: 10px; align-items: start; }
.form-grid .subgrid { display: grid; grid-template-columns: repeat(4, minmax(80px, 1fr)); gap: 8px; }

/* 字段两行布局：第一行 label 左对齐，第二行输入框铺满，眼睛图标右对齐叠放 */
.field { display: flex; flex-direction: column; gap: 6px; padding: 6px 2px; }
.field + .field { border-top: 1px dashed var(--macos-separator-weak); }
/* 使用全局 .label/.label-strong 样式替代 .field-label */
.field-input { position: relative; }
.field-input .input-macos { width: 100%; padding-right: 30px; }
.icon-input-end { position: absolute; top: 50%; right: 6px; transform: translateY(-50%); width: 24px; height: 24px; display: inline-flex; align-items: center; justify-content: center; border-radius: 6px; background: transparent; border: 1px solid transparent; color: var(--macos-text-secondary); }
.icon-input-end:hover { background: var(--macos-gray-hover); }

/* use global icon-chip-ghost; keep danger tone */

/* Add popover placement near left actions */
/* style teleported popover via deep selector so it applies to child component root */
:deep(.add-popover) { min-width: 180px; width: max-content; max-width: 360px; max-height: 420px; overflow-y: auto; overflow-x: hidden; pointer-events: auto; }
:deep(.add-popover .popover-item) { white-space: nowrap; }
:deep(.add-popover .popover-divider) { height: 1px; background: var(--macos-divider-weak); margin: 6px -8px; }

/* Popover 基础样式改由全局提供（styles/macos-components.scss） */

.field-hint { font-size: 11px; color: var(--macos-text-tertiary); margin-top: 4px; }

.profiles-head { display: flex; align-items: center; justify-content: space-between; font-weight: 600; color: var(--macos-text-secondary); margin: 10px 2px 6px; gap: 12px; }
/* 保留用于“新建 Provider”头部的样式 */
.detail-head { display: flex; align-items: center; justify-content: space-between; padding: 4px 2px 10px; }
.detail-head .title { font-weight: 600; color: var(--macos-text-secondary); }
.detail-head .actions { display: flex; gap: 8px; }
.profile-list { border-radius: 8px; overflow: hidden; border: 1px solid var(--macos-divider-weak); }
.list-header, .list-row { display: grid; grid-template-columns: 1.2fr .7fr .7fr .9fr .6fr; gap: 8px; align-items: center; padding: 6px 8px; }
.list-header { font-weight: 600; color: var(--macos-text-secondary); background: var(--macos-background-weak); border-bottom: 1px solid var(--macos-separator); }
.list-row { border-bottom: 1px dashed var(--macos-separator-weak); }
.list-row:last-child { border-bottom: none; }
.col { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.col.actions { display: flex; justify-content: flex-end; gap: 6px; }
.profile-empty { margin: 12px 0; padding: 16px; border-radius: 8px; background: var(--macos-background-weak); color: var(--macos-text-tertiary); text-align: center; }
.profile-add-row { padding: 8px 0; display: flex; justify-content: flex-end; }

.empty { display: grid; place-items: center; height: 280px; color: var(--macos-text-tertiary); }
.danger { color: var(--macos-danger-text); }

:host, .sr-root { --left-col: 160px; }
</style>
