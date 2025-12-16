<script setup lang="ts">
	import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
	import { Events } from "@wailsio/runtime"
	import { useI18n } from "vue-i18n"
	import { OpenDirectory, OpenDirectoryDialog } from "bindings/dreamcreator/backend/services/systems/service"
	import { GetTaskDbPath } from "bindings/dreamcreator/backend/api/pathsapi"
	import { Browser } from "@wailsio/runtime"

	import usePreferencesStore from "@/stores/preferences.js"
	import useLayoutStore from "@/stores/layout.js"
	import { applyAccent } from "@/utils/theme.js"
	import iconUrl from "@/assets/images/icon.png"
	import { Project } from "@/consts/global.js"
	import { GetAppVersion } from "bindings/dreamcreator/backend/services/preferences/service"

	import type { LLMAssetsKind, SettingsRoute, SettingsSection } from "./types"
	import SettingsSidebar from "./components/SettingsSidebar.vue"
	import DependencySection from "./sections/DependencySection.vue"
	import CookiesSection from "./sections/CookiesSection.vue"
	import ProvidersSection from "./sections/ProvidersSection.vue"
	import LLMAssetsSection from "./sections/LLMAssetsSection.vue"
	import { Card, CardContent } from "@/components/ui/card"
	import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
	import { Button } from "@/components/ui/button"
	import { Badge } from "@/components/ui/badge"
	import { Input } from "@/components/ui/input"
	import { Label } from "@/components/ui/label"
	import { Switch } from "@/components/ui/switch"
	import { ChevronLeft, ChevronRight, FileText, Github, Globe, RefreshCw, Twitter } from "lucide-vue-next"

	const { t, locale } = useI18n()
	const prefStore = usePreferencesStore()
	const layout = useLayoutStore()

	const section = ref<SettingsSection>("general")
	const providersProviderId = ref("")
	const llmAssetsKind = ref<LLMAssetsKind | undefined>(undefined)
	const llmAssetsId = ref<string | undefined>(undefined)
	const appVersion = ref("")

// Cross-window navigation from the main window / menu.
let offNavigate: null | (() => void) = null
const sectionKeys: SettingsSection[] = [
  "general",
  "appearance",
  "storage",
  "dependencies",
  "cookies",
  "providers",
  "llm_assets",
  "acknowledgements",
  "about",
]
function toSection(val: unknown): SettingsSection | null {
  const s = typeof val === "string" ? val : (val && typeof val === "object" ? String((val as any).section || "") : "")
  return (sectionKeys as string[]).includes(s) ? (s as SettingsSection) : null
}

const llmAssetsKinds: LLMAssetsKind[] = ["glossary", "target_languages", "profiles"]
function toLLMAssetsKind(val: unknown): LLMAssetsKind | null {
  const k = String(val || "") as LLMAssetsKind
  return llmAssetsKinds.includes(k) ? k : null
}

function toRoute(val: unknown): SettingsRoute | null {
  if (!val) return null
  if (typeof val === "string") {
    const sec = toSection(val)
    return sec ? { section: sec } : null
  }
  if (typeof val !== "object") return null
  const sec = toSection((val as any).section)
  if (!sec) return null
  const out: SettingsRoute = { section: sec }
  if (sec === "providers" && (val as any).providerId) out.providerId = String((val as any).providerId)
  if (sec === "llm_assets") {
    const kind = toLLMAssetsKind((val as any).llmAssetsKind)
    const id = String((val as any).llmAssetsId || "")
    if (kind && id) {
      out.llmAssetsKind = kind
      out.llmAssetsId = id
    }
  }
  return out
}

// macOS Settings-like navigation: remember visited pages and allow back/forward.
const history = ref<SettingsRoute[]>([{ section: "general" }])
const historyIndex = ref(0)
const canGoBack = computed(() => historyIndex.value > 0)
const canGoForward = computed(() => historyIndex.value < history.value.length - 1)

