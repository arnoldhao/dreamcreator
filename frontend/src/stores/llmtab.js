import { ref } from 'vue' 
import { defineStore } from 'pinia'
import { i18nGlobal } from "@/utils/i18n.js";
import { LLMContentItem, CurrentModelItem } from '../objects/llmContentItem'
import { ListLLMsAndModels, AddLLM, UpdateLLM, DeleteLLM, AddModel, UpdateCurrentModel, RestoreAIs, GetCurrentModel } from 'wailsjs/go/llms/Service'
const useLLMTabStore = defineStore('llmtab', {
    state: () => ({
        llmList: [],
        currentModel: {},
        configContent: {}
    }),
    getters: {
        llms() {
            return this.llmList
        },
        totoalLLMs() {
            return this.llmList.length || 0
        },
        totoalModels() {
            return this.llmList.reduce((total, llm) => total + llm.models.length, 0) || 0
        }
    },
    actions: {
        async initialize(refreshContent = false) {
            const { data, success, msg } = await ListLLMsAndModels()
            if (success) {
                this.llmList = data;
            } else {
                $message.error(msg)
            }
            if (refreshContent) {
                this.configContent = new LLMContentItem()
            }

            const { data: cdata, success: csuccess, msg: cmsg } = await GetCurrentModel()
            if (csuccess) {
                if (!cdata || (!cdata.llmName && !cdata.modelName)) {
                    this.currentModel = new CurrentModelItem('', '', i18nGlobal.t('ai.no_current_model_tip'))
                } else {
                    this.currentModel = new CurrentModelItem(cdata.llmName || '', cdata.modelName || '', '')
                }
            } else {
                $message.error(cmsg)
            }
        },
        async updateCurrentModel(llmName, modelName) {
            const { success, msg } = await UpdateCurrentModel(llmName, modelName)
            if (success) {
                this.currentModel = new CurrentModelItem(llmName, modelName, '')
                $message.success(i18nGlobal.t('ai.update_success'))
            } else {
                $message.error(msg)
            }
        },
        newLLm() {
            this.configContent = new LLMContentItem(true, '', '', '', '', '', true, [])
        },
        cancelLLM() {
            this.configContent =  new LLMContentItem()
        },
        editLLm(name) {
            const llm = this.llmList.find(llm => llm.name === name);
            if (llm) {
                this.configContent = new LLMContentItem(
                    llm.isNew = false,
                    llm.name,
                    llm.region,
                    llm.baseURL,
                    llm.APIKey,
                    llm.icon,
                    llm.show,
                    llm.models
                )
            } else {
                $message.error('LLM not found')
            }
        },
        async deleteLLM(llmName) {
            const { data, success, msg } = await DeleteLLM(llmName)
            if (success) {
                this.initialize(true)   // refresh llm list
                $message.success(i18nGlobal.t('common.delete_success'))
            } else {
              $message.error(msg)
            }
        },
        async saveLLM(isNew, llmData) {            
            const llm = { ...llmData }        
            let { success, msg } = {}
            if (isNew) {
                ({ success, msg } = await AddLLM(JSON.stringify(llm)))
            } else {
                ({ success, msg } = await UpdateLLM(JSON.stringify(llm)))
            }
            if (success) {
                this.initialize(true)   // refresh llm list
                $message.success(i18nGlobal.t('ai.save_success'))
            } else {
                $message.error(msg)
            }
        },
        async addModel(llmName, models) {
            const { success, msg } = await AddModel(llmName, models)
            if (success) {
                this.initialize(false)
                $message.success(i18nGlobal.t('ai.create_success'))
            } else {
                $message.error(msg)
            }
        },
        async restoreAIs() {
            const { success, msg } = await RestoreAIs()
            if (success) {
                this.initialize(true)   // refresh llm list
                $message.success(i18nGlobal.t('ai.restore_success'))
            } else {
                $message.error(msg)
            }
        }
    }
})

export default useLLMTabStore