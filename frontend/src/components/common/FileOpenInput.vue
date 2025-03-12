<script setup>
import { SelectFile } from 'wailsjs/go/systems/Service'
import { get, isEmpty } from 'lodash'

const props = defineProps({
    modelValue: String,  
    placeholder: String,
    disabled: Boolean,
    ext: String,
})

const emit = defineEmits(['update:modelValue'])  

const onInput = (val) => {
    emit('update:modelValue', val) 
}

const onClear = () => {
    emit('update:modelValue', '') 
}

const handleSelectFile = async () => {
    const { success, data } = await SelectFile('', isEmpty(props.ext) ? null : [props.ext])
    if (success) {
        const path = get(data, 'path', '')
        emit('update:modelValue', path) 
    }
}
</script>

<template>
    <div class="join w-full">
      <input 
        type="text" 
        class="input input-bordered join-item w-full" 
        :disabled="props.disabled"
        :placeholder="placeholder"
        :title="modelValue"
        :value="modelValue"
        @input="e => onInput(e.target.value)" />
      <button 
        class="btn join-item" 
        :disabled="props.disabled" 
        @click="handleSelectFile">
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
