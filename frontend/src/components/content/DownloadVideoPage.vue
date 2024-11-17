<template>
    <div class="h-full w-full flex flex-col">
        <!-- Title area-->
        <div class="p-4 pb-0">
            <h2 class="text-2xl font-bold text-base-content/80">{{ t('video_download.page_title') }}</h2>
        </div>

        <!-- URL area -->
        <div class="p-4 pt-2">
            <div class="form-control">
                <div class="flex gap-2">
                    <input type="text" v-model="url" :placeholder="t('video_download.url_placeholder')"
                        class="input input-bordered flex-1" />
                    <button @click="handleGet" class="btn btn-primary" :disabled="isLoading">
                        {{ isLoading ? t('video_download.parsing') : t('video_download.parse') }}
                    </button>
                </div>
            </div>
        </div>

        <!-- Parsing result area -->
        <div class="p-4">
            <div v-if="videoData?.info?.title" class="text-base font-medium mb-4 truncate max-w-2xl"
                :title="videoData.info.title">
                {{ videoData.info.title }}
            </div>
            <div class="space-y-6">
                <!-- Options area -->
                <div class="flex gap-4 items-start">
                    <!-- Format select -->
                    <div class="w-1/5">
                        <label class="label">
                            <span class="label-text font-medium">{{ t('video_download.format') }}</span>
                        </label>
                        <select v-model="selectedFormat" class="select select-bordered w-full">
                            <option disabled value="">{{ t('video_download.select_format') }}</option>
                            <option v-for="format in uniqueFormats" :key="format" :value="format">
                                {{ format }}
                            </option>
                        </select>
                    </div>

                    <!-- Quality select -->
                    <div class="w-2/5">
                        <label class="label">
                            <span class="label-text font-medium">{{ t('video_download.quality') }}</span>
                        </label>
                        <select v-model="selectedQuality" class="select select-bordered w-full">
                            <option disabled value="">{{ t('video_download.select_quality') }}</option>
                            <option v-for="quality in filteredQualities" :key="quality.id" :value="quality">
                                {{ formatQuality(quality.quality) }} ({{ (quality.size / 1024 / 1024).toFixed(2) }}MB)
                            </option>
                        </select>
                    </div>

                    <!-- Caption select -->
                    <div class="w-1/5">
                        <label class="label">
                            <span class="label-text font-medium">{{ t('video_download.caption') }}</span>
                        </label>
                        <select v-model="selectedCaption" class="select select-bordered w-full">
                            <option disabled value="">{{ t('video_download.select_caption') }}</option>
                            <option v-for="caption in videoData?.captions || []" :key="caption.id" :value="caption">
                                {{ caption.language }}
                            </option>
                            <option v-if="!videoData?.captions?.length" disabled value="">{{
                                t('video_download.no_caption') }}</option>
                        </select>
                    </div>

                    <!-- Download button -->
                    <div class="w-1/5">
                        <label class="label opacity-0">
                            <span class="label-text">{{ t('video_download.download') }}</span>
                        </label>
                        <button @click="download()" class="btn btn-primary w-full"
                            :disabled="!selectedQuality || isDownloading">
                            {{ t('video_download.download') }}
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Download task list -->
        <div class="flex-1 card bg-base-100 p-4 mt-4 flex flex-col min-h-0">
            <div class="shrink-0 flex justify-between items-center mb-4">
                <h3 class="text-xl font-bold">{{ t('video_download.tasks') }}</h3>
                <button @click="refreshInstantData" class="btn btn-ghost btn-sm">
                    <i class="fas fa-sync-alt mr-2"></i>{{ t('video_download.refresh') }}
                </button>
            </div>

            <div class="flex-1 overflow-auto min-h-0">
                <table class="table table-pin-rows w-full">
                    <thead class="sticky top-0 bg-base-100 z-10">
                        <tr>
                            <th>
                                <div class="flex items-center justify-center gap-2">
                                    <i class="fas fa-folder"></i>
                                    <span>{{ t('video_download.source') }}</span>
                                </div>
                            </th>
                            <th class="text-center">{{ t('video_download.title') }}</th>
                            <th class="text-center">{{ t('video_download.format') }}</th>
                            <th class="text-center">{{ t('video_download.size') }}</th>
                            <th class="text-center">{{ t('video_download.progress') }}</th>
                            <th class="text-center">{{ t('video_download.download') }}</th>
                            <th class="text-center">{{ t('video_download.saved_path') }}</th>
                        </tr>
                    </thead>
                    <tbody>
                        <tr v-for="item in instantData" :key="item.id">
                            <td>
                                <div class="flex justify-center">
                                    <YoutubeIcon v-if="getSiteIcon(item.source) === 'YoutubeIcon'" class="w-4 h-4" />
                                    <BilibiliIcon v-else-if="getSiteIcon(item.source) === 'BilibiliIcon'"
                                        class="w-4 h-4" />
                                    <i v-else :class="getSiteIcon(item.source)" :title="getSiteName(item.source)"></i>
                                </div>
                            </td>
                            <td class="max-w-xs">
                                <div class="truncate">
                                    <button class="hover:text-primary w-full text-left" @click="showDetailModal(item)">
                                        {{ item.title }}
                                    </button>
                                </div>
                            </td>
                            <td class="text-center">{{ item.format }}</td>
                            <td class="text-center">{{ formatFileSize(item.size) }}</td>
                            <td class="text-center">
                                <div class="flex items-center gap-2">
                                    <div class="flex-1">
                                        <template v-if="downloadFailed(item.total, item.finished, item.status)">
                                            <div class="flex items-center justify-center flex-1">
                                                <button class="btn btn-xs btn-ghost text-error gap-2"
                                                    @click="showErrorModal(item.error)">
                                                    {{ t('video_download.download_failed') }}
                                                    <i class="fas fa-info-circle"></i>
                                                </button>
                                            </div>
                                        </template>
                                        <template v-else>
                                            <div class="flex items-center gap-2">
                                                <progress class="progress progress-primary flex-1"
                                                    :value="Number(item.progress).toFixed(2)" max="100">
                                                </progress>
                                                <span class="text-sm min-w-[3.5rem]">
                                                    {{ `${Number(item.progress).toFixed(2)}%` }}
                                                </span>
                                            </div>
                                        </template>
                                    </div>
                                </div>
                            </td>
                            <td class="text-center">
                                {{ item.status === 'Downloading' ? item.speed : item.status }}
                            </td>
                            <td class="text-center">
                                <button class="btn btn-ghost btn-sm !bg-transparent flex justify-center items-center"
                                    @click="openFolder(item.savedPath)">
                                    <div class="flex justify-center items-center">
                                        <OpenDirectoryIcon class="w-4 h-4" />
                                    </div>
                                </button>
                            </td>
                        </tr>
                        <tr v-if="!instantData || instantData.length === 0">
                            <td colspan="5" class="text-center">{{ t('video_download.no_tasks') }}</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>

        <!-- Task error detail modal -->
        <dialog ref="errorModal" class="modal">
            <div class="modal-box max-w-fit min-w-[20rem] max-h-[80vh]">
                <h3 class="font-bold text-lg">{{ t('video_download.error_detail') }}</h3>
                <div class="py-4 overflow-y-auto">
                    <p class="whitespace-pre-wrap break-words select-text">{{ currentError }}</p>
                </div>
                <div class="modal-action flex gap-2">
                    <button class="btn btn-primary gap-1">
                        <i class="fas fa-copy"></i>{{ t('video_download.copy') }}
                    </button>
                    <form method="dialog">
                        <button class="btn">{{ t('video_download.close') }}</button>
                    </form>
                </div>
            </div>
        </dialog>

        <!-- Task detail modal -->
        <dialog ref="detailModal" class="modal">
            <div class="modal-box max-w-fit min-w-[30rem] max-h-[80vh]">
                <h3 class="font-bold text-lg mb-4">{{ t('video_download.detail_info') }}</h3>
                <div class="overflow-y-auto space-y-3">
                    <div class="grid grid-cols-[auto,1fr] gap-x-4 gap-y-2">
                        <div class="font-semibold">{{ t('video_download.title') }}：</div>
                        <div class="whitespace-pre-wrap break-words select-text">{{ currentItem.title }}</div>

                        <div class="font-semibold">{{ t('video_download.url') }}：</div>
                        <div class="break-all select-text">{{ currentItem.url }}</div>

                        <div class="font-semibold">{{ t('video_download.format') }}：</div>
                        <div>{{ currentItem.format }}</div>

                        <div class="font-semibold">{{ t('video_download.quality') }}：</div>
                        <div>{{ currentItem.quality }}</div>

                        <div class="font-semibold">{{ t('video_download.status') }}：</div>
                        <div>{{ currentItem.status }}</div>

                        <div class="font-semibold">{{ t('video_download.progress') }}：</div>
                        <div>{{ Number(currentItem.progress).toFixed(2) }}%</div>

                        <div class="font-semibold">{{ t('video_download.finished') }}：</div>
                        <div>{{ currentItem.finished }}</div>

                        <div class="font-semibold">{{ t('video_download.total') }}：</div>
                        <div>{{ currentItem.total }}</div>

                        <div class="font-semibold">{{ t('video_download.size') }}：</div>
                        <div>{{ formatFileSize(currentItem.size) }}</div>

                        <div class="font-semibold">{{ t('video_download.saved_path') }}：</div>
                        <div>{{ currentItem.savedPath }}</div>

                        <template v-if="downloadFailed(currentItem.total, currentItem.finished, currentItem.status)">
                            <div class="font-semibold">{{ t('video_download.error_info') }}：</div>
                            <div class="text-error">{{ currentItem.error }}</div>
                        </template>
                    </div>
                </div>
                <div class="modal-action flex gap-2">
                    <button class="btn btn-primary gap-1" @click="copyItemInfo">
                        <i class="fas fa-copy"></i>{{ t('video_download.copy_all') }}
                    </button>
                    <form method="dialog">
                        <button class="btn">{{ t('video_download.close') }}</button>
                    </form>
                </div>
            </div>
        </dialog>
    </div>
