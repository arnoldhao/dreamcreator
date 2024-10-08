<template>
    <n-card :bordered="false" embedded>
        <template #header>
            <n-button text>
                <template #icon>
                    <n-icon class="icon-size">
                        <AIIcon />
                    </n-icon>
                </template>
                <n-text strong class="title">{{ $t('ai.page_title') }}</n-text>
            </n-button>
        </template>

        <template #header-extra>
            <n-tooltip trigger="hover">
                <template #trigger>
                    <n-button @click="openLanguageModal" circle size="small" class="refresh-button">
                        <n-icon>
                    <LangIcon />
                </n-icon>
                    </n-button>
                </template>
                {{ $t('ai.manage_language') }}
            </n-tooltip>
        </template>

        <template #default>
            <n-select v-model:value="selectedLangs" filterable :multiple=true :options="TransLangs" />
        </template>

        <template #footer>
            <n-space justify="center">
                <n-button :disabled="!props.allowAction" @click="handleTranslation()">{{ $t('ai.translate') }}</n-button>
            </n-space>
        </template>
    </n-card>

    <div v-if="dataLoaded">
        <LanguageModal v-model:show="showLanguageModal" :langs="transferOptions" :common-langs="commonLangs"
            @update:langs="getTransLangs" />
    </div>
</template>

<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import AIIcon from '@/components/icons/AI.vue'
import LangIcon from '@/components/icons/Lang.vue'
import { GetLanguage } from 'wailsjs/go/languages/Service'
import { useDialog, NButton } from 'naive-ui'
import useSuperTabStore from 'stores/supertab.js';
import { useI18n } from 'vue-i18n'
import LanguageModal from '@/components/modal/LanguageModal.vue'

const { t } = useI18n()
const tabStore = useSuperTabStore()
const dialog = useDialog()
const selectedLangs = ref([])
const TransLangs = ref([])
const showLanguageModal = ref(false)

const props = defineProps({
    originalSubtitleId: {
        type: String,
        required: true,
    },
    originalSubtitleTitle: {
        type: String,
        required: true,
    },
    originalSubtitleLang: {
        type: String,
        required: true,
    },
    allowAction: {
        type: Boolean,
        default: false,
        required: true,
    },
})

const openLanguageModal = () => {
    showLanguageModal.value = true
}

function showConfirmDialog() {
    if (props.originalSubtitleLang.toLowerCase() != 'original') {
        dialog.warning({
            title: t('ai.confirm_operation'),
            content: t('ai.confirm_operation_content'),
            positiveText: t('common.confirm'),
            negativeText: t('common.cancel'),
            onPositiveClick: () => {
                proceedWithTranslation();
            },
            onNegativeClick: () => {
                $message.info(t('ai.operation_cancelled'));
            }
        });
        return true;
    }
    return false;
}

function handleTranslation() {
    if (!showConfirmDialog()) {
        proceedWithTranslation();
    }
}

const proceedWithTranslation = () => {
    if (props.originalSubtitleId === '' || props.originalSubtitleTitle === '') {
        $message.error(t('ai.please_select_subtitle'))
        return
    }
    if (selectedLangs.value.length === 0) {
        $message.error(t('ai.please_select_translation_language'))
        return
    }
    selectedLangs.value.forEach(lang => {
        tabStore.translateSubtitle(props.originalSubtitleId, props.originalSubtitleTitle, lang)
    });
}

async function getTransLangs() {
    const { data, success, msg } = await GetLanguage()
    if (success) {
        try {
            const parsedData = JSON.parse(data)
            if (parsedData && Array.isArray(parsedData.langs)) {
                TransLangs.value = parsedData.langs.map(lang => {
                    return {
                        ...lang,
                        children: Array.isArray(lang.children) ? lang.children : []
                    }
                })
            } else {
                TransLangs.value = []
            }
        } catch (error) {
            TransLangs.value = []
        }
    } else {
        TransLangs.value = []
    }
    await nextTick()
}

const dataLoaded = ref(false)
const commonLangs = computed(() => {
    const commonTarget = TransLangs.value.find(target => target.key === 'common')
    if (!commonTarget || !Array.isArray(commonTarget.children)) {
        return []
    }

    return commonTarget.children.map(lang => lang.value)
})

const transferOptions = computed(() => {
    return TransLangs.value.flatMap(target =>
        target.children.map(lang => ({
            label: lang.label,
            value: lang.value,
            isCommon: target.key === 'common'
        }))
    )
})

onMounted(async () => {
    await getTransLangs()
    dataLoaded.value = true
})
</script>

<style scoped>
.icon-size {
    font-size: 20px;
}

.title {
    font-weight: bold;
    font-size: 1.1em;
    font-style: italic;
}
</style>