function normalizeRoute(r: SettingsRoute): SettingsRoute {
  const out: SettingsRoute = { section: r.section }
  if (r.section === "providers" && r.providerId) out.providerId = String(r.providerId)
  if (r.section === "llm_assets") {
    const kind = toLLMAssetsKind(r.llmAssetsKind)
    const id = String(r.llmAssetsId || "")
    if (kind && id) {
      out.llmAssetsKind = kind
      out.llmAssetsId = id
    }
  }
  return out
}

function sameRoute(a: SettingsRoute, b: SettingsRoute) {
  const aa = normalizeRoute(a)
  const bb = normalizeRoute(b)
  return (
    aa.section === bb.section &&
    (aa.providerId || "") === (bb.providerId || "") &&
    (aa.llmAssetsKind || "") === (bb.llmAssetsKind || "") &&
    (aa.llmAssetsId || "") === (bb.llmAssetsId || "")
  )
}

function applyRoute(next: SettingsRoute) {
  const r = normalizeRoute(next)
  section.value = r.section
  providersProviderId.value = r.section === "providers" ? (r.providerId || "") : ""
  llmAssetsKind.value = r.section === "llm_assets" ? r.llmAssetsKind : undefined
  llmAssetsId.value = r.section === "llm_assets" ? r.llmAssetsId : undefined
}

function navigateTo(next: SettingsRoute, opts: { push?: boolean } = {}) {
  if (!next?.section) return
  const push = opts.push !== false
  const normalized = normalizeRoute(next)
  const current = history.value[historyIndex.value] || { section: "general" }
  if (sameRoute(current, normalized)) return
  applyRoute(normalized)
  if (!push) return
  history.value = history.value.slice(0, historyIndex.value + 1)
  history.value.push(normalized)
  historyIndex.value = history.value.length - 1
}

function navigateToSection(next: SettingsSection, opts: { push?: boolean } = {}) {
  navigateTo({ section: next }, opts)
}

function goBack() {
  if (!canGoBack.value) return
  historyIndex.value -= 1
  applyRoute(history.value[historyIndex.value] || { section: "general" })
}

function goForward() {
  if (!canGoForward.value) return
  historyIndex.value += 1
  applyRoute(history.value[historyIndex.value] || { section: "general" })
}

const contentTitle = computed(() => {
  switch (section.value) {
    case "general": return t("settings.general.name")
    case "appearance": return t("settings.sections.appearance")
    case "storage": return t("settings.sections.storage")
    case "dependencies": return t("settings.sections.dependencies")
    case "cookies": return t("settings.sections.cookies")
    case "providers": return t("settings.sections.providers")
    case "llm_assets": return t("settings.sections.llm_assets")
    case "acknowledgements": return t("settings.acknowledgments")
    case "about": return t("settings.about.title")
    default: return ""
  }
})

onMounted(() => {
  offNavigate = Events.On("settings:navigate", (evt) => {
    const next = toRoute(evt?.data)
    if (next) navigateTo(next)
  })
})
onBeforeUnmount(() => {
  try { offNavigate?.() } catch {}
  offNavigate = null
})

// --- General section state ---
const proxyAddress = ref("")
const isValidProxyAddress = (address: string) => /^(http|https|socks5):\/\/[a-zA-Z0-9.-]+:[0-9]+$/.test(address)
watch(
  () => prefStore.proxy,
  (p) => { proxyAddress.value = p?.proxy_address || "" },
  { immediate: true, deep: true },
)

async function onLanguageChange() {
  await prefStore.savePreferences()
  locale.value = prefStore.currentLanguage
  try { Events.Emit("app:localeChanged", prefStore.currentLanguage) } catch {}
}

const accentOptions = [
  { value: "blue", label: "Blue", swatchClass: "bg-[#007AFF]" },
  { value: "purple", label: "Purple", swatchClass: "bg-[#8E44AD]" },
  { value: "pink", label: "Pink", swatchClass: "bg-[#FF2D55]" },
  { value: "red", label: "Red", swatchClass: "bg-[#FF3B30]" },
  { value: "orange", label: "Orange", swatchClass: "bg-[#FF9500]" },
  { value: "green", label: "Green", swatchClass: "bg-[#34C759]" },
  { value: "teal", label: "Teal", swatchClass: "bg-[#30B0C7]" },
  { value: "indigo", label: "Indigo", swatchClass: "bg-[#5856D6]" },
]

