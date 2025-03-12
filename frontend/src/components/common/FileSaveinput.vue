<script setup>
import { SaveFile } from 'wailsjs/go/services/systemService.js'
import { get } from 'lodash'

const props = defineProps({
    modelValue: String,
    placeholder: String,
    disabled: Boolean,
    defaultPath: String,
})

const emit = defineEmits(['update:modelValue'])

const onInput = (val) => {
    emit('update:modelValue', val)
}

const onClear = () => {
    emit('update:modelValue', '')
}

const handleSaveFile = async () => {
    const { success, data } = await SaveFile(null, props.defaultPath, ['csv'])
    if (success) {
        const path = get(data, 'path', '')
        emit('update:modelValue', path)
    } else {
        emit('update:modelValue', '')
    }
}
</script>

<template>
    <div class="join w-full">
      <input 
        type="text" 
        class="input input-bordered join-item w-full" 
        :disabled="props.disabled"
        :placeholder="props.placeholder"
        :title="modelValue"
        :value="modelValue"
        @input="e => onInput(e.target.value)" />
      <button 
        class="btn join-item" 
        :disabled="props.disabled" 
        @click="handleSaveFile">
        ...
      </button>
      <button 
        v-if="modelValue" 
        class="btn join-item" 
        :disabled="props.disabled" 
        @click="onClear">
        <i class="ri-close-line"></i>
      </button>
    </div>
  </template>

<style lang="scss" scoped></style>
