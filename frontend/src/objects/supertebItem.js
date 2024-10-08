import { TranslationStateManager } from '@/utils/translationStateManager'

/**
 * super tab item
 */
export class SuperTabItem {
    constructor({
        id,
        title,
        filePath,
        blank,
        icon,
        loading = false,
        originalSubtileId,
        language,
        stream,
        captions,
    }) {
        this.id = id
        this.title = title
        this.filePath = filePath
        this.blank = blank
        this.icon = icon
        this.loading = loading
        this.originalSubtileId = originalSubtileId
        this.language = language
        this.stream = stream
        this.captions = captions
        this.translationState = new TranslationStateManager()
    }

    updateTranslationState(newState) {
        this.translationState.updateState(newState)
    }

    getTranslationState() {
        return this.translationState.getState()
    }
}