const customAccent = ref("#007AFF")
watch(
  () => prefStore.general.theme,
  (v: string) => { if (typeof v === "string" && v.startsWith("#")) customAccent.value = v },
  { immediate: true },
)

async function onPickAccent(name: string) {
  if (prefStore.general.theme === name) return
  prefStore.general.theme = name
  applyAccent(name, prefStore.isDark)
  await prefStore.savePreferences()
}

async function onPickCustomAccent(e: Event) {
  const val = (e.target as HTMLInputElement | null)?.value
  if (!val) return
  prefStore.general.theme = val
  applyAccent(val, prefStore.isDark)
  await prefStore.savePreferences()
}

async function onProxyChange() {
  if (prefStore.proxy.type === "manual") {
    if (!proxyAddress.value.trim() || !isValidProxyAddress(proxyAddress.value)) {
      $message?.error?.(t("settings.general.proxy_address_err"))
      return
    }
    prefStore.proxy.proxy_address = proxyAddress.value
  }
  await prefStore.setProxyConfig()
}

async function onSelectDownloadDir() {
  const { success, data, msg } = await OpenDirectoryDialog(prefStore.download?.dir || "")
  if (success && data?.path?.trim()) {
    prefStore.download.dir = data.path
    await prefStore.SetDownloadConfig()
  } else if (msg) {
    $message?.error?.(msg)
  }
}

async function onSelectLoggerDir() {
  const seed = prefStore.logger?.directory || ""
  const { success, data, msg } = await OpenDirectoryDialog(seed)
  if (success && data?.path?.trim()) {
    prefStore.logger.directory = data.path
    await prefStore.setLoggerConfig()
  } else if (msg) {
    $message?.error?.(msg)
  }
}

// --- Storage section state ---
const taskDbPath = ref("")
onMounted(async () => {
  try { const r = await GetTaskDbPath(); if (r.success) taskDbPath.value = r.data } catch {}
})

onMounted(async () => {
  try {
    const r = await GetAppVersion()
    if (r?.data?.version) appVersion.value = r.data.version
  } catch {}
})
const openDirectory = (p: string) => OpenDirectory(p)

function openUrl(url: string) {
  try { Browser.OpenURL(url) } catch { try { window.open(url, "_blank") } catch {} }
}

function openWebsite() { openUrl(Project.OfficialWebsite) }
function openGithub() { openUrl(Project.Github) }
function openTwitter() { openUrl(Project.Twitter) }
function openChangelog() { openUrl(Project.Github + "/releases") }

async function checkUpdates() {
  try { await prefStore.checkForUpdate(true) } catch {}
}

// Scroll state for the Settings content column (controls the frosted divider visibility).
const settingsScrollEl = ref<HTMLElement | null>(null)
const contentScrolled = ref(false)
const onSettingsScroll = () => {
  try { contentScrolled.value = (settingsScrollEl.value?.scrollTop || 0) > 0 } catch { contentScrolled.value = false }
}

onMounted(() => {
  const el = settingsScrollEl.value
  if (!el) return
  try {
    el.addEventListener("scroll", onSettingsScroll, { passive: true })
    onSettingsScroll()
  } catch {}
})
onBeforeUnmount(() => {
  try { settingsScrollEl.value?.removeEventListener("scroll", onSettingsScroll) } catch {}
})
</script>

