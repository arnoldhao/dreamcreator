<script setup lang="ts">
	import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue"
	import { Events } from "@wailsio/runtime"
	import { useI18n } from "vue-i18n"
	import { OpenDirectory, OpenDirectoryDialog } from "bindings/dreamcreator/backend/services/systems/service"
	import { GetPreferencesPath, GetTaskDbPath } from "bindings/dreamcreator/backend/api/pathsapi"
	import { Browser } from "@wailsio/runtime"

	import usePreferencesStore from "@/stores/preferences.js"
	import { applyAccent } from "@/utils/theme.js"
	import iconUrl from "@/assets/images/icon.png"
	import { Project } from "@/consts/global.js"
	import { GetAppVersion } from "bindings/dreamcreator/backend/services/preferences/service"
	import { copyText as copyToClipboard } from "@/utils/clipboard.js"

	import type { SettingsSection } from "./types"
	import SettingsSidebar from "./components/SettingsSidebar.vue"
	import DependencySection from "./sections/DependencySection.vue"
	import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
	import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar"
	import { Button } from "@/components/ui/button"
	import { Badge } from "@/components/ui/badge"
	import { Input } from "@/components/ui/input"
	import { Label } from "@/components/ui/label"
	import { Switch } from "@/components/ui/switch"
	import { FileText, Github, Globe, RefreshCw, Twitter } from "lucide-vue-next"

	const { t, locale } = useI18n()
	const prefStore = usePreferencesStore()

	const section = ref<SettingsSection>("general")
	const appVersion = ref("")

