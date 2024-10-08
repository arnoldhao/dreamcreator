export class LLMContentItem {
    constructor(isNew, name, region, baseURL, APIKey, icon, show, models) {
        this.isNew = isNew
        this.name = name
        this.region = region
        this.baseURL = baseURL
        this.APIKey = APIKey
        this.icon = icon
        this.show = show
        this.models = models
    }
}

export class CurrentModelItem {
    constructor(llmName, modelName, message) {
        this.llmName = llmName
        this.modelName = modelName
        this.message = message
    }
}