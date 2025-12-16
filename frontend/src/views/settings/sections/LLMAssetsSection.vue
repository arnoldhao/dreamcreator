<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue"
import { useI18n } from "vue-i18n"

import type { LLMAssetsKind } from "../types"

import { subtitleService } from "@/services/subtitleService.js"
import { useTargetLanguagesStore } from "@/stores/targetLanguages.js"
import { createGlobalProfile, deleteGlobalProfile, listGlobalProfiles, updateGlobalProfile } from "@/services/llmProviderService.js"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Separator } from "@/components/ui/separator"
import { Switch } from "@/components/ui/switch"
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { ChevronDown, ChevronRight, Plus, RefreshCw, Trash2, Undo2 } from "lucide-vue-next"

type GlossarySet = { id: string; name: string; description?: string }
type GlossaryEntry = {
  id: string
  set_id?: string
  source: string
  case_sensitive?: boolean
  do_not_translate?: boolean
  translations?: Record<string, string>
}
type TargetLanguage = { code: string; name?: string }
type GlobalProfile = {
  id: string
  name?: string
  temperature?: number
  top_p?: number
  json_mode?: boolean
  max_tokens?: number
  sys_prompt_tpl?: string
}

const props = defineProps<{
  activeKind?: LLMAssetsKind
  activeId?: string
}>()

const emit = defineEmits<{
  (e: "open-item", kind: LLMAssetsKind, id: string): void
  (e: "close-item"): void
}>()

const { t } = useI18n()
const tlStore = useTargetLanguagesStore()

const isDetail = computed(() => !!(props.activeKind && props.activeId))
const activeKind = computed(() => (props.activeKind || "") as LLMAssetsKind | "")
const activeId = computed(() => String(props.activeId || ""))

type ConfirmState = {
  title: string
  description: string
  confirmText: string
  confirmVariant: "default" | "destructive" | "secondary"
  onConfirm: null | (() => Promise<void> | void)
}
const confirmOpen = ref(false)
const confirming = ref(false)
const confirmState = reactive<ConfirmState>({
  title: "",
  description: "",
  confirmText: "",
  confirmVariant: "default",
  onConfirm: null,
})

function openConfirm(next: Partial<ConfirmState> & { onConfirm: () => Promise<void> | void }) {
  confirmState.title = next.title || t("common.confirm")
  confirmState.description = next.description || ""
  confirmState.confirmText = next.confirmText || t("common.confirm")
  confirmState.confirmVariant = next.confirmVariant || "default"
  confirmState.onConfirm = next.onConfirm
  confirmOpen.value = true
}

async function runConfirm() {
  if (!confirmState.onConfirm || confirming.value) return
  try {
    confirming.value = true
    await confirmState.onConfirm()
    confirmOpen.value = false
  } finally {
    confirming.value = false
  }
}

// -------- Glossary list --------
const glLoading = ref(false)
const glSets = ref<GlossarySet[]>([])
const glCounts = ref<Record<string, number>>({})

async function loadGlossarySets() {
  if (glLoading.value) return
  glLoading.value = true
  try {
    const sets = (await subtitleService.listGlossarySets()) as GlossarySet[]
    const list = (Array.isArray(sets) ? sets : []).filter(Boolean)
    list.sort((a, b) => String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" }))
    glSets.value = list
    const counts: Record<string, number> = {}
    await Promise.all(
      list.map(async (s) => {
        try {
          const entries = await subtitleService.listGlossaryBySet(s.id)
          counts[s.id] = Array.isArray(entries) ? entries.length : 0
        } catch {
          counts[s.id] = 0
        }
      }),
    )
    glCounts.value = counts
  } catch {
    glSets.value = []
    glCounts.value = {}
  } finally {
    glLoading.value = false
  }
}

// -------- Target languages --------
const langs = computed(() => ((tlStore.list || []) as TargetLanguage[]).filter(Boolean))
async function loadTargetLanguages() {
  try {
    await tlStore.ensureLoaded()
  } catch {}
}

// -------- Profiles --------
const pfLoading = ref(false)
const profiles = ref<GlobalProfile[]>([])

async function loadProfiles() {
  if (pfLoading.value) return
  pfLoading.value = true
  try {
    const list = (await listGlobalProfiles()) as GlobalProfile[]
    const arr = (Array.isArray(list) ? list : []).filter(Boolean)
    arr.sort((a, b) => String(a?.name || "").localeCompare(String(b?.name || ""), undefined, { sensitivity: "base" }))
    profiles.value = arr
  } catch {
    profiles.value = []
  } finally {
    pfLoading.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadGlossarySets(), loadTargetLanguages(), loadProfiles()])
}

