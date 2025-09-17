import { Environment } from 'wailsjs/runtime/runtime.js'

let os = ''

export async function loadEnvironment() {
    const env = await Environment()
    os = env.platform
    try { document.documentElement.setAttribute('data-platform', os || '') } catch {}
}

export function isMacOS() {
    return os === 'darwin'
}

export function isWindows() {
    return os === 'windows'
}
