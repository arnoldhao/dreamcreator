<script setup lang="ts">
import { computed, ref } from "vue"
import { Globe } from "lucide-vue-next"

const props = withDefaults(defineProps<{
  logoKey?: string
  alt?: string
  size?: number
  class?: string
}>(), {
  logoKey: "",
  alt: "",
  size: 16,
  class: "",
})

const failed = ref(false)

const src = computed(() => {
  const key = String(props.logoKey || "").trim()
  if (!key) return ""
  return `https://models.dev/logos/${encodeURIComponent(key)}.svg`
})
</script>

<template>
  <img
    v-if="src && !failed"
    :src="src"
    :alt="alt"
    :width="size"
    :height="size"
    class="shrink-0"
    :class="props.class"
    loading="lazy"
    decoding="async"
    referrerpolicy="no-referrer"
    @error="failed = true"
  />
  <Globe v-else aria-hidden="true" :style="{ width: `${size}px`, height: `${size}px` }" class="shrink-0 text-muted-foreground" :class="props.class" />
</template>

