<script setup>
import { useI18n } from "vue-i18n"
import { Cookie, Database, Layers, MoreHorizontal, Package, Settings2 } from "lucide-vue-next"
import { Events, Window } from "@wailsio/runtime"

import useSettingsStore from "@/stores/settings.js"

import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuSeparator, DropdownMenuTrigger } from "@/components/ui/dropdown-menu"
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem } from "@/components/ui/sidebar"

const settings = useSettingsStore()
const { t } = useI18n()

function openSettings(key) {
  settings.setPage(key)
  try { Events.Emit("settings:navigate", key) } catch {}
  try {
    const sw = Window.Get("settings")
    // Ensure the window becomes visible and comes to the front if already open.
    try { sw.UnMinimise() } catch {}
    try { sw.Show() } catch {}
    try { sw.Focus() } catch {}
  } catch {}
}
</script>

<template>
  <SidebarMenu>
    <SidebarMenuItem>
      <DropdownMenu>
        <DropdownMenuTrigger>
          <SidebarMenuButton
            as-child
            :tooltip="t('bottom.more')"
            class="data-[active=true]:text-sidebar-primary data-[active=true]:hover:text-sidebar-primary"
          >
            <button type="button" class="w-full text-left" :aria-label="t('bottom.more')">
              <MoreHorizontal aria-hidden="true" />
              <span class="group-data-[collapsible=icon]:hidden">{{ t("bottom.more") }}</span>
            </button>
          </SidebarMenuButton>
        </DropdownMenuTrigger>

        <DropdownMenuContent side="top" align="start">
          <DropdownMenuItem @select.prevent="openSettings('general')">
            <Settings2 aria-hidden="true" />
            <span>{{ t("settings.general.name") }}</span>
          </DropdownMenuItem>
          <DropdownMenuItem @select.prevent="openSettings('cookies')">
            <Cookie aria-hidden="true" />
            <span>{{ t("settings.sections.cookies") }}</span>
          </DropdownMenuItem>
          <DropdownMenuSeparator />
          <DropdownMenuItem @select.prevent="openSettings('providers')">
            <Database aria-hidden="true" />
            <span>{{ t("settings.sections.providers") }}</span>
          </DropdownMenuItem>
          <DropdownMenuItem @select.prevent="openSettings('llm_assets')">
            <Layers aria-hidden="true" />
            <span>{{ t("settings.sections.llm_assets") }}</span>
          </DropdownMenuItem>
          <DropdownMenuItem @select.prevent="openSettings('dependencies')">
            <Package aria-hidden="true" />
            <span>{{ t("settings.sections.dependencies") }}</span>
          </DropdownMenuItem>
        </DropdownMenuContent>
      </DropdownMenu>
    </SidebarMenuItem>
  </SidebarMenu>
</template>
