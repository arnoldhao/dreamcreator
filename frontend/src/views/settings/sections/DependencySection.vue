<script setup lang="ts">
import { computed, onMounted, ref } from "vue"
import { useI18n } from "vue-i18n"

import useDependenciesStore from "@/stores/dependencies"
import { ListMirrors } from "bindings/dreamcreator/backend/api/dependenciesapi"
import { OpenDirectory } from "bindings/dreamcreator/backend/services/systems/service"
import { isWindows } from "@/utils/platform.js"

import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group"
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip"
import { FolderOpen, RefreshCw, ShieldCheck, Trash2 } from "lucide-vue-next"

type Mirror = {
  name: string
  desc?: string
  recommended?: boolean
}

const { t } = useI18n()
const deps = useDependenciesStore()

const isBusy = computed(() => deps.loading || deps.validating)
const dependencyList = computed(() => Object.entries(deps.dependencies))

const mirrorSheetOpen = ref(false)
const mirrors = ref<Mirror[]>([])
const selectedMirror = ref("")
const currentDepType = ref("")
const currentAction = ref<"install" | "update" | "reinstall">("install")

function dirname(p: string) {
  const s = String(p || "").replace(/[\\/]+$/, "")
  const idx = Math.max(s.lastIndexOf("/"), s.lastIndexOf("\\"))
  return idx >= 0 ? s.slice(0, idx) : s
}

function commonDir(paths: string[]) {
  const nonEmpty = (paths || []).map((p) => String(p || "").trim()).filter(Boolean)
  if (!nonEmpty.length) return ""
  const sep = isWindows() ? "\\" : "/"
  const split = (p: string) => dirname(p).replace(/\\/g, "/").split("/").filter(Boolean)
  let common = split(nonEmpty[0])
  for (const p of nonEmpty.slice(1)) {
    const parts = split(p)
    while (common.length && common.some((seg, i) => parts[i] !== seg)) common.pop()
  }
  if (!common.length) return ""
  return (isWindows() ? (nonEmpty[0].includes(":") ? nonEmpty[0].slice(0, 2) + sep : "") : "/") + common.join(sep)
}

const dependenciesFolder = computed(() => {
  const paths = Object.values(deps.dependencies || {})
    .map((d: any) => d?.path || d?.execPath || "")
    .filter(Boolean)
  return commonDir(paths)
})

async function openDependenciesFolder() {
  const p = dependenciesFolder.value
  if (!p) {
    $message?.warning?.(t("common.not_set") || "Not set")
    return
  }
  try {
    await OpenDirectory(p)
  } catch (e: any) {
    $message?.error?.(e?.message || String(e))
  }
}

onMounted(async () => {
  try {
    deps.setupWebSocketListeners()
    await deps.loadDependencies()
  } catch {}
})

function showLastCheckError(depType: string) {
  const dep = (deps.dependencies as any)?.[depType]
  if (!dep) return
  const headerBase = t("settings.dependency.check_updates_failed")
  const codeKey = `settings.dependency.error.${dep.lastCheckErrorCode || "unknown"}`
  const title = dep.lastCheckErrorCode ? `${headerBase}: ${t(codeKey)}` : headerBase
  const msg = dep.lastCheckError || t("settings.dependency.check_updates_failed")
  $dialog?.error?.({ title, content: msg })
}

async function checkUpdates() {
  try { await deps.checkUpdates() } catch {}
}

async function validate() {
  try { await deps.validateDependencies() } catch {}
}

async function clean() {
  try { await deps.cleanDependencies() } catch {}
}

async function openMirrorSelector(depType: string, action: "install" | "update" | "reinstall") {
  currentDepType.value = depType
  currentAction.value = action
  mirrors.value = []
  selectedMirror.value = ""
  mirrorSheetOpen.value = true

  try {
    const resp = await ListMirrors(depType)
    if (!resp?.success) throw new Error(resp?.msg || "List mirrors failed")
    const parsed = JSON.parse(resp.data || "[]") as Mirror[]
    mirrors.value = Array.isArray(parsed) ? parsed : []
    const recommended = mirrors.value.find((m) => m.recommended)
    selectedMirror.value = recommended?.name || mirrors.value[0]?.name || ""
  } catch (e: any) {
    mirrorSheetOpen.value = false
    $message?.error?.(e?.message || t("settings.dependency.load_mirrors_failed"))
  }
}

