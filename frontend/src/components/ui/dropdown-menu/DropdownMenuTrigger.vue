<script setup lang="ts">
import type { DropdownMenuTriggerProps } from "reka-ui"
import { DropdownMenuTrigger } from "reka-ui"

const props = withDefaults(defineProps<DropdownMenuTriggerProps>(), {
  asChild: true,
})
</script>

<template>
  <DropdownMenuTrigger v-bind="props">
    <slot />
  </DropdownMenuTrigger>
</template>