</template>

<script setup>
import { Get, GetFFMPEGVersion, CheckTask } from 'wailsjs/go/downloads/WorkQueue'
import { ref, watch, computed, onMounted } from 'vue'
import emitter from '@/utils/eventBus'
import { EMITTER_EVENTS } from '@/consts/emitter'
import useDownloadStore from '@/stores/download'
import { storeToRefs } from "pinia";
import YoutubeIcon from '@/components/icons/Youtube.vue';
import BilibiliIcon from '@/components/icons/Bilibili.vue';
import OpenDirectoryIcon from '@/components/icons/OpenDirectory.vue';
import { OpenDirectory } from 'wailsjs/go/systems/Service'
import { useI18n } from 'vue-i18n';
const downloadStore = useDownloadStore()
const { instantData } = storeToRefs(downloadStore)
const { t } = useI18n();

const url = ref('')
const isLoading = ref(false)
const videoData = ref([])
const selectedCaption = ref({})
const selectedFormat = ref('')
const selectedQuality = ref(null)

const errorModal = ref(null)
const currentError = ref('')
const detailModal = ref(null)
const currentItem = ref({})

// Calculate all unique formats
const uniqueFormats = computed(() => {
    if (!videoData.value.qualities) return []
    return [...new Set(videoData.value.qualities.map(q => q.format))]
})

