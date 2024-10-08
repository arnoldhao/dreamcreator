<template>
    <n-list :bordered="true" embedded>
        <template #header>
            <n-button text>
                <template #icon>
                    <n-icon class="icon-size">
                        <SubtitleIcon />
                    </n-icon>
                </template>
                <n-text strong class="page-title">{{ $t('task.current_task') }}</n-text>
            </n-button>
        </template>

        <!-- file info area -->
        <n-list-item>
            <n-thing>
                <n-text strong class="ellipsis">{{ tabTitle }}</n-text>

                <template #footer>
                    <n-space size="small" style="margin-top: 4px">
                        <n-tag :bordered="true" type="success" size="small">
                            {{ language }}
                        </n-tag>
                    </n-space>
                </template>
            </n-thing>
            <template #suffix>
                <n-tooltip trigger="hover" :content="buttonTooltip">
                    <template #trigger>
                        <n-button circle size="small" class="refresh-button"
                            @click="showEditTitleDialog(tabId, tabTitle)">
                            <n-icon>
                                <EditIcon />
                            </n-icon>
                        </n-button>
                    </template>
                    {{ $t('task.rename') }}
                </n-tooltip>
            </template>
        </n-list-item>

        <!-- progress area -->
        <n-list-item v-if="isStream">
            <template #suffix>
                <!-- action -->
                <n-tooltip trigger="hover" :content="buttonTooltip">
                    <template #trigger>
                        <n-button v-if="isPending" :disabled="!isTranslating" quaternary circle type="info">
                            <template #icon>
                                <n-icon>
                                    <PendingIcon />
                                </n-icon>
                            </template>
                        </n-button>
                        <n-button v-else-if="isTranslating" @click="cancelConfirmDialog(tabId)"
                            :disabled="!isTranslating" quaternary circle type="warning">
                            <template #icon>
                                <CancelIcon />
                            </template>
                        </n-button>
                        <n-button v-else-if="isCanceled" quaternary circle type="error">
                            <template #icon>
                                <CanceledIcon />
                            </template>
                        </n-button>
                        <n-button v-else-if="isCompleted" quaternary circle type="success">
                            <template #icon>
                                <CompleteIcon />
                            </template>
                        </n-button>
                        <n-button v-else quaternary circle type="error">
                            <template #icon>
                                <UnknownIcon />
                            </template>
                        </n-button>
                    </template>
                </n-tooltip>
            </template>
            <!-- progress -->
            <n-thing>
                <n-progress type="line" :percentage="Number(translationProgress)" indicator-placement="inside"
                    :processing="isTranslating" :status="hasError ? 'error' : (isCompleted ? 'success' : 'default')">
                </n-progress>
            </n-thing>
        </n-list-item>

        <!-- description -->
        <n-list-item v-if="isStream">
            <n-flex justify="center">
                <n-text v-if="actionDescription" :type="hasError ? 'error' : 'default'">
                    {{ actionDescription }}
                </n-text>
                <n-text v-else-if="isTranslating">
                    {{ $t('task.translating') }}
                </n-text>
                <n-text v-else-if="isPending">
                    {{ $t('task.pending') }}
                </n-text>
                <n-text v-else>
                    {{ $t('task.waiting') }}
                </n-text>
            </n-flex>
        </n-list-item>

        <!-- action area -->
        <template #footer>
            <n-flex justify="center">
                <n-tooltip trigger="hover">
                    <template #trigger>
                        <n-button @click="switchToTab('blank')">
                            <template #icon>
                                <n-icon class="icon-size">
                                    <FileAddIcon />
                                </n-icon>
                            </template>
                        </n-button>
                    </template>
                    {{ $t('task.open_new_file') }}
                </n-tooltip>
                <n-tooltip trigger="hover">
                    <template #trigger>
                        <n-button :disabled="!allowAction">
                            <template #icon>
                                <n-icon class="icon-size">
                                    <AIIcon />
                                </n-icon>
                            </template>
                        </n-button>
                    </template>
                    {{ $t('task.format_file') }}
                </n-tooltip>
                <n-tooltip trigger="hover">
                    <template #trigger>
                        <n-button :disabled="!allowAction" @click="handleDownload(tabId, tabTitle)">
                            <template #icon>
                                <n-icon class="icon-size">
                                    <DownloadIcon />
                                </n-icon>
                            </template>
                        </n-button>
                    </template>
                    {{ $t('task.download_subtitles') }}
                </n-tooltip>
            </n-flex>
        </template>
    </n-list>
</template>

<script setup>
import { computed } from 'vue'
import AIIcon from '@/components/icons/AI.vue'
import EditIcon from '@/components/icons/Edit.vue'
import CancelIcon from '@/components/icons/Cancel.vue'
import CanceledIcon from '@/components/icons/Canceled.vue'
import CompleteIcon from '@/components/icons/Complete.vue'
import DownloadIcon from '@/components/icons/Download.vue'
import FileAddIcon from '@/components/icons/FileAdd.vue'
import PendingIcon from '@/components/icons/Pending.vue'
import UnknownIcon from '@/components/icons/Unknown.vue'
import SubtitleIcon from '@/components/icons/Subtitle.vue'
import { SaveSubtitles } from 'wailsjs/go/subtitles/Service'
import { NTooltip } from 'naive-ui'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const props = defineProps({
    isBlank: {
        type: Boolean,
        required: true
    },
    tabId: {
        type: String,
        required: true
    },
    tabTitle: {
        type: String,
        required: true
    },
    language: {
        type: String,
        required: true
    },
    isStream: {
        type: Boolean,
        required: true
    },
    translationProgress: {
        type: Number,
        required: true,
        default: 0.00
    },
    isTranslating: {
        type: Boolean,
        required: true
    },
    isCanceled: {
        type: Boolean,
        required: true
    },
    hasError: {
        type: Boolean,
        required: true
    },
    isCompleted: {
        type: Boolean,
        required: true
    },
    actionDescription: {
        type: String,
        default: ''
    },
    isPending: {
        type: Boolean,
        required: true
    },
    allowAction: {
        type: Boolean,
        required: true
    }
})

const emit = defineEmits(['switchTab', 'showEditTitleDialog', 'cancelConfirmDialog'])

const switchToTab = (id) => {
    emit('switchTab', id)
}

const showEditTitleDialog = (id, title) => {
    emit('showEditTitleDialog', id, title)
}

const cancelConfirmDialog = (id) => {
    emit('cancelConfirmDialog', id)
}

async function handleDownload(key, title) {
    const { data, success, msg } = await SaveSubtitles(key, title, ['srt', 'vtt', 'ass']);
    if (success) {
        $message.success(t('task.file_saved_to', { path: data?.path || '' }));
    } else {
        if (data.canceled) {
            $message.info(t('task.operation_cancelled'));
        } else {
            $message.error(t('task.save_file_failed') + msg);
        }
    }
}

const buttonTooltip = computed(() => {
    if (props.isPending) return 'Pending'
    if (props.isTranslating) return 'Cancel'
    if (props.isCanceled) return 'Canceled'
    if (props.isCompleted) return 'Completed'
    return 'Unknown'
})
</script>

<style scoped>
.icon-size {
    font-size: 20px;
}

.page-title {
    font-weight: bold;
    font-size: 1.1em;
    font-style: italic;
}
</style>