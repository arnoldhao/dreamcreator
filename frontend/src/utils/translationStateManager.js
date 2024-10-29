// define translation status
export const TranslationStatus = {
    NONEED: 'noneed',
    PENDING: 'pending',
    RUNNING: 'running',
    COMPLETED: 'completed',
    ERROR: 'error',
    CANCEL_PENDING: 'cancel_pending',
    CANCELED: 'canceled'
  }
  
  export const StreamStatus = {
    NONEED: 'noneed',
    STREAMING: 'streaming',
    TRANSLATION_DONE: 'translation_done',
    ERROR: 'error',
    CANCEL_PENDING: 'cancel_pending',
    CANCELED: 'canceled',
    CANCEL_ERROR: 'cancel_error'
  }
  
  export class TranslationStateManager {
    constructor(initialState = {}) {
      this.state = {
        streamStatus: StreamStatus.NONEED,
        translationStatus: TranslationStatus.NONEED,
        translationProgress: 0,
        actionDescription: undefined,
        isCompleted: true,
        ...initialState
      }
    }
  
    updateState(newState) {
      this.state = { ...this.state, ...newState }
      this.validateState()
    }
  
    validateState() {
      // add state validation logic here
      // for example, ensure translationProgress is between 0-100
      if (this.state.translationProgress < 0) this.state.translationProgress = 0
      if (this.state.translationProgress > 100) this.state.translationProgress = 100
  
      // ensure the consistency of the state combination
      if (this.state.streamStatus === StreamStatus.TRANSLATION_DONE) {
        this.state.translationStatus = TranslationStatus.COMPLETED
        this.state.translationProgress = 100
      }

      // ensure stream status is according to translation status
      if (this.state.translationStatus === TranslationStatus.COMPLETED) {
        this.state.streamStatus = StreamStatus.TRANSLATION_DONE
        this.state.isCompleted = true
      } else if (this.state.translationStatus === TranslationStatus.ERROR) {
        this.state.streamStatus = StreamStatus.ERROR
        this.state.isCompleted = true
      } else if (this.state.translationStatus === TranslationStatus.CANCELED) {
        this.state.streamStatus = StreamStatus.CANCELED
        this.state.isCompleted = true
      } else {
        this.state.isCompleted = false
      }
    }

    setRunning() {
      this.updateState({
        streamStatus: StreamStatus.STREAMING,
        translationStatus: TranslationStatus.RUNNING,
        isCompleted: false
      })
    }
  
    setCompleted() {
      this.updateState({
        streamStatus: StreamStatus.TRANSLATION_DONE,
        translationStatus: TranslationStatus.COMPLETED,
        translationProgress: 100,
        isCompleted: true
      })
    }
  
    setError(description) {
      this.updateState({
        streamStatus: StreamStatus.ERROR,
        translationStatus: TranslationStatus.ERROR,
        actionDescription: description,
        isCompleted: true
      })
    }

    getState() {
        return { ...this.state }
    }
}