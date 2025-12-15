<script setup>
import WindowMin from '@/components/window-controls/WindowMin.vue'
import WindowMax from '@/components/window-controls/WindowMax.vue'
import WindowRestore from '@/components/window-controls/WindowRestore.vue'
import WindowClose from '@/components/window-controls/WindowClose.vue'
import { computed } from 'vue'
import { Application, Window } from '@wailsio/runtime'

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

const handleMinimise = () => {
    try { Window.Minimise() } catch {}
}

const handleMaximise = () => {
    try { Window.ToggleMaximise() } catch {}
}

const handleClose = () => {
    try { Application.Quit() } catch {}
}
</script>

<template>
    <div class="flex items-center justify-center">
        <div class="btn-wrapper toolbar-chip" :data-tooltip="$t('menu.minimise')" data-tip-pos="top" @click="handleMinimise">
            <window-min />
        </div>
        <div v-if="maximised" class="btn-wrapper toolbar-chip" :data-tooltip="$t('menu.restore')" data-tip-pos="top" @click="handleMaximise">
            <window-restore />
        </div>
        <div v-else class="btn-wrapper toolbar-chip" :data-tooltip="$t('menu.maximise')" data-tip-pos="top" @click="handleMaximise">
            <window-max />
        </div>
        <div class="btn-wrapper btn-wrapper-close toolbar-chip" :data-tooltip="$t('menu.close')" data-tip-pos="top" @click="handleClose">
            <window-close />
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
        background: var(--macos-gray-hover);
    }

    &:active {
        background: var(--macos-gray-active);
    }

    &.btn-wrapper-close {
        &:hover {
            background: var(--macos-danger-text);
            color: #fff;
        }

        &:active {
            background: color-mix(in oklab, var(--macos-danger-text) 85%, black);
            color: #fff;
        }
    }
}
</style>
