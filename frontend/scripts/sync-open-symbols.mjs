#!/usr/bin/env node
// Sync a curated subset of open-symbols SVGs into src/assets/open-symbols
// Usage:
//   node scripts/sync-open-symbols.mjs [--src <path-to-open-symbols-repo>]

import fs from 'fs'
import path from 'path'
import { fileURLToPath } from 'url'
import { ICON_MAP } from '../src/icons/registry.js'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const projectRoot = path.resolve(__dirname, '..')
const defaultSrc = path.resolve(projectRoot, 'vendor', 'open-symbols')
const destDir = path.resolve(projectRoot, 'src', 'assets', 'open-symbols')

function parseArgs(argv) {
  const args = { src: defaultSrc, prefer: 'lucide' }
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i]
    if (a === '--src' && argv[i + 1]) {
      args.src = path.resolve(process.cwd(), argv[++i])
    } else if (a === '--prefer' && argv[i + 1]) {
      args.prefer = String(argv[++i]).toLowerCase()
    }
  }
  return args
}

function ensureDir(p) {
  fs.mkdirSync(p, { recursive: true })
}

function walkSvgFiles(dir, out = []) {
  if (!fs.existsSync(dir)) return out
  const entries = fs.readdirSync(dir, { withFileTypes: true })
  for (const e of entries) {
    const p = path.join(dir, e.name)
    if (e.isDirectory()) walkSvgFiles(p, out)
    else if (e.isFile() && e.name.toLowerCase().endsWith('.svg')) out.push(p)
  }
  return out
}

function extractSymbolFromPreview(svg) {
  // Try to extract the Regular-S symbol area from SymbolKit preview sheets
  const get = (re) => {
    const m = svg.match(re)
    return m ? parseFloat(m[1]) : null
  }
  const left = get(/id="left-margin-Regular-S"[^>]*x1="([0-9.]+)"/)
  const right = get(/id="right-margin-Regular-S"[^>]*x1="([0-9.]+)"/)
  const cap = get(/id="Capline-S"[^>]*y1="([0-9.]+)"/)
  const base = get(/id="Baseline-S"[^>]*y1="([0-9.]+)"/)
  const contentMatch = svg.match(/<g id="Regular-S">([\s\S]*?)<\/g>/)
  if ([left, right, cap, base].every(v => typeof v === 'number') && contentMatch) {
    const width = right - left
    const height = base - cap
    const content = contentMatch[1]
    // Normalize to a canonical square viewBox (24x24)
    const CANVAS = 24
    const scale = CANVAS / Math.max(width, height)
    const dx = (CANVAS - width * scale) / 2
    const dy = (CANVAS - height * scale) / 2
    const header = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 ' + CANVAS + ' ' + CANVAS + '">'
    const body = '<g transform="translate(' + dx.toFixed(4) + ',' + dy.toFixed(4) + ') scale(' + scale.toFixed(6) + ') translate(' + (-left) + ',' + (-cap) + ')">' + content + '</g>'
    const tail = '</svg>'
    return header + body + tail
  }
  return null
}

function normalizeSvg(svg) {
  // If it's a SymbolKit preview, extract symbol-only content
  let s = svg
  if (/SymbolKit|SFSymbolsPreviewWireframe|id=\"Notes\"/.test(s)) {
    const extracted = extractSymbolFromPreview(s)
    if (extracted) s = extracted
    else {
      // fallback: strip <style>, <text>, guide lines
      s = s.replace(/<style[\s\S]*?<\/style>/g, '')
      s = s.replace(/<text[\s\S]*?<\/text>/g, '')
      s = s.replace(/<line[\s\S]*?>/g, '')
      s = s.replace(/<rect[^>]*id=\"artboard\"[\s\S]*?>/g, '')
    }
  }
  // remove width/height attributes
  s = s.replace(/\s(width|height)="[^"]*"/g, '')
  // force currentColor while keeping 'none'
  s = s.replace(/fill="(?!none)[^"]*"/g, 'fill="currentColor"')
  s = s.replace(/stroke="(?!none)[^"]*"/g, 'stroke="currentColor"')
  // remove stray className attributes from paths
  s = s.replace(/\sclassName="[^"]*"/g, '')
  return s
}

function main() {
  const { src, prefer } = parseArgs(process.argv.slice(2))
  if (!fs.existsSync(src)) {
    console.error(`Source directory not found: ${src}`)
    process.exit(1)
  }

  const files = walkSvgFiles(src)
  if (!files.length) {
    console.error(`No SVGs found under: ${src}`)
    process.exit(1)
  }

  // Index by base filename without extension
  const index = new Map()
  for (const f of files) {
    const base = path.basename(f).replace(/\.svg$/i, '')
    if (!index.has(base)) { index.set(base, f); continue }
    const prev = index.get(base)
    const fPref = f.includes(`/${prefer}/symbols/`)
    const pPref = prev.includes(`/${prefer}/symbols/`)
    if (fPref && !pPref) { index.set(base, f); continue }
    if (pPref && !fPref) { continue }
    // tie-breaker: shorter path
    if (String(f).length < String(prev).length) index.set(base, f)
  }

  const needed = new Set(Object.values(ICON_MAP))
  ensureDir(destDir)

  let ok = 0, miss = 0
  for (const name of needed) {
    const srcPath = index.get(name)
    if (!srcPath) {
      console.warn(`[MISS] ${name}.svg not found in source repo`)
      miss++
      continue
    }
    const content = fs.readFileSync(srcPath, 'utf-8')
    const out = normalizeSvg(content)
    fs.writeFileSync(path.join(destDir, `${name}.svg`), out, 'utf-8')
    ok++
  }

  console.log(`Synced ${ok} icons -> ${path.relative(projectRoot, destDir)}${miss ? `, missing ${miss}` : ''}`)
  if (miss) {
    console.log(`Hint: you can try a different collection with --prefer remix|feather|tabler|heroicons|font-awesome`)
  }
}

main()
