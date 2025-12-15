<script setup lang="ts">
import type { RadioGroupItemProps } from "reka-ui"
import type { HTMLAttributes } from "vue"
import { reactiveOmit } from "@vueuse/core"
import { RadioGroupIndicator, RadioGroupItem } from "reka-ui"
import { cn } from "@/lib/utils"

const props = defineProps<RadioGroupItemProps & { class?: HTMLAttributes["class"] }>()
const delegatedProps = reactiveOmit(props, "class")
</script>

<template>
  <RadioGroupItem
    v-bind="delegatedProps"
    :class="
      cn(
        'dc-radio-group-item',
        'flex w-full items-start gap-3 rounded-lg border p-3 text-left transition-colors',
        'hover:bg-accent focus:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
        'data-[state=checked]:border-ring data-[state=checked]:ring-2 data-[state=checked]:ring-ring data-[state=checked]:ring-offset-2 data-[state=checked]:ring-offset-background',
        'disabled:cursor-not-allowed disabled:opacity-50',
        props.class,
      )
    "
  >
    <RadioGroupIndicator class="dc-radio-indicator mt-0.5 flex h-4 w-4 items-center justify-center">
      <div class="dc-radio-dot h-2.5 w-2.5 rounded-full bg-primary" />
    </RadioGroupIndicator>
    <div class="min-w-0 flex-1">
      <slot />
    </div>
  </RadioGroupItem>
</template>
