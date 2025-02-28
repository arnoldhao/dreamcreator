<template>
    <div class="card rounded-none bg-base-200 h-full overflow-hidden">
        <div class="card-body space-y-4 pt-2 px-4 overflow-y-auto">
            <!-- Editor -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.editor.name') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Font -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="co-font" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.font') }}</h2>
                    </div>
                    <select class="select select-sm select-bordered w-[9.5rem] text-right"
                            v-model="prefStore.editor.font"
                            @change="prefStore.savePreferences()">
                        <option v-for="option in prefStore.fontOption" 
                                :key="option" 
                                :value="option">
                            {{ option.value }}
                        </option>
                    </select>
                </div>
                <li class="divider-thin"></li>

                <!-- Font Size -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-font-size" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.font_size') }}</h2>
                    </div>
                    <div class="join w-[9.5rem]">
                        <button class="btn btn-sm join-item w-8" 
                                @click="prefStore.editor.fontSize = Math.max(8, prefStore.editor.fontSize - 1); prefStore.savePreferences()">
                            -
                        </button>
                        <input type="text" 
                               class="input input-sm input-bordered join-item w-24 text-center [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                               v-model="prefStore.editor.fontSize"
                               @change="prefStore.savePreferences()"
                               pattern="[0-9]*"
                               inputmode="numeric" />
                        <button class="btn btn-sm join-item w-8"
                                @click="prefStore.editor.fontSize = Math.min(32, prefStore.editor.fontSize + 1); prefStore.savePreferences()">
                            +
                        </button>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Show Line Numbers -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="bi-ui-checks" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.show_linenum') }}</h2>
                    </div>
                    <input type="checkbox" 
                           class="toggle toggle-primary"
                           v-model="prefStore.editor.showLineNumbers"
                           @change="prefStore.savePreferences()" />
                </div>
                <li class="divider-thin"></li>

                <!-- Show Code Folding -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="bi-ui-checks" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.show_folding') }}</h2>
                    </div>
                    <input type="checkbox" 
                           class="toggle toggle-primary"
                           v-model="prefStore.editor.showFolding"
                           @change="prefStore.savePreferences()" />
                </div>
                <li class="divider-thin"></li>

                <!-- Allow Drop Text -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="bi-ui-checks" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.drop_text') }}</h2>
                    </div>
                    <input type="checkbox" 
                           class="toggle toggle-primary"
                           v-model="prefStore.editor.allowDropText"
                           @change="prefStore.savePreferences()" />
                </div>
                <li class="divider-thin"></li>

                <!-- Support Links -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="bi-ui-checks" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.editor.links') }}</h2>
                    </div>
                    <input type="checkbox" 
                           class="toggle toggle-primary"
                           v-model="prefStore.editor.links"
                           @change="prefStore.savePreferences()" />
                </div>
            </ul>
        </div>
    </div>
</template>

<script setup>
import usePreferencesStore from '@/stores/preferences.js'
import { useI18n } from 'vue-i18n'

const prefStore = usePreferencesStore()
const { t, locale } = useI18n()

</script>

<style lang="scss" scoped>
.about-app-title {
    font-weight: bold;
    font-size: 18px;
    margin: 5px;
}

.about-link {
    cursor: pointer;

    &:hover {
        text-decoration: underline;
    }
}

.about-logo {
    width: 72px;
    height: 72px;
}

.card {
    @apply shadow-lg;
}

.menu li a {
    @apply hover:bg-base-200;
}

.about-copyright {
    font-size: 12px;
}
</style>
