<script setup>
import WindowMin from '@/components/icons/WindowMin.vue'
import WindowMax from '@/components/icons/WindowMax.vue'
import WindowRestore from '@/components/icons/WindowRestore.vue'
import WindowClose from '@/components/icons/WindowClose.vue'
import { computed } from 'vue'
import { Quit, WindowMinimise, WindowToggleMaximise } from 'wailsjs/runtime/runtime.js'

const props = defineProps({
    size: {
        type: Number,
        default: 35,
    },
    maximised: {
        type: Boolean,
    },
})

const buttonSize = computed(() => {
    return props.size + 'px'
})

const handleMinimise = async () => {
    WindowMinimise()
}

const handleMaximise = () => {
    WindowToggleMaximise()
}

const handleClose = () => {
    Quit()
}
</script>

<template>
    <div class="flex items-center justify-center">
        <div class="tooltip tooltip-bottom" :data-tip="$t('menu.minimise')">
            <div class="btn-wrapper" @click="handleMinimise">
                <window-min />
            </div>
        </div>
        <div v-if="maximised" class="tooltip tooltip-bottom" :data-tip="$t('menu.restore')">
            <div class="btn-wrapper" @click="handleMaximise">
                <window-restore />
            </div>
        </div>
        <div v-else class="tooltip tooltip-bottom" :data-tip="$t('menu.maximise')">
            <div class="btn-wrapper" @click="handleMaximise">
                <window-max />
            </div>
        </div>
        <div class="tooltip tooltip-bottom" :data-tip="$t('menu.close')">
            <div class="btn-wrapper btn-wrapper-close" @click="handleClose">
                <window-close />
            </div>
        </div>
    </div>
</template>

<style lang="scss" scoped>
.btn-wrapper {
    width: v-bind('buttonSize');
    height: v-bind('buttonSize');
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    --wails-draggable: none;

    &:hover {
        cursor: pointer;
        @apply bg-base-300;
    }

    &:active {
        @apply bg-base-200;
    }

    &.btn-wrapper-close {
        &:hover {
            @apply bg-error;
        }

        &:active {
            @apply bg-error-content;
        }
    }
}
</style>