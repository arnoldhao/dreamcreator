<template>
  <div class="chips" v-if="options && options.length">
    <button
      v-for="opt in options"
      :key="opt?.value ?? opt?.label ?? opt"
      class="chip-frosted chip-md chip-translucent"
      :class="chipClass(opt)"
      @click="onPick(opt)"
    >
      <span class="dot" :class="{ manual: opt?.source === 'manual' }" />
      <span class="label">{{ opt?.label ?? opt }}</span>
      <span v-if="opt?.badge" class="badge">{{ opt.badge }}</span>
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
  const value = opt && typeof opt === 'object' ? opt.value : opt
  if (value === undefined || value === null) return false
  if (value === '') return props.modelValue === ''
  return props.modelValue === value
}

const isPrimary = (opt) => {
  const value = opt && typeof opt === 'object' ? opt.value : opt
  if (!props.defaultOption) return false
  return props.defaultOption === value && !isActive(opt)
}

const onPick = (opt) => {
  const value = opt && typeof opt === 'object' ? opt.value : opt
  emit('update:modelValue', value || '')
  emit('picked', opt)
}

const chipClass = (opt) => ({
  active: isActive(opt),
  primary: isPrimary(opt),
  manual: opt && typeof opt === 'object' && opt.source === 'manual'
})
</script>

<style scoped>
.chips { display: flex; flex-wrap: wrap; gap: 8px; padding-top: 10px; justify-content: center; }
/* tune states on top of global chip-frosted */
.chip-frosted.active { border-color: var(--macos-blue); }
.chip-frosted.primary { border-color: var(--macos-blue); }
.chip-frosted.manual { border-color: color-mix(in oklab, var(--macos-blue) 40%, transparent); }
.dot { width: 6px; height: 6px; border-radius: 50%; background: var(--macos-text-tertiary); }
.chip-frosted.active .dot { background: var(--macos-blue); }
.dot.manual { background: var(--macos-blue); }
/* 使用全局 .label 尺寸与色彩 */
.badge { margin-left: 6px; padding: 0 6px; border-radius: 999px; font-size: 10px; background: color-mix(in oklab, var(--macos-blue) 15%, transparent); color: var(--macos-blue); text-transform: uppercase; }
</style>
