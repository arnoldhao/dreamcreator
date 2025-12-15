import { System } from '@wailsio/runtime'

let os = ''

export async function loadEnvironment() {
    try {
        const env = await System.Environment()
        // Wails v3 exposes OS via env.OS; keep fallbacks for any legacy shape
        os = env?.OS || env?.PlatformInfo?.OS || env?.platform || ''
        document.documentElement.setAttribute('data-platform', os || '')
    } catch {
        os = ''
    }
}

export function isMacOS() {
    return os === 'darwin'
}

export function isWindows() {
    return os === 'windows'
}
