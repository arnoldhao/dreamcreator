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
    <n-input-group>
        <n-input
            :disabled="props.disabled"
            :placeholder="placeholder"
            :title="modelValue"
            :value="modelValue"
            clearable
            @clear="onClear"
            @input="onInput" />
        <n-button :disabled="props.disabled" :focusable="false" @click="handleSelectFile">...</n-button>
    </n-input-group>
</template>

<style lang="scss" scoped></style>