// Filter qualities based on selected format
const filteredQualities = computed(() => {
    if (!selectedFormat.value || !videoData.value.qualities) return []
    return videoData.value.qualities.filter(q => q.format === selectedFormat.value)
})

// Reset quality selection when format changes
watch(selectedFormat, () => {
    selectedQuality.value = null
})

// Reset download status when quality changes
watch(selectedQuality, () => {
    isDownloading.value = false
})

async function handleGet() {
    isLoading.value = true
    try {
        const { data, success, msg } = await Get(url.value)
        if (!success) {
            $message.error(msg)
            return
        }

        // Parse video data
        const parsedData = JSON.parse(data)[0]
        if (!parsedData) {
            $message.error(t('video_download.no_data'))
            return
        }

        // Extract basic information
        const videoInfo = {
            id: parsedData.id || '',
            source: parsedData.source || '',
            title: parsedData.title || '',
            url: parsedData.url || '',
            site: parsedData.site || ''
        }

        // Extract all available video qualities
        const qualities = []
        if (parsedData.streams && typeof parsedData.streams === 'object') {
            Object.entries(parsedData.streams).forEach(([id, stream]) => {
                if (stream && stream.parts && stream.parts[0]) {
                    qualities.push({
                        id,
                        quality: stream.quality || '',
                        size: stream.size || 0,
                        format: stream.ext || '',
                        parts: stream.parts.length || 0,
                    })
                }
            })

            // Sort qualities by quality
            qualities.sort((a, b) => {
                const getQualityNumber = (quality) =>
                    parseInt(quality.match(/\d+/)?.[0] || '0')
                return getQualityNumber(b.quality) - getQualityNumber(a.quality)
            })
        }

        const captions = []
        if (parsedData.captions && typeof parsedData.captions === 'object') {
            Object.entries(parsedData.captions).forEach(([id, caption]) => {
                if (caption && caption.url) {
                    captions.push({
                        id,
                        language: id,
                    })
                }
            })
        } else {
            $message.error(t('video_download.no_caption'))
        }

        videoData.value = {
            info: videoInfo,
            qualities,
            captions,
        }

    } catch (error) {
        videoData.value = {
            info: null,
            qualities: [],
            captions: []
        }
    } finally {
        isLoading.value = false
    }
}

const isDownloading = ref(false)
async function download() {
    // if downloading, return
    if (isDownloading.value) return

    // check ffmpeg
    const { success, msg } = await GetFFMPEGVersion()
    if (!success) {
        $message.error(msg)
        return
    }

    // backend check
    let id = videoData.value.info.id
    const { data: checkId, success: checkSuccess, msg: checkMsg } = await CheckTask(id, selectedQuality.value.id, selectedCaption.value.id)
    if (!checkSuccess) {
        $message.error(checkMsg)
        return
    } else {
        if (checkId) {
            // means the task has downloaded, return new temp id to save db
            id = checkId
        }
    }

    try {
        isDownloading.value = true
        emitter.emit(EMITTER_EVENTS.DOWNLOAD, {
            id: id,
            stream: selectedQuality.value.id,
            caption: selectedCaption.value.id,
        })
    } finally {
        setTimeout(() => {
            downloadStore.setInstantData()
        }, 500)
    }
}

