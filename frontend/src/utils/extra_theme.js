/**
 * @typedef ExtraTheme
 * @property {string} titleColor
 * @property {string} sidebarColor
 * @property {string} splitColor
 */

/**
 *
 * @type ExtraTheme
 */
export const extraLightTheme = {
    // titleColor: '#F2F2F2',
    // ribbonColor: '#F9F9F9',
    ribbonActiveColor: '#E3E3E3',
    sidebarColor: '#F2F2F2',
    splitColor: '#DADADA',
    uniFrameColor: '#ffffff22',
    sidebarActiveBg: 'rgba(64, 158, 255, 0.12)',
    primaryColor: '#409EFF',
    contentBackgroundColor: '#F7F7F7'  // 浅灰色背景
}

/**
 *
 * @type ExtraTheme
 */
export const extraDarkTheme = {
    // titleColor: '#262626',
    // ribbonColor: '#2C2C2C',
    ribbonActiveColor: '#363636',
    sidebarColor: '#292929',
    splitColor: '#474747',
    uniFrameColor: '#ffffff22',
    sidebarActiveBg: 'rgba(64, 158, 255, 0.12)',
    primaryColor: '#409EFF',
    contentBackgroundColor: '#1E1E1E'  // 深色背景
}

/**
 *
 * @param {boolean} dark
 * @return ExtraTheme
 */
export const extraTheme = (dark) => {
    return dark ? extraDarkTheme : extraLightTheme
}
