<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

import useLLMStore from "@/stores/llm.js"
import { canDelete, canRename, listAddableProviders, resetLLMData } from "@/services/llmProviderService.js"

import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Separator } from "@/components/ui/separator"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Switch } from "@/components/ui/switch"
import { Badge } from "@/components/ui/badge"
import ProviderLogo from "@/components/providers/ProviderLogo.vue"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"

import { ChevronDown, ChevronRight, Eye, EyeOff, Plus, RefreshCw, Trash2, Undo2 } from "lucide-vue-next"

const props = defineProps<{
  activeProviderId?: string
}>()

const emit = defineEmits<{
  (e: "open-provider", id: string): void
  (e: "close-provider"): void
}>()

const { t } = useI18n()
const llm = useLLMStore()

const saving = ref(false)
const showKey = ref(false)
const deleteDialogOpen = ref(false)
const deleting = ref(false)

type Addables = { special?: Array<{ type: string; label: string }>; presets?: any[] }
const addables = ref<Addables>({ special: [], presets: [] })
const loadingAddables = ref(false)

const providers = computed(() => {
  const list = Array.isArray(llm.providers?.value) ? llm.providers.value : Array.isArray(llm.providers) ? llm.providers : []
  return (list || []).filter(Boolean).slice().sort((a: any, b: any) => String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" }))
})

const shouldShowProvider = (p: any) => {
  const pol = String(p?.policy || "").toLowerCase()
  if (pol !== "preset_hidden") return true
  if (p?.enabled) return true
  const hasKey = !!(p?.has_key || (p?.api_key && String(p.api_key).trim()))
  const hasModels = Array.isArray(p?.models) && p.models.length > 0
  return hasKey || hasModels
}

const visibleProviders = computed(() => (providers.value || []).filter(shouldShowProvider))

const hasKey = (p: any) => !!(p?.has_key || (p?.api_key && String(p.api_key).trim()))
const hasModels = (p: any) => Array.isArray(p?.models) && p.models.length > 0
const isAvailable = (p: any) => hasKey(p) && hasModels(p)

const availableProviders = computed(() => (visibleProviders.value || []).filter(isAvailable))
const unconfiguredProviders = computed(() => (visibleProviders.value || []).filter((p: any) => !isAvailable(p)))

const currentId = computed(() => String(props.activeProviderId || ""))
const current = computed(() => (currentId.value ? (providers.value || []).find((p: any) => String(p?.id || "") === currentId.value) : null) || null)
const canRenameCurrent = computed(() => (current.value ? canRename(current.value) : false))
const canDeleteCurrent = computed(() => (current.value ? canDelete(current.value) : false))

function logoKeyFromName(name: string) {
  const raw = String(name || "").trim()
  if (!raw) return ""
  const lower = raw.toLowerCase()
  if (lower.includes("openai")) return "openai"
  if (lower.includes("anthropic") || lower.includes("claude")) return "anthropic"
  if (lower.includes("ollama")) return "ollama"
  if (lower.includes("openrouter")) return "openrouter"
  if (lower.includes("groq")) return "groq"
  if (lower.includes("mistral")) return "mistral"
  if (lower.includes("together")) return "together"
  if (lower.includes("deepseek")) return "deepseek"
  if (lower.includes("perplexity")) return "perplexity"
  if (lower.includes("xai") || lower.includes("x.ai")) return "xai"
  if (lower.includes("github")) return "github"
  if (lower.includes("google")) return "google"
  if (lower.includes("azure")) return "azure"
  return lower.replace(/[^a-z0-9]+/g, "-").replace(/^-+|-+$/g, "")
}

function logoKeyForProvider(p: any) {
  const typ = String(p?.type || "").toLowerCase()
  if (typ === "anthropic_compat") return "anthropic"
  if (typ === "openai_compat") {
    const byName = logoKeyFromName(String(p?.name || ""))
    return byName || "openai"
  }
  return logoKeyFromName(String(p?.name || ""))
}

function logoKeyForAddItem(it: any) {
  const t = String(it?.type || "").toLowerCase()
  if (t === "openai_compat") return "openai"
  if (t === "anthropic_compat") return "anthropic"
  return logoKeyFromName(String(it?.label || it?.name || ""))
}

const form = ref({ name: "", base_url: "", api_key: "", enabled: true })

watch(current, (p) => {
  if (!p) {
    form.value = { name: "", base_url: "", api_key: "", enabled: true }
    showKey.value = false
    return
  }
  form.value = {
    name: String(p.name || ""),
    base_url: String(p.base_url || ""),
    api_key: String(p.api_key || ""),
    enabled: !!p.enabled,
  }
})

function uniqueName(seed: string) {
  const base = String(seed || "").trim() || "Custom Provider"
  const existing = new Set((providers.value || []).map((p: any) => String(p?.name || "").toLowerCase()))
  if (!existing.has(base.toLowerCase())) return base
  for (let i = 2; i < 1000; i++) {
    const next = `${base} ${i}`
    if (!existing.has(next.toLowerCase())) return next
  }
  return `${base} ${Date.now()}`
}

async function refreshProviders() {
  try {
    await llm.fetchProviders()
  } catch {}
}

async function loadAddables() {
  if (loadingAddables.value) return
  try {
    loadingAddables.value = true
    const data = (await listAddableProviders()) as Addables
    addables.value = data || { special: [], presets: [] }
  } catch {
    addables.value = { special: [], presets: [] }
  } finally {
    loadingAddables.value = false
  }
}

async function addSpecial(type: string, label: string) {
  try {
    const name = uniqueName(label)
    await llm.addProvider({ type, policy: "custom", name, base_url: "", api_key: "", enabled: true })
    await refreshProviders()
    const added = (providers.value || []).find((p: any) => p?.name === name)
    if (added?.id) emit("open-provider", String(added.id))
  } catch {
    $message?.error?.(t("providers.create_failed"))
  }
}

async function enablePreset(preset: any) {
  try {
    if (!preset?.id) return
    await llm.saveProvider(String(preset.id), { enabled: true })
    await refreshProviders()
    emit("open-provider", String(preset.id))
  } catch {
    $message?.error?.(t("providers.enable_preset_failed"))
  }
}

async function saveCurrent() {
  const p = current.value
  if (!p || saving.value) return
  try {
    saving.value = true
    const payload: any = { enabled: !!form.value.enabled }
    const base = (form.value.base_url || "").trim()
    const key = (form.value.api_key || "").trim()
    if (base) payload.base_url = base
    if (key) payload.api_key = key
    if (canRenameCurrent.value) {
      const nm = (form.value.name || "").trim()
      if (nm && nm !== String(p.name || "")) payload.name = nm
    }
    await llm.saveProvider(String(p.id), payload)
    try {
      if (payload.base_url || payload.api_key) {
        const r = await llm.refresh(String(p.id))
        if (r?.ok) await llm.fetchProviders()
      }
    } catch {}
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  const p = current.value
  if (!p) return
  try {
    const r = await llm.testConn(String(p.id))
    if (r?.ok) $message?.success?.(t("common.success"))
    else $message?.error?.(r?.error || t("common.failed"))
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.failed"))
  }
}

async function refreshModels() {
  const p = current.value
  if (!p) return
  try {
    const r = await llm.refresh(String(p.id))
    if (r?.ok) {
      await llm.fetchProviders()
      $message?.success?.(t("providers.models_refreshed"))
    } else {
      $message?.error?.(r?.error || t("common.failed"))
    }
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.failed"))
  }
}

function confirmDanger(content: string, onOk: () => void) {
  const dialog = (window as any)?.$dialog
  if (dialog?.warning) {
    dialog.warning({
      title: t("common.confirm"),
      content,
      positiveText: t("common.confirm"),
      negativeText: t("common.cancel"),
      onPositiveClick: onOk,
    })
    return
  }
  if (window.confirm(content)) onOk()
}

async function deleteCurrent() {
  const p = current.value
  if (!p || !canDeleteCurrent.value) return
  if (deleting.value) return
  try {
    deleting.value = true
    await llm.removeProvider(String(p.id))
    deleteDialogOpen.value = false
    emit("close-provider")
  } catch {} finally {
    deleting.value = false
  }
}

async function doReset() {
  confirmDanger(t("providers.init_confirm"), async () => {
    try {
      await resetLLMData()
      $message?.success?.(t("providers.init_done"))
      await refreshProviders()
    } catch {
      $message?.error?.(t("providers.init_failed"))
    }
  })
}

const modelsLimit = 24
const modelExpand = ref(false)
const allModels = computed(() => (Array.isArray(current.value?.models) ? current.value.models.filter(Boolean) : []))
const visibleModels = computed(() => (modelExpand.value ? allModels.value : allModels.value.slice(0, modelsLimit)))
const hasMoreModels = computed(() => allModels.value.length > modelsLimit)

watch(currentId, () => {
  modelExpand.value = false
  showKey.value = false
})

onMounted(async () => {
  await refreshProviders()
})
</script>

<template>
  <div class="space-y-4">
    <!-- List page (default) -->
    <div v-if="!currentId" class="space-y-4">
      <div class="flex items-center justify-between gap-3 px-1">
        <div class="text-sm font-medium">{{ t("providers.available_title") }}</div>
        <Badge variant="secondary" class="font-normal">{{ availableProviders.length }}</Badge>
      </div>
      <Card class="border bg-background/50">
        <CardContent class="p-2">
          <div v-if="!availableProviders.length" class="text-sm text-muted-foreground px-2 py-2">
            {{ t("providers.available_empty") }}
          </div>
          <div v-else class="grid gap-2">
            <button
              v-for="p in availableProviders"
              :key="p.id"
              type="button"
              class="group w-full rounded-md border border-input bg-background/40 px-2 py-1.5 text-left text-xs hover:bg-accent hover:text-accent-foreground"
              @click="emit('open-provider', String(p.id))"
            >
              <div class="flex items-center gap-2">
                <ProviderLogo :logo-key="logoKeyForProvider(p)" :alt="p.name" :size="14" class="rounded-sm" />
                <span class="truncate">{{ p.name }}</span>
                <Badge v-if="!p.enabled" variant="outline" class="ml-2 shrink-0">Off</Badge>
                <ChevronRight class="ml-auto h-4 w-4 text-muted-foreground group-hover:text-accent-foreground" />
              </div>
            </button>
          </div>
        </CardContent>
      </Card>

      <div class="flex items-center justify-between gap-3 px-1">
        <div class="text-sm font-medium">{{ t("providers.unconfigured_title") }}</div>
        <Badge variant="secondary" class="font-normal">{{ unconfiguredProviders.length }}</Badge>
      </div>
      <Card class="border bg-background/50">
        <CardContent class="p-2">
          <div v-if="!unconfiguredProviders.length" class="text-sm text-muted-foreground px-2 py-2">
            {{ t("providers.unconfigured_empty") }}
          </div>
          <div v-else class="grid gap-2">
            <button
              v-for="p in unconfiguredProviders"
              :key="p.id"
              type="button"
              class="group w-full rounded-md border border-input bg-background/40 px-2 py-1.5 text-left text-xs hover:bg-accent hover:text-accent-foreground"
              @click="emit('open-provider', String(p.id))"
            >
              <div class="flex items-center gap-2">
                <ProviderLogo :logo-key="logoKeyForProvider(p)" :alt="p.name" :size="14" class="rounded-sm" />
                <span class="truncate">{{ p.name }}</span>
                <Badge v-if="!p.enabled" variant="outline" class="ml-2 shrink-0">Off</Badge>
                <ChevronRight class="ml-auto h-4 w-4 text-muted-foreground group-hover:text-accent-foreground" />
              </div>
            </button>
          </div>
        </CardContent>
      </Card>

      <!-- Actions (inline, after list) -->
      <div class="flex justify-end gap-2 pt-2">
        <Button class="dc-settings-button" variant="outline" size="icon-sm" :aria-label="t('common.refresh')" @click="refreshProviders">
          <RefreshCw class="h-4 w-4" />
        </Button>

        <DropdownMenu @update:open="(v) => v && loadAddables()">
          <DropdownMenuTrigger as-child>
            <Button class="dc-settings-button w-auto px-2 gap-1" variant="default" size="icon-sm" :aria-label="t('providers.add')">
              <Plus class="h-4 w-4" />
              <ChevronDown class="h-4 w-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="center" class="text-xs">
            <DropdownMenuItem
              v-for="it in (addables.special || [])"
              :key="'sp:' + it.type"
              class="text-xs"
              @select.prevent="addSpecial(it.type, it.label)"
            >
              <ProviderLogo :logo-key="logoKeyForAddItem(it)" :alt="it.label" :size="16" class="rounded-sm" />
              <span>{{ it.label }}</span>
            </DropdownMenuItem>
            <template v-if="(addables.presets || []).length">
              <DropdownMenuSeparator />
              <DropdownMenuItem v-for="p in addables.presets" :key="'pr:' + p.id" class="text-xs" @select.prevent="enablePreset(p)">
                <ProviderLogo :logo-key="logoKeyForProvider(p)" :alt="p.name" :size="16" class="rounded-sm" />
                <span>{{ p.name }}</span>
              </DropdownMenuItem>
            </template>
          </DropdownMenuContent>
        </DropdownMenu>

        <Button
          class="dc-settings-button"
          variant="destructive"
          size="icon-sm"
          :aria-label="t('providers.init_danger')"
          @click="doReset"
        >
          <Undo2 class="h-4 w-4" />
        </Button>
      </div>
    </div>

    <!-- Detail page -->
    <Card v-else class="border bg-background/50">
      <CardContent class="p-4">
        <div v-if="!current" class="text-sm text-muted-foreground">
          {{ t("providers.empty_hint") }}
        </div>

        <div v-else class="space-y-6">
          <div class="grid gap-4">
            <div v-if="canRenameCurrent" class="grid gap-2">
              <Label>{{ t("providers.provider_name") }}</Label>
              <Input v-model="form.name" class="dc-settings-field" @blur="saveCurrent" />
            </div>

            <div class="grid gap-2">
              <Label>API Key</Label>
              <div class="relative">
                <Input v-model="form.api_key" class="dc-settings-field pr-12" :type="showKey ? 'text' : 'password'" @blur="saveCurrent" />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  class="absolute right-1 top-1 h-7 w-7"
                  :aria-label="showKey ? t('common.hide') : t('common.show')"
                  @click="showKey = !showKey"
                >
                  <EyeOff v-if="showKey" class="h-4 w-4" />
                  <Eye v-else class="h-4 w-4" />
                </Button>
              </div>
            </div>

            <div class="grid gap-2">
              <Label>{{ t("providers.api_base_url") }}</Label>
              <Input v-model="form.base_url" class="dc-settings-field" :placeholder="t('providers.base_url_hint')" @blur="saveCurrent" />
              <div class="text-xs text-muted-foreground">{{ t("providers.base_url_hint") }}</div>
            </div>

            <div class="flex items-center justify-between">
              <Label>Enabled</Label>
              <Switch v-model:checked="form.enabled" @update:checked="saveCurrent" />
            </div>

            <div class="flex items-center gap-2">
              <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="saving" @click="testConnection">
                {{ t("providers.test_connection") }}
              </Button>
              <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="saving" @click="refreshModels">
                {{ t("providers.models_refresh") }}
              </Button>
            </div>
          </div>

          <div class="space-y-2">
            <div class="flex items-center justify-between">
              <div class="text-sm font-medium">{{ t("providers.models_title") }}</div>
              <Button
                v-if="hasMoreModels"
                class="dc-settings-button"
                variant="secondary"
                size="sm"
                @click="modelExpand = !modelExpand"
              >
                {{ modelExpand ? t("providers.models_show_less") : t("providers.models_show_more") }}
              </Button>
            </div>

            <div v-if="!allModels.length" class="text-sm text-muted-foreground">
              {{ t("providers.models_empty") }}
            </div>
            <div v-else class="flex flex-wrap gap-2">
              <Badge v-for="m in visibleModels" :key="m" variant="secondary">{{ m }}</Badge>
            </div>
          </div>

          <div v-if="canDeleteCurrent" class="pt-2">
            <Separator class="mb-3" />
            <div class="flex justify-end">
              <Button class="dc-settings-button" variant="destructive" size="sm" @click="deleteDialogOpen = true">
                <Trash2 class="h-4 w-4" />
                {{ t("common.delete") }}
              </Button>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  </div>

  <Dialog v-model:open="deleteDialogOpen">
    <DialogContent class="dc-settings-dialog sm:max-w-md">
      <DialogHeader>
        <DialogTitle class="dc-dialog-title">{{ t("common.delete_confirm") }}</DialogTitle>
        <DialogDescription class="dc-dialog-description">
          {{ t("common.delete_confirm_detail", { title: current?.name || '' }) }}
        </DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-2">
        <Button class="dc-settings-button" variant="secondary" size="sm" @click="deleteDialogOpen = false">
          {{ t("common.cancel") }}
        </Button>
        <Button class="dc-settings-button" variant="destructive" size="sm" :disabled="deleting" @click="deleteCurrent">
          {{ t("common.delete") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
