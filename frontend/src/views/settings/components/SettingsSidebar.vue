<script setup lang="ts">
import type { Component } from "vue"
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Boxes, Cookie, Database, Folder, Handshake, Info, Layers, Palette, Settings } from "lucide-vue-next"
import type { SettingsSection } from "../types"

import {
  Sidebar,
  SidebarContent,
  SidebarGroup,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const props = withDefaults(defineProps<{
  modelValue: SettingsSection
}>(), {
  modelValue: "general",
})

const emit = defineEmits<{
  "update:modelValue": [value: SettingsSection]
}>()

const { t, locale } = useI18n()

const navItems = computed(() => {
  // force i18n dependency
  void locale.value
  return [
    { key: "general", title: t("settings.general.name"), icon: Settings },
    { key: "appearance", title: t("settings.sections.appearance"), icon: Palette },
    { key: "storage", title: t("settings.sections.storage"), icon: Folder },
    { key: "dependencies", title: t("settings.sections.dependencies"), icon: Boxes },
    { key: "cookies", title: t("settings.sections.cookies"), icon: Cookie },
    { key: "providers", title: t("settings.sections.providers"), icon: Database },
    { key: "llm_assets", title: t("settings.sections.llm_assets"), icon: Layers },
    { key: "acknowledgements", title: t("settings.acknowledgments"), icon: Handshake },
    { key: "about", title: t("settings.about.title"), icon: Info },
  ] satisfies Array<{ key: SettingsSection; title: string; icon: Component }>
})

function select(next: SettingsSection) {
  emit("update:modelValue", next)
}
</script>

<template>
  <Sidebar variant="floating">
    <!-- Spacer/drag region for macOS traffic lights (titlebar is hidden inset). -->
    <SidebarHeader class="dc-settings-dragbar h-[38px] p-0" />

    <SidebarContent>
      <SidebarGroup>
        <SidebarMenu class="gap-1">
          <SidebarMenuItem v-for="it in navItems" :key="it.key">
            <SidebarMenuButton
              as-child
              :is-active="props.modelValue === it.key"
              class="data-[active=true]:text-sidebar-primary data-[active=true]:hover:text-sidebar-primary"
            >
              <button type="button" class="w-full text-left" @click="select(it.key)">
                <component :is="it.icon" aria-hidden="true" />
                <span>{{ it.title }}</span>
              </button>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarGroup>
    </SidebarContent>
  </Sidebar>
</template>
