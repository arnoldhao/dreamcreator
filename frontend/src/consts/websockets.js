export const WS_NAMESPACE = {
    TRANSLATION: 'translation',
    DOWNLOAD: 'download',
    OLLAMA: 'ollama',
    CHAT: 'chat',
    PROXY: 'proxy',
}

export const WS_REQUEST_EVENT = {
    EVENT_TRANSLATION_START: 'request_translation_start',
    EVENT_TRANSLATION_CANCEL: 'request_translation_cancel',
    EVENT_OLLAMA_PULL: 'request_ollama_pull',
    EVENT_DOWNLOAD_START: 'request_download_start',
    EVENT_DOWNLOAD_CANCEL: 'request_download_cancel',
    EVENT_PROXY_TEST: 'request_proxy_test',
}

export const WS_RESPONSE_EVENT = {
    EVENT_TRANSLATION_PROGRESS: 'response_translation_progress',
    EVENT_TRANSLATION_CANCELED: 'response_translation_canceled',
    EVENT_TRANSLATION_COMPLETED: 'response_translation_completed',
    EVENT_TRANSLATION_ERROR: 'response_translation_error',
    EVENT_OLLAMA_PULL_UPDATE: 'response_ollama_pull_update',
    EVENT_OLLAMA_PULL_CANCELED: 'response_ollama_pull_canceled',
    EVENT_OLLAMA_PULL_COMPLETED: 'response_ollama_pull_completed',
    EVENT_OLLAMA_PULL_ERROR: 'response_ollama_pull_error',
    EVENT_DOWNLOAD_PROGRESS: 'response_download_progress',
    EVENT_DOWNLOAD_COMPLETED: 'response_download_completed',
    EVENT_DOWNLOAD_ERROR: 'response_download_error',
    EVENT_PROXY_TEST_RESULT: 'response_proxy_test_result',
    EVENT_PROXY_TEST_RESULT_CANCEL: 'response_proxy_test_cancel',
    EVENT_PROXY_TEST_RESULT_COMPLETED: 'response_proxy_test_completed',
    EVENT_PROXY_TEST_RESULT_ERROR: 'response_proxy_test_error',
}