// Cross-window navigation from the main window / menu.
let offNavigate: null | (() => void) = null
onMounted(() => {
  offNavigate = Events.On("settings:navigate", (evt) => {
    const next = typeof evt?.data === "string" ? evt.data : ""
    if (!next) return
    if (next === "dependency") section.value = "dependency"
    else if (next === "about") section.value = "about"
    else section.value = "general"
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
  { value: "blue", label: "Blue", color: "#007AFF" },
  { value: "purple", label: "Purple", color: "#8E44AD" },
  { value: "pink", label: "Pink", color: "#FF2D55" },
  { value: "red", label: "Red", color: "#FF3B30" },
  { value: "orange", label: "Orange", color: "#FF9500" },
  { value: "green", label: "Green", color: "#34C759" },
  { value: "teal", label: "Teal", color: "#30B0C7" },
  { value: "indigo", label: "Indigo", color: "#5856D6" },
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

// --- Paths section state ---
const prefPath = ref("")
const taskDbPath = ref("")
onMounted(async () => {
  try { const r = await GetPreferencesPath(); if (r.success) prefPath.value = r.data } catch {}
  try { const r = await GetTaskDbPath(); if (r.success) taskDbPath.value = r.data } catch {}
})

onMounted(async () => {
  try {
    const r = await GetAppVersion()
    if (r?.data?.version) appVersion.value = r.data.version
  } catch {}
})
const openDirectory = (p: string) => OpenDirectory(p)

// --- Listened ---
const wsListendAddress = computed(() => {
  const info = prefStore.listendInfo?.ws
  return info ? `${info.protocol}://${info.ip}:${info.port}/${info.path}` : ""
})
const mcpListendAddress = computed(() => {
  const info = prefStore.listendInfo?.mcp
  return info ? `${info.protocol}://${info.ip}:${info.port}/${info.path}` : ""
})

async function copyText(text: string) {
  await copyToClipboard(text, t)
}

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
</script>

<template>
  <SidebarProvider :force-mobile="false" :style="{ '--sidebar-width': '200px' }">
    <SettingsSidebar v-model="section" />

	    <SidebarInset>
	      <!-- Top draggable spacer (reserved for future actions like Sponsor, etc.) -->
	      <div class="dc-settings-dragbar flex h-[26px] shrink-0 items-center px-4" />

	      <div class="dc-settings-content flex-1 min-h-0 overflow-auto p-4">
        <Card v-if="section === 'general'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.general.name") }}</CardTitle>
          </CardHeader>
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
                  :class="prefStore.general.theme === c.value ? 'ring-2 ring-sidebar-primary ring-offset-2' : 'hover:ring-2 hover:ring-sidebar-border'"
                  :style="{ background: c.color }"
                  :title="c.label"
                  @click="onPickAccent(c.value)"
                />
                <label class="flex items-center gap-2 text-muted-foreground">
                  <span>Custom</span>
                  <input type="color" class="h-7 w-9 rounded-md border border-input bg-background" :value="customAccent" @input="onPickCustomAccent" />
                </label>
              </div>
            </div>

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
          </CardContent>
        </Card>

        <Card v-else-if="section === 'download'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.general.download") }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0 space-y-3">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.download_directory") }}</Label>
                <div class="text-muted-foreground truncate" :title="prefStore.download?.dir">
                  {{ prefStore.download?.dir || "-" }}
                </div>
              </div>
              <Button class="dc-settings-button" variant="secondary" @click="onSelectDownloadDir">
                {{ t("common.change") || "Change" }}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'logs'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.general.log") }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0 space-y-6">
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
	                  <div>
	                  <Label>{{ t("settings.general.log_console") }}</Label>
	                  </div>
	                  <Switch v-model:checked="prefStore.logger.enable_console" @update:checked="prefStore.setLoggerConfig()" />
	                </div>
	                <div class="flex items-center justify-between">
	                  <div>
	                  <Label>{{ t("settings.general.log_file") }}</Label>
	                  </div>
	                  <Switch v-model:checked="prefStore.logger.enable_file" @update:checked="prefStore.setLoggerConfig()" />
	                </div>
	              </div>

	              <div v-if="prefStore.logger.enable_file" class="flex items-center justify-between gap-3">
	                <div class="min-w-0">
	                <Label>{{ t("settings.general.log_dir") }}</Label>
	                  <div class="text-muted-foreground truncate" :title="prefStore.logger.directory">
	                    {{ prefStore.logger.directory || "-" }}
	                  </div>
	                </div>
	              <div class="flex items-center gap-2">
	                <Button class="dc-settings-button" variant="secondary" @click="openDirectory(prefStore.logger.directory)">
	                  {{ t("download.open_folder") }}
	                </Button>
	                <Button class="dc-settings-button" variant="secondary" @click="onSelectLoggerDir">
	                  {{ t("common.change") || "Change" }}
	                </Button>
	              </div>
	            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'paths'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.general.saved_path") }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0 space-y-4">
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.config_path") }}</Label>
                <div class="text-muted-foreground truncate" :title="prefPath">{{ prefPath || "-" }}</div>
              </div>
              <Button class="dc-settings-button" variant="secondary" :disabled="!prefPath" @click="openDirectory(prefPath)">{{ t("download.open_folder") }}</Button>
            </div>
            <div class="flex items-center justify-between gap-3">
              <div class="min-w-0">
                <Label>{{ t("settings.general.data_path") }}</Label>
                <div class="text-muted-foreground truncate" :title="taskDbPath">{{ taskDbPath || "-" }}</div>
              </div>
              <Button class="dc-settings-button" variant="secondary" :disabled="!taskDbPath" @click="openDirectory(taskDbPath)">{{ t("download.open_folder") }}</Button>
            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'listened'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.general.listend") }}</CardTitle>
          </CardHeader>
          <CardContent class="p-0 space-y-4">
            <div class="flex items-end gap-2">
              <div class="grid gap-2 min-w-0">
                <Label>WebSocket</Label>
                <Input class="dc-settings-control dc-settings-field" readonly :model-value="wsListendAddress" />
              </div>
              <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="!wsListendAddress" @click="copyText(wsListendAddress)">
                {{ t("common.copy") }}
              </Button>
            </div>
            <div class="flex items-end gap-2">
              <div class="grid gap-2 min-w-0">
                <Label>MCP Server</Label>
                <Input class="dc-settings-control dc-settings-field" readonly :model-value="mcpListendAddress" />
              </div>
              <Button class="dc-settings-button" variant="secondary" size="sm" :disabled="!mcpListendAddress" @click="copyText(mcpListendAddress)">
                {{ t("common.copy") }}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Card v-else-if="section === 'ack'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.acknowledgments") }}</CardTitle>
          </CardHeader>
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

        <div v-else-if="section === 'dependency'">
          <DependencySection />
        </div>

        <Card v-else-if="section === 'about'" class="border-0 bg-transparent shadow-none">
          <CardHeader class="dc-settings-dragbar p-0 pb-4">
            <CardTitle class="text-[15px] font-normal">{{ t("settings.about.title") }}</CardTitle>
          </CardHeader>
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
