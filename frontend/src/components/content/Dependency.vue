<script setup>
import { onMounted, onUnmounted } from 'vue'
import { InstallYTDLP } from 'wailsjs/go/api/DowntasksAPI'
import { GetYTDLPPath, GetFFMPEGPath, GetFFMPEGVersion } from 'wailsjs/go/api/PathsAPI'
import { useDt } from '@/handlers/downtasks'
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n'
import usePreferencesStore from '@/stores/preferences'

const { t } = useI18n()
const { initDt, onInstalling } = useDt()
const prefStore = usePreferencesStore()

const ytdlpStatus = async () => {
    try {
        const response = await GetYTDLPPath()
        if (response.success) {
            const data = JSON.parse(response.data)
            if (data.available) {
                prefStore.dependencies.ytdlp.installed = true
                prefStore.dependencies.ytdlp.path = data.path
                prefStore.dependencies.ytdlp.version = data.version
            } else {
                prefStore.dependencies.ytdlp.installed = false
            }
        } else {
            prefStore.dependencies.ytdlp.installed = false
            $message.warning(response.msg)
        }
    } catch (error) {
        prefStore.dependencies.ytdlp.installed = false
        console.error('Get ytdlp status failed:', error)
    }
}

const installYtdlp = async () => {
    if (prefStore.dependencies.ytdlp.installing) return
    prefStore.dependencies.ytdlp.installing = true
    prefStore.dependencies.ytdlp.installProgress = ''
    
    try {
        const response = await InstallYTDLP()
        if (response.success) {
            await ytdlpStatus()
            $message.success(t('dependency.install_success'))
        } else {
            $message.error(response.msg)
        }
    } catch (error) {
        console.error('Install ytdlp failed:', error)
        $message.error(t('dependency.install_failed'))
    } finally {
        prefStore.dependencies.ytdlp.installing = false
    }
}

const ffmpegStatus = async () => {
    try {
        const response = await GetFFMPEGPath()
        if (response.success) {
            const data = JSON.parse(response.data)
            if (data.available) {
                prefStore.dependencies.ffmpeg.installed = true
                prefStore.dependencies.ffmpeg.path = data.path
                prefStore.dependencies.ffmpeg.version = data.version
            } else {
                prefStore.dependencies.ffmpeg.installed = false
            }
        } else {
            prefStore.dependencies.ffmpeg.installed = false
            $message.warning(response.msg)
        }
    } catch (error) {
        prefStore.dependencies.ffmpeg.installed = false
        $message.error('Get ffmpeg status failed:', error)
    }
}

const ffmpegVersion = async () => {
    try {
        const response = await GetFFMPEGVersion()
        if (response.success) {
            prefStore.dependencies.ffmpeg.version = response.data
        } else {
            $message.warning(response.msg)
        }
    } catch (error) {
        $message.error('Get ffmpeg version failed:', error)
    }
}

const installFFMPEG = async () => {
    $dialog.info({
        title: t('settings.dependency.install_ffmpeg'),
        content: t('settings.dependency.install_ffmpeg_desc')
    })
}

const openDirectory = async (path) => {
    OpenDirectory(path)
}

const showFFMPEGVersion = async () => {
    await ffmpegVersion()
    $dialog.info({
        content: prefStore.dependencies.ffmpeg.version
    })
}

// lifecycle hooks
onMounted(() => {
    ytdlpStatus()
    ffmpegStatus()

    // init WebSocket handler
    initDt()

    // register installing callback
    const unsubscribeInstalling = onInstalling((progress) => {
        if (progress.stage == "installing") {
            prefStore.dependencies.ytdlp.installProgress = progress.percentage.toFixed(2) + '%'
        } else if (progress.stage == "installed") {
            prefStore.dependencies.ytdlp.installProgress = ''
            ytdlpStatus()
        } else {
            prefStore.dependencies.ytdlp.installProgress = ''
            $message.error("unknown stage: " + progress.stage)
        }
    })

    // cleanup function
    onUnmounted(() => {
        unsubscribeInstalling()
    })
})

</script>

<template>
    <div class="card rounded-none bg-base-200">
        <div class="card-body space-y-4 pt-2 px-4">
            <!-- Links Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <h2 class="font-semibold text-base-content">{{ $t('settings.dependency.ytdlp') }}</h2>
                    <div v-if="prefStore.dependencies.ytdlp.installed">
                        <button @click="ytdlpStatus()"
                            class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                $t('settings.dependency.refresh') }}</button>
                    </div>
                    <div v-else>
                        <div class="flex items-center gap-2 justify-end">
                            <span v-if="prefStore.dependencies.ytdlp.installProgress" class="text-sm text-base-content/60">
                                {{ prefStore.dependencies.ytdlp.installProgress }}
                            </span>
                            <button @click="installYtdlp()" :disabled="prefStore.dependencies.ytdlp.installing"
                                class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                    $t('settings.dependency.install') }}</button>
                        </div>
                    </div>
                </div>
                <div v-if="prefStore.dependencies.ytdlp.installed">
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.path') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ytdlp.path }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square" @click="openDirectory(prefStore.dependencies.ytdlp.path)">
                                <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-versions" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.version') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ytdlp.version }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square" @click="openDialog(prefStore.dependencies.ytdlp.version)">
                                <v-icon name="si-codereview" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                </div>
            </ul>

            <!-- ffmpeg -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <h2 class="font-semibold text-base-content">{{ $t('settings.dependency.ffmpeg') }}</h2>
                    <div v-if="prefStore.dependencies.ffmpeg.installed">
                        <button @click="ffmpegStatus()"
                            class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                $t('settings.dependency.refresh') }}</button>
                    </div>
                    <div v-else>
                        <div class="flex items-center gap-2 justify-end">
                            <button @click="ffmpegStatus()"
                            class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                $t('settings.dependency.refresh') }}</button>
                            <button @click="installFFMPEG()"
                                class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                                    $t('settings.dependency.install') }}</button>
                        </div>
                    </div>
                </div>
                <div v-if="prefStore.dependencies.ffmpeg.installed">
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.path') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ffmpeg.path }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square" @click="openDirectory(prefStore.dependencies.ffmpeg.path)">
                                <v-icon name="oi-file-directory" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                    <li class="divider-thin"></li>
                    <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                        <div class="flex items-center gap-2">
                            <v-icon name="oi-versions" class="h-4 w-4 text-base-content" />
                            <h2 class="text-base-content">{{ $t('settings.dependency.version') }}</h2>
                        </div>
                        <div class="join items-center">
                            <span class="text-sm text-base-content/60 w-[17rem] text-right truncate mr-2">
                                {{ prefStore.dependencies.ffmpeg.version }}
                            </span>
                            <button class="btn btn-sm btn-ghost btn-square" @click="showFFMPEGVersion()">
                                <v-icon name="si-codereview" class="h-4 w-4 text-base-content/60" />
                            </button>
                        </div>
                    </div>
                </div>
            </ul>
        </div>
    </div>
</template>

<style lang="scss" scoped>
.card {
    @apply shadow-lg;
}

.menu li a {
    @apply hover:bg-base-200;
}
</style>
