<template>
    <div v-bind="$attrs" class="flex flex-col h-full w-full">
        <div class="card bg-base-100 shadow-xl flex flex-col w-full h-full">
            <div class="card-body p-4 flex flex-col h-full w-full">
                <!-- header content -->
                <div class="flex-none">
                    <div class="flex justify-between items-center mb-2">
                        <button class="btn btn-ghost normal-case text-xl" @click="getAllSubtitles">
                            <HistoryIcon class="w-6 h-6 mr-2" />
                            {{ $t('history.history') }}
                        </button>
                        <div>
                            <button class="btn btn-sm btn-outline mr-2" @click="getAllSubtitles">{{ $t('history.refresh')
                                }}</button>
                            <button class="btn btn-sm btn-outline" @click="openClearModal">{{ $t('history.clear')
                                }}</button>
                        </div>
                    </div>
                    <p class="text-sm text-base-content/60 italic mb-2">{{ $t('history.converted_or_translated_files') }}
                    </p>
                </div>

                <!-- table container -->
                <div class="flex-grow overflow-auto">
                    <table class="table table-zebra w-full table-fixed">
                        <thead>
                            <tr>
                                <th class="whitespace-nowrap overflow-hidden text-ellipsis">{{ $t('history.file_name')
                                    }}</th>
                                <th class="whitespace-nowrap overflow-hidden text-ellipsis">{{ $t('history.content') }}
                                </th>
                                <th class="whitespace-nowrap overflow-hidden text-ellipsis">{{ $t('history.language') }}
                                </th>
                                <th class="whitespace-nowrap overflow-hidden text-ellipsis">{{ $t('history.model') }}
                                </th>
                                <th class="whitespace-nowrap overflow-hidden text-ellipsis">{{ $t('history.action') }}
                                </th>
                            </tr>
                        </thead>
                        <tbody>
                            <tr v-for="(row, index) in paginatedSubtitles" :key="index">
                                <td class="max-w-[200px] truncate" :title="row.FileName">{{ row.FileName }}</td>
                                <td class="max-w-[300px] truncate" :title="row.Brief">{{ row.Brief }}</td>
                                <td><span class="badge badge-info">{{ row.Language }}</span></td>
                                <td><span class="badge badge-ghost">{{ row.Models.length > 10 ? row.Models.slice(0,
                                    10)
                                    + '...' : row.Models }}</span></td>
                                <td>
                                    <div class="dropdown">
                                        <label tabindex="0" class="btn btn-sm m-1">{{ $t('history.action')
                                            }}</label>
                                        <ul tabindex="0"
                                            class="dropdown-content menu p-2 shadow bg-base-100 rounded-box z-50 whitespace-nowrap min-w-max">
                                            <li><a @click="handleEdit(row)">{{ $t('history.edit') }}</a></li>
                                            <li><a @click="handleDownload(row)">{{ $t('history.download') }}</a>
                                            </li>
                                            <li><a @click="handleAITranslateWebSocket(row)">{{
                                                $t('history.translate_to_english') }}</a></li>
                                            <li><a @click="handleDelete(row)">{{ $t('history.delete') }}</a></li>
                                        </ul>
                                    </div>
                                </td>
                            </tr>
                        </tbody>
                    </table>
                </div>

                <!-- pagination -->
                <div class="flex-none mt-4">
                    <!-- pagination -->
                    <div class="pt-4">
                        <div class="flex justify-center space-x-1">
                            <button class="join-item btn" :disabled="currentPage === 1"
                                @click="changePage(currentPage - 1)">«</button>
                            <button class="join-item btn" v-for="page in totalPages" :key="page"
                                :class="{ 'btn-active': page === currentPage }" @click="changePage(page)">
                                {{ page }}
                            </button>
                            <button class="join-item btn" :disabled="currentPage === totalPages"
                                @click="changePage(currentPage + 1)">»</button>
                        </div>
                    </div>
                </div>
            </div>
        </div>

        <!-- clear confirm modal -->
        <div class="modal" :class="{ 'modal-open': isClearModalOpen }">
            <div class="modal-box">
                <h3 class="font-bold text-lg">{{ $t('history.confirm_clear_history') }}</h3>
                <p class="py-4">{{ $t('history.please_select_the_way_to_clear') }}</p>
                <div class="flex justify-around my-4">
                    <button class="btn btn-outline" @click="confirmClear(7)">{{ $t('history.keep_7_days_data')
                        }}</button>
                    <button class="btn btn-outline" @click="confirmClear(30)">{{ $t('history.keep_30_days_data')
                        }}</button>
                    <button class="btn btn-outline btn-error" @click="confirmClear(0)">{{ $t('history.clear_all')
                        }}</button>
                </div>
                <div class="modal-action">
                    <button class="btn" @click="closeClearModal">{{ $t('history.cancel') }}</button>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { ListSubtitles, SaveSubtitles, DeleteSubtitles, DeleteSubtitleByKeepDays, DeleteAllSubtitles } from 'wailsjs/go/subtitles/Service';
