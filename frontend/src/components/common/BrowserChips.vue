<template>
  <div class="chips" v-if="options && options.length">
    <button v-for="opt in options" :key="opt" class="chip-frosted chip-md chip-translucent" :class="{ active: isActive(opt), primary: isPrimary(opt) }" @click="onPick(opt)">
      <span class="dot" />
      <span class="label">{{ opt }}</span>
    </button>
  </div>
</template>

<script setup>
import { defineProps, defineEmits } from 'vue'

const props = defineProps({
  options: { type: Array, default: () => [] },
  modelValue: { type: String, default: '' },
  defaultOption: { type: String, default: '' }
})

const emit = defineEmits(['update:modelValue','picked'])

const isActive = (opt) => {
  if (opt === '无浏览器') return props.modelValue === ''
  return props.modelValue === opt
}

const isPrimary = (opt) => {
  return props.defaultOption && props.defaultOption === opt && !isActive(opt)
}

const onPick = (opt) => {
  if (opt === '无浏览器') emit('update:modelValue', '')
  else emit('update:modelValue', opt)
  emit('picked', opt)
}
</script>

<style scoped>
.chips { display: flex; flex-wrap: wrap; gap: 8px; padding-top: 10px; justify-content: center; }
/* tune states on top of global chip-frosted */
.chip-frosted.active { border-color: var(--macos-blue); }
.chip-frosted.primary { border-color: var(--macos-blue); }
.dot { width: 6px; height: 6px; border-radius: 50%; background: var(--macos-text-tertiary); }
.chip-frosted.active .dot { background: var(--macos-blue); }
.label { font-size: var(--fs-sub); }
</style>
