<template>
  <div
    v-if="show"
    class="macos-modal"
    role="dialog"
    aria-modal="true"
  >
    <div class="modal-card">
      <div class="modal-header">
        <ModalTrafficLights @close="handleClose" />
        <div class="title">
          {{ mode === 'edit' ? $t('cookies.manual_edit_title') : $t('cookies.manual_create_title') }}
        </div>
      </div>
      <div class="modal-body">
        <form class="form-grid" @submit.prevent="submit">
          <div class="section-card name-card">
            <div class="section-header name-header">
              <div class="section-heading">
                <Icon name="edit" class="w-4 h-4" />
                <span>{{ $t('common.name') }}</span>
              </div>
              <div class="name-inline">
                <input
                  id="manual-collection-name"
                  v-model="form.name"
                  type="text"
                  :placeholder="$t('cookies.manual_default_name')"
                  @blur="handleNameBlur"
                  @keyup.enter.prevent="handleNameBlur"
                />
              </div>
            </div>
          </div>

          <div class="section-card">
            <div class="section-header">
              <div class="section-heading">
                <Icon :name="isEditing ? 'layers' : 'globe'" class="w-4 h-4" />
                <span>{{ isEditing ? $t('cookies.manual_paste_label') : $t('cookies.manual_preview_label') }}</span>
                <span v-if="!isEditing && preview.total" class="section-badge">{{ $t('cookies.manual_preview_title', { count: preview.total }) }}</span>
              </div>
              <div class="section-actions">
                <template v-if="isEditing">
                  <button type="button" class="btn-glass" @click="cancelEdit">{{ $t('common.cancel') }}</button>
                  <button type="button" class="btn-primary" @click="applyEdit">{{ $t('common.save') }}</button>
                </template>
                <button v-else type="button" class="btn-glass" @click="startEdit">{{ $t('common.edit') }}</button>
              </div>
            </div>

            <div class="section-body">
              <template v-if="isEditing">
                <div class="field">
                  <span>{{ $t('cookies.manual_input_format_hint') }} {{ $t('cookies.manual_save_hint_multi') }}</span>
                  <textarea
                    v-model="editBuffer"
                    rows="12"
                    :placeholder="$t('cookies.manual_paste_placeholder')"
                  />
                </div>
                <div class="field inline">
                  <div class="label-stack">
                    <span>{{ $t('cookies.manual_default_domain_label') }}</span>
                    <p class="hint">{{ $t('cookies.manual_default_domain_hint') }}</p>
                  </div>
                  <input
                    v-model="defaultDomain"
                    type="text"
                    :placeholder="$t('cookies.manual_default_domain_placeholder')"
                  />
                </div>
                <div class="section-hints">
                  <p class="hint">{{ $t('cookies.manual_netscape_hint') }} {{ $t('cookies.manual_json_hint') }} {{ $t('cookies.manual_header_hint') }}</p>
    
                </div>
              </template>

              <template v-else>
                <div v-if="preview.entries.length" class="preview-wrapper">
                  <div class="preview-list">
                    <div v-for="(cookie, index) in preview.entries" :key="index" class="preview-item">
                      <div class="primary-row">
                        <span class="cookie-name">{{ cookie.name }}</span>
                        <span class="cookie-domain" :title="cookie.domain">{{ cookie.domain }}</span>
                      </div>
                      <div class="secondary-row">
                        <span class="mono">{{ cookie.value }}</span>
                        <span class="meta">{{ cookie.path }}</span>
                        <span v-if="cookie.secure" class="meta">Secure</span>
                        <span v-if="cookie.includeSubdomains" class="meta">Subdomain</span>
                        <span v-if="cookie.expires" class="meta">{{ cookie.expires }}</span>
                      </div>
                    </div>
                  </div>
                  <div v-if="preview.total > preview.entries.length" class="preview-foot">
                    {{ $t('cookies.manual_preview_more', { total: preview.total }) }}
                  </div>
                </div>
                <div v-else class="empty-preview">
                  <Icon name="layers" class="w-6 h-6" />
                  <div class="title">{{ $t('cookies.manual_preview_empty') }}</div>
                  <div class="desc">{{ $t('cookies.manual_preview_hint') }}</div>
                </div>
              </template>
            </div>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script setup>
