<template>
    <div class="history-container rounded-tl-lg h-full w-full">
        <div class="h-full w-full bg-base-200">
            <div class="h-full p-6 overflow-y-auto">
                <!-- 标题区域 -->
                <div class="flex justify-between items-center mb-6">
                    <div class="flex items-center">
                        <v-icon name="ri-history-line" class="w-6 h-6 mr-3 text-primary"></v-icon>
                        <span class="text-xl font-bold">{{ $t('history.history') }}</span>
                    </div>
                    <div class="space-x-3">
                        <button class="btn btn-sm btn-primary btn-outline" @click="getAllSubtitles">
                            <v-icon name="hi-refresh" class="h-4 w-4 mr-1"></v-icon>
                            {{ $t('history.refresh') }}
                        </button>
                        <button class="btn btn-sm btn-error btn-outline" @click="openClearModal">
                            <v-icon name="ri-delete-bin-line" class="h-4 w-4 mr-1"></v-icon>
                            {{ $t('history.clear') }}
                        </button>
                    </div>
                </div>

                <!-- 表格区域 -->
                <div class="card bg-base-100 shadow-md overflow-hidden transition-all duration-200 hover:shadow-lg">
                    <div class="card-body p-0 overflow-visible">
                        <div>
                            <table class="table w-full table-fixed">
                                <thead class="bg-base-300">
                                    <tr>
                                        <th class="w-1/5 font-bold table-header">{{ $t('history.file_name') }}</th>
                                        <th class="w-1/3 font-bold table-header">{{ $t('history.content') }}</th>
                                        <th class="w-1/8 font-bold table-header">{{ $t('history.language') }}</th>
                                        <th class="w-1/8 font-bold table-header">{{ $t('history.model') }}</th>
                                        <th class="w-1/8 font-bold table-header">{{ $t('history.action') }}</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <tr v-for="(row, index) in paginatedSubtitles" 
                                        :key="index" 
                                        class="hover:bg-base-200 transition-colors duration-150">
                                        <td class="max-w-[200px] truncate font-medium" :title="row.FileName">{{ row.FileName }}</td>
                                        <td class="max-w-[300px] truncate text-sm" :title="row.Brief">{{ row.Brief }}</td>
                                        <td>
                                            <span class="badge badge-sm text-xs font-medium max-w-full truncate bg-purple-100 text-purple-800" :title="row.Language">{{ row.Language }}</span>
                                        </td>
                                        <td>
                                            <span class="badge badge-ghost badge-sm text-xs max-w-full truncate" :title="row.Models">{{ row.Models.length > 10 ? row.Models.slice(0, 10) + '...' : row.Models }}</span>
                                        </td>
                                        <td>
                                            <div class="dropdown dropdown-end">
                                                <label tabindex="0" class="btn btn-sm btn-ghost">
                                                    <svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 5v.01M12 12v.01M12 19v.01M12 6a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2zm0 7a1 1 0 110-2 1 1 0 010 2z" />
                                                    </svg>
                                                </label>
                                                <ul tabindex="0" class="dropdown-content menu p-2 shadow-lg bg-base-100 rounded-box z-[100] w-40 dropdown-menu-fixed">
                                                    <li>
                                                        <a @click="handleEdit(row)" class="font-medium hover:text-primary transition-colors">
                                                            {{ $t('history.edit') }}
                                                        </a>
                                                    </li>
                                                    <li>
                                                        <a @click="handleDownload(row)" class="font-medium hover:text-primary transition-colors">
                                                            {{ $t('history.download') }}
                                                        </a>
                                                    </li>
                                                    <li>
                                                        <a @click="handleAITranslateWebSocket(row)" class="font-medium hover:text-primary transition-colors">
                                                            {{ $t('history.translate_to_english') }}
                                                        </a>
                                                    </li>
                                                    <li>
                                                        <a @click="handleDelete(row)" class="text-error font-medium hover:text-error-dark transition-colors">
                                                            {{ $t('history.delete') }}
                                                        </a>
                                                    </li>
                                                </ul>
                                            </div>
                                        </td>
                                    </tr>
                                    <tr v-if="!paginatedSubtitles || paginatedSubtitles.length === 0">
                                        <td colspan="5" class="text-center py-8">
                                            <div class="flex flex-col items-center justify-center text-base-content/50">
                                                <svg xmlns="http://www.w3.org/2000/svg" class="h-12 w-12 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                                                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                                                </svg>
                                                <p>{{ $t('history.no_records') || 'No history records found' }}</p>
                                            </div>
                                        </td>
                                    </tr>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>

                <!-- 分页区域 -->
                <div class="flex justify-center mt-6" v-if="allSubtitles.length > 0">
                    <div class="join shadow">
                        <button class="join-item btn" :class="{'btn-disabled': currentPage === 1}" @click="changePage(currentPage - 1)">
                            <v-icon name="bi-chevron-left"></v-icon>
                        </button>
                        <button 
                            v-for="page in totalPages" 
                            :key="page" 
                            class="join-item btn" 
                            :class="{ 'btn-active': page === currentPage }" 
                            @click="changePage(page)">
                            {{ page }}
                        </button>
                        <button class="join-item btn" :class="{'btn-disabled': currentPage === totalPages}" @click="changePage(currentPage + 1)">
                            <v-icon name="bi-chevron-right"></v-icon>
                        </button>
                    </div>
                </div>
            </div>
        </div>

        <!-- Clear History Modal -->
        <div class="modal" :class="{'modal-open': isClearModalOpen}">
            <div class="modal-box">
                <h3 class="font-bold text-lg">{{ $t('history.clear_confirm_title') || 'Clear History' }}</h3>
                <p class="py-4">{{ $t('history.clear_confirm_text') || 'Choose how you want to clear history:' }}</p>
                <div class="flex flex-col gap-3 mt-2">
                    <button @click="confirmClear(0)" class="btn btn-error">
                        {{ $t('history.clear_all') || 'Clear All History' }}
                    </button>
                    <button @click="confirmClear(7)" class="btn btn-warning">
                        {{ $t('history.clear_7days') || 'Keep Last 7 Days' }}
                    </button>
                    <button @click="confirmClear(30)" class="btn btn-info">
                        {{ $t('history.clear_30days') || 'Keep Last 30 Days' }}
                    </button>
                </div>
                <div class="modal-action">
                    <button class="btn" @click="closeClearModal">{{ $t('common.cancel') || 'Cancel' }}</button>
                </div>
            </div>
            <div class="modal-backdrop" @click="closeClearModal"></div>
        </div>
    </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue';