<template>
  <SidebarProvider class="h-full min-h-0 w-full" :force-mobile="false" :sidebar-width="`${layout.ribbonWidth || 160}px`">
    <SettingsSidebar :model-value="section" @update:modelValue="(v) => navigateToSection(v)" />

	    <SidebarInset class="flex flex-col min-h-0 pt-2 pr-2 pb-2">
	      <!-- Content title: navigation + current section title (locked) -->
	      <div class="dc-settings-dragbar dc-settings-toolbar flex h-[38px] shrink-0 items-start gap-3 px-4">
          <div class="inline-flex items-center rounded-md border border-sidebar-border bg-sidebar/70 text-sidebar-foreground backdrop-blur-md overflow-hidden">
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              class="h-6 w-6 rounded-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground [&_svg]:size-3.5"
              :disabled="!canGoBack"
              :aria-label="t('common.previous') || 'Back'"
              @click="goBack"
            >
              <ChevronLeft aria-hidden="true" />
            </Button>
            <div class="h-4 w-px bg-sidebar-border" />
            <Button
              type="button"
              variant="ghost"
              size="icon-sm"
              class="h-6 w-6 rounded-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground [&_svg]:size-3.5"
              :disabled="!canGoForward"
              :aria-label="t('common.next') || 'Forward'"
              @click="goForward"
            >
              <ChevronRight aria-hidden="true" />
            </Button>
          </div>
          <div class="min-w-0 flex-1 h-6 flex items-center truncate text-[15px] font-normal text-foreground">
            {{ contentTitle }}
          </div>
        </div>

	      <div
          ref="settingsScrollEl"
          class="dc-settings-content flex flex-col flex-1 min-h-0 overflow-auto p-4 pb-2"
          :class="{ scrolled: contentScrolled }"
        >
        <Card v-if="section === 'general'" class="border-0 bg-transparent shadow-none">
	          <CardContent class="p-0 space-y-6">
	            <div class="grid gap-2">
	              <Label>{{ t("settings.general.language") }}</Label>
	              <select
	                class="dc-settings-control dc-settings-field flex rounded-md border border-input bg-background text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
	                v-model="prefStore.general.language"
	                @change="onLanguageChange"
	              >
	                <option v-for="option in prefStore.langOption" :key="option.value" :value="option.value">
	                  {{ option.label }}
	                </option>
	              </select>
	            </div>

	            <div class="grid gap-2">
	              <Label>{{ t("settings.general.proxy") }}</Label>
	              <select
	                class="dc-settings-control dc-settings-field flex rounded-md border border-input bg-background text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
	                v-model="prefStore.proxy.type"
	                @change="onProxyChange"
	              >
	                <option value="none">{{ t("settings.general.proxy_none") }}</option>
	                <option value="system">{{ t("settings.general.proxy_system") }}</option>
	                <option value="manual">{{ t("settings.general.proxy_manual") }}</option>
	              </select>
	            </div>

	            <div v-if="prefStore.proxy.type === 'manual'" class="grid gap-2">
	              <Label>{{ t("settings.general.proxy_address") }}</Label>
	              <Input
	                class="dc-settings-control dc-settings-field"
	                v-model="proxyAddress"
	                placeholder="http://127.0.0.1:7890"
	                @change="onProxyChange"
	              />
	            </div>

	            <div class="grid gap-2">
	              <Label>{{ t("settings.general.log_level") }}</Label>
	              <select
	                class="dc-settings-control dc-settings-field flex rounded-md border border-input bg-background text-sm ring-offset-background focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2"
	                v-model="prefStore.logger.level"
	                @change="prefStore.setLoggerConfig()"
	              >
	                <option value="debug">Debug</option>
	                <option value="info">Info</option>
	                <option value="warn">Warn</option>
	                <option value="error">Error</option>
	              </select>
	            </div>

	            <div class="grid grid-cols-2 gap-4">
	              <div class="grid gap-2">
	                <Label>{{ t("settings.general.log_max_size") }} (MB)</Label>
	                <Input class="dc-settings-control dc-settings-field" type="number" min="1" max="100" v-model="prefStore.logger.max_size" @change="prefStore.setLoggerConfig()" />
	              </div>
	              <div class="grid gap-2">
	                <Label>{{ t("settings.general.log_max_age") }} ({{ t("settings.general.days") }})</Label>
	                <Input class="dc-settings-control dc-settings-field" type="number" min="1" max="365" v-model="prefStore.logger.max_age" @change="prefStore.setLoggerConfig()" />
	              </div>
	            </div>

	            <div class="grid gap-4">
	              <div class="flex items-center justify-between">
	                <Label>{{ t("settings.general.log_console") }}</Label>
	                <Switch v-model:checked="prefStore.logger.enable_console" @update:checked="prefStore.setLoggerConfig()" />
	              </div>
	              <div class="flex items-center justify-between">
	                <Label>{{ t("settings.general.log_file") }}</Label>
	                <Switch v-model:checked="prefStore.logger.enable_file" @update:checked="prefStore.setLoggerConfig()" />
	              </div>
	            </div>
	          </CardContent>
	        </Card>

        <Card v-else-if="section === 'appearance'" class="border-0 bg-transparent shadow-none">
          <CardContent class="p-0 space-y-6">
            <div class="grid gap-2">
              <Label>{{ t("settings.general.appearance") }}</Label>
              <div class="flex flex-wrap items-center gap-2">
                <Button
                  v-for="opt in prefStore.appearanceOption"
                  :key="opt.value"
                  type="button"
                  size="sm"
                  class="dc-settings-button"
                  :class="prefStore.general.appearance === opt.value ? 'ring-1 ring-ring ring-offset-1 ring-offset-background' : ''"
                  :variant="prefStore.general.appearance === opt.value ? 'default' : 'secondary'"
                  @click="prefStore.general.appearance = opt.value; prefStore.savePreferences()"
                >
                  {{ t(opt.label) }}
                </Button>
              </div>
            </div>

            <div class="grid gap-2">
              <Label>{{ t("settings.general.theme") }}</Label>
              <div class="flex flex-wrap items-center gap-2">
                <button
                  v-for="c in accentOptions"
                  :key="c.value"
                  type="button"
                  class="dc-color-swatch rounded-full border border-border ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2"
                  :class="[
                    c.swatchClass,
                    prefStore.general.theme === c.value ? 'ring-2 ring-sidebar-primary ring-offset-2' : 'hover:ring-2 hover:ring-sidebar-border',
                  ]"
                  :title="c.label"
                  @click="onPickAccent(c.value)"
                />
                <label class="flex items-center gap-2 text-muted-foreground">
                  <span>Custom</span>
                  <input type="color" class="h-7 w-9 rounded-md border border-input bg-background" :value="customAccent" @input="onPickCustomAccent" />
                </label>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'storage'" class="border-0 bg-transparent shadow-none">
          <CardContent class="p-0 space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.download_directory") }}</Label>
                <div class="text-muted-foreground truncate" :title="prefStore.download?.dir">
                  {{ prefStore.download?.dir || "-" }}
                </div>
              </div>
              <div class="flex items-center gap-2">
                <Button class="dc-settings-button" variant="secondary" :disabled="!prefStore.download?.dir" @click="openDirectory(prefStore.download?.dir)">
                  {{ t("download.open_folder") }}
                </Button>
                <Button class="dc-settings-button" variant="secondary" @click="onSelectDownloadDir">
                  {{ t("common.change") || "Change" }}
                </Button>
              </div>
            </div>

            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.data_path") }}</Label>
                <div class="text-muted-foreground truncate" :title="taskDbPath">{{ taskDbPath || "-" }}</div>
              </div>
              <Button class="dc-settings-button" variant="secondary" :disabled="!taskDbPath" @click="openDirectory(taskDbPath)">
                {{ t("download.open_folder") }}
              </Button>
            </div>

            <div v-if="prefStore.logger.enable_file" class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.log_dir") }}</Label>
                <div class="text-muted-foreground truncate" :title="prefStore.logger.directory">
                  {{ prefStore.logger.directory || "-" }}
                </div>
              </div>
              <div class="flex items-center gap-2">
                <Button class="dc-settings-button" variant="secondary" :disabled="!prefStore.logger.directory" @click="openDirectory(prefStore.logger.directory)">
                  {{ t("download.open_folder") }}
                </Button>
                <Button class="dc-settings-button" variant="secondary" @click="onSelectLoggerDir">
                  {{ t("common.change") || "Change" }}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <div v-else-if="section === 'dependencies'" class="flex-1 min-h-0">
          <DependencySection />
        </div>

        <div v-else-if="section === 'cookies'" class="flex-1 min-h-0">
          <CookiesSection />
        </div>

        <div v-else-if="section === 'providers'" class="shrink-0">
          <ProvidersSection
            :active-provider-id="providersProviderId"
            @open-provider="(id) => navigateTo({ section: 'providers', providerId: id })"
            @close-provider="() => navigateToSection('providers')"
          />
        </div>

        <div v-else-if="section === 'llm_assets'" class="shrink-0">
          <LLMAssetsSection
            :active-kind="llmAssetsKind"
            :active-id="llmAssetsId"
            @open-item="(kind, id) => navigateTo({ section: 'llm_assets', llmAssetsKind: kind, llmAssetsId: id })"
            @close-item="() => navigateToSection('llm_assets')"
          />
        </div>

        <Card v-else-if="section === 'acknowledgements'" class="border-0 bg-transparent shadow-none">
          <CardContent class="p-0">
            <div class="rounded-lg border bg-background/50 p-3">
              <div class="flex items-start justify-between gap-3">
                <div class="min-w-0">
                  <div class="font-medium truncate">{{ t("dialogue.about.zhconvert") }}</div>
                  <p class="mt-1 text-sm text-muted-foreground">
                    {{ t("dialogue.about.zhconvert_desc") }}
                  </p>
                </div>
                <Button class="dc-settings-button shrink-0 self-center" variant="secondary" size="sm" @click="openUrl(Project.ZHConvert)">
                  <Globe />
                  {{ t("dialogue.about.website") }}
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'about'" class="border-0 bg-transparent shadow-none">
          <CardContent class="p-0 space-y-4">
            <div class="flex items-center justify-between gap-3 rounded-lg border bg-background/50 p-3">
              <div class="flex items-center gap-3 min-w-0">
                <img :src="iconUrl" alt="app icon" class="h-10 w-10 rounded-xl border" />
                <div class="min-w-0">
                  <div class="flex items-center gap-2 min-w-0">
                    <div class="truncate font-medium leading-none">{{ Project.Name }}（{{ Project.DisplayNameZh }}）</div>
                    <Badge variant="secondary" class="font-normal shrink-0">v{{ appVersion || "-" }}</Badge>
                  </div>
                  <div class="text-xs text-muted-foreground truncate">{{ Project.OfficialWebsite }}</div>
                </div>
              </div>

              <Button class="dc-settings-button shrink-0" variant="default" size="sm" @click="checkUpdates">
                <RefreshCw />
                {{ t("menu.check_update") }}
              </Button>
            </div>

            <div class="flex items-center justify-center gap-2 overflow-x-auto whitespace-nowrap">
              <Button class="dc-settings-button justify-start shrink-0" variant="secondary" size="sm" @click="openWebsite">
                <Globe />
                {{ t("dialogue.about.official_website") }}
              </Button>
              <Button class="dc-settings-button justify-start shrink-0" variant="secondary" size="sm" @click="openChangelog">
                <FileText />
                {{ t("settings.about.changelog") }}
              </Button>
              <Button class="dc-settings-button justify-start shrink-0" variant="secondary" size="sm" @click="openGithub">
                <Github />
                GitHub
              </Button>
              <Button class="dc-settings-button justify-start shrink-0" variant="secondary" size="sm" @click="openTwitter">
                <Twitter />
                Twitter
              </Button>
            </div>

            <div class="space-y-3">
              <div class="flex items-center justify-between gap-3 rounded-lg border bg-background/50 px-3 py-2">
                <Label>{{ t("settings.about.auto_check_update") }}</Label>
                <Switch v-model:checked="prefStore.general.checkUpdate" @update:checked="prefStore.savePreferences()" />
              </div>
              <div class="flex items-center justify-between gap-3 rounded-lg border bg-background/50 px-3 py-2">
                <Label>{{ t("settings.about.telemetry_opt_in") }}</Label>
                <Switch v-model:checked="prefStore.telemetry.enabled" @update:checked="prefStore.savePreferences()" />
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </SidebarInset>
  </SidebarProvider>
</template>