import { reactive, watch, computed, getCurrentInstance, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ModalTrafficLights from '@/components/common/ModalTrafficLights.vue'
import Icon from '@/components/base/Icon.vue'

const props = defineProps({
  show: { type: Boolean, default: false },
  mode: { type: String, default: 'create' },
  collection: { type: Object, default: null }
})

const emit = defineEmits(['close', 'submit'])

const { t } = useI18n()
const { proxy } = getCurrentInstance() || {}

const form = reactive({
  name: '',
  netscape: ''
})

const editBuffer = ref('')
const isEditing = ref(false)
const defaultDomain = ref('')
const lastSavedName = ref('')

const preview = computed(() => buildPreview(form.netscape))

watch(() => props.show, (visible) => {
  if (!visible) return
  initialiseForm()
})

watch(() => props.collection, () => {
  if (props.show) initialiseForm()
})

function initialiseForm() {
  const existing = generateNetscapeFromCollection(props.collection)
  form.name = props.collection?.name || ''
  form.netscape = existing
  editBuffer.value = existing
  const firstDomain = deriveFirstDomain(existing)
  defaultDomain.value = firstDomain
  isEditing.value = existing.trim().length === 0
  lastSavedName.value = form.name
}

function handleClose() {
  emit('close')
}

function submit() {
  const payload = buildPayload()
  if (!payload) return
  emit('submit', payload)
}

function startEdit() {
  editBuffer.value = form.netscape
  isEditing.value = true
}

function cancelEdit() {
  editBuffer.value = form.netscape
  if (!form.netscape.trim()) {
    isEditing.value = true
    return
  }
  isEditing.value = false
}

function applyEdit() {
  const raw = (editBuffer.value || '').trim()
  if (!raw) {
    showDialog(t('cookies.manual_empty_warning'))
    return
  }
  try {
    const result = parseCookieInput(raw, defaultDomain.value.trim())
    form.netscape = result.netscape
    editBuffer.value = result.original ?? raw
    if (result.defaultDomain) {
      defaultDomain.value = result.defaultDomain
    }
    isEditing.value = false
    persistManualCollection()
  } catch (error) {
    const message = error?.message || t('cookies.manual_parse_failed')
    showDialog(message)
  }
}

function handleNameBlur() {
  const current = form.name.trim()
  if (current === lastSavedName.value.trim()) {
    return
  }
  if (!form.netscape.trim()) {
    lastSavedName.value = form.name
    return
  }
  persistManualCollection()
}

function buildPayload() {
  const netscape = form.netscape.trim()
  if (!netscape) {
    showDialog(t('cookies.manual_empty_warning'))
    return null
  }
  return {
    name: form.name,
    netscape,
    cookies: [],
    replace: true
  }
}

function persistManualCollection() {
  const netscape = form.netscape.trim()
  if (!netscape) {
    return false
  }
  const payload = buildPayload()
  if (!payload) {
    return false
  }
  emit('submit', payload)
  return true
}

function showDialog(message) {
  if (window?.$dialog?.warning) {
    window.$dialog.warning({ content: message, positiveText: t('common.confirm') })
    return
  }
  if (window?.$message?.warning) {
    window.$message.warning(message)
    return
  }
  if (typeof proxy?.$message?.warning === 'function') {
    proxy.$message.warning(message)
    return
  }
  window.alert(message)
}

function parseCookieInput(raw, fallbackDomain) {
  const text = raw.trim()
  if (!text) {
    throw new Error(t('cookies.manual_empty_warning'))
  }

  // Try JSON array
  let jsonCandidate
  try {
    jsonCandidate = JSON.parse(text)
  } catch (err) {
    if (!(err instanceof SyntaxError)) {
      throw err
    }
  }
  if (Array.isArray(jsonCandidate)) {
    const entries = jsonCandidate.flatMap((item) => mapJsonCookie(item, fallbackDomain))
    if (!entries.length) throw new Error(t('cookies.manual_parse_failed'))
    const netscape = entriesToNetscape(entries)
    return {
      netscape,
      original: netscape,
      defaultDomain: entries[0]?.domain?.replace(/^\./, '') || fallbackDomain
    }
  }

  // Netscape format
  const netscapeEntries = parseNetscapeToEntries(text)
  if (netscapeEntries.length) {
    const netscape = entriesToNetscape(netscapeEntries)
    return {
      netscape,
      original: text,
      defaultDomain: netscapeEntries[0]?.domain?.replace(/^\./, '') || fallbackDomain
    }
  }

  // Header string format
  const headerEntries = parseHeaderString(text, fallbackDomain)
  if (headerEntries.length) {
    const netscape = entriesToNetscape(headerEntries)
    return {
      netscape,
      original: netscape,
      defaultDomain: headerEntries[0]?.domain?.replace(/^\./, '') || fallbackDomain
    }
  }

  throw new Error(t('cookies.manual_parse_failed'))
}

function mapJsonCookie(item, fallbackDomain) {
  if (!item || typeof item !== 'object') return []
  const name = item.name ?? ''
  const value = item.value ?? ''
  if (!name) return []

  let domain = item.domain || item.host || ''
  const hostOnly = item.hostOnly ?? (domain ? !domain.startsWith('.') : true)
  if (!domain) {
    domain = fallbackDomain
  }
  if (!domain) {
    throw new Error(t('cookies.manual_header_need_domain'))
  }
  domain = normalizeDomain(domain, hostOnly)

  const path = item.path || '/'
  const secure = Boolean(item.secure)
  const includeSubdomains = !hostOnly
  const httpOnly = Boolean(item.httpOnly ?? item.HttpOnly ?? item['http-only'] ?? item.HTTPOnly)

  let expires = 0
  const expCandidate = item.expirationDate ?? item.expires ?? item.expiry ?? item.Expiry
  if (typeof expCandidate === 'number') {
    expires = expCandidate > 1e12 ? Math.floor(expCandidate / 1000) : Math.floor(expCandidate)
  }

  return [{ domain, name, value, path, secure, includeSubdomains, expires, httpOnly }]
}

function parseHeaderString(text, fallbackDomain) {
  let working = text.trim()
  if (/^cookie\s*:/i.test(working)) {
    working = working.replace(/^cookie\s*:/i, '')
  }

  const domainHint = fallbackDomain?.trim()
  if (!domainHint) {
    throw new Error(t('cookies.manual_header_need_domain'))
  }

  const parts = working.split(';').map((segment) => segment.trim()).filter(Boolean)
  const entries = []
  for (const part of parts) {
    const idx = part.indexOf('=')
    if (idx <= 0) continue
    const name = part.slice(0, idx).trim()
    const value = part.slice(idx + 1).trim()
    if (!name) continue
    entries.push({
      domain: normalizeDomain(domainHint, true),
      name,
      value,
      path: '/',
      secure: false,
      includeSubdomains: false,
      expires: 0,
      httpOnly: false
    })
  }
  return entries
}

function parseNetscapeToEntries(text) {
  const entries = []
  const lines = text.split(/\r?\n/)
  for (const rawLine of lines) {
    const trimmed = rawLine.trim()
    if (!trimmed) continue

    let working = trimmed
    let httpOnly = false
    if (working.toLowerCase().startsWith('#httponly_')) {
      httpOnly = true
      working = working.slice('#HttpOnly_'.length)
    } else if (working.startsWith('#')) {
      continue
    }

    const parts = working.split('\t')
    if (parts.length !== 7) continue
    const [domainRaw, includeSub, path, secureFlag, expiresStr, name, value] = parts
    if (!name) continue
    const domain = domainRaw || ''
    const includeSubdomains = includeSub?.toUpperCase() === 'TRUE'
    const secure = secureFlag?.toUpperCase() === 'TRUE'
    let expires = 0
    if (expiresStr && expiresStr !== '0') {
      const num = Number(expiresStr)
      if (!Number.isNaN(num)) {
        expires = num
      }
    }
    entries.push({
      domain: domain || '',
      name,
      value,
      path: path || '/',
      secure,
      includeSubdomains,
      expires,
      httpOnly
    })
  }
  return entries
}

function entriesToNetscape(entries) {
  const header = ['# Netscape HTTP Cookie File', '']
  const lines = entries.map((cookie) => {
    const domain = cookie.domain || ''
    if (!domain) {
      throw new Error(t('cookies.manual_header_need_domain'))
    }
    const includeSubdomains = cookie.includeSubdomains ? 'TRUE' : 'FALSE'
    const path = cookie.path || '/'
    const secure = cookie.secure ? 'TRUE' : 'FALSE'
    const expires = cookie.expires && cookie.expires > 0 ? Math.floor(cookie.expires) : 0
    const domainField = cookie.httpOnly ? `#HttpOnly_${domain}` : domain
    return [domainField, includeSubdomains, path, secure, expires, cookie.name, cookie.value].join('\t')
  })
  return [...header, ...lines].join('\n').trim()
}

function buildPreview(netscapeText) {
  const entries = parseNetscapeToEntries(netscapeText || '')
  const limited = entries.slice(0, 8).map((cookie) => ({
    ...cookie,
    domain: cookie.domain,
    expires: cookie.expires ? formatExpires(cookie.expires) : ''
  }))
  return {
    entries: limited,
    total: entries.length
  }
}

function formatExpires(epochSeconds) {
  if (!epochSeconds) return ''
  const date = new Date(epochSeconds * 1000)
  if (Number.isNaN(date.getTime())) return ''
  return date.toISOString().replace('T', ' ').slice(0, 19)
}

function normalizeDomain(domain, hostOnly) {
  let trimmed = domain.trim()
  if (!trimmed) return trimmed
  if (hostOnly) {
    return trimmed.replace(/^\.+/, '')
  }
  return trimmed.startsWith('.') ? trimmed : `.${trimmed}`
}

function deriveFirstDomain(netscape) {
  const entries = parseNetscapeToEntries(netscape || '')
  if (entries.length === 0) return ''
  return entries[0].domain.replace(/^\./, '')
}

function generateNetscapeFromCollection(collection) {
  if (!collection?.domain_cookies) return ''
  const lines = ['# Netscape HTTP Cookie File', '']
  Object.keys(collection.domain_cookies).forEach((domainKey) => {
    const bucket = collection.domain_cookies[domainKey]
    if (!bucket?.cookies) return
    bucket.cookies.forEach((cookie) => {
      if (!cookie) return
      const name = cookie.Name ?? cookie.name ?? ''
      const value = cookie.Value ?? cookie.value ?? ''
      if (!name) return
      const rawDomain = cookie.Domain ?? cookie.domain ?? domainKey ?? ''
      const path = cookie.Path ?? cookie.path ?? '/'
      const secure = (cookie.Secure ?? cookie.secure) ? 'TRUE' : 'FALSE'
      const httpOnly = Boolean(cookie.HttpOnly ?? cookie.httpOnly)
      const includeSubdomains = rawDomain.startsWith('.') ? 'TRUE' : 'FALSE'
      let expires = 0
      const rawExpires = cookie.Expires ?? cookie.expires
      if (rawExpires) {
        if (typeof rawExpires === 'number') {
          const num = rawExpires
          if (!Number.isNaN(num) && num > 0) {
            expires = num > 1e12 ? Math.floor(num / 1000) : Math.floor(num)
          }
        } else if (typeof rawExpires === 'string') {
          const trimmed = rawExpires.trim()
          if (/^-?\d+(?:\.\d+)?$/.test(trimmed)) {
            const num = Number(trimmed)
            if (!Number.isNaN(num) && num > 0) {
              expires = num > 1e12 ? Math.floor(num / 1000) : Math.floor(num)
            }
          } else {
            const parsed = Date.parse(trimmed)
            if (!Number.isNaN(parsed) && parsed > 0) {
              expires = Math.floor(parsed / 1000)
            }
          }
        }
      }
      const domainForLine = httpOnly ? `#HttpOnly_${rawDomain || ''}` : (rawDomain || '')
      lines.push([
        domainForLine,
        includeSubdomains,
        path || '/',
        secure,
        expires,
        name,
        value
      ].join('\t'))
    })
  })
  return lines.join('\n').trim()
}
</script>

<style scoped>
.macos-modal { position: fixed; inset: 0; background: rgba(0,0,0,0.2); z-index: 2000; display:flex; align-items:center; justify-content:center; }
.modal-card { width: 720px; max-width: calc(100% - 40px); background: var(--macos-background); border-radius: 12px; border: 1px solid var(--macos-separator); box-shadow: var(--macos-shadow-3); display:flex; flex-direction: column; overflow:hidden; }
.modal-header { height: 36px; display:flex; align-items:center; justify-content: space-between; padding: 0 10px; border-bottom: 1px solid var(--macos-divider-weak); }
.modal-header .title { font-size: 14px; font-weight: 600; color: var(--macos-text-primary); }
.modal-body { padding: 16px 18px; max-height: 70vh; overflow-y: auto; }
.form-grid { display:flex; flex-direction: column; gap: 16px; }
.field { display:flex; flex-direction: column; gap: 6px; font-size: 12px; color: var(--macos-text-secondary); }
.field input[type="text"], .section-card input[type="text"] {
  width: 100%;
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  padding: 6px 8px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: 13px;
}
.field.inline { flex-direction: row; align-items: center; gap: 12px; }
.inline-center { align-items: center; }
.inline-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--macos-text-primary);
  min-width: 108px;
  display: flex;
  align-items: center;
}
.inline-center input { flex: 1 1 auto; }
.field.inline .label-stack { display:flex; flex-direction: column; gap: 4px; flex: 1 1 auto; }
.section-card { border: 1px solid var(--macos-separator); border-radius: 12px; background: var(--macos-background); padding: 14px; display:flex; flex-direction: column; gap: 14px; }
.section-header { display:flex; align-items:flex-start; justify-content: space-between; gap: 12px; }
.section-heading { display:flex; align-items:center; gap: 8px; font-size: 13px; font-weight: 600; color: var(--macos-text-primary); }
.name-card { gap: 0; }
.name-card .name-header { justify-content: flex-start; align-items: center; gap: 12px; }
.name-card .name-header .name-inline { flex: 1 1 auto; min-width: 0; display:flex; }
.name-card .name-header .name-inline input { width: 100%; }
.section-badge { font-size: 11px; font-weight: 500; color: var(--macos-text-secondary); background: color-mix(in oklab, var(--macos-blue) 14%, transparent); border-radius: 999px; padding: 2px 8px; }
.section-actions { display:flex; align-items:center; gap: 8px; }
.section-body { display:flex; flex-direction: column; gap: 16px; }
.section-hints { display:flex; flex-direction: column; gap: 4px; }
.section-card textarea {
  width: 100%;
  border: 1px solid var(--macos-separator);
  border-radius: 8px;
  padding: 6px 8px;
  background: var(--macos-background);
  color: var(--macos-text-primary);
  font-size: 13px;
  resize: vertical;
  min-height: 200px;
  font-family: var(--macos-mono, ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace);
  line-height: 1.4;
}
.hint { font-size: 11px; color: var(--macos-text-tertiary); }
.hint.primary { color: var(--macos-blue); }
.hint.muted { color: var(--macos-text-secondary); }
.callout { display:flex; align-items:center; gap: 8px; font-size: 12px; padding: 10px; border-radius: 10px; margin-top: 10px; }
.callout.warning { background: var(--macos-warning-background, rgba(255,159,10,0.08)); color: var(--macos-warning-text, #ff9f0a); border: 1px solid var(--macos-warning-border, rgba(255,159,10,0.25)); }
.preview-wrapper { display:flex; flex-direction: column; gap: 10px; }
.preview-list { display:flex; flex-direction: column; gap: 8px; max-height: 180px; overflow-y: auto; }
.preview-item { border: 1px solid var(--macos-divider-weak); border-radius: 8px; padding: 8px 10px; display:flex; flex-direction: column; gap: 4px; background: color-mix(in oklab, var(--macos-background) 96%, rgba(0,0,0,0.02)); }
.primary-row { display:flex; align-items:center; justify-content: space-between; gap: 12px; }
.cookie-name { font-weight: 600; color: var(--macos-text-primary); font-size: 12px; }
.cookie-domain { font-size: 11px; color: var(--macos-text-secondary); max-width: 60%; overflow:hidden; text-overflow: ellipsis; white-space: nowrap; }
.secondary-row { display:flex; flex-wrap: wrap; gap: 8px; font-size: 11px; color: var(--macos-text-tertiary); }
.mono { font-family: var(--macos-mono, ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace); max-width: 100%; overflow:auto; }
.meta { background: color-mix(in oklab, var(--macos-blue) 12%, transparent); color: var(--macos-blue); padding: 0 6px; border-radius: 999px; }
.preview-foot { font-size: 11px; color: var(--macos-text-tertiary); }
.empty-preview { border: 1px dashed var(--macos-separator); border-radius: 10px; padding: 24px 16px; text-align: center; color: var(--macos-text-tertiary); display:flex; flex-direction: column; gap: 6px; align-items:center; }
.empty-preview .title { font-weight: 600; color: var(--macos-text-secondary); }
.empty-preview .desc { font-size: 12px; }
.modal-footer { border-top: 1px solid var(--macos-divider-weak); padding: 10px 16px; display:flex; align-items:center; justify-content: space-between; background: color-mix(in oklab, var(--macos-background) 95%, rgba(0,0,0,0.02)); }
.modal-footer .actions { display:flex; align-items:center; gap: 8px; }
.btn-primary { background: var(--macos-blue); color: white; border: none; border-radius: 8px; padding: 6px 14px; font-size: 13px; transition: background .2s ease; }
.btn-primary:hover { background: color-mix(in oklab, var(--macos-blue) 85%, white); }
.btn-primary:disabled { background: rgba(60,60,67,0.15); color: var(--macos-text-tertiary); cursor: not-allowed; }
.left-hint { font-size: 11px; color: var(--macos-text-tertiary); }
</style>