import { ListSubtitles, SaveSubtitles, DeleteSubtitles, DeleteSubtitleByKeepDays, DeleteAllSubtitles } from 'wailsjs/go/subtitles/Service';
import useSuperTabStore from "stores/supertab.js";
import { storeToRefs } from "pinia";
import { useI18n } from 'vue-i18n';

const tabStore = useSuperTabStore();
const { nav } = storeToRefs(tabStore);
const { t } = useI18n();
// subtitles data
const allSubtitles = ref([]);

// pagination
const currentPage = ref(1);
const pageSize = ref(10);

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


</script>

<style scoped>
.history-container {
    height: 100%;
    width: 100%;
    min-width: 500px;
    flex-shrink: 0;
    overflow: hidden;
}

/* Add smooth transitions */
.card {
    transition: all 0.2s ease-in-out;
}

/* Add responsive table styles */
@media (max-width: 768px) {
    .table {
        font-size: 0.85rem;
    }
}

/* Add modal backdrop for better UX */
.modal-backdrop {
    background-color: rgba(0, 0, 0, 0.4);
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    z-index: -1;
}

/* Add custom badge styling for language */
.badge.bg-purple-100 {
    background-color: rgba(147, 51, 234, 0.15);
    color: rgba(107, 33, 168, 1);
    border: 1px solid rgba(147, 51, 234, 0.3);
}

/* Improve table header contrast - using class instead of direct element selector */
.table-header {
    color: hsl(var(--p)); /* Use primary color for better visibility in all themes */
    border-bottom: 2px solid rgba(147, 51, 234, 0.3); /* Subtle purple border */
}

thead.bg-base-300 th {
    /* Remove the fixed color that doesn't work in dark mode */
    /* color: rgba(31, 41, 55, 0.9); */ /* Dark text for better contrast */
    border-bottom: 2px solid rgba(147, 51, 234, 0.3); /* Subtle purple border */
}

/* Fix dropdown menu positioning */
.dropdown-menu-fixed {
    position: fixed;
    margin-top: 0;
}

/* Make sure the dropdown has high z-index and doesn't get clipped */
.card-body {
    overflow: visible !important;
}

.table-container {
    overflow: visible;
}

.card {
    overflow: visible !important;
}

/* Ensure table parent has proper overflow handling */
.overflow-x-auto {
    overflow-x: auto;
    overflow-y: visible !important;
}
</style>
