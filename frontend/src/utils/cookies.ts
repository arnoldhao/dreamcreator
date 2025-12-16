export type NormalCookie = {
  domain: string
  name: string
  value: string
  path: string
  secure: boolean
  includeSubdomains: boolean
  expires: number
  httpOnly: boolean
}

export type CookieParseResult = {
  netscape: string
  original?: string
  defaultDomain?: string
}

function normalizeDomain(domain: string, hostOnly: boolean): string {
  let d = String(domain || "").trim()
  if (!d) return ""
  d = d.replace(/^https?:\/\//i, "").replace(/\/.*$/, "").replace(/:\d+$/, "")
  if (!d) return ""

  if (hostOnly) {
    return d.startsWith(".") ? d.slice(1) : d
  }
  return d.startsWith(".") ? d : `.${d}`
}

function parseNetscape(text: string): NormalCookie[] {
  const out: NormalCookie[] = []
  const lines = String(text || "").split(/\r?\n/)
  for (const rawLine of lines) {
    const trimmed = rawLine.trim()
    if (!trimmed) continue

    let line = trimmed
    let httpOnly = false
    if (line.toLowerCase().startsWith("#httponly_")) {
      httpOnly = true
      line = line.slice(10)
    } else if (line.startsWith("#")) {
      continue
    }

    const parts = line.split("\t")
    if (parts.length !== 7) continue
    const [domain, includeSubdomains, path, secure, expires, name, value] = parts
    if (!name) continue
    out.push({
      domain: String(domain || "").trim(),
      name: String(name || "").trim(),
      value: String(value || ""),
      path: String(path || "/") || "/",
      secure: String(secure || "").toUpperCase() === "TRUE",
      includeSubdomains: String(includeSubdomains || "").toUpperCase() === "TRUE",
      expires: (() => {
        const n = Number(expires || "0")
        return Number.isFinite(n) ? Math.floor(n) : 0
      })(),
      httpOnly,
    })
  }
  return out
}

function toNetscape(cookies: NormalCookie[]): string {
  const header = ["# Netscape HTTP Cookie File", ""]
  const rows = (cookies || []).map((c) => {
    const domain = String(c.domain || "").trim()
    if (!domain) throw new Error("cookie domain missing")
    const includeSubdomains = c.includeSubdomains ? "TRUE" : "FALSE"
    const path = String(c.path || "/") || "/"
    const secure = c.secure ? "TRUE" : "FALSE"
    const expires = c.expires && c.expires > 0 ? Math.floor(c.expires) : 0
    const name = String(c.name || "")
    const value = String(c.value || "")
    const first = c.httpOnly ? `#HttpOnly_${domain}` : domain
    return [first, includeSubdomains, path, secure, String(expires), name, value].join("\t")
  })
  return [...header, ...rows].join("\n")
}

function parseHeaderCookies(raw: string, defaultDomain: string): NormalCookie[] {
  let s = String(raw || "").trim()
  if (!s) return []
  if (/^cookie\s*:/i.test(s)) s = s.replace(/^cookie\s*:/i, "")
  const d = String(defaultDomain || "").trim()
  if (!d) throw new Error("default domain required")
  const dom = normalizeDomain(d, true)
  if (!dom) throw new Error("default domain required")

  const parts = s.split(";").map((x) => x.trim()).filter(Boolean)
  const out: NormalCookie[] = []
  for (const part of parts) {
    const idx = part.indexOf("=")
    if (idx <= 0) continue
    const name = part.slice(0, idx).trim()
    const value = part.slice(idx + 1).trim()
    if (!name) continue
    out.push({
      domain: dom,
      name,
      value,
      path: "/",
      secure: false,
      includeSubdomains: false,
      expires: 0,
      httpOnly: false,
    })
  }
  return out
}

function parseJSONCookies(value: any, defaultDomain: string): NormalCookie[] {
  if (!value || typeof value !== "object") return []
  const name = value.name ?? value.Name ?? ""
  const val = value.value ?? value.Value ?? ""
  if (!name) return []

  let domain = value.domain || value.host || value.Domain || ""
  const hostOnly = Boolean(value.hostOnly ?? value.host_only ?? (!domain ? true : !String(domain).startsWith(".")))
  if (!domain) domain = defaultDomain
  if (!domain) throw new Error("default domain required")
  domain = normalizeDomain(String(domain), hostOnly)

  const path = value.path || value.Path || "/"
  const secure = Boolean(value.secure ?? value.Secure)
  const httpOnly = Boolean(value.httpOnly ?? value.HttpOnly ?? value["http-only"] ?? value.HTTPOnly)
  let expires = 0
  const exp = value.expirationDate ?? value.expires ?? value.expiry ?? value.Expiry
  if (typeof exp === "number" && Number.isFinite(exp)) {
    expires = exp > 1e12 ? Math.floor(exp / 1000) : Math.floor(exp)
  }

  return [
    {
      domain,
      name: String(name),
      value: String(val),
      path: String(path || "/") || "/",
      secure,
      includeSubdomains: !hostOnly,
      expires,
      httpOnly,
    },
  ]
}

export function parseCookieInput(rawInput: string, defaultDomain: string): CookieParseResult {
  const raw = String(rawInput || "").trim()
  if (!raw) throw new Error("empty")

  // JSON array export
  try {
    const parsed = JSON.parse(raw)
    if (Array.isArray(parsed)) {
      const cookies = parsed.flatMap((item) => parseJSONCookies(item, defaultDomain))
      if (!cookies.length) throw new Error("empty")
      const netscape = toNetscape(cookies)
      return {
        netscape,
        original: netscape,
        defaultDomain: (cookies[0]?.domain || "").replace(/^\./, "") || defaultDomain,
      }
    }
  } catch (e: any) {
    if (!(e instanceof SyntaxError)) throw e
  }

  // Netscape format
  const netscapeCookies = parseNetscape(raw)
  if (netscapeCookies.length) {
    const netscape = toNetscape(netscapeCookies)
    return {
      netscape,
      original: netscape,
      defaultDomain: (netscapeCookies[0]?.domain || "").replace(/^\./, "") || defaultDomain,
    }
  }

  // Cookie header
  const headerCookies = parseHeaderCookies(raw, defaultDomain)
  if (headerCookies.length) {
    const netscape = toNetscape(headerCookies)
    return { netscape, original: netscape, defaultDomain }
  }

  throw new Error("parse_failed")
}

export function buildCookiePreview(netscapeText: string, limit = 8) {
  const cookies = parseNetscape(netscapeText)
  return {
    total: cookies.length,
    entries: cookies.slice(0, Math.max(0, limit)).map((c) => ({
      domain: c.domain,
      name: c.name,
      value: c.value,
      path: c.path,
      secure: c.secure,
      includeSubdomains: c.includeSubdomains,
      httpOnly: c.httpOnly,
      expires: c.expires,
    })),
  }
}