// -------- Glossary detail --------
const glDetailLoading = ref(false)
const glEntries = ref<GlossaryEntry[]>([])
const glSetForm = reactive<{ id: string; name: string; description: string }>({ id: "", name: "", description: "" })
const glDraft = reactive<{ id: string; source: string; case_sensitive: boolean; mode: "dnt" | "specify"; lang: string; trans: string }>({
  id: "",
  source: "",
  case_sensitive: false,
  mode: "dnt",
  lang: "all",
  trans: "",
})

const glLangOptions = computed(() => {
  const codes = Array.isArray(tlStore.codes) ? tlStore.codes : []
  return ["all", ...codes]
})
const glLangName = (code: string) => {
  if (code === "all") return t("glossary.lang_all")
  return tlStore.getName(code) || code
}

function pickSingleTranslation(tr?: Record<string, string> | null) {
  if (!tr) return { lang: "all", val: "" }
  if (tr["all"]) return { lang: "all", val: tr["all"] || "" }
  const keys = Object.keys(tr || {})
  if (!keys.length) return { lang: "all", val: "" }
  keys.sort()
  const lang = keys[0]
  return { lang, val: tr[lang] || "" }
}

function resetGlossaryDraft() {
  glDraft.id = ""
  glDraft.source = ""
  glDraft.case_sensitive = false
  glDraft.mode = "dnt"
  glDraft.lang = "all"
  glDraft.trans = ""
}

function openGlossaryEntry(e: GlossaryEntry) {
  glDraft.id = e.id || ""
  glDraft.source = String(e.source || "")
  glDraft.case_sensitive = !!e.case_sensitive
  if (e.do_not_translate) {
    glDraft.mode = "dnt"
    glDraft.lang = "all"
    glDraft.trans = ""
  } else {
    glDraft.mode = "specify"
    const { lang, val } = pickSingleTranslation(e.translations || {})
    glDraft.lang = lang || "all"
    glDraft.trans = val || ""
  }
}

function glossaryTranslationSummary(e: GlossaryEntry) {
  if (e.do_not_translate) return t("glossary.dnt")
  const { val } = pickSingleTranslation(e.translations || {})
  return val ? val : t("glossary.no_translations")
}

async function loadGlossaryDetail(setId: string) {
  if (!setId || setId === "new") {
    glEntries.value = []
    resetGlossaryDraft()
    return
  }
  if (glDetailLoading.value) return
  glDetailLoading.value = true
  try {
    const list = await subtitleService.listGlossaryBySet(setId)
    const arr = (Array.isArray(list) ? list : []).filter(Boolean)
    glEntries.value = arr
  } catch {
    glEntries.value = []
  } finally {
    glDetailLoading.value = false
  }
}

async function saveGlossarySet() {
  const name = (glSetForm.name || "").trim()
  const description = (glSetForm.description || "").trim()
  if (!name) return
  try {
    const payload: any = { name, description }
    if (glSetForm.id) payload.id = glSetForm.id
    const saved = (await subtitleService.upsertGlossarySet(payload)) as GlossarySet
    await loadGlossarySets()
    if (saved?.id) emit("open-item", "glossary", String(saved.id))
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.save_failed"))
  }
}

async function deleteGlossarySet(setId: string, title: string) {
  if (!setId) return
  openConfirm({
    title: t("common.delete_confirm"),
    description: t("common.delete_confirm_detail", { title: title || setId }),
    confirmText: t("common.delete"),
    confirmVariant: "destructive",
    onConfirm: async () => {
    try {
      await subtitleService.deleteGlossarySet(setId)
      await loadGlossarySets()
      emit("close-item")
      $message?.success?.(t("common.deleted"))
    } catch {
      $message?.error?.(t("common.delete_failed"))
    }
    },
  })
}

