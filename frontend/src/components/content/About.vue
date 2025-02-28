<script setup>
import iconUrl from '@/assets/images/icon.png'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { GetAppVersion } from 'wailsjs/go/preferences/Service.js'
import { onMounted, ref } from 'vue'
import { Project } from "@/consts/global.js";
import usePreferencesStore from 'stores/preferences.js'

const prefStore = usePreferencesStore()
const version = ref('')

const check_update = () => {
    prefStore.checkForUpdate(true)
}

onMounted(() => {
    GetAppVersion().then(({ data }) => {
        version.value = data.version
    })
})

const onOpenSource = () => {
    BrowserOpenURL(Project.Github)
}

const onOpenWebsite = () => {
    BrowserOpenURL(Project.OfficialWebsite)
}

const openX = () => {
    BrowserOpenURL(Project.Twitter)
}

const onCheckUpdateChange = () => {
    prefStore.general.checkUpdate = !prefStore.general.checkUpdate
    prefStore.savePreferences()
}
</script>

<template>
    <div class="card rounded-none bg-base-200">
        <div class="card-body space-y-4 pt-2 px-4">
            <!-- About -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('settings.about') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>
                <div class="flex items-center justify-between p-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-4">
                        <div class="avatar">
                            <img :src="iconUrl" alt="logo" class="about-logo" />
                        </div>
                        <div class="space-y-2">
                            <h1 class="text-2xl font-semibold text-base-content">{{ $t('dialogue.app_name') }}</h1>
                            <p class="mb-2 text-base-content">{{ $t('dialogue.app_description') }}</p>
                            <div class="badge badge-primary badge-outline">v{{ version }}</div>
                        </div>
                    </div>
                    <button class="btn btn-sm border-1 border-base-300 font-normal text-base-content"
                        @click="check_update">
                        {{ $t('menu.check_update') }}
                    </button>
                </div>

                <li class="divider-thin"></li>

                <!-- Update Trigger -->
                <div class="form-control p-2 pl-4 rounded-lg bg-base-100">
                    <label class="label cursor-pointer text-base-content">
                        <span class="text-base-content">{{ $t('dialogue.about.turn_off_update_checking') }}</span>
                        <input type="checkbox" class="toggle toggle-sm toggle-primary"
                            :checked="!prefStore.general.checkUpdate" @change="onCheckUpdateChange" />
                    </label>
                </div>
            </ul>

            <!-- Links Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ri-global-line" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('dialogue.about.official_website') }}</h2>
                    </div>
                    <button @click="onOpenWebsite"
                        class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                            $t('dialogue.about.website') }}</button>
                </div>
                <li class="divider-thin"></li>
                <!-- Source Code-->
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="co-github" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('dialogue.about.github') }}</h2>
                    </div>
                    <button @click="onOpenSource"
                        class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                            $t('dialogue.about.website') }}</button>
                </div>
            </ul>

            <!-- Social Accounts Menu -->
            <ul class="menu p-2 rounded-lg border-2 border-base-300 bg-base-100">
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <h2 class="font-semibold text-base-content">{{ $t('dialogue.about.social_accounts') }}</h2>
                    </div>
                </div>
                <li class="divider-thin"></li>
                <div class="flex items-center justify-between p-2 pl-4 rounded-lg bg-base-100">
                    <div class="flex items-center gap-2">
                        <v-icon name="ci-x" class="h-4 w-4 text-base-content" />
                        <h2 class="text-base-content">{{ $t('dialogue.about.x') }}</h2>
                    </div>
                    <button @click="openX" class="btn btn-sm border-1 border-base-300 font-normal text-base-content">{{
                        $t('dialogue.about.website') }}</button>
                </div>
            </ul>

            <!-- Copyright Footer -->
            <div class="mt-4 pt-4 border-base-300 text-center text-sm copyright-text">
                Copyright &copy; 2025 Dreamapp.cc All rights reserved
            </div>
        </div>
    </div>
</template>

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

.copyright-text {
    color: rgba(115, 115, 115, 0.8);
    text-shadow: 0 1px 1px rgba(255, 255, 255, 0.4);
    font-weight: 300;
    letter-spacing: 0.5px;
}
</style>