const formatQuality = (quality) => {
    if (!quality) return ''
    // Split by semicolon, take the first part
    const beforeSemicolon = quality.split(';')[0]
    // Split by space, take the first part
    const beforeSpace = beforeSemicolon.split(' ')[0]
    return beforeSpace.split('/')[0]
}

const refreshInstantData = () => {
    downloadStore.setInstantData()
}

// Get data once when the component is loaded
onMounted(() => {
    refreshInstantData()
})

// Get site icon
const getSiteIcon = (source) => {
    const site = source?.toLowerCase()
    if (site?.includes('youtube')) return 'YoutubeIcon'
    if (site?.includes('bilibili')) return 'BilibiliIcon'
    return 'fas fa-folder' // Default icon
}

// Get site name
const getSiteName = (source) => {
    if (source?.toLowerCase().includes('youtube')) return 'Youtube'
    if (source?.toLowerCase().includes('bilibili')) return 'Bilibili'
    return 'Source'
}

const formatFileSize = (size) => {
    if (!size) return '0 B'
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    let index = 0
    let fileSize = Number(size)

    while (fileSize >= 1024 && index < units.length - 1) {
        fileSize /= 1024
        index++
    }

    return `${fileSize.toFixed(2)} ${units[index]}`
}

const openFolder = (savedPath) => {
    OpenDirectory(savedPath)
}

function downloadFailed(total, finished, status) {
    if (total === finished) {
        if (String(status).includes('Failed')) {
            return true
        }
    }
    return false
}

const showErrorModal = (error) => {
    currentError.value = error
    errorModal.value.showModal()
}

const showDetailModal = (item) => {
    currentItem.value = item
    detailModal.value.showModal()
}

const copyItemInfo = () => {
    const info = `${t('video_download.title')}: ${currentItem.value.title}
${t('video_download.url')}: ${currentItem.value.url}
${t('video_download.format')}: ${currentItem.value.format}
${t('video_download.quality')}: ${currentItem.value.quality}
${t('video_download.status')}: ${currentItem.value.status}
${t('video_download.progress')}: ${Number(currentItem.value.progress).toFixed(2)}%
${t('video_download.finished')}: ${currentItem.value.finished}
${t('video_download.total')}: ${currentItem.value.total}
${t('video_download.size')}: ${formatFileSize(currentItem.value.size)}
${t('video_download.saved_path')}: ${currentItem.value.savedPath}
${currentItem.value.error ? `${t('video_download.error_info')}: ${currentItem.value.error}` : ''}`

    navigator.clipboard.writeText(info)
}
</script>

<style scoped>
.max-w-xs {
    max-width: 20rem;
    /* or other suitable width */
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.table-container {
    overflow-y: auto;
}

.table thead tr {
    position: sticky;
    top: 0;
    background-color: var(--base-100);
    z-index: 1;
}

.table tbody tr {
    height: 3rem;
}

.table-container::-webkit-scrollbar {
    width: 8px;
    height: 8px;
}

.table-container::-webkit-scrollbar-track {
    background: #f1f1f1;
    border-radius: 4px;
}

.table-container::-webkit-scrollbar-thumb {
    background: #888;
    border-radius: 4px;
}

.table-container::-webkit-scrollbar-thumb:hover {
    background: #555;
}

.fa-youtube {
    color: #FF0000;
    /* YouTube red */
}

.fa-bilibili {
    color: #00A1D6;
    /* Bilibili blue */
}

.table {
    width: 100%;
    table-layout: fixed;
}

.table th:nth-child(1),
.table td:nth-child(1) {
    width: 5%;
    /* Source icon column */
}

.table th:nth-child(2),
.table td:nth-child(2) {
    width: 30%;
    /* Title column */
}

.table th:nth-child(3),
.table td:nth-child(3) {
    width: 8%;
    /* Format */
}

.table th:nth-child(4),
.table td:nth-child(4) {
    width: 12%;
    /* Size column */
}

.table th:nth-child(5),
.table td:nth-child(5) {
    width: 25%;
    /* Progress column */
}

.table th:nth-child(6),
.table td:nth-child(6) {
    width: 15%;
    /* Download column */
}

.table th:nth-child(7),
.table td:nth-child(7) {
    width: 5%;
    /* Directory column */
}

.table td {
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

:deep(.table) {
    margin-bottom: 0;
}
</style>