async function saveGlossaryEntry() {
  const setId = glSetForm.id
  if (!setId) return
  const source = (glDraft.source || "").trim()
  if (!source) return
  try {
    const payload: any = {
      set_id: setId,
      source,
      case_sensitive: !!glDraft.case_sensitive,
      do_not_translate: glDraft.mode === "dnt",
      translations: {},
    }
    if (glDraft.id) payload.id = glDraft.id
    if (glDraft.mode === "specify") {
      const lang = (glDraft.lang || "all").trim() || "all"
      const val = (glDraft.trans || "").trim()
      if (val) payload.translations = { [lang]: val }
      payload.do_not_translate = false
    }
    const saved = (await subtitleService.upsertGlossaryEntry(payload)) as GlossaryEntry
    if (saved?.id) {
      const idx = glEntries.value.findIndex((x) => x?.id === saved.id)
      if (idx >= 0) glEntries.value[idx] = saved
      else glEntries.value.unshift(saved)
      glCounts.value = { ...(glCounts.value || {}), [setId]: glEntries.value.length }
    }
    resetGlossaryDraft()
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.save_failed"))
  }
}

async function deleteGlossaryEntry(e: GlossaryEntry) {
  if (!e?.id) return
  openConfirm({
    title: t("common.delete_confirm"),
    description: t("common.delete_confirm_detail", { title: (e.source || "").slice(0, 120) }),
    confirmText: t("common.delete"),
    confirmVariant: "destructive",
    onConfirm: async () => {
    try {
      await subtitleService.deleteGlossaryEntry(e.id)
      glEntries.value = glEntries.value.filter((x) => x?.id !== e.id)
      if (glSetForm.id) glCounts.value = { ...(glCounts.value || {}), [glSetForm.id]: glEntries.value.length }
      if (glDraft.id === e.id) resetGlossaryDraft()
      $message?.success?.(t("common.deleted"))
    } catch {
      $message?.error?.(t("common.delete_failed"))
    }
    },
  })
}

// -------- Target language detail --------
const tlForm = reactive<{ originalCode: string; code: string; name: string }>({ originalCode: "", code: "", name: "" })

function resetTLForm() {
  tlForm.originalCode = ""
  tlForm.code = ""
  tlForm.name = ""
}

async function saveTargetLanguage() {
  const code = (tlForm.code || "").trim()
  const name = (tlForm.name || "").trim()
  if (!code) return
  try {
    if (tlForm.originalCode && tlForm.originalCode !== code) {
      await tlStore.remove(tlForm.originalCode)
    }
    await tlStore.upsert({ code, name })
    await loadTargetLanguages()
    emit("open-item", "target_languages", code)
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.save_failed"))
  }
}

async function deleteTargetLanguage(code: string) {
  const c = (code || "").trim()
  if (!c) return
  openConfirm({
    title: t("common.delete_confirm"),
    description: t("subtitle.target_languages.delete_confirm", { code: c }),
    confirmText: t("common.delete"),
    confirmVariant: "destructive",
    onConfirm: async () => {
    try {
      await tlStore.remove(c)
      await loadTargetLanguages()
      emit("close-item")
      $message?.success?.(t("common.deleted"))
    } catch {
      $message?.error?.(t("common.delete_failed"))
    }
    },
  })
}

async function restoreDefaultTargetLanguages() {
  openConfirm({
    title: t("common.confirm"),
    description: t("subtitle.target_languages.restore_confirm"),
    confirmText: t("common.reset"),
    confirmVariant: "destructive",
    onConfirm: async () => {
    try {
      await tlStore.resetToDefaults()
      $message?.success?.(t("subtitle.target_languages.restored"))
    } catch {
      $message?.error?.(t("common.save_failed"))
    }
    },
  })
}

