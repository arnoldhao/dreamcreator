<script setup lang="ts">
import { computed, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

import {
  CreateManualCollection,
  DeleteCookieCollection,
  ExportCookieCollection,
  ListAllCookies,
  SyncCookies,
  UpdateManualCollection,
} from "bindings/dreamcreator/backend/api/cookiesapi"

import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Badge } from "@/components/ui/badge"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"

import { Copy, Pencil, Plus, RefreshCw, Trash2 } from "lucide-vue-next"
import { copyText as copyToClipboard } from "@/utils/clipboard.js"
import { buildCookiePreview, parseCookieInput } from "@/utils/cookies"

type CookieCollection = any
type CookieCollections = {
  browser_collections?: CookieCollection[]
  manual_collections?: CookieCollection[]
}

const { t } = useI18n()

const isLoading = ref(false)
const isSyncing = ref<Record<string, boolean>>({})
const collections = ref<CookieCollections>({ browser_collections: [], manual_collections: [] })

const browserCollections = computed(() => (collections.value?.browser_collections || []).filter(Boolean))
const manualCollections = computed(() => (collections.value?.manual_collections || []).filter(Boolean))

function toJS(resp: any) {
  if (!resp || typeof resp !== "object" || !("success" in resp)) return resp
  if (!resp.success) throw new Error(resp.msg || "request failed")
  let data = resp.data
  if (typeof data === "string") {
    try { data = JSON.parse(data) } catch {}
  }
  return data
}

function getCookieCount(col: any) {
  const domains = col?.domain_cookies
  if (!domains) return 0
  try {
    return Object.values(domains).reduce((sum: number, d: any) => sum + (Array.isArray(d?.cookies) ? d.cookies.length : 0), 0)
  } catch {
    return 0
  }
}

function getDomainCount(col: any) {
  const domains = col?.domain_cookies
  if (!domains) return 0
  try { return Object.keys(domains).length } catch { return 0 }
}

function statusLabel(col: any) {
  const st = String(col?.status || "").toLowerCase()
  const map: Record<string, string> = {
    synced: t("cookies.status.synced"),
    never: t("cookies.status.never"),
    syncing: t("cookies.status.syncing"),
    error: t("cookies.status.error"),
    manual: t("cookies.status.manual"),
  }
  return map[st] || t("cookies.status.unknown")
}

function statusVariant(col: any) {
  const st = String(col?.status || "").toLowerCase()
  if (st === "synced") return "secondary"
  if (st === "error") return "destructive"
  return "outline"
}

async function reload() {
  if (isLoading.value) return
  try {
    isLoading.value = true
    const data = toJS(await ListAllCookies()) as CookieCollections
    collections.value = data || { browser_collections: [], manual_collections: [] }
  } catch (e: any) {
    $message?.error?.(t("cookies.fetch_error", { msg: e?.message || String(e) }))
  } finally {
    isLoading.value = false
  }
}

async function syncBrowser(browser: string) {
  const key = String(browser || "")
  if (!key) return
  if (isSyncing.value[key]) return
  try {
    isSyncing.value = { ...isSyncing.value, [key]: true }
    toJS(await SyncCookies("yt-dlp", [key]))
    $message?.success?.(t("cookies.sync_success"))
  } catch (e: any) {
    $message?.error?.(t("cookies.sync_error", { msg: e?.message || String(e) }))
  } finally {
    isSyncing.value = { ...isSyncing.value, [key]: false }
    reload()
  }
}

async function exportCollection(id: string) {
  try {
    const raw = toJS(await ExportCookieCollection(id))
    const text = typeof raw === "string" ? raw : JSON.stringify(raw, null, 2)
    await copyToClipboard(text, t)
    $message?.success?.(t("cookies.manual_export_copied"))
  } catch (e: any) {
    $message?.error?.(t("cookies.manual_export_failed"))
  }
}

