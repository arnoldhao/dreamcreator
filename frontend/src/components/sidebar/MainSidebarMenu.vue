<script setup>
import { computed } from "vue"
import { useI18n } from "vue-i18n"
import { Captions, Download } from "lucide-vue-next"
import useNavStore from "@/stores/nav.js"

import {
  SidebarGroup,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"

const props = defineProps({
  value: {
    type: String,
    default: "download",
  },
})

const emit = defineEmits(["update:value"])

const navStore = useNavStore()
const { t } = useI18n()

const iconByKey = computed(() => ({
  [navStore.navOptions.DOWNLOAD]: Download,
  [navStore.navOptions.SUBTITLE]: Captions,
}))

const mainItems = computed(() => {
  return (navStore.menuOptions || []).map((m) => ({
    key: m.key,
    title: t(m.label),
    Icon: iconByKey.value[m.key],
  }))
})

function select(key) {
  emit("update:value", key)
}
</script>

<template>
  <SidebarGroup class="flex-1">
    <SidebarMenu class="gap-1">
      <SidebarMenuItem v-for="it in mainItems" :key="it.key">
        <SidebarMenuButton
          as-child
          :is-active="props.value === it.key"
          :tooltip="it.title"
          class="data-[active=true]:text-sidebar-primary data-[active=true]:hover:text-sidebar-primary"
        >
          <button type="button" class="w-full text-left" :aria-label="it.title" @click="select(it.key)">
            <component :is="it.Icon" aria-hidden="true" />
            <span class="group-data-[collapsible=icon]:hidden">{{ it.title }}</span>
          </button>
        </SidebarMenuButton>
      </SidebarMenuItem>
    </SidebarMenu>
  </SidebarGroup>
</template>