// -------- Profile detail --------
const pfForm = reactive<{ id: string; name: string; temperature: number; top_p: number; json_mode: boolean; max_tokens: number; sys_prompt_tpl: string }>({
  id: "",
  name: "",
  temperature: 0.2,
  top_p: 1,
  json_mode: true,
  max_tokens: 2048,
  sys_prompt_tpl: "",
})

function resetProfileForm() {
  pfForm.id = ""
  pfForm.name = ""
  pfForm.temperature = 0.2
  pfForm.top_p = 1
  pfForm.json_mode = true
  pfForm.max_tokens = 2048
  pfForm.sys_prompt_tpl = ""
}

async function saveProfile() {
  const payload = {
    name: (pfForm.name || "").trim(),
    temperature: Number(pfForm.temperature ?? 0.2),
    top_p: Number(pfForm.top_p ?? 1),
    json_mode: !!pfForm.json_mode,
    max_tokens: Number(pfForm.max_tokens ?? 2048),
    sys_prompt_tpl: String(pfForm.sys_prompt_tpl || ""),
  }
  try {
    if (!pfForm.id) {
      const created = (await createGlobalProfile(payload)) as GlobalProfile
      await loadProfiles()
      if (created?.id) emit("open-item", "profiles", String(created.id))
    } else {
      await updateGlobalProfile(pfForm.id, payload)
      await loadProfiles()
    }
  } catch (e: any) {
    $message?.error?.(e?.message || t("common.save_failed"))
  }
}

async function deleteProfile(id: string, title: string) {
  if (!id) return
  openConfirm({
    title: t("common.delete_confirm"),
    description: t("common.delete_confirm_detail", { title: title || id }),
    confirmText: t("common.delete"),
    confirmVariant: "destructive",
    onConfirm: async () => {
    try {
      await deleteGlobalProfile(id)
      await loadProfiles()
      emit("close-item")
      $message?.success?.(t("common.deleted"))
    } catch (e: any) {
      $message?.error?.(e?.message || t("common.delete_failed"))
    }
    },
  })
}

// -------- Apply active route to forms --------
watch(
  [activeKind, activeId, glSets, langs, profiles],
  () => {
    if (activeKind.value === "glossary") {
      if (activeId.value === "new") {
        glSetForm.id = ""
        glSetForm.name = ""
        glSetForm.description = ""
        loadGlossaryDetail("new")
        return
      }
      const found = (glSets.value || []).find((s) => String(s?.id || "") === activeId.value) || null
      glSetForm.id = found?.id || activeId.value
      glSetForm.name = found?.name || ""
      glSetForm.description = found?.description || ""
      loadGlossaryDetail(glSetForm.id)
      return
    }

    if (activeKind.value === "target_languages") {
      if (activeId.value === "new") {
        resetTLForm()
        return
      }
      const found = (langs.value || []).find((l) => String(l?.code || "") === activeId.value) || null
      tlForm.originalCode = found?.code || activeId.value
      tlForm.code = found?.code || activeId.value
      tlForm.name = found?.name || found?.code || activeId.value
      return
    }

    if (activeKind.value === "profiles") {
      if (activeId.value === "new") {
        resetProfileForm()
        return
      }
      const found = (profiles.value || []).find((p) => String(p?.id || "") === activeId.value) || null
      pfForm.id = found?.id || activeId.value
      pfForm.name = found?.name || ""
      pfForm.temperature = Number(found?.temperature ?? 0.2)
      pfForm.top_p = Number(found?.top_p ?? 1)
      pfForm.json_mode = !!found?.json_mode
      pfForm.max_tokens = Number(found?.max_tokens ?? 2048)
      pfForm.sys_prompt_tpl = String(found?.sys_prompt_tpl || "")
      return
    }

    resetGlossaryDraft()
    resetTLForm()
    resetProfileForm()
  },
  { immediate: true },
)

onMounted(async () => {
  await refreshAll()
})
</script>

