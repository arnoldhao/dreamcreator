import TelemetryDeck from '@telemetrydeck/sdk'

let client = null
let currentKey = ''

export function startTelemetry(appId, clientUser, options = {}) {
  const cleanAppId = (appId ?? '').trim()
  const cleanClientUser = (clientUser ?? '').trim()
  if (!cleanAppId || cleanAppId === 'undefined' || cleanAppId === 'null' || !cleanClientUser) {
    return null
  }
  const key = `${cleanAppId}:${cleanClientUser}`
  if (client && currentKey === key) return client

  const { endpoint, ...restOptions } = options || {}

  try {
    const telemetryOptions = {
      appID: cleanAppId,
      clientUser: cleanClientUser,
      target: restOptions.target || endpoint,
      ...restOptions,
    }

    client = new TelemetryDeck(telemetryOptions)

    currentKey = key
    return client
  } catch (error) {
    console.warn('TelemetryDeck init failed', error)
    client = null
    currentKey = ''
    return null
  }
}

export function stopTelemetry() {
  client = null
  currentKey = ''
}

export async function sendTelemetry(eventName, payload = {}, options = {}) {
  if (!client || !eventName) return null
  try {
    const { version, ...restOptions } = options || {}
    const normalizedPayload = payload && typeof payload === 'object' ? { ...payload } : {}
    if (version && normalizedPayload.appVersion === undefined) {
      normalizedPayload.appVersion = version
    }

    return await client.signal(eventName, normalizedPayload, restOptions)
  } catch (err) {
    console.warn('TelemetryDeck signal failed', err)
    return null
  }
}

export function isTelemetryActive() {
  return !!client
}
