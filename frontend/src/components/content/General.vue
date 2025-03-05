<template>
    <div class="card rounded-none bg-base-200 h-full overflow-hidden">
        <div class="card-body space-y-4 pt-2 px-4 overflow-y-auto">
            <!-- General Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.name') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>
                <!-- Theme -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-moon-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.theme') }}</h2>
                    </div>
                    <div class="join w-[9.5rem]">
                        <button v-for="option in prefStore.themeOption"
                                :key="option.value"
                                class="btn btn-sm join-item min-w-[3rem] border-base-300"
                                :class="{'bg-primary text-white': prefStore.general.theme === option.value,
                                        'text-base-content hover:text-base-content': prefStore.general.theme !== option.value}"
                                @click="prefStore.general.theme = option.value; prefStore.savePreferences()">
                            <v-icon v-if="option.value === 'light'" name="ri-sun-line" class="h-4 w-4" />
                            <v-icon v-else-if="option.value === 'dark'" name="ri-moon-line" class="h-4 w-4" />
                            <v-icon v-else name="md-hdrauto" class="h-4 w-4" />
                        </button>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Language -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="co-language" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.language') }}</h2>
                    </div>
                    <select class="select select-sm select-bordered w-[9.5rem] text-right"
                            v-model="prefStore.general.language"
                            @change="onLanguageChange">
                        <option v-for="option in prefStore.langOption" 
                                :key="option.value" 
                                :value="option.value">
                            {{ option.label }}
                        </option>
                    </select>
                </div>
                <li class="divider-thin"></li>

                <!-- Font -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="co-font" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.font') }}</h2>
                    </div>
                    <select class="select select-sm select-bordered w-[9.5rem] text-right"
                            v-model="prefStore.general.font"
                            @change="prefStore.savePreferences()"
                            :title="$t('settings.general.font_tip')">
                        <option v-for="option in prefStore.fontOption" 
                                :key="option.value" 
                                :value="option.value">
                            {{ option.label }}
                        </option>
                    </select>
                </div>
                <li class="divider-thin"></li>

                <!-- Font Size -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-font-size" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.font_size') }}</h2>
                    </div>
                    <div class="join w-[9.5rem]">
                        <button class="btn btn-sm join-item w-8" 
                                @click="prefStore.general.fontSize = Math.max(8, prefStore.general.fontSize - 1); prefStore.savePreferences()">
                            -
                        </button>
                        <input type="text" 
                               class="input input-sm input-bordered join-item w-24 text-center [appearance:textfield] [&::-webkit-outer-spin-button]:appearance-none [&::-webkit-inner-spin-button]:appearance-none"
                               v-model="prefStore.general.fontSize"
                               @change="prefStore.savePreferences()"
                               pattern="[0-9]*"
                               inputmode="numeric" />
                        <button class="btn btn-sm join-item w-8"
                                @click="prefStore.general.fontSize = Math.min(32, prefStore.general.fontSize + 1); prefStore.savePreferences()">
                            +
                        </button>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Scan Size -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-font-size" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.scan_size') }}</h2>
                    </div>
                    <input type="number" 
                           class="input input-sm input-bordered w-[9.5rem] text-right"
                           v-model="prefStore.general.scanSize"
                           @change="prefStore.savePreferences()"
                           :title="$t('settings.general.scan_size_tip')"
                           min="100"
                           max="10000"
                           step="100" />
                </div>
                <li class="divider-thin"></li>

                <!-- Proxy-->
                <div class="flex flex-col gap-2 p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center justify-between">
                        <div class="flex items-center gap-2">
                            <v-icon name="ri-global-line" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.general.proxy') }}</h2>
                        </div>
                        <input type="checkbox" 
                               class="toggle toggle-sm toggle-primary"
                               v-model="prefStore.proxy.enabled"
                               @change="onProxyChange" />
                    </div>
                    <template v-if="prefStore.proxy.enabled">
                        <li class="divider-thin"></li>
                        <div class="flex items-center">
                            <div class="flex items-center gap-2 w-32">
                                <v-icon name="ri-settings-3-line" class="h-4 w-4 text-base-content" />
                                <span class="text-sm text-base-content">{{ $t('settings.general.proxy_address') }}</span>
                            </div>
                            <div class="flex items-center gap-2 justify-end flex-1">
                                <select class="select select-sm select-bordered w-[9.5rem] text-right"
                                        v-model="prefStore.proxy.protocal"
                                        @change="onProxyChange">
                                    <option v-for="option in prefStore.protocalOption" 
                                            :key="option.value" 
                                            :value="option.value">
                                        {{ $t(option.label) }}
                                    </option>
                                </select>
                                <input type="text" 
                                       class="input input-sm input-bordered w-[9.5rem] text-right" 
                                       v-model="prefStore.proxy.addr"
                                       placeholder="127.0.0.1"
                                       @change="onProxyChange" />
                                <input type="text" 
                                       class="input input-sm input-bordered w-[9.5rem] text-right" 
                                       v-model="prefStore.proxy.port"
                                       placeholder="1080"
                                       @change="onProxyChange" />
                            </div>
                        </div>
                    </template>
                </div>
            </ul>

            <!-- Download Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.general.download') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>

                <!-- Directory -->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('settings.general.download_directory') }}</h2>
                    </div>
                    <div class="join items-center">
                        <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2" 
                              :class="{'text-base-content/40': !prefStore.download?.dir}"
                              :title="prefStore.download?.dir">
                            {{ prefStore.download?.dir }}
                        </span>
                        <button class="btn btn-sm btn-ghost btn-square" 
                                @click="onSelectDownloadDir">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                        </button>
                    </div>
                </div>
            </ul>
        </div>
    </div>
</template>

<script setup>
import { OpenDirectoryDialog } from 'wailsjs/go/systems/Service'
import { computed } from 'vue'
import usePreferencesStore from '@/stores/preferences.js'
import { useI18n } from 'vue-i18n'

const prefStore = usePreferencesStore()
const { t, locale } = useI18n()

const onLanguageChange = async () => {
    await prefStore.savePreferences()
    locale.value = prefStore.currentLanguage
}

const onProxyChange = async () => {
    await prefStore.savePreferencesAndSetProxy()
}

const onSelectDownloadDir = async () => {
    const { success, data, msg } = await OpenDirectoryDialog(downloadDir.value)
    if (success && data?.path && data.path.trim() !== '') {
        prefStore.updateDownloadDir(data.path)
        await prefStore.savePreferences()
    } else if (msg) {
        $message.error(msg)
    }
}

const downloadDir = computed(() => {
    return prefStore.download?.dir || ""
})
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