<template>
  <div class="space-y-4">
    <!-- List page -->
    <div v-if="!isDetail" class="space-y-6">
      <!-- Glossary -->
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3 px-1">
          <div class="text-sm font-medium">{{ t("glossary.title") }}</div>
          <Badge variant="secondary" class="font-normal">{{ glSets.length }}</Badge>
        </div>
        <Card class="border bg-background/50">
          <CardContent class="p-2">
            <div v-if="!glSets.length" class="text-sm text-muted-foreground px-2 py-2">
              {{ t("glossary.sets") }}: 0
            </div>
            <div v-else class="grid gap-2">
              <div
                v-for="s in glSets"
                :key="s.id"
                class="group flex items-center gap-2 rounded-md border border-input bg-background/40 px-2 py-1.5 text-xs hover:bg-accent hover:text-accent-foreground"
              >
                <button type="button" class="flex-1 min-w-0 text-left" @click="emit('open-item', 'glossary', String(s.id))">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="truncate">{{ s.name }}</span>
                    <Badge variant="secondary" class="font-normal shrink-0">{{ glCounts[s.id] || 0 }}</Badge>
                    <ChevronRight class="ml-auto h-4 w-4 text-muted-foreground group-hover:text-accent-foreground" />
                  </div>
                </button>
                <Button
                  class="dc-settings-button h-[22px] w-[22px] p-0"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="t('common.delete')"
                  @click="deleteGlossarySet(String(s.id), s.name)"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
        <div class="flex justify-end gap-2 pt-2">
          <Button class="dc-settings-button h-[22px] w-9 p-0" variant="outline" size="icon-sm" :aria-label="t('common.refresh')" @click="loadGlossarySets">
            <RefreshCw class="h-3.5 w-3.5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button class="dc-settings-button h-[22px] w-auto px-1.5 gap-1" variant="default" size="icon-sm" :aria-label="t('common.add')">
                <Plus class="h-3.5 w-3.5" />
                <ChevronDown class="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="center" class="text-xs">
              <DropdownMenuItem class="text-xs" @select.prevent="emit('open-item', 'glossary', 'new')">
                <Plus class="h-3.5 w-3.5" />
                <span>{{ t("glossary.modal_create_title") }}</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>

      <!-- Target languages -->
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3 px-1">
          <div class="text-sm font-medium">{{ t("subtitle.target_languages.title") }}</div>
          <Badge variant="secondary" class="font-normal">{{ langs.length }}</Badge>
        </div>
        <Card class="border bg-background/50">
          <CardContent class="p-2">
            <div v-if="!langs.length" class="text-sm text-muted-foreground px-2 py-2">
              {{ t("subtitle.target_languages.empty") }}
            </div>
            <div v-else class="grid gap-2">
              <div
                v-for="l in langs"
                :key="l.code"
                class="group flex items-center gap-2 rounded-md border border-input bg-background/40 px-2 py-1.5 text-xs hover:bg-accent hover:text-accent-foreground"
              >
                <button type="button" class="flex-1 min-w-0 text-left" @click="emit('open-item', 'target_languages', String(l.code))">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="truncate">{{ l.name || l.code }}</span>
                    <Badge variant="outline" class="font-normal shrink-0">{{ l.code }}</Badge>
                    <ChevronRight class="ml-auto h-4 w-4 text-muted-foreground group-hover:text-accent-foreground" />
                  </div>
                </button>
                <Button
                  class="dc-settings-button h-[22px] w-[22px] p-0"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="t('common.delete')"
                  @click="deleteTargetLanguage(String(l.code))"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
        <div class="flex justify-end gap-2 pt-2">
          <Button class="dc-settings-button h-[22px] w-9 p-0" variant="outline" size="icon-sm" :aria-label="t('common.refresh')" @click="loadTargetLanguages">
            <RefreshCw class="h-3.5 w-3.5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button class="dc-settings-button h-[22px] w-auto px-1.5 gap-1" variant="default" size="icon-sm" :aria-label="t('common.add')">
                <Plus class="h-3.5 w-3.5" />
                <ChevronDown class="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="center" class="text-xs">
              <DropdownMenuItem class="text-xs" @select.prevent="emit('open-item', 'target_languages', 'new')">
                <Plus class="h-3.5 w-3.5" />
                <span>{{ t("subtitle.target_languages.create_title") }}</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
          <Button class="dc-settings-button h-[22px] w-9 p-0" variant="destructive" size="icon-sm" :aria-label="t('common.reset')" @click="restoreDefaultTargetLanguages">
            <Undo2 class="h-3.5 w-3.5" />
          </Button>
        </div>
      </div>

      <!-- Profiles -->
      <div class="space-y-2">
        <div class="flex items-center justify-between gap-3 px-1">
          <div class="text-sm font-medium">{{ t("profiles.inspector_title") }}</div>
          <Badge variant="secondary" class="font-normal">{{ profiles.length }}</Badge>
        </div>
        <Card class="border bg-background/50">
          <CardContent class="p-2">
            <div v-if="!profiles.length" class="text-sm text-muted-foreground px-2 py-2">
              {{ t("profiles.empty") }}
            </div>
            <div v-else class="grid gap-2">
              <div
                v-for="p in profiles"
                :key="p.id"
                class="group flex items-center gap-2 rounded-md border border-input bg-background/40 px-2 py-1.5 text-xs hover:bg-accent hover:text-accent-foreground"
              >
                <button type="button" class="flex-1 min-w-0 text-left" @click="emit('open-item', 'profiles', String(p.id))">
                  <div class="flex items-center gap-2 min-w-0">
                    <span class="truncate">{{ p.name || ("Profile " + String(p.id).slice(0, 6)) }}</span>
                    <Badge variant="outline" class="font-normal shrink-0">T={{ p.temperature ?? 0.2 }}</Badge>
                    <Badge variant="outline" class="font-normal shrink-0">TopP={{ p.top_p ?? 1 }}</Badge>
                    <Badge variant="outline" class="font-normal shrink-0">{{ p.json_mode ? "JSON" : "Text" }}</Badge>
                    <ChevronRight class="ml-auto h-4 w-4 text-muted-foreground group-hover:text-accent-foreground" />
                  </div>
                </button>
                <Button
                  class="dc-settings-button h-[22px] w-[22px] p-0"
                  variant="ghost"
                  size="icon-sm"
                  :aria-label="t('common.delete')"
                  @click="deleteProfile(String(p.id), p.name || ('Profile ' + String(p.id).slice(0, 6)))"
                >
                  <Trash2 class="h-3.5 w-3.5" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>
        <div class="flex justify-end gap-2 pt-2">
          <Button class="dc-settings-button h-[22px] w-9 p-0" variant="outline" size="icon-sm" :aria-label="t('common.refresh')" @click="loadProfiles">
            <RefreshCw class="h-3.5 w-3.5" />
          </Button>
          <DropdownMenu>
            <DropdownMenuTrigger as-child>
              <Button class="dc-settings-button h-[22px] w-auto px-1.5 gap-1" variant="default" size="icon-sm" :aria-label="t('profiles.add')">
                <Plus class="h-3.5 w-3.5" />
                <ChevronDown class="h-3.5 w-3.5" />
              </Button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="center" class="text-xs">
              <DropdownMenuItem class="text-xs" @select.prevent="emit('open-item', 'profiles', 'new')">
                <Plus class="h-3.5 w-3.5" />
                <span>{{ t("profiles.add") }}</span>
              </DropdownMenuItem>
            </DropdownMenuContent>
          </DropdownMenu>
        </div>
      </div>
    </div>

    <!-- Detail page -->
    <Card v-else class="border bg-background/50">
      <CardContent class="p-4">
        <!-- Glossary detail -->
        <div v-if="activeKind === 'glossary'" class="space-y-6">
          <div class="space-y-4">
            <div class="grid gap-2">
              <Label>{{ t("common.name") }}</Label>
              <Input v-model="glSetForm.name" class="dc-settings-field" :placeholder="t('glossary.name_placeholder')" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t("common.description") }}</Label>
              <Input v-model="glSetForm.description" class="dc-settings-field" :placeholder="t('glossary.desc_placeholder')" />
            </div>

            <div class="flex items-center justify-between gap-2">
              <div class="text-xs text-muted-foreground">
                <span v-if="glSetForm.id">{{ glSetForm.id }}</span>
                <span v-else>{{ t("glossary.save_set_first_hint") }}</span>
              </div>
              <div class="flex items-center gap-2">
                <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="!glSetForm.name.trim()" @click="saveGlossarySet">
                  {{ t("common.save") }}
                </Button>
                <Button
                  v-if="glSetForm.id"
                  class="dc-settings-button"
                  variant="destructive"
                  size="sm"
                  @click="deleteGlossarySet(glSetForm.id, glSetForm.name)"
                >
                  <Trash2 class="h-4 w-4" />
                  {{ t("common.delete") }}
                </Button>
              </div>
            </div>
          </div>

          <Separator />

          <div v-if="!glSetForm.id" class="text-sm text-muted-foreground">
            {{ t("glossary.save_set_first_hint") }}
          </div>

          <div v-else class="space-y-4">
            <div class="grid gap-4">
              <div class="grid gap-2">
                <Label>{{ t("glossary.term") }}</Label>
                <Input v-model="glDraft.source" class="dc-settings-field" :placeholder="t('glossary.term_placeholder')" />
              </div>

              <div class="flex items-center justify-between gap-3">
                <div class="flex items-center gap-2">
                  <Label class="text-sm">{{ t("glossary.case_sensitive") }}</Label>
                  <Switch v-model:checked="glDraft.case_sensitive" />
                </div>

                <ToggleGroup v-model="glDraft.mode" type="single" class="text-xs">
                  <ToggleGroupItem value="dnt" class="text-xs">{{ t("glossary.dnt") }}</ToggleGroupItem>
                  <ToggleGroupItem value="specify" class="text-xs">{{ t("glossary.specify") }}</ToggleGroupItem>
                </ToggleGroup>
              </div>

              <div v-if="glDraft.mode === 'specify'" class="grid gap-3 md:grid-cols-2">
                <div class="grid gap-2">
                  <Label>{{ t("glossary.lang") }}</Label>
                  <select
                    v-model="glDraft.lang"
                    class="dc-settings-control dc-settings-field flex rounded-md border border-input bg-background text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
                  >
                    <option v-for="opt in glLangOptions" :key="opt" :value="opt">
                      {{ glLangName(opt) }}
                    </option>
                  </select>
                </div>
                <div class="grid gap-2">
                  <Label>{{ t("glossary.translation_placeholder") }}</Label>
                  <Input v-model="glDraft.trans" class="dc-settings-field" :placeholder="t('glossary.translation_placeholder')" />
                </div>
              </div>

              <div class="flex items-center justify-end gap-2">
                <Button class="dc-settings-button" variant="secondary" size="sm" @click="resetGlossaryDraft">
                  {{ t("common.add") }}
                </Button>
                <Button class="dc-settings-button" variant="default" size="sm" :disabled="!glDraft.source.trim()" @click="saveGlossaryEntry">
                  {{ t("common.save") }}
                </Button>
              </div>
            </div>

            <div class="space-y-2">
              <div class="flex items-center justify-between">
                <div class="text-sm font-medium">{{ t("glossary.terms") }}</div>
                <Badge variant="secondary" class="font-normal">{{ glEntries.length }}</Badge>
              </div>

              <div v-if="!glEntries.length" class="text-sm text-muted-foreground">
                {{ t("glossary.no_terms") }}
              </div>
              <div v-else class="grid gap-2">
                <div
                  v-for="e in glEntries"
                  :key="e.id"
                  class="group flex items-center gap-2 rounded-md border border-input bg-background/40 px-2 py-1.5 text-xs hover:bg-accent hover:text-accent-foreground"
                >
                  <button type="button" class="flex-1 min-w-0 text-left" @click="openGlossaryEntry(e)">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="truncate">{{ e.source }}</span>
                      <Badge variant="outline" class="font-normal shrink-0">{{ glossaryTranslationSummary(e) }}</Badge>
                    </div>
                  </button>
                  <Button class="dc-settings-button" variant="ghost" size="icon-sm" :aria-label="t('common.delete')" @click="deleteGlossaryEntry(e)">
                    <Trash2 class="h-4 w-4" />
                  </Button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Target language detail -->
        <div v-else-if="activeKind === 'target_languages'" class="space-y-4">
          <div class="grid gap-2">
            <Label>{{ t("subtitle.target_languages.code") }}</Label>
            <Input v-model="tlForm.code" class="dc-settings-field" :placeholder="t('subtitle.target_languages.code_placeholder')" />
          </div>
          <div class="grid gap-2">
            <Label>{{ t("subtitle.target_languages.name") }}</Label>
            <Input v-model="tlForm.name" class="dc-settings-field" :placeholder="t('subtitle.target_languages.name_placeholder')" />
          </div>

          <div class="flex items-center justify-end gap-2 pt-2">
            <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="!tlForm.code.trim()" @click="saveTargetLanguage">
              {{ t("common.save") }}
            </Button>
            <Button
              v-if="tlForm.originalCode"
              class="dc-settings-button"
              variant="destructive"
              size="sm"
              @click="deleteTargetLanguage(tlForm.originalCode)"
            >
              <Trash2 class="h-4 w-4" />
              {{ t("common.delete") }}
            </Button>
          </div>
        </div>

        <!-- Profile detail -->
        <div v-else-if="activeKind === 'profiles'" class="space-y-4">
          <div class="grid gap-2">
            <Label>{{ t("profiles.name") }}</Label>
            <Input v-model="pfForm.name" class="dc-settings-field" :placeholder="t('profiles.name_placeholder')" />
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t("profiles.temperature") }}</Label>
              <Input v-model.number="pfForm.temperature" type="number" step="0.1" min="0" max="2" class="dc-settings-field" />
            </div>
            <div class="grid gap-2">
              <Label>{{ t("profiles.top_p") }}</Label>
              <Input v-model.number="pfForm.top_p" type="number" step="0.05" min="0" max="1" class="dc-settings-field" />
            </div>
          </div>

          <div class="grid gap-3 md:grid-cols-2">
            <div class="grid gap-2">
              <Label>{{ t("profiles.max_tokens") }}</Label>
              <Input v-model.number="pfForm.max_tokens" type="number" min="0" class="dc-settings-field" />
            </div>
            <div class="flex items-center gap-2">
              <Label class="text-sm">{{ t("profiles.json_mode") }}</Label>
              <Switch v-model:checked="pfForm.json_mode" />
            </div>
          </div>

          <div class="grid gap-2">
            <Label>{{ t("profiles.sys_prompt_tpl") }}</Label>
            <textarea
              v-model="pfForm.sys_prompt_tpl"
              rows="8"
              class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
              :placeholder="t('profiles.sys_prompt_placeholder')"
            />
          </div>

          <div class="flex items-center justify-end gap-2 pt-2">
            <Button class="dc-settings-button" variant="secondary" size="sm" @click="saveProfile">
              {{ t("common.save") }}
            </Button>
            <Button
              v-if="pfForm.id"
              class="dc-settings-button"
              variant="destructive"
              size="sm"
              @click="deleteProfile(pfForm.id, pfForm.name || ('Profile ' + pfForm.id.slice(0, 6)))"
            >
              <Trash2 class="h-4 w-4" />
              {{ t("common.delete") }}
            </Button>
          </div>
        </div>

        <div v-else class="text-sm text-muted-foreground">
          Unknown page
        </div>
      </CardContent>
    </Card>
  </div>

  <Dialog v-model:open="confirmOpen">
    <DialogContent class="dc-settings-dialog sm:max-w-md">
      <DialogHeader>
        <DialogTitle class="dc-dialog-title">{{ confirmState.title }}</DialogTitle>
        <DialogDescription class="dc-dialog-description">{{ confirmState.description }}</DialogDescription>
      </DialogHeader>
      <DialogFooter class="gap-2">
        <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="confirming" @click="confirmOpen = false">
          {{ t("common.cancel") }}
        </Button>
        <Button class="dc-settings-button" :variant="confirmState.confirmVariant" size="sm" :disabled="confirming" @click="runConfirm">
          {{ confirmState.confirmText }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
