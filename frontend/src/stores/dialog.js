import {defineStore} from 'pinia'

/**
 * connection dialog type
 * @enum {number}
 */
export const ConnDialogType = {
    NEW: 0,
    EDIT: 1,
}

const useDialogStore = defineStore('dialog', {
    state: () => ({
        connDialogVisible: false,
        /** @type {ConnDialogType} **/
        connType: ConnDialogType.NEW,
        connParam: null,

        groupDialogVisible: false,
        editGroup: '',

        /**
         * @property {string} prefix
         * @property {string} server
         * @property {int} db
         */
        newKeyParam: {
            prefix: '',
            server: '',
            db: 0,
        },
        newKeyDialogVisible: false,

        keyFilterParam: {
            server: '',
            db: 0,
            type: '',
            pattern: '*',
        },
        keyFilterDialogVisible: false,

        addFieldParam: {
            server: '',
            db: 0,
            key: '',
            keyCode: null,
            type: null,
        },
        addFieldsDialogVisible: false,

        renameKeyParam: {
            server: '',
            db: 0,
            key: '',
        },
        renameDialogVisible: false,

        deleteKeyParam: {
            server: '',
            db: 0,
            key: '',
        },
        deleteKeyDialogVisible: false,

        exportKeyParam: {
            server: '',
            db: 0,
            keys: [],
        },
        exportKeyDialogVisible: false,

        importKeyParam: {
            server: '',
            db: 0,
        },
        importKeyDialogVisible: false,

        flushDBParam: {
            server: '',
            db: 0,
        },
        flushDBDialogVisible: false,

        ttlDialogVisible: false,
        ttlParam: {
            server: '',
            db: 0,
            key: '',
            keys: [],
            ttl: 0,
        },

        decodeDialogVisible: false,
        decodeParam: {
            name: '',
            auto: true,
            decodePath: '',
            decodeArgs: [],
            encodePath: '',
            encodeArgs: [],
        },

        preferencesDialogVisible: false,
        preferencesTag: '',

        aboutDialogVisible: false,
    }),
    actions: {
        openNewDialog() {
            this.connParam = null
            this.connType = ConnDialogType.NEW
            this.connDialogVisible = true
        },
        closeConnDialog() {
            this.connDialogVisible = false
        },

        openPreferencesDialog(tag = '') {
            this.preferencesDialogVisible = true
            this.preferencesTag = tag
        },
        closePreferencesDialog() {
            this.preferencesDialogVisible = false
            this.preferencesTag = ''
        },

        openAboutDialog() {
            this.aboutDialogVisible = true
        },
        closeAboutDialog() {
            this.aboutDialogVisible = false
        },
    },
})

export default useDialogStore