import useSuperTabStore from "stores/supertab.js";
import HistoryIcon from '@/components/icons/History.vue';
import { storeToRefs } from "pinia";
import { useAttrs } from 'vue';
import { useI18n } from 'vue-i18n';

const tabStore = useSuperTabStore();
const { nav } = storeToRefs(tabStore);
const { t } = useI18n();
// subtitles data
const allSubtitles = ref([]);

// pagination
const currentPage = ref(1);
const pageSize = ref(15);

// calculate total pages
const totalPages = computed(() => {
    if (!allSubtitles.value || allSubtitles.value.length === 0) {
        return 1;
    }
    return Math.ceil(allSubtitles.value.length / pageSize.value);
});

// calculate current page data
const paginatedSubtitles = computed(() => {
    if (!allSubtitles.value || allSubtitles.value.length === 0) {
        return [];
    }
    const start = (currentPage.value - 1) * pageSize.value;
    const end = start + pageSize.value;
    return allSubtitles.value.slice(start, end);
});

// switch page
function changePage(page) {
    currentPage.value = page;
}

// get all subtitles
async function getAllSubtitles() {
    try {
        const { data, success, msg } = await ListSubtitles();
        if (success) {
            allSubtitles.value = data || [];
            currentPage.value = 1; // reset to first page
        } else {
            throw new Error(msg)
        }
    } catch (error) {
        console.error('Error loading subtitles:', error);
        allSubtitles.value = [];
    }
}

// handle edit
function handleEdit(row) {
    tabStore.editSubtitle(row.Key, row.FileName, row.Language);
}

// handle download
async function handleDownload(row) {
    const { data, success, msg } = await SaveSubtitles(row.Key, row.FileName, ['srt', 'vtt', 'ass']);
    if (success) {
        $message.success(t('history.save_success', { path: data?.path || '' }));
    } else {
        if (data.canceled) {
            $message.info(t('history.cancel_success'));
        } else {
            $message.error(t('history.save_failed') + msg);
        }
    }
}

const handleAITranslateWebSocket = (row) => {
    const transLanguage = 'English';
    tabStore.translateSubtitle(row.Key, row.FileName + "_" + transLanguage, transLanguage);
}

watch(nav, (newVal) => {
    if (newVal === 'history') {
        getAllSubtitles();
    }
});

// delete
async function handleDelete(row) {
    try {
        const { success, msg } = await DeleteSubtitles(row.Key);
        if (success) {
            $message.success(t('history.delete_success'));
            await getAllSubtitles();
        } else {
            throw new Error(msg);
        }
    } catch (error) {
        $message.error(t('history.delete_failed') + error.message);
    }
}

async function handleDeleteAll() {
    try {
        const { success, msg } = await DeleteAllSubtitles();
        return { success, msg };
    } catch (error) {
        $message.error(t('history.delete_failed') + error.message);
        return { success: false, msg: error.message };
    }
}

async function handleDeleteByKeepDays(keepDays) {
    try {
        const { success, msg } = await DeleteSubtitleByKeepDays(keepDays);
        return { success, msg };
    } catch (error) {
        $message.error(t('history.delete_failed') + error.message);
        return { success: false, msg: error.message };
    }
}

const isClearModalOpen = ref(false);

function openClearModal() {
    isClearModalOpen.value = true;
}

function closeClearModal() {
    isClearModalOpen.value = false;
}

async function confirmClear(days) {
    try {
        let result;
        if (days === 0) {
            result = await handleDeleteAll();
        } else {
            result = await handleDeleteByKeepDays(days);
        }
        if (result.success) {
            $message.success(result.msg);
            await getAllSubtitles(); // refresh list
        } else {
            throw new Error(result.msg);
        }
    } catch (error) {
        $message.error(t('history.delete_failed') + error.message);
    } finally {
        closeClearModal();
    }
}

// get subtitles when component mounted
onMounted(() => {
    getAllSubtitles();
});

const attrs = useAttrs();

</script>

<style scoped>
.card-body {
    display: flex;
    flex-direction: column;
    height: 100%;
}

.flex-grow {
    flex-grow: 1;
    min-height: 0;
}
</style>