async function performWithMirror() {
  const depType = currentDepType.value
  const mirror = selectedMirror.value
  const action = currentAction.value
  mirrorSheetOpen.value = false

  if (!depType) return
  try {
    if (action === "install") await deps.installDependency(depType, "latest", mirror)
    else if (action === "update") await deps.updateDependency(depType, mirror)
    else await deps.reinstallDependency(depType, mirror)
  } catch {}
}
</script>

<template>
  <Card class="border-0 bg-transparent shadow-none">
    <CardHeader class="dc-settings-dragbar p-0 pb-4 flex flex-row items-center justify-between gap-3">
      <CardTitle class="text-[15px] font-normal">{{ t("settings.dependency.title") }}</CardTitle>

      <div class="flex items-center gap-2 whitespace-nowrap">
        <Tooltip>
          <TooltipTrigger as-child>
            <Button
              variant="secondary"
              size="icon-sm"
              class="h-7 w-7 [&_svg]:size-3.5"
              :disabled="isBusy"
              :aria-label="t('settings.dependency.validate')"
              @click="validate"
            >
              <ShieldCheck />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" align="center">
            {{ t("settings.dependency.validate") }}
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger as-child>
            <Button
              variant="secondary"
              size="icon-sm"
              class="h-7 w-7 [&_svg]:size-3.5"
              :disabled="!deps.allowCheckUpdates"
              :aria-label="t('settings.dependency.check_updates')"
              @click="checkUpdates"
            >
              <RefreshCw />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" align="center">
            {{ t("settings.dependency.check_updates") }}
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger as-child>
            <Button
              variant="secondary"
              size="icon-sm"
              class="h-7 w-7 [&_svg]:size-3.5"
              :disabled="isBusy"
              :aria-label="t('settings.dependency.clean_cache')"
              @click="clean"
            >
              <Trash2 />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" align="center">
            {{ t("settings.dependency.clean_cache") }}
          </TooltipContent>
        </Tooltip>

        <Tooltip>
          <TooltipTrigger as-child>
            <Button
              variant="secondary"
              size="icon-sm"
              class="h-7 w-7 [&_svg]:size-3.5"
              :disabled="!dependenciesFolder"
              :aria-label="t('download.open_folder')"
              @click="openDependenciesFolder"
            >
              <FolderOpen />
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom" align="center">
            {{ t("download.open_folder") }}
          </TooltipContent>
        </Tooltip>
      </div>
    </CardHeader>

    <CardContent class="p-0 space-y-3">
      <div class="rounded-lg border bg-background/50 divide-y divide-border">
        <div
          v-for="[key, dep] in dependencyList"
          :key="key"
          class="p-3"
        >
          <div class="flex items-start justify-between gap-4">
            <div class="min-w-0 space-y-1">
              <div class="flex flex-wrap items-center gap-2 min-w-0">
                <div class="truncate font-medium">{{ dep.name }}</div>
                <Badge v-if="dep.available" variant="secondary" class="font-normal">{{ t("settings.dependency.installed") }}</Badge>
                <Badge v-else variant="outline" class="font-normal">{{ t("settings.dependency.not_installed") }}</Badge>
                <Badge v-if="dep.needUpdate" variant="destructive" class="font-normal">{{ t("settings.dependency.update") }}</Badge>

                <button
                  v-if="dep.lastCheckAttempted && !dep.lastCheckSuccess"
                  type="button"
                  class="text-xs text-destructive underline underline-offset-4"
                  @click="showLastCheckError(key)"
                >
                  {{ t("settings.dependency.check_updates_failed") }}
                </button>
              </div>

              <div v-if="dep.available" class="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground">
                <div class="truncate">
                  {{ t("settings.dependency.version") }}: {{ dep.version || "-" }}
                  <template v-if="dep.needUpdate"> → {{ dep.latestVersion || "-" }}</template>
                </div>
              </div>
            </div>

            <div class="shrink-0 flex items-center gap-2">
              <template v-if="dep.available">
                <Button
                  v-if="dep.needUpdate && !dep.installing"
                  class="dc-settings-button"
                  size="sm"
                  @click="openMirrorSelector(key, 'update')"
                >
                  {{ t("settings.dependency.update") }}
                </Button>
                <Button
                  v-if="!dep.installing"
                  class="dc-settings-button"
                  variant="secondary"
                  size="sm"
                  @click="openMirrorSelector(key, 'reinstall')"
                >
                  {{ t("settings.dependency.reinstall") }}
                </Button>
                <Button v-else class="dc-settings-button" size="sm" disabled>
                  {{ dep.currentAction === 'install' ? t("settings.dependency.installing") : t("settings.dependency.updating") }}
                </Button>
              </template>
              <template v-else>
                <Button
                  class="dc-settings-button"
                  variant="secondary"
                  size="sm"
                  :disabled="!deps.allowCheckUpdates"
                  @click="deps.repairDependency(key)"
                >
                  {{ t("settings.dependency.repair") }}
                </Button>
                <Button
                  class="dc-settings-button"
                  size="sm"
                  :disabled="!deps.allowCheckUpdates"
                  @click="openMirrorSelector(key, 'install')"
                >
                  {{ t("settings.dependency.install") }}
                </Button>
              </template>
            </div>
          </div>

          <div v-if="dep.installing" class="mt-3 space-y-2">
            <div class="h-1.5 w-full rounded-full bg-muted overflow-hidden">
              <div class="h-full bg-primary" :style="{ width: `${Math.min(100, Math.max(0, Number(dep.installProgressPercent) || 0))}%` }" />
            </div>
            <div class="flex items-center justify-between gap-3 text-xs text-muted-foreground">
              <div class="min-w-0 truncate">{{ dep.installProgress }}</div>
              <div class="shrink-0 tabular-nums">{{ Math.min(100, Math.max(0, Number(dep.installProgressPercent) || 0)) }}%</div>
            </div>
          </div>
        </div>
      </div>
    </CardContent>
  </Card>

  <Dialog v-model:open="mirrorSheetOpen">
    <DialogContent class="dc-settings-dialog sm:max-w-md">
      <DialogHeader>
        <DialogTitle>{{ t("settings.dependency.select_mirror") }}</DialogTitle>
        <DialogDescription class="flex items-center gap-2">
          <span class="text-muted-foreground">
            {{
              currentAction === "install"
                ? t("settings.dependency.install")
                : currentAction === "reinstall"
                  ? t("settings.dependency.reinstall")
                  : t("settings.dependency.update")
            }}
          </span>
          <span v-if="currentDepType" class="text-muted-foreground">·</span>
          <span v-if="currentDepType" class="text-foreground">{{ currentDepType }}</span>
        </DialogDescription>
      </DialogHeader>

      <div class="space-y-4">
        <div v-if="!mirrors.length" class="text-sm text-muted-foreground">
          {{ t("common.loading") }}
        </div>

        <RadioGroup v-else v-model="selectedMirror">
          <RadioGroupItem v-for="m in mirrors" :key="m.name" :value="m.name">
            <div class="flex flex-wrap items-center gap-2 min-w-0">
              <div class="truncate font-medium">{{ m.name }}</div>
              <Badge v-if="m.recommended" variant="secondary" class="font-normal">
                {{ t("settings.dependency.recommended") }}
              </Badge>
            </div>
            <div v-if="m.desc" class="mt-1 text-sm text-muted-foreground">{{ m.desc }}</div>
          </RadioGroupItem>
        </RadioGroup>
      </div>

      <DialogFooter class="mt-4">
        <Button class="dc-settings-button" variant="secondary" @click="mirrorSheetOpen = false">
          {{ t("common.cancel") }}
        </Button>
        <Button class="dc-settings-button" :disabled="!selectedMirror" @click="performWithMirror">
          {{ t("common.confirm") }}
        </Button>
      </DialogFooter>
    </DialogContent>
  </Dialog>
</template>
