<script setup>
import iconUrl from '@/assets/images/icon.png'
import useDialog from 'stores/dialog.js'
import { useThemeVars } from 'naive-ui'
import { BrowserOpenURL } from 'wailsjs/runtime/runtime.js'
import { GetAppVersion } from 'wailsjs/go/preferences/Service.js'
import { onMounted, ref } from 'vue'
import {Project} from "@/consts/global.js";

const themeVars = useThemeVars()
const dialogStore = useDialog()
const version = ref('')

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
</script>

<template>
    <n-modal v-model:show="dialogStore.aboutDialogVisible" :show-icon="false" preset="dialog" transform-origin="center">
        <n-space :size="10" :wrap="false" :wrap-item="false" align="center" vertical>
            <n-avatar :size="120" :src="iconUrl" color="#0000"></n-avatar>
            <div class="about-app-title">{{ $t('dialogue.app_name') }}</div>
            <n-text>{{ version }}</n-text>
            <n-space :size="5" :wrap="false" :wrap-item="false" align="center">
                <n-text class="about-link" @click="onOpenSource">{{ $t('dialogue.about.source') }}</n-text>
                <n-divider vertical />
                <n-text class="about-link" @click="onOpenWebsite">{{ $t('dialogue.about.website') }}</n-text>
            </n-space>
            <div :style="{ color: themeVars.textColor3 }" class="about-copyright">
                Copyright © 2024 Dreamapp.cc All rights reserved
            </div>
        </n-space>
    </n-modal>
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

.about-copyright {
    font-size: 12px;
}
</style>
