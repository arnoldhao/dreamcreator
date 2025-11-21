import MarkdownIt from 'markdown-it'

// Markdown renderer for chat messages:
// - 不允许原生 HTML，避免 XSS
// - 支持自动链接和换行
const md = new MarkdownIt({
  html: false,
  linkify: true,
  breaks: true,
})

function tryParseJson(text) {
  const trimmed = String(text || '').trim()
  if (!trimmed) return null
  if (!/^[\[{]/.test(trimmed)) return null
  try {
    return JSON.parse(trimmed)
  } catch {
    return null
  }
}

// 整条是否是“纯 JSON”（无前后多余内容）
function isWholeJson(text) {
  const trimmed = String(text || '').trim()
  if (!trimmed) return null
  if (!/^[\[{]/.test(trimmed)) return null
  // 粗略检查结尾，避免明显未闭合的流式片段误判
  const last = trimmed[trimmed.length - 1]
  if (!['}', ']'].includes(last)) return null
  try {
    return JSON.parse(trimmed)
  } catch {
    return null
  }
}

// 将整段文本按 ```fence``` 拆分为文本段与代码段
function splitByFences(raw) {
  const lines = String(raw || '').split(/\r?\n/)
  const segments = []
  let currentText = []
  let inFence = false
  let fenceLang = ''
  let fenceLines = []

  const flushText = () => {
    if (!currentText.length) return
    segments.push({
      kind: 'text',
      content: currentText.join('\n'),
    })
    currentText = []
  }

  for (const line of lines) {
    const m = line.match(/^```(\w+)?\s*$/)
    if (m) {
      if (!inFence) {
        // 开始 fence
        flushText()
        inFence = true
        fenceLang = m[1] || ''
        fenceLines = []
      } else {
        // 结束 fence：输出一个 fence 段（不包含 ``` 行本身）
        segments.push({
          kind: 'fence',
          lang: fenceLang,
          content: fenceLines.join('\n'),
        })
        inFence = false
        fenceLang = ''
        fenceLines = []
      }
      continue
    }
    if (inFence) {
      fenceLines.push(line)
    } else {
      currentText.push(line)
    }
  }

  if (inFence) {
    // 未闭合的 fence，当作普通文本原样输出（包含起始 ```lang）
    const fenceText = ['```' + (fenceLang || ''), ...fenceLines].join('\n')
    currentText.push(fenceText)
  }

  flushText()
  return segments
}

// 在文本中从某个位置起尝试提取一个 JSON 片段（支持嵌套 {}/[]，忽略字符串里的括号）
function extractJsonAt(src, start) {
  const text = String(src || '')
  const n = text.length
  if (start < 0 || start >= n) return null
  const first = text[start]
  if (first !== '{' && first !== '[') return null

  const stack = []
  let inString = false
  let escape = false

  for (let i = start; i < n; i += 1) {
    const ch = text[i]

    if (escape) {
      escape = false
      continue
    }
    if (ch === '\\') {
      if (inString) escape = true
      continue
    }
    if (ch === '"') {
      inString = !inString
      continue
    }
    if (inString) continue

    if (ch === '{' || ch === '[') {
      stack.push(ch)
    } else if (ch === '}' || ch === ']') {
      if (!stack.length) return null
      const last = stack[stack.length - 1]
      if ((last === '{' && ch === '}') || (last === '[' && ch === ']')) {
        stack.pop()
        if (!stack.length) {
          const candidate = text.slice(start, i + 1)
          try {
            JSON.parse(candidate)
            return { text: candidate, start, end: i + 1 }
          } catch {
            return null
          }
        }
      } else {
        return null
      }
    }
  }
  return null
}

// 将一段纯文本按“普通文本 + 内联 JSON”拆分
function splitTextWithInlineJson(text) {
  const src = String(text || '')
  const parts = []
  const n = src.length
  let lastPos = 0
  let i = 0

  while (i < n) {
    const ch = src[i]
    if (ch !== '{' && ch !== '[') {
      i += 1
      continue
    }
    const extracted = extractJsonAt(src, i)
    if (!extracted) {
      i += 1
      continue
    }
    if (extracted.start > lastPos) {
      const plain = src.slice(lastPos, extracted.start)
      if (plain) {
        parts.push({ kind: 'plain', content: plain })
      }
    }
    parts.push({ kind: 'json', content: extracted.text })
    lastPos = extracted.end
    i = extracted.end
  }

  if (lastPos < n) {
    const tail = src.slice(lastPos)
    if (tail) parts.push({ kind: 'plain', content: tail })
  }

  if (!parts.length) {
    return [{ kind: 'plain', content: src }]
  }
  return parts
}

export function parseChatBlocks(text) {
  const raw = String(text || '')
  if (!raw) return []

  // 1) 整条纯 JSON：单块 json 视图（会在流结束或结构闭合后自然生效）
  const wholeJson = isWholeJson(raw)
  if (wholeJson !== null) {
    return [{
      type: 'json',
      value: wholeJson,
      raw,
    }]
  }

  // 2) 先按 ```fence``` 拆成文本段和代码段
  const segments = splitByFences(raw)
  const blocks = []

  for (const seg of segments) {
    if (!seg || !seg.content) continue

    if (seg.kind === 'fence') {
      const lang = String(seg.lang || '')
      if (/^json$/i.test(lang)) {
        const parsed = tryParseJson(seg.content)
        if (parsed !== null) {
          blocks.push({
            type: 'json',
            value: parsed,
            raw: seg.content,
            lang: 'json',
          })
          continue
        }
      }
      blocks.push({
        type: 'code',
        lang,
        text: seg.content,
      })
      continue
    }

    // 文本段：进一步拆成 plain + 内联 JSON 片段
    const pieces = splitTextWithInlineJson(seg.content)
    for (const piece of pieces) {
      if (piece.kind === 'json') {
        const parsed = tryParseJson(piece.content)
        if (parsed !== null) {
          blocks.push({
            type: 'json',
            value: parsed,
            raw: piece.content,
          })
        } else {
          // 回退为普通文本
          blocks.push({
            type: 'text',
            text: piece.content,
          })
        }
      } else {
        const plain = String(piece.content || '')
        if (!plain.trim()) continue
        // 对非 JSON 片段使用 Markdown 渲染，保留标题/列表等能力
        const html = md.render(plain)
        blocks.push({
          type: 'markdown',
          html,
          raw: plain,
        })
      }
    }
  }

  if (!blocks.length) {
    return [{
      type: 'text',
      text: raw,
    }]
  }

  return blocks
}