function confirmDelete(title: string, onOk: () => void) {
  const content = t("common.delete_confirm_detail", { title })
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

async function deleteManual(col: any) {
  const id = String(col?.id || "")
  if (!id) return
  confirmDelete(String(col?.name || t("cookies.title")), async () => {
    try {
      toJS(await DeleteCookieCollection(id))
      $message?.success?.(t("common.delete_success"))
      reload()
    } catch (e: any) {
      $message?.error?.(t("common.delete_failed"))
    }
  })
}

// Manual editor dialog
const dialogOpen = ref(false)
const dialogMode = ref<"create" | "edit">("create")
const editing = ref<any>(null)
const formName = ref("")
const formRaw = ref("")
const defaultDomain = ref("")

const parsed = computed(() => {
  try {
    const raw = formRaw.value.trim()
    if (!raw) return null
    return parseCookieInput(raw, defaultDomain.value.trim())
  } catch {
    return null
  }
})

const preview = computed(() => {
  try {
    return buildCookiePreview(parsed.value?.netscape || "", 8)
  } catch {
    return { total: 0, entries: [] as any[] }
  }
})

watch(parsed, (v) => {
  if (!dialogOpen.value) return
  if (!v?.defaultDomain) return
  if (!defaultDomain.value.trim()) defaultDomain.value = v.defaultDomain
})

async function openCreate() {
  dialogMode.value = "create"
  editing.value = null
  formName.value = ""
  formRaw.value = ""
  defaultDomain.value = ""
  dialogOpen.value = true
}

async function openEdit(col: any) {
  dialogMode.value = "edit"
  editing.value = col
  formName.value = String(col?.name || "")
  defaultDomain.value = ""
  dialogOpen.value = true
  try {
    const raw = toJS(await ExportCookieCollection(String(col?.id || "")))
    formRaw.value = typeof raw === "string" ? raw : ""
  } catch {
    formRaw.value = ""
  }
}

function parseErrorToMessage(e: any) {
  const msg = String(e?.message || "")
  if (msg.includes("default domain required")) return t("cookies.manual_header_need_domain")
  if (msg === "empty") return t("cookies.manual_empty_warning")
  if (msg === "parse_failed") return t("cookies.manual_parse_failed")
  return t("cookies.manual_parse_failed")
}

async function saveManual() {
  const name = (formName.value || "").trim() || t("cookies.manual_default_name")
  const raw = (formRaw.value || "").trim()
  if (!raw) {
    $message?.warning?.(t("cookies.manual_empty_warning"))
    return
  }
  let parsedValue
  try {
    parsedValue = parseCookieInput(raw, defaultDomain.value.trim())
  } catch (e: any) {
    $message?.warning?.(parseErrorToMessage(e))
    return
  }

  const payload = { name, netscape: parsedValue.netscape, cookies: [], replace: true }
  try {
    if (dialogMode.value === "edit" && editing.value?.id) {
      toJS(await UpdateManualCollection(String(editing.value.id), payload))
    } else {
      toJS(await CreateManualCollection(payload))
    }
    $message?.success?.(t("common.saved"))
    dialogOpen.value = false
    reload()
  } catch (e: any) {
    $message?.error?.(t("common.save_failed"))
  }
}

reload()
</script>

<template>
  <Card class="border-0 bg-transparent shadow-none">
    <CardHeader class="p-0 pb-4 flex flex-row items-center justify-end gap-3">
      <div class="flex items-center gap-2">
        <Button class="dc-settings-button" variant="secondary" size="icon-sm" :disabled="isLoading" :aria-label="t('common.refresh')" @click="reload">
          <RefreshCw class="h-4 w-4" />
        </Button>
        <Button class="dc-settings-button" variant="secondary" size="sm" @click="openCreate">
          <Plus class="h-4 w-4" />
          {{ t("common.add") }}
        </Button>
      </div>
    </CardHeader>

    <CardContent class="p-0 space-y-6">
      <div class="space-y-2">
        <div class="text-sm text-muted-foreground">{{ t("cookies.panel_hint") }}</div>

        <div class="grid gap-2">
          <div class="text-xs text-muted-foreground">{{ t("cookies.title") }}</div>
          <div v-if="!browserCollections.length" class="rounded-lg border bg-background/50 p-3 text-sm text-muted-foreground">
            {{ t("cookies.no_cookies_found") }} · {{ t("cookies.try_sync") }}
          </div>
          <div v-else class="grid gap-2">
            <div v-for="col in browserCollections" :key="col.id || col.browser" class="rounded-lg border bg-background/50 p-3">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="font-medium truncate">{{ col.browser || col.name }}</div>
                    <Badge :variant="statusVariant(col)">{{ statusLabel(col) }}</Badge>
                  </div>
                  <div class="mt-1 text-xs text-muted-foreground">
                    {{ t("cookies.total_cookies", { count: getCookieCount(col) }) }} · {{ getDomainCount(col) }} {{ t("cookies.domains") }}
                  </div>
                </div>
                <Button
                  class="dc-settings-button shrink-0"
                  variant="secondary"
                  size="sm"
                  :disabled="isSyncing[col.browser]"
                  @click="syncBrowser(col.browser)"
                >
                  <RefreshCw class="h-4 w-4" />
                  {{ t("cookies.sync_with", { type: "yt-dlp" }) }}
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div class="grid gap-2">
        <div class="flex items-center justify-between">
          <div class="text-xs text-muted-foreground">{{ t("cookies.manual_collection_sets") }}</div>
        </div>
        <div v-if="!manualCollections.length" class="rounded-lg border bg-background/50 p-3 text-sm text-muted-foreground">
          {{ t("cookies.manual_empty_title") }}
        </div>
        <div v-else class="grid gap-2">
          <div v-for="col in manualCollections" :key="col.id" class="rounded-lg border bg-background/50 p-3">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <div class="font-medium truncate">{{ col.name }}</div>
                  <Badge variant="outline">{{ t("cookies.status.manual") }}</Badge>
                </div>
                <div class="mt-1 text-xs text-muted-foreground">
                  {{ t("cookies.total_cookies", { count: getCookieCount(col) }) }} · {{ getDomainCount(col) }} {{ t("cookies.domains") }}
                </div>
              </div>

              <div class="flex items-center gap-2 shrink-0">
                <Button class="dc-settings-button" variant="secondary" size="icon-sm" :aria-label="t('common.edit')" @click="openEdit(col)">
                  <Pencil class="h-4 w-4" />
                </Button>
                <Button class="dc-settings-button" variant="secondary" size="icon-sm" :aria-label="t('common.copy')" @click="exportCollection(col.id)">
                  <Copy class="h-4 w-4" />
                </Button>
                <Button class="dc-settings-button" variant="secondary" size="icon-sm" :aria-label="t('common.delete')" @click="deleteManual(col)">
                  <Trash2 class="h-4 w-4" />
                </Button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>

  <Dialog v-model:open="dialogOpen">
    <DialogContent class="dc-settings-dialog max-w-[720px]">
      <DialogHeader>
        <DialogTitle class="dc-dialog-title">
          {{ dialogMode === "edit" ? t("cookies.manual_edit_title") : t("cookies.manual_create_title") }}
        </DialogTitle>
        <DialogDescription class="dc-dialog-description">
          {{ t("cookies.manual_manage_hint") }}
        </DialogDescription>
      </DialogHeader>

      <div class="grid gap-4">
        <div class="grid gap-2">
          <Label>{{ t("common.name") }}</Label>
          <Input v-model="formName" class="dc-settings-field" :placeholder="t('cookies.manual_default_name')" />
        </div>

        <div class="grid gap-2">
          <Label>{{ t("cookies.manual_paste_label") }}</Label>
          <textarea
            v-model="formRaw"
            rows="10"
            class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
            :placeholder="t('cookies.manual_paste_placeholder')"
          />
          <div class="text-xs text-muted-foreground">
            {{ t("cookies.manual_input_format_hint") }} {{ t("cookies.manual_netscape_hint") }} {{ t("cookies.manual_json_hint") }} {{ t("cookies.manual_header_hint") }}
          </div>
        </div>

        <div class="grid gap-2">
          <Label>{{ t("cookies.manual_default_domain_label") }}</Label>
          <Input v-model="defaultDomain" class="dc-settings-field" :placeholder="t('cookies.manual_default_domain_placeholder')" />
          <div class="text-xs text-muted-foreground">{{ t("cookies.manual_default_domain_hint") }}</div>
        </div>

        <div class="grid gap-2">
          <Label>{{ t("cookies.manual_preview_label") }}</Label>
          <div v-if="preview.total" class="rounded-lg border bg-background/50 p-3">
            <div class="flex items-center justify-between">
              <div class="text-xs text-muted-foreground">{{ t("cookies.manual_preview_title", { count: preview.total }) }}</div>
            </div>
            <div class="mt-2 grid gap-2">
              <div v-for="(c, i) in preview.entries" :key="i" class="flex items-center justify-between gap-3">
                <div class="min-w-0">
                  <div class="truncate text-sm">{{ c.name }}</div>
                  <div class="truncate text-xs text-muted-foreground">{{ c.domain }}</div>
                </div>
                <Badge variant="secondary" class="shrink-0">{{ c.path }}</Badge>
              </div>
            </div>
            <div v-if="preview.total > preview.entries.length" class="mt-2 text-xs text-muted-foreground">
              {{ t("cookies.manual_preview_more", { total: preview.total }) }}
            </div>
          </div>
          <div v-else class="rounded-lg border bg-background/50 p-3 text-sm text-muted-foreground">
            {{ t("cookies.manual_preview_empty") }}
          </div>
        </div>
      </div>

      <DialogFooter class="gap-2">
        <Button class="dc-settings-button" variant="secondary" @click="dialogOpen = false">{{ t("common.cancel") }}</Button>
        <Button class="dc-settings-button" variant="default" @click="saveManual">{{ t("common.save") }}</Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
