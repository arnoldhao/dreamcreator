import { createPinia } from 'pinia'
import { createApp, nextTick } from 'vue'
import App from './App.vue'
import './index.css'
import './styles/style.scss'
import dayjs from 'dayjs'
import duration from 'dayjs/plugin/duration'
import relativeTime from 'dayjs/plugin/relativeTime'
import { i18n } from '@/utils/i18n.js'
import { setupDiscreteApi } from '@/utils/discrete.js'
import usePreferencesStore from 'stores/preferences.js'
import { loadEnvironment } from '@/utils/platform.js'
import { setupMonaco } from '@/utils/monaco.js'
import { OhVueIcon, addIcons } from 'oh-vue-icons'
import { CoGithub, 
    RiGlobalLine, 
    CiX, 
    RiCloudLine, 
    HiCubeTransparent, 
    RiSettings3Line, 
    RiInformationLine, 
    RiTvLine, 
    MdSubtitles, 
    LaRobotSolid, 
    RiHistoryLine, 
    RiDownloadCloudLine, 
    RiMoonLine, 
    RiSunLine, 
    MdHdrauto, 
    CoLanguage, 
    OiFileDirectory,
    CoFont,
    RiFontSize,
    MdEditnote,
    BiUiChecks,
    OiRocket,
    HiRefresh,
    RiDeleteBinLine,
    BiChevronLeft,
    BiChevronRight,
    BiThreeDotsVertical,
    RiLoader2Line,
    MdContentcopy,
    IoClose,
    RiFolderOpenLine,
    RiFileUnknowLine,
    MdTask,
    MdPending,
    MdDownloading,
    MdDownloaddone,
    MdPause,
    BiPuzzle,
    BiPuzzleFill,
    CoPuzzle,
    IoCloudDoneOutline,
    MdRunningwitherrors,
    MdFiledownloadoff,
    MdFreecancellation,
    MdFreecancellationOutlined,
    IoTimerOutline,
    MdClouddownload,
    MdSpeed,
    RiBilibiliLine,
    RiYoutubeLine,
    RiVideoLine,
    BiInbox} from 'oh-vue-icons/icons'

// Register the icon
addIcons(CoGithub, 
    RiGlobalLine, 
    CiX, 
    RiCloudLine, 
    HiCubeTransparent, 
    RiSettings3Line, 
    RiInformationLine, 
    RiTvLine, 
    MdSubtitles, 
    LaRobotSolid, 
    RiHistoryLine, 
    RiDownloadCloudLine, 
    RiMoonLine, 
    RiSunLine, 
    MdHdrauto, 
    CoLanguage, 
    OiFileDirectory,
    CoFont,
    RiFontSize,
    MdEditnote,
    BiUiChecks,
    OiRocket,
    HiRefresh,
    RiDeleteBinLine,
    BiChevronLeft,
    BiChevronRight,
    BiThreeDotsVertical,
    RiLoader2Line,
    MdContentcopy,
    IoClose,
    RiFolderOpenLine,
    RiFileUnknowLine,
    MdTask,
    MdPending,
    MdDownloading,
    MdPause,
    MdDownloaddone,
    BiPuzzle,
    BiPuzzleFill,
    CoPuzzle,
    IoCloudDoneOutline,
    MdRunningwitherrors,
    MdFiledownloadoff,
    MdFreecancellation,
    MdFreecancellationOutlined,
    IoTimerOutline,
    MdClouddownload,
    MdSpeed,
    RiBilibiliLine,
    RiYoutubeLine,
    RiVideoLine,
    BiInbox)

dayjs.extend(duration)
dayjs.extend(relativeTime)

async function setupApp() {
    const app = createApp(App)
    app.use(i18n)
    app.use(createPinia())

    // Register OhVueIcon component globally
    app.component("v-icon", OhVueIcon);

    await loadEnvironment()
    setupMonaco()
    const prefStore = usePreferencesStore()
    await prefStore.loadPreferences()
    await setupDiscreteApi()
    app.config.errorHandler = (err, instance, info) => {
        // TODO: add "send error message to author" later
        nextTick().then(() => {
            try {
                const content = err.toString()
                $notification.error(content, {
                    title: i18n.global.t('common.error'),
                    meta: i18n.global.t('message.console_tip'),
                })
                console.error(err)
            } catch (e) { }
        })
    }
    // app.config.warnHandler = (message) => {
    //     console.warn(message)
    // }
    app.mount('#app')
}

setupApp()
